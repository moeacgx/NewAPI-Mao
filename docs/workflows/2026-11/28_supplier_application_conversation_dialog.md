# 供货商申请对话弹窗与未读数

## 问题

供货商申请详情把整段对话嵌在资料下面。桌面抽屉偏窄，消息一多，资料和处理状态被对话挤掉。列表上的数字是消息总数，自己刚发出的消息和已经看过的对话也会占一个数，看不出还有没有新回复。

## 方案

- 基线是 `813412ee35e7bfc99a3f0786d8ffdc12ec2064cd`。Default 与 Classic、管理端与申请人端一起改。
- 详情只保留申请资料、站点、联系方式、分组报价，以及管理端默认折叠的处理状态。详情里不放对话记录、消息类型、输入框或发送按钮。
- 详情提供「对话」按钮。桌面弹窗宽 56rem（Classic 模态框 960），并且不超过视口；窄屏高度不超过 `100dvh`。弹窗挂到页面顶层，避免抽屉的变换把固定定位困在一条窄栏里。
- 弹窗保留原有对话能力。消息列表单独滚动，发送区固定在底部，对话槽使用 `min-height: 0` 和 `overflow: hidden`。关闭弹窗回到仍打开的详情；这段时间组件不卸载，未发送草稿还在。关掉详情后草稿可以丢掉。
- `message_count` 仍是对话总数。新增 `unread_count`：当前用户、当前申请、已读位置之后、不是自己发送的消息。内部备注不是对话行，自然不计。申请人响应仍不含内部备注、处理人和版本。
- 已读位置存在 `supplier_application_reads`，按用户和申请唯一。`POST /api/supplier-applications/:id/read` 与 `POST /api/supplier-application-admin/:id/read` 只在打开弹窗时调用，把游标推进到当时的最后一条消息，且只向前。管理端标记已读只需要 `read`。列表和详情的 GET 不写游标。不改余额、兑换码、渠道、分组或计费，标记已读不写审计。

## 验证

- `go test -count=1 -timeout 60s ./router/ -run "TestSupplierApplicationUnread|TestSupplierApplicationListMessageCount|TestSupplierApplicationMessagesAreIsolated"`
- Default：`bun run test -- src/features/supplier-applications/__tests__/supplier-applications.test.tsx`
- Classic：`bun run test:compat src/components/supplier/__tests__/message-count.compat.test.jsx`

三条都在 60 秒内结束。Go 覆盖未读计数、打开弹窗对应的已读接口清零、自己的消息不计、内部备注不计、只读管理员可标记已读但不能发消息。两套前端覆盖详情默认不显示对话、打开弹窗后可见并可发送、未读大于 0 才显示数字、只打开详情不标记已读。

未覆盖：浏览器实播、MySQL、PostgreSQL、生产站。没有推送、开 PR、合并或部署。

## 清理

没有新建数据库、容器或长期进程。插入文案时用过的一次性脚本已删除。
