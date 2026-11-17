# Issue #401 历史能力迁移独立核查

## 目标、范围和方法

核查 Classic 转 Default 时 Issue #401 正文与全部评论列出的历史能力，记录具体入口、数据契约、权限、当前测试证据及仍未完成的验收。审计基线为 `13f5fdbb93f88da11a157efa69094a9a5e472b60`，本地 HEAD 与 origin/custom-main 一致；不将工作区未提交 MCP 实现视为正式能力。

本工人只新增本文档，其他源码和共享文档只读。#398、#399/#400、#402 由其他工人实现，本文不覆盖他们的代码或代签其验收。无提交、推送、Issue/评论、部署或生产配置操作。

方案：读取历史专题与两个模板的路由、组件、API 和后端权限，沿请求构建、缓存身份、数据过滤及统计追踪；先复用有实际行为断言的既有测试，再列出真实后端和浏览器证据缺口。发现可证实缺陷立即向协调者回传源码位置和复现条件。

## 安全、兼容和验收计划

- 普通用户保持本人数据范围；管理员用户名/用户 ID 条件排他，列表、总数、统计与当前连接使用同一用户作用域。
- 分组名称只用于显示，提交保留稳定 ID/code；标签不扩大授权；不读取或输出完整令牌、生产凭据和对话正文。
- 保留多密钥转换、计费、历史插件版本和通知快照契约；不扩展任务日志或绘图日志的身份筛选。
- 每次既有测试进程最多 60 秒；静态发现、mock 组件测试、隔离数据库/HTTP 测试及真实服务浏览器验收分别记录，不相互替代。
- 双模板的管理员/普通用户、桌面/手机、浅色/深色、保存后刷新是最终迁移验收目标；没有执行的组合明确写为未验证。
- README 索引及能力总览由协调者串行集成，本任务授权不包含修改它们。

## 核查结果

结论为“历史能力逐项定位，累计发现三项新增 P2，修复后的代表场景获得组件和本地浏览器证据，完整迁移验收未完成”。不能仅据这些历史测试宣布所有功能对等或证明生产切换验收；后续用户已授权的 zzapi/Default 切换须按部署专题执行并验证。初轮静态/测试记录与后续浏览器补验分开保留；后者使用未提交的共享工作区，不升级为正式基线能力。

已通过 `gh issue view 401 --repo moeacgx/NewAPI-Mao --comments` 及 `--json title,body,comments` 读取正文与当时全部 1 条评论。以下矩阵覆盖正文 15 行以及评论补充的 #402 和普通日志身份查询边界。旧计划中的 B4 已进入 #398；C1 DeepSeek UI 和 Classic 更早历史来源仍未决，不据旧 Issue 关闭状态补造完成结论。

证据分级：**S** 为本轮源码链路核对；**T** 为本轮实际运行的测试；**B** 为后续真实 Chromium 页面配受控 API mock 的浏览器补验；**H** 为已有测试/历史记录，仅供后续复跑；**U** 为本轮未验收。B 不代表真实服务认证、数据库或供应商验收。下列源码路径均相对仓库根目录，Default 功能目录前缀为 `web/src/features/`，Classic 为 `web/classic/src/`。测试编号对应后文执行记录。

## 迁移矩阵

### 1. 渠道单 Key 转多 Key

- Default：`/channels` → `channels/components/drawers/channel-mutate-drawer.tsx` 与 `hooks/use-channel-mutate-form.ts`；Classic：`/console/channel` → `components/table/channels/modals/EditChannelModal.jsx`。
- 契约：`PUT /api/channel/`，显式 `convert_to_multi_key`、`key_mode`、`multi_key_mode`；无新 Key 时保留旧密钥，取消不残留转换字段。来源/目标 Vertex、Codex、io.net 托管等限制沿用原契约。
- 权限：渠道编辑需 ChannelWrite，转换及策略需 ChannelSensitiveWrite；读取原 Key 的 Root 权限不能替代写权限。
- 证据：S；T2 的 Default `channels/lib/__tests__/multi-key-conversion.test.ts`、T3 的 Classic `multi-key-editing.test.mjs` 通过。H：Default hook 的同名组件测试以及[既有独立验收](11_default_multi_key_conversion_qa.md)。
- B3：双模板桌面浅色及手机深色已完成单 Key 显式转换、追加、轮询和 mock 保存后刷新。U：保存到真实数据库、取消/切换渠道、受限凭据类型、真实敏感写权限及 Redis 轮询仍未验收。

### 2. 分组标签 CRUD、图标及选组筛选

- Default 管理入口实际为 `/system-settings/billing/group-pricing`，由 `billing/section-registry.tsx` → `RatioSettingsCard` → `GroupRatioForm` 挂载 `GroupTagSettings`；用户入口 `/keys` 的 `api-key-group-combobox.tsx`。Classic 为 `/console/setting` 分组倍率页及 `/console/token`。
- 契约：管理 `/api/group/tags` 与 `/:id`，名称、说明、排序、多图标、匹配词与稳定分组 ID；用户仅请求 `/api/user/self/groups` 的 `group_tags`，筛选现有授权候选，不扩大分组权限。令牌提交 ID/code，展示当前名称。
- 权限：管理 API AdminAuth；Default 的系统设置父路由仍 Root-only，此为已知页面边界，不把 API Admin 权限等同页面可达；用户选择仅 UserAuth。没有调整父路由权限。
- 证据：S；T9 的 `keys/components/__tests__/group-tag-submission.test.tsx` 实际表单/API mock 测试通过；H：`system-settings/models/__tests__/group-tags.test.tsx`、`group-tag-filtering.test.tsx`、`keys/components/__tests__/group-tag-selection.test.tsx`；Classic `pages/Setting/Ratio/__tests__/group-tags.compat.test.jsx`。历史固定版本验收见[标签独立验收](11_default_group_tags_qa.md)。
- B2：双模板用户令牌编辑器按标签筛选、跨标签保留选择、保存与刷新通过。U：管理 CRUD、图片失败回退、标签删除和真实用户授权交集仍未验收。普通日志分组选择另见 F1/B4。

### 3. 令牌不限组/上限、分组限定渠道及 Auto 顺序

