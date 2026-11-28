# NewAPI-Mao 二次开发能力

## Issue #398–#402 本轮交付状态

新一轮独立审查发现的活动列表刷新丢失、API 地址 500 全页跳转两项 P2 及日志 `__proto__` 名称丢失 P3 均已返修并独立复验；返佣、Logo、Classic 立即提交新独审 44 项通过。此前双模板构建结果只对应当时快照，不能作为这次返修后的发布构建。用户已授权提交推送 dev、部署 zzapi 并切换 Default，截至本段登记均尚未执行；dev 纯本轮提交与保留线上 money MCP overlay 的组合产物边界见[部署专题](../workflows/2026-11/16_issue401_dev_zzapi_deployment.md)，实际冻结、构建、发布和切换尚未执行，由协调者在全部工作者解决与最终门禁后统一登记；详见[收口记录](../workflows/2026-11/17_issue398_405_release_closeout.md)。并行 MCP 内容保留，不同步上游。

## Issue #403–#405 本轮交付状态

11 个翻译键（#404 倍率保护 9 键、#405 顶部导航 2 键）已写入 7 个语言包（`en/zh/zh-TW/fr/ja/ru/vi`）的 `translation` 命名空间；`bun run i18n:sync` 通过，无多余键、无键删除，MCP 条目未变；zh/zh-TW/ja 比例键术语返修（比例→倍率，レート→倍率），r3 验证 77/77 值、7770 原键不变（含 105 个 MCP 条目），exit 0；临时脚本已删除；翻译最终文档独审已通过（七语言包 SHA 一致，11 键 × 7 语言 = 77 值）。详见[i18n 集成记录](../workflows/2026-11/16_issue403_405_i18n_integration.md)。

#403 分组显示名称各专题状态（字母以各专题记录自身标题为准，不按列表顺序推断）：

- **A 用户分组名称**：独立审查 22 项通过（Default 14 + Classic 8）；与 F 合计 32 项。详见[工作记录](../workflows/2026-11/16_issue403_user_group_names.md)。
- **B 订阅分组名称**：独立审查 13 项通过（Default 6/6、Classic 6/6、已删除分组 code 夹具 1/1）；六个生产文件仅格式修正已独审通过；手机夹具 E1 缺陷已由返修复审关闭，冻结夹具无新增 P1/P2，该结论仅限夹具与证据口径，不代表订阅手机全部场景验收；真机、全量构建与类型检查、真实后端权限、支付和数据库边界未验证。详见[工作记录](../workflows/2026-11/16_issue403_subscription_group_names.md)。
- **C 模型分组名称**：15 文件范围独立审查通过，无新增 P1/P2（Go 定向测试首次 60 秒超时不计通过，重跑通过）；无全量构建、真实认证与 MySQL/PostgreSQL 验证。详见[工作记录](../workflows/2026-11/16_issue403_model_group_names.md)。
- **D 画布与安全审计名称**：前端限定范围独立审查通过，三个 P2 均已闭环，无剩余 P1/P2；浏览器证据为本地 mock，真实父页面刷新、认证全链、MySQL/PostgreSQL 与生产未验证。详见[工作记录](../workflows/2026-11/16_issue403_canvas_audit_names.md)。
- **E 可观测性分组名称**：额外原生路由 P2 已通过删除该路由关闭，使用既有扩展入口；最终 12 个有效成功场景、15 份证据（含 2 份旧失败与 1 份负控，不计入成功数）；v7 不作为 groups 提交证据，筛选提交原 code 以已独审的 v5 证据为准；v5 筛选与 Go 三项安装/升级测试已独审，最后的文档格式 P3 由 v9 关闭；390 为 Chrome 移动视口模拟，Classic 禁用模块仅后端拦截；未发布外仓，不称生产或真机通过。详见[工作记录](../workflows/2026-11/16_issue403_observability_group_names.md)。
- **F 渠道分组名称**：独立审查通过（Default 6 + Classic 4 共 10 项）；与 A 合计 32 项。详见[工作记录](../workflows/2026-11/16_issue403_channel_group_names.md)。
- **G 归档分组名称**：后端独立审查通过。详见[工作记录](../workflows/2026-11/16_issue403_archive_group_names.md)。
- **H 分组校验名称**：后端独立审查通过。详见[工作记录](../workflows/2026-11/16_issue403_group_validation_names.md)。
- **全局分组展示只读审计矩阵**（无字母）：记录问题发现与派发，只读审计，未验证边界保留；不等同于后端通过。详见[工作记录](../workflows/2026-11/16_issue403_group_display_audit.md)。
- **外置扩展分组名称**（无字母）：外置与内嵌同源三入口修复及既有回归；额外路由 P2 已随路由文件删除而关闭，浏览器与安装仅以已独审的 E v5/v7 及 Go 三定向证据为限，外仓未发布，真实生产与整体验收未验证，不称整体通过。详见[工作记录](../workflows/2026-11/16_issue403_external_group_names.md)。

订阅降级分组删除引用 P2：已独立审查确认通过（`model/group_identity.go:1977`、`model/group_identity.go:1980`），验证仅限 SQLite，MySQL/PostgreSQL 未验证，详见[降级分组引用记录](../workflows/2026-11/16_subscription_downgrade_group_reference.md)；著作权头部恢复（783 文件）尚未提交推送，不部署 zzapi。

**#404 令牌倍率保护**：源码二层审查（2+23+8+8+Auto28）已完成，Classic 三个用例各自通过；真实后端与数据库边界未验证，不宣称整批通过。详见[工作记录](../workflows/2026-11/16_issue404_token_ratio_protection.md)。

**#405 页头布局与导航**：当前代码已独立审查，无 P1/P2；`Navigation`/`Navigation menu` 两键已集成到七语。作者原始四件证据（`after-1100-dark-ordinary.json` 与展开、菜单、折叠截图）被覆盖且不可恢复，作者历史 16 场景矩阵不再作为完整通过证据，以独立复核的当前代表场景为准：Default 1100 深色、Classic 1100、Classic 390 深色及相关自动检查；访客登录链路真实浏览器、真实后端和全部视口/主题组合未验证。详见[工作记录](../workflows/2026-11/16_issue405_header_layout.md)。

**格式化头部保护脚本**：实现与测试已独立审查（56/56），安全契约仅为输出候选、`--check` 只读、`--write` 拒绝；本轮不执行该脚本任何模式。详见[工作记录](../workflows/2026-11/16_formatter_header_safety.md)。

**本轮收口**：全部专题的独立审查结论、待统一集成、冻结与构建事项见[收口记录](../workflows/2026-11/17_issue398_405_release_closeout.md)；用户明确全部工作者解决与最终门禁完成前不部署 zzapi，不提交推送，不访问生产。

## MCP 管理接入

- 新增 `/mcp` 管理工具入口和独立调用密钥，按工具授予权限、设置有效期并随时撤销；工具调用继续经过既有角色与业务权限检查。
- Default 与 Classic 提供接入配置、工具参考、调用密钥及调用记录四个页签；本地 Codex / Cursor 通过环境变量提供凭据。
- 工具目录覆盖渠道与模型运维、用户与权限、分组实体、本人令牌、工单回复与自动化、
  订阅财务、营销返佣、提示词、通知、插件扩展、系统配置、安全归档和部署管理。
  写入、正文和秘密读取独立授权；不开放通用路由代理或支付回调，不绕过宿主会话证明。
- 覆盖边界：368 个工具已覆盖模型元数据、上游模型发现/同步、价格配置、渠道批量测试及宿主提供的媒体任务测试等管理能力。单渠道测试支持指定模型、协议与流式参数，具体模式仍受渠道适配器限制；登录证明、支付回调及宿主未实现的接口不开放。
- 稳定性：完整 368 工具目录已做本地后端、真实 SDK 和双模板验证；完整后台扩展的
  验证与限制见 MCP 专题。
  PostgreSQL 15 隔离副本的迁移、双实例并发去重及跨实例撤销已通过；
  MySQL 实库与真实供应商业务操作尚未验证。数据表、重试与接口契约见
  [MCP 管理接入](mcp-management.md)。

## 渠道单密钥转多密钥

