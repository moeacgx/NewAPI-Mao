# 官方 Task Plugin 接入计划

## 目标

在保留现有二开扩展模块（`/extensions`、原生扩展 SDK、通知、归档、安全审计等）
和原生任务适配器的前提下，接入官方 `new-api` 的 JS Task Plugin 系统，供管理员
先在 zzapi 测试环境体验。当前不删除、不隐藏、不替换任何二开扩展，也不改变生产部署。

官方实现固定参考 `upstream/main@9fe0457ee`，核心架构起点为 `eb48396d5`：

- `pkg/jsplugin/`：Sobek 引擎、插件注册表、路由代际、请求上下文、JSON 状态和资源辅助。
- `plugins/`：官方内置插件源码与 embed 注册器（alibaba、doubao、google、hailuo、
  jimeng、kling、sora、sunoapi、vertex-ai、vidu）。
- `model/task_plugin.go`、`controller/task_plugin.go`、`service/task_plugin_*`、
  `middleware/task_plugin*`：插件存储、上传、启停、版本、市场、审计和渠道绑定。
- `relay/channel/task/jsplugin/`：任务提交、流式输出、轮询和响应解析适配器。
- `web/src/features/task-plugins/`：Default 管理页、市场、上传、版本、试运行和用量视图。
- `router/api-router.go`、`router/task_plugin*`、`dto/task_plugin.go`：管理和渠道选项 API。

## 并存契约

1. 保留 `/extensions` 路由、数据库、原生 SDK、扩展页面、通知事件和 Root 门禁。
2. 保留全部 Go 原生任务适配器和 `ChannelTypeAtlasCloud=61`，官方插件不得复用 61。
3. 官方插件使用独立的 Task Plugin 存储、版本和路由；停用插件不能隐式回退到二开扩展或原生适配器。
4. 历史任务按创建时的来源和版本读取；在途任务不能因当前插件版本变化而改变轮询/结算解释。
5. 插件执行必须复用本地分组授权、渠道能力检查、提示词审计、亲和性、失败指标、退款认领和日志脱敏。
6. 插件源码完整性校验、Root 上传权限、版本激活和市场来源必须与本地审计体系并存；不把 SHA-256 当作发布者签名。
7. 资源代理必须保留现有 SSRF、重定向、凭据清理和访问日志脱敏；短期签名 URL、撤销和对象存储能力未完成前，不开放生产资源上传。
8. 官方 UsageFacts/表达式价格不覆盖本地 `OtherRatios`、按次价格、钱包 int64 和 quota saturation 审计；计费适配单独验收。

## 分阶段范围

### P0：后端运行时和只读管理

- 移植模型、注册表、Sobek 运行时、内置插件 embed、任务适配器和 API 路由。
- 保留 AtlasCloud、原生适配器、源任务多分组授权、退款补偿和本地日志链。
- 迁移数据库字段/索引时兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+；不得复用 AtlasCloud 编号。
- 默认插件系统总开关保持关闭；管理员显式开启后才允许体验。

### P1：Default 与 Classic 管理入口

- Default 接入官方 `task-plugins` 页面、市场、上传、启停、版本、试运行、能力和用量展示。
- Classic 提供等价管理入口，保留 Semi UI、主题、通知、福利、发票、支付和上游模型显示。
- 两套模板分别验证权限、空状态、错误、i18n、插件禁用和二开扩展入口仍可用。

### P2：绑定、计费和运行时切换

- 渠道创建/编辑可选择官方插件并保存绑定信息，严格区分插件编号与 AtlasCloud。
- 完成插件版本 pin、源任务授权、轮询/资源读取、UsageFacts 到本地计费的安全转换。
- 先在 zzapi 用一个非生产插件和测试渠道演练，再评估是否启用更多内置插件。

## Agent 分工

### 当前实现责任与交接

当前本地功能基线为 `origin/custom-main@5d90556e2`（`.323`），上游参考仍固定为
`9fe0457ee`。各实现分支提交前重新核对远端基线，不在旧的 `353352428` 上交付。

