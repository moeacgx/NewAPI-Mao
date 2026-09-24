package jsplugin

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const performanceFilterPluginSource = `
export const meta = {
  apiVersion: 1, key: "performance-filter", name: "Performance filter", version: "1.0.0",
  author: {name: "Test"}, models: ["model"], fetchMode: "per_task",
  requiredCapabilities: ["task-performance-filter@1"],
  routes: [{method: "POST", path: "/performance-filter", type: "submit", decode: "decode", render: "render"}]
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function parseTaskResult() { return {}; }
export function buildQueryRequest() { return {}; }
export function shouldRecordPerformanceFailure(ctx, failure) {
  if (failure.stage === "http") return false;
  return null;
}
export const native = {decode() { return {kind:"submit", model:"model", requestBody:{}}; }, render(ctx, task) { return task; }};
`

func TestPerformanceFailureFilterCapabilityAndFailOpenContract(t *testing.T) {
	plugin, err := CompilePlugin(performanceFilterPluginSource, Options{})
	require.NoError(t, err)
	require.True(t, plugin.PerformanceFailureFilter)

	assert.False(t, plugin.ShouldRecordPerformanceFailure(context.Background(), PerformanceFailureInput{
		Model: "model", RequestPath: "/v1/systemone", Method: "POST",
	}, PerformanceFailure{Stage: "http", HTTPStatus: 502, ErrorCode: "upstream_error"}))
	assert.True(t, plugin.ShouldRecordPerformanceFailure(context.Background(), PerformanceFailureInput{
		Model: "model", RequestPath: "/v1/systemone", Method: "POST",
	}, PerformanceFailure{Stage: "parse", HTTPStatus: 502, ErrorCode: "parse_error"}))
}