- Default 渠道编辑增加显式转换入口，复用 Classic 已有的 `PUT /api/channel/` 契约；已存在的多密钥编辑和新建多密钥流程保留。
- 编辑要求 ChannelWrite，转换与多密钥策略要求 ChannelSensitiveWrite。来源或目标类型为 Vertex AI/Codex、io.net 托管渠道以及缺失有效来源类型时不提供转换；取回密钥的 RootAuth 不替代更新权限。
- 没有新密钥时保留原密钥，只发送调度策略；取消、重开或切换渠道清空转换状态。无迁移，回退 UI 不会撤销服务端已完成的渠道转换。
- 稳定性：P1 固定 head `e91adf6743fd29a7a907aae9fc47cc9720532744` 已获独立验收 pass；P2 组合 head `ffe87461b566dec6ea9119971d5d51f07b83a1f9` 的 P1 回归也已独立通过。正式基线候选仍单独验证和交验，见[集成交付](../workflows/2026-11/11_default_integration_delivery.md)。P1 验证命令、结果和范围见[工作记录](../workflows/2026-11/11_default_multi_key_conversion.md)。

- 2026-11-16 追加修复 Classic 多密钥选择器：在 `onSelect` 同步策略和追加/覆盖模式，避免下拉退出动画前立即保存旧值；请求字段与敏感写权限不变。Classic 10 项真实组件、8 项契约、Default 27 项对照及修后 12 次 mock 浏览器请求通过；Classic 独立重建 exit 0（33.51 秒），独立审查 `ctx_108ffd07f56c` 已通过，未发现 P1/P2。真实后端、Redis 轮询和生产权限仍未验收，见[工作记录](../workflows/2026-11/16_issue401_classic_key_mode.md)。

## 充值日志订单客户端 IP

- 新订单型充值日志使用建单时由 `c.ClientIP()` 解析的地址，遵守消费日志相同的强制／用户记录开关；回调和补单地址不作为用户 IP。
- 历史日志继续脱敏，不从最近消费或登录记录补填；普通用户不返回管理员审计字段。
- 稳定性与边界：仅本地验证，未读取线上配置或日志；无数据库迁移，不调整代理信任。契约与测试见[工作记录](../workflows/2026-10/26_topup_client_ip.md)。

## 使用日志技能筛选与角标

- Default 与 Classic 的使用日志类型下拉菜单中均包含「技能」选项（不设独立筛选框），选择时请求参数为 `type=0&skill=used`（仅实际使用：始终装载或 generation 已读，排除发现/未用）。切换到其他类型或”全部”时清除 `skill` 参数，避免残留。Default 角标从时间列迁移到令牌单元格内分组标签右上角，悬停/聚焦显示详情。纯前端虚拟值，不作为数值发给后端；旧链接自动映射。列表、总数、分页与统计使用同一技能判定，普通用户限本人。
- 稳定性与限制：两套模板组件与 hook 测试、typecheck 和生产构建通过，未做真实浏览器视觉验收；见[使用日志技能筛选与角标](../workflows/2026-10/26_default_skill_logs.md)。后端 `skill=used` 参数支持与验证见[接口与实现记录](../workflows/2026-10/26_usage_log_skill_filter.md)。未涉及数据迁移，历史日志按需解析；大时间窗候选扫描可能达到超时，建议缩小范围。

## 普通使用日志分组名称选择

- Default 普通日志已对齐 Classic 的名称选组入口：选项展示当前分组名称，提交原有 code；历史 URL 的 code/alias 只解析展示，不改写查询值，无法解析时允许保留或清除原值。
- 当前实现按管理员全部日志视图读取 `/api/group/details`，普通用户和管理员仅本人视图读取 `/api/user/self/groups`。两接口顶层新增可选 `group_names`，保留已有 data 等字段；普通用户映射限定在可用分组稳定 ID 对应的 code/alias，不按名称授权。
- 仅覆盖普通日志，不扩展任务、绘图类别；名称展示不参与鉴权、路由或计费。Classic 已有下拉与稳定标识提交契约，本轮接口追加字段不要求其改写查询。
- 稳定性：原双模板门禁保留为历史快照；新发现的合法 `__proto__` alias 改名后名称丢失通过查询级关闭结构共享修复，独立原夹具 4/4、定向 34/34 通过，未改授权和筛选标识；真实后端认证、数据库、多实例、历史标识和生产角色隔离仍未验收。无数据库结构变更。
  见[工作记录](../workflows/2026-11/16_issue401_log_group_selection.md)与[历史迁移矩阵](../workflows/2026-11/16_issue401_migration_matrix.md)，
  不以矩阵中的基线缺口描述代替修复后的验收结论。

## 管理员返佣业务与 Root 配置分离

- Default 当前源码增加 `/affiliate-management`，允许 role ≥ 10 管理邀请关系、返佣记录、提现、风险、申请与异常检测；未登录和普通用户不能进入。业务面板复用 `/api/affiliate/admin/*`，不挂载 Root 配置表单或读取 `/api/option/`。
- `/system-settings/billing/affiliate` 继续由系统设置父路由限制为 Root（role 100），不扩大配置接口或其他设置权限。返佣金额、账务与审核 API 的后端契约不变。
- Classic `/console/affiliate-admin` 的普通管理员路径已同步拆分为仅业务面板，避免进入 Root option 加载和支付配置确认流程；Root 保留原配置入口。
- 返佣规则内嵌：Default `/affiliate-management` 为 Root 增加可编辑“返佣规则”标签，复用 `web/src/features/system-settings/billing/affiliate-config.ts` 的默认值、校验与 `affiliate-settings-section.tsx` 同一表单，不跳转系统设置；role 10 不挂载该面板、不请求 `/api/option`，后端权限不变。
- 推广页提示：新增 `affiliate_setting.promotion_notice_enabled`（默认关闭）、`promotion_notice_title`、`promotion_notice_text`，由 `GET /api/affiliate/summary` 的 `promotion_notice` 返回；双模板以纯文本渲染，进入或刷新推广页时弹出一次，本次访问内关闭后刷新数据或切换标签不再弹出；保存使用 Root 专用原子批量接口 `PUT /api/option/affiliate-batch`（复用 `model.UpdateOptionsBulk`，写库前服务端校验开关与标题 100、正文 2000 个字符）；弹窗先提示后协议，一次只出现一个；申请协议与结算不变，无数据库迁移。详见[推广页提示记录](../workflows/2026-11/18_affiliate_promotion_notice.md)。
- 稳定性：双模板实现、独立复审及统一构建门禁已记录；真实 Redis、多实例、生产角色权限和交易链仍未验收，
  不代表已部署；无数据库迁移。详见[返佣管理权限记录](../workflows/2026-11/16_issue401_affiliate_admin_access.md)。

## 供货商申请与分组报价（#408）

- Default 新增登录用户的“供货商申请”：站点与联系信息、1–20 个分组报价（平台、倍率、主要模型标签、说明）、“我的提交”；管理员在“供货商申请管理”分页搜索、查看并更新处理状态与内部备注，记录处理人与处理时间。
- 申请不可修改，更正需重新提交；相同内容重复提交返回 409，同一用户 24 小时最多 20 次；个人视图在类型上不含内部备注与处理信息；站点地址只保存、不请求；服务端拒绝明显的接入凭据形态文本；处理状态不创建渠道、不改分组与计费。
- 权限资源 `supplier_applications`（read/write）。审计动作 `supplier_application.review` 与 `supplier_application.message` 不写入备注、消息正文或联系方式。申请对话只追加，议价、优惠和测试额度不改余额、兑换码、渠道、分组或计费。契约见[供货商申请专题](supplier-applications.md)，验证见[申请工作记录](../workflows/2026-11/26_supplier_applications.md)和[对话工作记录](../workflows/2026-11/27_supplier_application_messages.md)。
- Default 与 Classic 的管理列表和「我的申请」都在站点名称旁显示未读数量；数量来自当前用户的 `unread_count`，只统计别人发来且尚未看过的对话，自己的消息和内部备注不计，没有未读时不显示。`message_count` 仍是对话总数，页面不再用它做标记。见[对话弹窗与未读数](../workflows/2026-11/28_supplier_application_conversation_dialog.md)。
- 管理详情的处理状态默认折叠，只显示当前状态和「编辑」，点击后才设置状态和内部备注；无写权限时没有编辑，但仍可打开对话并标记已读。详情本身不再内嵌对话。Default 与 Classic 的管理端和申请人端都用「对话」按钮打开独立弹窗，桌面更宽，窄屏接近全屏；消息列表在弹窗内单独滚动，发送区固定在弹窗底部。见[发送区固定与处理状态折叠](../workflows/2026-11/28_supplier_application_composer_dock.md)。
- 稳定性：消息数量与处理块位置用过 SQLite 路由测试、Default 组件测试和 Classic 组件测试。发送区固定和处理折叠只用两套前端组件测试核对，没有浏览器实播。此前 Chromium 验证只覆盖当时的申请流程，不能代替这些界面调整。MySQL/PostgreSQL 与 Redis 限流路径仍未实测，未部署。

