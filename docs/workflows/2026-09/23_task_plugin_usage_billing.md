# 原生同步任务的用量表达式计费

## 目标与来源

从固定上游 `d04c118c8803f49e0c9bab74dcf5b5efeab9464a` 移植 `u()`、任务美元单位转换、用量快照与同步完成结算。不得把用量事实当作附加倍率相乘，不为 Jev 固定模型价格。

## 范围与契约

仅为声明 `retainResult: false` 的原生同步任务路由开放表达式；其他任务保留显式按次计费。提交前冻结表达式、分组倍率和经过适配器校验的估算用量；成功的 immediate 结果合并真实用量后重新求值。失败由原有预扣会话退款；非同步结果拒绝，不伪装为成功任务。钱包、订阅与令牌扣费继续使用现有账务链。

任务公式 `u("字段名")` 读取用量事实，其结果单位为美元，换算为 `结果 × QuotaPerUnit × 分组倍率`；聊天公式 `p/c` 继续保持每百万 token 单价语义。所有额度转换仍使用公共饱和/严格转换函数，实际结算饱和写入管理员日志。

## 管理员配置

通过已有选项 API `PUT /api/option/` 设置 `billing_setting.billing_mode` 和 `billing_setting.billing_expr` 的 JSON 字符串值。映射以客户端模型名为键，例如 `jev-1.13.0`，模式为 `tiered_expr`；表达式可用 `u("input_tokens") * 单个输入token美元价格`。价格须由管理员依据供应商账单和站点售价填写。更新整个映射前先读取现有值并合并，避免覆盖其他模型。无默认 Jev 价格，也不导入插件专属覆盖配置。

## 安全与兼容

保留插件渠道鉴权、历史源码 pin、明确失败退款和旧按次价格；不持久化用户请求正文。表达式只取宿主校验后的插件用量，缺失事实或非法结果不会形成负收费。异步任务 UsageFacts 的持久化与轮询结算、Responses 协议、两套前端价格编辑器不在本次范围。

## 验证计划

验证 u() 的数值/枚举/缺失语义，任务与聊天换算单位隔离，实际用量覆盖估算但不修改原快照，零用量、负费用与饱和边界，以及旧按次插件兼容。测试使用确定输入与 60 秒超时，不调用真实供应商或生产环境。

## 已执行验证

- `go test -mod=readonly -timeout=60s ./pkg/billingexpr ./setting/billing_setting -count=1` 通过，包含既有聊天公式与新用量读取、单位隔离、schema契约。
- `go test -mod=readonly -timeout=60s ./service -run TestEvaluateTaskCompletionUsage -count=1` 通过，覆盖冻结合同、实际覆盖估算、零费用、非法费用和超额饱和。
- `go test -mod=readonly -timeout=60s ./relay -run 'TestRemixInheritedRatiosSurvivePriceRebuildAndRetry|TestMigrationKeepsNativeDispatch|TestPluginSourceTaskAuthorizationUsesPersistedGroups' -count=1` 通过，覆盖原有倍率重建、原生渠道分发和源任务授权。
- 定向验证为本地执行；尚未据此宣称真实供应商、真实生产计费或异步用量链路已验收。完整原生路由集成结果统一记录于主工作项。
