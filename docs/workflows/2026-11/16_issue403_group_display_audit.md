# Issue #403 全局分组展示只读审计

日期：2026-11-16。任务：`task_87b512f48d18`。

## 目标、范围与验收方案

- 目标：区分已经修复但未部署的令牌列表问题，与仍缺失当前分组名称的真实展示调用链。
- 范围：Default、Classic 的列表、移动卡片、选择器、详情、通知、错误、导出及扩展宿主；外置扩展只读定位，不写子模块。
- 方案：先核对 Issue #403、项目约定及历史记录，再从 JSX 可见输出追踪 API 和名称来源，最后用名称不等于 code 的受控浏览器夹具验证代表路径。
- 安全：仅写本文及 `D:/Temp/issue403-group-audit`，不读取真实令牌、不连接生产 API、不修改源码、配置、数据库或子模块，不提交或部署。
- 兼容：展示名称不得替代 ID/code 的筛选、持久化、鉴权、计费及提交契约；已有 ID 优先，历史 alias 批量解析，真正未知或删除后才回退原标识；通知历史快照与当前名称分别判断。
- 验收：逐项标记源码证据、真实浏览器/mock 证据、未验证范围；不能凭搜索命中、路由存在、编译成功或旧截图宣布全局通过。每个测试进程最多 60 秒。

## 部署与证据边界

协调者只读运维回执指出，三节点仍运行 `zzapi-mcp-money-20261116-13f5fdbb9`，健康且主题为 Default，本轮令牌组名和 API 地址入口修复尚未部署。该信息来自协调者，不是本工人重新登录生产取得的证据。用户截图是旧线上版本证据，不能用于判定当前工作区修复失败。

已有 #401 证据见 [迁移矩阵](16_issue401_migration_matrix.md)，#402 见 [API 地址记录](16_issue402_api_addresses.md)；其中旧截图和旧测试结果仅作历史对照，本轮复验另记新结果。

## 审计方法与时点

本轮读取了 #403 正文、根及前端约定、开发文档总览、二开总览、分组标签、通知、扩展专题及 #401/#402 工作记录。源码扫描之后逐条核对调用方、响应数据、实际 JSX/文本输出；`value`、React `key`、查询参数、颜色种子、业务字典键本身不是展示缺陷。

核查发生于 2026-11-16 13:35–14:01（UTC+8）附近。协调者收到各批问题后已立即派出实施工人，工作区正在变化；以下行号是发现时定位，不是对他们后续修改的复审结论。本文不写任何共享实现，不将新出现的测试文件或已派修复状态当作验证通过。

## 展示面与契约矩阵

表中 Default 路径基于 `web/src/`，Classic 基于 `web/classic/src/`；源码位置的完整相对路径列于下一节。管理端 API 仍须维持原角色与能力校验，普通用户不得为了查名称改用管理分组接口。

