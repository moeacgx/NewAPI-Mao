# Issue #403：画布默认分组与安全审计名称

## 目标与范围

本工作项负责 Default 画布默认分组选项、事件分组摘要，以及 Classic 画布默认分组选项、内置策略分组选择、自动禁用白名单、请求归档范围与审计策略范围。初始六个源码文件已由协调者追加授权至下列十个文件；不修改共享翻译、其他工人组件及生产环境。

历史记录：2026-11-16 早些时候收到新的 AGENTS 与协调者消息 `msg_736abfb3dd2c`，前端实现改由 Claude Code 接续；当时 Codex 工人 D 按未完成交接停止写入并释放下列前端文件写权，不回滚已有成果。

当前状态：Claude 接续工人已完成实现与验证（见「已执行验证」），实现完成，待另一 Codex 工人（`ctx_b84dc7b11a8a`）对全部 D 范围做独立审查。

## 契约与方案

- 有名称的用户标签只展示名称，不拼接 code 或 ID；缺失名称时保留原标识作为降级值。
- 画布选项继续以 code 保存 `global.canvas_default_group`；可选 auto 置顶、停用组过滤规则保持。
- 内置策略与归档范围继续提交 code 数组；通用审计策略继续提交 ID 数组。
- 事件格式化只生成标签，不改写传入事件对象、原始请求/响应或审计快照；独立机器字段继续保留。
- 当前组选项由既有批量接口提供 name。与后端工人 G 确认：事件新增可选 `group_display_name`、`channel_groups[].display_name` 和 `token_groups[].display_name`；两套前端优先使用自有非空字符串展示投影，回退历史名称、code 或 ID。历史 `group_name`、嵌套 `name` 和原始证据不改写。
- Classic 已选历史 alias 从 `/api/group/details` 顶层 `group_names` 读取当前名称，不新增可选项，不改变已选 alias 提交值。只有自有非空字符串映射可使用；未知/删除项保留原标识。
- `getSecurityAuditGroups` 与新增 `getSecurityAuditGroupNames` 复用现有 API 实例在途去重，不持久缓存；登录与会话刷新通过 `updateAPI()` 创建新实例、建立独立在途表。真实 Axios adapter 测试已验证并发读取、主体切换隔离与后续改名刷新。
- 不新增接口、权限、存储或迁移；回滚为恢复前端标签格式，无数据回滚。

## 验证计划

先将现有画布与事件格式化测试改为名称契约，添加 Classic 真实 Semi UI 选择器交互回归；使用 name 不等于 code、名称刷新、空名称、未知/删除标识夹具，验证选择及提交仍是 code/ID。每次进程 60 秒上限，TEMP/TMP 位于 D 盘、限制 Node 内存与单 worker。运行受影响测试、定向 lint/格式检查、Default 类型检查及差异检查，不执行全量构建。

## 已发现边界

- 后端展示投影由 G（`ctx_11e38c7f7548`，任务 `task_ddd58df85a40`）负责，前端已按批准字段接入；后端真实数据库集成测试不属于本工人的结果。
- Classic `getGroupDisplayName` 已追加协调授权并修复原型属性误读；合法自有 `__proto__` 名称可用，`constructor`、`toString`、非字符串、空串和未知值回退 code。
- 开发索引链接由协调者统一补充，本工人禁止修改共享 README。
- 索引指向的 `docs/developer/prompt-security-audit.md` 当前不存在；已读取画布默认分组、安全审计范围全选和运行时加载专题记录。

## 精确文件清单与写权交接

以下路径均相对于仓库根目录。

实现文件（已释放写权）：

