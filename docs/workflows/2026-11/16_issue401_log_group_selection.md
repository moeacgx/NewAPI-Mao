# Issue #401：Default 普通日志分组选择迁移

## 目标、范围与根因

Default `/usage-logs/common` 普通日志仍把 group 作为自由文本展示，缺少 Classic 已有的按权限分组下拉；当前 code 链接会直接回显 code。仅修复普通日志，不扩展任务或绘图分类，不修改日志匹配语义。后端 `ResolveGroupLogIdentifiers` 已按当前 code、历史 alias、显示名称顺序解析，不应误报名称查询全部失效。

## 方案与接口契约（编码前记录）

- Default 复用原生 Select、React Query 与统一 API 客户端。管理员全部日志视图读取 `GET /api/group/details`，普通用户及管理员仅本人视图读取 `GET /api/user/self/groups`；选项分别来自原 data，展示当前 name，提交 code。
- 两接口顶层追加可选只读 `group_names: Record<string, string>`，保留 data、auto_group、group_tags、canvas_default_group 的原形状、默认值和权限。成功空映射为 `{}`；读取失败省略字段、记录服务端错误，前端保留筛选值及原选项并提供重试。
- 管理员复用 `GetGroupDisplayNameMap`，保留已有历史标签展示回退。普通用户通过专用 model 批量 helper，仅解析现有可用分组稳定 ID 的真实 code/alias，不读取 Option 历史标签，禁止按名称授权。空 ID 集合返回空映射；当前 code 的优先级覆盖全表，即无权限组当前 code 遮蔽可用组同名 alias 时不泄露名称。
- 历史 URL 中已选 code/alias 只解析展示名称，不把 URL 中的值改写为名称或新 code。未知、删除组允许回退原值；未获准标识不成为新的可选项，既有 URL 仍可保留/清空。名称解析不参与后端授权、日志用户作用域、计费或路由。
- 查询缓存按账户、会话和视图隔离，沿用分组配置保存时已有缓存失效前缀；账户或视图变化不沿用旧主体选项。处理加载、失败、重试、清空、空组、长名称和窄屏。

## 测试计划

- 先写失败测试，再实现：Default 当前名称不等于 code、名称刷新、alias 链接保持原值、未知/删除回退、清空、普通用户只取 self 接口、切换账户/视图、失败重试与空选项。
- Model：空授权、真实 alias、改名、删除、无权限 alias、Option 同名旧标签不得授权、全局当前 code 优先，SQLite 确定性测试；采用三数据库通用 GORM 查询。
- Controller：两个出口原字段兼容、管理员映射兼容、普通用户实体裁剪、读取失败省略映射，使用 require/assert。每次测试设置最长 60 秒。
- 核对 Classic 原有下拉与 code 提交契约，运行可用相关回归；运行 Default typecheck、受影响 lint/格式、Markdown 格式及 diff 检查。尽可能获取本地浏览器证据，明确模拟 API 与真实数据库边界。

## 协调者集成项

- 本 worker 不改共享 locales、开发索引与 custom-development。新增翻译键为 `Clear group filter` 和 `Failed to load group names`；其余复用 `Group`、`All Groups`、`Loading...`、`Failed to load groups`、`Retry`。共享集成 worker 已添加本轮翻译，本地浏览器已显示中文；最终七语言扫描由协调者统一执行。
- README 索引应链接本工作记录；能力说明建议为“Default 普通日志按视图权限加载分组，展示当前名称、保持稳定 code 提交，历史 alias 仅解析展示，保留筛选语义”。本 worker 不改共享文件，最终集成由协调者核对。
- 独立审查由其他 worker 执行；不提交、push、部署或向外部发送消息。

## 实施与验证结果

### 实现与兼容边界