## 工单中心

- Telegram 原通知在人工回复后自动编辑为「✅ 已回复」加原文，成功立即清理跟踪记录，失败异步有限重试；系统自动确认与单独改状态不打标。独立回执表不受最近五条历史清理影响，超过三天或停用/删除规则后自动清理；清理后及升级前已发消息不可补标。
- Default 与 Classic 的通知任务均支持 Telegram HTML 格式工具栏、emoji/变量光标插入、引用工单预设及服务端样例实时预览。普通文字前缀完全由模板决定，自定义模板保留；精确旧默认升级为 emoji、加粗和引用样式。预览、保存和发送共用格式与长度校验。数据、安全及验证边界见[本轮记录](../workflows/2026-11/02_ticket_notification_replied.md)。

- Classic 与 Default 数据看板提供待处理、进行中、已解决、已关闭四项工单统计，可点击进入对应筛选；管理员进入后台时的可关闭非阻塞浮动提醒仍仅统计 `open`。统计由管理员专用 `/api/ticket-admin/stats` 一次聚合获取，排除软删除并区分归档。稳定性与验收状态见[工单状态概览工作记录](../workflows/2026-10/26_ticket_status_overview.md)。

- Default 与 Classic 同时提供工单中心、管理员工单管理和登录控制台创建浮窗，支持分类、优先级、文本追问与回复、分页、关闭和重开。
- 用户接口始终按当前用户隔离，管理员管理权限为 tickets 的 read/write；管理员个人入口继续使用本人范围。
- 管理员可勾选当前页批量改状态或软删除（每批最多 100 条，版本检查、整批事务、二次确认）；创建后单次自动回复和超时归档均可配置，默认关闭。归档支持 1–3650 天和适用状态选择；使用主节点系统任务及数据库租约，保留操作审计与任务结果，详见[扩展记录](../workflows/2026-10/24_ticket_bulk_archive.md)。
- 工单通知只保留“新工单或用户追加回复”一个事件（`ticket_user_message`），沿用 Root 配置的任务、Bot、目标和模板；管理员回复不新发通知，而是编辑原群消息；状态变更不通知，不自动通知工单所有者。默认模板显示本次用户正文摘要（HTML 转义后 2000 字符预算），精确旧默认自动升级，自定义模板可加入 `{{content}}`；旧事件缺正文时显示历史提示，不回查后续消息。合并与迁移见[工单通知合并](../workflows/2026-10/25_ticket_notification_merge.md)，正文契约及验证见[正文通知](../workflows/2026-10/28_ticket_notification_content.md)。
- 工单、消息和操作审计三张业务表之外，本轮新增原通知回执关联表，不修改转发、计费或已有通知配置权限。技术契约与限制见[工单专题](tickets.md)，早期验证状态见[工作记录](../workflows/2026-10/22_ticket_center.md)。
- 稳定性：本地工单后端、双模板测试及生产构建通过；当前批量/归档补充测试和限制见[扩展记录](../workflows/2026-10/24_ticket_bulk_archive.md)。未部署，MySQL/PostgreSQL 实库及真实 Telegram 未在本地验证。

## 令牌分组数量上限

- Default 与 Classic 分组设置可读写 `MaxTokenAutoGroups`（令牌最大分组数）。默认 `0` 表示不限制，正整数启用数量上限；历史已保存的正整数不会自动改写。两套模板的设置编辑模式均说明 `0` 的含义。
- 创建和编辑令牌时，显式分组超过上限不能保存；两套模板在达到上限后禁用继续选择，并在已超限时提示「令牌最多可选 N 个分组」。自定义 Auto 快照继续使用同一上限。继承全局 Auto 顺序不受此值限制。
- 配置正数上限时，已保存的显式多分组超限令牌调用接口返回超限错误，不再显示「无效的令牌」。历史 Auto 自定义快照先按当前权限筛选，正数上限下截断，`0` 时保留全部有效分组。只修改启用状态不会改写分组。把分组数降到上限以内、提高上限或设为 `0` 后恢复可用。
- 稳定性与限制：沿用现有 options，无表结构变更。设置为 `0` 后回滚旧版前需恢复正整数，并检查已有令牌是否超限。原修复见[令牌分组数量上限](../workflows/2026-11/05_token_group_limit.md)，新契约和验证见[默认不限制](../workflows/2026-11/06_unlimited_token_groups.md)。

## 令牌周期额度与总额度

- Default 与 Classic 令牌表单都分为「周期额度（每日/每周，重置后恢复）」与「总额度（不随周期重置）」；总额度可独立设为不限，不关闭周期额度，也不影响账户余额。两套模板的列表都在总额度不限和有限时显示周期额度已用与上限。
- 接口沿用 `POST/PUT /api/token/`，字段 `quota_period`（`none`/`daily`/`weekly`）与 `quota_period_limit`（内部额度单位）；读取返回只读的 `quota_period_used`、`quota_period_reset_at`。账本字段由服务端维护，客户端不可写。
- 总额度与周期额度上限由 `common.MaxTokenQuota()` 校验（1e9 货币单位 × `QuotaPerUnit`），超限、负数、未知周期、选择周期但额度非正数均由后端拒绝，不截断；单次请求计费的 int32 饱和边界不变。
- 周期窗口按服务器时区的自然日或周一起始的自然周计算；预扣在单条条件 UPDATE 中检查并累加，结算差额与各类退款同步调整，无限总额度令牌同样受周期限制。
- 稳定性与限制：令牌表新增四个 AutoMigrate 列，旧令牌默认不设周期；真实 MySQL/PostgreSQL、Redis 多实例并发与浏览器渲染未验证。详见[令牌周期额度保存与总额度上限修复](../workflows/2026-11/26_token_quota_period_and_total.md)。

## 令牌按分组限定渠道

- Default 与 Classic 创建、编辑本人令牌时，管理员可为每个显式分组指定一个渠道 ID；留空沿用正常路由，同一令牌密钥无需切换。
- 接口新增 `group_channel_ids`，以稳定分组 ID 关联，界面展示当前分组名称；后端独立验证管理员身份、渠道启用状态及分组归属。
- 初选、亲和与重试均受限定，渠道不可用时不转向该组其他渠道；多显式分组保留既有跨组重试。带限定的令牌不兼容密钥渠道后缀，管理员降权后不能继续使用。
- 源分组有限定时，普通显式分组迁移的预览与执行均拒绝；管理员明确迁移至 `auto` 时清空所有显式绑定及限定。分组改名保留稳定 ID，删除/停用渠道不自动放宽限制。
- 稳定性与限制：当前为本地开发状态，新增 `token_groups.channel_id` 使用三库兼容的 GORM 字段；未部署、未调用真实供应商、未实测 MySQL/PostgreSQL。回滚旧应用前须清空限定，所有实例需同时升级后使用。
- 数据、权限、生命周期和验证记录见[令牌按分组限定渠道](../workflows/2026-10/23_token_group_channel_restriction.md)。

## Claude 流式用量与取消结算

