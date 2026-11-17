# Issue #403：订阅升级分组显示当前名称

## 目标与边界

订阅管理列表、套餐详情、用户购买列表和购买确认应展示当前分组名称；
`upgrade_group`、`downgrade_group` 继续保存和提交原 code，不参与名称鉴权。
本项仅修改订阅控制器的只读响应与 Default / Classic 订阅界面，不修改数据库结构、
升级规则、计费、购买和角色权限。不调用生产 API，不执行真实购买，不提交或部署。

## 接口方案

- `GET /api/subscription/plans` 与 `GET /api/subscription/admin/plans` 保留
  `data[].plan` 的原有字段，追加只读 `upgrade_group_name` / `downgrade_group_name`。
- 每次列表请求批量调用既有 `GetGroupDisplayNameMap`，按现有 code / 历史 alias
  映射当前名称；实体删除、名称缺失、无法解析或查询失败时回退原标识。
- DTO 字段不写入 `model.SubscriptionPlan`，新增、编辑仍接受原有 `plan` 契约。
- Default 编辑器使用既有管理员 `/api/group/details` 的名称与 code，Classic 已采用此接口；
  历史套餐携带的 alias / 删除标识保留选项，保存不隐式改写。
- 旧接口响应没有名称字段时，双模板回退 code，兼容升级过程。

## 测试计划

先编写失败回归，再实现：本地 SQLite 的真实列表控制器验证 name≠code、改名、
alias、删除、空分组、禁用套餐的用户/管理可见性；真实组件验证管理列、详情、
购买列表与确认、编辑保存继续 code。新增/编辑 API 使用本地 mock，无真实交易。
每个测试进程限时 60 秒，TEMP/TMP/GOTMPDIR 放 D 盘，Node 限制内存，Vitest 单 worker。
运行定向测试、TypeScript 检查、修改文件 lint/格式与 `git diff --check`，不做全构建。

## 国际化与验证记录

新增英文翻译键：无（复用现有文案；分组名称来自 API）。
独立审查、真实浏览器及生产验收由协调者另行安排；本项结果待补。

## 14:06 前端交接（写权已释放）

按协调者新指令，Codex 停止全部前端实现与测试修改，不回滚已完成成果。
以下文件交给 Claude 接续；本工人仅继续 `controller/subscription.go` 及专属 Go 测试。

已修改 Default：

- `web/src/features/subscriptions/types.ts`：可选只读名称字段。
- `web/src/features/subscriptions/api.ts`：模块内 `getGroups` 改读既有 `/api/group/details`。
- `web/src/features/subscriptions/components/subscriptions-columns.tsx`：名称标签。
- `web/src/features/subscriptions/components/dialogs/subscription-purchase-dialog.tsx`：购买确认名称。
- `web/src/features/subscriptions/components/subscriptions-mutate-drawer.tsx`：升级/降级名称选项、历史 alias 原值、在途请求失效保护。
- `web/src/features/wallet/components/subscription-plans-card.tsx`：钱包购买列表名称。
- `web/src/features/subscriptions/components/__tests__/group-names.test.tsx`：真实列、购买确认、API 提交、真实编辑保存四项。
- `web/src/features/wallet/components/__tests__/subscription-group-names.test.tsx`：钱包真实卡片一项。

已修改 Classic：

- `web/classic/src/components/table/subscriptions/SubscriptionsColumnDefs.jsx`：列及详情弹层。
- `web/classic/src/components/table/subscriptions/modals/AddEditSubscriptionModal.jsx`：历史 alias 选项与请求失效保护。
- `web/classic/src/components/topup/SubscriptionPlansCard.jsx`：购买卡片。
- `web/classic/src/components/topup/modals/SubscriptionPurchaseModal.jsx`：确认弹窗。
- `web/classic/src/components/table/subscriptions/__tests__/group-names.compat.test.jsx`：真实列、详情、列表、确认、编辑五项。

红绿与剩余事项：

- Default 初始三项均红（`default-red.log/.err`），最新两文件 **5/5 绿**（`default-green2.log/.err`）。
- Classic 首轮遇 Node 26 的 webstorage 问题；添加进程参数 `--no-experimental-webstorage` 后展示四项通过，
  随后在专属测试补齐 jsdom `Range.getBoundingClientRect` 并恢复原描述符。