- `common-logs-filter-bar.tsx` 的普通日志 Group 输入替换为新建 `LogGroupSelect`，沿用 Default 原生 Select、按钮和主题；任务、绘图筛选不变。选择只更新原筛选草稿，点击 Search 后仍提交 code。
- 新建 `useLogGroupOptions` 按当前日志视图选择接口，管理员本人视图同样读取 self 接口。查询键追加用户 ID、session sid、role、group，保留既有分组配置失效前缀；切换主体立即使用独立查询结果，不借用前一主体选项。
- 加载中允许清空；接口失败和名称映射缺失分别显示局部错误与 Retry。未知、删除或当前不可选的已选值以禁用项保留，只允许清空或切换到合法选项；成功空映射不会误报失败。
- 浏览器发现 StrictMode 首次挂载取消请求后，统一客户端的 GET 去重会让第二次挂载复用已取消的请求。通过正确启用根 StrictMode 的真实客户端测试先复现失败，再由 React Query 自行去重，调用客户端时设置 `disableDuplicate: true`。正常取消不弹错误通知，HTTP 500 归一为带原始 cause 的局部错误，避免全局查询错误处理把日志页跳转到 500 页面。
- 新建 model helper `GetLogGroupNamesForIDs`，使用两次批量 GORM 查询：读取全局当前 code 判断正式标识优先级，再读取授权 ID 的真实 alias。空授权不查全表，不把显示名称或 Option 旧标签作为授权依据，也不把无权限组信息写入响应；无持久化或 schema 变更。
- `controller/group.go` 两个出口仅追加可选名称映射。管理员保留既有 `GetGroupDisplayNameMap` 的历史标签展示回退；self 出口按既有可用组 ID 限定实体。不改变日志查询、鉴权、计费、路由或其它选组选项。

追加字段示意（省略原响应的其它字段，不表示删除它们）：

```json
{
  "success": true,
  "data": {
    "public": {
      "id": 42,
      "code": "public",
      "name": "当前普通日志分组"
    }
  },
  "group_names": {
    "public": "当前普通日志分组",
    "old-public": "当前普通日志分组"
  }
}
```

以上为 self 的 data 对象示例；管理员 data 继续为原数组。成功但没有可解析授权组时，`group_names` 为 `{}`；新增映射查询失败时，仍返回原有 `success: true` 与 data，省略 `group_names` 并记录服务端错误，前端可重试。历史 URL `?group=old-public` 的显示名称可变成当前名称，但请求仍提交 `old-public`；只有主动重新选择 public 才提交 `public`。映射不是授权列表，不能据此扩大选项或用户日志作用域。

### 测试命令与结果

所有临时产物使用 D 盘目录，TEMP、TMP、GOTMPDIR、GOCACHE 仅设置于测试子进程；没有清理共享缓存或改全局环境。单个测试用例/Go 测试命令设置 60 秒上限。

1. Go 定向回归通过：2 个包、3 项测试。覆盖名称与 code 不同、改名、真实 alias、删除/未知、未授权 alias、Option 同名旧标签不授权、全局当前 code 优先、原响应兼容、映射故障省略字段及恢复。

   ```powershell
   $env:GOTMPDIR='D:\temp\issue401-log-group-go'
   $env:TEMP=$env:GOTMPDIR
   $env:TMP=$env:GOTMPDIR
   $env:GOCACHE='D:\Temp\newapi-mcp-money-go\cache'
   go test ./model ./controller -run '^TestLogGroup' -count=1 -timeout 60s
   ```

2. Default 定向回归通过：4 个文件、18 项测试，最终一轮 7.88 秒。组件测试模拟统一 API 返回；hook 的两项测试保留真实 Axios 客户端和拦截器，仅替换网络 adapter，覆盖根 StrictMode 取消与重新挂载、HTTP 500 局部重试。原自动填充、用户名/ID 和 skill-type 用例继续通过。

   ```powershell
   # 在 web 目录执行；TEMP/TMP 指向 D:\temp\issue401-log-group-browser。
   $env:NODE_OPTIONS='--no-experimental-webstorage'
   bun run test src/features/usage-logs/components/__tests__/log-group-selection.test.tsx src/features/usage-logs/components/__tests__/filter-autofill.test.tsx src/features/usage-logs/components/__tests__/skill-type-filter.test.tsx src/features/usage-logs/hooks/__tests__/use-log-group-options.test.tsx --run --maxWorkers=1 --testTimeout=60000
   ```