- 后端唯一实现：Paseo `463ac670-a3eb-4a84-aa3d-0ee5fabf741d`，工作树
  `official-plugin-backend`；旧历史保留为 `backup/official-plugin-backend-before-20260915`。
- Classic 唯一实现：Paseo `c31b189d-4a65-4048-a4ac-ee2aa523ab07`，工作树
  `official-plugin-classic`。
- Default：原 Paseo `273f496e-7c8d-41c2-b79c-f4ea946f14c0` 因工具缺失没有交付，
  已明确停止写入。接替实现使用 `official-task-plugin-default`，分支
  `agent/official-task-plugin-default`。
- 兼容审查：Paseo `62837a70-2f96-4d72-a790-5c87432d76ad`，只读。

### 已固定的管理 API 契约

本节描述正在实现的接口合同，不表示已经合入、发布或完成任务运行验收。
接口统一返回 `{success:true,message:'',data:...}`。

- 渠道编号：`TaskPlugin=62`，`AtlasCloud=61`；插件绑定写入渠道
  `setting.task_plugin_key`，保存其他 setting 字段时不得丢失该键。
- `GET /api/plugin/task`：每个 key 一项，优先活动版本，否则按
  `created_at DESC,id DESC` 选最新归档版本。历史版本不混入列表。
- `GET /api/plugin/task/:key`：可用 `?version=...` 选择版本；省略时采用同一
  选择规则，首次安装没有 active 版本也能读取。
- 列表和详情保留 `key/version/api_version/source_hash/enabled/active/source_kind`，
  当前 `source_kind=builtin`；另有完整 `meta` 对象，前端不得自行解析源码获取元数据。
  `meta` 包含名称、模型、协议、路由和插件声明的用量字段等。
- `channel_count` 是启用的 type 62 渠道中绑定该 key 的数量；`in_flight_count`
  是该 key 的非终态任务数量。前端应按该定义标注，不称作全部历史绑定数。
- 详情仅 Root 返回 `source`，管理员响应省略源码；
  `GET /api/plugin/task/:key/versions` 返回历史版本且不带源码。
- Root `POST /api/plugin/task/:key/activate`，请求 `{version:string}`：激活并启用
  指定版本。新安装为 `active=false,enabled=false`，页面必须提供先激活的流程。
- Root `POST /api/plugin/task/:key/status`，请求 `{enabled:bool}`：修改已激活插件状态。
- `GET /api/plugin/task/runtime/status` 返回
  `{enabled,channel_type:62,builtin_only:true,billing:'per_call',resource_access:'authenticated_inline_only',production_ready:false}`。
  Root `PUT` 同路径携带 `{enabled:bool}` 管理 `TaskPluginEnabled`，默认 `false`。
  前端不通过通用 `/api/option/` 修改此开关。
- `GET /api/task_plugin_options`：返回 `key/name/version/models/channel_type` 供渠道选择。

关闭总开关仅阻止新提交；历史任务按持久化的 key/version/hash 解析，不能依赖当前
active、enabled 或总开关。前端关闭说明不得沿用“立即停止在途轮询，等待超时清理”。

当前上传、删除版本和 dryrun 返回 501；市场、S3 与匿名签名尚未开放。两套前端不启用
这些操作，不发送对应请求。它们仍是完整官方插件体验的剩余范围，不能把内置插件管理
阶段描述为全部完成。真实提交、源任务授权、持久化、轮询、计费和退款仍需组合验收。

- **后端运行时 Agent**：只拥有 `pkg/jsplugin/`、`plugins/`、任务模型/控制器/服务/中间件/路由和迁移；必须保留 `/extensions` 与 AtlasCloud。
- **Default Agent**：只拥有 `web/src/features/task-plugins/` 及其路由/API/i18n，复用后端契约，不改 Classic。
- **Classic Agent**：只拥有 `web/classic/` 的插件管理入口及 i18n，复用后端契约，不改 Default。
- **兼容审查 Agent**：只读审查编号、权限、版本、资源、计费、审计和双模板契约，不自行合并。