- （14:06 历史记录，已被文末“Claude 前端接手结果”取代）最新 Classic **4/5 通过**（`classic-green2.log/.err`），真实编辑测试已展示名称，
  但 `getByRole('button', { name: '提交' })` 未命中可访问名称，尚未走到保存断言；需 Claude 修正真实按钮查询。
- 前端尚未格式化、lint、typecheck；新增创建交互、删除回退选择器和手机实际页面验证尚待完善。
- 未改共享 `users/api.ts`、分组解析器、locales；无需新增翻译键。
- API 字段为 `plan.upgrade_group_name` / `plan.downgrade_group_name`；响应原 code 和编辑提交 code 保持不变。
- 文档 README 索引属协调者共享文件，需要协调者登记本记录。

证据目录：`D:/Temp/issue403-subscriptions`。已启动的前端验证均已结束并收取结果，
没有本工人在运行的前端写入或测试进程；本工人没有启动服务、浏览器或部署进程。
Go 编译先前三次 amd64 60 秒超时，切回本机原生 386 后已运行出断言；禁用套餐夹具需显式更新 false，
避免模型 `default:true` 覆盖零值，后端后续结果另记。

## 后端完成与证据边界

仅修改 `controller/subscription.go`，新增 `controller/subscription_group_names_test.go`。
控制器内独立嵌入 DTO 增强两个只读列表，名称只在响应序列化时计算，不修改
`model/subscription.go`、数据库列、购买、升级、计费、鉴权或路由。
名称查询每个列表批量执行一次现有解析器，失败记录系统错误并回退原标识。

最新定向命令：

```powershell
$env:TEMP = 'D:/Temp/issue403-subscriptions'
$env:TMP = $env:TEMP
$env:GOTMPDIR = $env:TEMP
$env:GOARCH = '386'
go test ./controller -run 'TestSubscription(ListsResolveCurrentGroupNamesWithoutChangingCodes|PlanWritesKeepCodesAndRejectDisplayNames)' -count=1 -timeout=55s -p=1
```

实际通过隐藏进程运行，外部 `WaitForExit(60000)` 限时。最新 `backend-final.log`
结果 `ok controller 0.138s`，标准错误为空；Go 1.25.5/windows/386，Bun 1.4.4。
`gofmt` 与限定后端/本记录的 `git diff --check` 通过。

真实行为覆盖：

- SQLite + Gin 列表控制器、JSON 输出：name≠code、当前/历史 alias、改名后再次请求、
  未知 code、名称为空、空分组、实际删除实体后残留 alias 和套餐。
- 管理列表包含禁用套餐，用户列表只包含启用套餐；禁用值由夹具明确更新 false。
- 实际新增/编辑控制器和 SQLite 持久化：仍保存原 code；只读名称输入不成为业务字段，
  以显示名称代替 code 的新增请求被原规则拒绝。无真实购买。
- 旧控制器对照：`git show HEAD:controller/subscription.go` 写入证据目录，
  使用 `before-overlay.json` 仅对 Go 编译输入做只读 overlay，未回滚工作区；
  `backend-before-red.log` 明确失败于缺失名称字段，随后相同回归在当前实现转绿。

未验收范围：真实认证中间件角色 E2E、MySQL/PostgreSQL 运行环境、生产浏览器和全构建。
角色与权限依赖原路由契约，本项没有更改；数据库实现复用跨库解析器，无迁移。
独立审查须由另一工人执行，本记录不代表审查已通过。

共享契约影响：仅两个套餐列表追加可选名称字段，旧客户端可忽略；code、套餐 ID、
认证、计费及购买请求保持原契约，不向其他仓库自动发消息。
`NewAPIForDouDi` 和 `NewAPIModifyByGang` 分别建议先核实是否具备“分组名称与 code 分离”
及订阅升级能力；具备时值得分享此 DTO 展示修复，不具备时不建议直接移植。

本项总体仍待 Claude 完成前端交接中的未验收项及独立审查；后端实现与上述定向验证完成。

## Claude 前端接手结果（task_76ee521a7ab5）

范围仅限上文 14:06 交接清单的前端源码与测试；生产代码未改动，未碰 users/api、共享解析器、locales、controller。
API 只读 `upgrade_group_name` / `downgrade_group_name`，界面展示名称，提交仍为 code，不提交名称字段。

根因（均在测试侧）：