3. Classic 组件兼容回归通过：2 个文件、5 项测试，36.62 秒。另有纯 Node 合同测试 2 个文件、36 项通过，334 毫秒；这 36 项不能表述为组件或浏览器验证。

   ```powershell
   # 在 web/classic 目录执行，设置同上进程环境。
   bun run test:compat src/components/table/usage-logs/__tests__/admin-user-filter-layout.compat.test.jsx src/components/table/usage-logs/__tests__/filter-autofill.compat.test.jsx --testTimeout 60000
   # 在仓库根目录执行。
   node --test --test-timeout=60000 web/classic/src/helpers/groupDetails.test.mjs web/classic/src/components/table/usage-logs/__tests__/user-filter.test.mjs
   ```

4. 受影响的 6 个 TS/TSX 文件经项目 oxfmt 格式化、保护版权头的定向格式检查、带 `.oxlintrc.json` 的 oxlint 检查通过；Go 文件已 gofmt。工作记录使用相同 Markdown 格式工具，定向 `git diff --check` 通过；新文件另行检查行尾空格及结尾换行。最终格式调整后，真实客户端 hook 2 项回归再次通过，3.13 秒。工作记录没有新增相对 Markdown 链接，API 路径与源码出口逐项核对。
5. 曾运行 Default typecheck，失败位置为并行返佣功能的 5 处 `ByRoleOptions` 类型错误，与本项文件无关。协调者明确暂停 worker 重复全量 typecheck/build，最终类型检查、双模板构建与独立审查由协调者统一执行，本记录不宣称这些检查通过。早期 Go 验证受 C 盘满及原缓存迁移影响，改用 D 盘后上述定向回归已通过。

### Hide 既有能力保留

授权修改既有 `filter-autofill.test.tsx` 的 Group 断言，没有删除原用户名、令牌自动填充和隐藏断言。使用 `vip-internal` / `专线当前名称` 验证选择后展示当前名称，Hide 后触发器和下拉项显示遮蔽字符，title 不含当前名称，aria-label 仅保留 Group，Search 提交仍为 `vip-internal`，Show 后恢复当前名称。该隐藏行为沿用既有隐私控制，不新增授权规则。

### 浏览器证据与模拟边界

本 worker 使用真实 Chromium 访问本地 Rsbuild 的 `/usage-logs/common`，操作真实 React 页面和原生选择器；**所有 `/api/**` 请求均由 Playwright route 模拟，没有证明真实后端认证或真实数据库端到端联通\*\*。后端权限和响应合同由上述独立 SQLite 测试覆盖。浏览器只覆盖普通用户本人视图；管理员切换由组件和控制器测试覆盖。

证据目录：`D:/temp/issue401-log-group-browser/`。

- `desktop-alias.png`：`group=old-public` 显示“当前普通日志分组名称（名称与标识不同）”，没有把 URL 改成名称或 public；随后主动选 public 并 Search 的 mock 网络请求仍为 `group=public`，未请求管理员组选项接口。
- `desktop-hidden.png`、`hidden-privacy.json`、`hidden-options-privacy.json`：当前名称和下拉项遮蔽；触发器 `ariaLabel` 为“分组”，`titles` 为空。Show 后当前名称恢复。
- `mobile-select.png`、`mobile-options.png`、`mobile-width.json`：390×844 窄屏抽屉及长名称下拉可用；文档宽度为 390，实际可见下拉左右边界为 16 / 338，没有横向溢出。
- `retry-error.png`、`retry-recovered.png`、`retry-recovered.json`：故意模拟 self groups HTTP 500 时日志页面保留并显示局部重试；重试成功后当前名称恢复、alert 清除、URL 仍为 `group=old-public`。控制台对应的 1 条 HTTP 500 来自此故障注入。
- `setup.cjs`、`retry.cjs` 为模拟数据和故障注入脚本。已关闭本 worker 的独立浏览器 session，不影响其它 worker 浏览器。

### 双模板覆盖、共享影响与限制