| 展示面                           | Default / Classic 对照                                                                                | 名称与业务字段、权限                                                                                                                    | 本轮结论与未验证范围                                                                                                                                                                     |
| -------------------------------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 令牌列表、编辑器、移动端         | `/keys` / `/console/token`；Default `ApiKeyGroupCell`，Classic `TokensColumnDefs`、`TokensMobileList` | `/api/token/` 返回 `group/group_ids/group_details`，本人可用组 `/api/user/self/groups`；展示 Name，提交 ID/code                         | **已有修复未部署**。本轮 Default 桌面单多组、alias、改名、原生 title、保存刷新通过；手机编辑器通过，Default 手机列表没有组字段。Classic 本轮只读代码及旧构建对照，不重复签署旧浏览器验收 |
| 令牌 Auto 顺序及 ARIA            | Default `AutoGroupOrderEditor`；Classic 令牌编辑独立实现                                              | `auto_groups` 为 code 顺序，名称来自可用组；不能改顺序或提交值                                                                          | **G04**：Default 可见正文和 ARIA 用 code；交 #404 工人。未重新运行改后顺序编辑                                                                                                           |
| 个人资料                         | `/profile` / `/console/personal`                                                                      | `/api/user/self`、登录/刷新安全 DTO；本人权限，`group` 保留业务标识、另需 `group_id/group_name`                                         | **G01**：双端直接展示 group，self DTO 原本没有名称。本轮桌面浅色 Default 与手机深色 Classic 均重现；不是服务器身份/认证测试                                                              |
| 用户管理列表、选择器             | `/users` / `/console/user`                                                                            | 管理 `/api/user/`、`/search`、`/:id`；Default 选择器原 `/api/group/` 仅 codes，Classic 已用 `/api/group/details`                        | **G01**：列表有 `group_name` fallback 但模型查询未填；Default 选择器 label=code。普通用户隔离和真实管理能力拒绝未运行                                                                    |
| 普通日志中的用户详情             | `/usage-logs/common` / `/console/log`                                                                 | `/api/user/:id`，沿用管理用户详情权限，不扩展普通用户可见范围                                                                           | **G01**：双模板详情直接输出 group；需同时修响应和展示                                                                                                                                    |
| 模型管理列表、卡片               | `/models` / `/console/models`                                                                         | Admin `/api/models/`，`enable_groups` 保留 code 数组，另给名称映射/数组                                                                 | **G03**：双端组徽标直接显示 code。卡片是否隐藏此字段按原布局，不要求为审计增设字段；新实现未复验                                                                                         |
| 公开模型广场、详情、性能、组选项 | `/pricing` / `/pricing`                                                                               | `/api/pricing` 的 `group_names` 对应 code；筛选、倍率仍 code                                                                            | 普通名称路径已存在；**G10**：Default 原型键改名刷新缓存丢自有属性后 helper 抛异常，Classic 未知原型键显示对象/函数文本。真实 helper + QueryClient 已复现，非完整浏览器                   |
| 订阅购买、钱包卡、管理桌面/手机  | `/wallet`、`/subscriptions` / `/console/topup`、`/console/subscription`                               | 用户 `/api/subscription/plans`、本人订阅；Admin `/api/subscription/admin/*`；`upgrade_group/downgrade_group/prev_user_group` 保留业务值 | **G02**：购买/钱包/管理列裸 upgrade code；Default 升降组选项裸 code，Classic 编辑已有名称选项。账务、扣款、订阅迁移未运行                                                                |
| 渠道列表、移动卡、编辑、标签明细 | `/channels` / `/console/channel`                                                                      | `/api/channel/` 的 `group_ids/group_details`，服务端稳定绑定加载并将兼容 group 归一为当前 code；Admin/敏感字段裁剪                      | 常规列表已有名称链路，不因 `GroupBadge group=code` 就判失败。未补真实后端别名/删除/权限浏览器；**G07** 单列标签批改和日志详情遗漏                                                        |
| 渠道标签批改                     | Default `TagBatchEditDialog`；Classic 作为独立对照，不假定共用组件                                    | Default `getGroups()` → `/api/group/`，值 code，标签不改变授权                                                                          | **G07**：Default groupOptions label=code；已交 F，不能只修普通渠道列表                                                                                                                   |
| 日志渠道详情                     | Default `ChannelInfoDialog`；Classic 任务 `ChannelInfoModal`                                          | `/api/channel/:id`，已有 `group_details`；管理详情权限不放宽                                                                            | **G07**：Default 原始 group 字符串、Classic split 后原 code 标签；仅已有详情展示，不扩展任务分类                                                                                         |
| 普通日志列表、筛选、详情         | `/usage-logs/common` / `/console/log`                                                                 | `/api/log/`、`/self`、`/stat`、`/active`；`group_name` + 管理/本人裁剪 `group_names`；业务筛选仍原标识                                  | 已有批量 `GetGroupDisplayNameMap`、`FormatGroupDisplayNames`，#401 组选项已修；此次只读核对，不把旧 B1/B4 浏览器计为本轮，也不声称真实 COUNT/统计/本人隔离重验                           |
| 任务日志列表、任务详情           | `/usage-logs/task` / `/console/task`                                                                  | `controller/task.go` 批量组名映射，DTO `group_name`；列表和任务详情 Name 优先                                                           | 静态名称链路已存在；只有渠道信息弹窗另列 G07。绘图日志不新增分组或分类能力；真实任务、供应商及计费未运行                                                                                 |
| 提示词、Skills、操练场选择器     | 双端 PromptGroupSelector、TokenPicker、Playground                                                     | 实体 `id/code/name`，本人可用组；标签/名称不授权                                                                                        | 两模板组选项已 Name 优先；可见 code 搜索和业务 code 值不列缺陷。未运行 Skills 付费正文、安装/撤权、流式操练场                                                                            |
| 分组管理、标签、倍率规则         | 双端 GroupRatio/GroupTag/规则编辑器                                                                   | `/api/group/details`、`/tags`；ID 稳定，标签只是分类，配置读写权限保持                                                                  | 常规名称编辑/标签选项已存在；显式迁移诊断中的独立 ID/code 技术字段不删除。已知错误提示遗漏见 G09，完整原子保存/并发/三数据库未验收                                                       |
| 无限画布默认组选项               | Default `canvas-default-group-options.ts` / Classic `groupDetails.js`                                 | 已有实体 name/code，配置保存 code，Root 设置页门禁不变                                                                                  | **G05**：硬拼 `Name (code)`，不是未知 fallback；交 D。未运行真实画布会话                                                                                                                 |
| 安全审计策略、事件与归档筛选     | 双模板安全审计页面                                                                                    | `/api/security-audit/*`、`/api/request-archive/*`，Admin/Root 与能力约束按既有路由                                                      | **G05**：Classic 多个选择器追加 code；Default 事件名称格式器追加 code。原始请求正文及明确的诊断字段可保留机器值，不作全局替换                                                            |
| 请求归档、导出、对话归档         | 后端捕获 → 队列 → 加密元数据/JSON；双模板读取同契约                                                   | `BuildRequestArchiveRequest`、`RequestArchiveRequest.GroupId/GroupCode/GroupName` → 导出 `actor.group_name`；下载沿原权限               | **G08**：请求归档 Name 直接取 UsingGroup code；UserGroupId 与请求组语义还需采集时实体核定，不能声称是最终路由。历史 AAD 不改写。对话归档沿 snapshot 名称，未取得真实解密/导出数据        |
| 渠道可观测性                     | 双端 ChannelObservability 的筛选、分组稳定性、渠道视图                                                | 聚合 `group_name`、配置组 `group`；Admin，查询参数 code                                                                                 | **G06**：Default 分组 subtitle、Classic 选项/稳定性追加 code，Classic 配置组裸 code；宿主 service/DTO 需独立补名称。未做生产监控调用                                                     |
| 通知中心、Bot、扩展事件          | 通知模板预览/投递；模型校验扩展 source                                                                | `model/upstream_model_guard.go` 按 GroupID 取 Name，事件 payload `group` 保存事件时名称                                                 | 静态证明模型校验通知已保存名称；通用通知中心渲染既有负载，不应追溯改历史事件。工单/发票事件无对应业务组展示不强造映射；真实 Bot 投递、旧自定义模板未验收                                 |
| 福利/兑换记录                    | 双端 activity/voucher 卡片                                                                            | `group_name_snapshot`，活动编辑/生成时写名称，历史凭证保持快照契约                                                                      | 未发现直出当前 code 的确定路径；此处为历史快照，不能据此声称改名刷新当前组名已通过。未运行活动改名/领取/退款                                                                             |
| 扩展宿主与本仓模型校验扩展       | Native SDK、iframe 宿主；`extensions/upstream-model-guard/public/source/page.mjs`                     | `group_codes` 为规则值，选项 name、记录 `group_name`；原生/iframe 权限保持                                                              | 宿主不负责重写任意扩展页面的 DOM；本仓规则选项与记录已有名称优先。channel-quality 修前缺口见 **G11**；后续已授权同步主仓内嵌与外仓 D 盘副本，内嵌纳入待审候选，外仓未获推送发布授权      |
| API 地址入口（#402）             | Default keys PrimaryButtons，Classic 沿用原入口                                                       | 公开 `/api/status` 的 `api_info`，仅展示/复制地址，非分组身份契约                                                                       | **已有修复未部署**。当前工作区桌面浅/手机深真实浏览器打开 mock 地址成功；money 源快照无该 Default 入口。未测生产地址连通性                                                               |