- Semi `IconSave` 的 span 带 `aria-label=save`，按钮可访问名称为 `save 提交`，精确匹配 `提交` 失败；改用 `/提交/` 定位。
- 受控 Semi Select 仅在 popover 关闭动画结束后才触发 `onChange`，jsdom 不运行动画；
  选项点击后对 `[class*="animation-hide"]` 派发 `animationEnd`（沿用 `task-plugin-binding.compat.test.jsx` 既有做法）。

本次改动（均未提交）：

- Classic `group-names.compat.test.jsx`：修复真实编辑用例，新增“真实创建表单选项展示名称并提交code”，补 AGPL 版权头（该文件原本为未跟踪新文件，从无版权头，并非事故丢失），按 prettier 格式化。
- Default `group-names.test.tsx`：新增“真实创建抽屉选项展示名称，保存提交code且不提交名称”，合并重复 import、去除非空断言，按 oxfmt 格式化。

定向命令均直接调用底层工具，未运行 `format-with-protected-headers`；每个测试进程 60 秒上限、顺序执行、TEMP 在 D 盘、`--no-experimental-webstorage`、`--maxWorkers=1`：

| 项目 | 结果 | 退出码 |
| --- | --- | --- |
| Default 订阅 `group-names.test.tsx` | 5/5 通过 | 0 |
| Default 钱包 `subscription-group-names.test.tsx` | 1/1 通过 | 0 |
| Classic `group-names.compat.test.jsx` | 6/6 通过 | 0 |
| oxlint（Default 测试） | 无问题（首轮 5 项错误已修） | 0 |
| oxfmt --check（Default 测试） | 通过 | 0 |
| prettier --check（Classic 测试） | 通过 | 0 |
| eslint（Classic 测试，含 header 规则） | 无问题 | 0 |

日志在 `D:/Temp/issue403-subscription-claude/`：`final-default-subscriptions.log`、`final-default-wallet.log`、`final-classic.log`。

最终 sha256：

- `web/src/features/subscriptions/components/__tests__/group-names.test.tsx`：`80e0791bea360bc8f418c60721fa0ffce6c54c8ccd3be1ee95894d409007826d`
- `web/src/features/wallet/components/__tests__/subscription-group-names.test.tsx`：`b42bdd49e38fe2f99b703f43911913be4d9c12ed55fe32ab988c6424f3c095d9`（本次未改动）
- `web/classic/src/components/table/subscriptions/__tests__/group-names.compat.test.jsx`：`3bd31849cd0e930b84f086d1a6af0ce7b3f5a3ef8f25eef98c7881e3354d8922`

仍未验收（不标绿）：

- 真实手机浏览器页面未运行；仅有静态依据：Default 抽屉升级/降级选择器为 `grid-cols-1 sm:grid-cols-2`，窄屏单列堆叠。Classic 弹窗窄屏表现未实测。
- 未跑全量 build/typecheck；10 个已修改生产文件的 lint/格式未在本轮复查（本轮仅限两个测试文件）。
- 未覆盖“分组被删除后选择器回退”的真实选择器交互用例。
- 未提交、推送、部署；独立审查须由另一工人执行。

### 独立审查与 390px 手机验收登记（task_e0d1651ea493）

- 独立审查（Codex ctx_bdc068b5c219）通过，无 P1/P2；报告 `D:/Temp/issue403-subscription-review/report.md`。复测：Default 6/6、Classic 6/6、已删除分组选择器回退 1/1。通过边界仅限上述单测与只读审查，不含全量构建/typecheck/真机。
- 390x844 Default 手机验收：`D:/Temp/issue403-subscription-claude/mobile-cdp-test.mjs` 13/13（含 12 项展示/布局与 1 项 `GET /api/option/` 实际命中及合规字段断言）。干净 Chrome、role 100、合规状态由 React 自然启用、订阅行唯一编辑按钮；历史 force-enable 轮次不计。仅为 Default 展示/布局 mock 证据，不验证保存请求，不代表 Classic。
- Classic 手机端尚未验证。
- 格式漂移 6 文件（Default oxfmt、Classic prettier）仅在 D 盘副本生成格式化结果，真实源码未改；待协调者授权仅格式变更。

### 格式化落地与 Classic 390px 验收（Claude Code，协调者授权）

