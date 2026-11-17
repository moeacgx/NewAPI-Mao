# Issue #403 外置渠道可观测性分组名称：本地修复与交接

## 当前状态与写权

2026-11-16，本工作项尚未完成，未发布、未部署。用户更新分工后，协调者要求前端实现交由 Claude Code；本工人已停止全部前端代码和测试写入，并释放写权。现有成果保留在 `D:/Temp/issue403-channel-quality`，不得把本地修补称作线上生效。

主仓的三处内嵌入口尚未复制或修改。主仓本工人仅新增本工作记录；README 索引由协调者统一登记。独立审查必须由另一名工人执行，本工人没有进行独立审查。

## 目标、基线与根因

目标：分组显示当前名称，不追加 code；筛选值、树节点身份和业务提交继续使用原有 code/ID。名称缺失、未知或删除时回退原标识。

已通过 `gh api` 核对外仓 `moeacgx/maolaonewapi-extensions` 的 main，随后独立浅克隆到专属目录，未初始化主仓子模块：

- 提交：`c5dd5b1446e091bcf475c149f8b47a6954721c9a`。
- Git tree：`eba98393007e8b31dbe7769954dd3a6a8351c74b`。
- 模块清单版本：`0.4.2`；`host` 为 `{}`，声明 `ui.native`、admin 角色。
- `app.js` 内部 UI 结构版本仍是历史 `0.3.7`，本轮没有改动任何版本。
- 外仓与主仓修改前的 `app.js` Git blob 相同：`3e7defd427287ad96eb9754742be30f761ec55df`。
- Default 原生入口 Git blob：`7153f71f103a2df263858431e85785a15390bf3d`。
- Classic 原生入口 Git blob：`b92d62ec067f5c9b5fd638c6280f08dd2a7df6b2`。

外仓原生入口为自包含压缩 ES 模块，没有对应 `public/source` 源码或构建脚本。稳定性页面在名称后再次展示 code，Classic 下拉拼接 `名称 (code)`，Classic 渠道行直接显示 `group`。静态 `app.js` 另有相同问题，但当前 `index.html` 与 `compat.html` 是“模块依赖未就绪”占位页，实际安装入口由 manifest 指向两套原生模块，不能只修 `app.js`。

最初本地计划考虑恢复可读源码重建；协调者随后允许“可重现构建或最小明确补丁过程”，实际采用后者：`apply-local-patch.py` 对固定基线进行每处唯一匹配替换，其他字节保留，另产出格式化前后差异便于审查。没有添加原生源码目录，没有重建 CSS，也没有从并发修改中的主仓复制业务源码。外仓 AGENTS 要求的完整源码重建、发布脚本及文档闭环仍需接手人处理，不能称为已发布交付。

## 已修改文件与精确摘要

以下路径均相对于专属外仓 `D:/Temp/issue403-channel-quality/repo`：

1. `modules/channel-quality/public/app.js`：稳定性去掉 code 副标签；空白名称回退；新增仅用于展示的 `groupName`，渠道行消费完整 `group_name`，原始 `group` 保留。共 4 项唯一匹配替换。
2. `modules/channel-quality/public/native/default.mjs`：矩阵不再追加 code；矩阵与筛选项空白名称回退 code。共 3 项唯一匹配替换。现有 Default 渠道表没有配置分组文字，本轮不新增该业务展示。
3. `modules/channel-quality/public/native/classic.mjs`：矩阵去掉 code 副标签、名称空白回退；筛选项只显示名称；渠道行消费完整 `group_name`。共 4 项唯一匹配替换。
4. `scripts/channel-quality-group-names.test.mjs`：新增 6 项 Node 行为回归，执行真实产物函数和 JSX 树，SDK/Hook 采用模拟夹具。当前无外部 npm 依赖，仍需格式化与主仓测试落点整理。

SHA-256 精确记录见专属目录 `patch-summary.json`：

