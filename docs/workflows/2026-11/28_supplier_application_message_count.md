# 供货商申请消息数量与处理块位置

## 问题

管理列表「全部供货商申请」和申请人的「我的申请」无法看出哪些申请已有对话。管理详情里，状态下拉和内部备注放在对话下面，处理时要先翻过整段对话。

## 方案

- 个人和管理列表、详情都返回 `message_count`。它等于该申请的对话条数，四种对话类型都计入，内部备注不计入。
- Default 与 Classic 只在数量大于 0 时，于站点名称旁显示消息图标和数字。前端不按申请再请求对话。
- 发送一条新对话后，两套模板都把当前列表里的数量加一，不重新拉取对话。
- 管理详情只把处理块移到申请人、网站、分组报价和对话之前。状态流转、权限和字段含义不变。申请人详情仍然没有状态编辑和内部备注。
- 不改余额、兑换码、渠道、分组、计费，也不改对话的写入规则。

## 验证

- `go test -count=1 -timeout 60s ./router -run TestSupplierApplicationListMessageCountIgnoresInternalNotes`
- Default：`bun run test -- src/features/supplier-applications/__tests__/supplier-applications.test.tsx`
- Classic：`bun run test:compat src/components/supplier/__tests__/message-count.compat.test.jsx`

未覆盖：MySQL/PostgreSQL 实库、浏览器实播、生产站。

## 清理

测试使用隔离 SQLite，随测试进程结束。不保留本任务的临时库、临时进程或脚本。