## 可复现缺陷与精确待改范围

以下均已逐批发给协调者。P2 表示用户可见契约遗漏或页面异常，不推断鉴权绕过；每个“已派”都仍需要实现完成及另一工人独立复审。

### G01：用户资料、列表、选择器和日志用户详情缺名称（P2）

- 源头：`model/user.go:102` 的 `GroupName` 是 `gorm:"-"`，发现时 `GetAllUsers`、`SearchUsersWithSort`、`GetUserById` 直接查询，没有填名称；`controller/user.go:537` 的 `buildSelfUserData` 仅返回 group。登录、刷新与 self 复用该 DTO。
- 可见输出：Default `features/profile/components/profile-header.tsx:158`、`features/usage-logs/components/dialogs/user-info-dialog.tsx:134`；Classic `components/settings/personal/components/UserInfoHeader.jsx:162,210`、`components/table/usage-logs/modals/UserInfoModal.jsx:120`。
- Default `features/users/components/users-mutate-drawer.tsx:375,389` → `features/users/api.ts:getGroups` → `/api/group/`，label/value 均 code。双模板用户列表虽然已有名称优先，源头缺字段仍会 fallback。
- 复现：组 ID 11、code `stable_a`、name `Standard Display`，本人页显示 stable_a；本轮即使 fixture 已提供 `group_name`，两个页面仍忽略它。
- 已派 A `ctx_ee3ec9e898c7`，包括 controller/user.go、双端资料/日志用户详情及 Default users API/选择器。不得改变普通用户查询范围。

