# #404 Default 令牌倍率保护

## 范围与根因

已通过 gh 读取 #404 完整正文，核对当前 AGENTS/web 规范及令牌相关专题。Default 的 ApiKey 类型、表单转换和编辑抽屉均未消费 `group_ratio_limits`，编辑其他字段时会省略该字段；后端 UpdateToken 直接赋值，因此可能清空已有保护。Classic 已提供逐组选项，不能以倍率 Badge 代替。

本项独占 Default `api-keys-mutate-drawer.tsx`、`lib/api-key-form.ts`、`types.ts`，新增倍率保护专用组件/helper/测试及本记录。不修改 combobox、后端、共享文档和 locale；不提交或部署。

## 已核对的契约与计划

- 后端 Token.GroupRatioLimits 是 JSON 字符串，map 的键为稳定 group code，正数为保护阈值；`GetGroupRatioLimitsMap` 对空白、无效 JSON 返回空 map，移除空键及非正数。JSON 数值无法表达 NaN/Infinity，后端未定义业务最大阈值；前端不凭空增加上限。
- Classic 输入 min=0、step=0.1、precision=4，空/0/负数/非有限均删除相应保护，保存仅保留选中的非 auto 分组。Default 明确允许空/0 清除，对负数、非有限及非法输入显示错误防止误删；不改后端执行语义，不把负值保存为正值。
- 新增表单草稿与序列化；有效既有保护必须回显并保留，非法 JSON/类型需显式提示并阻止静默保存，用户可明确清除后重新配置。加载时不通过清理 effect 删除保护，只有用户改选组时清理已移除组。
- 名称用当前选项/详情的 name；alias 通过 group_ids/group_details 与原 group 顺序解析为 canonical code，无可靠关联时保留原键供用户确认，不能按显示名猜测。Auto 与继承不提供新增逐组保护；已有异常历史保护不因编辑其他字段静默消失。
- 正常显式单/多组提供逐组输入；普通用户与管理员共用本人可用组，既有管理员限定渠道和组数上限保持不变。不读取完整密钥，不请求生产修改。
- 先新增真实抽屉提交与纯转换失败回归，再实现；覆盖修改、清除、选组移除、别名、非法值、取消重开、保存刷新及角色。运行 Classic 相关契约和既有后端定向测试；每个子进程 60 秒、D 盘 TEMP/TMP、Node 兼容参数、maxWorkers=1。
- 使用本地开发页面模拟 API 做桌面/手机深浅色保存刷新与取消验证，不将模拟 API 视为真实后端或生产验收；不重复全量构建。

## 验证与待集成

共享 README/custom-development/locale 交协调者集成；另一 worker 独立审查。2026-11-16 收到新的前端执行角色要求后，本工人停止前端写入，将最终校验与收尾交给 Claude，不能将当前状态视为最终通过。

## #403 追加范围

协调者明确追加授权 `auto-group-order-editor.tsx` 及专属测试。已确认自定义 Auto 顺序使用 code 作为可见正文与拖动/上移/下移/移除的 ARIA 标签；改为从已有批量 options 解析当前 label，仅排序、value 和回调继续用 code，缺失名称才回退标识，不增加请求。与倍率保护同批测试及交独立审查。

## 实现契约

- `GET /api/token/:id` 读取保护字符串，`POST /api/token/`、`PUT /api/token/` 发送同名 JSON 字符串；修改名称、额度等无关字段也会带回已有有效保护。普通用户提交仍移除管理员专用 `group_channel_ids`，不新增管理员接口请求。
- 新增专用草稿转换模块与有边框的逐组保护区，输入不提前舍入；空白和 0 序列化为无条目，全空发送空字符串。十进制/科学计数正有限值被接受，负数、非有限、十六进制、非数值和非法历史 JSON 阻止提交；历史非法配置必须点击明确清除后才可保存。
- 删选组立即清除该组草稿，重新加回不会复活已删保护；切换 Auto 或继承会移除原显式组保护。打开编辑器只恢复原值，不执行这种清理，因而历史 Auto/继承保护不因无关字段编辑丢失。
- 既有继承模式被旧表单的“至少一组”校验拦截，测试真实复现后移除此限制；空组选使用已有 `buildGroupSelectionPayload` 输出 `group_mode=inherit`、空 group、空 IDs。普通新建默认值从 `['']` 归一为 `[]`，避免生成错误的显式空组选，界面明确提示继承用户分组。没有改变后端路由或计费语义。
- Auto 顺序的排序键和提交列表继续使用 code；可见正文、拖动/移动/移除的 ARIA 使用 options 当前 label。全局顺序标题保留真正描述，描述缺失或等于 code 时使用 label；不再把等于 code 的默认描述作为隐藏正文，倍率保持原样。
- 组名、ID 和可用范围继续复用抽屉已有批量接口，无逐行请求或新映射表；未知/删除的历史保护键保留原标识回退，避免静默误删。后端仍按现有稳定身份绑定规则处理未解析的历史配置，不承诺此类历史条目一定生效。

