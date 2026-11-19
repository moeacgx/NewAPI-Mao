# 返佣规则内嵌与推广页提示

## 目标与范围

- Default `/affiliate-management` 内直接提供可编辑的“返佣规则”标签，对齐 Classic `/console/affiliate-admin` 的“返佣规则”；不跳转系统设置，`/system-settings/billing/affiliate` 旧深链保持可用。
- 两套模板的推广设置新增“推广页提示”：开关（默认关闭）、标题、纯文本正文，使用同一持久化配置。
- 不改动返佣申请协议（agreement）与结算逻辑；后端权限不变。

## 配置契约

- 选项键：`affiliate_setting.promotion_notice_enabled`（bool，默认 false）、`affiliate_setting.promotion_notice_title`（默认空）、`affiliate_setting.promotion_notice_text`（默认空）。保存走原子批量接口 `PUT /api/option/affiliate-batch`（见下），单键 `PUT /api/option/` 仍可用，无数据库迁移，旧部署缺省即关闭。
- 用户接口 `GET /api/affiliate/summary` 新增 `promotion_notice: {enabled, title, text}`；协议接口未改。
- 服务端校验（`setting.ValidateAffiliatePromotionNoticeOption`，由 `model.validateOptionValue` 在任何写库前调用，单键与批量共用）：开关只接受 `true`/`false`；标题最多 100、正文最多 2000 个 Unicode 字符（按 rune 计，不是字节），必须是合法 UTF-8；空串合法，表示使用各模板既有的本地化默认文案。非法值整体拒绝，不落库、不更新内存配置。
- 两套模板前端按同样的字符数限制：Default 用 `Array.from(value).length`（zod refine + 输入截断），Classic 用 `getValueLength` + `maxLength`，与服务端一致。
- 正文按纯文本渲染（保留换行，不解析 HTML）。

## 弹出语义

- 开启后，进入或刷新推广页时弹出一次；本次访问内关闭后，刷新数据、翻页、切换周期或标签页均不再弹出；离开后重新进入视为新的访问并再次弹出。
- 实现：首次加载 summary 后用 ref 记录已处理，页面卸载后重置。
- 请求顺序：两套模板的 `refresh()` 均带递增序号，只有最新一次请求可写入 summary、结束 loading 或决定是否弹出；较早发出、较晚返回的响应被丢弃，不会采用旧设置，也不会在关闭后重开。首次请求失败（401/429/网络错误）时不会锁死“已处理”，后续一次成功刷新仍会弹出一次。
- 弹窗次序（一次只有一个活动弹窗）：先推广提示，关闭后再出现 Classic 的协议弹窗；协议的出现条件与完成规则（`can_invite && agreement_enabled && agreement_text`、确认后写入）不变。Classic 的提示弹窗已移出“已具备邀请资格”分支，未通过申请的访客（`AffiliateApplicationGate`）同样看到一次提示，关闭后申请门禁依旧存在，后端 `can_invite` 不受影响。Default 的协议是页内门禁（非弹窗），提示对不具备资格的访客同样弹出一次且不解除门禁。
- Classic 动画串行（第三轮修复）：协议弹窗的 `visible` 为 `agreementModalVisible && noticeSettled && !noticeVisible`。`noticeSettled` 只在两种情况下变为 true：首个成功的 summary 判定“不弹提示”；提示弹窗的 `afterClose`（Semi 在退出动画真正结束、`animationend` 之后才触发，随后弹窗才被卸载）。**没有定时器回退**。提示是否弹出由 `refresh()` 与 `setSummary` 在同一批状态更新中决定（不再依赖随后执行的被动 effect），因此首个 summary 未返回或失败（含 429）时门禁保持“未知”，协议不会先于提示打开，重试成功后同样先提示再协议。同一时刻任意一帧都只有一个弹窗处于可见状态。
- Default `refresh()` 增加 `catch`：首次 summary 失败（如 429）时由共享 http client 弹出错误提示，页面不再产生未处理的 Promise rejection（浏览器 `pageerror`）；`noticeHandledRef` 不受影响，后续成功重试仍只弹出一次。

## 批量保存接口

- `PUT /api/option/affiliate-batch`（Root，位于 `RootAuth` 路由组），请求体 `{"options":[{"key":"affiliate_setting.xxx","value":...}]}`。
- 限制：1–64 项；键必须以 `affiliate_setting.` 开头且不得重复；值只接受 bool / 数字 / 字符串。
- 实现：复用 `model.UpdateOptionsBulk`——先逐键规范化并校验，再在同一个数据库事务内按排序后的键写入，提交后才更新内存配置；任一项失败整体回滚。成功后记录一条 `option.update` 审计日志（含键列表）。
- 前端不再并行逐键写：Default `updateAffiliateOptionsBatch`（`web/src/features/system-settings/api.ts`）与 Classic `SettingsAffiliateCommission.jsx` 的保存均改为一次请求；服务端拒绝时保留表单草稿并展示服务端原因，修正后可再次提交。
- SQLite：默认 `common.SQLitePath` 带 `busy_timeout(30000)`、WAL 与 `_txlock=immediate`，批量写入在同一事务内串行化。未带 `busy_timeout` 的自定义 SQLite 连接串在多个管理员同时保存时仍可能返回 `database is locked (SQLITE_BUSY)`，该失败会如实返回给界面且不产生部分写入。

