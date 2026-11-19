# Issue #403 E：原生渠道可观测页面分组名称

日期：2026-11-16。范围：Default 与 Classic 原生渠道可观测组件。

参考：[分组标签契约](../../developer/group-tags.md)、[开发文档索引](../../developer/README.md)。

## 目标与契约

- 用户标签使用当前分组名称；已有名称时不追加 code 或历史 alias。
- 删除、未知或缺失名称时回退原标识；无分组时保留已有未标记状态。
- 筛选、下钻、查询键继续使用原 group code 或 ID，不将名称写入请求。
- 覆盖 Default 稳定性矩阵，以及 Classic 筛选、稳定性矩阵和渠道列表。
- 初始四个独占组件及专属测试；协调者于本轮明确追加授权 `service/channel_analytics.go`、`dto/channel_analytics.go` 与专属 service 回归。
- 不改公共布局、翻译文件、外置扩展子模块或 MCP。

## 已核对的数据来源与根因

- `/api/channel-analytics/filters` 提供 `groups[].code/name`；Classic 标签主动拼接 `(code)`。
- `/api/channel-analytics/stability` 提供 `group/group_name`；两模板主动把原标识作为副标题。
- `/api/channel-analytics/channels` 当前只返回渠道配置 `group`，Classic 直接展示；不能只删文案掩盖缺少名称字段。
- `channelAnalyticsGroupDisplayNames` 原本从可用分组配置取描述，不一定是当前实体名称且不解析 alias；获授权后统一复用 `model.GetGroupDisplayNameMap`。
- 原生统计路由均要求 `AdminAuth` 及 `ChannelRead`；筛选项仍来自指标中的原分组集合，名称映射只装饰已有结果，不向响应追加其他组。
- 名称查询失败记录日志后沿用已有配置名称及原标识回退；指标主查询错误仍正常返回，不因展示降级吞掉错误。

## 安全、兼容与覆盖边界

不改变管理员权限、路由、计费、统计、配置保存或数据库结构。后端追加字段如获分派，必须批量复用既有解析器，保留旧字段。

此子项验证原生组件；用户截图来自外置 `channel-quality` 扩展，扩展 ZIP 的 UI 不因此自动修复。公共页头布局由 #405 工人负责，本记录不混称扩展验收。

## 验证计划

1. 真实组件配合受控 API，先验证名称不同于 code 的失败用例，再最小修改转绿。
2. 覆盖改名及历史 alias、删除未知回退、筛选和下钻请求仍提交 code。
3. 长名称在桌面与手机浏览器中检查可读性、完整名称和页面溢出。
4. 单次测试限制 60 秒，D 盘 TEMP/TMP，限制 Node 内存与一个 worker；不重复全量构建。
5. 执行受影响测试、类型检查、定向 lint/格式化、文档链接与差异检查；由另一工人独立审查。

## 验证结果与交接

### 2026-11-16 前端写权释放与 Claude 交接

收到协调者 `msg_386f1ad95975` 后，停止全部前端代码和测试修改，前端写权已释放；不回滚已有成果。后端授权范围仍由本工人完成。共享文档索引和独立审查由协调者统筹。

已修改的前端文件：

- `web/src/features/channel-observability/stability-view.tsx`：删除分组 code 副标题，保留渠道和模型副标题逻辑。
- `web/classic/src/pages/ChannelObservability/index.jsx`：筛选标签改用 `name || code`，选项 value 保持 code。
- `web/classic/src/pages/ChannelObservability/OperationsView.jsx`：分组不附 code，渠道与模型元数据仍显示。
- `web/classic/src/pages/ChannelObservability/ChannelsView.jsx`：配置分组消费 `group_name || group`。
- `web/src/features/channel-observability/__tests__/group-names.test.tsx`：新增名称、刷新改名、历史标识下钻和删除/未知回退用例。
- `web/classic/src/pages/ChannelObservability/__tests__/group-names.compat.test.jsx`：新增矩阵、渠道配置和筛选用例；仍有一个 JSDOM 筛选交互失败需要 Claude 继续。

已取得的证据：

