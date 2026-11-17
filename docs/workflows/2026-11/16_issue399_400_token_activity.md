# Issues #399 / #400：Default 令牌状态与实时活动

## 目标、边界与根因

基线为 `13f5fdbb9`，已完整读取 #399 与 #400 正文。Default 的令牌状态列已有 URL 筛选，但请求、queryKey 与搜索分支遗漏 status；默认表格也没有本人活动摘要和 active_only 搜索。Classic 已实现相应接口契约，作为只读参照。

修改 `web/src/features/keys/` 中 API、类型、表格、列、活动模块及测试，以及协调者后续明确授权的 `web/src/routes/_authenticated/keys/index.tsx`。不修改功能入口 index.tsx、主要操作按钮、API 地址组件、共享 usage-logs hook、后端和已有 MCP 改动。共享文档与 locale 由协调者集成。

进一步核查发现，共享名称/密钥防抖输入在浏览器后退时可能将旧草稿写回 URL。因此新增令牌专用搜索输入，在 URL 改变时取消旧草稿，保留输入法组合输入，未修改共享 hook。

## 实施方案与契约

- `/api/token/search` 合并 keyword、token、status、active_only、p、size；以完整条件及登录身份隔离查询缓存。
- 状态由服务端判定，结果和 total 由服务端统一分页，禁止再次在客户端按原始 status 丢弃行。
- 本人活动固定使用 `/api/log/self/active`，管理员也不查询其他用户；用 summary.by_token 的稳定 ID 标记令牌，不能从当前连接预览页或令牌名称推算。
- ended 的 5 秒保留记录不计活动摘要；仅活动令牌集合变化及错误恢复时刷新活动筛选列表。
- 2 秒轮询、10 秒请求截止、页面隐藏停止/取消、卸载取消；身份变化立即隔离状态。错误保留最后快照并明确陈旧，恢复后刷新。
- 桌面及手机共用状态来源；稳定摘要布局与 ID 徽标。无数据库或计费、密钥权限变更。
- 路由继续接受原 page、pageSize、status 数组、filter、token，新增 `active_only: z.boolean().optional().catch(false)`；实际路由直接使用提取的 `apiKeySearchSchema`，组件测试复用同一 schema。开关、状态、名称和密钥变更回到第一页；重置一次清空所有条件；刷新及浏览器后退恢复 URL。
- 查询 key 为身份、会话、完整筛选、分页和刷新标识的组合。仅同一身份与筛选下翻页保留旧数据；筛选及账号切换不会短暂显示旧结果。活动变动导致总页数缩小时回到最后有效页，total 为零回到第一页。
- 活动请求失败转换为局部 Error，避免全局 Axios 500 处理器将整个页面重定向；首次失败不显示伪造零值。活动快照超过 5 秒未更新、发生错误或页面隐藏均标为陈旧；正常轮询不替换已有摘要为加载行。

## 测试计划

先建立请求/组件失败用例，再实现；验证 status 与 active_only 组合、URL 恢复、分页归位、身份隔离、重名 ID、结束保留、错误恢复、隐藏及卸载。每轮前端测试截止 60 秒，使用现有后端测试核验跨页 total 及本人隔离；尽可能通过实际组件浏览器验证，明确模拟接口与真实本地后端的边界。

## 验证结果

### 自动化与构建

- 请求测试先复现 5 个失败、1 个通过，确认 status 和 active_only 未进入请求；实现后全部通过。
- Default 新增 3 个测试文件、共 21 项通过，最终格式化后复跑耗时 38.52 秒，覆盖四种状态请求、组合编码、真实路由与表格交互、桌面/手机、URL 恢复、清除/重置、越界归位、重名 ID、本人接口、身份及缓存隔离、结束保留排除、错误/恢复、隐藏/卸载取消、活动集合定向刷新。API 使用 mock，路由、QueryClient 和表格组件真实运行。
- Go 原有契约回归共 58 项通过，涉及 model、controller、router、pkg/activerequests 四个包。使用真实本地 Go/GORM 测试夹具验证先过滤后分页、跨页 total、状态与名称/密钥/活动 ID 交集、普通用户隔离、外部 user_id 不改变本人范围、结束记录 5 秒保留不计活动、密钥脱敏。未连接生产服务，也未运行 MySQL/PostgreSQL 实库矩阵。
- Default 独立目录构建通过，耗时 13.2 秒；未覆盖共享 web/dist。该构建包含核心交互实现，早于后续局部 Axios 错误归一化和最终格式化；后续源代码由针对性回归验证，最终集成构建交协调者统一执行。
- 曾在并行修改较少的检查点执行 `bun run typecheck` 成功；后续全量检查出现其他 worker 正在处理的 affiliate-management 和 usage-logs 文件错误。协调者要求不重复全量检查，最终全量类型检查交集成阶段执行。
- 范围内 13 个 TS/TSX 文件使用 oxfmt 格式化并恢复版权头，oxlint 通过；不调用全仓格式化脚本。

