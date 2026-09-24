# 官方 JS 任务插件后端

开发或排障前先使用[任务插件开发与验收技能](../../.agents/skills/task-plugin-development/SKILL.md)，
按当前宿主能力选择官方参考并逐项验证权限、计费、实际用量与日志；下文的历史阶段记录不能代替当前版本验证。

## 范围

初始阶段固定上游 `9fe0457ee`，架构来源 `eb48396d5`，基线 `5d90556e2`。
2026-09-23 从固定上游 `d04c118c8803f49e0c9bab74dcf5b5efeab9464a` 补齐原生 JSON submit
分发、同步呈现和相应用量计费；来源、范围与验证见[同步记录](../workflows/2026-09/23_task_plugin_native_sync.md)。
保留 `/extensions`、原生适配器、AtlasCloud=61、源任务分组授权、退款认领和 quota 契约。
官方插件使用独立渠道编号 62，默认关闭，不自动接管原生渠道。

## 管理接口合同

响应沿用 `{success, message, data}`。管理员读取，状态写入要求 Root。

| 接口                                             | 权限  | 行为                                               |
| ------------------------------------------------ | ----- | -------------------------------------------------- |
| `GET /api/plugin/task`                           | Admin | 每个 key 一项，优先 active，否则最近归档版本       |
| `GET /api/plugin/task/:key?version=...`          | Admin | 不指定版本时同列表规则；仅 Root 返回 source        |
| `GET /api/plugin/task/:key/versions`             | Admin | 历史版本，不返回源码                               |
| `GET /api/plugin/task/runtime/status`            | Admin | 总开关、编号及能力限制                             |
| `PUT /api/plugin/task/runtime/status`            | Root  | `{enabled: boolean}`，更新 TaskPluginEnabled       |
| `POST /api/plugin/task/:key/activate`            | Root  | `{version: string}`，激活并启用                    |
| `POST /api/plugin/task/:key/status`              | Root  | `{enabled: boolean}`，修改已激活版本状态           |
| `GET /api/task_plugin_options`                   | Admin | key/name/version/models/channel_type               |
| `POST /api/plugin/task`                          | Root  | JSON `{source}`，编译校验后保存自定义版本（1 MiB） |
| `DELETE /api/plugin/task/:key/versions/:version` | Root  | 删除无历史引用的非活动版本                         |
| `GET /api/plugin/task/marketplace/sources`       | Admin | 读取多个索引源；显式空数组不回退默认值             |
| `PUT /api/plugin/task/marketplace/sources`       | Root  | 保存 `{name,index_url}` 裸数组，最多 16 项         |

列表和详情包含 `key/version/api_version/source_hash/enabled/active/source_kind`。
`meta` 为服务端解析的完整官方元数据，包括 name/key/version/apiVersion/models，以及存在时的
description/author/protocols/routes/usageSchema。元数据声明不代表宿主已经开放相应协议。
`channel_count` 为启用且绑定该 key 的 62 类渠道数，`in_flight_count` 为该 key 非终态任务数。
数据库失败返回错误，不伪造零值。

首次安装的内置版本为 active=false、enabled=false，详情仍可读取。先 activate，再按需 status。
运行入口必须同时满足总开关开启和版本 active/enabled。关闭总开关仅阻新提交，历史 pin 继续读取和轮询。
渠道沿用现有 sensitive_write 权限，绑定字段为 `setting.task_plugin_key`，不新增专属权限。
非 62 类渠道不能带绑定，62 类必须绑定已知插件。AtlasCloud 始终是 61。
自定义源码上传仅接受 JSON source，由宿主编译校验并以 disabled/inactive 保存；重复 key/version
必须源码 hash 相同，否则冲突。删除仅允许无历史任务引用的非活动版本；活动版本必须先切换，
避免影响在途任务。上传、删除和激活均由 Root 操作并写入管理审计。
市场支持多个 HTTPS 索引源，默认 NewAPI 官方源与独立公开仓库 `moeacgx/maolaonewapi-plugins`。
Admin 读取源和索引，Root 管理源、预览并安装；浏览器不携带网关凭据、不跟随重定向。
安装提交源码、SHA-256、预期 key/version 和 marketplace 来源对象，宿主校验后保存为禁用、未激活版本。
移除源不删除已装版本，来源记录仅用于追溯，不等于发布者签名。详见[多源契约与发布](../workflows/2026-09/16_task_plugin_sources_plan.md)。
发布者签名、S3 和匿名签名资源尚未开放；在线试运行仍返回 501。