### G02：订阅及钱包购买入口裸分组 code（P2）

- Default `features/subscriptions/components/subscriptions-columns.tsx:197`、`dialogs/subscription-purchase-dialog.tsx:365`、`features/wallet/components/subscription-plans-card.tsx:568`。
- Default `subscriptions-mutate-drawer.tsx:450,494` 升级/降级选项也直接 code。
- Classic `components/table/subscriptions/SubscriptionsColumnDefs.jsx:96,194`（桌面/手机）、`components/topup/SubscriptionPlansCard.jsx:935`、`modals/SubscriptionPurchaseModal.jsx:311`；Classic `AddEditSubscriptionModal` 已用详情名称选项。
- 触发：套餐 upgrade_group 指向名称不同于 code 的真实组，进入列表/购买卡即显示 code。响应需追加展示字段，不能把 upgrade/downgrade/prev_user_group 改成名称影响订阅恢复。
- 已派 B `ctx_b7801a31f5ad`；当前未复验实现或真实账务。

### G03：模型管理 enable_groups 原样渲染（P2）

- Default `features/models/components/models-columns.tsx:365` 把每个 code 直接传无名称查询能力的 `GroupBadge`。
- Classic `components/table/models/ModelsColumnDefs.jsx:73–81` 的 renderGroups 直接 `{g}`。
- 源头 `/api/models/` 的 enable_groups 是业务 code 数组；应批量补展示数据，已有 MCP 投影不能当正式 Web API 名称填充。
- 已派 C `ctx_f98bc9ba83b2`，controller/model.go 与双端列表，保留他人工程 Logo/MCP 差异。公开 pricing 原型键另见 G10。

### G04：Default Auto 顺序正文与无障碍名称使用 code（P2）