可复现命令（在对应目录执行，外层进程限制 60 秒）：

```sh
# web/
node --no-experimental-webstorage node_modules/vitest/vitest.mjs run src/features/keys/__tests__/api-filters.test.ts src/features/keys/hooks/__tests__/token-activity.test.tsx src/features/keys/components/__tests__/token-filters.test.tsx --maxWorkers=2 --testTimeout=60000 --hookTimeout=60000

# 仓库根目录
go test ./model ./controller ./router ./pkg/activerequests -run 'Test(SearchUserTokens|SearchTokens|GetSelfActiveRequests|GetAdminActiveRequests|AdminActiveRequestsRoute|RegistryIsolatesUsersAndCountsTokensAcrossPages|FinishedRequestExpiresAtFiveSeconds|ActiveRequestsPrecedeRetainedEntries)' -timeout 60s -count=1
```

### 浏览器证据及双模板边界

- Playwright/Chromium 实际加载上述 Default 构建产物，在 1440px 和 390px 验证状态筛选请求、跨页命中、两个同名令牌的不同 ID 徽标、active_only 开启与刷新恢复、503 故障保留旧状态及重试恢复。两种尺寸均无页面异常、无横向溢出，截图已人工查看。
- 浏览器使用模拟 `/api/**`：22 个令牌中只有第 21、22 个禁用；活动摘要为 2 个令牌、3 个请求，同时包含结束保留的预览记录。实测发出的请求含 `status=2&active_only=true&p=1&size=20`，只按服务端筛选结果展示；没有以浏览器 mock 冒充真实后端验收。
- 本机证据目录：`D:/Temp/newapi-issue399-400-20261116/`，包含 `browser_check.py`、`result.json`、`token-1440.png`、`token-390.png` 与独立 dist。此目录为本机临时产物，不作为仓库交付文件。
- Classic 页面、useTokensData 和活动组件已核对为既有能力，本次不修改。尝试运行其 tokens/activity 兼容回归时，Node 26 默认 Web Storage 导致 localStorage 兼容错误；改用进程 `NODE_OPTIONS=--no-experimental-webstorage` 后，组合运行及 tokens 单文件仍在收集阶段达到外层 60 秒截止，未取得有效通过结果。不能将这些中止记录报告为 Classic 回归通过；共享后端 58 项契约回归有效。
- 当前 C 盘空间耗尽，后续命令使用 D 盘独占临时目录和进程 TEMP/TMP，未清理共享缓存。完整 keys 目录早期尝试存在其他旧测试的 localStorage 清理级联失败，未作为本项通过依据。

### 交付文件

修改：

- `web/src/features/keys/api.ts`
- `web/src/features/keys/types.ts`
- `web/src/features/keys/components/api-keys-columns.tsx`
- `web/src/features/keys/components/api-keys-table.tsx`
- `web/src/routes/_authenticated/keys/index.tsx`

新增：

- `web/src/features/keys/components/api-key-search-input.tsx`
- `web/src/features/keys/components/token-activity.tsx`
- `web/src/features/keys/hooks/use-token-activity.ts`
- `web/src/features/keys/hooks/use-api-keys-query.ts`
- `web/src/features/keys/lib/api-key-search.ts`
- `web/src/features/keys/__tests__/api-filters.test.ts`
- `web/src/features/keys/hooks/__tests__/token-activity.test.tsx`
- `web/src/features/keys/components/__tests__/token-filters.test.tsx`
- 本工作记录。

同目录下 API 地址组件、其测试和主要按钮改动属于其他 worker，不计入本项交付。

## 收尾集成状态

