# 供货商申请列表上的对话按钮

## 问题

管理列表只有站点、申请人、联系方式、分组、状态、提交时间和处理人，没有对话入口。未读为 0 时站点名称旁什么都不显示。详情抽屉底部还有一个没有图标的「对话」文字按钮。

## 方案

- 基线是 `bc075ee74b902e67ec65bc0c747376d22780571a`。不从 `custom-main` 或旧的消息计数提交另起对话。
- Default 与 Classic 的管理列表、我的申请列表，每一行站点名称旁都放同一个带消息图标的按钮。不新增一列。
- 按钮沿用现有表格控件：Default 用小号描边按钮，Classic 用 Semi 小号按钮。无未读时只有图标；`unread_count > 0` 才在按钮上显示数量。数量仍只算别人发来的对话，自己的消息和内部备注不计，也不显示 `message_count`。
- 点击列表按钮直接打开现有对话弹窗，使用列表里已经返回的 `messages`，不先打开详情，也不另请求消息接口。打开弹窗仍调用原来的已读接口。
- 详情底部的纯文字按钮删除。详情标题旁保留同一套图标按钮。弹窗、已读语义、余额、渠道、兑换码、分组和计费都不改。
- 按钮文案继续用已有的 `Conversation` 和 `{{count}} unread`，两套模板的现有语言文件里都有，没有新增词条。

## 验证

- Default：在 `web/` 执行 `bun run test -- src/features/supplier-applications/__tests__/supplier-applications.test.tsx`，17 项通过，退出码 0。
- Classic：在 `web/classic/` 执行 `bun run test:compat src/components/supplier/__tests__/message-count.compat.test.jsx`，4 项通过，退出码 0。

未覆盖：浏览器实播、MySQL、PostgreSQL、生产站。没有推送、开 PR、合并或部署。

## 清理

没有新建数据库、容器或长期进程。原来只负责显示未读数字的 `message-count.tsx` 和 `SupplierMessageCount.jsx` 已删除，数量改由按钮自己显示。