Default 修复普通日志分组选择与历史链接当前名称展示；Classic 只读核对 `UsageLogsFilters.jsx` 的按角色接口和 `createGroupOptions` / `createUserGroupOptions`，原来已经展示 name 并提交 code，新增顶层字段不改变其 data 读取。本 worker 没有修改 Classic 源码；现有组件回归 5 项和 Node 合同 36 项通过，没有本 worker 的 Classic 浏览器证据。

共享接口影响仅为两个已有 GET 响应追加可选只读 `group_names`，请求路径、提交标识、权限和原 data 合同兼容；不自动向其它项目发送通知。MySQL、PostgreSQL 没有实际数据库运行证据，查询只使用通用 GORM 能力；最终类型检查、双模板构建、共享翻译扫描和原 reviewer 复审结果见本文末尾门禁段。

分别评估分享：`NewAPIForDouDi` 若同样存在普通日志自由文本选组和稳定分组实体模型，值得参考本修复以减少名称/标识混淆；应先适配其 ID、alias 与合法组选项权限。`NewAPIModifyByGang` 若也采用双模板且存在同类迁移缺口，值得参考下拉迁移与历史链接只解析展示的做法；不应在没有稳定身份及别名授权模型时直接照搬后端映射。以上均仅为建议，没有创建 Issue、PR 或发送外部消息。

## 独立审查两项 P2 返修

独立 reviewer `ctx_c9eb27ce243a` 确认：分组设置保存只对
`['system-settings', 'group-details']` 精确执行 `setQueryData`，没有失效此前缀下
按用户及会话隔离的日志选项查询。已有查询仍处于 10 秒新鲜期时，立即返回日志页面
会继续显示旧名；已挂载观察者也不会重取。初次实现记录中“沿用既有分组配置失效前缀”
仅描述查询键设计，不能证明实际保存链路已覆盖该前缀。

本次追加授权范围为 `web/src/features/system-settings/models/group-ratio-form.tsx`、
其专属新测试及本工作记录；修改前检查该表单没有其它未提交改动。修复复用既有
`groupDetailsQueryKey` 前缀执行正常查询失效，保留原保存响应写入和其它缓存刷新行为，
不在分组设置中导入日志专属键、不改变 staleTime、不改变 API 或普通用户权限。

先新增真实表单保存、真实 QueryClient 和真实 LogGroupSelect 的集成测试，只模拟网络边界：
覆盖改名后立即返回（缓存仍在新鲜期）、已挂载管理员选项自动刷新、选中和提交仍为
稳定 code，以及普通用户只从 self 接口取得合法组选项。测试先在修复前确认失败，
再验证修复后通过；每个用例最长 60 秒，TEMP/TMP 使用 D 盘，保留 Node 兼容参数。
本修复仅影响 Default 的 React Query；Classic 不使用此缓存体系，原合同及前次证据不变。

同一 reviewer 追加 P2：名称字典直接用 URL 筛选值索引时，未知 `__proto__` 会读取
`Object.prototype` 并作为 React 子节点导致渲染失败，`constructor` / `toString` 也会命中
继承成员。修复前先追加原型属性名、真实自有同名字符串、空/非字符串名称的回归；
实现只接受自有且为非空字符串的名称，其他情况回退原筛选值，不改变提交或授权。
限定核查本项新组件和 hook 的其它字典读取，不扩大到全项目重构。

### 实现和契约

`group-ratio-form.tsx` 在保存成功后的原有失效列表中追加
`invalidateQueries({ queryKey: groupDetailsQueryKey })`。此处使用既有分组详情业务前缀，
不引用日志私有查询键。保存响应仍写入父查询、渠道分组查询，原有状态更新和其它失效
全部保留；活跃的日志派生查询立即重取，未挂载查询标记过期并在再次进入时重取。
不调整 10 秒 staleTime，不增加延迟，也不依赖等待缓存自然过期。

`LogGroupSelect` 先保留合法选项的当前名称；对历史标识的名称映射先用
`Object.hasOwn` 确认是自有属性，再验证值为非空字符串，其他值均回退原筛选标识。
合法自有的 `__proto__`、`constructor`、`toString` 字符串名称仍可正常展示，
没有将这些标识列入禁用名单。核查本项 hook 的字典遍历使用 `Object.entries`，
不会读取继承成员；组件的任意键映射读取仅此一处，已加入运行时校验。