- Claude 空流不把预估输入计作消费，缓存读写即使普通输入/输出为零也按实际字段计费。
- Responses 经 Claude Messages 转换时，下游取消保留已报告 usage 供统一结算；缺失 usage 的取消流不补估 token，预扣交外层退款，真实上游错误仍保留。
- 稳定性与限制：本地 HTTP 桥接及 SQLite 钱包测试已验证，随 `.352` 部署正式站三节点；生产仅验收部署和基础接口，未实测真实上游取消结算。成功流断开后最多继续读取 180 秒收集最终 usage，可能继续产生上游费用，原入口截止时间和空闲超时仍生效。窗口内无 usage 不补估；真实上游错误仍保留，不能保证追回全部上游成本。
- 契约与验证：[Claude 流式计费对账修复](../workflows/2026-10/21_claude_billing_reconciliation_fix.md)。

## 双主题当前连接与令牌使用状态

- 两套主题使用日志已在本地实现限量卡片及完整分页抽屉；Default 新增独立原生入口。
  Classic 令牌页按稳定令牌 ID 聚合正在使用数量；本轮 Default 令牌页补齐同一能力。
- Default 与 Classic 令牌活动均使用本人 `/api/log/self/active` 摘要，以 `summary.by_token` 的稳定 ID 标记使用状态，管理员令牌页也不查询其他用户。Default 新增活动令牌数、请求数与 `active_only` 服务端筛选，错误时明确显示上次快照；状态和名称等筛选组合、分页总数继续交给服务端。
- Default 本轮状态请求、缓存键与活动 UI 已接线；轮询、隐藏页面取消和身份切换隔离的实现及验证见[令牌状态与活动记录](../workflows/2026-11/16_issue399_400_token_activity.md)。历史独立复审和集成门禁已记录；新发现的首次列表在途刷新丢失已修复，pending 按主体与完整查询上下文隔离并在请求完成后合并补查，新独审活动 37/37 及原失败夹具通过；真实 Redis 多实例、认证和生产验收仍不属于下述历史 `.352` 发布结论。
- Classic 与 Default 管理员按用户 ID 或精确用户名提交日志查询时，面板与抽屉联动显示该用户连接，
  清空提交后恢复本人；后端独立管理员接口保护跨用户查询。本项随 `.352` 发布并部署正式站三节点，
  验证见[查询联动记录](../workflows/2026-10/22_admin_user_active_connections.md)。
- 连接结束后保留 5 秒并冻结耗时，结束记录不计入活动请求或正在使用的凭据；接口与轮询边界见
  [结束保留记录](../workflows/2026-10/22_active_connections_finished_retention.md)。
- 共享 Redis 汇总多节点请求，单进程支持无 Redis；无数据表迁移，不影响计费与路由。
- 历史验证：双主题日志查询联动随 `.352` 发布并部署正式站三节点。后端、两套主题组件与
  桌面/手机预览、Default 类型检查通过；`.352` 整合时双模板完整构建通过。
  连接观察超时可恢复，切换用户隔离旧数据；未调用真实收费供应商。
  契约、生命周期与覆盖边界见[当前连接专题](active-requests.md)。

## 令牌提示词中心与技能徽标

- Classic 与 Default 均提供模板管理、单技能/合集、稳定分类关联、装载分组白名单和令牌数量徽标。
  每令牌最多安装 8 包，合集计一项。普通用户只读名称、用途与分组摘要并安装、卸载，不返回正文。
  管理员可导入 ZIP、公开 GitHub 链接及仓库内置/在线商店，编辑未发布草稿后保存独立站点副本；
  商店原件与站点修改互不覆盖。分组关联用稳定 ID，界面展示当前名称。

- 管理员可为技能模板选择白名单分组或“不限分组”（`all_groups=true`）；不限分组模板只对启用的
  实际请求组生效，令牌仍需至少一个可用启用分组。旧节点按白名单语义处理，不会越权装载。
  设计、授权、兼容与验证见[技能白名单不限分组](../workflows/2026-10/26_prompt_skills_all_groups.md)。

- 管理员可按分类版本删除技能分类，其下模板在同一事务内改为未分类；模板版本不变，删除与编辑
  通过数据库行锁串行。接口、权限、回滚和验证见[技能分类删除](../workflows/2026-10/26_prompt_skill_category_delete.md)。

- 管理员编辑模板白名单时可按分组标签筛选并跨标签多选，卡片默认折叠支持分组名称；标签不参与授权，
  保存仍使用稳定分组 ID，运行时沿用当前请求分组白名单校验。Default 与 Classic 均提供此交互，
  不新增数据迁移；见[分组标签选择记录](../workflows/2026-10/27_prompt_center_group_tag_picker.md)。

- 随 `.355` 发布并滚动部署生产的同批内容还包含 Responses 网关 Skills 未识别工具的安全回退、工单通知合并、
  使用日志技能徽标与 Default 任务日志路由兼容。生产基础验收完成，但未验证 Skills 开关开启时的真实收费请求；
  发布与部署证据见[.355 发布记录](../workflows/2026-10/26_gateway_skills_355_release.md)。

- GitHub 导入的技能记录来源链接、引用、目录、固定提交与内容指纹，管理员可在两套模板一键更新；
  站内改过正文或引用文件时需确认覆盖，更新保留模板 ID、安装、分类、白名单、上下架和排序。
  管理员可按版本删除技能，同一事务自动卸载令牌安装并清理文件。用途说明不再限制 512 字符，
  仅保留单条 16 KiB 字节存储边界。状态：#329 已合入并更新 zzapi，未发布新版本，正式站未更新；
  见[GitHub 技能来源与更新记录](../workflows/2026-10/25_github_skill_updates.md)。

- 使用日志按 `other.prompt_templates` 与 `other.gateway_skills` 显示技能徽标：已使用技能、可用但未读取、
  私有发现轮次三种状态，管理员与日志所属用户可见，只显示技能名称；纯前端，见
  [使用日志技能徽标](../workflows/2026-10/25_usage_log_skill_badge.md)。

- 新建与导入默认 `load_mode=progressive`：按需读取主文件及技能根内引用文本，单文件 8 MiB、
  每包 64 MiB、最多 128 个技能和 4096 个文件；ZIP 最多 16 MiB。元数据、文件索引与正文分块
  独立入表，旧表正文列不做破坏性变更。旧空值及 `eager` 保持全文装载，单模板 32768 字节、
  令牌启用全文项合计 65536 字节，全文合集仍为 2–16 项。

- 网关使用私有非流式阶段完成目录发现、主文和引用资料读取，提供
  `newapi_skills_list/read/files` 三个内部工具；每次按实际分组、安装状态及版本重新授权。
  文件每页 24 KiB，每客户端请求实际读取总计 192 KiB，最多 8 轮、每轮 8 次工具调用；目录也分页。
  用户询问可用云端 Skills 时，正式生成会收到本轮成功查询且重新授权后的目录元数据，最多 64 KiB；
  目录查询只用于发现，不触发已使用技能标识。正式生成恢复客户端工具、流式及输出格式，只注入目录元数据和已读页。
  私有阶段误返原请求声明的客户端函数工具时，读取非空正文后安全交接；未读时仅纠偏一次，
  连续误调、未知工具及无效调用 ID 仍拒绝，不执行或泄漏私有客户端调用。
  范围与回归见[Agent 工具交接修复](../workflows/2026-10/24_gateway_skill_client_tool_handoff.md)。
  不执行脚本、客户端工具或二进制附件，
  不向客户端 Agent 本地下载技能。两套管理员表单均支持模式、用途说明与引用文本编辑。

- 私有模型轮次和正式生成分别计费、写用量日志与渠道指标，会增加延迟和费用；
  已结算读取不会因最终生成失败退款。跨组重试清除上一组资料，正常读取不计渠道重试。
  渐进式场景的 Responses 非 `null` `conversation`、Gemini 非空或类型不合法的 `cachedContent`
  在上游调用前返回 400 并退回入口预扣；应传完整 `input/contents`，不承诺
  `previous_response_id` 历史续接。
  旧全文及未命中渐进式技能的请求不受这两项新增限制。