## 权限

- Root（role 100）可见并编辑返佣规则与推广页提示；role 10 保留业务标签，不挂载规则面板，也不请求 `/api/option`。
- Default 规则面板仅在 Root 且切到“返佣规则”标签时挂载；Classic 沿用 `businessOnly` 分支。

## 复用方式

- Default：`web/src/features/system-settings/billing/affiliate-config.ts` 集中默认值、zod 校验与 `buildAffiliateFormValues`；`affiliate-settings-section.tsx` 为唯一表单；`web/src/features/affiliate/rules-panel.tsx` 读取选项后渲染该表单，系统设置页与管理页共用。
- Classic：`SettingsAffiliateCommission.jsx` 规则标签内新增“推广页提示”面板；`PaymentSetting.jsx` 解析三个新键。

## 国际化

- Default 七语 `web/src/i18n/locales/{en,fr,ja,ru,vi,zh-TW,zh}.json` 各新增 9 键；Classic 八个语言包各新增 8 键。仅追加，未重排既有内容。

## 验证

- Go（编译为测试二进制后运行，`go test ./controller` 本身编译超过 60 秒上限）：`model/option_affiliate_notice_test.go`（100/2000 字边界、空串、非法值与非法 UTF-8 被拒、批量中一项非法时数据库与内存均回滚）、`controller/option_affiliate_batch_test.go`（批量接口原子性、空/非 affiliate 键/重复键/非法值/嵌套值均不落库）、`controller/affiliate_summary_notice_test.go`。
- Default：admin-settings、admin-rules（批量请求、混合保存仅一次请求、服务端拒绝保留草稿）、promotion-notice（一次弹出、纯文本、刷新不重开、重新进入重开、开关关闭、默认文案、乱序响应、较早响应不锁定、不具备资格的访客）、admin-panel、admin-routes、admin-navigation；`bun run typecheck`。
- Classic：`promotion-notice.compat.test.jsx`（先提示后协议且同时仅一个弹窗、不具备资格的访客、首次失败后刷新、过期响应被忽略等 8 项）、`affiliate-admin-access.compat.test.jsx`（批量保存、混合修改一次请求、服务端拒绝保留草稿等 5 项）。
- 真实 Chromium + 隔离 SQLite（复制自评审夹具，本地网关 127.0.0.1，Root/Admin/Member 账号）：旧构建复现了无校验（`not-a-bool`、101 字标题、2001 字正文均 HTTP 200 并落库）与“无 busy_timeout 时并行逐键写 160 次失败 132 次 SQLITE_BUSY”；新构建单键非法值被拒、边界值与空值通过、混合保存 ratio+开关+标题+正文一次成功、含一项非法的批量整体回滚、Admin/Member 调用批量接口均 403、带 busy_timeout 的 160 次并发批量请求 0 失败且终态始终来自某一次完整批量；Classic 与 Default 的 Chromium 场景：硬加载/硬刷新先弹提示、关闭后（Classic）协议弹窗、不具备资格访客的提示且门禁保留、首次请求 429 后手动刷新弹出一次、（Classic）较晚返回的旧响应不覆盖也不重开。
- 格式：仅对触及文件运行 oxlint、oxfmt、prettier；`index.tsx` 与 `types.ts` 在 HEAD 已不符合 oxfmt，未改动以免无关重写。

## 已知限制

- 未部署、未提交。真实浏览器验证使用评审夹具的隔离 SQLite 副本，不涉及生产。
- “Classic 桌面硬刷新偶尔不弹提示”：用旧构建在评审夹具上复现，关闭全局限流时 12 次硬刷新（含 50/200/800ms 延迟）与延迟响应均稳定弹出；开启默认全局限流并反复登录时出现 `429`，首个 summary 请求失败即无弹出。结论为首次请求失败所致而非确定性的顺序缺陷；修复后失败不再锁死“已处理”，后续成功刷新会弹出一次。该结论基于复现实验，未对评审当时的确切环境再次取证。
- Default 的 `web/src/lib/http-client.ts` 会合并相同 URL 的在途 GET，乱序响应在 Default 中无法由真实浏览器构造，该场景由 vitest 覆盖（移除序号保护即失败）。
- 使用未配置 `busy_timeout` 的 SQLite 连接串且多个管理员同时保存时，仍可能出现 `SQLITE_BUSY`（如实返回、无部分写入）；默认连接串不受影响。
- Semi Modal 在 jsdom 中关闭后不移除 DOM，Classic 测试以 `semi-modal-content-animate-hide` 判断已关闭；jsdom 不运行 CSS 动画，序列测试通过手动派发 `animationend` 模拟退出结束，真实的逐帧 DOM 与 opacity 采样见真实 Chromium 验证（`requestAnimationFrame` 采样，覆盖首次进入、硬刷新、关闭、离开返回与 429 重试）。
- 协议弹窗的出现完全依赖提示弹窗的 `afterClose`：若某环境从不派发 `animationend`，协议弹窗会一直等待而不会与提示重叠，这是有意取舍（宁可不叠加，不用定时器绕过退出）；该环境未实测，仅依据 Semi 源码（`Modal.js` 的 `updateState`/`afterHide`）推断。