- `web/src/features/system-settings/models/canvas-default-group-options.ts`
- `web/src/features/security-audit/event-routing-display.ts`
- `web/src/features/security-audit/types.ts`
- `web/classic/src/helpers/groupDetails.js`
- `web/classic/src/pages/SecurityAudit/BuiltinPolicyTab.jsx`
- `web/classic/src/pages/SecurityAudit/RequestArchiveTab.jsx`
- `web/classic/src/pages/SecurityAudit/PolicyTab.jsx`
- `web/classic/src/pages/SecurityAudit/EventsTab.jsx`
- `web/classic/src/pages/SecurityAudit/event-origin.js`
- `web/classic/src/pages/SecurityAudit/api.js`
- `web/src/features/system-settings/models/global-settings-card.tsx`（Claude 接续期间曾为超出原清单的必要修复：给 Select 补 `items`，使改名后选项显示当前名称，仍提交 code；该范围外修复的事实与当时授权情况保留不改写。协调者已核对该文件当前完整 QuantumNous AGPL 许可证头在位，SHA256 为 `DFDE0E4C8CAF6F206836118CB965CF13F7E5E3B44A0E54C1EF6F351D1EE7B160`；783 个文件头恢复后独立审查已证实正文未变。）

既有测试修改（已释放写权）：

- `web/src/features/system-settings/models/__tests__/canvas-default-group-options.test.ts`
- `web/src/features/security-audit/event-routing-display.test.ts`
- `web/classic/src/helpers/groupDetails.test.mjs`

新增测试（已释放写权）：

- `web/src/features/system-settings/models/__tests__/canvas-group-selection.test.tsx`
- `web/src/features/security-audit/__tests__/group-reference-names.test.ts`
- `web/classic/src/helpers/__tests__/canvas-group-names.compat.test.jsx`
- `web/classic/src/pages/SecurityAudit/__tests__/group-labels.compat.test.jsx`
- `web/classic/src/pages/SecurityAudit/__tests__/group-names-api.compat.test.jsx`
- `web/classic/src/pages/SecurityAudit/__tests__/event-current-names.test.mjs`

文档为本文件；README 索引仍由协调者统一登记。

## 返修计划：停用当前值显示名称（task_83fa6addfeed）

- 缺陷：`global.canvas_default_group` 已保存的值是停用分组（`status !== 1`）时，`buildCanvasDefaultGroupOptions` 只保留启用分组，传给 Base UI `Select` 的 `items` 缺少该项，`SelectValue` 回退显示原始 code。独立审查复现：`disabled-code` 对应当前名称 `停用当前名`、`status=0`，下拉框仍显示 `disabled-code`。
- 契约：停用的当前值必须显示当前名称；停用分组不得成为可新选项；保存值仍为稳定 code，权限与提交契约不变；名称只取自有的非空字符串（原型安全）；未知或已删除的值回退原 code；`auto` 与空值（`__fallback__`）行为不变。
- 方案：把“当前值显示映射”与“可选项”分离。选项构建逻辑继续只返回启用分组；当前值显示项仅在当前值不在可选项中时，从 group details 解析当前名称，并只加入 `Select` 的 `items`，不渲染为 `SelectItem`。
- 步骤：先写失败测试并复现，再做最小修复，随后补充用例：停用当前值与改名、切换到有效分组后停用分组不可再选、未知或已删除值回退、auto 与空值回退、保存 code 稳定。
- Classic：先核对 `web/classic/src/helpers/groupDetails.js` 与 `web/classic/src/helpers/__tests__/canvas-group-names.compat.test.jsx` 的真实画布入口；若存在同样缺陷，先上报精确文件，再决定是否修改。
- 验证：仅运行限定文件的 Vitest（`--maxWorkers=1`、外部 60 秒上限、`TEMP=TMP=D:/Temp`）、oxfmt、oxlint 与 `git diff --check`；不运行 typecheck 和全量构建；最终记录文件哈希与各命令退出码。

## Classic 返修计划：停用当前值显示名称（task_73844cf8bd12）