- Default `/keys` → `api-keys-mutate-drawer.tsx`、`lib/api-key-form.ts`；Classic `/console/token` → `components/table/tokens/modals/EditTokenModal.jsx`。
- 契约：`/api/token/` 与 `/:id`、`/api/token/auto-groups`；`group_ids`、`group_channel_ids`、`auto_groups_mode`、`auto_groups`。`MaxTokenAutoGroups=0` 不限制，正整数限制显式组及自定义 Auto；继承全局 Auto 不按此数量裁剪。限定渠道用稳定分组 ID，普通用户提交前删去 `group_channel_ids`，后端仍需独立鉴权。
- 权限：本人令牌 UserAuth；按组限定渠道仅管理员。顺序保存、回显与授权候选分别处理。
- 证据：S；T9 的 `keys/lib/__tests__/auto-group-form.test.ts` 通过；H：`components/__tests__/auto-group-order-editor.test.tsx`、`api-keys-mutate-drawer.test.tsx`；Classic `token-group-limit.compat.test.jsx`；[上限专题](06_unlimited_token_groups.md)。
- B2/B6：双模板普通用户多组保存/刷新、提交稳定 ID/code，Default 改名与历史 alias 回显已补证。U：完整 Auto 顺序、正数上限、管理员按组限定渠道、真实数据库与转发匹配仍未验收；#399/#400 不纳入本工人的旧令牌契约通过声明。

### 4. 提示词/Skills、安装徽标、来源更新、分类与白名单

- Default `/prompt-center` 与 `/keys` 安装徽标；Classic `/console/prompt-center`、令牌表安装弹窗。
- 契约：`/api/prompt-center/`、`categories/`、`admin/`、`admin/import/archive`、`admin/import/repository`、`admin/:id/update-repository`、`/api/token/:id/prompts`；稳定模板 ID、`version`、`load_mode`、`all_groups`、白名单分组 ID、来源提交和正文指纹。仓库更新保留安装，站内编辑覆盖需确认，分类删除归入未分类。
- 权限：普通用户只拿名称/用途/授权分组摘要并管理本人令牌安装；正文、导入和模板编辑走管理员链。分组标签只筛选候选，不是授权依据。
- 证据：S；H：Default `__tests__/repository-update.test.tsx` 具体覆盖来源链接、普通用户隐藏来源、版本冲突及覆盖确认；另有 `token-prompts`、`progressive`、`group-selector`、`category-delete`；Classic 对应 `pages/PromptCenter/__tests__/*`。T4 组合复跑超时，不将其中任何文件标通过。
- U：真实仓库下载、正文不泄露的真实响应、保存后刷新、卸载与运行时撤权、收费 Skills 读取/结算不在本轮验证范围。

### 5. 操练场本人令牌切换与 Skills 回复徽标

- Default `/playground` → `playground/api.ts`、令牌选择与流式控制器；Classic `/console/playground` → `components/playground`。
- 契约：`/pg/chat/completions`、`/pg/token/models`；仅发 `X-Playground-Token-Id`，不取完整 Key；选择令牌时删除账号分组参数；`X-New-API-Skills-Display: metadata` 与响应 `x-new-api-gateway-skills` 保存为元数据，不能拼入下一轮消息正文。
- 权限：登录会话与后端本人令牌所有权检查，模型发现错误不能回落账号模型绕过令牌范围。
- 证据：S；T2 `__tests__/token-request.test.ts` 通过（非流式、流式、模型发现与令牌头）；H：Default `token-switch.test.tsx`、`streaming/__tests__/gateway-skills.test.ts`，Classic `token-switch.compat.test.jsx`、`gateway-skills.compat.test.jsx`。T4 组合切换测试超时。
- U：本轮无真实模型调用、实际密钥所有权拒绝和浏览器停止生成/切换令牌验证；没有收费供应商请求。

### 6. 当前连接、管理员指定用户及结束保留

- Default `/usage-logs/common` → `active-connections-panel.tsx`、`use-active-connections.ts`；Classic `/console/log` → `components/active-requests`。
- 契约：`GET /api/log/self/active`、管理员 `GET /api/log/active`；`user_id` 与 `username` 严格二选一，`page/page_size/search/status`，快照 `summary`、`subject`、`observed_at`、`scope`。结束记录保留五秒且冻结耗时，不计活动数量。
- 权限：无指定主体时查询本人，普通用户不能调用管理员端点；管理员清空查询回本人连接，不是全站连接。实时连接不按历史日志时间、技能或历史消费类型聚合。
- 证据：S；T1 的 `active-connections.test.tsx` 11 项通过，包含中止旧请求、迟到响应、本人端点、错误恢复、结束状态、三条预览/抽屉分页、切用户清空抽屉；T3 Classic 查询 hook 同步主体、错误与竞态断言通过。H：`router/active_requests_router_test.go`、`controller/active_requests_test.go`；T5 后端超时，未获得新运行通过证据。
- B1：双模板管理员身份双向切换后连接主体同步，重置返回本人，普通用户只请求 self；连接返回为 mock 快照，不证明实际请求活动或五秒生命周期。
- U：Redis 集群与单实例回退、真实认证会话/跨用户请求、五秒真实后端生命周期、后台标签页真实浏览器行为未验收。

### 7. 普通使用日志身份/技能筛选、响应模型与分组显示

- Default `/usage-logs/common`；Classic `/console/log`。Default `common-logs-filter-bar.tsx` 已有管理员 Username/User ID 切换；不重复实现。
- 契约：`GET /api/log`、`/stat`；本人 `/api/log/self`、`/self/stat`；`user_id` 或 `username`、`type=0&skill=used`、`group`、模型、时间和请求 ID。列表与统计共用 `buildApiParams`，分页总数来自服务端。`other` 中响应模型及技能元数据由相应 column/details 组件解析；分组列优先 `group_name`。
- 权限和缓存：`buildSearchParams`、`buildApiParams` 均优先 userId 排他提交；普通用户忽略跨用户参数并走 self，后端从上下文 `id` 取用户。表格和统计 `querySubject` 变化不延用旧主体占位数据；同主体翻页可保留旧页等待新页。当前连接跟随已提交路由身份。
- 证据：S；T1 31 项最终通过，T3 Classic 15 个日志 hook 用例通过。H：`model/log_user_filter_test.go` 断言列表与统计；`log_skill_filter_test.go` 断言跨页、总数、本人隔离、取消不返部分结果及溢出错误；本轮 T5 未跑完。
- 缺口：正式基线存在 **F1 分组名称下拉没有迁移**，共享工作区已修并取得 B4 局部浏览器证据。B1 验证身份参数、总数显示、统计请求与当前连接联动；不代表跨页总数及统计数据库端已重新验收。
- 分类边界：`/usage-logs/task`（Classic `/console/task`）仅保留既有 username 字段；绘图 `/usage-logs/drawing`（Classic `/console/midjourney`）无本次双身份补齐范围。未把普通日志的结论外推到这两个分类。

### 8. 工单、四状态统计/提醒、批量管理及通知格式