- Default 红：新增 2 项中 1 项准确发现 `legacy-enterprise` 副标题；绿：新增 2 项与原 API 3 项共 5 项通过，见 `D:/Temp/issue403-observability/default-red.log`、`default-green.log`。
- Classic 红：矩阵附码、渠道缺名、筛选名称无法精确匹配；已修复测试环境的 `IntersectionObserver` 和 Node 26 `localStorage` 边界。
- Classic 最新：4 项中 3 项通过；筛选用例能找到纯名称和删除回退选项，但 JSDOM 点击后未观察到追加 `groups` 的请求；见 `classic-final.log`。不能把这一项报为通过。
- 真实 Orca 浏览器中，Classic 选择名称后实际请求为 `groups=legacy-enterprise`，页面不显示原 code；矩阵在 1440 宽下 `scrollWidth=1440`，390 宽下 `scrollWidth=390`，移动表格内部横向滚动宽 1510。已验证长名称和 `deleted-code` 回退；桌面截图为 `D:/Temp/issue403-observability/classic-desktop-matrix.png`。
- 已编译双模板原生组件最小夹具，均在 D 盘输出，不是整站构建；Default 浏览器交互、双模板移动截图及浏览器改名复验尚未完成。Classic 夹具仅替换 API 网络边界，真实页面和 Semi UI 保留。
- `bun run typecheck` 被非归属 `api-keys-mutate-drawer.tsx:287/291` 的语法错误阻塞，已上报协调者；见 `typecheck.log`。尚未执行前端定向格式化与 lint。

接手顺序：修复 Classic 筛选测试的真实选择交互；确认旧附码实现能够失败；完成前端格式化/lint、typecheck 与 Default 浏览器及移动/改名复验；随后由另一工人独立审查。不得把原生组件夹具结果当成当前外置扩展页面的完整验收。

进程与浏览器归属：

- 本工人启动的只读本地静态服务器：Default PID `57816`（`127.0.0.1:5199`），Classic PID `41476`（`127.0.0.1:5200`）；无生产连接，可交接复用。
- Orca 页面 `1556dec5-ee03-487a-aad8-3c126bf0550e` 当前为 Classic 移动宽度 390 的矩阵；只操作此页面，不触及其他工人页面。
- 夹具脚本、构建产物、60 秒超时测试包装器和全部日志在 `D:/Temp/issue403-observability`。脚本 `build.mjs` 两次定向构建曾分别遇到路径编码和开发热刷新夹具问题，已修正；Classic 用 API 边界替身避开无关 helpers 聚合导出冲突。
- 所有已启动测试已结束，无遗留测试进程；上面两个本地服务器仍保留供接手者复用。未启动或停止任何共享开发服务器。

对其他工人的接口回执：已回复外置/内嵌 `channel-quality` 工人 `msg_0cfa8e5fc278`，渠道 `group_name` 为按原组顺序、逗号加空格连接的完整展示字符串，`group` 不变；`stability.group_name` 与 `filters.groups[].name` 均由后端批量解析。

没有新增翻译键或待翻译文案；未执行提交、推送、部署或生产写操作。

### 后端交付与最终验证

后端文件：`service/channel_analytics.go`、`dto/channel_analytics.go`、新增 `service/channel_analytics_group_names_test.go`。

- `/channels` 只读响应新增 `group_name`，使用完整配置组列表批量映射，`group` 原值不变。
- `/filters` 与 `/stability` 共用的名称解析改为当前实体/alias 映射；只装饰已有统计结果，未采样分组不会因为全局名称表而进入筛选项。
- 成功读取名称表时，删除、未知标识回退原标识；名称表读取失败时单独记录错误，并沿用已有配置名称及原标识兜底，不吞掉指标查询错误。
- 新测试使用 SQLite 显式夹具，每个子用例独立初始化，覆盖当前名称、改名、真实历史 alias、删除/未知、未采样分组不扩散、业务筛选原值、名称查询失败与主查询错误。
- 已有路由权限回归证明普通用户无法调用统计接口；代码核对 `/filters`、`/channels`、`/stability` 使用同一 `AdminAuth + ChannelRead` 中间件，不新增授权入口。

最新命令（2026-11-16 14:09，退出码 0）：

```powershell
$env:GOTMPDIR='D:/Temp/issue403-observability'
$env:GOCACHE='D:/Temp/newapi-mcp-money-go/cache'
python D:/Temp/issue403-observability/run.py backend-final-verified `
  'D:/脚本程序/开源程序二开/newapi' go test ./service ./dto ./router `
  -run 'TestChannelAnalytics|TestParseChannelAnalytics|TestFeatureRoutesRejectInsufficientAuthorizationBeforeHandlers' `
  -count=1 -timeout=60s -vet=off -p=2 -json