| 文件        | 修复前 SHA-256                                                     | 修复后 SHA-256                                                     |
| ----------- | ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| app.js      | `9f5a836729be89f7bf1175b1a92f35ffe2d9a4818583fd796acab79c2d0e7e3b` | `9ada368f5dfa4a78c9c99a913d2458d550789508f5d53f88914a8205b6b60b96` |
| default.mjs | `937ae91358466984927948fe9701ad8e1198608d8760c247be945a360b9d38a3` | `2f92f6017f5404951160c1116a8c1da1b2854d9915f479a053d46d071986df3e` |
| classic.mjs | `e8c7242a409d6ea63ca40df66f6f040e0851a4c7ef3146692ce0da30b23b460d` | `3f0811d8bacfe9c991a7276af4f90e9a3d2953ca302bac83e9d9a8e05c439e74` |

审查产物：`external-tracked.diff`、`external-test.diff`、`default-readable.diff`、`classic-readable.diff`。`PLAN.md` 是实施前记录；`apply-local-patch.py` 是本轮最小补丁过程，要求干净基线，每处旧文本只匹配一次，重复运行会拒绝。不要直接在已修补树上再次运行。该脚本本轮没有写入主仓或外仓提交，若要进入长期维护应由接手人决定合适落点。

## 宿主接口对齐

已与宿主负责人 dispatch `ctx_252bb29925b1` 直接确认：

- `GET /api/channel-analytics/channels` 的 `items[].group_name` 为 `FormatGroupDisplayNames` 返回的完整展示字符串，按原顺序以逗号加空格连接，例如 `当前企业分组, unknown`；`group` 仍为 `legacy-enterprise,unknown`。
- `GET /api/channel-analytics/stability` 的 `items[].group_name`，以及 `GET /api/channel-analytics/filters` 的 `groups[].name`，批量复用 `GetGroupDisplayNameMap` 解析当前 code 与历史 alias。
- 删除或未知标识回退原值；不将全局映射的额外分组加入原筛选集合。名称查询失败单独记日志并回退已有配置名称或 code，不吞主业务错误。
- 所有相关路由继续要求 `AdminAuth + ChannelRead`，前端不新增分组读取接口，不逐行请求，不构建容易受 `__proto__` 键影响的对象映射。
- 宿主实现和 Go 测试由该负责人负责，本记录不代替其最终验证结论。

## 已执行验证及限制

测试由专属 `run60.py` 启动，单个子进程最多 60 秒，`TEMP/TMP/TMPDIR` 均指向 `D:/Temp/issue403-channel-quality/tmp`。超时才终止该测试进程树，未终止共享服务。

```powershell
python D:/Temp/issue403-channel-quality/run60.py node D:/Temp/issue403-channel-quality/repo/scripts/channel-quality-group-names.test.mjs D:/Temp/issue403-channel-quality/repo/modules/channel-quality/public
```

- 红阶段：Default 矩阵、Classic 矩阵、Classic 渠道和 Classic 下拉均因原 code 仍显示而失败；Default 查询保持 code 的断言原先就通过。静态夹具初次另缺 `window.addEventListener`，修正该测试环境后再执行。
- 绿阶段：6/6 通过，覆盖当前名称、改名、空串/空白回退、未知多组、原型名称 code、静态 HTML 转义、Default 查询仍提交旧 code。测试实际执行两个原生产物的组件逻辑与静态页逻辑，不仅做源码字符串搜索。
- Classic 浏览器：真实本地 Classic 构建宿主加载本轮原生入口和 CSS，所有 API 均由 Playwright 模拟；1440px 桌面矩阵与渠道页面通过、无 pageerror、无已解析 `legacy-enterprise` 泄露。证据为 `classic-browser-mock.json`、`classic-stability-mock.png`、`classic-channels-mock.png`。
- Default 浏览器：未通过。共享 `127.0.0.1:5174` 开发服务在加载页面前报 SWC 语法错误，定位当时的 `web/src/features/keys/components/api-keys-mutate-drawer.tsx:287`：`Expected ':', got 'const'`。扩展接口尚未发出，不能算扩展失败或成功；该源文件不在本工人范围。证据为 `default-browser-mock.json` 与 `default-failed-mock.png`，需相关实现恢复后重新验证。
- 浏览器尚未验证窄屏、真实宿主授权/资源 Cookie、真实后端数据库、安装升级及逐节点刷新。没有生产证据。
- `git diff --check` 尚未通过：两份原生压缩文件保留了基线已有 CRLF；Git 对变更后的整行报告 trailing whitespace。已核对 HEAD 和工作区均为单行 CRLF；接手人需按仓库约定处理换行后重新生成 SHA 和验证。本工人收到停止写入后未自行修正。
- 未运行 `scripts/publish.py`：当前 manifest 仍是已发布 `0.4.2`，脚本会拒绝覆盖已发布的同版本包，且本轮没有发布授权。
- 未运行 `scripts/validate.py` 或完整构建；现有目录校验仅验证已发布 ZIP，并不能证明本轮未打包修改已入包。接手人应完成格式化、语法、发布准备和独立审查。