- 路由：已获授权自行完成，无待串行集成的路由字段；未修改 routeTree.gen.ts。
- 文档：开发 README、active-requests.md 与 custom-development.md 已登记双模板令牌活动能力，并链接本轮总交付记录。
- 翻译：下列使用键已核对全部 7 个 Default locale 的 `translation` 对象存在，无需本 worker 修改共享 JSON：`Token activity`、`{{count}} tokens in use`、`{{count}} active requests`、`Connection status is unavailable.`、`Loading connection status…`、`Stale`、`Only active tokens`、`Current instance only`、`Showing the last known state.`、`Retry`。英文已有计数字符串会出现 `1 active requests`，可在后续使用 i18next 复数键完善，本次保持已有契约。
- 最终 Default typecheck、Default build、Classic build、独立复审和 Classic 动态回归结果已由协调者登记在[总交付记录](16_issue401_delivery.md)；本实现 worker 不将其改写为自身审查结论。

## 双模板与分享

Default 为修复及补齐范围；Classic 保持既有实现，已完成共享后端契约验证，动态组件回归受上述环境和时限约束未完成。共享 API 没有新增字段或行为，其他系统无需修改；未改变鉴权、完整密钥读取、计费或业务请求语义。

- NewAPIForDouDi：建议在其确实使用同源 Default 令牌页时评估分享服务端筛选和 URL 恢复修复；实时活动部分需先确认本人摘要及活动过滤 API。
- NewAPIModifyByGang：建议作为可选迁移参考，先核对模板及后端 registry 能力；仅有 UI 时不能获得可靠跨页活动 total，不建议直接搬用。

风险：活动是观察快照，轮询与列表查询之间可能发生请求开始/结束，因此计数与列表短暂不同；跨节点准确性取决于后端 scope，instance 模式明确提示。未部署、提交、推送或向兄弟仓库发送消息。

## 独立审查返修及追加分组展示（2026-11-16）

- P2 已复现：自身 URL 值为空时，未提交名称或密钥草稿不会因 Reset 或仅改变其他字段的历史恢复而取消，500ms 后可能重新污染 URL。先新增真实表格回归，再通过显式重置与历史导航事件取消草稿；普通字段提交不应取消另一个输入的草稿，输入法组合输入必须保留。
- 协调者追加授权 `components/api-key-group-cell.tsx` 及专属新测试：核对后端 `group_details`，以稳定 ID 对应当前显示名称，覆盖单组、多组、tooltip、改名及无法解析的回退；不改变 Auto、倍率和提交契约，不逐行查询。
- 仅进行范围内回归，每次运行截止 60 秒，临时目录使用 D 盘；不重复全量构建。最终交原独立审查者复审，共享文档更新交协调者集成。

### Reset 与历史恢复修复

- 在真实表格测试中先新增名称/密钥各自的 Reset、BACK、FORWARD 共 6 个回归，全部先失败；同时验证组合输入用例通过。每个失败用例都在输入 pending 后立即操作，先断言清空输入，再等待 550ms 断言 URL 与 API 不出现 pending，没有先等防抖提交。
- 表格新增 `searchResetVersion`：显式 Reset 递增；订阅实际 router.history 的 BACK/FORWARD/GO 恢复事件递增。两个输入以各自字段名和该版本作为 React key，卸载时清理旧定时器，并同时重建输入法状态。正常 PUSH/REPLACE 不触发重建，避免名称、密钥或活动条件更新自动丢弃另一输入的草稿。
- 原有输入组件仍负责自身 URL 值变化时清理草稿；显式重置和历史恢复不再依赖该值变化。新增交错输入测试覆盖两个普通输入先后提交；输入法测试覆盖组合中暂不提交、另一个字段及活动开关更新后保留草稿、结束组合后提交。

### 分组名称展示修复