- 缺陷：独立审查在真实 `SettingGlobalModel` 表单复现：已存 `disabled-code`，groups 含 `停用当前名`（`status=0`），combobox 仍显示 code。根因同 Default：`buildCanvasDefaultGroupOptions` 只保留启用分组，`Form.Select` 的 `optionList` 缺少当前值。
- 契约：与 Default 一致。已选停用组显示已知当前名；停用组不可新选，切换到有效分组后不能再选回；保存值仍为原 code；`auto`、空值、未知或已删除值回退原 code；改名后显示新名称；名称仅取自有非空字符串（原型安全）。不放宽停用过滤，不改通用 `getGroupDisplayName` 与其他调用方的选项语义。
- 方案：保持 `optionList` 只含可选项；仅当当前值不在可选项中、且为停用分组的非空自有名称时，才在 `SettingGlobalModel.jsx` 内为该值渲染一个 `disabled` 的当前值项，使其可显示但不可新选。必要时在 `groupDetails.js` 增加画布专属辅助函数，不改动现有导出的行为。
- 步骤：先写真实表单红测并复现，再做最小修复，随后补充用例：改名、不可新选、切换后不可回选、auto/空值/未知/删除回退、原型键、保存 code 不变。
- 验证：定向 Vitest（`--maxWorkers=1`、60 秒上限、`TEMP=TMP=D:/Temp`、`--no-experimental-webstorage`）、prettier、eslint（如可用）、`git diff --check`；不全量构建或类型检查；记录哈希与退出码。

## 已执行验证

环境：Bun 1.4.4、Node v26.13.0；每个测试/类型检查通过 Python `subprocess.run(timeout=60)` 限时。进程级 `TEMP=TMP=D:/Temp`、`NODE_OPTIONS=--max-old-space-size=2048 --no-experimental-webstorage`，Vitest `--maxWorkers=1`。Node 26 原生 localStorage 会干扰 Classic jsdom，首轮测试发现后增加 `--no-experimental-webstorage`，未改系统环境。

- 初始红测：Default 名称标签用例 6 项失败，Classic 四个选择器失败；当前投影红测分别证实 Default、Classic 显示历史名称；Classic 原型键红测得到 `function Object()`，历史 alias 红测无法找到名称选项。
- Default 纯逻辑：3 文件、21 项通过（画布选项、原事件路由测试、当前名称投影测试）。日志：`D:/Temp/issue403-projection-green.log`。
- Classic 原生回归：`groupDetails.test.mjs`、既有 `src/security-audit-event-origin.test.mjs` 和新 `event-current-names.test.mjs` 共 30 项通过。日志：`D:/Temp/issue403-classic-projection-green.log`；既有源代码检查不是新增组件行为测试的替代。
- Classic 画布真实 Semi Select、原型键安全、真实 Axios adapter 并发/主体切换/刷新 3 项通过。日志：`D:/Temp/issue403-classic-alias-green.log`（同轮审计页面仍有失败，整条命令退出 1）。
- Claude 接续后的最终结果（均单进程、60 秒上限、`--maxWorkers=1` 顺序运行）：
  - Classic 审计页面 `group-labels.compat.test.jsx` 6/6 通过（20.61s）：事件列表名称投影、通用策略 ID 选择与改名、内置策略两选择器（按 `waitFor` + 同名选项取第二个选择器的选项，不依赖关闭下拉）、归档 alias 名称与保存稳定 code、归档输入名称筛选、名称读取失败降级。
  - 名称筛选用例保留区分性断言：筛选后选项文本包含当前名称且不含 `unknown`，再按 `textContent === '当前名称'` 点击，保存 payload 的 `event_group_codes` 为 `['deleted','legacy-alias','stable-code']`。早先“合成输入事件会关闭 Semi 下拉”的诊断有误，真实原因是按可访问名称正则匹配 Semi 选项失败；`RequestArchiveTab.jsx` 无缺陷。
  - Default：`canvas-default-group-options`、`event-routing-display`、`canvas-group-selection` 3 文件 15/15 通过；`group-reference-names.test.ts` 13/13 通过，新增非字符串、null、数组、空白显示字段回退及仅原型链显示字段被忽略的用例。
  - Default 真实画布组件测试：按钮定位改为真实文案 `Save Changes`；改名刷新失败的真实根因是 Base UI Select 未传 `items` 时 SelectValue 显示原始 code，已在 `web/src/features/system-settings/models/global-settings-card.tsx` 的 `<Select>` 增加 `items`（含 `__fallback__` 回退项），保存仍提交 code。
  - Classic 原生 `node --test` 30/30；Classic 画布与 Axios adapter 并发/主体切换/刷新 3/3。
  - `bun run typecheck`（`tsgo -b`）重跑退出 0。
  - 格式与检查：Default 7 个指定文件 oxfmt 无差异；Classic 12 个指定文件 `prettier --check` 通过；Default 8 个文件 oxlint 退出 0；自有 21 个文件加本文档 `git diff --check` 退出 0（仅 `groupDetails.test.mjs` 有 CRLF 提示）。
