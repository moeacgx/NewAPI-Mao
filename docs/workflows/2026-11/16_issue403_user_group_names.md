# Issue #403 A：用户资料与日志用户详情分组名称

## 编码前计划与契约

- 根因：User 已有非持久化 `group_name` 字段，用户查询未填充；本人安全 DTO 未返回名称；Default 与 Classic 资料和日志用户详情直接显示 `group`。
- 本人 `/api/user/self` 及共用本人 DTO 的登录、刷新响应仅增加只读 `group_name`；管理员用户详情、列表和搜索填充已有字段。原 `group`、身份、权限、筛选、计费与提交契约不变。
- 有正数 `GroupId` 时批量读取稳定 ID 对应当前名称；ID 已删除或名称缺失时回退该用户原标识，不借同名 code/alias 归属其他组。仅无 ID 的历史用户调用既有 `GetGroupDisplayNameMap` / `FormatGroupDisplayNames`。
- 展示解析采用通用 GORM 查询，不变更 schema；查询失败记录服务端错误并回退原标识。本人响应只含自己的单一名称，不返回全组映射；管理员详情原角色门禁保持；不读取密码或完整管理密钥。
- Default `ProfileHeader` / `UserInfoDialog` 与 Classic `UserInfoHeader` / `UserInfoModal` 优先展示非空 `group_name`，旧响应回退原 `group`，不追加 code。无需新翻译键。
- 用户管理列表核对：Default `users-columns.tsx` 和 Classic `UsersColumnDefs.jsx` 已优先读取 `group_name`，本轮补齐后端即可。

## 验证计划

- 先增加失败回归：后端本人/管理员响应名称不同于 code、稳定 ID 与字符串冲突、改名与历史 alias、删除/未知值、敏感字段投影和原角色门禁；列表批量补齐及原筛选值不变。
- 双模板真实组件验证新字段展示、改名刷新、旧响应与空名称回退；Default 日志详情覆盖主体切换后名称更新。
- 每项测试最长 60 秒，TEMP/TMP/GOTMPDIR 指向 D 盘；前端 `NODE_OPTIONS=--no-experimental-webstorage`、单 worker。执行必要类型检查、变更文件 lint、格式化及差异检查，不运行全构建。
- README 统一由协调者集成；不修改共享解析器、模型、locales 或其他工人的文件。独立审查、提交与部署由协调者安排；本记录不代表线上验收。

## 实施与验证结果

### 实现

- `controller/user.go` 新增复用的批量展示填充方法，在 `GetAllUsers`、`SearchUsers`、`GetUser` 与 `buildSelfUserData` 四处调用。正数 ID 以一次 `IN` 查询读取名称，无 ID 才读取既有历史映射；失效 ID 不降级匹配另一个现存组。
- 本人 DTO 追加 `group_name`，未追加 `group_id`、全组名称映射、管理员备注或敏感字段。管理员原角色比较和查询的密码、管理密钥排除字段保持不变。未增加数据库 schema 或数据库专用语法。
- Default 与 Classic 的资料和日志用户详情优先显示 `group_name.trim()`，缺失或空白名称回退原标识。Classic 资料的桌面与移动区域均覆盖。
- Default 日志用户详情另有已复现的乱序请求问题：旧用户请求晚到会覆盖当前用户。通过 effect 清理标记忽略过期响应、加载时清空旧数据，并按用户 ID 限定展示；回归使用受控 Promise，未使用固定等待。
- 用户列表两套已有名称优先展示，无额外页面修改；后端列表和搜索测试验证返回名称且原 group 筛选仍有效。

### 协调者追加授权：Default 用户编辑器

- 增加独占 `web/src/features/users/api.ts`、`components/users-mutate-drawer.tsx` 与专属测试；原 `getGroups()` 被渠道模块复用，继续保留其 `/api/group/` 字符串数组契约。
- 新 `getUserGroupOptions()` 使用管理员现有 `/api/group/details`，下拉展示 `name`，值仍为 `code`。缓存按管理员用户 ID 分隔，沿用 `groups` 失效前缀；打开编辑器获取当前名称。
- 不在当前选项中的历史 alias 或删除组，保留原选中值并用用户响应的当前名称展示；不会悄悄改为名称或另一个 code。失败时禁用选择、显示错误和重试。
- Classic 用户编辑器原先已读取 `/api/group/details` 并调用 `createGroupOptions`，本项未改动。其用户资料、日志弹窗组件已单独验证；未把 Default 修改视为自动覆盖 Classic。
- 本轮无需新翻译键；复用 `Failed to load groups`、`Retry` 等已有键。

### 精确文件清单

后端与文档：

- `controller/user.go`
- `controller/user_group_names_test.go`（新增）
- `docs/workflows/2026-11/16_issue403_user_group_names.md`（新增）

