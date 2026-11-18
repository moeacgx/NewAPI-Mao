# Classic 令牌编辑关闭后快速重开竞态修复

## 目标

Classic 令牌编辑弹窗提交成功关闭后，在关闭动画窗口内（500ms）快速重开同一条或另一条令牌，编辑身份不能被上一次关闭遗留的回调清除；真正的新建入口仍走 POST，正常关闭、表单重置和卸载清理保持不变。不改 API 契约，不靠等待或禁用 500ms 掩盖问题。

## 问题复现（独立验收 attempt10）

令牌 `expired_time=-1`，编辑提交 PUT 成功并关闭；弹窗不可见后 500ms 内重开同一记录，设置一天并保存，请求变成 POST 并新增 id=2，原 id=1 仍为 `-1`。等待 650ms 后重开则正常。该缺陷在 money 基线中已存在，不是本轮引入。

## 根因

`web/classic/src/hooks/tokens/useTokensData.jsx` 的 `closeEdit` 先 `setShowEdit(false)`，再 `setTimeout(500)` 无条件执行 `setEditingToken({ id: undefined })`。重开入口（`TokensColumnDefs.jsx` 编辑按钮）只设置一次 `editingToken` 和 `showEdit=true`，不会取消旧定时器；旧回调触发后把 `editingToken.id` 清空，`EditTokenModal.jsx` 以 `props.editingToken.id !== undefined` 判断 `isEdit`，于是提交走新建分支 POST。

## 调用链核对

- 打开：`TokensColumnDefs.jsx` 编辑按钮、`TokensActions.jsx` 创建按钮，均直接调用 `setEditingToken` 与 `setShowEdit(true)`。
- 关闭：`EditTokenModal.jsx` 提交成功与取消、SideSheet `onCancel`，均经 `props.handleClose` 即 `closeEdit`。
- `editingToken` 只在 `components/table/tokens/index.jsx` 传给 `EditTokenModal`。

## 方案

仅修改 `web/classic/src/hooks/tokens/useTokensData.jsx`：新增 `closeEditTimerRef` 保存关闭定时器；`closeEdit` 先 `clearTimeout` 旧定时器再设置新的；`showEdit` 变为 `true` 时的 effect 取消尚未触发的定时器；卸载 effect 清理定时器。关闭动画期间保持原行为（500ms 后清空身份用于下次新建），打开入口、EditTokenModal 与 API 契约均不改。

## Default 评估

`web/src/features/keys` 的 `ApiKeysMutateDrawer` 在 `onOpenChange(false)` 后同步 `form.reset()`，`currentRow` 没有延迟清空定时器，未发现同类风险；以源码和必要定向用例记录依据，不做无意义的双模板改动。

## 测试计划

新增 `web/classic/src/components/table/tokens/__tests__/token-edit-reopen-race.compat.test.jsx`，使用真实 `useTokensData` 与真实 `EditTokenModal`，受控假定时器：

1. 同一令牌提交 PUT 成功关闭后，窗口内重开并保存，断言再次 PUT 原 id 且无 POST。
2. 关闭后窗口内改开另一条令牌，断言 PUT 的 id 为新记录，不串 ID。
3. 关闭并等待定时器后点击新建，断言 POST 且无 PUT。
4. 纯 hook 级：关闭后窗口内重开，过了原定清空时间编辑身份仍保留；真正关闭后才清空。

卸载取消定时器的用例已评估并删除：该用例在旧代码上同样通过，不保护任何可观察行为，卸载清理仅作为防御保留在实现中。

先在未修复代码上运行确认第 1、2 项失败，再修复确认通过；同时运行现有 `token-expiry-sentinel` 与 `token-group-limit` 回归，每个进程限 60 秒，并设置 `NODE_OPTIONS=--no-webstorage`。

## 结果

- 修复前 `useTokensData.jsx` SHA256 `31099da0031044b739744f762aba8cb5bbf6013fbf29875da28723f69ce24cc1`，修复后 `b41f40500e40d519bf04dfddfb8d4dd26143c4b26e423a531a754d4620f8b1e4`。
- 红：旧代码上同一令牌重开用例失败（弹窗退回新建态找不到原令牌表单值）、另一令牌用例失败（找不到另一令牌表单值）、hook 级用例失败（`expected undefined to be 1`）；正常新建用例在旧代码上通过作为基线。
- 绿：修复后新测试文件 4 用例通过；`token-expiry-sentinel` 3 例、`token-group-limit` 3 例回归通过，永不过期修复保留。均为 `NODE_OPTIONS=--no-webstorage`，每进程 60 秒上限。
- 新测试文件整文件首次运行曾因 60 秒超时被杀（exit 124），之后按用例拆分定位，最终整文件单进程通过。
- Default：`ApiKeysMutateDrawer` 关闭时同步 `form.reset()`，`currentRow` 无延迟清空定时器，未发现同类风险，未改动 `web/src`。
- prettier 检查与 `git diff --check` 通过。
- 未覆盖：真实浏览器、完整构建与部署；旧 r2 产物不含此修复。