- 禁止使用 `format-with-protected-headers`，未做全局格式化或全量构建。
- 返修（task_83fa6addfeed）：停用的当前 `canvas_default_group` 显示原始 code。
  - 红测：`canvas-group-selection.test.tsx` 新增用例“停用的当前画布默认组显示当前名称，且不能被新选择”，修复前失败：期望文本 `停用当前名`，实际 `disabled-code`（退出 1）。
  - 修复：`canvas-default-group-options.ts` 新增 `buildCanvasDefaultGroupDisplayItems(options, groups, currentValue)`，仅当当前值不在可选项中且分组自有名称为非空字符串时，把 `{ value: code, label: name }` 追加到 `Select` 的 `items`；`global-settings-card.tsx` 只改 `items`，`SelectItem` 列表仍只含启用分组，停用分组不可新选，保存值仍为 code。
  - 单元用例（`canvas-default-group-options.test.ts` 共 9 项）：停用当前值显示名称且不进入可选项、改名后显示新名称、已在可选项中不重复、未知或已删除或缺名值回退原 code、空值与 `auto` 不变、原型键与非字符串名称被忽略。
  - 结果：`canvas-group-selection` 2/2、`canvas-default-group-options` 9/9、`event-routing-display` 11/11，格式化后重跑均退出 0。
  - 检查：oxfmt 首次 `--check` 退出 1，预览差异仅为新增行的换行风格，格式化后 `--check`、oxlint、`git diff --check` 均退出 0，AGPL 头未变。
  - Classic 原状：`SettingGlobalModel.jsx` 与 `groupDetails.js` 的 `buildCanvasDefaultGroupOptions` 只用启用分组生成 `optionList`，独立审查在真实表单中复现已存停用 code 显示原始 code；`web/classic/src/pages/Canvas/index.jsx` 仅把该值作回退。
- Classic 返修（task_73844cf8bd12 / ctx_9e21d72b5d47，协调者已批准追加 `SettingGlobalModel.jsx` 与同目录真实表单测试）：
  - 红测：新增 `web/classic/src/helpers/__tests__/canvas-default-group-form.compat.test.jsx`，渲染真实 `SettingGlobalModel`，修复前失败：`expected 'disabled-code' to contain '停用当前名'`（退出 1）。
  - 修复：`groupDetails.js` 新增画布专属 `buildCanvasDefaultGroupCurrentItem(groups, options, currentValue)`，仅当当前值为非空字符串、不在可选项中、且分组自有 `code` 与自有非空字符串 `name` 时返回 `{ value, label: name, disabled: true }`，否则返回 null；`SettingGlobalModel.jsx` 只把它追加到 `Form.Select` 的 `optionList`。`buildCanvasDefaultGroupOptions` 与通用 `getGroupDisplayName` 未改，停用分组仍不可新选（选项 `aria-disabled=true`），保存值仍为原 code。
  - 用例（真实表单共 10 项）：停用当前值显示名称、改名后显示新名称、停用项禁用且切换为有效值后消失，以及未知、已删除、停用无名称、原型键 `constructor`、仅继承名称、`auto`、空值的回退。
  - jsdom 限制：最小用例验证 Semi `Form.Select` 在 jsdom 中点选选项不触发 `onChange`（普通 `Select` 正常，保存值为空时同样失败），与本修复无关。故“切换有效分组”一步用 `rerender` 把保存值设为有效 code 验证，未放宽任何断言。
  - 页眉：`.eslintrc.cjs` 的 `header/header` 要求 `Copyright (C) 2025 QuantumNous`；两个新增的 Classic 测试文件原用 2023-2026，导致 eslint 报错，已仅将年份行改为 2025（该失败由新增文件引起，并非既有配置问题），未使用 `--fix`。
  - 结果：两个 Classic 测试文件共 12/12 通过，eslint 无问题，prettier `--check` 通过，`git diff --check` 退出 0；Default 冻结文件哈希未变。
