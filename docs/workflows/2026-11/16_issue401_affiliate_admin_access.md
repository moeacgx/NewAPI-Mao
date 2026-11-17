# Issue #401：普通管理员返佣业务入口

## 目标与根因

Default 管理员侧栏的返佣入口指向 `/system-settings/billing/affiliate`，其父路由仅允许 role 100；业务接口 `/api/affiliate/admin/*` 实际允许 role >= 10，因此普通管理员点击入口会收到 403。

## 编码前方案与范围

- 增加独立 `/affiliate-management` 业务路由，按现有管理员路由模式拒绝未登录和 role < 10。
- 提取既有返佣邀请、记录、提现、风险、申请与异常检测业务组件供新页面使用；Root 参数表单保留在系统设置中。
- 侧栏返佣管理入口指向新路由并明确 `requiredRole: ROLE.ADMIN`，保留同文件其他并发修改。
- 不修改系统设置父路由、后端权限或配置 API，不手工修改生成路由树。
- 修改既有设置组件与 Classic 文件前向协调者申报范围；共享翻译、文档索引与路由树由协调者串行集成。

## 接口与安全契约

- 新页面沿用 `/api/affiliate/admin/*` 与管理员用户搜索接口；不得读取或写入 `/api/option/`，不得挂载 Root 参数表单。
- 返佣金额、账务、权限、审核操作及后端数据模型保持原合同，无数据库迁移。
- Root 配置入口保持 `/system-settings/billing/affiliate`，仅 role 100 可进入；普通用户不可进入业务管理。
- Classic `/console/affiliate-admin` 已允许 role >= 10，但引用 `PaymentSetting onlyAffiliate`，该组件会读取 Root option 并包含支付合规确认，需核查实际影响并单独报告。

## 验证计划

- 先以回归测试复现普通管理员入口错误，再验证 role 1/10/100 的业务与 Root 配置边界。
- 挂载真实业务组件并拦截 API 边界，验证仅发起业务请求，无 option 请求、无可保存的 Root 表单。
- 验证侧栏入口权限、空列表和业务标签页切换；类型检查、定向 lint、格式检查与 `git diff --check`。
- 每次测试最多运行 60 秒；路由树生成与浏览器验证需要协调者串行处理时明确记录限制。

## 实现与双模板影响

- Default 新路由 `/_authenticated/affiliate-management/` 对外为 `/affiliate-management`，独立校验 role >= 10；侧栏明确管理员角色，并通过 `configUrls` 复用原 `affiliate_admin` 模块开关。
- `features/affiliate/admin-panel.tsx` 从原设置组件提取七个业务标签页，使用 Card 容器；`management.tsx` 使用 Default 原生 `SectionPageLayout`。业务面板不导入配置 hooks、不挂载参数表单。
- 原 `affiliate-settings-section.tsx` 只保留返佣规则与反欺诈参数，原 Root 父路由和保存 API 均不变。业务区原本在 Form 外使用的 `FormLabel` 改为带 `htmlFor` 的普通 `Label`，避免手动绑定和风控页依赖不存在的表单上下文。
- Classic 保留 `/console/affiliate-admin` 的 `AdminRoute`。`PaymentSetting` 仅对 `onlyAffiliate && !isRoot()` 分支挂载业务组件，普通用户直接拒绝；Root 和一般支付设置继续挂载原配置组件及合规确认。
- Classic `SettingsAffiliateCommission businessOnly` 默认显示用户邀请，隐藏返佣规则与反欺诈配置面板，并在保存函数中拒绝业务模式保存。
- 已核对 `router/api-router.go`、`controller/affiliate.go`、返佣 controller/model 文件中的合规调用：管理业务沿用 AdminAuth，未发现 `requirePaymentCompliance` / `IsPaymentComplianceConfirmed` 管理操作门禁；Root option、充值和订阅等原有合规检查不变。本修复不改变返佣发放资格、用户协议、风控确认和任何后端业务校验。
- 新页面全部复用既有翻译键，本轮没有新增翻译键；未修改共享 locales。

## 精确验证证据

- 修复前 `admin-navigation.test.tsx` 两例失败，实际入口仍为 `/system-settings/billing/affiliate` 且缺少 `requiredRole`；修复后该文件三例通过（含原模块开关关闭时隐藏入口），耗时 2.79 秒。
- `admin-routes.test.tsx` 四例通过，覆盖未登录及 role 1/10/100：仅 role >= 10 可进入业务管理，只有 role 100 可进入系统设置，耗时 47.36 秒。
- `admin-panel.test.tsx` 两例通过：空邀请列表、全部业务标签页切换、六个原业务 GET、无 Root 表单/option 请求；提现审核取消不提交、确认后调用 `/api/affiliate/admin/withdrawals/17/approve`。
- `admin-settings.test.tsx` 一例通过：Root 配置将三分钟保存为 180 秒，无业务列表 GET，反欺诈配置仍可保存。与业务面板测试一起运行耗时 48.88 秒，合计三例通过。
- Classic `affiliate-admin-access.compat.test.jsx` 三例通过，耗时 17.42 秒：role 10 无 option 请求/Root 配置，可查询业务并在真实确认弹窗中批准提现；role 1 无任何请求且跳转禁止页；role 100 保留 option 加载、配置标签及未确认合规门禁。测试使用真实 `AdminRoute`、`PaymentSetting`、返佣组件及 Semi UI，仅模拟 Axios API 响应。
- Default 定向 oxlint、Classic 定向 ESLint 无错误；Classic 三文件 Prettier 检查通过。Default 使用现有 oxfmt，按仓库保护版权头的机制只格式化本任务八个文件，未运行会写整个目录的全量格式脚本。
- 全量 `bun run typecheck` 曾返回五个本任务测试错误（Testing Library 的 ByRoleOptions 不接受 `exact`），已删除这些多余参数；协调者随后要求停止重复全量检查，最终全量 typecheck/build 由协调者统一执行，不能把此前失败当作通过。
- C 盘空间耗尽导致 Classic 首轮 ENOSPC；随后仅在测试子进程设置 D 盘 TEMP/TMP。Node 26 使用进程级 `NODE_OPTIONS=--no-experimental-webstorage`，Default 使用自身 Vitest 4，Classic 使用自身 Vitest 2，避免混用依赖。