## 进程归属与未完成交接

- 自有 Node 单元测试已正常退出；两次 Python 浏览器验证已退出。
- 自有 Classic 测试 HTTP 服务在浏览器脚本内启动，端口 5227，已执行 shutdown；不需其他工人清理。
- Default 端口 5174 为既有共享服务，本工人只读取，未启动、停止或修改其配置。
- 没有后台 worker，没有子智能体，没有待收结果的测试进程。

接手顺序：

1. 确认当前前端写权归属；本工人已经释放。修复逻辑在外仓临时检出，主仓三入口仍是以上旧 blob。
2. 审查最小差异和上述宿主契约；处理原生 CRLF 检查问题、外仓 README 与维护文档、测试可读性和复用落点。
3. 在协调者授权的主仓 `extension/builtin/channel-quality/public/app.js`、`public/native/default.mjs`、`public/native/classic.mjs` 同步已审逻辑，新增专属回归；不要改 manifest、CSS 或安装逻辑，除非另获授权。
4. 复跑三入口单元测试、两套真实宿主浏览器模拟接口验证，补筛选/展开仍用 code 的浏览器请求断言及窄屏检查；不要将已有 Classic 截图扩张为全套 E2E 已过。
5. 协调者更新主仓开发索引/扩展专题，并安排另一名工人独立审查；所有工人和独审完成后才能进入部署门禁。

## 后续 zzapi 安装升级方案（未执行）

`extension/builtin.go` 的 `shouldInstallBuiltinModule` 会在已安装模块版本与内嵌版本不同、或资源内容不一致时恢复内嵌资源。因此只上传外置新包，下一次旧宿主启动仍可能被覆盖；版本号高于内嵌版本也不能阻止此覆盖。

当前协调者选择将同源三入口同步到主仓候选，通过已授权的宿主部署使修复持久化：

1. 将宿主 `group_name` 契约和内嵌三入口放入同一已审候选，完整构建两套前端及宿主；版本和资源 hash 以最终候选重新登记。
2. 部署前经 CloudSSH 核对 zzapi 节点、现有模块实际版本/目录/启用状态，备份可回滚镜像、模块目录及状态文件。此阶段未执行、未读取任何密钥。
3. 串行更新节点，确认启动生成的内嵌资源为最终修复内容；如果采用单独上传包流程，必须同步保证所有后续启动节点的内嵌资源一致。
4. 使用 Root 逐节点调用 `POST /api/extension-admin/refresh`，通过各节点 `GET /api/extension-admin/?all=true` 检查启用状态、清单错误和资源修订号；不能仅刷新负载均衡入口一次。
5. 使用真实后台会话加载两套 native v1 入口，核对资源 Cookie、最终资源 hash、name!=code 的矩阵与渠道展示，以及筛选仍发原 code。管理 PAT 不代替资源 Cookie 验收。
6. 如失败，恢复已备份的宿主镜像和同源模块/状态，逐节点刷新与核验；仅回滚外置资源而保留会恢复另一版本的宿主并不可靠。

外仓独立发布是另一项未授权操作：选择未发布的新版本，更新 manifest 与 `release-notes.json`、模块 README，按维护规则重建原生入口，再执行 `python scripts/publish.py` 和 `python scripts/validate.py <旧提交>`。已发布 `published/<id>/<version>/` 不能覆盖。包为 ZIP + SHA-256 完整性校验，仓库流程没有发布者签名，不能称为签名包。