- 云端 Skills 性能优化纳入 `.353` 发布范围，已通过本地后端回归与容量对照：请求内复用不可变历史；正文页及目录使用
  32 MiB 估算容量的版本化 LRU，不缓存授权；读取采用共享锁并保留正式生成前的实时复核。
  每实例默认最多 16 个在途技能请求、原始正文合计 32 MiB，超额明确返回 429/413；
  每次路由尝试的整个私有发现阶段默认期限 120 秒、每轮输出最多 4096 Token，正式回答不继承私有期限与输出上限。
  本地限额或技能期限不计为渠道故障，不触发渠道重试；已结算轮次仍保留消费。
  这是资源保护和重复处理优化，不减少模型调用轮数，也不承诺进程内存硬上限。
  配置、缓存一致性和验证范围见[性能优化记录](../workflows/2026-10/24_gateway_skill_performance.md)；本次优化尚未部署。

- Classic 与 Default 操练场增加本人 API 令牌切换；服务端核验归属后沿用正式鉴权、模型发现、
  分组与提示词装载链路，普通/流式消费归属该令牌并扣减额度；账号模式保留。
  不增加数据库字段，不向浏览器读取完整密钥。验证与部署状态见[工作记录](../workflows/2026-10/22_playground_token_switch.md)。

- Classic 操练场按 Default 布局调整为居中对话与底部输入，参数和调试按需打开；保留自定义请求、
  附件及配置导入导出。此布局改动仅涉及 Classic，Default 沿用现有布局，见[对齐记录](../workflows/2026-10/22_classic_playground_alignment.md)。

- 渐进式当前状态：完整后端、四协议本地模拟端到端、auto 跨组计费、双 UI 回归与构建通过；
  携双前端的 Go 1.25.5 amd64 本地整包构建通过。已按顺序更新 zzapi 三节点，保持 Classic 主题；
  同镜像通过隔离 SQLite 真实 HTTP 模拟验收，以及 PostgreSQL 15 应用数据恢复与迁移验收，
  公网版本和资源一致。真实供应商与 MySQL 实库未实跑，正式站未更新；恢复时排除维护扩展
  `pg_repack` 的两条定义/注释 TOC，边界及部署证据见[zzapi 记录](../workflows/2026-10/23_skill_progressive_zzapi.md)。
  契约、数据生命周期、回滚和验证边界见[令牌提示词中心](token-prompt-center.md)与
  [渐进式主记录](../workflows/2026-10/23_skill_progressive_disclosure.md)。只有尚无渐进式业务模板时
  才回滚旧程序，单纯停用或卸载不足以保证旧管理页面兼容；已有数据时向前修复并保留资料备份。

- 此前全文版本已部署 zzapi，随后随 `.352` 部署正式站三节点；历史隔离实例和在线商店验收见
  [.352 记录](../workflows/2026-10/22_skills_claude_352_release.md)及
  [生产部署](../workflows/2026-10/22_maolaoapi_352_deployment.md)。正式站仍为该全文版本。

- 共享契约：密钥、模型 ID 和原路由权限不变，管理摘要增加模式与技能用途；
  渐进式一次客户端请求可能产生多轮用量及消费日志，接入方不能再假定一次请求只有一次上游消费。
  最终客户端工具调用继续交由 Agent 执行，不新增客户端工具执行服务。

## 网关云端 Skills 回复标识

- 四类对话接口在正式回答实际读取技能后返回
  `X-New-API-Gateway-Skills` 元数据；未请求元数据模式时，成功的文本回答末尾追加
  `☁ 网关云端 Skills · 技能名称`。响应头只包含模板/技能稳定 ID、版本和标题，不包含正文或资源路径。
- Classic 与 Default 操练场请求 `X-New-API-Skills-Display: metadata`，将成功助手消息底部显示为云端
  Skills 徽标；徽标元数据会随本地历史和导出保存，但不会进入下一次发送给模型的消息正文。
- 失败、取消、纯工具调用、未实际读取技能和跨组重试不会伪造标识；外部 Agent 需要自行处理尾部文字可能对
  严格 JSON 或代码输出造成的影响。契约与验证见[网关云端 Skills 回复标识](../workflows/2026-10/24_gateway_skill_response_indicator.md)。

## dev 网关兼容与后台管理更新

- 流式请求区分正常结束后的取消收尾，并增加上游响应头超时；容器过载拒绝要求连续约 15 秒超阈值，请求体超限返回 413。
- Claude 保留显式零用量和缓存用量；DeepSeek 增加 V4.1 与 Search/Fetch，独立端点按模型按次价结算并写消费审计。
- 自定义上游错误按请求作用域脱敏；Sub2API 查询区分钱包余额和无限订阅标记。
- Default 后台提供渠道优先级拖拽与并发冲突保护、个人通知类型开关。Classic 不包含这些新增交互，通知中心渠道自动启停投递规则保持独立。
- 状态：组合后端、独立 relaykit 与前端回归验证通过；真实上游工具请求及 MySQL 实库未在此次部署中调用。部署与回滚证据见 [dev 整合记录](../workflows/2026-10/18_dev_integration_zzapi.md)。

## DeepSeek Search/Fetch

- `/v1/tools/search` 与 `/v1/tools/fetch` 以路由模型名的按次价格预扣和结算，管理员自定义价格及显式零价生效；聊天内置 Search/Fetch 的工具附加费独立配置。
- 返回非空搜索结果或页面内容才进入成功消费链路，免费成功调用也保留消费日志、请求次数和渠道成功指标；空结果退款，上游错误及读取失败不成功结算。
- 稳定性与限制：本地覆盖钱包/令牌扣费、分组倍率、审计统计、Claude Search 与 Responses 计数；未验证生产真实付费请求。协议与验证见[工作记录](../workflows/2026-10/07_deepseek_v41_search_fetch.md)。

## Classic 令牌界面与多人兑换码

- Classic 列表采用紧凑表格、清晰字重层次与行操作菜单；支持名称/密钥片段和服务端状态筛选，保留取密钥安全与分页契约。
- 创建/编辑令牌支持标签直接选组、已选分组标题、草稿自由追加和保存前独立组冲突校验。
- 多人兑换码恢复每码上限与独立用户使用记录，额度发放和名额占用在同一事务内；旧兑换码默认 1 次。
- 文档：[令牌列表](../workflows/2026-10/02_classic_token_search_api_info.md)、[创建编辑](../workflows/2026-10/01_token_tag_picker_flow.md)、[兑换码修复](../workflows/2026-10/01_redemption_multi_user_limit.md)。本地 SQLite 验证不能代替 MySQL/PostgreSQL 实跑。

## 双模板令牌 API 地址入口

- Classic 与 Default 均在令牌操作区提供地址入口，读取公开 `/api/status` 的 `api_info_enabled`、`api_info` 和 `server_address`；有效列表展示线路名称、原始 URL 与说明，复制时不追加 `/v1` 或改写路径。
- 列表关闭、缺失或没有有效地址时，优先使用去除首尾空白的站点地址，再回退当前 origin；不读取完整令牌密钥。Default 原生 Popover 提供加载、失败原地重试、复制成功/失败反馈和键盘焦点恢复。
- 稳定性：Default 10 项组件回归及独立审查通过，模拟 API 浏览器覆盖三种宽度、深浅色与精确复制；Classic 3 项真实组件回归通过，仅覆盖线路展示及两级回退，不能扩大为 Classic 复制、异常或手机主题验收。上述构建只对应历史快照；新发现的 HTTP 500 全页跳转已修复，保留认证/取消语义，原失败夹具及新独审地址 16/16 通过，返修后的发布构建待执行；仍无本轮生产验收或数据迁移。历史说明保留在[Classic 记录](../workflows/2026-10/02_classic_token_search_api_info.md)，本轮证据见[地址入口记录](../workflows/2026-11/16_issue402_api_addresses.md)。

## 分组标签

- 在分组与模型定价设置中维护标签、说明、排序、多个内置 Logo 或图片 URL，并按稳定分组 ID 绑定。
- 令牌创建/编辑按标签过滤候选组，保留跨标签多选、排序、独立分组和倍率保护；分组管理支持同样的分类过滤。
- 标签只管理展示分类，不改变鉴权、路由、计费或分组可用性；普通用户只能获取可用分组对应的标签。
- Default 在分组定价页补齐同一标签 CRUD、分组表筛选和本人令牌标签筛选。普通用户只消费 UserAuth 分组响应的 `group_tags`，不请求管理接口，不改变授权集合和原有提交契约。
- 稳定性与限制：Classic 历史交付已有后端、组件、构建及浏览器验收；P2 Default 固定 head `ffe87461b566dec6ea9119971d5d51f07b83a1f9` 已独立通过 P2 与 P1 回归，见[独立报告](../workflows/2026-11/11_default_group_tags_qa.md)。正式基线集成另行验证和交验，不以历史结论代替。复用既有数据表，无本轮迁移或后端契约变更，未作生产验证。
- 契约见[分组标签](group-tags.md)，本轮命令、工具版本与限制见 [Default 实施记录](../workflows/2026-11/11_default_group_tags.md)。

