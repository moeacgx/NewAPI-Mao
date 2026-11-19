# Issue #403 F：渠道批量编辑及日志渠道详情分组名称

## 目标、范围与方案

修复 Default 标签批量编辑把 `/api/group/` 标识当名称，以及 Default 日志渠道详情、Classic 任务日志渠道详情忽略 `group_details` 的问题。

- Default：`channels-dialogs.tsx` 挂载标签批量编辑；`usage-logs/index.tsx` 挂载渠道详情，任务日志列通过 provider 打开。
- Classic：`channels/index.jsx` 的 `EditTagModal` 已使用管理员 `/api/group/details` 与 `createGroupOptions`，保存 `group`/`group_ids`；只验证，不制造无关修改。任务日志 `index.jsx` 挂载 `ChannelInfoModal`；普通日志仅展开渠道 ID 与渠道名称，没有对应分组详情弹窗。
- 批量编辑复用现有 `getGroupDetails` 和 `['group-details']` 缓存，以当前名称生成标签，原 `/api/group/` 返回的可选标识和提交契约保持不变。
- 详情使用渠道响应中的批量分组引用；有 ID 时按 ID 解析，缺名称或已删除时回退原标识，旧响应兼容 code。后端已有 alias 解析及按稳定 ID 水合，无逐行或逐组额外请求。

## 安全、兼容与协作边界

不改权限、接口、数据库、筛选与保存值。不改共享 helpers、API/types、locale、共享索引；不提交、部署或全量构建。新增文案及文档索引交协调者统一集成。

## 验证计划

先以名称不同于 code 的夹具证明红灯；覆盖多组、稳定 ID 优先、旧标识、改名刷新、缺名称/删除回退、原标识保存及名称可访问属性。分别执行 Default Vitest 与 Classic compat，子进程超时 60 秒，TEMP/TMP 指向 D 盘且清除 NODE_OPTIONS；执行定向格式、lint、类型及 diff 检查。

## 验证结果

- 红灯：Default 两个详情名称用例失败、标签选项名称用例失败；Classic 两个详情名称用例失败，历史无详情回退通过。证据：`D:/Temp/issue403-f-default-red.log`、`issue403-f-tag-red.log`、`issue403-f-classic-red.log`。
- 最后一次完整验证：Default 5/5、Classic 4/4、Classic 既有 groupDetails 21/21；`bun run typecheck` 通过，Classic 定向 ESLint 通过。证据：`D:/Temp/issue403-f-default-test.log`、`issue403-f-classic-test.log`、`issue403-f-classic-helper.log`、`issue403-f-typecheck.log`、`issue403-f-classic-lint.log`。
- Default 定向 lint 首轮发现批量编辑原有三项错误；已修复 Array.from、无用 catch 参数和单子项 fragment，但修复后尚未重跑 lint。
- Node 26 清空 NODE_OPTIONS 仍启用原生 webstorage，导致 localStorage 未定义；设置进程级 `NODE_OPTIONS=--no-experimental-webstorage` 后正常。所有测试子进程使用 60000ms 硬超时，TEMP/TMP 为 `D:/Temp`。
- 已执行定向 oxfmt、Classic Prettier、工作记录 Prettier 与三个实现文件 `git diff --check`，当时通过；最后一轮补充尚未复验，见交接说明。

## Claude Code 接续：F 最新改动复验（2026-11-16）

- 复验对象：批量编辑对 `/api/group/details` 顶层 `group_names` 的别名映射、`7`/`legacy` 参数化保存测试、详情缺名称的可选链保护。
- 真实结果（单 worker，60 秒上限，TEMP/TMP 在 `D:/Temp/issue403-ax`，`NODE_OPTIONS=--no-experimental-webstorage`）：Default `tag-group-names.test.tsx` 2/2；Default `channel-group-names.test.tsx` 4/4；Classic `channel-group-names.compat.test.jsx` 4/4。格式修正后两个 Default 测试再次通过。
- 格式与 lint：oxfmt 在 LF 副本上仅发现三处纯换行折行差异（`tag-group-names.test.tsx`、`tag-batch-edit-dialog.tsx`、`channel-info-dialog.tsx`），已在工作区写入，未改逻辑与断言。随后 oxlint 报 `eslint(curly)`（`tag-batch-edit-dialog.tsx` 的单行 if），已加花括号；最终 13 个 A+F Default 文件 oxlint 退出 0，LF 副本 oxfmt 检查通过，Classic ESLint 无输出。
- 这三个文件为 LF，其余 A 文件多为 CRLF，未统一换行符。
- 本轮尚未重跑 `bun run typecheck`；独立审查、浏览器与真实 API 验收仍未做。上一节“待接手工作”中的复测项已完成，类型检查与独立审查除外。

