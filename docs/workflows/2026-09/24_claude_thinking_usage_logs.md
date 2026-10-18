# Claude 思考参数日志恢复

## 目标与根因

恢复 `.243` 中已有、在 `.244` 官方集成提交 `d182efadc` 被删除的 Claude
思考参数日志能力，并补齐透传请求。旧实现将日志展示值写入中继的
`ReasoningEffort`；当前该字段也参与协议转换，不能再写入 `thinking:预算`。

## 范围与契约

- 原生 Messages 和带 `thinking` 的 OpenAI 兼容请求采用独立日志快照。
- 明确的 effort 优先；否则显示 `thinking:预算`、`thinking`、`adaptive` 或
  `disabled`。只有 `enabled` 使用预算生成展示值，`adaptive/disabled` 保留状态，
  不把预算猜测成 `low/high/max`。
- 保留 `other.reasoning_effort` 展示契约，补充 `thinking_type` 与
  `thinking_budget_tokens` 供审计；不记录提示词或思考正文。
- 透传记录入站参数且不改变请求字节；普通请求记录覆盖后的出站参数，删除参数时
  清除快照，重试按新请求重新采集，避免上一渠道的数据残留。
- OpenRouter 将 `thinking` 转为 `reasoning.max_tokens` 时保留实际预算日志；
  Claude 与 Responses 的互转在最终请求封装前采集。未带 `thinking` 的普通
  OpenAI 请求维持现有等级日志行为，不扩大为通用 OpenRouter 参数采集。
- 不改变 `BillingRequestInput`、预扣费或结算。`output_config.effort == max`
  条件仍匹配原始请求，不将 `thinking` 自动视作 `max`。
- Classic 与 Default 已支持非空 `other.reasoning_effort`，复用现有渲染，无前端改动。
- 无数据库迁移；部署前历史日志不会补写。回滚只影响后续日志的记录能力。

## 验证计划

验证预算、adaptive、disabled、明确等级优先、OpenAI 兼容参数、全局/渠道透传
字节一致、覆盖修改/删除、跨渠道重试，以及同一 max 规则的实际结算倍率。
执行受影响 Go 包回归（每次测试超时 60 秒）、Markdown 格式与链接检查、差异检查。

## 验证结果

- 已先运行新增日志回归，确认旧实现缺失预算、状态和透传等级，再实现修复。
- `go test ./relay/common ./service ./relay/channel/openai ./relay/channel/claude ./relay/helper -count=1 -timeout 60s` 全部通过。
- 原始 `output_config.effort=max` 的冻结请求按 3 倍结算；仅预算请求按 1 倍，
  出站日志参数被删除不会改变原始计费输入。
- `go build ./relay/... ./service` 通过。
- `go test ./relay -count=1 -timeout 60s` 全包通过，新增 10 个真实 handler
  模拟上游案例验证出站正文与日志一致，包含全局/渠道透传逐字节比较、预算覆盖、
  等级删除、OpenRouter 预算转换、Claude 强制 Responses 与 Responses 转 Claude。
- 跨协议测试保留转换器既有能力边界：最终出站不含思考参数时日志不再显示入站等级；
  不新增转换器尚不支持的等级映射。日志表示显式出站配置，不能证明模型内部实际思考量。
- 两份 Markdown 的 Prettier 格式、新增索引链接、`git diff --check` 及新增文件空白检查通过。
- 本地验证使用模拟请求，无真实供应商或生产部署验证。根应用构建需另行准备两套
  前端嵌入产物；本次不修改前端，未执行前端构建。

## 2026-09-25 集成复核

- 从 `origin/custom-main` 的 `ca5d5ebbd` 移植原独立工作区实现和全部回归，原工作区保持不动。
- 提交前同步至 `512910188`，保留 PR #277 的充值日志隐私过滤。
- 与 xAI 缓存分片修复组合验证；日志快照仅影响展示，`ReasoningEffort` 和冻结的
  `BillingRequestInput` 仍各自承担协议转换和计费职责。
- 在同步后的基线上，`go test ./relay/... ./service -count=1 -timeout 60s`、
  `go vet ./relay/... ./service`、`go build ./relay/... ./service` 均通过。
  Markdown 格式、新增链接与 `git diff --check` 通过；本地未调用真实供应商。
