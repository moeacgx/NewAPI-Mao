# Classic 使用日志用户名 / 用户 ID 筛选（MAO-4）

## 目标与范围

Classic `/console/log` 的管理员筛选支持在“用户名”和“用户 ID”之间切换。
用户收到 ID 后可直接查询日志，不需要先反查用户名；本次不修改 Default 模板。

## 接口与行为契约

- 模式为“用户名”时，管理员日志列表和统计继续发送 `username`，保持原有匹配行为。
- 模式为“用户 ID”时，发送 `user_id`，后端按 `logs.user_id = ?` 精确筛选列表及统计，支持用户改名前的历史日志。
- 空筛选不发送有效用户限制；非法、零、负数或溢出 ID 由管理员接口返回 HTTP 400，避免把输入错误静默当成无结果。
- 普通用户日志列表仍使用 `/api/log/self/`，统计使用当前会话用户 ID；查询参数不能扩大可见范围。
- 既有模型函数保留无用户 ID 参数的兼容包装，新增的按 ID 函数只由日志控制器使用。

## 修改范围与回滚

修改 Classic 筛选组件、使用日志请求 Hook、Go 日志控制器和日志模型；新增控制器 ID 解析、模型列表/统计回归测试及 Classic 筛选契约测试。
不涉及数据库迁移、计费、认证或部署；回滚这些文件即可恢复原行为。

## MAO-4 对抗修复计划

- 目标：仅修复 Classic 使用日志 Hook 的查询参数编码和请求 loading 生命周期，不改变后端、Default 模板、筛选字段或分页/统计语义。
- 方案：四条管理员/普通用户列表与统计请求统一使用 Axios `params`，让 Axios 负责编码；请求失败由 Hook 捕获，`loading`/`loadingStat` 在 `finally` 中复位。统计函数返回成功状态，刷新在统计失败时停止列表请求，避免同一错误重复提示。
- 行为测试：用实际 `useLogsData` Hook、esbuild JSX 转译、标准 `react-test-renderer` 和外部 helpers/API mock，验证特殊字符筛选值不会生成额外参数、用户名/用户 ID 模式及空值语义、管理员/普通用户四条请求、HTTP 400/网络失败后的 loading 复位，以及 refresh 失败单提示和下一次成功更新列表/统计。
- 验证计划：受影响行为测试（单次不超过 60 秒）、Classic 受影响文件 ESLint、Prettier、`git diff --check` 和 Classic `npx --yes bun run build`；不重复本轮已有 Go 测试，不进行浏览器或部署验收。测试所需 `react-test-renderer@18.3.1` 作为 Classic 精确 devDependency 声明。

## 验证结果

- `go test ./controller ./model -run 'TestLogUserID|TestSearchUsers' -count=1 -timeout=60s`：通过；覆盖解析边界、非法 ID 400、列表按用户 ID 隔离及统计精确值。
- `node --test --test-timeout=60000 web/classic/src/components/table/usage-logs/__tests__/user-filter.test.mjs`：4 项通过，耗时约 0.16 秒；覆盖真实 Hook 查询参数、失败 finally、refresh 单提示和后续成功更新。
- `npx --yes eslint src/hooks/usage-logs/useUsageLogsData.jsx src/components/table/usage-logs/UsageLogsFilters.jsx src/components/table/usage-logs/__tests__/user-filter.test.mjs`：通过。
- `npx --yes prettier --check ...`（Hook、筛选组件、行为测试、Classic `package.json` 和本专题文档）：通过。
- `git diff --check`：通过；仅报告 `bun.lock` 的工作树换行提示，无空白错误。
- `npx --yes bun run build`（Classic）：通过，约 1 分 12 秒；仅有 Browserslist 数据过旧、第三方 `eval` 和大分块提示。
- 未进行真实浏览器交互或线上部署。

## MAO-4 用户筛选布局回归

- 背景：后续 Classic 使用日志筛选将管理员用户名/用户 ID 模式选择器与输入框放在不同网格单元，导致窄屏或换行时两者不相邻，控件关系不清晰。
- 修改范围：仅调整 `UsageLogsFilters.jsx` 中管理员字段顺序，将渠道筛选提前，并把 `userSearchType` 与 `username` 输入收进同一容器；不重构整个筛选表单。
- 行为契约：桌面端模式选择器与输入框相邻；窄屏下保持成组且不被其他筛选项插入。输入提示随用户名/用户 ID 模式切换；保留 `username`、`userSearchType` 字段、默认用户名模式、清空、重置和现有查询参数行为；仅管理员可见。
- 测试计划：使用 Classic Vitest 与 React Testing Library 渲染真实筛选组件，验证组合语义、模式提示变化、重置回默认用户名以及普通用户隐藏管理员筛选。
- 验证结果：Classic 组件与元数据测试 4 项通过；本地真实 Chrome 验证桌面并排、窄屏相邻、用户 ID 切换和输入成功，生产构建通过。完整范围与证据见 [网站 SEO 与 Classic 日志筛选工作记录](23_site_seo_and_classic_log_filters.md)。