- 6 个格式漂移文件已在授权下用各模板自带格式化器就地仅换行/缩进格式化（备份 `D:/Temp/issue403-subscription-claude/backup/*.orig`，头部版权不变，无 BOM，去空白/尾逗号后内容与原文件一致）。格式化后 SHA256 前缀：api.ts 1e97e43e、subscriptions-columns.tsx ad9f365e、subscriptions-mutate-drawer.tsx 63665ebc、subscription-purchase-dialog.tsx 5ad5c364、SubscriptionsColumnDefs.jsx c9ba9cb9、AddEditSubscriptionModal.jsx 93668208。
- 检验：Default oxfmt --check 通过；Classic prettier --check 通过、eslint 退出 0；git diff --check 干净。oxlint 对 subscriptions-provider.tsx:27、constants.ts:19 的 no-import-type-side-effects 为既有问题，非本次改动文件。
- Classic 390x844 手机验收（自有临时 vite 127.0.0.1:5190，mock 接口，无生产请求，验后已停止）：`D:/Temp/issue403-subscription-claude/classic-cdp-test.mjs` 12/12。覆盖列表显示名称/不暴露 code、列表与编辑弹层无横向溢出、编辑按钮自然启用、下拉选项显示名称、保存 PUT 的 `plan.upgrade_group` 为 `premium-code` 且不含 `*_group_name`。弹层溢出需待滑入动画结束后测量（动画中会瞬时误报）。
- 未做：全量构建/typecheck、真机；Default 13/13 未重复；未提交/推送/部署。待观察：Default 钱包页 goober 类 go427126709 delta=12，未判定是否既有。

### 390px 手机验收重做（task_46b6a75b14bb，Claude Code）

- 上文 Default 13/13、Classic 12/12 经独立审查认定存在验收缺口（断言偏弱、未固定 390x844、溢出过滤不严、失败不一定非零退出），仅作历史记录，不再作为验收依据。
- 新夹具（均在 `D:/Temp/issue403-subscription-claude/`，旧版原样归档于 `history-before-task_46b6a75b14bb/`）：`default-390-audit.mjs`（sha256 前缀 7f13d25f）、`classic-390-audit.mjs`（0cb96bef）。统一改动：每次测量断言 innerWidth===390 且 innerHeight===844，不放宽视口；页内溢出分类器（document-hscroll、known-hidden-devtools、legit-inner-scroll、clipped、transparent-pseudo-switch、ancestor-of-pseudo-switch、contained-glyph、unclassified），只有 unclassified 与 document-hscroll 判失败；仅等待有限动画且上限 3 秒；断言数不符 19 或任一断言失败或异常均非零退出，55 秒守护超时退出码 124；mock 的 /api/status 补 success 与 setup；MOCK_PLAN 补 subtitle；不强制启用任何按钮；删除已作废的 1280 加宽与旧过滤代码。
- 结果：Default 19/19，退出码 0（第一次 17/19，抽屉顶部与底部因开关 ::after 溢出向祖先传播被判未分类，以按实测右边缘判定的 ancestor-of-pseudo-switch 规则修复，真实溢出的子元素仍判 unclassified）。Classic 19/19，退出码 0（首次运行因 contained-glyph 规则尚未存在，classic-options390 有 3 个 semi-select-option-icon 判未分类，sw=14、cw=12，子元素右边缘 45、45、43.7，不超过 390px 视口；新增规则要求无文本、溢出不超过 4px、后代右边缘不超视口且不超父级右边缘，其余仍判失败）。Classic 保存断言：PUT、URL 以 /api/subscription/admin/plans/1 结尾、plan.upgrade_group 为 premium-code、顶层与 plan 均不含 *_group_name，负载见 `evidence-task_46b6a75b14bb/classic-save-payload.json`。
- 证据（同目录 `evidence-task_46b6a75b14bb/`）：classic-browser-audit.json（c62bbb90）、classic-save-payload.json（e707b14f）、default-clean-audit.json（5b6a490f）；截图 8 张：classic-list-fixed390、classic-modal-top390、classic-modal-group390、classic-options390、default-list-390、default-clean-drawer-top-390、default-clean-drawer-bottom-390、default-clean-wallet-scrolled-390。
- 进程：Default 使用既有 127.0.0.1:5173（PID 52188，全程未动）；Classic 使用自有隐藏 Vite 127.0.0.1:5198，启动器 PID 48504、监听 PID 49480，验后已停止，5198 无监听。每个测试进程均在 60 秒内，TEMP/TMP 指向 D 盘，NODE_OPTIONS 禁用 experimental-webstorage。
- 仍未验收：真机；全量 build/typecheck；Default 列表页名称列在窄屏为 mobileHidden，仅在抽屉与钱包页验证名称；Default 无保存路径，未验证保存请求；超长名称窄屏场景未覆盖；Default 钱包页 goober delta 是否既有仍未判定；contained-glyph 规则依据是 Semi 图标自带的 2px 内部溢出，属夹具侧分类，未改生产代码。未提交、推送、部署；需另一工人独立复审。