- Default `/tickets`、`/ticket-management`、`/dashboard`、`/notification-center`；Classic `/console/tickets`、`/console/ticket-management`、`/console`、`/notification-center`。
- 契约：`/api/tickets` 本人链、`/api/ticket-admin` 管理链及 `/stats`；批量 `batch/status`、`batch/delete`，自动回复/归档 `automation/auto-reply|auto-archive`；四状态 `open/in_progress/resolved/closed`，版本冲突检查。通知 `/api/notification/tasks|preview`，`ticket_user_message` 保留事件时正文快照，人工回复编辑原通知回执。
- 权限：个人始终按当前用户，管理接口 AdminAuth + tickets read/write；通知配置 Root。浮动提醒只统计 open，不把它等同全部四状态。
- 证据：S；H：Default `tickets/__tests__/account-switch`、`batch-actions`、`pending-tickets`、`notification-center/__tests__/ticket-content.test.tsx`；Classic `pages/Tickets/__tests__/tickets.compat.test.jsx`、`NotificationCenter/__tests__/ticket-events.compat.test.jsx`。T4 组合账号切换测试超时；本轮未签署工单交互通过。
- U：真实本人/他人工单权限、批量事务、归档调度、Telegram 保存/预览/投递与编辑回执、状态统计浏览器刷新均未验收。

### 9. 福利活动/额度券、多人兑换码与优惠码清理

- Default 管理 `/redemption-codes` 的 redemption/promo-codes/benefit-vouchers 三个页签；用户 `/benefits`、`/wallet`。Classic `/console/redemption`、`/console/benefits`、`/console/topup`。
- 契约：`/api/benefit/admin/activities`（publish/pause/resume/end/terminate/report）、`vouchers`；用户 `/api/benefit/activities/:id/claim`；`/api/redemption/`、`/api/promo_code/` 与 batch/invalid 清理。内部额度和 CNY 实付门槛分开，分组关联 ID；软删除保留支付与账务关联。
- 权限：管理 AdminAuth、领取 UserAuth 本人范围；活动不复制/授权分组。福利流水 API/组件仍存在，但双模板有意移除可见入口，不把未挂载流水当漏项恢复。
- 证据：S；T3 Classic `benefit-contract.test.mjs` 通过，但其中多项是源码结构断言，不能等同真实表单验收；H：Default `activity-form`、`activity-delete`、`user-benefits` 交互测试及[福利专题](../../developer/benefit-vouchers.md)。
- U：显示币种切换与精度、CNY 门槛、跨券抵扣/退款、多人并发兑换、支付 reservation 与软删除、真实 DB/浏览器未验收。

### 10. 发票与返佣

- Default 发票 `/invoices`、`/invoice-management`；Classic `/console/invoice`、`/console/invoice-admin`。Default 用户返佣 `/affiliate`，正式基线管理挂 `/system-settings/billing/affiliate`；共享工作区 F2 修复新增 `/affiliate-management` 业务入口。Classic `/console/affiliate`、`/console/affiliate-admin`。
- 契约：发票 `/api/user/invoice/config|orders|preview|request|payment|self`，服务费走外部方式，`pay_methods` 和费用快照；关闭释放来源订单但保留 `closed_orders_snapshot`，closed 为终态。返佣用户 `/api/affiliate/summary|records|withdraw/preview|withdraw`，管理 `/api/affiliate/admin/*`；配置仍 `/api/option/`。
- 权限：本人发票/返佣 UserAuth；发票管理与返佣业务 AdminAuth；全局返佣参数 Root。**F2：Default 非 Root 管理员点击返佣管理进入 Root 父路由而 403**。
- 证据：S；T2 Default `invoices/__tests__/payment-methods.test.ts`、`status.test.ts`、`affiliate/lib.test.ts` 通过；T3 Classic `invoice-total-amount.test.mjs` 为源码展示契约断言通过。H：`model`/`controller` 的关闭订单释放和支付验证测试，未在本轮运行。
- U：支付实际回调/金额/商户验证、关闭后重新合并开票、返佣审核/提现业务页面与普通管理员身份未做真实浏览器验收；F2 修复尚未签署。

### 11. 任务插件及在线扩展

- Default `/task-plugins`、`/extensions` 与 `/extensions/:moduleId/:pageKey`；Classic `/console/task-plugins`、`/console/extensions` 与子页。
- 契约：任务插件 `/api/plugin/task`、`/:key/versions|activate|status`、`runtime/status`、`marketplace/sources`；`key/version/source_hash/active/enabled`；渠道类型 62、`setting.task_plugin_key`（AtlasCloud 61 保留）。安装不等于激活，历史任务按版本/hash 固定。扩展 `/api/extension-admin/marketplace|upload|:id/enabled`，ZIP ID/版本/hash 复核；运行页面走会话保护的 `/api/extensions/`。
- 权限：插件 Admin 读取，Root 源码/写入/激活；扩展管理及本地子页 Root。代理运行 API 仍验证扩展页面与用户能力，不能仅以导航隐藏判断权限。
- 证据：S，已读[插件技能](../../../.agents/skills/task-plugin-development/SKILL.md)及[官方插件专题](../../developer/official-task-plugins.md)；T6 Default `task-plugins/__tests__/management-contract.test.tsx` 8 项通过，覆盖显式 false、归档版本激活、错误恢复、Admin 不能改市场源/激活/读源码；H：`marketplace-management.test.tsx`、`extensions/__tests__/marketplace-panel.test.tsx`，Classic `pages/TaskPlugins/__tests__/management.compat.test.jsx` 与扩展市场测试。
- U：远程市场安装、真实源码编译到分发/预扣/结算/统计的全链、历史轮询、真实供应商/S3/匿名资源均未验收。query/dynamic、在线试运行、异步 UsageFacts 等宿主已知限制不伪装为 UI 漏项，也不宣布插件生产可用。

### 12. 安全审计与对话归档

- Default `/security-audit`，归档 `/extensions/:moduleId/:pageKey`；Classic `/console/security-audit`、扩展对应子页。
- 契约：`/api/security-audit`、`/request-archive`、`/api/extensions/conversation-archive`；策略来源 `cyber_policy/biological_risk`，归档按分组 code 与用户 ID AND 筛选，保留期/数量裁剪，正文仅纯文本，配置密钥后加密存储。
- 权限：前端与后端 Root，普通管理员不得读正文；清空需明确确认，模块停用与历史内容保留边界独立。
- 证据：S；T2 `security-audit/__tests__/policy-action-sources.test.ts` 通过；T3 Classic `security-audit-request-archive-filters.test.mjs` 通过，其中既有部分为源码结构断言；H：`scripts/conversation-archive-native.test.mjs`、`router/security_audit_router_test.go` 与专题。
- U：真实权限拒绝、正文脱敏/加密、保留期任务、清空、native 扩展实际加载和多节点行为未验收；未读取任何生产对话正文。

### 13. 动态分组定价与模型详情