在线安装通过 Root `GET /api/extension-admin/marketplace` 获取固定目录，浏览器下载后 POST multipart `/api/extension-admin/upload`，文件字段 `file`，在线元数据为 `archiveSha256/expectedId/expectedVersion/catalogUrl/archivePath`，五项完整提交。首次安装关闭，升级保留启用状态。`host.min/max` 不构成实际安装门禁，应以 SDK、API 和权限能力实测为准。相关契约见 [扩展模块开发](../../developer/extensions.md)。

## 共享契约与兄弟项目建议

已确认影响仅为渠道统计的附加展示字段及分组标签，code/ID、计费、认证与筛选契约保留。没有向兄弟项目创建 Issue、PR 或消息。

- `NewAPIForDouDi`：若采用相同 channel-quality 原生扩展和宿主统计接口，可分享名称/稳定标识分离的修复与用例；需先确认其 SDK 和内嵌覆盖策略，不能直接替换压缩包。
- `NewAPIModifyByGang`：同样仅在存在该统计契约和双模板入口时值得分享；否则只分享规则与测试思路，不建议搬运本仓产物。

## Claude Code 接手记录（2026-11-16，task_56f03bd86946）

本节为追加内容，原作者文字未改动；其中与前文冲突的「内嵌入口尚未复制」「Classic 筛选用例仍失败」两处以本节为准。

### Classic 筛选 JSDOM 用例（已绿）

- 文件：`web/classic/src/pages/ChannelObservability/__tests__/group-names.compat.test.jsx` 筛选用例。
- 根因：生产代码正确（utils.js 提交 groups=filters.group）。受控 Semi Select 把 onChange 延后到 Popover afterClose，而 afterClose 依赖 DOM animationend，JSDOM 不会触发。
- 修复：仅改测试，点击选项后在 waitFor 内对选项节点触发 fireEvent.animationEnd；生产属性未改。
- 结果：vitest 4/4 通过，日志 D:/Temp/issue403-observability-claude/classic-filter3.log；尾部 Collapsible/Skeleton/Card 为非致命 React 警告。

### 主仓内嵌 channel-quality 三入口（同源补丁）

- 目录：extension/builtin/channel-quality/public；补丁前与 HEAD blob 字节一致（LF）。
- 补丁前 SHA-256：app.js 9F5A8367…0E7E3B；native/default.mjs 163C8867…DA8133；native/classic.mjs 2ACC5E48…C54C7F。
- 补丁后 SHA-256：app.js 9ADA368F5DFA4A78C9C99A913D2458D550789508F5D53F88914A8205B6B60B96（与外置完全相同）；native/default.mjs 34AD3DA40227B4F6D8B8F2781DB8B916DEC1B56311BA96594B98A37E7F77F416；native/classic.mjs 8854BBD80324E5584807C692F15CD2C61F65A9E1D8315081F339C88A02948005。
- 两个 mjs 与外置版仅末行换行不同（外置 CRLF，内嵌 LF），源码相同。git diff --stat：3 文件，6 增 5 删。

### 主仓专属测试与红绿边界

- 新增 extension/channel_quality_group_names.test.mjs（SHA-256 9D8292FBFBF1DBEA18297ADB794E4B83E469ECA3E962E03CD92FC6D7500208F0），放在 builtin 之外，因为 extension/builtin.go 使用 `//go:embed builtin/*`。与外置测试的唯一差异是默认 root 相对本文件定位。
- 绿：对补丁后内嵌入口 6/6 通过；对外置入口复验同样 6/6 通过，三入口哈希与记录一致。
- 红：对 HEAD 入口 5 失败 1 通过；唯一通过的是 Default 查询继续提交历史 code，旧代码本就提交原始 code，符合预期。

### 仍未完成

