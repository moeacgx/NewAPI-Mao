# 渠道优先级拖拽排序

## 目标

在 Default 渠道管理页提供可撤销的拖拽排序，让管理员能够直观看到并调整渠道优先级，同时保持原有筛选、分组和渠道能力排序契约。

## 交互范围

- 仅在按优先级排序、未启用标签聚合且拥有渠道写权限时启用拖拽模式。
- 表格和卡片视图共用拖拽句柄；支持键盘方向键移动，Escape 取消预览。
- 拖拽释放前显示目标优先级预览，释放后保存；保存失败可撤销并显示冲突提示。
- 使用 revision 做乐观并发控制，检测到其他管理员已修改时要求刷新后重试。
- 渠道优先级变化同步到对应的 `ability.priority`，保持能力路由选择一致。

## API 契约

- `GET /api/channel/priority/order` 返回当前 revision、可排序渠道和变更信息。
- `POST /api/channel/priority/reorder` 接收 revision、源渠道、目标渠道和位置（`before`、`after` 或 `same`），并返回新的排序结果。
- revision 不匹配、目标不在可排序范围或请求超出权限时返回明确错误，客户端保留当前列表并提示用户。

## 权限与数据库

查询要求 `ChannelRead`，重排要求 `ChannelWrite`。重排事务使用数据库行锁和 revision 校验，兼容 SQLite、MySQL 和 PostgreSQL；优先级使用既有整型字段，不改变渠道 ID、分组或计费标识。

## 验证

- `go test -timeout 60s ./model ./router ./controller`
- Default 前端相关 Vitest 与 typecheck
- `git diff --check`