- Default `/pricing`、`/pricing/:modelId` → `pricing/components/model-details.tsx`，已消费 dynamic-price/fixed-price/billing-adjustment；Classic `/pricing` → `components/table/model-pricing`。
- 契约：共享 pricing 响应中的模型价格、`group_ratio`、`group_names`、按次/按秒单位、变体与动态规则；显示币种只影响展示，不替换计费表达式的美元/每百万 token 合同。零倍率必须保留，fallback 不可用真值判断吃掉 0。
- 权限：模型广场的公开/登录可见范围依状态配置；管理员修改价格仍走 Root 选项链。没有调整计费代码。
- 证据：S；T2 `pricing-contracts.test.ts`、`price-currency.test.ts` 通过，覆盖 0 倍率、名称回退、价格单位、折扣口径及币种；H：Classic `billing/utils.test.mjs` 与 `groupVisuals.test.mjs`。已阅读计费表达式文档的设计和单位部分，本次不审计完整结算实现。
- U：完整模型详情浏览器显示、所有动态阶梯/时区组合、价格保存后刷新与实际收费账单对账未验收。

### 14. 渠道拖拽优先级、TokensPro 并发与通知去重

- Default `/channels`、`/notification-center`；Classic `/console/channel`、`/notification-center`。
- 契约：`/api/channel/priority/order|reorder` 的 revision 与预期优先级快照防止覆盖并发编辑；`settings.tokenspro_overview_sync_enabled`、同步间隔和 last\_\* 诊断；通知 `filter_config.prefix_dedup_seconds`、错误关键词。TokensPro 失败不改变并发，0 allowed 的禁用语义不是“不限制”。
- 权限：渠道敏感写、通知配置 Root；同步仍由宿主任务执行，不从浏览器直接调用 TokensPro。
- 证据：S；H：Default `hooks/__tests__/priority-order.test.tsx` 明确覆盖并发改值/删除后拒绝旧目标；`retained-channel-contracts.test.ts` 覆盖保留未改设置和显式 0；两模板 notification filter-config 测试；T7 结果见执行记录。
- U：真实并发管理员保存冲突、鼠标/触屏/键盘拖拽、上游同步、真实 Bot 去重投递未验收。

### 15. SEO、登录与绑定恢复

- Default `/system-settings/site/system-info`、`/sign-in`、`/profile` 及 OAuth callback；Classic `/console/setting`、`/login`、`/console/personal`、`/oauth/:provider`。
- 契约：`SystemName/SystemDescription` 写 `/api/option/`，`/api/status` 公开只读；HTML 首屏 title/description/Open Graph 与状态刷新一致，清空描述不保留旧值；OAuth bind 与 login 模式区分，绑定不替换当前用户，失败/429 不伪装成功。
- 权限：SEO Root，绑定需当前用户会话；不得因恢复登录扩大用户或管理权限。
- 证据：S；T2 `auth/lib/__tests__/oauth-callback-mode.test.ts` 通过；H：Default `system-info-section.test.tsx`、`site-metadata.test.ts`、`email-bind-dialog.test.tsx`、`api.test.ts`；Classic `site-metadata-save.compat.test.jsx`、`oauth-binding.compat.test.jsx`、`account-binding.compat.test.jsx`；T7 包含 SEO 表单复跑。
- U：真实 OAuth 提供商、邮件/短信、429/刷新浏览器全链、真实服务器首屏 HTML/清空与多节点传播未验收。

## 可复现新增缺陷

### F1：Default 普通日志缺少按名称选组（P2，协调者已接收）

1. 准备 `code=vip-internal`、`name=专业组` 的分组及历史日志，访问 Classic `/console/log`。`UsageLogsFilters.jsx:43-73,128-141` 按角色加载分组下拉，按名称选取并提交稳定标识。
2. 同一账号访问 Default `/usage-logs/common?group=vip-internal`。基线 `common-logs-filter-bar.tsx:345-356` 为自由文本输入，直接展示 `vip-internal`，不存在相应按权限加载的分组下拉。
3. 影响：迁移丢失可发现的名称选组入口，历史链接回显 code。不得描述成“所有手输名称查询失败”：`model/group_identity.go:486` 的解析顺序为当前 code → alias → 当前名称，`model/log.go` 再将解析结果扩展为当前 code 与历史 alias；未知/已删标识回退原值。当前 code 与另一组名称冲突时仍优先 code，故选项应提交稳定 code，不能改发 label。

修复验收应覆盖名称与 code 不同、改名后仍查历史、alias、未知/已删除值保留可清除、Admin 全分组与普通用户 self/groups 裁剪、切用户/Only Mine 不残留旧选项。仅限普通日志。已发送 Orca 消息 `msg_381bfb857d76`，协调者确认另派实现；初轮不签署修复结果，后续仅对 B4 所列 mock 浏览器范围给出通过证据。

### F2：非 Root 管理员返佣管理链接必达 403（P2，协调者已接收）

1. 使用 role=10 的管理员账号，打开 Default 管理侧栏“Affiliate Commission”。`web/src/hooks/use-sidebar-data.ts:242-245` 指向 `/system-settings/billing/affiliate`。
2. `web/src/routes/_authenticated/system-settings/route.tsx` 的 beforeLoad 仅允许 `ROLE.SUPER_ADMIN=100`，该账号被重定向 `/403`，无法访问返佣审核/提现/风险面板。
3. Classic `App.jsx:440` 附近的 `/console/affiliate-admin` 使用 `AdminRoute`（role≥10），挂载 `PaymentSetting onlyAffiliate`；`SettingsAffiliateCommission.jsx` 的业务面板独立请求 `/api/affiliate/admin/*`。后端 `router/api-router.go:301-327` 为 AdminAuth。Classic 同页 Option 读取仍 Root-only，不能借迁移修复开放 `/api/option/` 或全部系统设置。

静态链路可确认 Default 阻挡；初轮没有 role=10 浏览器证据，也未断言 Classic 全部配置流程可用。已发送 `msg_7678cfdd896c`；协调者纳入修复，拆出 Admin 业务页并保留 Root 参数配置边界。后续 B5 补了模拟角色的浏览器路由证据；真实三角色账号与后端权限仍未验收。

### F3：Default 令牌列表忽略已有分组名称（P2，已修并局部复验）

1. 原 fixture 返回 `group=stable_a`、`group_ids=[11]`、`group_details=[{id:11,code:"stable_a",name:"Standard Display"}]`，`/api/user/self/groups` 同样有显示名称。
2. Default `/keys` 列表仍显示 `stable_a`；同页编辑器及 Classic `/console/token` 显示 `Standard Display`。修复前 `api-keys-columns.tsx:213-221` 只传原 group，`api-key-group-cell.tsx:50-58` 把该字符串用于徽标及 tooltip，忽略已有详情。
3. 触发条件为名称与 code 不同；多组也受影响。已发 `msg_7c1c93e34211`，协调者交原令牌实现工人处理，避免代码归属冲突。

