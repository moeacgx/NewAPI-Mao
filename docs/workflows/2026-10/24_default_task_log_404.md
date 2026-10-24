# Default 任务日志请求 404 修复

## 问题与根因

2026-10-24 核验 `https://zzapi.maolaoapi.com/usage-logs/task` 时，页面自身返回新版 Default 的 HTML，应用也能匹配任务日志页；但管理员日志查询请求 `/api/task?p=...` 返回 404，响应为 `Invalid URL (GET /api/task)`。

当前后端只注册管理员列表路由 `GET /api/task/`，个人列表路由为 `GET /api/task/self`。因此页面地址不是 404 来源，管理员日志 API 少了后端要求的尾斜杠才是根因。

## 范围与方案

- 仅修改 Default 的任务日志 API 调用和对应回归测试；Classic 不在本次范围。
- Default 管理员调用改为 `/api/task/`，个人调用继续使用 `/api/task/self`。
- 后端为 `GET /api/task` 增加与既有管理员列表相同权限的兼容路由，避免旧版或外部客户端继续收到 404。
- 不改变筛选参数、响应结构、鉴权、数据、数据库或任务日志业务语义。

## 验证计划

- 前端契约测试确认管理员列表带尾斜杠，个人列表使用 `/self`。
- 路由测试确认三条路径均已注册且拒绝匿名访问；管理员与个人的权限边界同时通过路由注册代码复核。
- 执行受影响 Vitest、Default TypeScript 检查、变更文件 lint、Go 路由测试、Markdown 格式检查及 `git diff --check`。
- 线上只读复核规范路径和旧路径状态；不执行部署或线上写操作。

## 验证结果

- 线上只读核验：`GET /api/task?p=1&page_size=20` 返回 404 `Invalid URL (GET /api/task)`；`GET /api/task/?p=1&page_size=20` 与 `GET /api/task/self?p=1&page_size=20` 均进入鉴权并返回 401，确认错误仅在管理员根路径缺少尾斜杠。
- Chrome 线上页面运行时，Default 任务日志页面和所有静态资源正常加载；性能资源记录显示页面实际请求 `/api/task?...` 并收到 404。没有触发任何写操作。
- 2026-10-25 提 PR 前同步至 `origin/custom-main` 的 `8d0f1874d`，保留最新文档索引条目并重新验证。
- `bun run test -- src/features/usage-logs --run`：16 个测试文件、61 项测试通过；包含管理员路径与个人 `/self` 路径的前端契约测试。
- `bun run typecheck`、`bun run build`：通过。
- `bun x oxlint -c .oxlintrc.json src/features/usage-logs/api.ts src/features/usage-logs/__tests__/api.test.ts`：通过。
- `go test ./router -run 'TestSetApiRouterRegisters(TaskRootWithAndWithoutTrailingSlash|ChannelRootWithAndWithoutTrailingSlash|SelfUpdateStatusRoutes)$' -count=1 -timeout 60s`：通过；回归测试验证无斜杠、有斜杠和个人路径都进入鉴权。
- `node web/node_modules/oxfmt/bin/oxfmt --check` 对本次 TypeScript 和 Markdown 文件、Markdown 索引链接检查、`git diff --check`：通过。Bun 直接执行 Markdown 格式检查出现 `DataCloneError`，使用同版本工具的 Node 入口完成检查。
- 全仓 `bun run format:check` 仍报告既有 Classic 格式问题；全仓 lint 仍报告其他既有文件问题。本次修改文件的定向 lint 和格式检查通过。

## 模板与跨项目影响

仅涉及 Default (`web/src`) 和共享 Go API 路由。Classic (`web/classic`) 未修改。
修复沿用既有任务日志 API，不改变与 TokensPro、Sub2API 等系统交互的请求/响应或计费契约。

兄弟项目 `NewAPIForDouDi`、`NewAPIModifyByGang` 均使用自身前端与路由实现；当前未核实其存在相同的尾斜杠错配，不建议照搬。