## 运行与安全

显式 JSON 提交入口为 `POST /v1/task/plugins/:plugin_key`，通过本地令牌认证、
分组/模型限制、渠道选择、提示词审计和 quota 预扣结算。
活动插件也可通过 `meta.routes` 声明原生 JSON submit 路径，例如官方 TypeSafe 的
`POST /typesafe/v1/systemone`。原生入口复用同一权限、审计、渠道和资金链路，成功直接调用
插件 `native[render]` 返回结构化结果。原生 query/dynamic 和 decoder 返回的 `originTaskIds`
目前返回 501；通用任务读取入口保持既有权限合同。
`originTaskIds` 最多 16 个，逐个校验用户、分组、模型和渠道能力；
多源必须同渠道、同分组、同源码版本，暂不支持跨版本引用。

任务私有字段 `execution.task_plugin` 保存 key/version/API version/source hash/source kind，
`plugin_state` 保存受大小限制的状态，`plugin_data` 保存原始响应，`plugin_immediate` 保存即时终态，
`plugin_result_url` 仅保存内联资源。供应商凭据和原始响应不放入公开 Data。
内置源码归档到独立 task_plugins 表。
历史轮询按 key/version/hash 读取并校验源码；缺失时禁止回退当前版本。
源码 hash 是完整性校验，不是发布者签名。

普通任务继续支持本地显式按次价格。声明 `retainResult:false` 的原生同步提交另外支持
官方 `u()` 用量表达式：以估算用量预扣，按实际成功用量结算，任务表达式单位是美元。
不把 UsageFacts 当作 OtherRatios 连乘；既有聊天公式仍以每百万 token 计价。
表达式计费的非同步结果拒绝，详见[用量计费](../workflows/2026-09/23_task_plugin_usage_billing.md)。
组合资金来源暂不支持，已预扣时沿用本地会话退款；本阶段已验证钱包，订阅保留原链但尚未组合验收。
终态沿用本地 CAS、退款认领、失败恢复和补偿；审计管理员信息与 Root 源码归因分开显示。

通用提交解析完成后先保存任务，再结算并返回公开 ID；原生同步提交返回插件格式的结果。
原生同步成功直接持久化终态，不再轮询；`retainResult:false` 保留账务行，但不保留响应数据、
插件状态或上游密钥，并令通用查询和资源端点返回 404。按次异步结果遵循官方合同保留轮询数据。
原生路径在落库前通过 `Billing.Reserve` 补足实际费用；落库后的结算故障返回 500，保留预扣待对账，
不误触发提交失败的全额退款。上游成功但数据库写入失败时返回错误且不自动重试，
沿用预扣会话退款；供应商端可能留下未登记任务，尚无跨系统事务或补录队列，不应作为生产就绪证明。
插件提交发生错误时不自动重试，以避免不确定的上游受理状态造成重复任务。

消费日志、渠道指标和模型广场 `perf_metrics` 是三条独立统计链路。原生同步 `retainResult:false`
只有在真实请求已发送且返回同步终态后才写模型广场样本：成功使用实际输出 Token，失败经过内容策略及
失败过滤规则，客户端取消、预扣/解码/宿主持久化/结算错误不计为供应商失败。非流式同步任务的延迟是
提交处理到返回前的整体耗时，TTFT 固定为 0，TPS 使用输出 Token 除以该耗时；渠道指标仍单独记录。
上游错误的稳定分类码会保留给性能过滤，但不会进入客户端错误响应。

数据库和缓存选渠都在优先级、权重及并发选择前隔离 62 与目标 key；亲和性和指定渠道复查同一规则。
常规 HTTP 和原生 Go 入口排除 62；插件原生路径仅选择绑定目标 key 的 62 类渠道。
原生 Go 适配器及 AtlasCloud 不变。
轮询 fetch/read/parse/unknown-status 只传播稳定分类、公开 ID、key/version 和 HTTP 状态；
不包装供应商异常原文。禁用总开关和版本不影响已保存快照的轮询。

资源只允许已认证用户读取已存任务的有限内联媒体，远程资源返回 501。
不签发匿名 URL，不承诺供应商 URL 过期后的归档读取。
JS 有执行限时与并发限制，但没有进程级内存硬隔离；当前不宣称生产可用。

