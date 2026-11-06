# 令牌分组数量上限

> 本文记录 `.365` 的原始修复和当时验证结果。后续 `.366` 将默认值改为 `0`（不限制），并允许保存 `0`；当前配置契约见[令牌分组默认不限制](06_unlimited_token_groups.md)。下文“默认 5、拒绝 0”属于历史行为。

## 问题与根因

`MaxTokenAutoGroups` 默认 5，并且已经能通过通用选项读写。分组设置保存接口的可写选项名单没有包含它，Classic 分组设置也不展示该值，所以运营侧实际改不到上限。

令牌创建和编辑只限制自定义 Auto 快照。显式多分组仍可保存超过上限的 `group` 列表。请求进入 `TokenAuth` 后，`ParseTokenGroupList` 发现分组数大于上限时返回 `token.invalid`（「无效的令牌」）。历史超限令牌因此在模型列表和对话接口上像密钥失效。

## 方案与安全边界

- 分组配置保存允许写入 `MaxTokenAutoGroups`。值必须是正整数，缺省和未覆盖时仍是 `DefaultMaxTokenAutoGroups`（5）。
- Default 与 Classic 的分组设置都展示「令牌最大分组数」，说明同时限制显式分组和自定义 Auto 快照，继承全局 Auto 顺序不受限。
- 创建和编辑令牌时，显式分组（分组 code 列表或稳定分组 ID）超过上限则拒绝保存，返回 `token.groups_too_many`（「令牌最多可选 N 个分组」）。仅改状态不改分组，便于停用历史令牌。
- `TokenAuth` 对已经保存的显式多分组超限令牌返回同一超限文案，不再使用「无效的令牌」。分组列表无法解析时仍按无效令牌处理。提高上限或把分组数降到上限以内后，原令牌可以继续通过鉴权。历史 Auto 自定义快照仍沿用既有筛选和截断行为，本次不改为鉴权 403。
- 保存前复用模型分组解析，按归一化后的实际分组数校验；同一分组的 code 与历史 alias 不重复计数，重复项和空项也不会绕过上限。
- 不新增表、不改历史令牌数据，也不放宽分组权限、倍率或 Auto 继承范围。

## 验证计划与结果

### 二次审查与补充验收计划

- Default 入口位于 `/system-settings/billing/group-pricing`，Classic 入口位于设置页 `?tab=ratio` 的分组相关设置。两套模板均需验证可见输入、修改、提交 `PUT /api/group/details` 的 `option_updates.MaxTokenAutoGroups` 和重新加载，不能仅凭字段源码存在宣称可配置。
- 为两套模板补充设置保存交互回归，覆盖合法正整数及无效值；Classic 同时覆盖可视化和手动编辑模式。
- 核查显式分组字符串含重复项、空项时的校验与模型归一化契约，防止先绕过上限校验再保存超限分组。
- 根据用户要求，根目录及前端 `AGENTS.md` 统一规定所有功能同步覆盖 Default、Classic，分别记录验证依据。

本地 SQLite 已通过：

- `go test ./controller -run 'TestAddAndUpdateTokenRejectExplicitGroupsOverLimit|TestUpdateTokenStatusOnlyKeepsHistoricalOverLimitGroups|TestAddTokenRejectsInvalidAutoGroups|TestGetTokenAutoGroupsReturnsFullFilteredGlobalOrderAndLimit'`
- `go test ./middleware -run TestTokenAuthOverLimitGroupsUsesExplicitError`
- `go test ./model -run 'TestMaxTokenAutoGroupsOptionReadWrite|TestValidateOptionValueRejectsInvalidMaxTokenAutoGroups'`

覆盖超限创建/编辑拒绝（中英文文案）、达到上限可以保存、仅改状态保留历史超限分组、鉴权 403 超限文案，以及把上限提高到所选分组数后同一令牌恢复 204。选项测试覆盖写入 8、拒绝 0，以及分组设置白名单接受 6、拒绝 -1。

前端：

- Default：`bun run test` 覆盖 `auto-group-form.test.ts`、`api-keys-mutate-drawer.test.tsx`、`api-key-group-combobox.test.tsx`（27 项通过）。达到上限后未选中的普通分组被禁用，保存仍提交已选分组。
- Classic：`node scripts/run-compat-tests.mjs src/components/table/tokens/__tests__/token-group-limit.compat.test.jsx` 通过。达到上限后额外分组按钮禁用，已超限选择显示明确文案。

未在浏览器里登录后台点选设置页和令牌抽屉。未跑 MySQL / PostgreSQL。本次没有新增 SQL，分组数量判断在请求校验和鉴权中完成。仅本地验证，不包含合并、发布或生产部署。

### 二次审查验证结果（Windows 本地）

- Go 1.25.5 windows/386：三包定向测试通过；扩大到控制器令牌相关测试后，50 项通过，命令为 `go test ./controller -run 'Test(AddToken|UpdateToken|GetToken|TokenGroup)' -timeout 60s -count=1`。新增回归同时验证创建和编辑中的尾逗号、重复项超限拒绝，以及 alias 指向同一分组时不误伤。
- Default：原有三个令牌测试文件 27 项、新增设置保存交互 5 项通过；可视化与 JSON 模式提交字符串上限，保留稳定分组 ID/code 和当前显示名称，0、负数、小数不发送请求。TypeScript 类型检查及定向 oxlint 通过。
- Classic：设置交互 4 项及令牌选择 1 项共 5 项通过。覆盖可视化与手动编辑模式，修改 5→8、保存字符串选项、父组件传入新配置后的回显和清空值阻止提交；为可视化数字输入补齐无障碍名称。使用 Classic 自身 ESLint 配置检查相关文件通过。测试环境存在 Semi UI 的 React 弃用警告，不影响断言结果。
- 本机未找到 Bun 可执行文件，前端使用现有依赖的 Node CLI 运行，没有安装或修改依赖。组件测试仅替代网络边界，未替代真实组件；不等于真实浏览器与后端联调或生产验收。
- 共享契约变化为：超限创建/编辑返回明确业务错误，历史显式超限令牌继续返回 403 但正文改为超限说明；密钥格式、模型 ID、分组提交标识与计费不变。
- 分享评估：`NewAPIForDouDi`、`NewAPIModifyByGang` 均适合在确认存在相同多分组上限及保存/鉴权不一致后参考此修复；依赖各自分组归一化与配置接口，不能直接照搬。未向兄弟仓库发送通知。