- Classic 空值标签空白返修（task_d8fcd2a080c1 / ctx_27922e830b0e，来自独审 ctx_b84dc7b11a8a 的 P2）：
  - 缺陷：在真实 Chrome 中先选有效分组、再选回“跟随系统默认”（值为空字符串）时，Classic combobox 标签空白。根因：Semi `Form` 的 `withField`（`web/classic/node_modules/@douyinfe/semi-ui/lib/es/form/hoc/withField.js`，约 285-301 行）在未设置 `allowEmptyString` 或 `allowEmpty` 时，会把等于 `emptyValue`（无默认值）的值转成 `undefined`，受控 Select 因而显示占位。Select 本身发出的值是完整的空字符串。
  - 修复：只在 `web/classic/src/pages/Setting/Model/SettingGlobalModel.jsx` 的画布默认分组 `Form.Select` 上加 `allowEmptyString`，未改动 `node_modules`。保存仍为空字符串，没有哨兵值进入 API，停用分组不可新选的规则不变。
  - 真实 Chrome 红绿（证据目录 `D:/Temp/issue403-classic-empty-value`，`probe-blank.py`，Python Playwright 加本机 Chrome）：去掉该属性后选回系统默认，标签文本为空（`after_empty_text` 为空）；加回属性后为 `跟随系统默认`，探针无错误，各步退出码 0。去属性前后的源码 SHA 已核对一致，修复文件已恢复。
  - 回归测试：`web/classic/src/helpers/__tests__/canvas-default-group-form.compat.test.jsx` 新增属性契约用例，通过 `?raw` 读取源码，断言该字段带 `allowEmptyString`，并保留空值选项。有属性时 11/11 通过（退出 0）；去属性时该用例失败（退出 1），报 `expected '<Form.Select ... label…' to match /\ballowEmptyString\b/`。第一版用例用 `readFileSync(new URL(..., import.meta.url))`，在有无属性时都以同样错误失败，已判定为用例本身的缺陷并改写，不计入红绿证据。
  - 边界：jsdom 无法驱动 Semi 下拉产生 onChange，该用例只锁定属性契约，行为证据以真实 Chrome 为准。本次探针的 `requests` 为空，没有捕获 PUT；空字符串 PUT 的完整数组与父级 refresh 计数由独审在其独占 browser-final 中验证，本工人未改动其夹具和历史证据。
  - 校验：eslint 退出 0，prettier `--check` 退出 0，`git diff --check` 退出 0（均只针对这两个文件，未用 `--fix`）。`SettingGlobalModel.jsx` 的 SHA256 为 `cf764b272d32b6a26279d8364911a9d503fe1e520a568a184ddd288f136ae952`，测试文件为 `e2b594aa1916980b668ae5239df1aa042d7308cc118b018627755e46566eb5dc`。未提交、未推送、未部署，Default 冻结文件未触碰。

## Claude 接续事项处理结果

已完成：

