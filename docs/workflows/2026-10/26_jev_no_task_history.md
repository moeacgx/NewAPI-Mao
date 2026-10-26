# TypeSafe Jev 不写入任务记录

## 目标与范围

TypeSafe Jev 是原生同步模型。成功请求不应在 `model.Task` 中留下任务记录，
同时保留现有的预扣、按实际用量结算、消费日志、渠道指标和模型广场统计。

仅处理原生 `typesafe` 或历史 `cloudflare-jev` 插件路由声明 `retainResult:false` 且返回同步成功的请求。
异步任务、同步失败和其他任务插件仍按原有合同持久化；不修改成功消费日志、费用、
性能采样、公开响应、插件源码、价格配置或数据库结构。已有任务记录不删除。

## 实现方案

- 使用固定的 TypeSafe 插件 key、历史 Cloudflare Jev key 和原生路由的即时成功状态识别 Jev，避免依赖用户模型别名。
- 在写入 `model.Task` 前跳过该模型的任务行。
- 保持请求成功后的结算、`LogTaskConsumption`、渠道指标和模型广场采样链路。
- 原生结果受理后，结算失败不能触发提交失败退款。独立错误日志保存请求、用户、令牌、渠道、模型、
  插件版本/hash、预扣与实际应收额度、资金来源及用量，标记 `settlement_pending`。
  资金和令牌可能仅部分结算，需按证据核对，不承诺自动恢复；不保存答案、密钥或原始异常。
- 更新插件开发文档、技能说明和集成回归，明确任务表与消费日志的区别。

## 验证计划

- 扩展 `TestJevNativeIntegration`：成功和结算失败后均无 Jev `model.Task` 行；
  成功场景验证用户/令牌余额及消费日志，包含真实用量字段。
- 断言上游失败仍不写任务行并按现有会话退款；其他任务插件的持久化行为保持。
- 运行定向 router/controller 测试、相关 `go vet` 和 `git diff --check`；
  不调用真实供应商、不访问生产环境、不部署。

## 验证结果

已完成本地 SQLite + 模拟上游验证：Jev 同步成功、上游失败和结算失败均不新增 `model.Task` 行；
成功仍完成 21 quota 结算并写入 1000 个输入 Token 的消费日志；上游失败仍保持原有预扣退款；
结算失败返回 500 且保留 1376 quota 预扣。未调用真实供应商、未访问生产、未部署。

- `TestJevTaskHistoryPersistenceBoundary`：官方 TypeSafe、Cloudflare Jev 与别名零任务行；
  其他插件、异步、失败、保留结果路由和通用任务接口仍持久化。
- `TestNativeTaskActualTokenLogs`：固定按次/表达式的实际 Token、余额和模型广场样本回归通过。
- `TestTaskPluginSubmitPinnedPollingAndRefund`：Sora 持久版本、异步轮询和一次退款通过。
- `TestJevNativeIntegration`：真实 TokenAuth、同步响应、消费日志版本 hash、请求关联待核对审计和停用门禁通过。
- 定向测试统一使用 `-timeout=60s`；`go vet -mod=readonly ./controller ./router` 通过。
- Go 格式化、新增五处文档链接与 `git diff --check` 通过。现有 oxfmt 缺 Windows 原生绑定，
  改用仓库 Classic 已安装的 Prettier 3.4.2 完成 Markdown 格式化和检查，无依赖变更。

本次为共享后端写入行为，Default 和 Classic 无需页面修改；新增 Jev 行不会进入任一模板的任务列表。
API 路径、请求/响应、模型别名、认证和计费价格不变，已存在任务仍可按原权限查询。
MySQL/PostgreSQL、订阅资金和真实供应商未进行现场验证；没有修改 SQL 或数据库结构。