```

包装器为每个进程设置 D 盘 TEMP/TMP 与 Node 内存限制，并施加 60 秒总超时。JSON 日志 `backend-final-verified.log`：service 19 项、dto 2 项、router 1 项顶层测试全部通过，另有 22 个子测试通过，0 失败。新增名称测试先在旧实现准确报缺少 `group_name` 与 alias 名称不匹配，再转绿；初次冷缓存及带默认 vet 的补充尝试达到 60 秒后已终止，不能把这些尝试视作通过，独立 vet 尚未完成。

`gofmt -l` 对三个 Go 文件输出为空，后端与本工作记录的 `git diff --check` 通过；MySQL/PostgreSQL 未运行集成实例，本改动没有迁移或新增 SQL，名称解析复用现有跨数据库 GORM 实现。没有改动 relaykit，不涉及独立模块构建门禁。

后端写权随本工人最终回执一并释放。整体子项仍未完成：前端接手者补齐剩余验证和格式化，协调者登记共享文档索引，再由另一工人执行独立审查；未做整站构建或线上验收。

跨项目影响仅为统计响应追加兼容的 `group_name` 字段及显示名称修正，不改变模型、鉴权、计费、筛选或路由契约。对 `NewAPIForDouDi` 和 `NewAPIModifyByGang` 都可以分享“名称与稳定标识分离”的修复方案，但两仓库是否具备相同统计接口及 GroupAlias 尚未核对，不建议直接移植；未发送 Issue、PR 或其他外部通知。

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

## Claude Code 路由 P2 最小撤销与复验（2026-11-16，task_533834b15e00）

本节为追加内容，原文未改动。以本节为准：前文所述新增 `/channel-observability` 路由已撤销。

### 撤销范围

- 新增的 `web/src/routes/_authenticated/channel-observability/index.tsx` 已不存在：Glob 无结果，`git ls-files --error-unmatch` 退出 1（从未被跟踪）。
- `web/src/routeTree.gen.ts` 中没有 channel-observability 登记；相对 HEAD 仅余 45 行新增，全部属于 affiliate-management 与 mcp-management，予以保留，未用 HEAD 整件覆盖。
- 未新增第二入口，沿用原 `/extensions/$moduleId/$pageKey` 的 SUPER_ADMIN 与模块启用门禁。`use-sidebar-config.ts:105` 与 `use-sidebar-data.ts:117` 仍含 `/channel-observability` 字符串，仅用于 configUrls 映射，本轮未改，也未单独证明其无副作用。
- 保留 E 分组名称修复（`web/src/features/channel-observability/stability-view.tsx`、Classic 三个页面文件）与内嵌 channel-quality 三入口。

### 复验（单进程、60 秒上限、TEMP/TMP 指向 D:/Temp/issue403-e-route-fix-claude、Node --no-experimental-webstorage）

- 内嵌三入口 `node --test extension/channel_quality_group_names.test.mjs`：6/6，退出码 0。
- Default vitest（group-names、api、原扩展路由 -index.test.tsx）：3 文件 8 项通过，退出码 0，23.47 秒；含旧入口 role10 拒绝契约。
- Classic vitest group-names.compat：4/4，退出码 0，35.01 秒；筛选选中后仍提交原 code；尾部 React 警告为非致命。
- 日志与哈希快照：D:/Temp/issue403-e-route-fix-claude 下的 embedded.log、default.log、classic.log、sha-before.txt（未覆盖旧证据目录）。

### 冻结 SHA-256（撤销后、本轮复验前取得）

- extension/builtin/channel-quality/public/app.js：9ada368f5dfa4a78c9c99a913d2458d550789508f5d53f88914a8205b6b60b96
- extension/builtin/channel-quality/public/native/default.mjs：34ad3da40227b4f6d8b8f2781db8b916dec1b56311ba96594b98a37e7f77f416
- extension/builtin/channel-quality/public/native/classic.mjs：8854bbd80324e5584807c692f15cd2c61f65a9e1d8315081f339c88a02948005
- extension/channel_quality_group_names.test.mjs：9d8292fbfbf1dbea18297adb794e4b83e469eca3e962e03cd92fc6d7500208f0
- web/src/features/channel-observability/stability-view.tsx：b61d663b1c9a11bf7a3e6fc5031b2c45a97dfb981569fafb7d62668acde9dd3a
- web/classic/src/pages/ChannelObservability/index.jsx：2ee3ce71bca3b472f935844196d9a36657fa9365d54d045bf6e93da244f27e4e
- web/classic/src/pages/ChannelObservability/OperationsView.jsx：669220c52c6b497021829f6c21df8e0360c45041c86542a922fbb8b586573312
- web/classic/src/pages/ChannelObservability/ChannelsView.jsx：dd6ffc63c2c527146aa8203b31373ee1d179b568833009463ae5d75a995b3f5c
- web/src/routeTree.gen.ts：7380ea6686ad50e5ea9f4fc4d16e66f390197136c8a41aee38709c9223feac63（共享文件，其他并发增量会使该值变化，仅作本次快照）

### 未验证边界

- 禁用模块的真实页面用例本轮未重跑；旧入口门禁仅由现有 -index.test.tsx 的角色拒绝用例覆盖。
- 桌面与 390 宽度浏览器验收、Go 安装升级实测（上一轮 60 秒超时、退出码 124）本轮未完成，不宣称通过。
- 未运行全量构建、typecheck、全仓 lint 与 format，未运行 protected-headers 脚本，未提交、推送或部署。
- 独立审查须由另一名 Codex 执行，本工人未自审。

## Claude Code 本地验收 v5（2026-11-17，task_4c4ceb754327 / ctx_aa2b0cf7cc9b）

以下为本次实测，与上文历史记录分开，不覆盖旧结论。证据目录 `D:/Temp/issue403-e-acceptance-v5-ctx_aa2b0cf7cc9b`（v2/v3/v4 与审查目录只读）。HEAD `13f5fdbb93f88da11a157efa69094a9a5e472b60`，分支 custom-main。

### 方法

- 真实宿主：Default（rsbuild，127.0.0.1:5451）与 Classic（vite，127.0.0.1:5452）加载主仓内嵌 `extension/builtin/channel-quality/public/native/{default,classic}.{mjs,css}`；入口仅 `/extensions/channel-quality/index`（Classic 为 `/console/extensions/channel-quality/index`），未重建 `/channel-observability`。
- Chrome 157 + Python Playwright（executable_path），`page.route` 在网络层 mock `/api/**`；字段形状按 `dto/channel_analytics.go` 与 `controller/user.go` 对齐。夹具 code=`legacy-enterprise`，name=`企业专属模型稳定性监控分组`（name!=code）。未匹配请求一律 404 并计入 unmatched，不用成功对象掩盖。
- 全部为普通点击（高级筛选 → 所有分组/全部请求分组 → 名称），不 force-click、不清 disabled；390 宽使用 is_mobile 视口，溢出按可滚动祖先判定，不以 aria-hidden 或类名排除。

### 实测矩阵（Default/Classic × 1440/390，各格 exit=0，unmatched=[]）

- 启用+选择：下拉只显示名称，DOM 无 code；summary/trend/status-codes 均提交 `groups=legacy-enterprise`。Default 1440/390 页面错误 0；390 溢出元素均被 `overflow-x:auto` 容器裁剪，body scrollWidth=clientWidth=390。
- 改名后刷新（`改名后的企业分组`）：四格通过，旧名与 code 均不在 DOM，仍提交原 code。Classic 1440 首次误用标签「所有分组」失败，保留为 `classic-1440-rename`，以 `-r2` 为有效证据。
- 当前组多名称（另加 `团队B专属分组`/`团队C专属分组`）：四格通过，选 B 提交 `groups=legacy-team-b`，额外 code 不外露。
- 缺失名称：四格通过，回退显示原 code `deleted-code`（允许），请求提交 `groups=deleted-code`。
- 模块禁用真实入口：Default 1440/390 显示「未找到扩展页面」，无 `native/index` 与统计请求，0 错误。Classic 1440/390 仅请求 `style-0` 并得到 403，页面显示「原生扩展加载失败」，无 entry 与统计请求。

### 发现（只读，未改产品源码）

- Classic `ExtensionModulePage`（`web/classic/src/pages/Extensions/index.jsx:512-590`）只判断 `!module || !page`，不读 `module.enabled/error`；后端 `extension/native.go:107` `OpenNativeAsset` 对禁用模块返回错误（`controller/extension.go:178` 映射 403），这是 Classic 禁用时的唯一门禁。Default 在 `extension-page.tsx:267` 自行拦截。两模板行为不同，非统计泄漏。
- 夹具缺陷（非产品）：初版 mock 未模拟该 403，Classic 禁用首跑（`classic-1440-disabled`）照常加载入口与统计，已作为异常证据保留；修正 mock 后以 `classic-1440-disabled-r2` 为准。
- Classic 控制台错误为既有 React 18 告警（StrictMode、findDOMNode、ReactDOM.render、iconOnly 非布尔属性、setState 更新告警）；另有 4 条「聊天数据解析失败」来自 Classic 自身，未进一步归因。

### Go 安装升级测试（目的二）

环境 go1.25.5 windows/386，GOTOOLCHAIN=local，GOPROXY=off，独占 GOCACHE/GOTMPDIR/TEMP 位于 D 盘。原 55 秒无输出是冷编译缓存，并非测试挂死：warm1 `go build ./extension` 退出 124（60s，冷编译），warm2 退出 0（1s），warm3 退出 0（25s）。逐例 `-count=1 -timeout=55s -p=1 -v`：`TestInstallBuiltinModulesIncludesChannelQualityNativeAssets` 首跑 124，复跑 0（4s，PASS）；`TestInstallBuiltinModulesRefreshesChangedAssetsAtSameVersion` 0（28s，PASS）；`TestInstallBuiltinModulesRefreshesOlderBuiltinVersion` 0（4s，PASS）。三例通过来自两批命令（预热后复跑），详见 `go-summary.txt`（a4f7c1f0…cebfca）。

### 源码前后哈希与夹具

- 前后一致（`sha-before.txt` 与 `sha-after.txt`）：routeTree.gen.ts 7380ea66…ac63；app.js 9ada368f…6b60b96；native/default.mjs 34ad3da4…f77f416；native/classic.mjs 8854bbd8…2948005；extension/registry_test.go 72ff05a4…2d62f574；extension/builtin.go 39d3ecfa…42de。git status 仍为 302 项。
- 夹具：mock_host.py 4e69f635ea00d16986555e339fea60688ad42516f7376505f24d842b19888327；run_scenarios.py 05994bc65d6509e17dbd6e7fa80752225ec06412ba04cf4c6aee306e4f21e7d4；run_load.py f628c2ce…3603；run_interact.py 8c679c35…fdbb。dom/net/shot 各 32 个，逐项哈希见 `sha-*.txt`。

### 未验证边界

真机/真实触屏设备、真实后端（含真实 403 响应链）、生产环境、zzapi 部署均未验证；mock 的 analytics 数据为空数组，未验证图表数据渲染；未运行全量 build/typecheck/format 与 protected-headers 脚本；未提交、推送或部署；独立核验须由另一 Codex 完成。

## Claude Code 覆盖补验 v7（2026-11-17，task_b6dde86a9901 / ctx_6fa6cd30a821）

本节追加，不改旧结论。针对独立审查 P2（无 /stability、/channels 请求；多组以多个筛选候选冒充；渠道为空）与 P3（点击失败仍 exit 0）。证据目录 `D:/Temp/issue403-e-acceptance-v7-claude`（独占；v5、v6、审查目录只读）。HEAD `13f5fdbb93f88da11a157efa69094a9a5e472b60`。

### v5 与 v7 的覆盖范围

- v5 矩阵仅覆盖：启用后的筛选、改名刷新、缺名回退、模块禁用，以及多个筛选候选；32 份网络记录不含 `/stability`、`/channels`。
- v7 新格：同一条记录 `group=legacy-enterprise,legacy-team`，`group_name` 为当前显示名；在真实入口点击现有 stability 视图与 channels 视图，并捕获 `/api/channel-analytics/stability` 与 `/channels` 请求（dimension=group、traffic_source=relay、page_size=30）。channels 返回 `mock-channel #1`，非空。
- Default 的 stability 视图：分组行显示 `group_name?.trim()||group`；channels 视图产品代码无分组文字，故只验请求与非空渠道行，未新增 UI。Classic 的 channels 行显示「配置分组 + group_name 或 group」。
- 标签：Default「操作矩阵」「渠道和模型」；Classic「运维矩阵」「渠道与模型」。

### 夹具修正

- mock `/api/status` 补 `chats:[]`，Classic 4 条聊天解析错误消失；仍有既有 React warning（Classic 有、Default 为 0），pageerror 为 0，不能写零 console。
- `run_v7.py` 为新断言执行器：点击失败、名称/code 断言失败、unmatched、pageerror 任一出现即 exit 非零。stability 分 initial / after-refresh 两阶段断言。
- 隔离负控 `v7-negctl-click-fail`（点击不存在的标签）exit 1，证明点击失败返回非零。

### 12 个有效成功格（Default/Classic × 1440/390，另有 3 份保留证据）

- base 4 格、rename 4 格、missing 4 格均 exit 0，overflow 0，unmatched []，pageerror 0。多组名称在 DOM 同一行完整显示（如 `改名后的企业分组,团队B专属分组`），不附 code。
- 改名刷新后显示新名称，旧名与 code 均不在 DOM；缺名时回退为原 code `legacy-enterprise,legacy-team`；v7 无任何 groups 查询，本轮不验证筛选提交原 code，该项引用 v5 已独审的筛选提交证据，本轮未重跑；response 中的 group 为 mock 输入，不能作提交证据。
- 保留失败证据：`v7-default-1440-rename`、`v7-classic-1440-rename` exit 1，原因是脚本在两阶段都断言新名称（脚本缺陷，非产品问题），修为分阶段后以 `-r2` 重跑通过。
- 截图 15 张，dom/net 逐格落盘；net 记录含 path、query、body。

### 哈希与命令退出码

- 产品源码前后一致（`sha-before.txt` 与 `sha-after-src.txt`）：routeTree 7380ea66…ac63；app.js 9ada368f…b60b96；default.mjs 34ad3da4…77f416；classic.mjs 8854bbd8…948005；registry_test.go 72ff05a4…2d62f574；builtin.go 39d3ecfa…42de。
- 夹具：mock_host.py fa639872…ffbd8；run_v7.py 9afaea1c…c6ea；run_scenarios.py d996352a…17b8；run_interact.py 0c17afb2…de24。
- 各格命令均为 `timeout 60 python -I scripts/run_v7.py ...`，TEMP 在 D 盘；退出码见证据目录 `*.exit.txt`。

### 未验证边界

真实后端、真机触屏、生产与 zzapi 部署未验；Default channels 视图无分组文字，故其分组名称仅按 stability 视图验证；未重跑 Go 三例及全量 build/typecheck/format；未提交、推送或部署。

## Claude Code P3 返修 v8（2026-11-17，task_09df88f3e3a4 / ctx_6e432421613c）

本节追加并更正上文 v7 口径，旧证据只读不动。证据目录 `D:/Temp/issue403-e-acceptance-v8-p3`（独占）。

- 口径更正：v7 的有效成功格为 12 个（base、rename、missing 各 Default/Classic × 1440/390 四格）；v7 目录的 15 份证据包含这 12 个成功格、2 份保留的旧 rename 失败（脚本缺陷）和 1 份点击负控。v7 没有任何 groups 查询，因此本轮不验证筛选提交原 code；该项引用 v5 已独审的筛选提交证据，本轮未重跑，response 中的 group 为 mock 输入，不能作提交证据。
- console 与 pageerror：Classic 仍有既有 React warning，Default 为 0 条，pageerror 全部为 0，不能写成零 console。
- P3 返修：v7 `mock_host.py` 对未知 `/api/channel-analytics/*` 外层 `route.fulfill` 默认 200。v8 复制到新目录后改为失败体返回 404，`unmatched` 仍记录，`run_v7.py` 的 `check(not mock.unmatched, ...)` 守卫使进程退出 1。
- 定向测试（真实 `Mock.handle` 加假 route，记录 `fulfill` 实参）：`test_handler.py` exit 0，未知 `/api/does-not-exist` 与未知 `/api/channel-analytics/bogus` 均为 404 且 unmatched 非空，stability 为 200 且 2 条、channels 为 200 且 1 条。`test_guard.py` exit 0：复用 run_v7 同款守卫的子进程，未知 analytics 退出 1，未知 api 退出 1，已知 stability 对照退出 0。
- 未做浏览器负控：本轮不启动 dev server，也不把任何 probe 的 exit 0 当作浏览器负控通过。未重跑 12 格、Go、build 或 typecheck，产品源码未改。
- SHA：mock_host.py 前 fa639872…ffbd8，后 41d21e34…44466；run_v7.py 9ea71025…df604e；test_handler.py ee5c4b13…187a1。