1. Default `canvas-group-selection.test.tsx`：保存按钮定位改为真实文案 `Save Changes`，已补许可证头并格式化；改名刷新断言改为 `waitFor` 等待名称出现。真实根因是 Base UI Select 未传 `items` 时 SelectValue 显示原始 code，已在 `global-settings-card.tsx` 补 `items`。1/1 通过。
2. Classic 内置策略两选择器测试：不再等待下拉关闭，改用 `waitFor` 加同名选项取第二个选择器的选项并断言数量大于 1；Classic Semi 下拉在 jsdom 中无法通过 Escape 或 body 点击关闭，匹配选项应使用 `textContent`。
3. Classic 归档用例已补输入名称筛选（区分性断言：当前名称在、`unknown` 不在）和 `/api/group/details` 读取失败降级（显示 `Unavailable group: legacy-alias`，保存不丢 code），6/6 通过。
4. Default 已补非字符串、null、数组、空白及仅原型链展示字段的回退用例。
5. 最终定向 lint、格式检查、差异检查与 typecheck 已执行，结果见上节。

仍待处理（不在本工人范围）：

- 本轮前端工人未执行任何后端、数据库或生产验证；前端测试使用 mock 投影，不能替代后端真实验证。项目已有的 G 后端证据已另行独立完成：G 后端经独立 SQLite 定向验证，见 `D:/Temp/issue403-archive-review/report.md` 与 `docs/workflows/2026-11/16_issue403_archive_group_names.md`。线上或生产验证仍未做，不属于本工人范围。
- `global-settings-card.tsx` 的许可证头曾在本工人编辑前缺失，那是 783 个文件头事故中的历史问题；现已由协调者核实头部完整在位，无待恢复事项，不得重复补头。
- 由协调者安排另一工人独立审查；禁止提交、推送、全量构建和生产部署。

建议复用的测试命令：

```text
# web/，分别运行或一次限定目录，仍设置外部 60 秒上限
bun run test src/features/system-settings/models/__tests__/canvas-group-selection.test.tsx --maxWorkers=1
bun run test src/features/system-settings/models/__tests__/canvas-default-group-options.test.ts src/features/security-audit/event-routing-display.test.ts src/features/security-audit/__tests__/group-reference-names.test.ts --maxWorkers=1
bun run typecheck

# web/classic/
bun x vitest run src/pages/SecurityAudit/__tests__/group-labels.compat.test.jsx src/pages/SecurityAudit/__tests__/group-names-api.compat.test.jsx src/helpers/__tests__/canvas-group-names.compat.test.jsx --maxWorkers=1
node --test src/pages/SecurityAudit/__tests__/event-current-names.test.mjs src/security-audit-event-origin.test.mjs src/helpers/groupDetails.test.mjs
```

## 进程与权限状态

Claude 接续工人（task_b70dc131c783 / ctx_187258dd376a）未启动任何服务器或后台常驻进程；所有验证进程均单进程顺序运行、60 秒上限并已返回结果。期间曾遇协调者全局写入冻结（`msg_a50cb2e9487d`）并在解除（`msg_1ebe0119ca45`）后恢复；全程未使用 `format-with-protected-headers`，未做全局格式化或全量构建。未触碰其他工人的进程与文件，没有提交、推送、部署或生产写入。写权仅限本文档清单内的实现与测试文件；`global-settings-card.tsx` 为协调授权外的一处必要修复（补 Select `items`），已在上节说明。本工作项实现与验证已完成，等待另一 Codex 工人（`ctx_b84dc7b11a8a`）独立审查；本轮仅修订本文档过期事实，未再写源码或测试、未重复测试、构建或部署，未写共享 README 与 custom 文档（开发索引已由共享工人完成）。

## 国际化与跨项目影响

未新增用户文案和翻译键，保留现有 auto、失效分组与未知分组翻译，不修改 shared locales。网关请求、鉴权、计费、选择和筛选值不变；新增事件展示字段来自 G 的向后兼容只读投影。

- NewAPIForDouDi：若具有可改名分组和相同审计快照模型，值得分享名称投影及原型键保护思路；需先核对其接口，不能直接套用。
- NewAPIModifyByGang：若仍将 name/code 拼接展示，值得分享标签与稳定业务值分离的回归方法；其模板/后端兼容性未核验。
- 未向兄弟项目发送 Issue、PR 或消息。