- Default 浏览器验收：被非本任务文件 web/src/features/keys/components/api-keys-mutate-drawer.tsx 的编译错误阻断，已交 ctx_16f4b6439284。
- 窄屏验收、外置仓库安装/升级与发布校验未做；按约束不推送外仓、不改 manifest 版本与安装逻辑。
- 未运行全量构建、typecheck、lint、format；git diff --check 在 CRLF 单行压缩文件上有既有告警。
- 协调已通知停止前端写入（版权头恢复期间），解除前不再写前端文件；独立审查由另一名工人执行。

## Claude Code 路由 P2 最小撤销后的内嵌三入口复验（2026-11-16，task_533834b15e00）

本节为追加内容，原文未改动。

- 新增的 `/channel-observability` 路由文件已撤销（不存在、从未被跟踪），`routeTree.gen.ts` 无该登记，其余 affiliate-management 与 mcp-management 并发增量保留；沿用原 extensions 入口的角色与模块开关，未加第二入口。详见 `16_issue403_observability_group_names.md` 同名追加节。
- 内嵌三入口行为测试 `node --test extension/channel_quality_group_names.test.mjs`：6/6，退出码 0；TEMP/TMP 指向 D:/Temp/issue403-e-route-fix-claude，单进程、60 秒上限、Node --no-experimental-webstorage；日志 embedded.log。
- 三入口与测试文件哈希未变，与前文记录一致：app.js 9ada368f…b60b96；native/default.mjs 34ad3da4…77f416；native/classic.mjs 8854bbd8…948005；测试 9d8292fb…0208f0。
- Default 8/8（含原扩展路由 role10 拒绝）与 Classic 4/4 名称刷新仍提交原 code，均退出码 0，见对应专题。
- 仍未验证：Go 安装升级实测（上一轮 60 秒超时，退出码 124）、桌面与 390 宽度浏览器验收、外置仓库发布。不宣称整体通过，等待另一名 Codex 独立审查。

## Claude Code 本地验收 v5 摘要（2026-11-17，task_4c4ceb754327 / ctx_aa2b0cf7cc9b）

与上文历史分开；完整方法、矩阵与哈希见 `16_issue403_observability_group_names.md` 同名章节，证据目录 `D:/Temp/issue403-e-acceptance-v5-ctx_aa2b0cf7cc9b`。HEAD `13f5fdbb93f88da11a157efa69094a9a5e472b60`。

- 真实 Default 与 Classic 宿主加载主仓内嵌 native 资源，1440 与 390 宽各自实测：启用选择、改名刷新、当前组多名称、缺失名称回退原 code、模块禁用，各格 exit=0、unmatched=[]。已知名称下拉不露 code，请求始终提交原 code（`groups=legacy-enterprise` / `legacy-team-b` / `deleted-code`）。
- 禁用模块：Default 显示未找到扩展页面且无统计请求；Classic 因 `index.jsx:512-590` 无 enabled 判断，仅靠后端 403（`extension/native.go:107`），页面显示加载失败，entry 与统计均未请求。
- Go 安装升级三例逐例退出 0（Test1 首跑 124 为冷编译，预热复跑 0；Test2 28s；Test3 4s），通过来自两批命令。
- 六个产品源码文件前后 SHA-256 一致；仅追加本两份专题，未覆盖旧历史。
- 未验证：真机、真实后端、生产、外置仓库发布；mock analytics 为空数据；4 条 Classic「聊天数据解析失败」未归因。
- 兄弟项目分享建议：本次为验证，不新增产品功能，无需向 NewAPIForDouDi / NewAPIModifyByGang 推荐。Classic 禁用门禁依赖后端的差异可作为后续可选加固点，仅供参考。

## Claude Code 覆盖补验 v7 摘要（2026-11-17，task_b6dde86a9901 / ctx_6fa6cd30a821）

与上文历史分开；完整方法与矩阵见 `16_issue403_observability_group_names.md` 同名章节，证据目录 `D:/Temp/issue403-e-acceptance-v7-claude`。HEAD `13f5fdbb93f88da11a157efa69094a9a5e472b60`。

