# Classic 工单中心实现记录

## 目标与范围

基于 origin/custom-main 的 dea7b5d18，在独立工作树实现 Classic 个人工单中心、管理员工单管理与控制台浮动创建入口。Default 和后端由其他工作项处理；本工作项不修改其实现。

## 接口与安全边界

遵循主工作树 docs/developer/tickets.md：个人接口 /api/tickets，管理接口 /api/ticket-admin；个人中心始终只请求本人数据。管理读取与写入分别检查 tickets:read、tickets:write。正文仅以纯文本呈现，不进入外部通知文案。

详情返回最近 50 条升序消息及 has_more，使用 before_message_id 加载历史。分类 general/billing/technical/account，优先级 low/normal/high，状态 open/processing/resolved/closed/archived。关闭或归档后先重新打开才能回复；用户只能设置 open/closed。创建标题最多 120 个 Unicode 字符，正文最多 10000 个。

## 方案

复用 Classic Semi UI Card、Modal、表单与列表组件，增加懒加载路由、侧栏默认配置及个人/管理员设置入口。浮窗仅在登录后的控制台展示，与个人列表复用创建表单。请求失败保留草稿，并显示可重试错误；详情更新隔离过期请求，历史消息去重。

通知中心复用服务端事件目录，补充四个工单事件的七语文案和变量说明。新增 locale 键通过脚本统一写入并执行 i18n:sync，不修改锁文件。

## 验证计划

使用 \*.compat.test.jsx 测试真实组件交互与 API 边界：个人/管理路径、创建失败保留草稿、状态变更、关闭后重开、历史消息加载、权限限制、空/错/加载状态与移动端容器。执行受影响测试（总超时 60 秒）、修改文件格式/lint、Classic 生产构建及 git diff --check。

## 验证结果

### 交互与权限

- 路由为 `/console/tickets` 和 `/console/ticket-management`，详情支持 `?ticket=<id>`；浮窗创建成功后进入个人详情。
- 个人侧栏键为 `tickets`，管理键为 `ticket_management`；管理能力读取 `permissions.admin_permissions.tickets.read/write`。
- 保存回复或状态时禁止关闭按钮、Escape 和遮罩退出，失败保留草稿；关闭状态必须先重新打开。
- 列表及管理权限按当前账号 ID 与角色隔离，浮窗按账号 ID 重挂载。旧账号创建回调、列表、详情、历史消息和权限请求不能更新新账号界面。工单读取与权限读取不复用跨组件尚未完成的 GET Promise。
- 通知事件名称、描述及后端错误通过现有 i18next 翻译；新增 80 个键覆盖 en、zh-CN、zh-TW、fr、ja、ru、vi。通知中心保持 Root 配置边界，目标由通知任务配置，不自动向工单所有者推送。

### 已执行检查

- `node scripts/run-compat-tests.mjs src/pages/Tickets/__tests__/tickets.compat.test.jsx src/pages/Tickets/__tests__/lifecycle.compat.test.jsx src/pages/Tickets/__tests__/localization.compat.test.jsx`：23 项通过，10.46 秒。
- `node scripts/run-compat-tests.mjs src/pages/NotificationCenter/__tests__/ticket-events.compat.test.jsx`：1 项通过，42.70 秒。覆盖真实通知任务编辑、四类事件、默认模板切换、变量提示和保存原接收目标。jsdom 不执行 CSS 动画，测试只补发原生 animationEnd，不替换业务组件。
- `node --test src/pages/NotificationCenter/__tests__/filter-config.test.mjs src/pages/NotificationCenter/__tests__/modal-viewport.test.mjs`：15 项通过。
- 修改与新增源码文件 ESLint、Prettier 检查通过；`git diff --check` 通过。
- `bun run i18n:sync` 成功，七语已同步；临时翻译脚本已删除。
- Classic 当前为 JSX 工程，package.json 没有 typecheck 脚本；本次未新增 TypeScript。

依赖使用现有缓存 Bun 1.3.14 运行 `install --frozen-lockfile`，未修改依赖或锁文件。默认 npm exec 获取 Bun 1.4.4 的 Windows postinstall 失败后，改用已存在的本机二进制，不进行全局安装。

`bun run build` 最终生产构建通过，49.25 秒，已直接核验子进程退出码为 0。既有 Semi UI React 弃用提示、Browserslist 数据陈旧提示以及打包体积警告未在本工作项扩大处理。本地验证不代表生产部署、真实 Telegram 投递或人工浏览器截图验收。