- `features/keys/components/auto-group-order-editor.tsx:101,108,116,130,143` 用 props.group；调用方 `api-keys-mutate-drawer.tsx:637` 传入 code 顺序。
- 触发：令牌选 Auto 后使用自定义顺序，实体名称与 code 不同。正文、拖拽/上移/下移/删除 ARIA 都要用名称，`Reorder.Item value` 及提交数组必须仍为 code。
- 协调者追加给 #404 工人 `ctx_955fa16bac06`；本工人未触碰独占表单/type/倍率文件。

### G05：画布与安全审计追加 code，不是合法 fallback（P2）

- Default `features/system-settings/models/canvas-default-group-options.ts:34`；Classic `helpers/groupDetails.js:248`：硬拼 `Name (code)`。
- Classic `pages/SecurityAudit/BuiltinPolicyTab.jsx:457,680`、`RequestArchiveTab.jsx:896`、`PolicyTab.jsx:134`：名称后硬追加 code。
- Default `features/security-audit/event-routing-display.ts:172`：有 primary Name 仍把 code 放入 metadata。需区分普通组身份摘要与明确的原始请求/审计技术字段，后者不盲删。
- 已交协调者的 D 范围；Classic helper 自有属性安全回退也由该独占工人处理。完整事件详情/历史快照与真实权限未运行。

### G06：宿主可观测性配置组/分组副标题显示 code（P2）

- Default `features/channel-observability/stability-view.tsx:387`：group_name 与 group 不同便显示 group subtitle。
- Classic `pages/ChannelObservability/index.jsx:495`、`OperationsView.jsx:538` 追加 code，`ChannelsView.jsx:298` 配置组直接 record.group。
- 已派 E，补充 `service/channel_analytics.go` 与 `dto/channel_analytics.go` 独占；不能只去除前端字段却不为当前配置组提供名字。外置页面是独立 G11。

### G07：渠道标签批改与日志渠道详情遗漏（P2）

- Default `features/channels/components/dialogs/tag-batch-edit-dialog.tsx:73–79`，`/api/group/` 纯 code 生成 label。
- Default `features/usage-logs/components/dialogs/channel-info-dialog.tsx:146`，`getChannel(/api/channel/:id)` 后直接 channel.group。
- Classic `components/table/task-logs/modals/ChannelInfoModal.jsx:74,144`，已有详情按 group split 后直接标签，忽略 group_details。
- 已派 F `ctx_faa487ddfc16`。此处只修现有弹窗，不扩展任务/绘图日志分类；Admin/敏感数据权限维持。

### G08：归档导出 group_name 实际写入 code（P2）

- `service/request_archive_scope.go:46,51` 从 UserGroupId 与 UsingGroup 填 GroupId/GroupName。
- `service/request_archive_runtime.go:132` 持久化，`service/request_archive_json.go:78` 输出 `actor.group_name`；不是单纯机器字段命名或 JSX 扫描误报。
- 常规路由 `PromptAudit` 先于 `Distribute`，故应该确认**采集阶段实际实体**，不能编造最终选中路由组；至少 Name 字段当前取 code 已可证实。已知实体按 ID/current Name，历史 alias 正常解析。
- 已派 G `ctx_11e38c7f7548`。不重写历史加密 AAD、已归档内容或通知快照；未运行真实加解密/存储/下载。

### G09：已知实体的校验错误仍直接报 code（P2）

- `model/group_bindings.go:455`：code/alias 已查得 Group，拒绝禁用组时仍报 code。
- `model/group_identity.go:2183,2186`：normalizeGroupName 已完成，负倍率/负并发仍报 code；`:1732` 独立组 Auto 拒绝只 select id/code，应获取名称但保持拒绝逻辑。
- 补充 `model/group_token_migration.go:526–536`：groupsByID 已拿到实体，稳定倍率保护与旧 JSON 不一致时仍 group.Code；`controller/group.go:193,217` ApiError 返回页面。
- 补充 `service/prompt_audit_config.go:197–206`：已查得但停用实体的错误仍 code；真正不存在的标识须保留回退。
- 后续状态（协调者正式回执）：全部上述范围，包括补充的 `model/group_token_migration.go` 与 `service/prompt_audit_config.go`，均已授权 H `ctx_08091730587e`；H 已完成 7 文件实现及定向测试，尚待另一工人独立审查，未标为已验收。本工人未重跑这些测试，原修前定位与证据保留。原始迁移碰撞、非法未知输入、数据库技术诊断不能一律换名称；不扩大鉴权、倍率或迁移规则。

