# Default 工单中心实施记录

## 目标与范围

基于 `origin/custom-main` 的 `dea7b5d18`，在独立工作树实现 Default 模板的个人工单中心、管理员工单管理、登录控制台浮动创建入口与通知中心工单文案。Classic 和后端由独立 Agent 实现，不在本工作树修改。

## 数据、接口与权限

遵循主工作项 `docs/developer/tickets.md`：个人 API 为 `/api/tickets`，管理 API 为 `/api/ticket-admin`；本人入口对管理员仍限定本人。管理入口使用 `tickets:read`，回复和状态变更使用 `tickets:write`。消息采用 `before_message_id` 分页，默认最近 50 条。关闭工单必须重新打开才能回复。

标题最多 120 个 Unicode 字符，消息最多 10000 个。正文只在工单详情展示，不进入通知事件。管理员不能在管理接口代用户创建工单。

## 实现方案

- 复用 React Query、React Hook Form、Zod、Card、Dialog、表格与原生选择器。
- 个人与管理员页面共享业务组件，查询键包含登录用户与视角，避免切换视角串用缓存。
- 创建失败保留输入，详情加载失败提供重试；历史消息可继续加载，回复后刷新列表与详情。
- 浮窗放入已登录控制台布局，使用弹窗焦点管理，移动端限制高度并可滚动。
- 同步侧栏默认配置、显示开关、通知事件与变量说明，以及全部七种语言。

## 验证计划

真实交互测试覆盖创建校验和失败保留、空列表、失败重试、筛选分页、管理只读、关闭重开后回复、加载历史消息、键盘与弹窗焦点恢复。执行受影响测试（60 秒上限）、类型检查、修改文件 lint、生产构建、格式与 `git diff --check`。

## 验证结果

- 新增工单模块 22 项测试通过，涵盖实际表单操作、请求失败、筛选分页、关闭重开、管理员只读、管理员回复、历史游标、刷新、文本安全、焦点恢复与输入边界。合并执行现有侧栏和通知测试共 7 个测试文件、32 项测试通过。
- 测试命令：`node node_modules/vitest/vitest.mjs run src/features/tickets/__tests__ src/hooks/use-sidebar-config.test.ts src/features/notification-center/__tests__ --maxWorkers=1 --pool=threads`，外层 Node `spawn` 设置 60000 毫秒超时。Windows 默认 fork 池曾启动超时，threads 单 worker 可在限制内完成。
- `bun run typecheck`、受影响 TypeScript/TSX 的 oxlint、`bun run build` 均通过；构建自动生成了两条工单路由。测试中的查询参数类型与配置联合类型已修正，未放宽生产类型。
- 七语通过 `add-missing-keys.mjs` 写入，再执行 `bun run i18n:sync`；各语言新增 78 个键，既有翻译值保持不变。临时写入脚本在验证后移除。
- 共享侧栏配置明确为 `personal.tickets` 和 `admin.ticket_management`；个人 URL `/tickets`，管理 URL `/ticket-management`。
- `npm exec --yes --package=bun -- bun install --frozen-lockfile` 安装成功，锁文件未变化。随后 npm 最新 Bun 1.4.4 的 postinstall 在本机退出 1；固定 `npm exec --yes --package=bun@1.4.3 -- bun run ...` 完成检查，未升级项目依赖。
- 本工作项未提交、推送或部署；仅验证 Default。本地交互测试模拟 HTTP 边界，不代表真实后端联调、浏览器像素验收或 Telegram 实际投递。

## 账号切换异步回调隔离

新增延迟 Promise 交互回归：账号 A 提交创建、回复或状态变更，在响应前切换账号 B 并填写新草稿，再完成 A 的成功或失败响应。修复前 8 个场景全部失败：创建成功会通过共享父回调关闭 B 的弹窗，创建失败及详情的回复/状态回调会向 B 展示旧账号通知。按用户 ID 重挂载组件并不会取消 React Query mutation 的 options 回调。