- 已核对 `model/group_bindings.go` 的 `GroupReference`（id/code/name/exclusive）和令牌批量加载关联：`group_details` 提供当前名称，`group_ids` 为稳定关联，旧 group 字符串可能仍是历史 alias。既有 `ApiKey` schema 已包含相应字段，无需扩展后端或类型。
- `api-keys-columns.tsx` 把 group_ids/group_details 传入单元格；原本只抽取倍率的 `/api/user/self/groups` 查询同时保留名称映射供兜底，没有新增逐行请求。
- `api-key-group-cell.tsx` 在标识与 ID 数量完整对应时优先按稳定 ID 匹配详情名称，再使用批量分组映射。无 ID 的旧数据按 code 匹配；详情缺名或已删除、不可解析才回退原标识。ID 已知但不匹配时不会以 code 错贴其他组的名称，绑定数量不完整时不按位置猜测。
- 单组、多组徽标与悬浮提示统一展示名称，不额外追加 code；保留原始 group 供颜色、倍率查找等既有行为使用。Auto、Cross-group、空值继承 User Group 和倍率逻辑没有改变，不写回 ID/code，不修改提交或业务匹配。
- 新增 `components/__tests__/api-key-group-display.test.tsx`，覆盖名称不等于 code、稳定 ID 解析旧 alias、详情无序、多组、改名后提示更新、批量映射兜底、删除/未知标识、ID 冲突及空继承。真实表格另验证查询失效后的名称刷新及原 ID/code 保持；先执行时 5 个展示相关用例失败，再实现修复。
- 本轮分组展示修改桌面原有分组列；Default 手机列表原本不展示该列，因此未新增手机分组模块。Classic 已有基于 group_details 的名称徽标与多组 tooltip，本轮未修改其代码。

### 本轮交付与验证边界

- 最终 5 个测试文件共 **39 项通过**，耗时 **18.36 秒**；5 个本轮 TSX 文件 oxfmt 与 oxlint 通过。测试命令如下，外层 `subprocess.run(..., timeout=60)`，进程 TEMP/TMP 指向 `D:/Temp/newapi-issue399-400-20261116`，NODE_OPTIONS 为 `--no-experimental-webstorage`：

  ```sh
  node node_modules/vitest/vitest.mjs run src/features/keys/components/__tests__/token-filters.test.tsx src/features/keys/components/__tests__/api-key-group-display.test.tsx src/features/keys/components/__tests__/api-key-group-cell.test.tsx src/features/keys/hooks/__tests__/token-activity.test.tsx src/features/keys/__tests__/api-filters.test.ts --maxWorkers=2 --testTimeout=60000 --hookTimeout=60000
  ```

- 本轮修改 `api-keys-table.tsx`、`api-keys-columns.tsx`、`api-key-group-cell.tsx`、`token-filters.test.tsx`，新增 `api-key-group-display.test.tsx`，并更新本记录。其余原任务文件和 reviewer 位于 D 盘的证据未修改。
- 针对性测试使用真实 React/TanStack Router/QueryClient 和组件，API mock；不等于浏览器连真实后端验收。原浏览器截图和原构建为返修之前的证据，本轮没有将它们复用为返修通过结论，也未重复全量构建。
- 原有 `api-key-group-cell.test.tsx` 4 项 Auto/倍率/减弱动画回归一并执行。Tooltip 在现有 Base UI 包装下未设置 tooltip role，新测试实际触发悬浮并断言弹层内容，不 mock 掉展示组件。
- 本轮没有新增文案或 locale key。共享翻译和能力记录已补充“Reset/历史恢复取消未提交草稿”和
  “Default 令牌列表按稳定 ID 显示当前分组名称”两项返修结果。
- 原独立 reviewer `ctx_9682f8bd635f` 复审确认 Reset、名称和历史恢复两个 P2 通过；原失败
  1 项已通过，实施回归 39 项通过，额外 `history.go` 与批量 fallback 3 项通过。F3 浏览器
  fixture 的桌面单/多组、改名 alias 按 ID、保存刷新稳定提交通过；手机编辑器通过，列表原本
  没有分组字段不适用；自定义 tooltip 浮层未观察到，不能写作已通过。
- 证据来自组件/fixture 与 API mock：不等于生产认证、真实 Redis 多节点、MySQL/PostgreSQL
  或完整 E2E。未读取完整令牌密钥；统一门禁见[总交付记录](16_issue401_delivery.md)，
  未提交、推送、部署或发送外部消息。

## 新一轮独立审查：首个列表请求在途时丢失活动刷新

