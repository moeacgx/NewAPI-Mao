package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillsCatalogueReachesFinalAssistant(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		t.Run(fmt.Sprintf("passthrough=%t", passthrough), func(t *testing.T) {
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				calls++
				assert.NotContains(t, string(body), "main-secret-start")
				assert.NotContains(t, string(body), "reference-secret-only-after-read")
				assert.NotContains(t, string(body), "irrelevant-skill-must-stay-private")
				message := map[string]any{"role": "assistant", "content": "SKILL_CONTEXT_READY"}
				switch calls {
				case 1:
					assert.Contains(t, string(body), service.GatewaySkillsListTool)
					message = map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{
						"id": "list_installed", "type": "function", "function": map[string]any{"name": service.GatewaySkillsListTool, "arguments": "{}"}}}}
				case 2:
					assert.Contains(t, string(body), "代码审查")
					assert.Contains(t, string(body), "翻译")
					assert.Contains(t, string(body), "list_installed")
				case 3:
					assert.Contains(t, string(body), "代码审查")
					assert.Contains(t, string(body), "审查代码并参考相关规范")
					assert.Contains(t, string(body), "翻译")
					assert.Contains(t, string(body), "client_lookup")
					assert.Equal(t, "0", gjson.GetBytes(body, "temperature").Raw)
					assert.NotContains(t, string(body), service.GatewaySkillsListTool)
					message["content"] = "可用技能：代码审查、翻译。"
				}
				w.Header().Set("Content-Type", "application/json")
				response, err := common.Marshal(map[string]any{"id": "catalogue-answer", "object": "chat.completion", "model": canvasRouteBillingModel,
					"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": "stop"}},
					"usage":   map[string]int{"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}})
				require.NoError(t, err)
				_, _ = w.Write(response)
			}))
			defer upstream.Close()
			engine, _, token, _ := setupGatewaySkillsRelay(t, upstream.URL, passthrough, 10000)
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"有哪些网关云端 Skills？"}],"temperature":0,"tools":[{"type":"function","function":{"name":"client_lookup","parameters":{"type":"object"}}}],"tool_choice":"auto"}`, canvasRouteBillingModel)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			assert.Equal(t, 3, calls)
			assert.Contains(t, recorder.Body.String(), "可用技能：代码审查、翻译。")
			assert.Empty(t, recorder.Header().Get(service.GatewaySkillsResponseHeader))
			assert.NotContains(t, recorder.Body.String(), "☁ 网关云端 Skills")
		})
	}
}

func TestGatewaySkillsCatalogueReauthorizesBeforeFinalBody(t *testing.T) {
	for _, change := range []string{"uninstall", "unpublish", "version"} {
		t.Run(change, func(t *testing.T) {
			_, user, token, template := setupGatewaySkillsRelay(t, "http://127.0.0.1:1", false, 10000)
			c, session := gatewaySkillSessionContext(t, token.Id)
			result := executeGatewaySkill(t, c, session, service.GatewaySkillsListTool, map[string]any{"query": "翻译"})
			require.False(t, result.Get("error").Exists(), result.Raw)
			assert.EqualValues(t, 1, result.Get("total").Int())
			switch change {
			case "uninstall":
				require.NoError(t, model.ReplaceTokenPromptBindings(user.Id, token.Id, nil, nil))
			case "unpublish":
				template.Enabled = false
				_, err := model.SavePromptTemplate(*template)
				require.NoError(t, err)
			case "version":
				template.Skills[1].Description = "新版翻译用途"
				_, err := model.SavePromptTemplate(*template)
				require.NoError(t, err)
			}
			final, err := session.FinalBody(c, []byte(`{"model":"test","messages":[{"role":"user","content":"列出技能"}]}`))
			require.NoError(t, err)
			catalogue := gatewaySkillCatalogueFromBody(t, final)
			assert.Zero(t, catalogue.Get("lists.0.total").Int())
			assert.Empty(t, catalogue.Get("lists.0.skills").Array())
			assert.NotContains(t, string(final), "新版翻译用途")
			assert.NotContains(t, string(final), template.Skills[0].Content)
			assert.Nil(t, service.GatewaySkillResponseUsageFromContext(c))
		})
	}
}

func gatewaySkillCatalogueFromBody(t *testing.T, body []byte) gjson.Result {
	t.Helper()
	text := gjson.GetBytes(body, "messages.0.content").String()
	_, remainder, found := strings.Cut(text, "<gateway_skill_catalogue>\n")
	require.True(t, found, text)
	encoded, _, found := strings.Cut(remainder, "\n</gateway_skill_catalogue>")
	require.True(t, found, remainder)
	require.True(t, gjson.Valid(encoded), encoded)
	return gjson.Parse(encoded)
}

func TestGatewaySkillsCatalogueQueryContract(t *testing.T) {
	for _, test := range []struct {
		name      string
		args      map[string]any
		wantTitle string
		wantTotal int64
		wantEnd   int64
		wantError bool
	}{
		{name: "search", args: map[string]any{"query": " 翻译 "}, wantTitle: "翻译", wantTotal: 1, wantEnd: 1},
		{name: "offset", args: map[string]any{"offset": 1}, wantTitle: "翻译", wantTotal: 2, wantEnd: 2},
		{name: "empty-page", args: map[string]any{"offset": 99}, wantTotal: 2, wantEnd: 2},
		{name: "invalid-query", args: map[string]any{"offset": -1}, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, token, _ := setupGatewaySkillsRelay(t, "http://127.0.0.1:1", false, 10000)
			c, session := gatewaySkillSessionContext(t, token.Id)
			original := []byte(`{"model":"test","messages":[{"role":"user","content":"列出技能"}]}`)
			unqueried, err := session.FinalBody(c, original)
			require.NoError(t, err)
			assert.Equal(t, original, unqueried)
			result := executeGatewaySkill(t, c, session, service.GatewaySkillsListTool, test.args)
			assert.Equal(t, test.wantError, result.Get("error").Exists())
			final, err := session.FinalBody(c, original)
			require.NoError(t, err)
			if test.wantError {
				assert.Equal(t, original, final)
				return
			}
			catalogue := gatewaySkillCatalogueFromBody(t, final)
			assert.Equal(t, test.wantTitle, catalogue.Get("lists.0.skills.0.title").String())
			assert.Equal(t, test.wantTotal, catalogue.Get("lists.0.total").Int())
			assert.Equal(t, test.wantEnd, catalogue.Get("lists.0.next_offset").Int())
			assert.True(t, catalogue.Get("lists.0.eof").Bool())
			assert.NotContains(t, string(final), "main-secret-start")
			assert.NotContains(t, string(final), "SKILL.md")
			assert.Nil(t, service.GatewaySkillResponseUsageFromContext(c))
			executeGatewaySkill(t, c, session, service.GatewaySkillsListTool, test.args)
			repeated, err := session.FinalBody(c, original)
			require.NoError(t, err)
			assert.Len(t, gatewaySkillCatalogueFromBody(t, repeated).Get("lists").Array(), 1)
			session.Audit(c, "generation", 0)
			audit, _ := c.Get("gateway_skills_audit")
			encoded, err := common.Marshal(audit)
			require.NoError(t, err)
			assert.False(t, gjson.GetBytes(encoded, "catalogue_queries.0.query").Exists())
			assert.True(t, gjson.GetBytes(encoded, "catalogue_queries.0.query_bytes").Exists())
		})
	}
}

func TestGatewaySkillsCatalogueBudgetPreservesWholePages(t *testing.T) {
	_, _, token, template := setupGatewaySkillsRelay(t, "http://127.0.0.1:1", false, 10000)
	template.Skills = nil
	for index := 0; index < 64; index++ {
		template.Skills = append(template.Skills, model.PromptSkill{Title: fmt.Sprintf("技能 %02d", index),
			Description: strings.Repeat("用途", 256), Path: fmt.Sprintf("skill-%02d/SKILL.md", index), Content: "不应装载的技能正文"})
	}
	_, err := model.SavePromptTemplate(*template)
	require.NoError(t, err)
	c, session := gatewaySkillSessionContext(t, token.Id)
	for _, offset := range []int{0, 18, 36} {
		result := executeGatewaySkill(t, c, session, service.GatewaySkillsListTool, map[string]any{"offset": offset})
		require.False(t, result.Get("error").Exists(), result.Raw)
	}
	final, err := session.FinalBody(c, []byte(`{"model":"test","messages":[{"role":"user","content":"列出技能"}]}`))
	require.NoError(t, err)
	catalogue := gatewaySkillCatalogueFromBody(t, final)
	assert.LessOrEqual(t, len(catalogue.Raw), 64*1024)
	assert.True(t, catalogue.Get("truncated").Bool())
	assert.NotEmpty(t, catalogue.Get("lists").Array())
	assert.Less(t, len(catalogue.Get("lists").Array()), 3)
	assert.Equal(t, "技能 00", catalogue.Get("lists.0.skills.0.title").String())
	assert.NotContains(t, string(final), "不应装载的技能正文")
}
