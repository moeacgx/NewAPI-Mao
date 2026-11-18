# Issue 410 令牌技能弹窗：计数归零、刷新生命周期与移动端入口

## 目标

令牌页打开「技能」安装弹窗后，同一令牌的列表刷新（请求在途）以及 `prompt_count` 1→0 的快照变化都不能卸载已打开的弹窗；用户按 Escape 或点击关闭后才关闭，0→1 后徽标恢复并可重新打开。Default 手机端令牌卡片必须有可用的 Skills 入口。不改 API 契约、不改弹窗内部逻辑、不改保存与鉴权语义。

## 问题与根因

1. 计数归零（第一轮，已修复并保留）：`web/src/features/prompt-center/components/token-prompt-badge.tsx` 原先在 `prompt_count` 为 0 时直接 `return null`，徽标与同组件渲染的 `TokenPromptDialog` 一起被卸载。
2. Default 桌面刷新在途即丢弹窗：`useApiKeysColumns` 每次渲染都重建列定义，名称列的 `cell` 是内联函数。`DataTableRow` 经 `flexRender` 渲染它，新函数被当作新的组件类型，于是整个名称单元格（含徽标和弹窗）在任何重渲染时重新挂载，请求在途期间的重渲染也不例外。令牌活动轮询每秒还会重渲染整张表。
3. Default 手机无入口：`ApiKeysMobileList`（`api-keys-table.tsx`）自绘卡片，没有渲染 `TokenPromptBadge`。
4. Classic 手机刷新替换卡片：`TokensMobileList.jsx` 在 `loading || searching` 时整体换成骨架，卡片和弹窗随之卸载。Classic 桌面表格不走该分支，无此问题。

## 方案

- `api-key-name-cell.tsx`（新增）：名称单元格改为稳定的组件类型 `ApiKeyNameCell`，`api-keys-columns.tsx` 的名称列直接引用它，行以令牌 ID 为键，同一令牌刷新前后单元格身份不变。
- `api-key-activity-context.ts`（新增）：活动计数与陈旧标记经 `ApiKeyActivityContext` 传入单元格，不再进入每次重建的列定义闭包；`useApiKeysColumns` 改为只接收 `now`。
- `api-keys-table.tsx`：用该上下文包裹表格，并在手机卡片的名称区渲染 `TokenPromptBadge`，计数为 0 时不显示，容器 `empty:hidden`。
- 上下文与单元格各放独立文件，是因为 `oxlint` 的 `react(only-export-components)` 禁止在导出 Hook 的文件里同时放组件或上下文。
- `TokensMobileList.jsx`（Classic）：仅在尚无令牌的首次加载时显示骨架；同一列表刷新时保留卡片，并给列表加 `aria-busy`。初次加载骨架行为不变。
- 弹窗归属：弹窗由徽标的 `open` 状态持有，只在用户操作（Escape、关闭按钮、保存成功）时关闭。令牌从列表中被移除（行消失、单元格卸载）时弹窗随之关闭，这是有意保留的行为；计数变化、刷新在途、列定义重建都不会关闭它。

## 测试

Default（Vitest，模块专属 `__tests__/`）：

- `web/src/features/prompt-center/__tests__/token-prompts.test.tsx`：第一轮新增三个用例，未改动。弹窗打开时计数更新为 0，弹窗仍可见、徽标隐藏；归零后 Escape 关闭；0→1 后徽标恢复且弹窗节点不变。徽标不可见的断言使用 `hidden: true`，因为弹窗打开时背景带 `aria-hidden`。
- `web/src/features/keys/components/__tests__/token-skill-dialog-refresh.test.tsx`（新增）：挂载真实 `ApiKeysTable`，用 `invalidateQueries` 触发真实列表重取并暂扣响应，在请求在途、计数 1→0、Escape 关闭、0→1 重取后再次点击徽标重新打开，全过程断言同一弹窗节点。桌面（`matchMedia` 为否）与手机（`max-width: 640px`）各跑一遍，手机分支同时保护 Skills 入口。

Classic：