## 页面溢出修正（2026-10-23）

用户截图显示工单中心与固定导航重叠。本地真实登录复现：1872×872 下标题顶边为 35px，导航底边为 64px；390px 和 320px 窄屏同样没有为导航预留空间。320×640 的创建弹窗沿用组件默认顶部外边距，加上表头和正文后超出屏幕。

修正计划：个人列表、管理列表及权限加载/错误状态统一使用已有 `classic-console-page` 和容器约定，保留业务面板边框与内边距；仅在工单样式内限制弹窗整体宽高，由正文滚动保证提交与回复按钮可达。避免修改全站导航和公共页面布局。使用真实 Chromium 对 1872×872、1440×1000、768×800、390×844、320×640 分别检查标题避让、文档宽度、浮窗边界、弹窗边界和提交按钮可达，并运行既有工单交互测试、lint、构建。

实际修复与验证：

- 个人与管理页面统一复用控制台外壳，业务面板保留 16px 内边距。弹窗使用视口宽高减 32px 的边界，正文滚动、表头固定保留，未修改全站公共布局。
- 23 项工单交互、生命周期、多语测试通过，15.92 秒。首次与构建同时运行触发 60 秒总超时；构建完成后单独重跑成功，没有延长单元测试超时。
- 工单页面 ESLint、页面与 CSS 的 Prettier 检查通过；最终 `node node_modules/vite/bin/vite.js build` 成功，1 分 14 秒。既有 Browserslist、lottie eval 和打包体积警告未扩大处理。
- 对最终构建产物执行 `.local-tests/ticket-preview/audit_classic_layout.py after`，未注入候选 CSS。正式表单登录后，5 种尺寸的个人与管理列表均无页面横向溢出，标题顶边 81px，高于导航底边 64px；浮窗保留屏幕右侧留白。
- 每种尺寸均打开创建弹窗与包含 120 字符连续长标题、5000 字符连续正文的详情，检查弹窗完整处于视口内、详情正文无横向溢出，提交和回复按钮可滚动访问。脚本退出码 0，并人工检查桌面列表和 320px 创建弹窗截图。
- 最终测量与 20 张截图位于忽略目录 `.local-tests/ticket-preview/layout/classic-after*`；文档布局约定同步到工单专题。本次仅修复本地界面，未提交或部署。

## 新建工单首次自动回复配置（2026-10-24）

管理员工单管理页新增“工单自动化”设置面板，允许具有 `tickets:read` 权限的管理员读取配置，具有 `tickets:write` 权限的管理员保存配置。配置使用现有认证会话，通过以下接口读写：

- `GET /api/ticket-admin/automation/auto-reply`：返回 `{ enabled, content }`。
- `PUT /api/ticket-admin/automation/auto-reply`：提交 `{ enabled, content }`；启用时内容不能为空，正文按工单正文上限校验。

开关默认关闭，内容预填“您好，已收到您的工单，我们会尽快处理。”。开启后，用户新建工单时由后端在同一工单事务中只发送一次配置内容；Classic 页面不会模拟回复，也不会在保存失败时清空草稿。只读管理员可以查看配置但不能编辑或保存。界面支持七语翻译，并覆盖加载失败重试、超长内容和空内容校验。

新增 `TicketAutomationSettings` 组件及兼容测试 `automation.compat.test.jsx`，验证配置加载、保存请求、启用时拒绝空内容和只读权限边界。

## 批量操作与超时归档（2026-10-24）

本次仅实现 Classic 工单页面；Default 与后端由独立工作项负责。遵循 `24_ticket_bulk_archive.md` 中的共同接口契约。

