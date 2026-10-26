# 网关 Skills Chat 私有流式响应兼容修复

## 现象与根因复核

2026-10-27，线上 `.355` 节点在 `POST /v1/chat/completions` 的技能读取阶段记录
`技能读取阶段未返回唯一助手消息` 并返回 502。实际运行镜像为
`ghcr.io/moeacgx/newapi-mao@sha256:62fdd34becf5d62ff2444b21bbaa797081be4bc891b78271248cef9d174f5e12`，
版本 `v1.0.0-rc.10.1.10.355`。

同请求的 discovery 消费日志记录 `is_stream=true`。原错误同时覆盖 JSON 解码失败、
空候选和多候选，单凭错误文字不能认定上游返回多个候选。此前多候选归因和修复已撤回。
源码确认：私有请求设置 `stream=false`，但兼容转发会根据上游 `text/event-stream`
切换流式响应；私有 writer 捕获 SSE 后直接交给 JSON 解析器，产生上述 502。

## 修改

- 仅对私有 Chat SSE 聚合助手正文、推理及按索引分片的工具调用，再复用既有协议校验。
- 保留单一助手候选约束，拒绝畸形、截断、输出限长及不安全的工具调用。
- 隔离私有 SSE 响应头状态，确保正式流式响应重新设置正确响应头。
- 沿用中继的正常 EOF 语义：已收到 `stop/tool_calls` 完整结束消息后，允许上游正常 EOF；
  中继补出的 `[DONE]` 不作为原始上游完整性证据。缺少正常 finish、扫描异常或上下文失败仍拒绝。

## 验证

- 先构造上游忽略 `stream=false` 返回 SSE 的完整 relay 回归，再修复。
- 验证实际技能正文读取、客户端工具和流式配置恢复、私有响应隔离、逐轮结算及失败退款边界。
- `go test ./service ./controller -run TestGatewaySkill -count=1 -timeout=60s`：197 项通过。
- `go vet ./service ./controller`、修改 Markdown 的 Prettier 检查、`git diff --check` 通过。
- 独立只读复核通过；未重放真实付费请求，不以模拟测试冒充真实供应商验收。

## 兼容性与限制

该调整只影响 Skills 私有 Chat 发现阶段；正式生成仍使用客户端原始请求。
不改变 Responses/Claude/Gemini 私有解析、请求鉴权、技能授权或用量计费规则。
