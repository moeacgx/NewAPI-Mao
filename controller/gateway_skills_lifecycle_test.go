package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func gatewaySkillSessionContext(t *testing.T, tokenID int) (*gin.Context, *service.GatewaySkillSession) {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test","messages":[]}`))
	common.SetContextKey(c, constant.ContextKeyTokenId, tokenID)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, canvasRouteBillingModel)
	session, err := service.NewGatewaySkillSession(c, types.RelayFormatOpenAI)
	require.NoError(t, err)
	require.NotNil(t, session)
	return c, session
}

func executeGatewaySkill(t *testing.T, c *gin.Context, session *service.GatewaySkillSession, name string, args map[string]any) gjson.Result {
	t.Helper()
	encoded, err := common.Marshal(args)
	require.NoError(t, err)
	results, err := session.Execute(c, []service.GatewaySkillCall{{ID: "read", Name: name, Arguments: encoded}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	return gjson.ParseBytes(results[0].Content)
}

func TestGatewaySkillsLargeFilesUseIndependentReadBudget(t *testing.T) {
	_, _, token, template := setupGatewaySkillsRelay(t, "http://127.0.0.1:1", false, 10000)
	template.Skills[0].Content = strings.Repeat("大", 120*1024) + "unread-tail"
	saved, err := model.SavePromptTemplate(*template)
	require.NoError(t, err)
	c, session := gatewaySkillSessionContext(t, token.Id)
	args := map[string]any{"template_id": saved.ID, "skill_id": saved.Skills[0].ID, "version": saved.Version, "path": "SKILL.md", "offset": 0}
	offset := int64(0)
	for page := 0; page < 8; page++ {
		args["offset"] = offset
		result := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
		require.False(t, result.Get("error").Exists(), result.Raw)
		assert.True(t, utf8.ValidString(result.Get("content").String()))
		assert.Len(t, result.Get("content").String(), model.MaxSkillReadBytes)
		offset = result.Get("next_offset").Int()
		assert.EqualValues(t, (page+1)*model.MaxSkillReadBytes, offset)
	}
	args["offset"] = offset
	exhausted := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
	assert.Contains(t, exhausted.Get("error").String(), "预算已用完")
	args["offset"] = 0
	repeated := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
	assert.False(t, repeated.Get("error").Exists(), repeated.Raw)
	assert.Zero(t, repeated.Get("remaining_bytes").Int())
	final, err := session.FinalBody(c, []byte(`{"model":"test","messages":[{"role":"user","content":"question"}]}`))
	require.NoError(t, err)
	text := gjson.GetBytes(final, "messages.0.content").String()
	assert.Equal(t, 8*model.MaxSkillReadBytes/len("大"), strings.Count(text, "大"))
	assert.NotContains(t, text, "unread-tail")
	_, err = service.ApplyTokenPromptText([]byte(`{"messages":[]}`), types.RelayFormatOpenAI, "test", []string{strings.Repeat("a", 128*1024)})
	require.Error(t, err, "旧全文模式仍保留原有限额")
}

func TestGatewaySkillsResourceDirectoryPagesAndRevocation(t *testing.T) {
	_, user, token, template := setupGatewaySkillsRelay(t, "http://127.0.0.1:1", false, 10000)
	template.Skills[0].Content = "请按需阅读引用资料。"
	template.Skills[0].Resources = nil
	for index := 0; index < 80; index++ {
		path := fmt.Sprintf("references/r%03d-%s.md", index, strings.Repeat("x", 300))
		template.Skills[0].Resources = append(template.Skills[0].Resources, model.PromptSkillResource{Path: path, Content: fmt.Sprintf("private-resource-%d", index)})
	}
	saved, err := model.SavePromptTemplate(*template)
	require.NoError(t, err)
	c, session := gatewaySkillSessionContext(t, token.Id)
	lastPath := saved.Skills[0].Resources[79].Path
	args := map[string]any{"template_id": saved.ID, "skill_id": saved.Skills[0].ID, "version": saved.Version, "path": lastPath}
	before := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
	assert.Contains(t, before.Get("error").String(), "先读取 SKILL.md")
	args["path"] = "SKILL.md"
	main := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
	require.False(t, main.Get("error").Exists(), main.Raw)
	assert.EqualValues(t, 80, main.Get("resources_total").Int())
	assert.Positive(t, main.Get("resources_next_offset").Int())
	assert.Less(t, main.Get("resources_next_offset").Int(), int64(80))
	assert.LessOrEqual(t, len(main.Get("resources").Raw), 8*1024)
	args["path"] = lastPath
	undisclosed := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
	assert.True(t, undisclosed.Get("error").Exists())
	directoryArgs := map[string]any{"template_id": saved.ID, "skill_id": saved.Skills[0].ID, "version": saved.Version, "query": "r079"}
	directory := executeGatewaySkill(t, c, session, service.GatewaySkillsFilesTool, directoryArgs)
	require.False(t, directory.Get("error").Exists(), directory.Raw)
	assert.Equal(t, lastPath, directory.Get("resources.0").String())
	assert.True(t, directory.Get("eof").Bool())
	assert.NotContains(t, directory.Raw, "private-resource")
	read := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
	assert.Equal(t, "private-resource-79", read.Get("content").String())
	directoryArgs["query"] = ""
	directoryArgs["offset"] = main.Get("resources_next_offset").Int()
	next := executeGatewaySkill(t, c, session, service.GatewaySkillsFilesTool, directoryArgs)
	assert.NotEqual(t, main.Get("resources.0").String(), next.Get("resources.0").String())

	common.SetContextKey(c, constant.ContextKeyUsingGroup, "other")
	_, err = session.Execute(c, []service.GatewaySkillCall{{ID: "list", Name: service.GatewaySkillsListTool, Arguments: []byte(`{}`)}})
	require.ErrorContains(t, err, "分组已改变")
	_, err = session.FinalBody(c, []byte(`{"messages":[]}`))
	require.ErrorContains(t, err, "分组已改变")
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
	require.NoError(t, model.ReplaceTokenPromptBindings(user.Id, token.Id, nil, nil))
	revoked := executeGatewaySkill(t, c, session, service.GatewaySkillsReadTool, args)
	assert.True(t, revoked.Get("error").Exists())
	_, err = session.FinalBody(c, []byte(`{"messages":[]}`))
	require.ErrorContains(t, err, "授权已失效")
	cancelled, cancel := context.WithCancel(c.Request.Context())
	c.Request = c.Request.WithContext(cancelled)
	cancel()
	_, err = session.Execute(c, []service.GatewaySkillCall{{ID: "list", Name: service.GatewaySkillsListTool, Arguments: []byte(`{}`)}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestGatewaySkillsOnlyRefundUnsettledRound(t *testing.T) {
	for _, test := range []struct {
		name       string
		quota      int
		failAt     int
		invalidAt  int
		wantCalls  int
		wantLogs   int
		wantRemain int
	}{
		{name: "final-insufficient-quota", quota: 250, wantCalls: 2, wantLogs: 2, wantRemain: 50},
		{name: "second-private-upstream-fails", quota: 10000, failAt: 2, wantCalls: 2, wantLogs: 1, wantRemain: 9900},
		{name: "final-upstream-fails", quota: 10000, failAt: 3, wantCalls: 3, wantLogs: 2, wantRemain: 9800},
		{name: "settled-response-has-invalid-tool", quota: 10000, invalidAt: 1, wantCalls: 1, wantLogs: 1, wantRemain: 9900},
	} {
		t.Run(test.name, func(t *testing.T) {
			var template *model.PromptTemplate
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				calls++
				w.Header().Set("Content-Type", "application/json")
				if calls == test.failAt {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = io.WriteString(w, `{"error":{"message":"upstream failed","type":"upstream_error"}}`)
					return
				}
				message := map[string]any{"role": "assistant", "content": "SKILL_CONTEXT_READY"}
				if calls == 1 {
					args, marshalErr := common.Marshal(map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version})
					assert.NoError(t, marshalErr)
					name := service.GatewaySkillsReadTool
					if test.invalidAt == calls {
						name = "client-tool-must-not-run"
					}
					message["tool_calls"] = []any{map[string]any{"id": "read_main", "type": "function", "function": map[string]any{"name": name, "arguments": string(args)}}}
				}
				body, marshalErr := common.Marshal(map[string]any{"id": fmt.Sprintf("round-%d", calls), "object": "chat.completion", "model": canvasRouteBillingModel,
					"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": "stop"}},
					"usage":   map[string]int{"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}})
				assert.NoError(t, marshalErr)
				_, _ = w.Write(body)
			}))
			defer upstream.Close()
			engine, user, token, saved := setupGatewaySkillsRelay(t, upstream.URL, false, test.quota)
			template = saved
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review code"}]}`, canvasRouteBillingModel)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			assert.GreaterOrEqual(t, recorder.Code, 400, recorder.Body.String())
			assert.Equal(t, test.wantCalls, calls)
			require.Eventually(t, func() bool {
				var storedUser model.User
				var storedToken model.Token
				if model.DB.First(&storedUser, user.Id).Error != nil || model.DB.First(&storedToken, token.Id).Error != nil {
					return false
				}
				return storedUser.Quota == int64(test.wantRemain) && storedToken.RemainQuota == test.wantRemain
			}, 3*time.Second, 10*time.Millisecond, "只退回尚未结算轮次，已成功的私有调用继续计费")
			var logs []model.Log
			require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Find(&logs).Error)
			assert.Len(t, logs, test.wantLogs)
			for _, log := range logs {
				assert.Equal(t, 100, log.Quota)
			}
		})
	}
}