## 待协调者统一集成的英文翻译键

以下 9 个键需要集成到 Default 的 en/zh/zh-TW/fr/ru/ja/vi；本工人不写共享 JSON：

- `Ratio protection`
- `Set a maximum group ratio. Empty or zero removes protection.`
- `Stored ratio protection is invalid. Clear it explicitly before saving.`
- `Ratio limit for {{name}}`
- `Enter a finite ratio of zero or greater.`
- `Select explicit groups to configure ratio protection.`
- `Clear ratio protection`
- `Check ratio protection before saving.`
- `No group selected. Inherit your user group.`

共享开发文档索引及能力总览需关联本记录，标注 Default 补齐保护读写及 #403 Auto 顺序名称规则，Classic 保持现有实现。前端校验比 Classic 严格（负值不静默清空），API 字段、执行和计费阈值语义不变。

## 2026-11-16 前端角色切换交接（未最终完成）

协调者指令 `msg_875212f0763d` 要求前端实现转交 Claude，停止新增代码/测试修改；本工人已遵守并释放以下全部前端文件写权，没有回滚成果、提交、部署或启动替代 worker。只有本工作记录继续补交接。

### 精确改动清单

1. `web/src/features/keys/types.ts`：令牌响应增加可选保护字符串，保存类型增加必带保护字符串；保留此前 #399/#400 的 status/active_only 改动。
2. `web/src/features/keys/lib/api-key-form.ts`：草稿 schema/校验/默认值/回显/提交；支持既有继承模式；默认空组归一。
3. `web/src/features/keys/lib/ratio-protection.ts`（新）：JSON 严格解析、稳定 ID 关联 alias、数值校验、序列化及显式改组选清理。
4. `web/src/features/keys/components/api-key-ratio-protection.tsx`（新）：逐组名称和输入、错误、清除操作、主题/手机容器。
5. `web/src/features/keys/components/api-keys-mutate-drawer.tsx`：挂载保护字段，选组同步清理，继承提示；保留已有渠道限定等实现。
6. `web/src/features/keys/components/auto-group-order-editor.tsx`：#403 当前名称与 ARIA、默认 code 描述回退；排序仍用 code。
7. `web/src/features/keys/lib/__tests__/ratio-protection.test.ts`（新）：23 项数据契约/输入边界测试。
8. `web/src/features/keys/components/__tests__/api-key-ratio-protection.test.tsx`（新）：8 项真实抽屉/QueryClient 测试。
9. `web/src/features/keys/components/__tests__/auto-group-display.test.tsx`（新）：2 项真实顺序编辑器名称、改名、重排稳定值、未知回退和 ARIA/title 测试。
10. `web/src/features/keys/components/__tests__/api-keys-mutate-drawer.test.tsx`：仅将 3 处旧 `Remove vip` 断言更新为当前名称 `Remove Priority models`；已提前向协调者报告此相关测试范围。
11. 本工作记录。

未修改 Classic 源码、后端、combobox、共享 locale/README/custom-development、其他 worker 证据或 MCP 代码。

### 已获得的真实证据

所有测试使用 D 盘 TEMP/TMP、Node `--no-experimental-webstorage`、maxWorkers=1；每个子进程外层 60 秒限制。证据目录为 `D:/Temp/newapi-issue404-20261116`。