### G10：pricing 原型属性名回退及改名重取异常（P2）

- Default `features/pricing/lib/group-names.ts:25`：`groupNames[groupCode]?.trim()` 未检查自有字符串。
- `features/pricing/hooks/use-pricing-data.ts` 的 `['pricing']` 查询保持默认结构共享。使用真实 QueryClient，先后写入 JSON.parse 的自有 `__proto__` 对应旧名/新名后，该属性在缓存中丢失，随后真实 helper 抛 TypeError。
- 未知 `constructor`、`toString`、`__proto__` 也会抛异常；Classic `helpers/groupDetails.js:59` 则返回对象/原生函数字符串，不是真未知标识回退。
- 真实运行脚本及原失败 JSON：`pricing-prototype.cjs/json`、`pricing-classic-prototype.cjs/json`。Default 的 13 项旧名称/契约测试同时全绿，不能反证此边界没有问题。
- Default 已交 C，Classic 共用 helper 交 D，避免并写；只调整相关查询/展示解析，不改全局缓存策略。此证据为 helper + QueryClient，尚非浏览器页面崩溃复现。

### G11：channel-quality 修前直出/追加 code（P2，内嵌与外仓分别交付）

- 仓库 `moeacgx/maolaonewapi-extensions`，只读 main 树 SHA `c5dd5b1446e091bcf475c149f8b47a6954721c9a`。
- 确切源 `modules/channel-quality/public/app.js`，blob SHA `3e7defd427287ad96eb9754742be30f761ec55df`。
- `:1062` 的 stabilityIdentity 在 Name 后 `<em>` 追加 code；`:1523` normalizeChannel 仅取 group/channel_group；`:1600` 配置组直接 channel.group。
- 本仓库子模块未初始化。本次只通过 gh API 抓到 D 盘副本，没有初始化、更新或写入子模块。
- 后续归属：`ctx_8e656052ecae` 已获主仓 `extension/builtin/channel-quality/public/app.js`、`extension/builtin/channel-quality/public/native/default.mjs`、`extension/builtin/channel-quality/public/native/classic.mjs` 写权，负责与外仓 D 盘副本同步本地修复；主仓内嵌资源会纳入待审发布候选，仍需独立审查和发布验收。外仓没有推送或发布授权，本地副本修复不等于外仓已发布。
- 内嵌必须同步的原因：`extension/registry.go` 初始化调用 `installBuiltinModules`；`extension/builtin.go` 不仅比较清单版本，还逐项比较内嵌与安装目录文件，内容不一致时会用内嵌资源恢复安装目录。因此仅改外仓副本或运行目录，旧主程序再次初始化仍可能恢复旧资源；必须把本次内嵌修复随主仓候选一同构建，不能继续笼统写成“主仓部署不会修复它”。此处只读核对启动恢复调用链，未运行安装或部署。

## 本轮运行验证

全部新脚本与证据在 `D:/Temp/issue403-group-audit`。浏览器使用真实 Chromium，Default 复用本地 `127.0.0.1:5174` 应用，Classic 对已有 `web/classic/dist` 开临时 `127.0.0.1:5196` 静态服务。网络仅允许本地资源，`/api/*` 全部受控 fulfill，外部地址请求 abort；fixture 没有真实 Key，保存仅更新内存。Classic 旧构建结果只能证明该构建，不能替代正在改动的 Classic 源码重建。

