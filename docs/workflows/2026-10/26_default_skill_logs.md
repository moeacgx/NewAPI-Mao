# Default / Classic 使用日志技能角标与「技能」类型

## 目标与范围

- Default：技能图标从时间列迁到令牌单元格内分组标签右上角（不增列），悬停/键盘聚焦展示数量、名称或发现轮次；详情对话框保留。
- Default 与 Classic：「技能」作为“全部日志”类型下拉菜单中的一个选项，**不设独立技能筛选框**。Classic 的角标位置已合入，保持不变；其独立技能筛选框已移除。
- 不改 Go 后端：沿用已合并的 `skill=used` 参数。

## 交互与契约

| 菜单选择 | URL | 请求参数 |
| --- | --- | --- |
| 技能 | `type=0&skill=used` | `type=0&skill=used`（实际使用：始终装载或 generation 阶段已读；发现/未用排除） |
| 消费等原类型 | `type=N`，不含 `skill` | `type=N` |
| 全部类型 | `type=0`，不含 `skill` | `type=0` |
| 重置 | 清除 `skill` | — |

- “技能”是纯前端虚拟菜单值（Default 为 `skill`，Classic 表单值为 `skill`），提交时转换，**不会作为数值发给后端**。
- 切换类型会清除残留的 `skill`；选择技能回到第一页，翻页保留条件。
- Default：列表与费用统计共用 `buildApiParams`，`skill=used` 固定 `type=0`；仅带 `skill=used` 的旧链接同样映射为 `type=0` 并在下拉中显示“技能”。路由 search schema 增加 `skill`（仅接受 `used`）。
- Classic：`getFormValues` 将 `logType==='skill'` 映射为 `logType=0` + `skill='used'`，列表/统计/连接逻辑沿用既有 `skill` 参数。
- 文案复用现有 `Skills` / `技能`；类型下拉新增无障碍名称复用 `Log Type` / `日志类型`。

## 验证

- Default：`skill-type-filter.test.tsx`（类型菜单唯一、选择技能/消费/全部/重置的 URL 提交、列表与统计参数一致、旧链接映射）、`skill-usage-badge.test.tsx`（令牌列角标）；usage-logs 与 routes 全量 Vitest（102 日志 + 61 tickets + 7 locale）、`tsgo -b`、oxlint、oxfmt。
- Classic：`admin-user-filter-layout.compat.test.jsx`（无独立技能框，菜单选择”技能”与重置，管理员/普通用户）、`user-filter.test.mjs`（logType=skill 映射 type0、切消费 type2 和全部 type0 清 skill）；usage-logs 目录 compat 测试 20 组件 + 15 hook。
- 两套模板生产构建通过；未做真实浏览器视觉验收。