修复后 B6 使用原 fixture，在 Default 桌面核对单组/多组列表、原生 `title`、改名后历史 alias 按 ID 回显和保存刷新；手机仅验证编辑器，既有手机卡片没有分组字段，不扩展额外功能。自定义 tooltip 浮层未观察到可见状态，不签署该浮层视觉通过。不是数据库改名或真实权限验收；见后文具体证据与限制。

## 已分派缺口与正式基线边界

| 项目                | 基线状态                                  | 本报告处理                                                    |
| ------------------- | ----------------------------------------- | ------------------------------------------------------------- |
| #398 Logo URL       | Default loader 把 URL 当图标名路径解析    | 他人实现，需 URL 保存/刷新/失败兜底与各消费点验收             |
| #399 令牌状态       | 状态缺请求与 queryKey 接线                | 他人实现，需跨页总数与服务端筛选验收                          |
| #400 令牌实时使用   | Default 缺活动摘要、ID 徽标与 active_only | 他人实现，需按本人 ID 聚合与服务端分页验收                    |
| #402 API 地址       | Default 缺 Classic 地址列表/复制入口      | 他人实现，需 api_info/备用地址/当前 origin 回退验证           |
| F1 普通日志分组选择 | 本工人确认的新增 P2                       | B4 通过；两项 P2 修复及原 reviewer 复审通过，真实后端仍未验收 |
| F2 管理员返佣入口   | 本工人确认的新增 P2                       | B5 路由投影和独立复审通过，真实后端鉴权仍未验收               |
| F3 令牌列表分组名称 | 本工人浏览器确认的新增 P2                 | B6 名称/保存刷新及独立复审通过；自定义 tooltip 浮层未观察到   |
| MCP                 | 当前工作区未提交                          | 排除正式能力结论，不覆盖并行修改                              |

## 本轮实际验证与限制

环境为 Windows，Bun 1.4.4、Node 26.13.0、Go 1.25.5 windows/386；与 CI 的版本/架构存在差异。除首轮外，前端子进程传入 `NODE_OPTIONS=--no-experimental-webstorage`。进程采用 Python `subprocess` 的 60 秒硬上限；后续直接启动 Vitest 的超时处理终止本次进程树，未修改系统环境变量或依赖。

| 编号 | 实际范围                                                                                                                                                                                                                       | 结果与证据边界                                                                                                                                                                                                                                |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T1   | Default 日志 `data-identity.test.tsx`、`active-connections.test.tsx`、`skill-type-filter.test.tsx`、`api.test.ts`                                                                                                              | 首轮 42.74 秒，3 文件/25 项通过，skill-type-filter 6 项因 Node 26 原生 localStorage 为 undefined 失败；使用上述 NODE_OPTIONS 单独重跑该文件，28.02 秒、6/6 通过；合计 31 项有通过证据。API mock + 真实 React Query/组件，不是浏览器到后端 E2E |
| T2   | Default 9 个契约文件：multi-key-conversion、pricing-contracts、price-currency、invoice payment-methods/status、affiliate lib、oauth-callback-mode、playground token-request、policy-action-sources                             | 18.47 秒，9 文件/50 项通过；纯逻辑/API mock 边界，不证明服务端结算、完整页面或角色拒绝                                                                                                                                                        |
| T3   | Classic `node --test`：usage-logs user-filter、channels multi-key-editing、benefits benefit-contract、security-audit-request-archive-filters、invoice-total-amount                                                             | 1.20 秒，42 项通过；日志 hook 有请求参数/竞态断言，多密钥为纯函数，福利/发票/归档中存在源码文本断言，明确不当作完整 UI 验收                                                                                                                   |
| T4   | Default plugin management-contract、prompt repository-update、playground token-switch、tickets account-switch 合并运行                                                                                                         | 60 秒硬超时，无完整报告；超时输出处理又触发 str.decode 异常，不记任何文件通过；后续单独复跑插件见 T6                                                                                                                                          |
| T5   | `go test ./model ./controller ./router -run 'TestLogSkill\|TestLogUserID\|TestParseLogUserID\|TestActiveRequests\|Test.*Active.*Route' -count=1 -timeout=60s`；后改 GOARCH=amd64，并精确选中 AdminActiveRequestsRoute 测试重试 | 两次均 60 秒总进程超时，无包级通过报告；首次默认 386、重试 amd64。没有把历史 Go 结果或测试文件存在当作本轮通过；源码只读核查仍有效                                                                                                            |
| T6   | Default task-plugins management-contract 单文件复跑                                                                                                                                                                            | 46.81 秒，1 文件/8 项通过；仅 API mock + 组件管理契约                                                                                                                                                                                         |
| T7   | Default notification filter-config、channels retained-channel-contracts/priority-order、playground gateway-skills、SEO system-info-section                                                                                     | 42.28 秒，5 文件/28 项通过；命令还包含不存在的 keys/lib/api-key-form.test.ts 路径，Vitest 未选中该文件，不计为覆盖                                                                                                                            |
| T8   | GOARCH=amd64，单独 model 的日志身份、技能和历史分组测试                                                                                                                                                                        | 编译失败：并行 F1 实现中新出现的未跟踪 `model/log_group_options_test.go` 引用尚未实现的 `GetLogGroupNamesForIDs`；不是本工人发现的正式基线缺陷，不能据此改动他人文件；本轮后端运行仍未验收                                                    |
| T9   | Default keys `auto-group-form.test.ts`、`group-tag-submission.test.tsx`                                                                                                                                                        | 23.33 秒，2 文件/22 项通过；使用执行时共享工作树，不代签 #399/#400 的全部改动                                                                                                                                                                 |

最终有通过证据的 Default 用例合计 139 项、Classic 42 项；统计不包含超时组合、后端编译失败或未选中的文件。首轮失败未隐藏，最终通过仅指对应文件的补跑结果。复现命令如下，均应在外部加 60 秒进程上限，并给子进程设置上述 NODE_OPTIONS。

