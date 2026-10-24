package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillsAdmissionRejectsBeforeUpstreamAndRefunds(t *testing.T) {
	for _, test := range []struct {
		name       string
		budget     string
		occupied   bool
		uninstall  bool
		wantStatus int
		wantCode   string
	}{
		{"count", "1048576", true, false, http.StatusTooManyRequests, "gateway_skills_busy"},
		{"context", "1", false, false, http.StatusRequestEntityTooLarge, "gateway_skills_context_too_large"},
		{"without-skills", "1048576", true, true, http.StatusOK, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GATEWAY_SKILLS_MAX_CONCURRENT", "1")
			t.Setenv("GATEWAY_SKILLS_MAX_INFLIGHT_CONTEXT_BYTES", test.budget)
			var calls atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"final","object":"chat.completion","model":"`+canvasRouteBillingModel+`","choices":[{"index":0,"message":{"role":"assistant","content":"answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25}}`)
			}))
			defer upstream.Close()
			engine, user, token, _ := setupGatewaySkillsRelay(t, upstream.URL, false, 10000)
			if test.uninstall {
				require.NoError(t, model.ReplaceTokenPromptBindings(user.Id, token.Id, []model.TokenPromptBindingInput{}, nil))
			}
			if test.occupied {
				admission, apiErr := service.TryAcquireGatewaySkillRequest(1)
				require.Nil(t, apiErr)
				defer admission.Release()
			}
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review code"}]}`, canvasRouteBillingModel)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			require.Equal(t, test.wantStatus, recorder.Code, recorder.Body.String())
			wantCalls, wantQuota := int32(0), 10000
			if test.uninstall {
				wantCalls, wantQuota = 1, 9900
			} else {
				assert.Equal(t, test.wantCode, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
			}
			assert.Equal(t, wantCalls, calls.Load())
			require.Eventually(t, func() bool {
				var storedUser model.User
				var storedToken model.Token
				return model.DB.First(&storedUser, user.Id).Error == nil && model.DB.First(&storedToken, token.Id).Error == nil &&
					storedUser.Quota == int64(wantQuota) && storedToken.RemainQuota == wantQuota
			}, 3*time.Second, 10*time.Millisecond, "本地拒绝不收费，普通请求仍正常结算")
			var count int64
			require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Count(&count).Error)
			assert.EqualValues(t, wantCalls, count)
		})
	}
}

type gatewaySkillContextRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip gatewaySkillContextRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestGatewaySkillsAdmissionCoversFinalStreamAndReleases(t *testing.T) {
	for _, cancelClient := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%t", cancelClient), func(t *testing.T) {
			t.Setenv("GATEWAY_SKILLS_MAX_CONCURRENT", "1")
			finalStarted, releaseFinal := make(chan struct{}), make(chan struct{})
			var closeFinal sync.Once
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				if !gjson.GetBytes(body, "stream").Bool() {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"id":"ready","object":"chat.completion","model":"`+canvasRouteBillingModel+`","choices":[{"index":0,"message":{"role":"assistant","content":"ready"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25}}`)
					return
				}
				close(finalStarted)
				select {
				case <-releaseFinal:
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = io.WriteString(w, "data: {\"id\":\"final\",\"object\":\"chat.completion.chunk\",\"model\":\""+canvasRouteBillingModel+"\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"answer\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":5,\"total_tokens\":25}}\n\ndata: [DONE]\n\n")
				case <-r.Context().Done():
				}
			}))
			defer upstream.Close()
			defer closeFinal.Do(func() { close(releaseFinal) })
			engine, _, token, _ := setupGatewaySkillsRelay(t, upstream.URL, false, 10000)
			client := service.GetHttpClient()
			transport := client.Transport
			client.Transport = gatewaySkillContextRoundTripper(func(r *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				r.Body = io.NopCloser(bytes.NewReader(body))
				_, hasDeadline := r.Context().Deadline()
				assert.Equal(t, !gjson.GetBytes(body, "stream").Bool(), hasDeadline, "只有私有读取阶段继承技能时限")
				return transport.RoundTrip(r)
			})
			t.Cleanup(func() { client.Transport = transport })
			clientContext, cancel := context.WithCancel(context.Background())
			defer cancel()
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review code"}],"stream":true}`, canvasRouteBillingModel))).WithContext(clientContext)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			recorder := httptest.NewRecorder()
			done := make(chan struct{})
			go func() {
				defer close(done)
				engine.ServeHTTP(recorder, request)
			}()
			select {
			case <-finalStarted:
			case <-time.After(5 * time.Second):
				t.Fatal("正式回答未开始")
			}
			probe, apiErr := service.TryAcquireGatewaySkillRequest(1)
			if probe != nil {
				probe.Release()
			}
			require.NotNil(t, apiErr, "正式流式回答结束前不能归还技能额度")
			assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
			if cancelClient {
				cancel()
			} else {
				closeFinal.Do(func() { close(releaseFinal) })
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("请求结束后未完成收尾")
			}
			if !cancelClient {
				assert.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
				assert.Contains(t, recorder.Body.String(), "answer")
			}
			probe, apiErr = service.TryAcquireGatewaySkillRequest(1)
			require.Nil(t, apiErr, "完成或取消后必须归还额度")
			probe.Release()
		})
	}
}

func TestGatewaySkillsDiscoveryTimeoutPreservesSettledUsage(t *testing.T) {
	for _, settledRound := range []bool{false, true} {
		t.Run(fmt.Sprintf("settled=%t", settledRound), func(t *testing.T) {
			t.Setenv("GATEWAY_SKILLS_DISCOVERY_TIMEOUT_SECONDS", "1")
			var calls atomic.Int32
			var template *model.PromptTemplate
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(io.Discard, r.Body)
				assert.NoError(t, err)
				if calls.Add(1) == 1 && settledRound {
					args, err := common.Marshal(map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version})
					assert.NoError(t, err)
					body, err := common.Marshal(map[string]any{"id": "read", "object": "chat.completion", "model": canvasRouteBillingModel,
						"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": nil,
							"tool_calls": []any{map[string]any{"id": "read_main", "type": "function", "function": map[string]any{"name": service.GatewaySkillsReadTool, "arguments": string(args)}}}}, "finish_reason": "tool_calls"}},
						"usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}})
					assert.NoError(t, err)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(body)
					return
				}
				<-r.Context().Done()
			}))
			defer upstream.Close()
			engine, user, token, saved := setupGatewaySkillsRelay(t, upstream.URL, false, 10000)
			template = saved
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review code"}]}`, canvasRouteBillingModel)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusGatewayTimeout, recorder.Code, recorder.Body.String())
			assert.Equal(t, "gateway_skills_timeout", gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
			wantCalls, wantQuota, wantLogs := int32(1), 10000, int64(0)
			if settledRound {
				wantCalls, wantQuota, wantLogs = 2, 9900, 1
			}
			assert.Equal(t, wantCalls, calls.Load(), "技能期限到期不得重新发起收费重试")
			require.Eventually(t, func() bool {
				var storedUser model.User
				var storedToken model.Token
				return model.DB.First(&storedUser, user.Id).Error == nil && model.DB.First(&storedToken, token.Id).Error == nil &&
					storedUser.Quota == int64(wantQuota) && storedToken.RemainQuota == wantQuota
			}, 3*time.Second, 10*time.Millisecond)
			var count int64
			require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Count(&count).Error)
			assert.Equal(t, wantLogs, count)
			probe, apiErr := service.TryAcquireGatewaySkillRequest(1)
			require.Nil(t, apiErr)
			probe.Release()
		})
	}
}