每个 Agent 使用独立 Paseo worktree，模型固定 `gpt-6-astra`、思考程度 `high`。
提交前必须同步最新 `origin/custom-main`，提交、推送并创建范围准确的 PR；未完成能力用 draft PR 标识。

## 验证与上线门禁

- 后端：根 Go 定向测试、`go vet`、数据库迁移测试；`relaykit` 使用 `GOWORK=off` 独立 build/test。
- Default：Bun install、typecheck、task-plugin 定向测试、build；保留历史 lint 基线。
- Classic：Bun install、插件页面定向/原生扩展测试、build；保留历史 ESLint/Prettier 基线。
- 组合：权限矩阵、AtlasCloud 编号回归、禁用插件不回退、在途任务 pin、退款和日志脱敏。
- 部署：本计划只允许后续明确授权后在 zzapi 体验；不触碰 maolaoapi。

## 当前阶段交付与组合验证

已完成内置 JSON 任务、按次计费和认证内联资源阶段，完整官方插件能力仍在后续范围。
四个 PR 的审查对象如下，合并后按目标分支同步产生的新提交继续核对差异：

- 文档：#219。
- 后端：#222，`583755592d91cf6bca196b359bf8d41b1eaf7f36`。
- Classic：#221，`742ffb829a5ab1a7bfe753e82883743831113a02`。
- Default：#220，`fd03a488325a03a1c656e10e4c42955f0a2bde61`；包含 UTF-8 修复与真实语言包回归。

编号冲突已通过独立的 62 解决。数据库与缓存选渠、亲和性及指定渠道都隔离插件 key，
原生路径排除 62。历史 pin 轮询、Sora 钱包和 token 一次退款、Hailuo 真实 JS 异常、
fetch/read/parse/unknown 四类错误脱敏、公开状态及内联资源授权均有定向测试。
Sora 和资源用例手工注入身份上下文，不等于完整 TokenAuth HTTP 端到端。

2026-09-15 组合工作树 `official-task-plugin-gate` 的实际门禁通过：

1. 以锁文件安装两套前端依赖，分别运行 Rsbuild 和 Vite 构建；Classic 用时 70 秒，
   保留既有大 chunk 警告。没有使用占位 HTML 或缺失静态资源的替代物。
2. 两套真实 dist 存在后执行 `go build -mod=readonly -o .local-tests/new-api-official.exe .`
   成功，Windows 二进制大小 176357888 字节。
3. `go test -mod=readonly ./controller ./service ./router ./relay -run
'TestTaskPlugin|TestPluginSourceTaskAuthorizationUsesPersistedGroups' -timeout 60s -count=1`
   四包通过。实现代码逐路径与上述三个固定提交相同，开发索引和能力登记保留并集。
4. Default 修复后插件目录 8 文件 67 项通过；真实七语资源及启停/激活后选项更新通过。
   Classic 独立组件 13 项、作者扩展范围 22 项及原生扩展 11 项分别记录，不累计为一个测试集。

仅管理和本阶段运行闭环通过；未执行 zzapi 或 maolaoapi 部署。下列工作继续单独验收：

- MySQL/PostgreSQL 新表现场迁移、订阅退款和故障恢复、完整 TokenAuth 端到端。
- 真实供应商视频和远程媒体成功交付；当前仅有限内联资源，远程资源返回 501。
- S3、匿名签名及撤销、市场、上传、完整 UsageFacts 表达式、Responses 等完整协议外观。
- 未启用的 `task_plugin_full_protocol` 测试不能算作普通插件离线测试已经覆盖。

## 回滚

体验阶段关闭总开关即可禁止新插件请求；若已经产生插件任务，必须按持久化版本完成轮询或先排空再回退。
禁止通过删除数据库记录、删除插件源码或删除 `/extensions` 来回滚。任何数据库迁移都需先备份并提供逐库恢复步骤。
