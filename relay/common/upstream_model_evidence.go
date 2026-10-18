package common

import (
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/constant"
)

// UpstreamModelEvidence 保留上游模型声明的来源，不承诺识别底层模型身份。
type UpstreamModelEvidence struct {
	Model  string
	Source string
}

// CaptureCodexResponseHeaders 只观察上游成功响应中的明确模型声明。
// 缓冲标志、套餐和 turn-state 长度不能证明模型替换，不参与守卫判定。
func (info *RelayInfo) CaptureCodexResponseHeaders(resp *http.Response) {
	if info == nil {
		return
	}
	info.CodexFasterModelName = ""
	if resp == nil || resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return
	}
	values := resp.Header.Values("X-Codex-Safety-Buffering-Faster-Model")
	if len(values) != 1 {
		return
	}
	name := strings.TrimSpace(values[0])
	if name == "" || len(name) > 255 || !utf8.ValidString(name) {
		return
	}
	for _, character := range name {
		if character == ',' || unicode.IsSpace(character) || unicode.IsControl(character) {
			return
		}
	}
	info.CodexFasterModelName = name
	if info.OnUpstreamModelEvidence != nil {
		info.OnUpstreamModelEvidence(info, UpstreamModelEvidence{Model: name, Source: constant.UpstreamModelSourceCodexFasterModel})
	}
}