| 编号 | 执行内容                                             | 实际结果                                                                                                            | 证据文件                                                                               |
| ---- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| V1   | Default `/keys` 桌面 1440×1000、浅色、普通用户       | 单组名称、多组名称、按 ID 解析历史 alias、改名重载、原生 title、保存刷新通过；提交 `[11,12]` 和 `stable_a,stable_b` | `F3-desktop-light.json`、`F3-desktop-light-tooltip.png`、`F3-desktop-light-editor.png` |
| V2   | Default `/keys` 手机 390×844、深色、普通用户         | 编辑器名称、改名重载、保存刷新与稳定提交通过；手机列表无组字段，列展示不适用                                        | `F3-mobile-dark.json`、`F3-mobile-dark-editor.png`                                     |
| V3   | Default `/profile` 桌面浅色、普通用户                | 复现 raw code 1 处、display Name 0 处；截图可见 stable_a，响应提供 Name 仍被忽略                                    | `profile-default-desktop-light.json/png`                                               |
| V4   | Classic `/console/personal` 手机深色、普通用户       | 复现 raw code 2 处、display Name 0 处；真实浏览器的既有构建证据                                                     | `profile-classic-mobile-dark.json/png`                                                 |
| V5   | Default API 地址桌面浅色、普通用户                   | 入口按钮可见，打开后显示 fixture 线路及 `https://gateway.example.test/v1`；不请求该地址                             | `addresses-desktop-light.json/png`                                                     |
| V6   | Default API 地址手机深色、普通用户                   | 同 V5，通过                                                                                                         | `addresses-mobile-dark.json/png`                                                       |
| V7   | keys group-cell 与 pricing-contracts 两份既有 Vitest | **2 文件 13 项通过，exit 0，子进程 23.01 秒**                                                                       | `focused-tests.json/log`                                                               |
| V8   | 当前 Default 真实名称 helper + 真实 QueryClient      | 原型键改名后 own=false，调用抛 TypeError；普通 unknown 正常回退                                                     | `pricing-prototype.cjs/json`                                                           |
| V9   | 当前 Classic 真实 groupDetails helper                | `__proto__` 输出 `[object Object]`，constructor/toString 输出原生函数文本；普通 unknown 正常回退                    | `pricing-classic-prototype.cjs/json`                                                   |

V7 的确切命令（cwd=`web`）：

```powershell
node node_modules/vitest/vitest.mjs run src/features/keys/components/__tests__/api-key-group-cell.test.tsx src/features/pricing/lib/__tests__/pricing-contracts.test.ts --maxWorkers=1
```

运行器使用 `subprocess.run(..., timeout=60)`，`TEMP/TMP` 仅设为本轮 D 盘目录；Vitest 子进程的 `NODE_OPTIONS=--no-experimental-webstorage`，没有修改系统环境。浏览器每个场景同样独立 60 秒上限。已有测试不能覆盖本轮全部新缺口。

失败与限制如实保留：

- 最初 profile fixture 把 `/api/user/sessions` 返回成通用对象，Default 页面到 500；补成真实契约的数组后页面正常并重现组名缺陷。首次等待显示名失败、Classic 定位等待失败不是产品认证故障，未按缺陷上报。
- V8 首次尝试加载本地 `typescript` 模块失败，未安装依赖；改用 Node 26 的 `stripTypeScriptTypes` 去除真实源码类型后执行，未复制改写 helper 逻辑。
- V1 截图仍有开发请求 `canceled` toast；分组输出和提交断言成功不等于页面无其他错误。
- V1 只通过原生 title 文本；自定义 Tooltip 浮层未观察到，不签署浮层视觉。
- 本轮不重复 #401 全量测试、超时后端组合或构建；没有数据库、Redis、供应商、真实身份切换、通知投递或生产验证。

## 已修未发布与不能列为缺陷的命中

除协调者线上回执，本轮只读检查本地 money 源快照 `D:/Temp/newapi-mcp-money-20261116/snapshot/`：