func TestPerformanceFailureFilterCapabilityRequiresExactExport(t *testing.T) {
	for _, test := range []struct {
		name, metadata, hook string
		wantErr              string
	}{
		{"capability without hook", `requiredCapabilities:["task-performance-filter@1"],`, "", "performance"},
		{"hook without capability", "", `export function shouldRecordPerformanceFailure(){return false;}`, "capability"},
		{"non function hook", `requiredCapabilities:["task-performance-filter@1"],`, `export const shouldRecordPerformanceFailure = true;`, "function"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := routingTestPluginSource("performance-filter-contract", 0, `["model"]`, test.metadata, test.hook)
			_, err := CompilePlugin(source, Options{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestPerformanceFailureFilterKeepsLegacyPluginsCompatible(t *testing.T) {
	plugin, err := CompilePlugin(routingTestPluginSource("legacy-performance", 0, `["model"]`, "", ""), Options{})
	require.NoError(t, err)
	assert.False(t, plugin.PerformanceFailureFilter)
	assert.True(t, plugin.ShouldRecordPerformanceFailure(t.Context(), PerformanceFailureInput{}, PerformanceFailure{}))
}

func TestPerformanceFailureFilterOnlyFalseExcludesFailures(t *testing.T) {
	for _, test := range []struct {
		name, body string
		wantRecord bool
	}{
		{"false", "return false;", false},
		{"true", "return true;", true},
		{"null", "return null;", true},
		{"undefined", "return undefined;", true},
		{"string", `return "false";`, true},
		{"object", "return {record:false};", true},
		{"boxed false", "return new Boolean(false);", true},
		{"boxed true", "return new Boolean(true);", true},
		{"object getter", `return {get dangerous(){throw new Error("private getter");}};`, true},
		{"number", "return 0;", true},
		{"exception", `throw new Error("private-supplier-error");`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := routingTestPluginSource("performance-result", 0, `["model"]`,
				`requiredCapabilities:["task-performance-filter@1"],`,
				"export function shouldRecordPerformanceFailure(){"+test.body+"}")
			plugin, err := CompilePlugin(source, Options{})
			require.NoError(t, err)
			assert.Equal(t, test.wantRecord, plugin.ShouldRecordPerformanceFailure(t.Context(), PerformanceFailureInput{}, PerformanceFailure{}))
		})
	}
}

func TestPerformanceFailureFilterExposesOnlyDeclaredClassificationFields(t *testing.T) {
	var messages []string
	source := routingTestPluginSource("performance-context", 0, `["model"]`,
		`requiredCapabilities:["task-performance-filter@1"],`, `
export function shouldRecordPerformanceFailure(ctx, failure) {
  console.log(JSON.stringify({ctx, failure}));
  if (Object.keys(ctx).sort().join(",") !== "method,model,pluginKey,pluginVersion,requestPath,upstreamModel") return true;
  if (Object.keys(failure).sort().join(",") !== "errorCode,httpStatus,stage") return true;
  return !(ctx.pluginKey === "performance-context" && ctx.pluginVersion === "1.0.0" &&
    ctx.model === "client-alias" && ctx.upstreamModel === "mapped-model" &&
    ctx.requestPath === "/v1/systemone" && ctx.method === "POST" &&
    failure.stage === "http" && failure.httpStatus === 400 && failure.errorCode === "invalid_request");
}`)
	plugin, err := CompilePlugin(source, Options{Log: func(message string) { messages = append(messages, message) }})
	require.NoError(t, err)
	// 请求上下文可能带有私有数据，但性能钩子只能收到显式构造的分类字段。
	type privateContextKey string
	ctx := context.WithValue(t.Context(), privateContextKey("apiKey"), "private-channel-key")
	ctx = context.WithValue(ctx, privateContextKey("headers"), map[string]string{"Authorization": "private-bearer-token"})
	ctx = context.WithValue(ctx, privateContextKey("body"), "private-user-prompt")
	assert.False(t, plugin.ShouldRecordPerformanceFailure(ctx, PerformanceFailureInput{
		Model: "client-alias", UpstreamModel: "mapped-model", RequestPath: "/v1/systemone", Method: "POST",
	}, PerformanceFailure{Stage: "http", HTTPStatus: 400, ErrorCode: "invalid_request"}))
	require.Len(t, messages, 1)
	assert.NotContains(t, messages[0], "private-channel-key")
	assert.NotContains(t, messages[0], "private-bearer-token")
	assert.NotContains(t, messages[0], "private-user-prompt")
	assert.NotContains(t, messages[0], "Authorization")
}

func TestPerformanceFailureFilterRejectsUnboundedOrNonIdentifierErrorCodes(t *testing.T) {
	source := routingTestPluginSource("performance-error-code", 0, `["model"]`,
		`requiredCapabilities:["task-performance-filter@1"],`, `
export function shouldRecordPerformanceFailure(ctx, failure) {
  return failure.errorCode !== ctx.model;
}`)
	plugin, err := CompilePlugin(source, Options{})
	require.NoError(t, err)
	for _, test := range []struct {
		name, input, want string
	}{
		{"empty", "", ""},
		{"identifier characters", "Code_Error.Type-v1:400", "Code_Error.Type-v1:400"},
		{"maximum length", strings.Repeat("a", 128), strings.Repeat("a", 128)},
		{"too long", strings.Repeat("a", 129), "invalid_error_code"},
		{"non ascii", "上游错误", "invalid_error_code"},
		{"control character", "error\nprivate-token", "invalid_error_code"},
		{"url", "https://upstream.invalid/?key=private-token", "invalid_error_code"},
		{"json", `{"key":"private-token"}`, "invalid_error_code"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.False(t, plugin.ShouldRecordPerformanceFailure(t.Context(), PerformanceFailureInput{Model: test.want},
				PerformanceFailure{ErrorCode: test.input}))
		})
	}
}

func TestPerformanceFailureFilterBoundsHookExecution(t *testing.T) {
	source := routingTestPluginSource("performance-timeout", 0, `["model"]`,
		`requiredCapabilities:["task-performance-filter@1"],`, `
export function shouldRecordPerformanceFailure() { for (;;) {} }
export function stillUsable() { return "ok"; }`)
	plugin, err := CompilePlugin(source, Options{Timeout: 10 * time.Second, Concurrency: 1})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan bool, 1)
	go func() {
		result <- plugin.ShouldRecordPerformanceFailure(ctx, PerformanceFailureInput{}, PerformanceFailure{})
	}()
	select {
	case record := <-result:
		assert.True(t, record)
	case <-time.After(2 * time.Second):
		cancel()
		<-result
		t.Fatal("性能钩子没有在宿主预算内结束")
	}
	value, err := plugin.Engine.Call(t.Context(), "stillUsable")
	require.NoError(t, err)
	assert.Equal(t, "ok", value)
}

func TestPerformanceFailureFilterBoundsQueueAdmission(t *testing.T) {
	for _, shortRequestDeadline := range []bool{false, true} {
		name := "host budget"
		if shortRequestDeadline {
			name = "request deadline"
		}
		t.Run(name, func(t *testing.T) {
			occupied := make(chan struct{}, 1)
			var hookCalls atomic.Int32
			source := routingTestPluginSource("performance-admission", 0, `["model"]`,
				`requiredCapabilities:["task-performance-filter@1"],`, `
export function occupy() { console.log("occupied"); for (;;) {} }
export function shouldRecordPerformanceFailure() { console.log("filter_called"); return false; }`)
			plugin, err := CompilePlugin(source, Options{
				Timeout: 10 * time.Second, Concurrency: 1,
				Log: func(message string) {
					if strings.HasSuffix(message, " occupied") {
						occupied <- struct{}{}
					}
					if strings.HasSuffix(message, " filter_called") {
						hookCalls.Add(1)
					}
				},
			})
			require.NoError(t, err)
			busyCtx, cancelBusy := context.WithCancel(t.Context())
			busyDone := make(chan error, 1)
			go func() {
				_, callErr := plugin.Engine.Call(busyCtx, "occupy")
				busyDone <- callErr
			}()
			t.Cleanup(func() {
				cancelBusy()
				select {
				case callErr := <-busyDone:
					assert.Error(t, callErr)
				case <-time.After(2 * time.Second):
					t.Error("并发槽占用调用未在取消后结束")
				}
			})
			select {
			case <-occupied:
			case <-time.After(2 * time.Second):
				t.Fatal("无法建立实际占满的插件并发槽")
			}
			ctx, cancel := context.WithCancel(t.Context())
			if shortRequestDeadline {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 20*time.Millisecond)
			}
			defer cancel()
			result := make(chan bool, 1)
			go func() {
				result <- plugin.ShouldRecordPerformanceFailure(ctx, PerformanceFailureInput{}, PerformanceFailure{})
			}()
			select {
			case record := <-result:
				assert.True(t, record)
			case <-time.After(2 * time.Second):
				cancel()
				<-result
				t.Fatal("性能钩子排队没有遵守期限")
			}
			assert.Zero(t, hookCalls.Load(), "超时排队不得执行排除钩子")
			if shortRequestDeadline {
				assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			} else {
				assert.NoError(t, ctx.Err(), "宿主钩子预算不得取消客户端上下文")
			}
		})
	}
}

func TestPerformanceFailureFilterUsesPinnedPluginAfterRegistryReplacement(t *testing.T) {
	registry := NewRegistry()
	pinned, err := registry.Register(performanceFilterPluginSource, Options{})
	require.NoError(t, err)
	updated := strings.Replace(performanceFilterPluginSource, `version: "1.0.0"`, `version: "2.0.0"`, 1)
	updated = strings.Replace(updated, `if (failure.stage === "http") return false;`, `if (failure.stage === "http") return true;`, 1)
	newPlugin, err := registry.Register(updated, Options{})
	require.NoError(t, err)
	active, ok := registry.Get("performance-filter")
	require.True(t, ok)
	require.Same(t, newPlugin, active)
	require.NotSame(t, pinned, active)
	input := PerformanceFailureInput{Model: "model", RequestPath: "/v1/systemone", Method: "POST"}
	failure := PerformanceFailure{Stage: "http", HTTPStatus: 400, ErrorCode: "invalid_request"}
	assert.False(t, pinned.ShouldRecordPerformanceFailure(t.Context(), input, failure))
	assert.True(t, active.ShouldRecordPerformanceFailure(t.Context(), input, failure))
}