- v5 矩阵仅覆盖启用后的筛选、改名刷新、缺名回退与模块禁用子集，且 32 份网络记录不含 `/stability`、`/channels`。v7 新增格补足该缺口：同一条记录 `group=legacy-enterprise,legacy-team`，`group_name` 为当前显示名，在真实入口点击现有 stability 与 channels 视图，捕获 `/api/channel-analytics/stability` 与 `/channels` 请求，channels 返回非空渠道 `mock-channel #1`。
- Default/Classic × 1440/390 共 12 个有效成功格（base 4、rename 4、missing 4）；15 份证据另含 2 份旧 rename 失败与 1 份点击负控，均保留。有效格均 exit 0、overflow 0、unmatched []、pageerror 0（仍有既有 React warning，非零 console）。多组名称在同一行完整显示且不附 code；改名刷新后显示新名称；缺名回退为原 code。v7 无 groups 查询，筛选提交原 code 引用 v5 已独审证据，本轮未重跑，response 的 group 为 mock 输入，不作提交证据。
- Default channels 视图产品代码无分组文字，只验请求与非空渠道行，未新增 UI；Classic channels 行显示「配置分组 + 名称」。
- P3：`run_v7.py` 在点击失败、断言失败、unmatched、pageerror 时 exit 非零；隔离负控 `v7-negctl-click-fail` exit 1。保留 `v7-default-1440-rename`、`v7-classic-1440-rename` 两次 exit 1（脚本缺陷，已分阶段断言，`-r2` 重跑通过）。
- mock `/api/status` 补 `chats:[]`，Classic 4 条聊天解析错误消失，其余为既有 React 告警。
- 产品源码六文件前后 SHA-256 一致（`sha-before.txt` 与 `sha-after-src.txt`），HEAD 未变；夹具 mock_host.py fa639872…ffbd8、run_v7.py 9afaea1c…c6ea。
- 未验证：真实后端、真机触屏、生产与 zzapi 部署；未重跑 Go 三例及全量 build/typecheck/format；未提交、推送或部署；独立复审须由另一 Codex 完成。
- 兄弟项目分享建议：本次为验证补充，不新增产品功能，无需向 NewAPIForDouDi / NewAPIModifyByGang 推荐。

## Claude Code P3 返修 v8（2026-11-17，task_09df88f3e3a4 / ctx_6e432421613c）

本节追加并更正上文 v7 口径，旧证据只读不动。证据目录 `D:/Temp/issue403-e-acceptance-v8-p3`（独占）。

- 口径更正：v7 的有效成功格为 12 个（base、rename、missing 各 Default/Classic × 1440/390 四格）；v7 目录的 15 份证据包含这 12 个成功格、2 份保留的旧 rename 失败（脚本缺陷）和 1 份点击负控。v7 没有任何 groups 查询，因此本轮不验证筛选提交原 code；该项引用 v5 已独审的筛选提交证据，本轮未重跑，response 中的 group 为 mock 输入，不能作提交证据。
- console 与 pageerror：Classic 仍有既有 React warning，Default 为 0 条，pageerror 全部为 0，不能写成零 console。
- P3 返修：v7 `mock_host.py` 对未知 `/api/channel-analytics/*` 外层 `route.fulfill` 默认 200。v8 复制到新目录后改为失败体返回 404，`unmatched` 仍记录，`run_v7.py` 的 `check(not mock.unmatched, ...)` 守卫使进程退出 1。
- 定向测试（真实 `Mock.handle` 加假 route，记录 `fulfill` 实参）：`test_handler.py` exit 0，未知 `/api/does-not-exist` 与未知 `/api/channel-analytics/bogus` 均为 404 且 unmatched 非空，stability 为 200 且 2 条、channels 为 200 且 1 条。`test_guard.py` exit 0：复用 run_v7 同款守卫的子进程，未知 analytics 退出 1，未知 api 退出 1，已知 stability 对照退出 0。
- 未做浏览器负控：本轮不启动 dev server，也不把任何 probe 的 exit 0 当作浏览器负控通过。未重跑 12 格、Go、build 或 typecheck，产品源码未改。
- SHA：mock_host.py 前 fa639872…ffbd8，后 41d21e34…44466；run_v7.py 9ea71025…df604e；test_handler.py ee5c4b13…187a1。
