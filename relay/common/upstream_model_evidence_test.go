package common

import (
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexResponseHeaderEvidencePreservesBodyModel(t *testing.T) {
	info := &RelayInfo{}
	var observations []UpstreamModelEvidence
	info.OnUpstreamModelEvidence = func(_ *RelayInfo, evidence UpstreamModelEvidence) {
		observations = append(observations, evidence)
	}
	info.CaptureCodexResponseHeaders(&http.Response{StatusCode: http.StatusOK, Header: http.Header{
		"X-Codex-Safety-Buffering-Faster-Model": {" gpt-5.6-luna "},
	}})
	assert.Empty(t, info.UpstreamResponseModelName)
	info.SetUpstreamResponseModelName("gpt-5.6-sol")
	assert.Equal(t, "gpt-5.6-sol", info.UpstreamResponseModelName)
	assert.Equal(t, "gpt-5.6-luna", info.CodexFasterModelName)
	assert.Equal(t, []UpstreamModelEvidence{
		{Model: "gpt-5.6-luna", Source: constant.UpstreamModelSourceCodexFasterModel},
		{Model: "gpt-5.6-sol", Source: constant.UpstreamModelSourceResponseBody},
	}, observations)
}

func TestCodexResponseHeaderEvidenceIgnoresAmbiguousOrMissingSignals(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []string
		status int
	}{
		{name: "缺失"}, {name: "空值", values: []string{" "}},
		{name: "重复响应头", values: []string{"model-a", "model-b"}},
		{name: "合并值", values: []string{"model-a, model-b"}},
		{name: "内部空白", values: []string{"model a"}},
		{name: "控制字符", values: []string{"model\x00a"}},
		{name: "无效字符", values: []string{"model\xff"}},
		{name: "超长", values: []string{strings.Repeat("a", 256)}},
		{name: "错误响应", values: []string{"other"}, status: http.StatusTooManyRequests},
		{name: "重定向", values: []string{"other"}, status: http.StatusTemporaryRedirect},
	} {
		t.Run(test.name, func(t *testing.T) {
			status := test.status
			if status == 0 {
				status = http.StatusOK
			}
			info := &RelayInfo{CodexFasterModelName: "previous"}
			info.OnUpstreamModelEvidence = func(_ *RelayInfo, _ UpstreamModelEvidence) {
				t.Fatal("无有效模型证据时不应触发守卫")
			}
			info.CaptureCodexResponseHeaders(&http.Response{StatusCode: status, Header: http.Header{
				"X-Codex-Safety-Buffering-Faster-Model": test.values,
				"X-Codex-Safety-Buffering-Enabled":      {"true"},
				"X-Codex-Turn-State":                    {"opaque-content"},
				"X-Codex-Plan-Type":                     {"plus"},
			}})
			require.Empty(t, info.CodexFasterModelName)
		})
	}
}