- `web/classic/src/components/table/tokens/__tests__/token-skill-dialog-refresh.compat.test.jsx`（新增）：首次加载且无令牌仍显示骨架；同令牌 loading、searching 及计数 1→0 期间弹窗保持同一节点且卡片与 `aria-busy` 正确；Escape 关闭后 0→1 可重新打开。Escape 按 Semi 的实现（`document` 上监听、检查 `keyCode`）派发。

## 结果

- 基线 SHA `8cbf48d9f8f6d879c8dce567b22eee3a9f3b8045`。
- 第一轮计数归零：未修复代码上定向 Vitest 退出码 1，2 项新增用例失败；修复后 17/17 通过。
- Default 红：新用例在未修复代码上两个视口均失败。桌面在刷新在途后找不到弹窗，手机找不到 Skills 按钮。
- Default 绿：修复后 `token-skill-dialog-refresh.test.tsx` 与 `token-prompts.test.tsx` 共 19/19 通过。
- Classic 红：还原原始 `TokensMobileList.jsx` 后，刷新用例失败（找不到 `article` 名称 `codexQ`），另外两个用例通过；之后已恢复修复版本并逐字节比对。
- Classic 绿：新文件 3/3 通过；`center`、`installation-ui`、`token-default-mobile` 三个邻近套件 25/25 通过。
- Default 邻近套件分批运行（各 60 秒上限）：keys 组件（token-filters）/hooks/lib 共 86/86，群组相关 42/42，prompt-center 109/109 通过。
- `bun run typecheck` 退出码 0；本次改动文件 `oxlint` 退出码 0；`format-with-protected-headers.mjs --check` 对 5 个 Default 文件退出码 0；Classic 改动文件 `prettier --check` 通过。
- `api-keys-mutate-drawer.test.tsx` 的「管理员清空一个渠道并移除另一组后提交空限定」在两次运行中各失败一次（批量与单独重跑），随后在原始文件上 8/8 通过，带修复的文件上连续两次 8/8 通过。该测试及其被测组件不引用本次改动的任何文件，判断为间歇性失败；本次未能把它归因到任何一方，也未改动该文件。
- `oxlint` 在未改动的 `constants.ts`、`data-table-bulk-actions.tsx`、`api-keys-provider.tsx`、`api-keys-multi-delete-dialog.tsx` 上有既有错误，与本次无关，未处理。
- 每个命令均设 60 秒上限；Node 需要 `--localstorage-file` 才有 `localStorage`，运行用 `NODE_OPTIONS` 指向 Temp 下的独立文件，未写入工作树。
- Classic 在该工作树的 `node_modules`（已被 gitignore）原为空，现指向既有依赖目录的联接，未安装新依赖，`git status` 无 `node_modules` 变更。

## 第三轮：关闭后的键盘焦点

独立 Chromium 复验（四个单元格的生命周期均通过）发现 P3：徽标在计数归零时被卸载、0→1 时重建，Escape 关闭弹窗后 `document.activeElement` 是 `BODY`（Default 与 Classic，1440 与 390 均如此；Classic 桌面基线本就如此；计数仍为 0 时 Escape 同样落在 `BODY`）。

### 根因

关闭时库只把焦点还给打开前的那个按钮。徽标重建后它已脱离文档；计数归零时更没有触发按钮。

### 方案

- Default：`token-prompt-dialog.tsx` 新增可选属性 `finalFocus`，原样传给 Base UI `Dialog.Popup` 的 `finalFocus`（回调返回元素、`false` 或 `null`），弹窗内容带 `data-token-prompt-dialog`。`token-prompt-badge.tsx` 提供回调，顺序如下：
  1. 焦点已落在弹窗之外的非 `body` 元素（例如另一个对话框刚打开、路由已切换）时返回 `false`，不抢焦点。
  2. 当前仍连接的徽标按钮。
  3. 计数为 0 时，点击时记录的所在行（`tr` 或手机卡片的 `[data-token-row]`）内第一个可聚焦且未禁用的控件。
  4. 以上都没有则返回 `null`，保持库默认行为。
  - `api-keys-table.tsx` 的手机卡片根节点增加 `data-token-row`，使手机端也有稳定行容器。
  - 行内查找不再排除 `aria-hidden` 祖先：弹窗打开时背景整体带 `aria-hidden`，排除会让回退永远找不到控件。