```powershell
# 在 web/ 中：T1；首次用了 bun run test，以下直接调用同一 Vitest。
node node_modules/vitest/vitest.mjs run src/features/usage-logs/components/__tests__/data-identity.test.tsx src/features/usage-logs/__tests__/active-connections.test.tsx src/features/usage-logs/components/__tests__/skill-type-filter.test.tsx src/features/usage-logs/__tests__/api.test.ts --maxWorkers=2
# T2
node node_modules/vitest/vitest.mjs run src/features/channels/lib/__tests__/multi-key-conversion.test.ts src/features/pricing/lib/__tests__/pricing-contracts.test.ts src/features/pricing/lib/__tests__/price-currency.test.ts src/features/invoices/__tests__/payment-methods.test.ts src/features/invoices/status.test.ts src/features/affiliate/lib.test.ts src/features/auth/lib/__tests__/oauth-callback-mode.test.ts src/features/playground/__tests__/token-request.test.ts src/features/security-audit/__tests__/policy-action-sources.test.ts --maxWorkers=2
# T6
node node_modules/vitest/vitest.mjs run src/features/task-plugins/__tests__/management-contract.test.tsx --maxWorkers=1
# T7：这里去掉实测命令中未选中的无效路径。
node node_modules/vitest/vitest.mjs run src/features/notification-center/__tests__/filter-config.test.ts src/features/channels/lib/__tests__/retained-channel-contracts.test.ts src/features/channels/hooks/__tests__/priority-order.test.tsx src/features/playground/lib/streaming/__tests__/gateway-skills.test.ts src/features/system-settings/general/__tests__/system-info-section.test.tsx --maxWorkers=2
# T9
node node_modules/vitest/vitest.mjs run src/features/keys/lib/__tests__/auto-group-form.test.ts src/features/keys/components/__tests__/group-tag-submission.test.tsx --maxWorkers=1
# 在 web/classic/ 中：T3
node --test src/components/table/usage-logs/__tests__/user-filter.test.mjs src/components/table/channels/modals/__tests__/multi-key-editing.test.mjs src/hooks/benefits/__tests__/benefit-contract.test.mjs src/security-audit-request-archive-filters.test.mjs src/invoice-total-amount.test.mjs
```

初轮 T1–T9 未运行真实浏览器，后续已补以下 B1–B6；两轮都没有启动独立后端测试实例，也没有真实账号登录证据。未运行全库测试、全量类型/lint/生产构建、MySQL/PostgreSQL、Redis 多实例、供应商/支付/Bot/OAuth 实网。当前共享工作树还在并行变化，结果仅对执行时源码/产物有效；最终集成版本必须固定差异并由独立审查者验收。

## 后续本地浏览器补验（2026-11-16）

### 环境、可复现方法及证据边界

使用 Python 3.13.9、Playwright 1.55.0 和真实 Chromium headless。桌面视口为 1440×1000，手机为 390×844；手机是响应式视口，不是真机触屏。角色通过受控登录刷新响应与本地状态投影，用户 ID 为 7，管理员查询目标 ID 为 22；不能把这些角色 fixture 称为真实认证账号。

Default 复用同工作区 Rsbuild 开发服务，先为 `127.0.0.1:5173`，该进程退出后核对 5174 进程命令确为同仓库 Rsbuild，再复用 `127.0.0.1:5174`。Classic 使用已有 `web/classic/dist`，通过脚本内只读 SPA 静态服务器在 `127.0.0.1:5196` 提供；没有全量构建，旧 dist 不包含本轮所有并行修复。HEAD 仍为 `13f5fdbb9`；当前 Default 工作区与 Classic 既有产物不可冒充同一固定发布版本。

所有业务 `/api/`、`/pg/` 请求在浏览器路由层拦截并返回 fixture；非本地主机请求直接中止。没有请求生产 API、读取生产凭据、连接数据库或执行供应商调用。mock 的“保存”仅更新脚本内对象，刷新再读该对象；它证明 UI 请求与回显链路，不证明落库、事务、计费或后端访问控制。

脚本、请求记录和新截图均保存在本工人目录 `D:/Temp/newapi-issue401-browser-audit`，包括 `audit.py`（共享 fixture）、`logs.py`、`keys.py`、`channels.py`、`groups.py`、`permissions.py`、`f3.py` 及对应 JSON/PNG。`manifest.json` 记录源码与 Classic HTML 的 SHA-256 和记录时间；这是执行后共享工作区快照，不能反推所有早期测试时点完全相同。机器临时目录不随本报告进入仓库，后续归档应由协调者保留该目录。

复现单个脚本时，以每个子进程 60 秒为上限；TEMP/TMP 只传给测试子进程，不修改系统环境。Default 服务端口变化时只改自有脚本 URL；不修改项目配置。示例：

```powershell
@'
import os, subprocess
evidence = "D:/Temp/newapi-issue401-browser-audit"
subprocess.run(
    ["python", evidence + "/logs.py", "default", "--mobile", "--dark"],
    env=dict(os.environ, TEMP=evidence, TMP=evidence),
    timeout=60,
    check=True,
)
'@ | python -
```

脚本为了在断言失败时也保存截图/JSON，会捕获异常，因此以 JSON 的 `status`、`checks` 和 `error` 为准，不能仅凭子进程退出 0 判通过。B1 的普通用户参数为 `--user`；B2/B3 脚本传 `default` 或 `classic` 与可选 `--mobile --dark`；B4 用 `groups.py` 及 `--user --mobile`；B5 为 `permissions.py`；B6 为 `f3.py` 及 `--mobile`。

### 已执行结果

| 编号         | 实际运行及结果                                                                                                                                                                         | 证明范围与剩余边界                                                                                                                                                                                                                                                       |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| B1           | 双模板各 4 组：桌面管理员浅色、手机管理员深色、桌面普通用户深色、手机普通用户浅色；最终 8 组通过，JSON 前缀为 `default-` / `classic-`                                                  | 管理员 Username=`alice` → User ID=`22` → Username=`alice`，列表、stat、active 三请求参数互斥；重置列表/stat 无身份条件，active 返回本人；Default Only Mine 走 self。两模板普通用户无身份输入，所有日志请求走 self；Default 加跨用户 URL 参数仍忽略。没有真实服务拒绝证据 |
| B1 总数/统计 | fixture 按主体返回 total=42/22/99/5，RPM=111/222，TPM=1111/2222；Default 桌面/手机与 Classic 桌面总数显示均断言，Default RPM 显示与两模板 stat 参数联动已核对                          | Classic 手机只断言请求、主体与行切换，未单独断言分页器页数换算；mock 列表仅一条示例行配不同 total，验证分页元数据消费，不能证明服务端 COUNT/聚合正确或真实跨页结果                                                                                                       |
| B2           | 双模板各桌面浅色、手机深色普通用户，共 4 组通过，`keys-*.json/png`                                                                                                                     | 标签筛选选项、跨标签保留两个选择，保存 `group_ids=[11,12]` / `group=stable_a,stable_b`，无 `group_tags`/`group_channel_ids`，不请求 `/api/group/tags`；刷新编辑器回显两个名称。只覆盖 MaxTokenAutoGroups=0 与显式组选项                                                  |
| B3           | 双模板各桌面浅色、手机深色管理场景，共 4 组通过，`channels-*.json/png`                                                                                                                 | 单 Key 显式转多 Key，提交 `convert_to_multi_key=true`、`key_mode=append`、`multi_key_mode=polling`、稳定 `group_ids=[11]`；刷新后显示轮询且 Default 不再显示单转多开关。未读取原 Key，输入仅为无效测试字符串；不证明旧 Key 后端保留或 Redis 轮询                         |
| B4           | F1 修复后 Default 桌面管理员浅色、手机普通用户深色 2 组通过，`groups-*.json/png`                                                                                                       | `historical_alias` 回显当前名称且查询保留 alias；选择显示名称提交 `stable_a`；未知/已删除标识 fixture 保留并能清除；普通用户选项不含无权组、不请求管理员组接口；管理员 Only Mine 切到 self 候选。真实 alias/已删组解析由 mock 提供，服务端边界未证明                     |
| B5           | Default 4 个角色/路由投影通过，`permissions-default.json` 及 `permissions-*.png`                                                                                                       | role=1 访问 `/channels`、`/affiliate-management` 均 403 且无业务请求；role=10 进入返佣业务页请求 `/api/affiliate/admin/*`，未请求 `/api/option/`；同角色进入 `/system-settings/billing/affiliate` 仍403。只验路由与前端请求；不证明后端权限或返佣审批/提现事务           |
| B6           | F3 修复后 Default 桌面浅色与手机深色 2 组通过，`F3-desktop-light.json` / `F3-mobile-dark.json` 和新 `F3-*-editor.png`；列表新图为 `F3-desktop-light-tooltip.png` / `F3-list-fixed.png` | 桌面单/多组名称、改名后 alias 按 ID 回显、原生 title 文本、保存刷新与稳定 ID/code 提交通过；手机仅编辑器通过，既有手机卡片无分组字段，不适用列表名称断言。截图文件名含 tooltip 不代表自定义浮层已出现；该浮层未观察到，不签署视觉通过                                    |