项目名称、默认仓库和镜像规则见[项目身份与发布兼容](project-identity.md)。本次更名无新增业务功能或数据迁移；旧版自更新资产名和配套仓库保持兼容，首次新镜像发布仍须验收。

## 双模板供应商自定义 Logo

- 在供应商图标字段填写 HTTP/HTTPS 图片直链，或继续使用 LobeHub 图标名和链式参数。
- Classic 与 Default 已有的供应商 Logo 展示入口通过各自共享加载器支持图片 URL；Default 覆盖模型名称与供应商列、模型广场及榜单。Default 渠道供应商筛选和编辑选项目前仅显示名称，不计为 Logo 展示入口。URL 图片等比显示，加载失败时回退默认占位，切换 URL 后可重新加载。
- 复用现有 `icon` 字段，最长 128 字符，无数据迁移；保留 LobeHub 名称、链式参数及项目自定义图标。图片由浏览器直接请求并禁用 referrer，HTTPS 页面加载 HTTP 图片可能受到浏览器限制。
- 本轮独立复审通过 Default 13 项与 Classic 14 项组件回归，另有 7 项独立边界测试通过，并核对本地模拟接口的桌面、手机及浅深色截图；未执行真实登录后的跨模板保存或生产图片验证。统一 Default typecheck、Default build 和 Classic build 已通过；真实认证、跨模板保存和生产图片仍未验收。保留 [Classic 供应商自定义 Logo URL](../workflows/2026-09/23_classic_vendor_logo_url.md) 历史记录，本轮实现与限制见 [Default Logo 补齐](../workflows/2026-11/16_issue398_logo.md)。

## xAI 渠道 Messages 兼容

- 缓存分片合并及回包/账单一致性见[缓存工作记录](../workflows/2026-09/21_xai_cache_usage_merge.md)；解析修复不能保证供应商实际缓存命中。
- `/v1/messages` 经 xAI 渠道转为 Chat Completions，上游返回转换回 Anthropic JSON/SSE，保留 Grok 缓存用量和现有结算口径。
- 强制 Responses、请求体透传及原生端点沿用既有优先级；无需新增开关或数据库迁移。
- 实施与验证状态见[工作记录](../workflows/2026-09/21_xai_messages_compat.md)，实际供应商兼容性和上线需单独验收。

## 账号绑定身份保持

- 邮箱验证码绑定仅更新当前现存用户的 email，保留并发权限、状态、分组和资金修改；验证码匹配后一次性消费，不创建用户或签发新会话。
- Classic 社交绑定使用明确的 bind flow，个人资料回读保留同一用户的会话凭证；Default 邮箱补齐 Turnstile，Classic 邮箱使用参数编码与失败恢复。
- 实现、双模板差异、测试和现场核验边界见[工作记录](../workflows/2026-09/21_account_binding_identity.md)。不自动合并历史账号，不改变数据库 schema；验证码仍为进程内存储。

## 复制令牌密钥独立限流

- 单条和批量读取密钥按认证用户共用独立额度，默认每分钟 60 次请求，避免同出口登录等关键操作导致首次复制 429。
- 继续执行 GA、UserAuth、密钥归属和批量上限；Default、Classic 无需修改，无数据库迁移。配置与验证见[工作记录](../workflows/2026-09/21_token_key_copy_shared_limit.md)。

## 主程序上游模型检测来源

- 主程序仅以响应正文声明的模型执行现有白名单、连续容错和关渠规则；不再采集 `x-codex-safety-buffering-faster-model` 作为自动检测证据。
- 历史来源列、API 字段、旧头部记录和通知保留。两套模板继续使用同一后端，现有模块无需更换；模型声明仍不能证明底层模型身份。
- 当前实现与验证见[旧响应头路径清理](../workflows/2026-10/26_model_guard_header_cleanup.md)；[原头部实现记录](../workflows/2026-09/21_model_guard_codex_header_evidence.md)仅作历史证据。

## 上游模型校验渠道 ID 白名单

- `upstream-model-guard` 模块 `0.2.1` 在 Default、Classic 支持直接填写渠道 ID，与现有勾选白名单同步；沿用 `excluded_channel_ids` 与 Root 权限。
- 记录：[白名单 ID 输入](../workflows/2026-09/21_model_guard_allowlist_ids.md)。外置模块需单独安装，宿主更新不会自动升级旧模块。

## 渠道强制使用 Responses 上游

- 文档：[渠道强制使用 Responses 上游](channel-force-responses.md)。
- 配置：`setting.force_responses` 默认关闭，要求渠道敏感写权限，Default 与 Classic 均提供入口。
- 能力：支持 OpenAI、Azure、xAI、Codex、Sub2API、NewAPI 渠道，将 Chat/Claude/Gemini 对话转为 Responses，客户端返回协议不变。
- 边界：开启后优先于透传与 Responses 转 Chat；其他业务端点不改路由，无数据库迁移。
- 缓存契约：Chat ↔ Responses 保留 `prompt_cache_options`、`prompt_cache_retention`、`prompt_cache_key`；Responses 原生快照中的缓存读取详情在结算时补入缺失的 Prompt 缓存字段，已有值和显式零优先。
- 本地回归：[WorkBuddy 缓存结算与参数修复](../workflows/2026-10/30_workbuddy_responses_cache_billing.md)，覆盖流/非流转换、快照隔离、倍率折扣与表达式 `cr`；上游缓存策略是否实际生效仍需供应商验收。
- 稳定性：复用既有协议转换和结算链路；供应商实际兼容性与线上部署需单独验证。

## 管理员管理请求豁免与令牌取密钥限流隔离

- 文档：[管理员管理请求豁免与令牌取密钥限流隔离](../workflows/2026-09/19_token_key_read_rate_limit.md)。
- 能力：有效 Admin/Root 的管理 Session/PAT 豁免后台 GA、搜索、取密钥及管理路由 CT/UC；普通用户取密钥按用户共用 60 次 / 60 秒额度，与登录 CT 隔离。
- 边界：保留登录等独立限流及个人 2FA/Passkey、账号自身、会话管理和个人返佣接口的 GA，模型调用不变；管理员仍只能读取自己的令牌，RootAuth、权限、渠道密钥二次验证、批量 100 条上限与无缓存响应不变。纯 Cookie 扩展资源仍经过 GA。
- 稳定性：已实现后端，部署状态以交付为准；多节点共享计数需要 Redis，客户端遇 429 仍须退避并禁止使用空密钥。

## 认证上游兼容阶段

- 文档：[认证差异与兼容矩阵](../workflows/2026-09/15_auth_upstream_compatibility.md)。
- 已实现：Telegram/Passkey 在途流程固定会话版本、登录签发固定认证快照版本、Argon2id 有界读取、PAT POST 生成与 DELETE 撤销及指纹审计。
- 稳定性：后端兼容增量；保留本地 JWT、Session、刷新轮换和撤销栅栏。尚未切换 Telegram OAuth、统一多因子登录、操作上下文单次 proof 或独立审计存储；不能当作上游安全功能全部移植。

本页登记可复用的二次开发能力及其稳定性边界。长期专题文档负责接口和行为契约，
`docs/workflows/` 负责单次问题的根因、变更和验证记录。

## Default 上游兼容增强