- Classic：Semi 弹窗只会把焦点还给打开前的旧按钮，没有等价的 `finalFocus`。`TokenPromptBadge.jsx` 记录触发按钮和所在行（`closest('tr, article')`），并在 `open` 由真变假后的 effect 中，仅当焦点仍在 `body`（或为空）时，聚焦当前连接的徽标，否则聚焦行内第一个可聚焦且未禁用的控件。焦点已在别处则不处理。

### 测试

- `web/src/features/prompt-center/__tests__/token-prompt-focus.test.tsx`（新增，4 项）：无刷新 Escape 回到同一徽标；打开期间 1→0→1 重建后焦点落在新徽标；计数仍为 0 时落到同行复选框且不是 `body`；焦点已在其他控件时不抢焦点。
- `web/src/features/keys/components/__tests__/token-skill-dialog-refresh.test.tsx`：计数归零 Escape 用例追加焦点不在 `body` 且位于该令牌行内的断言；新增桌面/手机各一项，真实 `ApiKeysTable` 上 1→0→1 重取后 Escape，焦点在重建的徽标上。
- `web/classic/src/components/table/tokens/__tests__/token-skill-dialog-refresh.compat.test.jsx`：新增 `describe.each`（手机卡片、桌面表格）× 4 项，覆盖无刷新、重建徽标、计数为 0 回退、不抢焦点。Escape 同样派发到 `document` 并带 `keyCode: 27`。

### 结果

- Default 红：临时去掉 `finalFocus` 后，focus 文件与 refresh 文件共 8 项中 7 项失败（仅「不抢焦点」不依赖该修复而通过，符合预期）；已逐字节恢复。
- Classic 红：临时禁用关闭后聚焦的 effect，11 项中 4 项失败（两种布局的重建徽标与计数为 0 回退）；已恢复，`git diff` 确认无残留。
- Default 绿：focus 文件 4/4、refresh 文件 4/4；prompt-center 整目录与 refresh 共 14 个文件 117/117。
- Classic 绿：该文件 11/11；`src/components/table/tokens` 11 个文件 59/59；`src/pages/PromptCenter` 9 个文件 90/90（分目录运行，合并运行会触及 60 秒上限）。
- `bun run typecheck` 退出码 0；改动的 5 个 Default 文件 `oxlint` 退出码 0（曾有 `unicorn/prefer-spread` 一处，已改为展开运算符）；`format-with-protected-headers.mjs --check` 退出码 0；Classic 改动文件 `prettier` 无变更。
- Classic 首次运行该文件出现一次 60 秒上限（退出码 124，日志仅有启动信息）；单独按布局运行和随后的整文件重跑（约 16 秒，11/11）均通过，判断为冷启动停顿，未归因到代码。

### 未覆盖

- 未在真实 Chromium 复验，由独立验收按 1440/390 的 Escape、关闭与 0→1 重开执行；未构建、提交或部署。
- 路由切换只通过「焦点已在弹窗外」这一条件间接覆盖，没有单独的路由切换用例。
- Classic 的不抢焦点判断只在焦点仍为 `body` 时才聚焦；若另一元素恰好在关闭瞬间持有焦点会被保留，这是有意选择。

## 模板范围

- Default：本次修复范围（稳定单元格、活动上下文、手机入口）。
- Classic：仅修改移动列表的首次加载判断。桌面表格走 `column.render`，行以 `record.id` 为键，既有 `TokenPromptBadge.jsx` 已有 `!count && !open` 保护，无需改动；桌面此前已通过，不属于本次回归面。

## 未覆盖

- 本工作项未在真实 Chromium 中复验，由独立验收按 1440/390、暂扣请求、Escape 与 0→1 重开执行；未做完整构建与部署。
- 未覆盖弹窗打开期间令牌切换为另一个 ID 的组合场景。
- 令牌行从列表消失导致弹窗关闭的行为由结构保证，没有单独的回归用例。
- 未运行全量 Vitest、全量 lint 与 Classic 全量 compat，仅运行受影响文件及邻近套件。