B1 已分别检查最终主题 class / Classic `body[theme-mode]`，8 组均未检测到文档水平溢出、未捕获运行时 `pageerror`。这不等于完整可访问性、所有弹窗布局或全部浏览器兼容验收。抽屉截图在动画结束后重新取得；早期动画中途截图不作布局失败结论。

### 首次失败、差异与未关闭事项

- C 盘临时空间首次耗尽，截图报 `OSError: [Errno 28] No space left on device`，不是功能失败。改用上述 D 盘自有目录并仅设置子进程 TEMP/TMP 后运行完成；未清理任何共享/他人目录。
- Node 26 测试环境设置仍为进程级 `NODE_OPTIONS=--no-experimental-webstorage`，已向协调者提供 T3 原生 Node 的确切命令，未混同为 Classic Vitest/浏览器覆盖。
- 首批浏览器脚本遇到同名 Search、Classic 非 button 控件、异步页面尚未出现、Semi checkbox 外壳拦截等定位失败，按实际 DOM 改为可见控件/标签及状态等待后重跑。fixture 最初漏 `model_limits`、`model_mapping` 等字段导致 Classic 错误；补齐本地响应后重跑，未将这些夹具错误报为产品缺陷。
- Default 开发模式 Router Devtools 遮住保存按钮中心点；使用键盘聚焦并 Enter 激活完成保存，未强制绕过业务校验。部分页面仍有 `canceled` toast，可能关联开发挂载/请求中止；本轮未定位所有来源，因此页面功能契约通过不能表述为视觉无错误。F1 选项 hook 的取消去重处理已随他人工程修复，不外推到全部页面。
- Classic 转换测试首次选择 Polling 后立即提交，抓到 `multi_key_mode=random`；等待控件显示 Polling 并额外 500ms 后桌面/手机均提交 polling。该结果仅签署稳定状态后的保存；该初轮证据不能排除即时提交竞争；后续已用当前真实组件红测及修前 3/3 错误请求确认产品缺陷，追加修复和验证见下段。
- Classic 搜索类型切换只改 `userSearchType`，复用 `field=username` 输入，ID → Username 后草稿仍为 `22`；Default 会清空草稿。最终 Classic 改填用户名后，三类请求均排他且 Reset 清除生效。报告不宣称两模板草稿行为完全一致，也不把共享草稿直接解释成跨用户数据泄露。
- F3 自定义 `role=tooltip` 浮层 hover 断言未成功；只确认徽标原生 `title` 为完整当前名称。手机不提供分组列表字段，不为达到覆盖数量新增产品功能。
- F1/F2/F3 已有上述局部修复证据。协调者另有令牌组件 39+3 项及独立复审回执，此处为转述而非本工人重跑。
  日志两项 P2 已由 `ctx_608ae81b5cfc` 复审通过：独立 6 文件 40 项 46.21 秒通过，原失败
  edge fixture 现 2 项通过（含 `__proto__`），覆盖真实 GroupRatioForm/QueryClient 保存、活动
  观察者和普通用户 self 范围；网络仍为 mock。本报告不把共享工作区 MCP 写作已发布能力。

### B3 追加修复：Classic 立即保存旧模式

两选择器已改用 `onSelect` 同步状态及表单，不等待退出动画；API 字段、敏感写权限及空密钥保留不变。实现工人的真实组件 10/10、契约 8/8、Default 对照 27/27 通过，修后桌面鼠标、移动视口键盘、覆盖和追加四组各三次，共 12 次受控 API mock 浏览器提交通过，未添加固定等待。这 12 次为 B3 缺陷专项增量，不重算原 B1–B6 的 24 个代表场景。

受影响 Classic 已独立重建 exit 0（33.51 秒），原结果和产物保留；Default 源码未再变化，未重复其构建和类型检查。独立审查 `ctx_108ffd07f56c` 已通过，未发现 P1/P2；真实后端、数据库、Redis 轮询和生产权限仍未验收。命令、产物和日志见[追加工作记录](16_issue401_classic_key_mode.md)。

### 15 类能力仍未验收的具体范围

