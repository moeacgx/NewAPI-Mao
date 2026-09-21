# xAI 渠道兼容 Anthropic Messages

## 问题与官方基线

Grok 经 xAI 渠道处理 `/v1/messages` 时，本地 `ConvertClaudeRequest` 直接返回 `not available`，尚未发送上游请求。
官方 `QuantumNous/new-api` 的 `main` 在核查时为 `9c293e8c02371bda844af79e3500ff2d516d1dda`，同样尚未实现此转换。
这个结论限于常规 Messages 转 Chat 路径；显式强制 Responses 使用独立桥接路径。

## 目标与实现边界

- 复用现有 Claude → OpenAI 转换器，再应用 xAI 的模型后缀、搜索和推理配置；保留可表达的显式零值。
- 仅已转换的 Claude 请求改投 `/v1/chat/completions`，记录 Claude → OpenAI 转换链；客户端路径保留 `/v1/messages`。
- JSON 与 SSE 返回复用现有 OpenAI → Claude 转换器，保留文本、推理、工具调用和终止事件。
- xAI usage 继续先归一化缓存字段、逐字段合并 SSE usage；最后才输出 Claude 终止用量，结算保留 OpenAI 总输入口径。
- 原生 OpenAI、Responses、图片、强制 Responses 及请求体透传沿用现有选择规则，不通过 Grok 模型名称全局改路由。
- 不增加配置、权限、数据库迁移或前端入口；Default、Classic 均通过同一后端获得兼容行为。
- 入口测试发现共享 Claude → Chat 转换器遗漏 `tool_choice`，补齐 `auto` / `any` / `tool` / `none` 与显式并行调用标志。该修复也适用于其他复用此转换器的渠道及 Claude → Chat → Responses 桥接。

## 测试计划

请求与 HTTP 入口覆盖普通/流式、系统提示词、工具、显式零值、模型映射、搜索后缀、推理后缀、参数覆盖以及透传/强制 Responses 的优先级。
响应覆盖 JSON/SSE、文本/推理/工具、分段或缺失 usage、缓存优先级、模型证据与流式错误。
运行相关 Go 包测试（单次 `-timeout 60s`）、静态检查、根模块构建及 `relaykit` 的 `GOWORK=off go build ./...`。

## 验证与发布边界

线上只读精确查询确认所报请求命中 xAI 渠道、未开启强制 Responses，记录为 `convert_request_failed` / HTTP 500 / `not available`，请求路径为 `/v1/messages`，未扣费。与适配器缺口一致。
PR #256 已合并。`go test ./relay/... -count=1 -timeout 60s`、relaykit 转换器测试与独立构建通过；
CI `35589322338` 后端 vet/build/test、前端检查及构建通过。首次后端 CI 因既有 Realtime
测试清理时序失败，同提交重跑通过；CodeRabbit 本次跳过审查，不代表实质审查通过。
后续经授权发布 .332，部署记录见[maolaoapi .332 发布](21_maolaoapi_grok_332_deployment.md)。
协议兼容使用模拟上游验证，尚未通过真实供应商请求验收。