修复方案：在每次 mutation 的 `onMutate` 中记录提交身份，成功和错误回调执行前读取当前 auth store，只有身份一致时才更新表单、弹窗、通知或使查询缓存失效。保持 API 请求契约不变，不取消已提交的服务端操作。

修复后 8 项账号切换回归全部通过；合并原有工单、侧栏和通知测试后，共 8 个文件、40 项测试通过（33.20 秒，外层限制 60 秒）。类型检查、三个受影响 TypeScript/TSX 文件的 oxlint、Default 生产构建均通过。原有同账号成功/失败路径也在此次完整回归中通过。

## 页面溢出回归（2026-10-23）

- 真实浏览器在 1872×872、1440×1000、390×844、768×800 检查长标题和 5000 字符正文；修复前详情 `scrollWidth=46781`，原因是网格子项的最小内容宽度被连续文本撑开。
- Default `TicketDetailDialog` 增加视口两侧 2rem 留白、单列可收缩网格和标题任意位置换行；正文改为纵向滚动，详情 `scrollWidth === clientWidth`，回复按钮可滚动到达。
- 列表标题按钮改为块级、`max-w-72` 和正常换行，长标题不会覆盖用户、状态和更新时间列。新增两条布局回归测试，相关 15 项通过；Default 类型检查、oxlint、格式检查和生产构建通过。
- 768px 独立新页面加载确认文档宽度等于视口宽度；此前连续调整视口造成的瞬态宽度不属于实际页面溢出，未修改共享布局。
- 截图证据：`.local-tests/ticket-preview/screenshots/default-after-1872-long-detail.png`、`default-after-1440-long-detail.png`、`default-after-390-long-detail.png`、`default-after-768-long-detail.png`。

## 新建工单自动回复配置（2026-10-24）

管理员在 Default 模板的“工单管理”页面可配置“用户提交后自动回复一次”。配置卡片仅在管理员拥有工单读取权限时显示；读取使用 `tickets:read`，保存使用 `tickets:write`，无写权限时以只读方式展示。

前端调用 `GET` 和 `PUT /api/ticket-admin/automation/auto-reply`，请求和响应字段均为 `enabled`、`content`。启用时内容必填，最大 10000 个 Unicode 字符；默认关闭并预填“您好，已收到您的工单，我们会尽快处理。”。保存成功后刷新配置缓存并提示结果，失败时保留编辑内容。

后端在创建工单事务中最多插入一条管理员自动回复消息，不生成额外通知事件；前端只负责配置，不模拟自动回复执行。

## 批量操作与超时归档（2026-10-24）

按 `24_ticket_bulk_archive.md` 的共享契约扩展 Default：管理员可勾选当前页工单，查看所选数量，并二次确认批量修改状态或批量删除。只读管理员不展示选择与批量操作。选择保留所见 `id/version` 快照；分页、筛选、刷新和账号切换清空选择，失败时保留选择并提示刷新后重试。旧账号的延迟响应不修改当前账号选择或弹出通知。

- 批量状态：`PUT /api/ticket-admin/batch/status`，提交 `{tickets:[{id,version}],status}`。
- 批量删除：`POST /api/ticket-admin/batch/delete`，提交 `{tickets:[{id,version}]}`。确认框明确说明双方不再可见、会话与审计保留、当前无恢复入口。
- `archived` 独立展示为已归档，进入筛选及管理员状态列表。归档工单与关闭工单一样必须先重开才能回复；个人用户可以重开自己的归档工单。
- 自动归档：`GET/PUT /api/ticket-admin/automation/auto-archive`，字段 `enabled/inactive_days/statuses`。默认关闭、30 天、已解决和已关闭；前端校验 1–3650 整数天数及非空状态集合。界面说明按回复或状态变化计算无活动时间、启用后覆盖已有工单，以及主动选择待处理/处理中状态的影响。
- 自动回复与归档配置查询均按账号隔离，后台刷新不覆盖未保存草稿；保存失败保留输入，旧账号保存响应不写入新账号缓存。保留既有单次自动回复功能。
- 新增 27 个界面文案及 1 个完整写权限说明的七语翻译，经 `add-missing-keys.mjs` 写入及 i18n 同步；临时脚本已移除。写权限说明明确回复、改状态、删除和管理自动化规则。批量按钮允许换行，确认框限定视口宽高并可滚动，不改动共享布局。