Default：

- `web/src/features/profile/types.ts`
- `web/src/features/profile/components/profile-header.tsx`
- `web/src/features/profile/components/__tests__/profile-group-name.test.tsx`（新增）
- `web/src/features/usage-logs/types.ts`（仅 UserInfo 新字段）
- `web/src/features/usage-logs/components/dialogs/user-info-dialog.tsx`
- `web/src/features/usage-logs/components/dialogs/__tests__/user-group-name.test.tsx`（新增）
- `web/src/features/users/api.ts`
- `web/src/features/users/components/users-mutate-drawer.tsx`
- `web/src/features/users/components/__tests__/user-group-options.test.tsx`（新增）

Classic：

- `web/classic/src/components/settings/personal/components/UserInfoHeader.jsx`
- `web/classic/src/components/settings/personal/components/__tests__/user-group-name.compat.test.jsx`（新增）
- `web/classic/src/components/table/usage-logs/modals/UserInfoModal.jsx`
- `web/classic/src/components/table/usage-logs/modals/__tests__/user-group-name.compat.test.jsx`（新增）

### 验证证据与边界

- 工具链实际为 Go 1.25.5 windows/386、Bun 1.4.4；运行器配置 60 秒上限，TEMP/TMP/GOTMPDIR 位于 `D:/Temp/issue403-user-groups`。前端关闭实验 webstorage，并使用 `--maxWorkers=1`。最终一次前端运行达到上限，Bun 的 Vitest 子进程稍后自行退出并输出 66.06 秒报告，说明该运行器未对整个子进程树强制执行上限；后续运行器应修正此限制，本次超时不计为通过。
- 后端先用原 `controller/user.go` 的 Go overlay 验证真实失败，再用当前实现通过同一组测试。夹具显式初始化 SQLite 方言列和所需表，并跳过用户写入规范化钩子，以保留需要验证的历史关联。
- 已通过：`go test ./controller -run 'TestUserGroupNames|TestSearchUsersController|TestManageUser' -count=1 -timeout=60s`；覆盖本人和管理员响应、角色门禁、原字段、改名、alias、删除、未知值及列表搜索。首次冷编译触及 60 秒上限，重试使用缓存后通过，不将超时计为通过。
- Default：真实组件和 API adapter 边界测试；资料 4 项、日志详情 5 项、用户编辑器 5 项。最终报告为 13 通过、1 失败，失败项是编辑器改名刷新，详见下方交接。此前未包含改名/重试两项的 12 项回归全通过。Classic：真实 Semi 组件测试，两文件共 8 项通过。未执行双模板全构建或生产请求。
- 最新 `bun run typecheck`、Default 9 个变更 TS/TSX 文件 oxlint、Classic 4 个变更 JSX 文件 eslint 均通过；Go 已 gofmt，前端已限定文件 oxfmt/Prettier 并保留各模板版权头；跟踪文件 `git diff --check` 通过（仅 LF/CRLF 提示）。
- MySQL/PostgreSQL 未运行数据库实例回归；实现仅用既有 GORM 查询和既有解析函数，无 schema 变更。本人登录/刷新复用同一 DTO，本轮未单独执行真实认证登录流程。
- 浏览器视觉、全局审计其他入口、独立审查、README 索引集成及 dev 发布由协调者继续完成；本工人未提交、部署或写入生产。

### 共享契约与分享建议

- 对外仅追加本人 `group_name` 并填充管理员已有同名字段，所有原 `group` 值保持不变；没有认证密钥、模型 ID、计费、路由或配额契约变化。
- `NewAPIForDouDi`：若已有稳定分组 ID 和独立显示名，值得分享这项响应与双模板展示修复；需要先核对其用户 DTO 与权限投影，不能直接覆盖。
- `NewAPIModifyByGang`：若同样存在 code/名称分离，建议参考批量填充及乱序响应回归；若仍以 code 作为唯一组名，收益有限，应先确认需求。本轮不向兄弟仓库发 Issue、PR 或消息。

## Claude Code 接续：编辑器改名刷新根因与最终定向验收（2026-11-16）

