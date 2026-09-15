# 官方 JS 任务插件后端

## 范围

固定上游 `9fe0457ee`，架构来源 `eb48396d5`，基线 `5d90556e2`。
保留 `/extensions`、原生适配器、AtlasCloud=61、源任务分组授权、退款认领和 quota 契约。
官方插件使用独立渠道编号 62，默认关闭，不自动接管原生渠道。

## 管理接口合同

响应沿用 `{success, message, data}`。管理员读取，状态写入要求 Root。

| 接口                                    | 权限  | 行为                                         |
| --------------------------------------- | ----- | -------------------------------------------- |
| `GET /api/plugin/task`                  | Admin | 每个 key 一项，优先 active，否则最近归档版本 |
| `GET /api/plugin/task/:key?version=...` | Admin | 不指定版本时同列表规则；仅 Root 返回 source  |
| `GET /api/plugin/task/:key/versions`    | Admin | 历史版本，不返回源码                         |
| `GET /api/plugin/task/runtime/status`   | Admin | 总开关、编号及能力限制                       |
| `PUT /api/plugin/task/runtime/status`   | Root  | `{enabled: boolean}`，更新 TaskPluginEnabled |
| `POST /api/plugin/task/:key/activate`   | Root  | `{version: string}`，激活并启用              |
| `POST /api/plugin/task/:key/status`     | Root  | `{enabled: boolean}`，修改已激活版本状态     |
| `GET /api/task_plugin_options`          | Admin | key/name/version/models/channel_type         |

列表和详情包含 `key/version/api_version/source_hash/enabled/active/source_kind`。
`meta` 为服务端解析的完整官方元数据，包括 name/key/version/apiVersion/models，以及存在时的
description/author/protocols/routes/usageSchema。元数据声明不代表宿主已经开放相应协议。
`channel_count` 为启用且绑定该 key 的 62 类渠道数，`in_flight_count` 为该 key 非终态任务数。
数据库失败返回错误，不伪造零值。

首次安装的内置版本为 active=false、enabled=false，详情仍可读取。先 activate，再按需 status。
运行入口必须同时满足总开关开启和版本 active/enabled。关闭总开关仅阻新提交，历史 pin 继续读取和轮询。
渠道沿用现有 sensitive_write 权限，绑定字段为 `setting.task_plugin_key`，不新增专属权限。
非 62 类渠道不能带绑定，62 类必须绑定已知插件。AtlasCloud 始终是 61。
上传、删除版本和在线试运行返回 501；市场、S3 和匿名签名资源尚未开放。

## 运行与安全

显式 JSON 提交入口为 `POST /v1/task/plugins/:plugin_key`，通过本地令牌认证、
分组/模型限制、渠道选择、提示词审计和 quota 预扣结算。
`originTaskIds` 最多 16 个，逐个校验用户、分组、模型和渠道能力；
多源必须同渠道、同分组、同源码版本，暂不支持跨版本引用。

任务私有字段 `execution.task_plugin` 保存 key/version/API version/source hash/source kind，
`plugin_state` 保存受大小限制的状态，`plugin_data` 保存原始响应，`plugin_immediate` 保存即时终态，
`plugin_result_url` 仅保存内联资源。供应商凭据和原始响应不放入公开 Data。
内置源码归档到独立 task_plugins 表。
历史轮询按 key/version/hash 读取并校验源码；缺失时禁止回退当前版本。
源码 hash 是完整性校验，不是发布者签名。

第一阶段仅支持本地显式按次价格，不把 UsageFacts 数值直接当作 OtherRatios 连乘，
不修改已有表达式或原生倍率。不支持的计价模式在预扣与上游请求前拒绝。
组合资金来源暂不支持，已预扣时沿用本地会话退款；本阶段已验证钱包，订阅保留原链但尚未组合验收。
终态沿用本地 CAS、退款认领、失败恢复和补偿；审计管理员信息与 Root 源码归因分开显示。

提交解析完成后先保存任务，再结算并返回公开 ID。上游成功但数据库写入失败时返回错误且不自动重试，
沿用预扣会话退款；供应商端可能留下未登记任务，尚无跨系统事务或补录队列，不应作为生产就绪证明。
插件提交发生错误时不自动重试，以避免不确定的上游受理状态造成重复任务。

数据库和缓存选渠都在优先级、权重及并发选择前隔离 62 与目标 key；亲和性和指定渠道复查同一规则。
原生与 HTTP 入口排除 62，原生 Go 适配器及 AtlasCloud 不变。
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

下一阶段尚未验收：MySQL/PostgreSQL 真实迁移、订阅退款及故障恢复的插件组合测试、
完整 TokenAuth HTTP 端到端、真实供应商与远程媒体成功、S3、匿名短期签名及撤销、
市场、上传、在线试运行、UsageFacts 表达式计费、Responses 流式/同步协议外观。
十个内置源码与固定上游一致，但仅 Sora 提交退款和 Hailuo 异常传播做了宿主完整组合验证；
其他供应商只证明离线脚本契约可运行。未移植依赖未接入 Responses 外观的上游测试。
Default 与 Classic 由各自工作树实现，本工作项仅修改后端和开发文档。