- 文档：[Default 上游集成](../workflows/2026-09/15_default_upstream_integration.md)。
- 稳定性：按功能片段复用现有接口，新增渠道多 Key 策略编辑、统一供应商识别、用户额度详情和兑换码可选导出；修复时间规则、价格单位和 Passkey 能力检测。
- 权限和生命周期：多 Key 策略要求敏感写权限；兑换码导出只使用本次创建响应，在浏览器内生成文件，默认不下载，关闭后清空导出状态。
- 兼容边界：保留 ApiPanelWatch、通知、实际响应模型、稳定分组标识和本地定价展示。Classic 不在范围内；新任务插件、供应商版本化管理和安全中心需后端协同。
- 已知限制：不迁移既有计费表达式；真实 Passkey 硬件和生产页面交互未验证。

## TokensPro overview 渠道并发对齐

- 文档：[TokensPro overview 渠道并发对齐](tokenspro-overview-concurrency.md)
- 稳定性：复用现有渠道 Key、代理、系统任务租约和自动禁用状态；不新增协议类型。
- 权限与默认值：渠道编辑保存 `settings.tokenspro_overview_sync_enabled`；旧渠道默认关闭。
- 边界：只写 `concurrency.allowed`；失败不改并发和状态；`allowed=0` 因 NewAPI `0=不限制` 会自动禁用，恢复时不打开人工禁用渠道。
- 关闭任务：`TOKENSPRO_OVERVIEW_SYNC_TASK_ENABLED=false`。

## Classic 渠道编辑密钥聚合

- 文档：[Classic 编辑渠道启用密钥聚合](../workflows/2026-09/09_classic_channel_multikey_conversion.md)
- 稳定性：复用现有多密钥调度、管理和持久化；通过显式转换字段原位升级单密钥渠道。
- 权限与默认值：要求渠道敏感写权限；默认保留旧密钥并追加，支持显式覆盖及随机/轮询。
- 边界：本期只提供 Classic 编辑入口；Codex、Vertex 与 io.net 托管渠道不提供转换。
  不支持反向转换；并发编辑沿用现有渠道更新语义，应避免同时编辑同一渠道。

## 任务插件迁移

- 文档：[任务插件迁移](task-plugin-migration.md)。
- 稳定性：原生任务计费修复与迁移分析阶段；在线任务继续使用原生适配器，本补丁不引入 JS 运行时或依赖。
- 契约：AtlasCloud 渠道类型 61、源任务多分组授权与别名/继承、退款认领及补偿状态保留。
- 历史阶段限制：该迁移分析当时未接入 JS 管理和路由；当前能力以紧接其后的「官方 Task Plugin 并存接入」及专题文档为准，仍禁止直接全量切换。

## 官方 Task Plugin 并存接入

- 2026-09-23 补齐官方原生 JSON submit 路由、同步结果和 `retainResult:false` 的用量表达式计费，原样验证官方 TypeSafe Jev 插件。

- 2026-10-26 TypeSafe Jev 同步成功不写入任务记录，保留结算、消费日志和性能统计，详见[免任务记录工作记录](../workflows/2026-10/26_jev_no_task_history.md)。

- 稳定性：本地模拟上游验证阶段，未部署；尚未开放原生 query/dynamic、源任务 intent 或异步 UsageFacts。Default/Classic 页面未改动。

- 契约与限制见[原生同步链路](../workflows/2026-09/23_task_plugin_native_sync.md)及[用量计费](../workflows/2026-09/23_task_plugin_usage_billing.md)。

- 文档：[官方 Task Plugin 接入计划](../workflows/2026-09/15_official_task_plugin_integration_plan.md)。

- 范围：接入官方 JS 插件运行时、内置插件、插件管理和市场；现有 `/extensions` 二开扩展与 Go 原生任务适配器继续保留。

- 默认值：插件总开关默认关闭；体验阶段只在 zzapi 显式启用，不触碰 maolaoapi。

- 编号：AtlasCloud 的渠道类型 61 保留，官方插件使用 62，渠道绑定为 `setting.task_plugin_key`。

- 当前合同：内置与自定义插件列表、详情、版本、Root 上传/激活/启停/删除；专用 runtime/status 管理总开关。支持多个插件源，默认官方源与独立公开的 `moeacgx/maolaonewapi-plugins`。安装后需显式激活，关闭只阻新提交，历史任务仍按版本和 hash 轮询。

- 稳定性：内置 JSON/per_call/认证内联资源阶段已通过组合验证。多源安装与发布契约见[工作记录](../workflows/2026-09/16_task_plugin_sources_plan.md)，其部署状态以最终交付为准；S3、匿名资源签名、在线试运行、UsageFacts、真实远程媒体与完整 TokenAuth 端到端仍待验收。

## Responses WebSocket 集成准备

- 文档：[Responses WebSocket 专项集成审查](responses-websocket-integration.md)。
- 稳定性：仅 relaykit DTO 兼容准备，运行时未接入，不是可用功能或发布候选。
- 开关：渠道 `setting.responses_websocket_enabled` 缺失为 false；本分支设为 true 仍不启用入口。
- 限制：上游审计、计费错误传播、失败指标和错误替换尚未满足本地契约，禁止直接摘取整个提交。
- 范围：本阶段交付 DTO 与协议夹具，不增加界面；完整运行时启用前必须分别实现 Default 与 Classic 的开关、保存恢复和权限验收。无数据库迁移，无部署。

## 福利营销时效额度券

- 文档：[福利营销时效额度券](benefit-vouchers.md)
- 稳定性：已实现后端模型、接口、组合计费、流水和 Default/Classic 页面；流水接口与数据表
  继续用于审计，但两套福利页面不提供查看流水入口；两套模板共享 API、权限、状态和金额语义，
  组件实现与构建链路分别维护。
- 金额边界：内部 quota 是计费真值；页面按当前 USD/CNY/CUSTOM/TOKENS 展示，货币精度
  为 0.01，Tokens 为整数。领取门槛始终是 CNY 实付快照，API/表单仅按当前展示单位回显。
- 删除边界：福利活动、兑换码、优惠码均使用管理员鉴权、关键操作限流和 GORM 软删除；
  单条/批量接口返回 `deleted_ids` 与逐项 `skipped`，保留券、流水、充值、支付和审计。
  优惠码已有 payment reservation 允许通过 `Unscoped` 回调结算，删除后禁止新 reservation。
- 关键契约：活动可绑定多个分组，库存按活动共享，每人每活动限领一张；同一张券在绑定分组间共享余额。
  显式分组才抵扣；该分组可用的福利券按失效时间跨券抵扣；福利券 -> 订阅 -> 钱包；请求总价仍受
  token 上限约束；所有差额和回滚按 `request_id` 幂等。组合结算补偿使用独立
  `settle_rollback` 流水和原 `request_id`/类型组合键，日志 breakdown 关联
  `activity_id`/`voucher_id`/`request_id`/`log_id`。多券时 `voucher_id` 为历史兼容的首券摘要，
  不代表总消费归属；逐券账务以福利券流水为准。
- 已知限制：分组并发为单实例进程内限制；活动不自动复制渠道/分组；本期不覆盖图像和
  视频异步任务；Default 前端时间输入显式按 `Asia/Shanghai` 转换为 Unix 秒。管理 HTTP
  接口使用服务端时间，忽略请求体 `now`；活动结束仅关闭领取，个人券领取后的有效期独立
  按配置小时数计算，不受活动结束时间截断；有效期对外按小时传输，数据库内部仍按
  秒保存并兼容旧秒字段；模型层时间参数仅供内部测试注入。
- 历史清理：福利活动批量删除仅归档可安全删除的 `draft`/`ended`/`terminated` 状态；
  `published`、`paused` 或仍有 active 券的活动会跳过。兑换码、优惠码批量删除和失效清理
  沿用软删除并保留账务关联。三类资源接口均限制最多 500 个 ID，仅管理员可调用，返回
  实际删除和跳过原因；重复请求幂等。
- 迁移/回滚：额度配置迁移仅填充空字段，按历史 share quota 或旧人民币语义转换一次，
  异常即停止且不补零；回滚代码保留新表、新列和历史审计，不物理删除数据。本次 Task 10
  未部署、未推送、未开 PR，发布前仍需重新确认目标实例与回滚方案。

## 扩展模块