API 路径、响应、授权组、历史 alias 和提交值没有变化。本返修没有新增翻译键。
Default 分组设置和日志使用的 React Query 缓存受到修复；Classic 使用独立组件与请求
生命周期，没有这组查询键或新增名称字典索引，不需要改动 Classic 源码。
Classic 原有名称展示和 code 提交合同仍由前次组件 5 项及 Node 合同 36 项证据覆盖，
这些是前次验证结果，不能算作本轮新增测试或浏览器证据。

### 红绿验证与边界

修复前先运行真实保存链路专属测试，3 项中 2 项失败、1 项通过，总耗时 8.72 秒：
管理员立即返回和保持挂载的观察者都仍显示“改名前当前名称”，普通用户 self 范围项通过。
修复前的名称边界回归 19 项中 6 项失败、13 项通过，总耗时 8.17 秒，
并出现 2 个未捕获 React 错误；覆盖原型属性读取、对象崩溃及数字/数组错误展示。
没有隐藏控制台错误或删除隐私断言来取得通过。

格式化和最终断言补齐后的定向回归共 6 个文件、40 项通过，耗时 39.89 秒：

```powershell
# 在 web 目录执行；仅为当前测试进程设置环境。
$env:TEMP='D:\temp\issue401-log-group-browser'
$env:TMP=$env:TEMP
$env:NODE_OPTIONS='--no-experimental-webstorage'
bun run test src/features/system-settings/models/__tests__/group-ratio-log-cache.test.tsx src/features/system-settings/models/__tests__/token-group-limit-settings.test.tsx src/features/usage-logs/components/__tests__/log-group-selection.test.tsx src/features/usage-logs/components/__tests__/filter-autofill.test.tsx src/features/usage-logs/components/__tests__/skill-type-filter.test.tsx src/features/usage-logs/hooks/__tests__/use-log-group-options.test.tsx --maxWorkers=1 --testTimeout=60000
```

新增保存链路 3 项执行真实 `GroupRatioForm`、表单、保存 API 包装、QueryClient、
`LogGroupSelect` 和 auth store，只有 GET/PUT 网络返回被模拟。断言实际调用
`PUT /api/group/details`、保存 payload 保留稳定 id/code 和 option_updates，
名称更新后模拟日志提交仍是 `public`；切换普通用户只读取
`/api/user/self/groups`，下拉不包含管理员私有组。测试使用真实 10 秒新鲜期、
无等待过期或手动调用失效方法。日志提交按钮和页面切换属于测试夹具，
不构成真实路由、完整日志请求或真实后端联通的端到端证明。

新增名称边界 11 项覆盖三个未知原型属性名、三个真实自有同名字符串名称，以及
空字符串/null/数字/对象/数组。既有历史 alias、改名、未知/删除、清空、Hide、
用户名/ID、skill-type、客户端失败重试和分组令牌限制配置回归同时通过。
本轮没有新增浏览器截图；前次 Chromium 证据与独立 reviewer 的动态失败证据保持原样，
不将 jsdom 组件回归描述为浏览器验证。

最终定向格式检查 8 个 TS/TSX 文件通过，本轮 4 个源码/测试文件 oxlint 通过，
工作记录 Markdown 格式检查通过。首次 lint 发现新测试夹具的 4 个按钮缺少显式
`type`，补为 `type='button'` 后 lint 通过，保存链路 3 项再次通过（6.28 秒）。
Markdown 工具拒绝包含 `..` 的路径，改从仓库根目录传入规范相对路径后通过。
已跟踪修改的定向 `git diff --check` 通过，新建文件另查行尾空格和最终换行。

本轮修改清单：

- 既有授权文件 `web/src/features/system-settings/models/group-ratio-form.tsx`：仅追加业务前缀失效。
- 新增 `web/src/features/system-settings/models/__tests__/group-ratio-log-cache.test.tsx`：真实保存和缓存集成回归。
- 本工作项新建的 `web/src/features/usage-logs/components/log-group-select.tsx`：名称映射运行时校验。
- 本工作项新建的 `web/src/features/usage-logs/components/__tests__/log-group-selection.test.tsx`：11 项边界回归。
- 本工作记录：两项独审发现、方案、红绿证据和模拟边界。

