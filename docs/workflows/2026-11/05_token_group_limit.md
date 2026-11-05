# 令牌分组数量上限

## 问题与根因

`MaxTokenAutoGroups` 默认 5，并且已经能通过通用选项读写。分组设置保存接口的可写选项名单没有包含它，Classic 分组设置也不展示该值，所以运营侧实际改不到上限。

令牌创建和编辑只限制自定义 Auto 快照。显式多分组仍可保存超过上限的 `group` 列表。请求进入 `TokenAuth` 后，`ParseTokenGroupList` 发现分组数大于上限时返回 `token.invalid`（「无效的令牌」）。历史超限令牌因此在模型列表和对话接口上像密钥失效。

## 方案与安全边界

- 分组配置保存允许写入 `MaxTokenAutoGroups`。值必须是正整数，缺省和未覆盖时仍是 `DefaultMaxTokenAutoGroups`（5）。
- Default 与 Classic 的分组设置都展示「令牌最大分组数」，说明同时限制显式分组和自定义 Auto 快照，继承全局 Auto 顺序不受限。
- 创建和编辑令牌时，显式分组（分组 code 列表或稳定分组 ID）超过上限则拒绝保存，返回 `token.groups_too_many`（「令牌最多可选 N 个分组」）。仅改状态不改分组，便于停用历史令牌。
- `TokenAuth` 对已经保存的超限令牌返回同一超限文案，不再使用「无效的令牌」。分组列表无法解析时仍按无效令牌处理。提高上限或把分组数降到上限以内后，原令牌可以继续通过鉴权。
- 不新增表、不改历史令牌数据，也不放宽分组权限、倍率或 Auto 继承范围。

## 验证计划与结果

本地 SQLite 已通过：

- `go test ./controller -run 'TestAddAndUpdateTokenRejectExplicitGroupsOverLimit|TestUpdateTokenStatusOnlyKeepsHistoricalOverLimitGroups|TestAddTokenRejectsInvalidAutoGroups|TestGetTokenAutoGroupsReturnsFullFilteredGlobalOrderAndLimit'`
- `go test ./middleware -run TestTokenAuthOverLimitGroupsUsesExplicitError`
- `go test ./model -run 'TestMaxTokenAutoGroupsOptionReadWrite|TestValidateOptionValueRejectsInvalidMaxTokenAutoGroups'`

覆盖超限创建/编辑拒绝（中英文文案）、达到上限可以保存、仅改状态保留历史超限分组、鉴权 403 超限文案，以及把上限提高到所选分组数后同一令牌恢复 204。选项测试覆盖写入 8、拒绝 0，以及分组设置白名单接受 6、拒绝 -1。

前端：

- Default：`bun run test` 覆盖 `auto-group-form.test.ts`、`api-keys-mutate-drawer.test.tsx`、`api-key-group-combobox.test.tsx`（27 项通过）。达到上限后未选中的普通分组被禁用，保存仍提交已选分组。
- Classic：`node scripts/run-compat-tests.mjs src/components/table/tokens/__tests__/token-group-limit.compat.test.jsx` 通过。达到上限后额外分组按钮禁用，已超限选择显示明确文案。

未在浏览器里登录后台点选设置页和令牌抽屉。未跑 MySQL / PostgreSQL。本次没有新增 SQL，分组数量判断在请求校验和鉴权中完成。仅本地验证，不包含合并、发布或生产部署。