- 根因：测试夹具问题，不是抽屉逻辑。Vitest 的 afterEach 按栈序（后注册先执行）运行，本文件的 `client.clear()`、adapter 还原先于 `test-setup.ts` 的 `cleanup()`，上一个用例的抽屉仍挂载时缓存被清空，残留 scroll lock、portal 与 inert 节点使下一用例的 combobox 一直 disabled。证据：单独运行改名用例通过；整文件默认序 4/5；`--sequence.hooks=list` 5/5。
- 修复：`web/src/features/users/components/__tests__/user-group-options.test.tsx` 从 `@testing-library/react` 引入 `cleanup`，`afterEach` 首行先 `cleanup()` 再 `client.clear()`。未修改任何断言，也未改动 `users-mutate-drawer.tsx`、`users/api.ts`；`getGroups()` 保持 `/api/group/` 契约。
- 本轮真实结果（默认钩子顺序，单 worker，60 秒上限，TEMP/TMP 在 `D:/Temp/issue403-ax`，`NODE_OPTIONS=--no-experimental-webstorage`）：Default 编辑器 5/5、资料 4/4、日志用户详情 5/5，共 14/14；Classic 资料 compat 4/4、日志用户详情 compat 4/4。
- 检查：Default 13 个 A+F 文件 oxlint 退出 0；Classic 三个源文件 ESLint 无输出；`git diff --check` 退出 0，仅有 CRLF 将被替换为 LF 的提示。
- 换行符说明：A 范围内 9 个 Default 文件工作区为 CRLF，而 `.gitattributes` 要求 ts/tsx 为 LF，因此对 CRLF 原文件直接运行 `oxfmt --check` 会全部报错。已在 LF 副本上复验，13 个文件格式均通过；未整体改写换行符。
- 本轮尚未重跑 `bun run typecheck`，不沿用此前通过记录作为最终结论；交独立审查者或协调者补跑。

## 新 AGENTS 生效后的交接（前端未完成，写权已释放）

以下为上一位工人的历史交接，其中编辑器改名刷新失败已由上节解决。

- 收到协调者 `msg_f24c8bdaa05e` 后停止所有前端代码与测试写入，不回滚现有成果，不自行启动替代工人。上述 Default 9 文件、Classic 4 文件的前端写权全部释放，后续由协调者派 Claude 接管。
- **待处理问题**：`user-group-options.test.tsx` 的“分组改名刷新选项后显示新名称且保留原code”未通过。第一次在 `invalidateQueries(['groups'])` 后仍显示旧名称；增加等待初始请求结束后，最新运行在等待 combobox 解除 disabled 时失败。尚未确认是传输请求去重、缓存时序还是组件更新问题，不得把推测当根因。
- Claude 优先核对 `web/src/features/users/components/__tests__/user-group-options.test.tsx`、`web/src/features/users/components/users-mutate-drawer.tsx`、`web/src/features/users/api.ts`。其余已实现组件仍需纳入最终 14 项回归；不要覆盖另一工人拥有的共享解析器、model、locales、layout、keys 等文件。
- 最新证据：`backend-verified.log`（后端通过）、`default-green-expanded.log`（此前 12 项通过）、`classic-green.log`（8 项通过）、`default-verified.log` 与 `default-final.log`（14 项中 1 失败）、`default-lint-final.log` / `classic-lint-final.log`（退出 0，无输出）、`typecheck-final.log`（退出 0）。文件位于 `D:/Temp/issue403-user-groups`。
- 已启动的执行会话 85873、8640、51290 均已收取并结束；通过 Win32_Process 核对，没有命令行匹配本子项测试名/证据目录的存活 Node/Bun/Python/Go 进程。未停止任何共享进程。
- 与协调者接口：README 索引统一集成，既有 `getGroups()` 字符串接口保留以兼容渠道调用者；后端当前通过版本可独立审查；最终前端测试、独立审查、双模板构建和发布仍未完成。此记录作为部分完成交接，不能用于声明整个 Issue #403 A 已验收。

## 最终定向验收（2026-11-16 15:43，task_a037cd18d196）

以下为本任务 Claude Code 接续工人的最终独立验收运行，不依赖此前任何日志文件，全部证据来自本轮实际执行。

**运行环境：** `TEMP=D:/Temp`、`TMP=D:/Temp`、`NODE_OPTIONS=--no-experimental-webstorage`、`--maxWorkers=1`、单次完整运行；vitest v4.1.10（Default）/ v2.1.9（Classic）。

**Default A 范围（14/14）：**
- `src/features/users/components/__tests__/user-group-options.test.tsx` 5/5（含改名刷新根因修复后以默认钩子顺序通过）
- `src/features/profile/components/__tests__/profile-group-name.test.tsx` 4/4
- `src/features/usage-logs/components/dialogs/__tests__/user-group-name.test.tsx` 5/5

**Classic A 范围（8/8）：**
- `src/components/settings/personal/components/__tests__/user-group-name.compat.test.jsx` 4/4
- `src/components/table/usage-logs/modals/__tests__/user-group-name.compat.test.jsx` 4/4

同批次一并运行了 F 范围全部测试文件（Default 6/6、Classic 4/4），见 `16_issue403_channel_group_names.md`。

**未执行项：** `bun run typecheck`（上轮历史记录退出 0，本轮未重跑）、浏览器与真实 API 验收、独立审查——由协调者或另一工人补充。
