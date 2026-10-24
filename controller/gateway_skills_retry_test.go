package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillsAutoRetryReloadsActualGroupAndPreservesSettledUsage(t *testing.T) {
	clientBody := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review my code"}],"temperature":0,"stream":false,"tools":[{"type":"function","function":{"name":"client_lookup","parameters":{"type":"object"}}}],"tool_choice":"required"}`, canvasRouteBillingModel)
	var template *model.PromptTemplate
	var firstGroupCalls atomic.Int32
	firstUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		call := firstGroupCalls.Add(1)
		assert.NotContains(t, string(body), "irrelevant-skill-must-stay-private")
		assert.NotContains(t, string(body), "unused-resource-must-stay-private")
		assert.NotContains(t, string(body), "main-unread-tail")
		w.Header().Set("Content-Type", "application/json")
		if call == 4 {
			assert.Contains(t, string(body), "main-secret-start")
			assert.Contains(t, string(body), "reference-secret-only-after-read")
			assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.function.name").String())
			assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
			assert.NotContains(t, string(body), service.GatewaySkillsReadTool)
			assert.NotContains(t, string(body), "SKILL_CONTEXT_READY")
			var settledReadLogs int64
			assert.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("type = ? AND quota = ?", model.LogTypeConsume, 100).Count(&settledReadLogs).Error)
			assert.EqualValues(t, 3, settledReadLogs, "successful discovery rounds must settle before final generation fails")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":{"message":"first group generation failed","type":"server_error","code":"server_error"}}`)
			return
		}
		if !assert.LessOrEqual(t, call, int32(3), "the first group must be excluded after its generation fails") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		assert.False(t, gjson.GetBytes(body, "stream").Bool())
		assert.NotContains(t, string(body), "client_lookup")
		message := map[string]any{"role": "assistant", "content": "SKILL_CONTEXT_READY"}
		if call <= 2 {
			args := map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version, "path": "SKILL.md"}
			if call == 1 {
				assert.NotContains(t, string(body), "main-secret-start")
			} else {
				assert.Contains(t, string(body), "main-secret-start")
				assert.NotContains(t, string(body), "reference-secret-only-after-read")
				args["path"] = "references/rules.md"
			}
			arguments, marshalErr := common.Marshal(args)
			if !assert.NoError(t, marshalErr) {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			message = map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{
				"id": fmt.Sprintf("read_%d", call), "type": "function",
				"function": map[string]any{"name": service.GatewaySkillsReadTool, "arguments": string(arguments)},
			}}}
		} else {
			assert.Contains(t, string(body), "main-secret-start")
			assert.Contains(t, string(body), "reference-secret-only-after-read")
		}
		encoded, marshalErr := common.Marshal(map[string]any{
			"id": fmt.Sprintf("chatcmpl-first-skill-%d", call), "object": "chat.completion", "model": canvasRouteBillingModel,
			"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25},
		})
		if !assert.NoError(t, marshalErr) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(encoded)
	}))
	defer firstUpstream.Close()

	var secondGroupCalls atomic.Int32
	secondUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		secondGroupCalls.Add(1)
		// 新分组不在技能白名单中，必须从原始客户端消息和工具重新构造请求。
		assert.JSONEq(t, gjson.Get(clientBody, "messages").Raw, gjson.GetBytes(body, "messages").Raw)
		assert.JSONEq(t, gjson.Get(clientBody, "tools").Raw, gjson.GetBytes(body, "tools").Raw)
		assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
		assert.Equal(t, "0", gjson.GetBytes(body, "temperature").Raw)
		assert.False(t, gjson.GetBytes(body, "stream").Bool())
		for _, privateText := range []string{
			"main-secret-start", "reference-secret-only-after-read", "unused-resource-must-stay-private",
			"irrelevant-skill-must-stay-private", "main-unread-tail", "SKILL_CONTEXT_READY",
			"newapi_skills", "read_1", "read_2", template.Skills[0].ID,
		} {
			assert.NotContains(t, string(body), privateText)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chatcmpl-second-final","object":"chat.completion","model":"`+canvasRouteBillingModel+`","choices":[{"index":0,"message":{"role":"assistant","content":"second-group-final-answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":40,"completion_tokens":10,"total_tokens":50}}`)
	}))
	defer secondUpstream.Close()

	engine, user, token, saved := setupGatewaySkillsRelay(t, firstUpstream.URL, false, 10000)
	template = saved
	previousRetryTimes := common.RetryTimes
	previousMaxAutoGroups := setting.GetMaxTokenAutoGroups()
	previousUsableGroups := setting.UserUsableGroups2JSONString()
	previousRetryStatuses := operation_setting.AutomaticRetryStatusCodesToString()
	t.Cleanup(func() {
		common.RetryTimes = previousRetryTimes
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(fmt.Sprintf("%d", previousMaxAutoGroups)))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousUsableGroups))
		require.NoError(t, operation_setting.AutomaticRetryStatusCodesFromString(previousRetryStatuses))
	})
	common.RetryTimes = 1
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("2"))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"通用模型组","vip":"备用模型组","auto":"自动选择"}`))
	require.NoError(t, operation_setting.AutomaticRetryStatusCodesFromString("500"))
	token.Group = "auto"
	token.CrossGroupRetry = true
	require.NoError(t, token.SetAutoGroups([]string{"default", "vip"}))
	require.NoError(t, model.DB.Model(&token).Updates(map[string]any{
		"group": token.Group, "cross_group_retry": token.CrossGroupRetry, "auto_groups": token.AutoGroups,
	}).Error)
	var firstChannel model.Channel
	require.NoError(t, model.DB.Where("name = ?", "canvas-route-upstream").First(&firstChannel).Error)
	vipGroup := model.Group{Code: "vip", Name: "备用模型组", Ratio: 1, Status: model.GroupStatusActive, CreatedTime: time.Now().Unix(), UpdatedTime: time.Now().Unix()}
	require.NoError(t, model.DB.Create(&vipGroup).Error)
	priority, weight, secondURL := int64(0), uint(100), secondUpstream.URL
	secondChannel := model.Channel{
		Type: constant.ChannelTypeOpenAI, Key: "second-group-upstream-key", Status: common.ChannelStatusEnabled,
		Name: "second-group-upstream", Weight: &weight, Models: canvasRouteBillingModel,
		Group: "vip", Priority: &priority, BaseURL: &secondURL,
	}
	require.NoError(t, model.DB.Create(&secondChannel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group: "vip", GroupId: vipGroup.Id, Model: canvasRouteBillingModel, ChannelId: secondChannel.Id,
		Enabled: true, Priority: &priority, Weight: weight,
	}).Error)
	model.InitChannelCache()

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(clientBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer sk-"+token.Key)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.EqualValues(t, 4, firstGroupCalls.Load())
	assert.EqualValues(t, 1, secondGroupCalls.Load(), "non-whitelisted group must skip skill discovery")
	assert.Equal(t, "second-group-final-answer", gjson.Get(recorder.Body.String(), "choices.0.message.content").String())
	assert.NotContains(t, recorder.Body.String(), "newapi_skills")
	assert.NotContains(t, recorder.Body.String(), "main-secret-start")
	assert.NotContains(t, recorder.Body.String(), "reference-secret-only-after-read")

	// 只收取三轮成功读取和第二组最终生成；第一组失败生成的预扣不得成为额外费用。
	require.Eventually(t, func() bool {
		var storedUser model.User
		var storedToken model.Token
		if model.DB.First(&storedUser, user.Id).Error != nil || model.DB.First(&storedToken, token.Id).Error != nil {
			return false
		}
		return storedUser.Quota == 9600 && storedToken.RemainQuota == 9600
	}, 3*time.Second, 10*time.Millisecond)
	var logs []model.Log
	require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Order("id").Find(&logs).Error)
	require.Len(t, logs, 4)
	for index, log := range logs {
		assert.Equal(t, 100, log.Quota)
		assert.NotContains(t, log.Other, "main-secret-start")
		assert.NotContains(t, log.Other, "reference-secret-only-after-read")
		if index < 3 {
			assert.Equal(t, "default", log.Group)
			assert.Equal(t, firstChannel.Id, log.ChannelId)
			assert.Equal(t, "discovery", gjson.Get(log.Other, "gateway_skills.stage").String())
			assert.Equal(t, 20, log.PromptTokens)
			assert.Equal(t, 5, log.CompletionTokens)
		} else {
			assert.Equal(t, "vip", log.Group)
			assert.Equal(t, secondChannel.Id, log.ChannelId)
			assert.False(t, gjson.Get(log.Other, "gateway_skills").Exists())
			assert.False(t, gjson.Get(log.Other, "prompt_templates").Exists())
			assert.Equal(t, 40, log.PromptTokens)
			assert.Equal(t, 10, log.CompletionTokens)
		}
	}
}