- 初始真实抽屉红测：4 项因不存在倍率输入失败，16.01 秒；初始 Auto 名称红测：1 项缺少 Standard Display，9.11 秒（见本会话工具记录，未将这些初始终端输出另存文件）。
- `inherit-red.log`：已有 inherit 令牌修改名称没有发出保存，复现旧“至少一组”校验阻断；另一个失败是测试关闭按钮选中了两个同名按钮，随后改为真实第一个 Close。
- `ratio-green.log`：真实抽屉 8 项 + 纯模块 23 项全部通过，31.45 秒；包含普通/admin、创建、保留保护、清除、取消、Auto/继承、删组重选、alias、非法 JSON/数值。
- `drawer-final.log`：既有真实抽屉 8 项 + 新抽屉 8 项通过，35.40 秒。已包含移除 ByRoleOptions 非法 `exact` 的修正；测试框架字符串 name 本身即精确匹配。
- `auto-regression.log`：30 项中 29 项通过，唯一失败是旧测试要求全局顺序的合法描述继续作为 title。随后已经修复为“有真实描述保留描述，否则当前名称”，但**该最后修正尚未复跑**。Auto 表单 20 项、名称新测 2 项均在该轮通过。
- `classic.log`：Classic 既有真实令牌表单/组数限制 3 项通过，21.65 秒；覆盖不限组、稳定 IDs、编辑/创建。Classic 源码已核对保护读写、输入和选组清理契约。
- 额外创建了仓外 `classic-ratio.compat.test.jsx` 和 `classic-ratio.config.mjs`，复用 Classic 真实表单，尝试加保护原值回显/提交断言。首轮因仓外 JSX runtime 路径收集失败；修正配置后第二轮到 60 秒外层超时，**没有通过证据**，不可计入通过数。`classic-ratio.log` 留存的是首轮导入失败，第二轮超时见会话工具结果；可由接手者决定继续，未修改任何 Classic 仓库文件。
- `backend.log`：既有真实 SQLite/model 6 项通过，包耗时 0.374 秒，覆盖稳定身份、顺序/模式、倍率替换、去重、旧数据/Auto、非法 JSON 回滚及 alias。运行选择为 `TestTokenGroupBindingsPreserveOrderAndModes`、`TestMigrateTokenGroupReplacesBindingAndRatioLimit`、`TestMigrateTokenGroupDeduplicatesAndKeepsTargetOrderAndLimit`、`TestMigrateTokenGroupBackfillsLegacyTokenAndLeavesAutoUntouched`、`TestMigrateTokenGroupRejectsInvalidTargetAndRollsBackInvalidRatioJSON`、`TestMigrateTokenGroupResolvesSourceAliases`。
- `browser_check.py`、四份 `browser-{1440|390}-{light|dark}.json` 及 `ratio-*`/`auto-*` PNG：本地 Rsbuild 页面、真实 Chromium、模拟 API，四场景通过。验证回显名称/数值、负数阻止、取消恢复、0 清除、稳定 IDs/code 提交、普通用户无管理员字段、保存刷新、切换 Auto 和自定义顺序当前名称；模块无横向溢出、无 pageerror。已实际查看手机深色及桌面浅色截图。
- 浏览器边界：没有真实后端写入、计费执行或生产验收；四种状态均由模拟 API 持久化本次内存 fixture。页面共享请求取消出现过 `canceled` toast；开发专用 TanStack Devtools 曾挡住保存按钮，因此之后用真实键盘 focus + Enter 完成保存，未修改产品源码绕过。截图包含上述开发态提示，不应声称生产页面无提示。

### 当前未完成项和最后写入状态

1. 需 Claude 对最后改动重新运行定向测试和 lint，再由协调者运行统一 typecheck。收到停止指令后不再运行新的测试或修改源码。
2. 交接前 lint 曾报本项嵌套三元表达式、测试多行 if 缺花括号、测试 button 缺 type。已在停止指令前改写保护区为平行条件渲染，拆出抽屉继承说明，测试改为 mode 映射、button type，并对**该新测试文件单独**执行 oxlint --fix（退出 0）；尚未对全部改动复跑 lint。
3. 协调者报告 `api-keys-mutate-drawer.tsx:287 TS1005 / :291 TS1128` 来自上述调整的中间态。停止指令到达前最后一次 apply_patch 已修正，当前源码 280 行附近是完整 `maxAutoGroups > 0 ? ... : ...` 结束，然后独立 `const selectedGroupDescription = selectedGroups.length === 0 ? ... : groupSelectionDescription`；接手者仍须复核最终编译，不依据本说明代替检查。
4. 最后一次 oxfmt 因传入含 `..` 的 Markdown 路径被拒绝，尚未完成最终格式化；此前已格式化核心 9 个文件并恢复版权头，但后续新增测试/修正需再格式化。推荐显式绝对 Markdown 路径，或从仓库根使用正确 formatter 配置；不要运行覆盖全仓的写入命令。
5. 最终 `git diff --check`、完整范围格式检查和最新快照回归尚未完成。未跑全量 build/typecheck，符合协调者避免并行全量检查的要求。
6. 本项新增 9 个英文翻译键和共享文档索引仍需协调者集成。完成最终检查后交另一名 Codex 工人独立审查；实现人没有冒充独立审查。

### 2026-11-16 Claude 接手终验（以此节为准，覆盖上文“当前未完成项”）