### Default 夹具 go 类豁免 P2 返修（task_65296803b025，Claude Code）

- 独立复审 ctx_ef378e16b3c8 指出：上文 Default 19/19（default-390-audit.mjs，sha256 前缀 7f13d25f）的溢出分类器把任何 goober 类名（go + 数字）都当作已知隐藏 devtools 豁免，存在 E1 缺口，该 19/19 仅作历史记录，不再作为验收依据。属夹具问题，不是已确认的产品缺陷。
- 根因：节点 go2821754102（分隔 DIV，文本 -，overflowX visible）位于可见的固定按钮 Open TanStack Router Devtools 内部；goober 数字是构建相关哈希，必须按祖先关系与真实可见性判断，不能按类名。
- 返修在独占新目录 `D:/Temp/issue403-subscription-claude/repair-task_65296803b025/`（含 evidence/、evidence-negative/），旧版原样备份于 `history-before-task_65296803b025/`（2136 个文件校验一致）。移除 go 类豁免；新增 devtools-aria-hidden（位于 .tsqd-parent-container 且有 aria-hidden 祖先）与窄规则 devtools-router-button-inner-visible-overflow（overflowX 为 visible、完全位于 router-devtools 按钮内、按钮在 innerWidth 内）。已否决按祖先整体豁免，因为与原缺陷同型。
- 结果：返修后运行 RUN_EXIT=0，断言 19、失败 0；窄规则加入前 4 个快照因 DIV.go2821754102 sw=4 cw=1 失败，证明旧豁免曾掩盖它。负向对照（make-negative.mjs）NEG_EXIT=1：在 devtools 之外注入可见 go9999999999 判未分类并失败，aria-hidden devtools 下的隐藏 go8888888888 被识别。
- sha256：default-390-audit.mjs 523b8aed8690dd5fbe3d6b7bdc232b836490a6fb353eb68b49064b417c3ab0f2；default-390-audit-negative.mjs e23f075a5fa6a52784b04fc8231566d61c4f32a3df65ac2b8c171809b6296a3f；make-negative.mjs 前缀 ccb5d31c；evidence/default-clean-audit.json 97b13daebd65af449b522f61f460bb62c3c7b7f27e703b139c74cc8cccd56ab8；evidence/default-list-390.png 59076c1a6eeafda07b6f4d8f57241b3362e18e7e804549984396c2d372d79e2b；evidence-negative/default-clean-audit.json 135f502906035b9eaffa5b7bf22cbce9ffc40db7d13b2a3f7c3d519f894226e9。原件未改：default-390-audit.mjs 7f13d25fee5ab38cf3b3f667974b00c77e1fcb983ffa9248586527a6735a9e75；evidence-task_46b6a75b14bb/default-clean-audit.json 5b6a490fc3452a6c5f95cf8185f3037be05d8f0d1512b23e4aab6960103370df。
- 仍未验收：真机；全量 build/typecheck；Default 列表页名称列窄屏 mobileHidden；Default 无保存路径；超长名称场景；钱包页 go427126709 delta 是否既有。未改生产代码，未提交、推送、部署；需另一工人独立复审。

### Default 夹具 aria-hidden 豁免 P2 二次返修（task_181ef100c408，Claude Code）