- 2026-11-16 新 reviewer 的只读夹具确认：首次无缓存列表尚未返回时，活动集合由 31 变为 32；invalidateQueries 复用旧 promise，旧响应完成后清除失效标记，而 hook 已记录新 ID，后续相同摘要不再查询，列表永久停在 31。
- 本轮仅授权修改 `hooks/use-api-keys-query.ts`、专属 hook 回归及本记录，不修改 reviewer 夹具、共享文档、其他源码或 MCP，不提交、部署或全量构建。
- 计划先用真实两个 hooks 与 QueryClient 建立可控延迟响应回归，再按完整查询上下文记录待刷新需求；在途变化合并，等待当前请求结束后补查一次，切换查询上下文或卸载时不遗留跨主体补查。验证无界重查、故障恢复、主体/会话切换取消、卸载取消及普通列表请求数量不变。
- 测试子进程每次限制 60 秒，使用 D 盘 TEMP/TMP 与 `NODE_OPTIONS=--no-experimental-webstorage`；先红后绿及最终定向回归结果随后补入。

### 本轮修复与真实验证结果

- 新增 `web/src/features/keys/hooks/__tests__/api-keys-activity-race.test.tsx`，使用真实 `useTokenActivity`、`useApiKeysQuery` 和 QueryClient，仅 mock API 边界，以可控 promise 精确安排无缓存列表和活动摘要的先后顺序。先红：连续活动变化与故障后同集合恢复 2 项失败（只发 1 个旧列表请求），生命周期与普通列表 4 项通过，耗时 5.33 秒。
- 修复在 `previous` 中保留 pending；活动变化和故障恢复只更新待补查状态，`query.isFetching` 结束后才消费 pending 并失效查询。多个变化合并为一次补查；待刷新状态按用户、会话、完整筛选、分页和刷新标识隔离。没有新增定时任务或异步回调链；卸载、换主体和换会话由现有 QueryClient 观察者及已消费的 AbortSignal 取消请求，旧响应不能触发旧主体补查。
- 新增第 7 项验证旧请求和补查均失败时显示错误，同集合轮询不形成无界重试。最终针对性回归 **4 文件 37 项全部通过，17.98 秒**，包含新竞态 7 项、既有活动生命周期 8 项、API 条件 6 项、真实表格交互 16 项。
- `bun run typecheck` 通过；本轮两个 TS/TSX 文件 oxfmt 与 oxlint 通过，范围 `git diff --check` 通过。本次没有全量构建、提交或部署，没有修改 reviewer 的夹具和日志；旧失败夹具交新 reviewer 独立复跑。

最终定向命令（`web/`，外层 Python `subprocess.run(..., timeout=60)`，D 盘 TEMP/TMP，Node 兼容参数）：

```sh
node node_modules/vitest/vitest.mjs run src/features/keys/hooks/__tests__/api-keys-activity-race.test.tsx src/features/keys/hooks/__tests__/token-activity.test.tsx src/features/keys/__tests__/api-filters.test.ts src/features/keys/components/__tests__/token-filters.test.tsx --maxWorkers=2 --testTimeout=60000 --hookTimeout=60000
```

本次仅修改 hook、新增该测试及本专题记录。没有新增文案、翻译或 API 字段；Classic 与后端契约不变。此证据是 hook/组件与模拟 HTTP 边界验证，不等于浏览器、生产或真实多节点验收。实现交接时独立复审尚待执行；以下记录追加正式复验结论，已获授权的提交部署仍由协调者执行。

## 新一轮最终独立复验（2026-11-16）

- reviewer `ctx_b066fbb302a4` 正式通过本范围：原未改 `activity-race.test.tsx` 从红转绿；与地址原失败夹具组合 2/2，exit 0，4.36 秒。活动四文件 37/37，exit 0，28.81 秒；原活动夹具与 SID 隔离额外夹具组合 2/2，4.43 秒。报告 `D:/Temp/issue401-fresh-review-keys/review-report.md` 保留初始失败与最终复验，原夹具 SHA256 未变。
- 复验覆盖首次无 data、连续活动变化合并、错误恢复、普通列表不增加请求、用户/SID/卸载取消及旧请求与补查失败不循环；本范围无剩余已证实 P1/P2。
- 证据是真实 hooks/QueryClient/组件加 mock 传输，不新增真实浏览器、Go、实库、Redis 多实例或生产验收；历史构建不代表最新返修产物。用户已授权 dev 提交推送及 zzapi/Default 部署切换，但截至此段登记尚未执行，后续见[部署专题](16_issue401_dev_zzapi_deployment.md)。