- 在线分发：独立公开 `maolaonewapi-extensions` 仓库维护模块源码、版本 ZIP 和完整性目录；主仓库关联扩展与任务插件仓库。
- 安装入口：Default/Classic Root 点击“在线模块”打开弹窗后读取固定目录、下载并核对包，宿主复核 hash/ID/版本后安装；保留手工 ZIP，不增加服务端远程下载或自动升级。实现与边界见[工作记录](../workflows/2026-09/18_extension_repository_marketplace.md)。

### 上游模型校验

- 增量能力：渠道白名单、同渠道跨模型/分组共享连续计数（默认 2 次）、匹配清零、每请求去重与异常计数记录。`0.2.0` 需配套宿主及数据库迁移，尚未部署。

- 文档：[上游模型校验扩展](upstream-model-guard.md)

- 能力：可上传 ZIP 独立安装的外置扩展，配置分组和模型对应关系，根据响应声明模型关闭整条出错渠道，并通过通知中心 Bot 通知；支持 Default 与 Classic。

- 展示契约：页面与通知使用分组显示名称，记录 API 以 `group_name` 提供当前名称；ID/code 保留原有匹配与提交语义。名称修复尚未重新部署。

- 稳定性：已完成本地后端与双模板验收，并在 zzapi、maolaoapi 各三节点部署 `.326.guard.1` 配套补丁、验证真实 ZIP 上传和插件表；默认关闭，尚未正式发布。

- 限制：需要包含 `channel.upstream-model-guard` 的宿主支持，旧版不能单独上传即用。只覆盖正常转发中已采集响应模型的协议，模型声明不构成真实性证明；不撤回当前响应或在途请求。多节点须逐节点开启模块。

扩展模块的宿主、权限和通知契约见 [扩展模块开发](extensions.md)。新增扩展页面时，
同时登记入口、所需角色、API 和回滚方式。

### 对话归档扩展

- 文档：[扩展模块开发](extensions.md#%E5%AF%B9%E8%AF%9D%E5%BD%92%E6%A1%A3%E6%89%A9%E5%B1%95)
- 用途：按多个分组和指定用户 ID 采集对话，清洗后在线预览，用于定位蒸馏或异常调用来源。
- 稳定性：宿主 API 与 `native v1` 页面同版本发布；未识别协议仅保存可识别的有限文本，不保证还原原始请求。
- 权限：仅 Root 可读写配置和查看详情，普通用户及普通管理员不能通过扩展路由绕过安全审计权限。
- 数据契约：筛选字段为空表示不限制；用户与分组筛选同时存在时使用 AND。分组使用稳定 `GroupCode`，用户 ID 必须为正整数并受数量上限约束。
- 安全边界：不存储 Authorization、Cookie、请求头、URL 查询、媒体和工具 schema；详情仅返回纯文本消息。配置 `CRYPTO_SECRET` 后正文以 AES-GCM 密文存储，服务端解密后再返回。
- 生命周期：正文在筛选命中后才物化，进入持久队列后按保留期和全局最近会话数清理；默认最多保留 1000 条，设置降低后立即裁剪最旧记录。Root 可二次确认清空全部归档，采集未关闭时后续匹配请求会重新写入。
- 已知限制：普通流式请求只采集认证阶段可见的请求正文；OpenAI Realtime 会在连接结束时合并文本增量，但不保存音频、工具定义或 `*.done` 汇总正文；不支持的协议不会保存完整原始 JSON。

## 安全审计上游策略来源

- 文档：[安全审计上游策略来源泛化](../workflows/2026-09/03_security_audit_policy_sources.md)
- 稳定性：内置安全审计支持将 `cyber_policy` 与 `biological_risk` 分别记录为审计来源，
  并通过 `policy_action_sources` 选择哪些来源参与当前会话屏蔽和阈值自动禁用；缺失该字段的
  旧配置按仅启用 `cyber_policy` 兼容。
- 边界：生物风险仅接受状态码 500 且包含固定上游提示的响应，普通 500 不触发；来源选择不
  改变审计事件记录开关及渠道/分组作用范围。Default 与 Classic 分别维护页面，但共享同一
  管理 API 契约。

## 通知中心

通知事件、模板变量和投递边界见 [通知中心与模块事件](notifications.md)。新增通知事件
时必须补充事件类型、权限、失败重试和敏感字段处理。

## 发票中心

发票中心支持用户选择近 30 天内符合条件的充值或订阅订单申请发票。开票服务费必须
使用已配置的外部支付方式；零服务费申请不产生实际支付。

### 关闭与订单释放

- 关闭申请会事务性释放来源订单，仍在近 30 天范围内的订单可重新选择并合并开票。
  关闭不自动退还服务费，原支付状态、费用和历史明细保留。
- 已关闭是终态，只能补充管理员备注，不能重新开启；重新申请使用新的发票记录。
- 合并订单明细存入 `invoice_records.closed_orders_snapshot`（可空 TEXT），原 `orders`
  响应字段仍返回关闭前的明细。来源关联唯一索引继续保护未关闭申请。
- 订单列表、预览和提交按当前用户惰性释放旧版关闭申请（包含软删除关闭记录），不影响
  新申请占用。未关闭的软删除记录仍阻止重复开票。
- Default 与 Classic 管理页均限制已关闭记录重开。详见
  [关闭后释放订单工作记录](../workflows/2026-10/26_invoice_closed_order_release.md)。

### 开票服务费支付

- 管理员在支付设置的 `PayMethods` 中配置易支付方式。默认配置包括 `alipay`（支付宝）
  和 `wxpay`（微信）；发票中心不提供账户余额支付。
- 易支付地址、商户 ID、商户密钥和支付合规确认均满足条件时，发票配置接口返回已配置
  的易支付方式。
- Default 与 Classic 发票中心读取 `/api/user/invoice/config` 的 `pay_methods`，按
  配置展示支付选项，并将所选类型提交到 `/api/user/invoice/payment`。
- 外部支付订单先进入 `payment_pending`；易支付回调验签、金额和商户快照校验通过后
  才转为 `pending` 待开票状态。
- 事件键、支付订单号和回调处理保持幂等。未完成支付的申请不会触发待开票通知。

### 相关接口

- `GET /api/user/invoice/config`：返回发票配置、可用支付方式和支付链信息。
- `POST /api/user/invoice/preview`：计算所选订单的开票服务费。
- `POST /api/user/invoice/request`：仅用于零服务费时提交申请；正服务费请求会被拒绝。
- `POST /api/user/invoice/payment`：创建外部支付申请并返回易支付收银台参数。
- `GET|POST /api/invoice/epay/notify`：易支付异步回调。
- `GET|POST /api/invoice/epay/return`：易支付同步回跳。

### 模板边界

Default 使用 `web/src/features/invoices`，Classic 使用
`web/classic/src/components/invoice`。两套模板分别读取相同的后端配置契约，修改一套
不会自动改变另一套。Classic `/console/invoice` 使用全宽透明玻璃业务卡片，表格按容器宽度
铺开，右侧操作列固定在边缘。

## 官方 JS Task Plugin 后端

见 [官方任务插件后端](official-task-plugins.md)：内置与 Root 上传的自定义插件并存，默认关闭；
生产资源签名和 S3 尚未验收。

- 性能过滤：`task-performance-filter@1` 为原生同步任务提供可选失败样本过滤钩子，
  日志、成功用量和账务保持；参数白名单、固定插件版本及 100ms 排队/执行预算由宿主控制。
- 稳定性：本地引擎及 Gin 路由回归验证；尚未发版/部署。不包含异步轮询性能、
  未选渠请求插件回调或历史数据修正。依赖该能力的插件必须先升级宿主。

## 网站 SEO 设置

- 文档：[网站 SEO 设置](site-seo-settings.md)。
- 契约：复用 Root 保护的 `/api/option/` 更新 `SystemName` 与 `SystemDescription`；公开 `/api/status` 只读返回两者。描述缺省为空，最多 200 个 Unicode 码点，可清空。
- HTML：Classic 与 Default 首屏按请求读取当前选项，输出标题、唯一 description、`og:title`、`og:site_name`、`og:description`；空描述保持空值，不回落到静态旧文案。保留静态资源、generator、版权及内嵌脚本。
- 稳定性：不新增匿名写接口或数据库专用结构；通过现有 Option 持久化与多节点同步生效。双模板前端表单分别维护、API 契约共用。