- `web/src/features/keys/components/api-keys-primary-buttons.tsx` 只有创建按钮，没有 `ApiKeyApiAddresses`；当前工作区已 import/render 且 V5/V6 实测可见。
- 快照 `api-key-group-cell.tsx` 的 tooltip 与 GroupBadge 均直接 props.group；当前工作区传 groupIds/groupDetails/批量组数据，V1/V2 用新截图复验。
- 这是本地冻结源码对照，不是重新访问生产或证明当前镜像文件 hash；镜像与节点版本由协调者运维证据负责。本次审计未发布任何内容。

以下命中不作为用户分组展示缺口：

- Uptime 面板 `monitor.group` 来自 `controller/uptime_kuma.go` 的 `pg.Name`，是上游监控分组名称，不是 NewAPI Group.Code。
- SQL `Group(...)`、表单 `role=group`、React `key`、请求参数 `group`、规则 map key、倍率计算、存储 code、渠道调度 scope 都是机器语义。
- 普通/任务日志的 `group_name || group` 有后端批量填充；缺失/删除时 fallback 本身合法，不能只看 `|| group` 判错。
- 渠道列表依赖稳定绑定加载后的 `group_details` 和归一 code，不能凭前端按 code 查名称就认定 ID 优先丢失。
- 模型校验事件在产生时按 GroupID 保存名称到 payload；历史通知快照不因改名追溯重写。
- 审计原始请求 JSON、显式标识迁移预检、未知/已删除实体错误需保留实际标识；不得全局替换 code 字符串。

## 交付状态与待验收

本轮交付是审计矩阵、原始失败夹具及可证实缺口清单，不是 G01–G11 实现完成声明。协调者已分配 A–H、#404 与扩展同步工作；G09 两处补充已授权 H，7 文件实现及定向测试完成、待独立审查；G11 已授权内嵌与外仓 D 盘副本同步，内嵌进入待审候选，外仓无推送发布授权。本工人唯一仓库写入为本文。

逐项剩余边界：

1. G01–G09：各范围按正式回执推进；G09 已完成实现及定向测试，尚待另一工人对实际新实现独立复审；需按姓名不同于 code、ID/code 冲突、alias、乱序、删除、未知值及角色裁剪补回归，不能复用本轮修前截图签署修后通过。
2. G10：保留当前最小失败夹具，修后原夹具及真实 hook 缓存刷新至少复验；本轮没有公开 pricing 页面浏览器的完整改名重取操作。
3. G11：修前只读定位保留；已授权同步主仓内嵌资源与外仓 D 盘副本，本仓内嵌将纳入待审发布候选。仍需检查启动恢复后的资源与宿主 API 兼容；外仓版本、推送、发布和安装需另有授权，不能把主仓测试当成外仓发布证据。
4. 真实生产截图所对应的 money 镜像：尚未包含本轮组名与 #402 修复。部署、更新扩展资源和发布后入口验证由协调者负责；用户明确要求等全部 worker 完成才部署，当前尚无部署。
5. 其余静态名称链路：未运行全角色×双模板×桌面/手机×浅深色组合，也没有完整导出、Bot、历史快照、数据库和计费测试；不宣称“全局已统一”。

本报告不改变兄弟项目契约。NewAPIForDouDi 与 NewAPIModifyByGang 若也使用 code/alias 与独立显示名，可参考“批量名称字段、ID 优先、原型键安全”的审计结论；是否采纳需各自确认现有身份模型。没有向兄弟仓库发送 Issue、PR 或消息。

文档使用仓库现有 `oxfmt` 格式化并检查；本文相对链接指向的两份历史工作记录均存在，独占文件未发现冲突标记。本文是未跟踪新增文件，除限定 `git diff --check` 外，另执行 `git diff --no-index --check NUL <本文>` 检查完整新增正文，退出 1 表示与空文件不同，空白诊断应为空。D 盘证据目录的 `evidence-manifest.json` 保存本轮脚本、运行报告与截图的 SHA256 和修改时间，供后续复验区分新旧结果。