## 2026-11-16 前端写权释放交接

收到协调者 `msg_483c1dee4d5b` 后停止全部前端代码、测试修改；前端写权已释放，不回滚既有成果，不自行启动工人。当前没有仍运行的本任务子进程，最后验证进程已正常结束。以下为精确改动文件，均尚未提交：

1. `web/src/features/channels/components/dialogs/tag-batch-edit-dialog.tsx`
2. `web/src/features/usage-logs/components/dialogs/channel-info-dialog.tsx`
3. `web/classic/src/components/table/task-logs/modals/ChannelInfoModal.jsx`
4. `web/src/features/channels/components/dialogs/__tests__/tag-group-names.test.tsx`（新增）
5. `web/src/features/usage-logs/components/dialogs/__tests__/channel-group-names.test.tsx`（新增）
6. `web/classic/src/components/table/task-logs/modals/__tests__/channel-group-names.compat.test.jsx`（新增）
7. 本工作记录（新增）。

待接手工作：

- 最后完整绿灯之后，补充了批量编辑对 `/api/group/details` 顶层 `group_names` 的别名映射读取，以及对应 `7`/`legacy` 参数化保存测试；补充详情缺失 name 的可选链保护。这些最新改动尚未执行测试、lint、类型、格式检查，不能沿用之前绿灯宣称最终通过。
- `group_names` 在当前 `controller/group.go` 已由 `GetGroupDisplayNameMap` 生成；本工人没有修改后端或共享 types。组件以属性存在性与类型守卫读取，后续可由协调者决定是否补充共享响应类型。现有 `data` 名称仍可独立工作。
- 标签测试网络夹具目前在全部响应附加 `group_names`，可收紧为仅详情响应；检查名称选项、选中标签、title/ARIA 与原 code 保存。详情已覆盖稳定 ID、引用乱序、多组、改名、空名称、删除回退。
- 重跑两个 Default 专属测试、Classic 专属 compat 测试（均 60 秒），定向 lint/格式、`bun run typecheck`、全部本任务文件 diff 检查；再由另一名 Codex 工人独立审查。未完成独立审查、整页浏览器、真实 API 权限与数据库验收；未进行构建部署。
- Classic 标签批量编辑仅核对源码及既有 helper 测试，未新增该完整弹窗交互验收。其普通日志没有分组详情弹窗，故未制造对应修改。
- 无新增翻译文案，无 locale 变更；需协调者把本记录登记到 `docs/developer/README.md`，本工人按范围未写共享索引。未修改 `users/api.ts`、`controller/channel.go` 或共享 helpers，无额外文件写权请求。
- 跨项目契约没有变更，界面继续提交原 code；NewAPIForDouDi 与 NewAPIModifyByGang 均建议先核对是否具有相同 group_details/group_ids 契约，再分享此显示修复，不建议直接照搬，未发送外部消息。

可复跑命令：在 `web/` 运行 Vitest 并传入第 4、5 项相对路径；在 `web/classic/` 运行 `bun run test:compat src/components/table/task-logs/modals/__tests__/channel-group-names.compat.test.jsx`。Default 检查使用 `bun x oxlint -c .oxlintrc.json` 加四个 Default 文件路径，Classic 使用本地 ESLint 加两文件路径；格式工具为本地 oxfmt / Prettier，保护原版权头。交接文档追加后未再执行格式化，交由接手者一并检查。

## 最终定向验收（2026-11-16 15:43，task_a037cd18d196）

以下为本任务 Claude Code 接续工人的最终独立验收运行，不依赖此前任何日志文件，全部证据来自本轮实际执行。

**运行环境：** `TEMP=D:/Temp`、`TMP=D:/Temp`、`NODE_OPTIONS=--no-experimental-webstorage`、`--maxWorkers=1`、单次完整运行；vitest v4.1.10（Default）/ v2.1.9（Classic）。

**Default F 范围（6/6）：**
- `src/features/channels/components/dialogs/__tests__/tag-group-names.test.tsx` 2/2
- `src/features/usage-logs/components/dialogs/__tests__/channel-group-names.test.tsx` 4/4

**Classic F 范围（4/4）：**
- `src/components/table/task-logs/modals/__tests__/channel-group-names.compat.test.jsx` 4/4

同批次一并运行了 A 范围全部测试文件（Default 14/14、Classic 8/8），见 `16_issue403_user_group_names.md`。

**未执行项：** `bun run typecheck`（上轮历史记录退出 0，本轮未重跑）、浏览器与真实 API 验收、Classic 批量标签编辑完整交互弹窗验收、独立审查——由协调者或另一工人补充。