| 类别               | 此次浏览器增量                          | 未验收范围与原因                                                                                                        |
| ------------------ | --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| 1 渠道单转多       | B3 双模板保存刷新                       | 真后端旧 Key 保留/替换、取消转换、受限渠道类型、非授权敏感写、Redis 轮询；仅创建了受控保存 fixture，未建隔离 API/数据库 |
| 2 分组标签         | B2 用户标签选组                         | 管理标签 CRUD/排序/图标失败、删除改名及真实授权交集；本次选择用户侧代表路径，不代替管理事务                             |
| 3 令牌规则         | B2/B6 多组、名称、保存刷新              | 正数上限、自定义 Auto 顺序、管理员限定渠道与实际模型请求；未执行相应 UI 场景/真实网关                                   |
| 4 提示词/Skills    | 无新增浏览器结论                        | 导入仓库、分类删除、收费读取、安装与撤权全链；需要受控仓库/正文和计费数据，既有超时组合未通过                           |
| 5 操练场           | 无新增浏览器结论                        | 令牌切换中止、流式 Skills 徽标、本人所有权真实拒绝；未创建流式协议/隔离后端，本轮禁止供应商请求                         |
| 6 当前连接         | B1 身份作用域                           | 真实连接生命周期、结束保留、迟到响应浏览器竞态、Redis/多实例；本轮 active 响应为快照                                    |
| 7 普通日志         | B1/B4 身份、统计参数、total显示、组选项 | 服务端列表/COUNT/统计/权限联合、数据库历史 alias、Skills 跨页、换账号迟到请求；后端未完成运行且没有真实用户数据         |
| 8 工单通知         | 无新增浏览器结论                        | 本人隔离、四状态统计、批量事务、定时归档、Bot送达/编辑；需要工单与通知回执夹具和独立后端，未执行                        |
| 9 福利兑换         | 无新增浏览器结论                        | 领取/多人并发、支付占用、退款、CNY门槛及多币种；未建立支付/账户状态，不以纯 mock 推断账务                               |
| 10 发票返佣        | B5 F2 路由角色投影                      | 发票金额、返佣审批/提现/冲正、Root配置、Classic新拆分产物与后端权限；未创建真实交易链                                   |
| 11 任务插件/扩展   | 无新增浏览器结论                        | 插件安装同步/轮询/预扣结算/实际用量、扩展动态资源权限；按技能只读，不安装或执行插件                                     |
| 12 安全审计归档    | 无新增浏览器结论                        | 归档解密/导出/保留清理、真实角色数据裁剪；没有敏感归档数据和独立密钥环境                                                |
| 13 动态价格        | 无新增浏览器结论                        | 公式保存刷新、预扣结算差额、跨币种精度、供应商实际用量；本次未执行价格编辑和计费服务                                    |
| 14 渠道优先级/并发 | B3仅保存转换配置                        | 优先级拖拽并发冲突、容量抢占/释放、失败切换和缓存；没有并发上游或独立调度后端                                           |
| 15 SEO/认证        | B5只复用认证fixture                     | SEO编辑刷新、真实OAuth/Passkey/重置/会话过期；mock auth refresh用于页面加载，不能反证真实认证安全                       |

未覆盖完整 2模板×2角色×2尺寸×2主题笛卡尔积；采用 8 组日志代表组合覆盖各维度，再追加关键保存/权限路径。浏览器脚本没有替代 T1 的迟到响应组件证据，也没有扩大任务/绘图日志分类范围。

## 2026-11-16 新一轮独立审查与交付授权

本轮新审查确认的两项 P2 和一项 P3 均已由其他工人返修，再由独立 reviewer 复验闭环：

- 令牌活动首次列表在途时丢失刷新（P2）：按完整主体/查询 scope 保留 pending 并在请求结束后合并补查；reviewer `ctx_b066fbb302a4` 的原失败活动夹具未改动转绿，活动四文件 37/37 通过（28.81 秒）。
- API 地址 HTTP 500 全页跳转（P2）：局部归一化非认证 Axios 故障，保留 401/403、取消原错误和请求 signal；同 reviewer 的原失败地址夹具未改动转绿，地址 16/16 通过（4.09 秒），两份原失败夹具组合 2/2 通过（4.36 秒）。报告：`D:/Temp/issue401-fresh-review-keys/review-report.md`。
- 日志合法 `__proto__` alias 改名重取后丢名称（P3）：仅对该组选项查询关闭结构共享；`ctx_434aefa8fec0` 正式复验原未改 edge 4/4（6.17 秒）、四文件 34/34（24.31 秒）均 exit 0，无新授权、请求循环或 Hide 隐私回归。报告：`D:/Temp/issue401-fresh-review-groups/recheck-report.md`。
- 新 UI reviewer `ctx_855cca0b2af7` 对返佣、Logo、Classic 即时提交独立验证 44/44，通过且无可证实新增 P1/P2/P3：Default 23、Classic 即时提交 10、Classic 返佣 3、原生契约 8。报告：`D:/Temp/issue401-fresh-review-ui/report.md`；未重新执行浏览器，不能把历史截图计为本次执行。

上述真实 hooks/组件/QueryClient 测试使用模拟网络；401/403 为原错误保留及拦截器静态核查，非真实登录续期验收。原生 Node 合同断言也不等于浏览器验收。未因此签署真实认证、数据库、Redis 多实例、支付、供应商或完整 15 类 E2E 通过，未扩大 MCP 审查范围。

用户已授权 review 后提交推送 dev、部署 zzapi 并切换 Default；截至本段登记，提交、推送、此次发布构建、部署和模板切换均尚未执行。dev 提交仅包含本轮经审查的修复，部署镜像另需保留线上 money 基线的 MCP overlay，二者不是同一份纯 dev 产物；冻结、摘要、构建与发布证据由[部署专题](16_issue401_dev_zzapi_deployment.md)维护。本轮是本仓库 Classic/Default 能力对照，不同步上游。

## 后续验收与交接

本文档已使用仓库 `oxfmt` 格式化并通过 `--check`；15 个能力条目、6 个相对链接均核查完整，缺失目标为 0，无待填结果或冲突标记。限定本文的 `git diff --check` 无输出；本文尚未跟踪，另用 `git diff --no-index --check NUL <本文>` 检查新增正文，无空白诊断（退出 1 表示与空文件存在差异）。总交付记录已登记，MCP 仍排除在本矩阵正式结论之外。

1. 原六子项 #398–#400、#402、F1/F2/F3 的实现和独立审查已收口；B3 追加的 Classic 立即保存修复已重建，独立审查已通过；历史门禁和新审查结果见[总交付记录](16_issue401_delivery.md)，历史版本 pass 保持原样。
2. 优先给普通日志补真实隔离后端夹具：两个用户、跨页日志、消费和 Skills generation/discovery、同名不同 code/历史 alias 分组；验证列表 total、统计、当前连接主体、清除/Only Mine/换用户迟到响应及普通用户越权参数。
3. 使用同一固定集成版本分别验收 Default/Classic 管理员、普通用户、Root 的关键入口；F2 必须覆盖 role=10 业务页可达而 Option/全系统设置仍不可达。
4. B1–B6 已补 24 个 Chromium/mock 代表场景；真实双用户/跨页/alias/权限和 MySQL/PostgreSQL/
   Redis 多实例、完整 15 类 E2E 仍需隔离环境补验，mock 不替代真实认证/数据库。历史 Default
   build、Classic build、Default typecheck 已 exit 0，Classic 无独立 typecheck 结果；本次三项返修后的发布构建与 dev/zzapi/Default 执行结果待部署专题登记。

本报告无 API、计费、数据库、密钥或跨项目部署契约变更。`NewAPIForDouDi`：若同样保留 Classic 名称选组且 Default 缺失，F1 的展示/提交分离值得分享，但需先有用户分组裁剪接口；`NewAPIModifyByGang`：若返佣业务同样误挂 Root 设置，F2 的权限入口分离值得参考，但不得照搬并放宽配置权限。仅记录建议，未向任何兄弟项目发送 Issue、PR 或消息。