- 管理列表的桌面表格和手机卡片均支持勾选当前页工单，展示选中数量、批量目标状态、批量修改状态和批量删除入口。只读管理员与个人工单列表没有勾选和批量入口。
- 两类操作均需二次确认。状态变更调用 `PUT /api/ticket-admin/batch/status`，删除调用 `POST /api/ticket-admin/batch/delete`，均携带当前列表返回的 `{ id, version }`。删除为软删除，确认框明确说明双方列表隐藏、保留会话及审计、暂无恢复入口。
- 翻页、筛选、刷新及账号切换清空选择；请求失败保留选择并提示刷新重试。操作进行中禁用筛选、翻页、选择及确认框关闭，避免同时改变本批次范围。旧账号延迟结果不会刷新新账号列表或触发新账号提示。
- `archived` 与 `closed` 一样阻止回复，个人用户可重新打开归档工单，管理员可通过状态选择恢复。归档状态同步进入列表筛选、批量目标和详情状态选择。
- 新增独立超时归档卡片，通过 `GET/PUT /api/ticket-admin/automation/auto-archive` 读写 `{ enabled, inactive_days, statuses }`。默认关闭、30 天、已解决和已关闭；天数校验为 1–3650 整数，至少选择一种状态。可以显式包含待处理和处理中。
- 文案说明计时依据为最近创建、回复或状态变化；开启也适用于已有工单。只读权限禁用全部配置控件；读取失败没有可保存的默认表单，保存失败保留草稿。
- 修正原首次自动回复设置在校验/保存失败时隐藏表单的问题，允许直接修改草稿重试；保存回调增加卸载保护，配置 GET 关闭跨账号请求复用。
- 七语文案通过临时 `add-missing-keys.mjs` 脚本统一写入，随后使用缓存 Bun 1.3.14 执行 `bun run i18n:sync`，未更新依赖或锁文件。

验证计划和执行记录：

- 新增 `batch.compat.test.jsx` 和 `archive.compat.test.jsx`，覆盖版本请求、二次确认、取消删除、失败保留选择、翻页/筛选清空、只读权限、手机卡片、归档重开、配置校验与账号切换；补充原自动回复失败草稿和延迟回调回归。
- 实现前归档测试明确因缺少 `TicketArchiveSettings` 组件失败。初始批量测试冷启动触发 60 秒总超时，未把该超时当作业务断言失败；除并行编译争用外，还定位到原自动回复组件整体导入 `helpers` 引入无关依赖，改为直接导入 `helpers/utils`，后续批次均在 60 秒内完成。
- `node scripts/run-compat-tests.mjs src/pages/Tickets/__tests__/batch.compat.test.jsx src/pages/Tickets/__tests__/archive.compat.test.jsx src/pages/Tickets/__tests__/automation.compat.test.jsx`：17/17 通过，32.12 秒。
- `node scripts/run-compat-tests.mjs src/pages/Tickets/__tests__/tickets.compat.test.jsx src/pages/Tickets/__tests__/lifecycle.compat.test.jsx src/pages/Tickets/__tests__/localization.compat.test.jsx`：23/23 通过，26.57 秒。合计 40 项测试通过。
- Tickets JSX/JS 与测试的 ESLint、整个 Tickets 目录及七语文件的 Prettier、限定范围 `git diff --check` 通过。Classic 为 JSX，本次无 TypeScript 或独立 typecheck 脚本。
- 七语同步新增 27 个键，包含批量确认、归档配置、计数变量、扩展后的权限说明及 `actor_role` 的 user/admin/system 枚举说明；临时脚本已删除。
- 首轮 `node node_modules/vite/bin/vite.js build` 成功，4 分 18 秒。沿用已有 Browserslist 数据陈旧、lottie eval 和分包体积警告，没有扩大依赖调整范围。
- 主工作项真实浏览器确认批量改状态、软删除、归档配置以及 390/320/768px 页面无横向溢出；滚动到规则区发现导航透明导致文字穿过 Logo。追加工单限定 CSS：仅工单页面的固定导航采用 `--semi-color-bg-1` 不透明主题底色，面板预留 80px 滚动定位空间，不修改公共布局。新增 `layout.test.mjs` 先失败后通过。
- 通知中心仅在工单事件含 `actor_role` 变量时展示 user/admin/system 说明，覆盖自动归档的 system 操作方。该页面同样将整体 helpers 导入改为等价的直接导入，消除无关依赖带来的测试冷启动超时；新增变量说明断言先失败，随后补实现。
- 导航和通知说明修正后，`node --test src/pages/Tickets/__tests__/layout.test.mjs` 通过；通知中心交互和七语回归 3/3 通过，15.64 秒（含已计入前述总数的 2 项七语测试）。本次共有 42 个不同回归用例通过。
- 最终 `node node_modules/vite/bin/vite.js build` 成功，4 分钟，包含导航底色和 system 角色说明，退出码为 0；最终修改文件的 ESLint、Prettier 与限定范围 `git diff --check` 均通过。4174 静态预览已更新，最终导航视觉复核交由主工作项执行。