测试复现命令（在对应模板目录执行，外层进程限定 60 秒）：

```powershell
# Default，可按上面的测试文件分批运行，避免并发构建时超时。
$env:NODE_OPTIONS = '--no-experimental-webstorage'
node node_modules/vitest/vitest.mjs run src/features/affiliate/__tests__/admin-navigation.test.tsx --maxWorkers=1
node node_modules/vitest/vitest.mjs run src/features/affiliate/__tests__/admin-routes.test.tsx --maxWorkers=1
node node_modules/vitest/vitest.mjs run src/features/affiliate/__tests__/admin-settings.test.tsx src/features/affiliate/__tests__/admin-panel.test.tsx --maxWorkers=1

# Classic，使用自己的 Vitest 2。
node node_modules/vitest/vitest.mjs run src/components/settings/__tests__/affiliate-admin-access.compat.test.jsx --pool=forks --poolOptions.forks.singleFork
```

## 浏览器验证与模拟边界

- Orca 内置浏览器访问本地 `http://127.0.0.1:5173/affiliate-management`，模拟完整 role 10 AuthBundle 和空业务列表；真实构建页面显示七个业务标签页，点击手动推荐后表单可见，无 rules/anti-fraud 配置标签。
- 同一模拟普通管理员访问 `/system-settings/billing/affiliate`，实际跳转 `/403`。网络记录显示六类返佣业务 GET；无生产 API 调用。
- 截图证据：`.playwright-cli/affiliate-temp/default-admin.png`；测试 fixture、日志也存于该任务专属目录，不作为产品代码提交。
- 模拟覆盖前端路由、渲染和请求契约，不代表真实后端登录、真实提现审核或线上权限验收。Classic 使用组件交互测试，未做完整浏览器验收。
- 本轮模拟路由已解除，任务浏览器标签已关闭。

## 协调者串行集成事项

### 独立审查结果

另一名只读审查工人完成了业务组件拆分前后的结构对照、角色与模块开关、Root 配置保存以及 Classic 合规边界核查，未发现本子项新增 P1/P2 或阻断缺陷。独立复跑 Default 四个文件 10 项测试通过（20.46 秒），Classic 三项组件测试通过（19.74 秒），每个进程均限制在 60 秒内，定向差异检查通过。

上述证据中，Default 路由测试直接调用真实 `beforeLoad` 守卫；组件及 Classic MemoryRouter 使用模拟 API。Root 配置回归验证三分钟保存为 180 秒及反欺诈保存入口，不代表逐项执行全部风控、扫描和配置保存。后端权限仅作源码核对，没有执行真实提现审核或生产权限操作。该审查结论仅覆盖返佣拆分，统一门禁结果见下节。

## 最终独立复审与集成门禁（2026-11-16）

- 独立 reviewer `ctx_cbdb5e0bca8c` 复审 Default 10 项、Classic 3 项，全部通过且无 P1/P2
  或阻断；业务组件提取、角色入口与 Root 配置边界保持。
- D 盘最终结果中 Default build exit 0（10.61 秒）、Classic build exit 0（46.72 秒）、
  Default typecheck exit 0（12.24 秒）；Classic 没有单独 typecheck 结果，仅有既有 chunk
  体积警告。详细结果见[总交付记录](16_issue401_delivery.md)。
- 角色 10/100 的组件与路由 mock、AdminAuth 源码和提现事务调用已覆盖；真实认证、生产权限、
  MySQL/PostgreSQL 实库和完整业务 E2E 未验证。未提交、推送、部署或关闭 Issue。

- `routeTree.gen.ts` 没有由本工人修改；工作期间已有开发服务器自动识别新路由，当前生成树已包含 `/affiliate-management`。协调者核对生成差异并保留 MCP；若需重建，在 `web/` 执行 `bun run build`，由 `rsbuild.config.ts` 中 `tanstackRouter` 插件生成，再执行 `bun run typecheck`。
- 将本记录登记到 `docs/developer/README.md`；`custom-development.md` 更新管理员返佣业务与 Root 配置拆分的入口和权限边界。共享文档不由本工人修改。
- 独立审查须由另一名工人执行，重点审查组件提取是否遗漏业务分支、Root 配置保存和 Classic 合规边界。
- 后端 API、计费、认证和数据库契约未改变；本轮不需要通知其他项目。`NewAPIForDouDi` 与 `NewAPIModifyByGang` 仅在同样存在管理员业务复用 Root 设置页时值得参考该权限拆分，不建议直接移植本项目 UI；本轮不向兄弟仓库发送任何消息。
