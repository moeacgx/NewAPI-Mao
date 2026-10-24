package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
)

var gatewaySkillBenchmarkBody []byte

// 对比八轮真实请求构造的分配；无时长阈值，stateless 使用保留的无状态 API。
func BenchmarkGatewaySkillRequestContinuation(b *testing.B) {
	for _, fixture := range gatewaySkillRequestFixtures() {
		for _, historyBytes := range []int{16 * 1024, 1024 * 1024} {
			b.Run(fmt.Sprintf("%s/%dKiB", fixture.name, historyBytes/1024), func(b *testing.B) {
				message := map[string]any{"role": "user", "content": strings.Repeat("x", historyBytes)}
				if fixture.format == types.RelayFormatGemini {
					message = map[string]any{"role": "user", "parts": []map[string]any{{"text": strings.Repeat("x", historyBytes)}}}
				}
				body, err := common.Marshal(map[string]any{"model": "test", fixture.historyKey: []map[string]any{message}})
				if err != nil {
					b.Fatal(err)
				}
				turns := make([]*GatewaySkillTurn, 8)
				results := make([][]GatewaySkillResult, len(turns))
				for round := range turns {
					response := strings.ReplaceAll(fixture.response, "call_page", fmt.Sprintf("call_%d", round))
					turns[round], err = ParseGatewaySkillTurn([]byte(response), fixture.format, nil)
					if err != nil {
						b.Fatal(err)
					}
					results[round] = []GatewaySkillResult{{Call: turns[round].Calls[0], Content: json.RawMessage(fmt.Sprintf(`{"content":"selected-page","next_offset":%d}`, (round+1)*24576))}}
				}
				b.Run("stateful", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(body)))
					for range b.N {
						request, err := NewGatewaySkillRequest(body, fixture.format, "test", "")
						if err != nil {
							b.Fatal(err)
						}
						gatewaySkillBenchmarkBody = request.Bytes()
						for round := range turns {
							if err := request.Continue(turns[round], results[round]); err != nil {
								b.Fatal(err)
							}
							gatewaySkillBenchmarkBody = request.Bytes()
						}
					}
				})
				b.Run("stateless", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(body)))
					for range b.N {
						request, err := PrepareGatewaySkillRequest(body, fixture.format, "test", "")
						if err != nil {
							b.Fatal(err)
						}
						for round := range turns {
							request, err = ContinueGatewaySkillRequest(request, fixture.format, turns[round], results[round])
							if err != nil {
								b.Fatal(err)
							}
						}
						gatewaySkillBenchmarkBody = request
					}
				})
			})
		}
	}
}