### 验证

先补交互回归再实现；首轮冷启动验证触发 60 秒上限，未取得缺失能力断言结果，因此不将其计为有效红灯证据。后续按不超过 60 秒的独立批次验证：

- 批量/API/输入校验/侧栏：4 文件 15 项通过，36.39 秒。
- 自动归档与自动回复配置：2 文件 11 项通过，21.38 秒。
- 既有交互/账号切换/布局：3 文件 24 项通过，55.99 秒。
- 合计 9 文件 50 项通过。包含归档后阻止回复、显式重开恢复、批量确认/取消/错误保留、选择跨账号隔离、后台刷新保留草稿及旧账号保存回调隔离。
- `tsgo -b`、工单目录 oxlint 与 oxfmt、Markdown 格式化、限定范围 `git diff --check` 均通过；Default Rsbuild 首轮生产构建成功，耗时 87.5 秒。补齐七语完整权限说明后再次最终构建成功，耗时 52.2 秒；当前预览产物包含全部本轮改动。
- 七语脚本同步完成后，再通过 `npm exec --offline --yes --package=bun@1.4.3 -- bun run i18n:sync` 执行仓库规定的 Bun 同步流程，未新增或升级项目依赖。

该批组件测试模拟 HTTP 边界，不能替代真实后端联调；双模板与后端的浏览器验收由主工作项统一记录。本代理没有提交、推送或发起 PR。

### 通知变量说明补齐

自动归档事件的 `actor_role` 可为 `system`。Default 通知中心、静态翻译键与七种语言说明同步改为“用户、管理员或系统”，移除只列用户和管理员的旧键。此项为描述修正，不改变变量名称、通知负载或模板接口。通知中心既有 2 文件 6 项测试通过，类型/lint/格式通过；该描述已纳入下述最后一次生产构建。

### 320px 公共页头边界修复

主工作项真实业务验收发现 320px 屏幕的文档宽度为 346px。复用一次正式登录，只读对照 `/ticket-management` 与 `/dashboard/overview`，两页均复现同样的公共页头溢出：右侧操作区从 `x=172.47` 延伸至 `346.47`，个人头像位于视口外；工单卡片不是根因。带“搜索”文本的搜索按钮占 66px。

获得主代理授权后，仅调整 `app-header.tsx` 的手机搜索按钮为 32px 图标入口；保留 `aria-label`、搜索点击行为及桌面宽度，不隐藏品牌、通知、语言或个人账户入口，不对整个页面裁剪溢出。同步清理该文件原有的类型导入和单子节点 Fragment 两条 lint 问题，不改动 `Header`、`SystemBrand`、`Search` 等公共组件实现。

新增 `components/layout/components/__tests__/app-header-layout.test.tsx`，先确认旧代码因缺少手机宽度限制失败（19.25 秒），实现后 1 项通过（15.20 秒），验证可访问名称、点击搜索行为及品牌/语言入口保留。最终类型检查、修改文件 lint 与格式通过；包含此次页头及通知说明的最后一次 Default 生产构建成功，耗时 75.7 秒。

使用真实生产构建与一次正式登录，复测 320/390/768/1440px 的工单管理和控制台概览共 8 组页面：文档 `scrollWidth` 全部等于视口宽度，所有可见页头按钮均在视口内；每组均可点击搜索打开弹层并使用 Escape 关闭。320px 两页由 346px 修为 320px，手机搜索按钮实测 32px。只读验收未创建工单或修改配置。

截图与测量记录位于 `.local-tests/ticket-preview/bulk-screenshots/`：`header-after-320-ticket-management.png`、`header-after-320-overview.png`、`header-after-1440-ticket-management.png`、`header-after.json`。已目视检查手机和桌面截图。主工作项继续完成双模板业务流程总验收。
