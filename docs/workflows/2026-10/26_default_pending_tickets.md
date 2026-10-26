# Default 待处理工单看板与浮动提醒

## 目标与范围

补齐 Default 数据看板与认证后台中的待处理工单提醒，沿用 Classic 的 open 口径。不修改后端、计费、外部通知或 Classic。

## 数据与权限

仅管理员且具有 tickets:read 时启用查询；每轮先复核 /api/user/self，再读取 /api/ticket-admin?p=1&page_size=1&status=open 的分页前 total。权限和列表请求各限时 10 秒，失败不得保留旧数量或阻塞后台。查询按用户、会话隔离，卸载取消请求，不持久保存统计。

## 交互与验证计划

认证布局共享 TanStack Query 查询，每 60 秒仅在可见时更新，返回标签页和工单变更使缓存失效。数据看板显示数量和处理入口；右上角通知可关闭、无背景遮罩、不抢焦点，连续后台导航不重复弹出。管理页自动隐藏通知，跳转携带 status=open 并联动筛选。

测试覆盖真实分页总数、权限、账号切换、请求失败/超时恢复、关闭后导航、工单变更缓存失效和 URL 筛选。执行相关 Vitest、类型检查、lint、格式及构建；生产部署和实际登录验收不在本次范围。

## 验证记录与限制

`features/tickets/__tests__/pending-tickets.test.tsx` 共 11 个测试：权限与 open 查询参数、0 数量不浮窗、普通用户零请求、权限收回、失败隔离、管理页隐藏；以及账号 A 未返回时切 B（A 的 signal 被取消且结果不泄漏）、超时清空旧数并重试恢复（两个请求均 timeout=10000）、`invalidateQueries(['tickets', userId])` 实际刷新、关闭后换页与新数量不重开、可见 60 秒轮询/隐藏暂停/回到可见聚焦立即刷新（fake timers）。`src/features/tickets` 其余与新增测试合计 61 个通过，另有 `pending-locales.test.ts` 7 个语言键测试，合计 68 个；新增文件 oxfmt、oxlint、tsgo 通过。

限制：未执行生产构建和真实浏览器验收；`/api/user/self` 与列表请求之间的服务端竞态以单次查询内复核身份为界，不做事务保证。

## 最终验证结果

Default 整合验证：`tsgo` 与 Default、Classic 两套构建通过，`src/features/tickets` 测试通过（工单 61 个 + 语言键 7 个 = 68 个），独立审查无阻断。未做真实浏览器登录验收，未部署；Classic 不在本 PR 范围。