- 独立复审 ctx_ef378e16b3c8 再指出：上一轮（task_65296803b025，default-390-audit.mjs sha256 523b8aed8690dd5fbe3d6b7bdc232b836490a6fb353eb68b49064b417c3ab0f2）约 307 行的 devtools-aria-hidden 规则在 visible(e) 为真之后，仅凭 .tsqd-parent-container 与 aria-hidden 祖先就豁免；负向对照注入的 neg-hidden-devtools-go 无任何隐藏 CSS（display:block、visibility:visible、opacity:1、sw110>cw1）仍被判 devtools-aria-hidden。属夹具问题，不是已确认的产品缺陷。上文 7f13d25f 的 19/19 与 523b8aed 的 19/19 均仅作历史记录（存在 E1 缺口），不再作为验收依据。
- 根因：aria-hidden 不是视觉隐藏证据。返修原则：只能依据实际计算样式或布局（display:none、visibility:hidden、opacity 为 0、零尺寸矩形，由 visible(e) 判定）排除；因此删除 devtools-aria-hidden 分支，保留 visible(e) 前置排除与窄规则 devtools-router-button-inner-visible-overflow（go2821754102 仍据此分类）。E1-E5 口径、19 条断言与视口 390x844 不变。
- 独占目录 `D:/Temp/issue403-subscription-claude/repair-task_181ef100c408/`：before/（上一轮版本副本与 before-sha256.txt，未改动）、evidence/、evidence-negative/。EVIDENCE_DIR 已指向本目录，不覆盖旧证据与 reviewer 冻结目录 `D:/Temp/issue403-subscription-mobile-review-repair65296803`。
- 负向对照 make-negative.mjs 改为三个注入节点：#neg-visible-go（go9999999999，可见、在 devtools 之外，必须失败）；#neg-visible-aria-hidden-devtools-go（go8888888888，位于 aria-hidden 的 .tsqd-parent-container 内，无隐藏 CSS，必须失败）；#pos-truly-hidden-devtools-go（go7777777777，父级 display:none，必须被排除）。display:none 节点 scrollWidth 为 0，不会进入 goober 或 overflow 列表，故新增显式探针断言 Hidden devtools positive：存在、parentDisplay=none、w=0、h=0（实测 PASS）。负向变体因此多一条断言，仅在负向变体把 EXPECTED_ASSERTIONS 由 19 改为 20，真实夹具仍为 19。
- 命令与退出码（TEMP/TMP=D:/Temp/issue403-subscription-claude/tmp，NODE_OPTIONS=--no-experimental-webstorage，每进程 60 秒内）：node --check default-390-audit.mjs / make-negative.mjs / default-390-audit-negative.mjs 均 0；node make-negative.mjs 0；真实 node default-390-audit.mjs 断言 19、无 FAIL，RUN_EXIT=0（对既有 127.0.0.1/localhost:5173，PID 52188 未动）；负向 node default-390-audit-negative.mjs NEG_EXIT=1，唯一失败为 default-list-390: no unclassified overflow，仅命中 DIV.go9999999999 与 DIV.go8888888888（sw=110 cw=1），Hidden devtools positive 为 PASS，无 assertion count 不符。
- 证据 JSON 中 go9999999999、go8888888888 的 kind 均为 unclassified；go2821754102 为 devtools-router-button-inner-visible-overflow；devtools 开关容器为 clipped。
- 完整 sha256：default-390-audit.mjs 3b67e185feb53e5b9bc454e87dcf9fa5b94e6d5c0a59d86419a7c2e165f18d6e；default-390-audit-negative.mjs 43dfc93006d2a48a455fdd3f353e74254216e2d9c6a1d09ed7356567240567cf；make-negative.mjs 906d0c3c5956fdcec5a71b76e03a347a8a063b9173be69f4ad0c9d138cf3ee1b；evidence/default-clean-audit.json a361ab1d0ea208718c6a5b54bce7fec3b73b2098d5e05ff18391b55a19601d12；evidence/default-list-390.png 00e9aeec8bcc4d1ce088272c6a71e3c043958557a1083abd461bd685fee1658d；evidence/default-clean-drawer-top-390.png cf7eed7bb083c385a9fd0398e2c3500a8ba80692108e8dcffb38a676cedb0199；evidence/default-clean-drawer-bottom-390.png 4ff1f452929529ff4768d4065c75b77f1fc99ee98ef638f63b3a6e4b8a533472；evidence-negative/default-clean-audit.json 5302fbbb270ea61940be41536f18c970a79c613294cf7d0414ffb016f512d59a；evidence-negative/default-list-390.png 9de22baf20206ae6686f20eb924254454a2bdff86469a1a2716a5c74627d8c68。
- 仍未验收：真机；全量 build/typecheck；Default 列表页名称列窄屏 mobileHidden；Default 无保存路径；超长名称场景；钱包页 go427126709 delta 是否既有；Classic 夹具本轮未重跑。未改生产代码，未提交、推送、部署、未运行全局 formatter；需同一 Codex 独立复验。
