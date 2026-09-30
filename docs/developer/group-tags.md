# 分组标签（MAO-6）

## 目标与范围

Classic 后台「分组与模型定价设置」新增「分组标签」页签，管理员可创建、编辑、删除标签，
配置名称、说明、排序、多个 Logo，并绑定已有分组。令牌创建和编辑可先按标签筛选再选分组；
分组管理表也提供标签过滤。截图对应 Classic，Default 不在本次页面范围内。

## 数据与接口契约

- `group_tags` 保存标签；`icons` 在数据库中存为 TEXT JSON，API 中为有序字符串数组。
- `group_tag_bindings` 以 `(tag_id, group_id)` 为联合主键。同一分组可属于多个标签，
  所有关联使用稳定 `Group.Id`，页面展示当前 `Group.Name`，不改变 code、历史 alias、路由或计费。
- `GET /api/group/tags` 返回所有标签，`POST /api/group/tags` 创建，
  `PUT /api/group/tags/:id` 完整更新单个标签，`DELETE /api/group/tags/:id` 删除标签及关联。
  四个接口均沿用分组管理的 AdminAuth 权限。
- 请求字段：`name`（1–64 字符）、`description`（最多 512 字符）、`sort_order`（整数，
  -1000000 至 1000000）、`icons`（最多 6 个，每个最多 2048 字符）、`group_ids`（最多 10000 个正 ID）。
  空 `icons` 使用默认图标；空 `group_ids` 允许先建标签。重复 Logo 和 ID 去重；不存在的分组拒绝保存。
- 返回标签包含 `id/name/description/sort_order/icons/group_ids`；按 `sort_order,id` 排序。
- `/api/user/self/groups` 和兼容的用户分组出口保留原 `data` 形状，追加 `group_tags`。
  仅下发与当前可用分组有交集的标签，关联 ID 也裁剪为该交集；不扩大选组权限。

示例：

```json
{
  "name": "国产模型",
  "description": "DeepSeek 与 Kimi 可用分组",
  "sort_order": 10,
  "icons": ["DeepSeek.Color", "Moonshot", "https://cdn.example.com/logo.png"],
  "group_ids": [311, 371]
}
```

## 安全与生命周期

- Logo 允许 LobeHub 名称（如 `OpenAI`、`Claude.Color`）或不含用户名/密码的 HTTP(S) 绝对地址。
  不接收脚本、HTML、data URL 或任意 SVG 文本。复用供应商 Logo 渲染器，浏览器加载图片、
  不发送 Referer，失败使用兜底；后端不下载图片。
- 标签仅用于分类展示。标签切换不更改已选分组、顺序、独立分组互斥、倍率保护、auto 或继承语义。
  `auto` 无实体 ID，不能绑定标签，在筛选中保留可达入口。
- 没有标签时保持原有选择方式；配置标签后提供「全部分组」「未分类」以及各标签。
- 前台创建/编辑令牌仅使用标签筛选，不包含标签管理操作。新建、编辑、删除标签仅在后台
  「分组与模型定价设置 → 分组标签」中提供。标签卡片统一 Logo、名称和数量槽位；
  长名称最多显示两行并保留完整标题，多 Logo 可分两行组合，避免相邻卡片上下错位。
- 分组改名或迁移 code 无需重绑；删除分组时在同一事务清理标签关联。
  删除标签不删除分组或更改令牌。旧分组配置接口不写标签，因此旧客户端保存不清空标签。
- 新表加入串行和并行迁移清单，使用 GORM、TEXT 和普通关联表兼容 SQLite/MySQL/PostgreSQL。
  回滚应用可保留新表，旧版本不使用标签。部署和生产数据库操作不属于本任务。

## 验证方法与边界

- 后端：CRUD、非法输入、更新回滚、空标签、多 Logo、按权限裁剪、分组改名/alias、删除清理；
  使用 SQLite 确定性测试，并检查 SQL 不依赖数据库专有语法。
- Classic：标签卡片键盘操作、多 Logo/URL、空态、切换保留跨标签多选、auto/独立分组互斥、
  编辑已有令牌和实际提交仍使用原 ID/code；管理绑定展示名称与 code 不同的数据。
- 执行受影响测试（单次最大 60 秒）、涉及文件 lint/格式化、Classic 构建、Markdown 链接与差异检查。
  实际执行结果见 [MAO-6 验收记录](../workflows/2026-09/27_group_tags.md)；未运行的数据库或线上验证不得视为通过。
