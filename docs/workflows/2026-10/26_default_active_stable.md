# Default 当前连接正常轮询布局稳定

## 问题与根因

Default 使用日志的当前连接面板和详情抽屉共用 `ConnectionNotice`（`web/src/features/usage-logs/components/active-connections/active-connections-panel.tsx`）。已有快照且每 2 秒轮询进入 `refreshing` 时，`ConnectionNotice` 会插入一行 `Loading connection status…`，轮询结束后该行消失，内容高度随之跳动。

## 修改范围

- 新鲜快照的正常轮询静默刷新，不再渲染加载提示行。
- 保留首次加载提示、延迟超过阈值时的 `Connection data is delayed…` 提示、错误与重试、过期提示、实例范围提示。
- 面板与详情抽屉复用同一个 `ConnectionNotice`，两处同时生效。
- 不新增 UI 元素、文案或语言包条目。请求、权限、计费、轮询间隔、分页和查询身份隔离均不变。

Classic 已在独立工作项修复，本次只修改 Default。Default 令牌页面没有接入此面板或独立的令牌活动栏，因此不新增令牌入口。

## 验证

- `web/src/features/usage-logs/__tests__/active-connections.test.tsx` 中“正常轮询”用例：第二次请求进行中，断言没有加载提示行、没有 `status` 角色元素、连接卡片保留；请求变慢后出现延迟提示；请求成功后提示消失。
- 运行相关 Vitest、Default 类型检查，以及修改文件的 lint 和格式检查。
- 没有生产请求、发布或部署。

## 实际验证结果

- `active-connections.test.tsx`：11 个测试全部通过。
- `tsgo -b`、`oxlint`（改动文件）退出码 0；改动文件与本文档 `oxfmt --check` 通过；`git diff --check` 通过。
- 主会话整合验证：Default 日志/路由 102 项、工单 61 项、语言包 7 项，Classic 组件 20 项与 hook 15 项通过；Default `tsgo` 与两套构建通过。
- 未做浏览器或截图验收。抽屉未单独断言正常轮询，依靠与面板共用 `ConnectionNotice` 保证。