接手后只改独占的 10 个文件和本记录。未提交、未推送、未部署，未改 Classic 仓库源码、后端、locales 和共享文档。每个测试进程 60 秒、D 盘 TEMP/TMP、`--no-experimental-webstorage`、maxWorkers=1，串行运行；日志在 `D:/Temp/newapi-issue404-20261116`。

- 语法与换行：`api-keys-mutate-drawer.tsx:281-293` 是两个独立的单层三元式，可解析。10 个文件原为 CRLF，与索引 `eol=lf` 不一致，已仅对这 10 个文件转为 LF。
- 格式化：转 LF 后 `oxfmt -c .oxfmtrc.json --check` 仍报 5 个文件有真实差异（如 `auto-group-order-editor.tsx:280` 的 title 三元式折行）。已备份到 `fmt-backup/`，只对这 5 个文件执行 `--write`。随后 10 个文件 `--check` 通过，QuantumNous 版权头均保留，CRLF 为 0。未运行仓库级 formatter 的写入模式。
- lint：项目使用 oxlint。对 10 个文件运行 `oxlint -c .oxlintrc.json`，退出码 0 且无输出（`lint-postfmt.log`）。先前误用 ESLint 的 exit=2 是工具用错，不计入结果，且未改动 lockfile。
- 格式化后重跑的 Default 定向测试全部为绿：Auto 名称显示 2/2（11.82s）、`ratio-protection.test.ts` 23/23（2.60s）、`api-key-ratio-protection.test.tsx` 8/8（12.20s）、既有 `api-keys-mutate-drawer.test.tsx` 8/8（12.71s，含 3 处 `Remove Priority models`）。旧的 31 组件加 16 抽屉绿不等于最终绿，以上才是最终文件的结果。Auto 对 29/30 的 title 修补由 Auto 2/2 以及既有 Auto 回归复验，lint 也已复跑。
- Classic 倍率保护专测（仓外 `classic-ratio.compat.test.jsx` 加 `classic-ratio.config.mjs`，不属于仓库）：按用例拆成三个独立进程。分组上限选择器 1/1（19.98s）、创建 1/1（26.05s）、编辑 1/1（修正后 18.04s）。每个进程耗时 18-27s，主要是 collect（导入 Classic `EditTokenModal`）。此前 60 秒超时很可能来自三个用例在同一个冷启动进程中，这是假设，当时没有日志证据。
- 编辑用例曾真实失败：`getByDisplayValue('1.75')` 找不到元素。探针 `classic-ratio-probe.log` 显示 input 值为 `1.7500` 和 `2.0000`，因为 Classic 的 `InputNumber` 设置了 `precision={4}`。倍率已正确回显，因此不是 Classic 缺陷，也不是 #404 缺陷，而是仓外测试的显示格式假设有误。只在仓外测试把期望改为 `1.7500` 和 `2.0000`（备份 `.bak-precision`）。提交断言保持原样：group `base,plus,fast,smart,media,backup`，group_ids `[11..16]`，`group_mode` 为 explicit，倍率字段为编辑时 `{"base":1.75,"plus":2}`、创建时为空。仓内未弱化任何断言。
- `git diff --check`：对已跟踪的 5 个文件，在 LF 转换后重跑，退出码 0 且无警告。

未验收边界：
1. Classic 三个用例从未在同一进程里一起通过，只有逐用例的通过证据，且位于仓库外。
2. 未做全量 build 和 typecheck，符合协调者要求，仍需协调者统一运行。
3. 抽屉测试的多组增删覆盖到 2 组场景，没有更大规模的多组场景。
4. 浏览器检查仍使用模拟 API，不是真实后端或生产验收。
5. 仍需协调者集成上文 9 个英文翻译键和共享文档索引，并由另一名工人独立审查。

### 进程与协作边界

- 所有已启动测试结果已收取；交接时进程检查未发现命令行含 issue404/classic-ratio 的 Node/Python 剩余进程。
- 本地 Rsbuild PID 52188、父进程 51484 是已有共享开发服务，本工人只复用其 `127.0.0.1:5174` 页面，未启动/结束该进程；命令行请求端口 5173，实际监听回退为 5174。不得按本工人临时进程清理。
- 其它 worker 的组名/用户/模型/订阅/可观测变更及共享翻译仍由协调者编排；本项前端写权已经释放，接手者不需要等待此 worker 再写。
- API 共享影响只增加 Default 对既有保护字符串的完整往返，不改 ID/code、权限或计费语义。NewAPIForDouDi 若有同类新模板遗漏值得分享此回归；NewAPIModifyByGang 若已有倍率保护则只建议借鉴名称/稳定身份与保留字段测试，需先核对其接口，未向任何兄弟仓库发送通知。