公开读取接口为 `GET /v1/task/plugins/:plugin_key/:task_id`，资源列表及下载为
`GET /v1/task/plugins/:plugin_key/:task_id/artifacts` 和其 `/:artifact_key` 子路径。
目前仅提供 `result` 内联资源，要求任务成功，Base64 解码后不超过 8 MiB；
只允许 PNG/JPEG/WebP/GIF、MP4/WebM、MPEG/WAV/OGG，拒绝 SVG，响应为 no-store、nosniff、attachment。
供应商返回数据本身另有 1 MiB 上限，因此实际可保存的内联资源也受其限制。

## 迁移与回滚

新增 task_plugins 表和私有任务 JSON 字段，不改历史平台及渠道编号。
源码列容纳 1 MiB，迁移须验证 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。
上游数据导入必须确认来源：官方 type=61 显式转换到 62；本地 61 保留 AtlasCloud。
禁止只按编号批量重写，当前没有自动导入和生产数据库操作。

回滚先关闭总开关，保留当前二进制完成历史轮询与退款，排空插件任务后才能回退。
保留版本源码、任务及审计，不删除扩展、不做破坏性逆迁移。
旧二进制不识别插件平台和版本快照，不能在有在途任务时直接回退。

## 验证与边界

已通过本地验证（2026-09-15，SQLite 与假 HTTP 上游，不调用真实供应商）：

- `TestTaskPluginManagementContractAndPermissions`：Admin/Root、首次未激活但 enabled 的拒绝、详情元数据。
- `TestTaskPluginMixedChannelRealDistribution`：数据库及缓存下 61/目标 62/其他 key 62 同模型并存；真实 Gin 分发和指定错渠拒绝。
- `TestPluginSourceTaskAuthorizationUsesPersistedGroups`：用户、分组撤权、令牌模型、渠道能力和版本隔离。
- `TestTaskPluginSubmitPinnedPollingAndRefund`：内置 Sora 提交、钱包与令牌预扣、版本切换及禁用后轮询、重复轮询仅退款一次、历史源码篡改拒绝。
- `TestTaskPluginPollingErrorsDoNotExposeProviderPayload`：内置 Hailuo 抛出带假凭据的异常，返回值与日志无泄漏。
- `TestTaskPluginPollingErrorBoundariesRedactPayload`：fetch/read/parse/unknown-status 四个传播边界。
- `TestTaskPluginPublicViewAndAuthenticatedInlineResources`：公开信息最小化、匿名/跨用户/撤权拒绝、内联媒体及远程/SVG 拒绝。
- `go test -mod=readonly -timeout=60s ./pkg/jsplugin ./plugins ./relay/channel/task/jsplugin -count=1`：离线运行时和内置插件契约通过。
- 根模块定向任务、退款、计费、源任务和选渠回归通过；相关包 `go vet -mod=readonly` 通过。
- `cd relaykit` 后 `GOWORK=off go build ./...` 和 `go test -timeout=60s ./... -count=1` 通过。
- Markdown 使用仓库相同版本 `oxfmt@0.57.0` 格式化；本专题、本次新增索引链接和 `git diff --check` 通过。
  全索引检查发现既有 `../workflows/2026-08/15_topup_payment_fee_included_hint.md` 断链，本工作项未修复。

根目录 `go build -mod=readonly ./...` 被缺失的 `web/classic/dist` 嵌入产物阻断；
本工作项禁止写入两套前端，未生成占位产物，因此不声称完整应用二进制已构建。

2026-09-15 的阶段验证不包含原生同步链路；2026-09-23 的 TokenAuth、模拟 TypeSafe 与用量计费
组合证据见同步记录。仍未验收：MySQL/PostgreSQL 现场迁移、订阅退款及故障恢复组合、
真实供应商、远程媒体、S3、匿名短期签名及撤销、在线试运行、异步 UsageFacts、Responses 协议外观。
十个内置源码与固定上游一致，但仅 Sora 提交退款和 Hailuo 异常传播做了宿主完整组合验证；
其他供应商只证明离线脚本契约可运行。未移植依赖未接入 Responses 外观的上游测试。
Default 与 Classic 由各自工作树实现，本工作项仅修改后端和开发文档。