未修改共享翻译、README、custom-development 或其它 worker 文件；最终全量类型检查、
双模板构建由协调者统一执行。最终结果见下节。

## 最终独立复审与集成门禁（2026-11-16）

- 原 reviewer `ctx_608ae81b5cfc` 已复审通过，确认 `groupDetailsQueryKey` 保存失效和
  `Object.hasOwn` 非空字符串映射两项 P2 闭环，无新增阻断。独立 6 文件 40 项耗时
  46.21 秒通过；原失败 edge fixture 现 2 项通过，包含 `__proto__`。
- 复审覆盖真实 `GroupRatioForm`/QueryClient 保存链路、活动观察者与普通用户 self 范围；
  GET/PUT 网络仍为 mock，不等于真实认证、数据库、MySQL/PostgreSQL 或生产权限验收。
  Classic 原有名称展示和稳定标识提交合同保留。
- 统一结果见[总交付记录](16_issue401_delivery.md)：Default build、Classic build、
  Default typecheck 均 exit 0；Classic 没有单独 typecheck 结果，保留既有 chunk 体积警告。
  未提交、推送、部署或关闭 Issue，MCP 改动不纳入本项。

## 新一轮独审 P3：结构共享丢失合法历史名称

### 编码前计划

2026-11-16，新独立 reviewer `ctx_1d278d2d3bc6` 的仓库外夹具确认：
合法授权 alias `__proto__` 首次显示当前名称，但改名并失效查询后，
TanStack Query 默认结构共享重建普通对象，赋值该键没有创建自有属性，
缓存丢失名称并回退原标识。组件已有 `Object.hasOwn` 防护正确，应继续保留。

本轮只修改 `use-log-group-options.ts`、该 hook 专属测试及本工作记录。
计划仅对这个组选项查询设置 `structuralSharing: false`，使每次成功响应保留完整
自有名称映射；不改变全局 QueryClient、依赖、查询键、失效前缀、staleTime、
API、合法选项、稳定 ID/code/alias 提交或权限。代价是本查询成功重取时不再复用
相同子对象引用，可能多一次观察者渲染；组选项数据量有限，没有新增网络请求。
相比引入 Map 和修改展示接口，此方案保留现有响应和消费结构，变更范围更小。

先用真实 hook、QueryClient 和 Axios 客户端，仅模拟网络适配层，分别验证管理与 self
首次加载后改名、失效、再次获得当前名称；覆盖 `old-public`、`__proto__`、
`constructor`、`toString`，检查稳定 code 和按角色请求范围。先确认失败再实现修复，
随后运行已有名称选择、Hide、故障重试和保存刷新定向回归。每个测试子进程
最多 60 秒，TEMP/TMP 使用 D 盘并设置 Node 兼容参数，不运行全构建或提交部署。
原 reviewer 夹具只读保留，交付后由新 reviewer 独立复跑。

该问题限于 Default 新增查询缓存链路；Classic 不使用此 hook 或 QueryClient，
仍按原 data 展示当前名称并提交 code，本轮不改其源码和共享契约。

### 实现与红绿证据

按计划仅在 `useLogGroupOptions` 查询选项增加 `structuralSharing: false` 和中文原因注释；
没有改动组件、自有属性校验、全局 QueryClient 或依赖。每次成功重取直接保留该响应的
完整名称映射，选项仍只来自原 `data`，名称映射不参与授权或修改筛选提交值。
这不改变缓存复用、请求去重、刷新时机或用户/session 隔离，只放弃成功更新时的
深层引用复用。没有新增持久化数据、API 字段、翻译或共享文档变更。

hook 专属测试新增 8 项：管理/self 两种范围，分别以 `old-public`、`__proto__`、
`constructor`、`toString` 作为合法历史 alias。真实 QueryClient、hook、Axios 客户端
及拦截器执行首次加载 → 改名 → 原业务前缀失效 → 再次加载；仅替换网络 adapter。
断言首次和再次返回的自有名称、当前选项值仍为 `public`、请求只使用对应管理/self
接口且没有额外请求。组件可见展示及实际选择提交由同批既有组件回归覆盖。

最初把四种 alias 合放同一映射时测试意外通过；核对依赖的 `isPlainObject` 后确认，
自有字符串 `constructor` 会影响普通对象判定并掩盖 `__proto__` 的复制缺陷。
因此改为每个 alias 独立 fixture，未修改生产实现即得到稳定红测：

- `red-isolated.log`：hook 10 项中 2 失败、8 通过，exit 1，3.11 秒。
  两项失败均为 `__proto__` 在管理/self 改名刷新后失去自有属性。
- `green.log`：修复后 4 文件 34 项通过，exit 0，28.55 秒。包含 hook 10 项、
  选择器 19 项、FilterBar 2 项和真实保存链路 3 项；覆盖故障重试、StrictMode、
  Hide、未知/非法名称回退、稳定标识、管理员保存刷新和普通用户范围。
- `final-hook.log`：格式化及将测试非空断言改为安全读取后，hook 10 项再次通过，
  exit 0，3.54 秒。两个修改文件的定向 oxfmt 和 oxlint 均通过。

日志和运行脚本位于 `D:/temp/issue401-log-group-p3/`；`run-tests.cjs` 对每个 Vitest
子进程使用 `spawnSync timeout: 60000`，同时设置 D 盘 TEMP/TMP、
`NODE_OPTIONS=--no-experimental-webstorage`、`--maxWorkers=1 --testTimeout=60000`。
在 `web` 目录执行：

```powershell
node D:/temp/issue401-log-group-p3/run-tests.cjs green src/features/usage-logs/hooks/__tests__/use-log-group-options.test.tsx src/features/usage-logs/components/__tests__/log-group-selection.test.tsx src/features/usage-logs/components/__tests__/filter-autofill.test.tsx src/features/system-settings/models/__tests__/group-ratio-log-cache.test.tsx
node D:/temp/issue401-log-group-p3/run-tests.cjs final-hook src/features/usage-logs/hooks/__tests__/use-log-group-options.test.tsx
```

本轮仅修改 hook、hook 专属测试、本记录三个文件，没有新增仓库文件。
原 reviewer 的 `edge.test.tsx`、`edge.config.mjs`、报告及失败证据保持原样；
实现阶段交协调者安排独立复跑，实施者不自行签署独审；正式复验结果追加如下。
本轮无新增浏览器、后端实库或生产验证；没有运行全构建、提交或部署。
Classic 源码只读核对仍使用组件内请求和状态，不经过本次结构共享链路；
本轮没有重跑 Classic，前文构建/测试结果属于对应历史轮次。

## P3 最终独立复验（2026-11-16）

- reviewer `ctx_434aefa8fec0` 正式通过，报告 `D:/Temp/issue401-fresh-review-groups/recheck-report.md`：原未改 edge 夹具 SHA256 保持一致，4/4，exit 0，6.17 秒；四文件 34/34，exit 0，24.31 秒，原 `__proto__` 名称丢失 P3 闭环。
- 确认关闭结构共享仅作用于当前查询，admin/self、主体/session/role/group 隔离、请求和取消信号不变，Object.hasOwn、稳定 code/alias 提交及 Hide 回归通过，无新授权、请求循环或隐私回归。首次从夹具目录启动因 setup 路径失败为 0 项执行，改回原仓库根目录后通过；未改夹具，此失败不计产品回归。
- 真实 QueryClient/组件使用模拟网络，未新增浏览器、Go、真实认证、数据库、Redis 多实例或生产验证。所有测试子进程限 60 秒且使用 D 盘 TEMP/TMP 和进程级 NODE_OPTIONS。
- 用户已授权提交推送 dev、部署 zzapi 并切换 Default，当前三项新审查缺陷均独立闭环，但截至此段登记提交、发布构建和部署均尚未执行；结果由[部署专题](16_issue401_dev_zzapi_deployment.md)登记，不把早期构建当作最新返修产物。
