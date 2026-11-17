# Default 令牌页 API 地址入口（Issue #402）

## 目标、根因与范围

2026-11-16 基于 `13f5fdbb9` 核对 Issue #402：Default 的 `ApiKeysPrimaryButtons`
仅提供创建按钮，缺少 Classic `TokenApiInfoPopover` 的地址入口。
历史记录 `02_classic_token_search_api_info.md` 中“新版 Default 已有 API 地址入口”
与当前源码不一致，本次以源码和回归测试为准。

本工人仅修改 Default 顶部操作按钮、新建地址组件及专属测试、本记录。
不改令牌表格、活动统计、密钥接口、共享翻译文件和其他工人的 MCP 改动。

## 方案与契约

- 使用项目原生 Base UI Popover、Button 与 React Query，打开时读取 `GET /api/status`。
- `api_info_enabled !== false` 且 `api_info` 是数组时，采用其中非空字符串 `url`；
  展示 `route`、原始 `url`、`description`，复制原始地址，不追加 `/v1` 或规范化 URL。
- 列表关闭、缺失或无有效条目时，使用去除首尾空白的 `server_address`，再回退当前 origin。
  与 Classic 现有判断保持一致（包括不擅自限定 URL 协议）。
- 请求失败显示错误和重试按钮，不把请求失败伪装成空配置；重新打开重新取状态。
- 展示 loading，复制成功/失败反馈；保留原生键盘与焦点管理，窄屏折行和限高滚动，使用主题变量。
- 只读公开配置，无完整令牌读取、权限调整、数据库迁移或持久化；名称和描述按普通文本转义。

## 验证计划

先补缺入口失败回归，然后验证多地址及精确复制、列表关闭、无有效数据、两级回退、
loading、请求失败重试、复制失败、键盘打开/关闭、重新打开刷新配置。
每次测试进程限制 60 秒；运行相关 Default/Classic 组件测试、Default 类型检查、定向 lint/格式检查。
尽可能使用本地真实浏览器检查窄屏长 URL 和深浅色；最终生产构建与独立审查由协调者串行集成。

## 协调者集成事项

本工人不改共享 locales、`docs/developer/README.md`、`custom-development.md`。
协调者需登记本记录链接和能力条目，并补齐七种语言中的以下新增英文翻译键：

| 英文键                         | 简体中文建议      |
| ------------------------------ | ----------------- |
| `API addresses`                | API 地址          |
| `Copy address`                 | 复制地址          |
| `Copied successfully`          | 复制成功          |
| `Default API address`          | 默认 API 地址     |
| `Failed to load API addresses` | 加载 API 地址失败 |

已有键 `Current domain`、`Loading...`、`Retry`、`Copy failed` 继续复用。
Classic 已具备地址入口与同一配置回退，本次仅回归核查，不制造重复改动。

## 验证结果

- Default：`bun run test src/features/keys/components/__tests__/api-key-api-addresses.test.tsx`
  共 10 项通过，约 8 秒。覆盖操作栏入口、无初始请求、多线路、精确复制、关闭列表、
  无有效地址、异常列表类型、站点/origin 回退、加载、网络及业务失败、重试、复制失败、
  键盘焦点和重新打开刷新。测试先因目标组件缺失失败，再实现组件后通过。
- `bun run typecheck` 中途全量运行通过；第一次执行时其他并行工人的筛选与 Logo 实现尚未完成，
  出现过其他文件错误，未修改其文件，随后全量重跑通过。
  交付前再次运行时，另一工人新建的 `usage-logs/components/__tests__/log-group-selection.test.tsx`
  引用尚未创建的 `../log-group-select`，导致全量检查暂时失败；本任务文件无类型错误。
  协调者需在所有工人写入完成后重跑，不能将中途通过视为最终集成通过。
- 三个 TSX 文件定向 `oxlint` 通过；使用 `oxfmt` 格式化这三个文件和本记录。
- Classic：既有 `token-api-info-popover.compat.test.jsx` 三项测试尝试执行，
  收集阶段约 65 秒，达到本任务 60 秒进程上限，不作为通过证据。
  首次残留日志另报 Node `localStorage` 未提供导致公共 afterEach 失败；
  用 `node --no-experimental-webstorage` 启动的第二次仍在收集阶段超时。
  当时交接为在适配的 Classic 测试运行环境重跑三项回归，后续结果见下项。

- 后续独立审查（协调者转交 `ctx_21255e4335ff` 证据）：未发现 P1/P2 或阻断缺陷，
  独立复跑 Default 10/10 组件测试，并核实六种尺寸/主题的模拟 API 浏览器证据。
  随后在 `web/classic` 中给测试进程设置 `NODE_OPTIONS=--no-experimental-webstorage`，
  使用 `node scripts/run-compat-tests.mjs src/components/table/tokens/__tests__/token-api-info-popover.compat.test.jsx --maxWorkers=1 --reporter=verbose`
  获得 exit 0、3/3 通过；Node 26.13.0、Vitest 2.1.9，耗时 47.78 秒
  （收集 44.53 秒、测试 379 毫秒），未修改依赖或配置，仅有 Semi 既有警告。
  此次正确运行器及进程级环境解决了上述测试环境阻碍；保留首次失败与超时证据。
  覆盖 Classic 真实 RTL 组件读取 `/api/status` 的线路名称、原地址、说明、
  关闭且为空的列表回退 origin、缺失列表时 server_address 优先；
  不包含 Classic 复制、失败重试、异常类型、手机或主题，不是生产验证。
  该段记录当时并行实现尚未稳定的阶段性状态；最终统一门禁见下方“最终独立复审与集成门禁”。

## 真实浏览器证据

使用 Playwright CLI、现有 `http://127.0.0.1:5173/keys` 开发服务、虚构普通用户 `role=1`
及完整模拟 `/api/*`。未调用真实生产接口或使用用户令牌。

- 1440、390、320 像素，每种宽度分别验证浅色/深色共六种组合；连续约 400 字符 URL
  和长描述在浮层内换行，浮层及页面均无横向溢出，列表限高并允许滚动。
- Enter 打开浮层，Escape 关闭后焦点返回入口；验证期间无页面异常、无完整令牌密钥请求。
- 实际浏览器剪贴板读取与原始长 URL 严格相等，保留自定义路径、尾部字符和查询参数。
- 模拟 HTTP 503 后出现错误及重试；重试恢复后，关闭列表的配置不会显示其中地址，
  正确回退至去除首尾空白的站点地址，复制值也严格一致。
- 截图保存在本机临时目录 `issue402-{1440,390,320}-{light,dark}.png`，
  已人工查看 `issue402-320-dark.png`，长地址、说明、复制按钮和主题边界均可用。
  脚本在同目录 `issue402-browser-*.js`；模拟数据仅用于本功能，不作为其他并行模块验收。

## 兼容性、剩余工作与分享建议

- Default 新增入口；Classic 原有入口、字段读取和回退顺序经源码核对保持一致。
  Classic 的已有错误提示及重新打开重试不在本工人写入范围，Default 额外提供原地重试按钮。
- 不改变后端响应、鉴权、配额、计费、模型名或跨系统调用契约，无新增兄弟项目 API 影响。
- 共享翻译、README 索引和能力登记、最终统一生产构建、Classic 运行环境复验及独立审查，
  由协调者整合；本工人未提交、推送、评论或部署。
- `NewAPIForDouDi`：若采用 Default 且缺少地址入口，值得分享；依赖原生 Base UI 与公开状态字段，
  需先核对其组件体系，不直接复制 Classic UI。
- `NewAPIModifyByGang`：同样仅在存在此迁移缺口时建议分享；功能价值是保留线路发现和精确复制，
  无后端迁移成本。未向任一仓库发送消息。

## 最终独立复审与集成门禁（2026-11-16）

- 独立 reviewer `ctx_21255e4335ff` 未发现 P1、P2 或阻断缺陷；Default 10/10 组件回归、
  1440/390/320 三种宽度的浅深色六视口 mock 浏览器证据均已核实。
- Classic 动态回归由 `ctx_1ca347affac2` 在 Node 26.13.0、Vitest 2.1.9、进程级
  `NODE_OPTIONS=--no-experimental-webstorage` 下通过 3/3。覆盖真实 RTL 组件加载、
  `/api/status` 线路名称、原地址、描述，以及关闭空列表回退 origin、缺失列表优先
  `server_address`；不覆盖 Classic 复制、失败重试、异常类型、手机主题或生产环境。
- 早期 localStorage/collection 超时和环境失败原因保留在上文，不能覆盖为生产失败；本次
  通过使用 compat 脚本和 Web Storage 进程开关解决。统一结果见[总交付记录](16_issue401_delivery.md)，
  三项均 exit 0，Classic 没有单独 typecheck 结果，且仅有既有 chunk 体积警告。
- 未验证真实认证、生产后端、MySQL/PostgreSQL、Redis 多实例或供应商服务；未提交、推送、
  部署或关闭 Issue。

## 新一轮独立审查返修计划（2026-11-16）

- reviewer `ctx_b066fbb302a4` 使用真实组件、QueryClient 和从 `main.tsx` 提取的真实
  `QueryCache.onError` 复现 HTTP 500 跳转 `/500`，证据在
  `D:/Temp/issue401-fresh-review-keys/address-global-error.test.tsx` 与 `edge.log`。
  `skipErrorHandler` 只控制 HTTP 拦截器提示，不能阻止 QueryCache 的 AxiosError 分支。
- 最小修复仅限本组件、其测试及本文：局部请求的非认证 Axios 故障转为普通 Error，
  保留 401/403 原错误及 Axios 取消，不改全局 main/http-client；将查询 signal 传入请求，
  使卸载取消由 React Query 管理，不把取消误报为局部服务器失败。
- 增加真实全局 onError 的 500/503 局部重试恢复、401/403 原错误保留以及卸载取消回归；
  保留原有地址、复制和键盘测试。每个测试子进程最多 60 秒，TEMP/TMP 在 D 盘，
  使用进程级 NODE_OPTIONS；不重复全构建，不修改 reviewer 夹具。
- 原审查与构建结论属于先前快照；本次返修完成后交新 reviewer 复跑旧失败夹具，
  不能提前宣称独立复审、提交或部署完成。

### 返修实现与验证

- `ApiKeyApiAddresses` 的 queryFn 将 signal 传给 `/api/status`，仅把非认证、非取消的
  AxiosError 转为携带 cause 的普通 Error；401/403 及 CanceledError 原对象继续抛出。
  没有设置 skipAuthRefresh，也没有改全局 HTTP 拦截器。源码核对确认 401 的刷新与登录
  重定向发生在组件 catch 之前，skipErrorHandler 仅关闭提示；403 原处理链不变。
- 原 10 项组件测试全部保留，新增 6 项：真实应用 QueryCache.onError 下 HTTP 500/503
  保持当前页面、无全局错误提示并局部重试成功；401/403 原错误保留；卸载中止请求且不调用
  全局 onError；独立 Axios 取消错误保持原对象。测试从当前 `src/main.tsx` 读取并提取真实
  回调，仅替换导航和提示外部边界，不复制一份简化全局错误策略。
- 红测 `red.log`：15 项中 13 通过、2 失败（HTTP 500 确实导航 `/500`、请求未带 signal），
  子进程 4.128 秒。修复后最终 `green-verified.log`：16/16 通过，exit 0，子进程
  4.284 秒，Vitest 4.1.10 报告 3.67 秒。证据目录为
  `D:/Temp/issue402-local-errors-ctx726dc2704c08/`，每次运行均有对应 JSON 退出码与时长。
- 测试命令（cwd=web）：`node node_modules/vitest/vitest.mjs run src/features/keys/components/__tests__/api-key-api-addresses.test.tsx --maxWorkers=1`。
  外层运行器设置 60 秒强制上限，TEMP/TMP 指向上述 D 盘目录，
  `NODE_OPTIONS=--no-experimental-webstorage` 仅作用于测试子进程。
- 初次通过后为满足 lint 将测试的 `?raw` 导入改为文件读取；中间使用 import.meta.url 的
  夹具在 jsdom 下不是 file URL，六项新增测试因此失败，保留 `green-final.log`。
  改为从约定的 web 工作目录读取真实入口后最终 16 项通过；这不是产品回归。
- 两个 TSX 的定向 oxlint 通过，三份文件经 oxfmt 与差异空白检查。
  没有新增翻译键或修改共享文档、全局 main/http-client、其他 worker 文件及 reviewer 夹具。
- 本次证据为真实组件/QueryClient 配合 API mock，认证测试验证原错误身份与状态保留，
  不冒充真实登录刷新端到端验证；未新增浏览器、Classic、全构建或生产验证。
  Classic 入口不使用此 Default QueryCache，代码保持原样，先前动态回归边界仍有效。
  实现交接时新 reviewer 复跑及独立复审尚待安排，当时未提交、推送或部署；最终复验结果追加如下。

## 新一轮最终独立复验（2026-11-16）

- reviewer `ctx_b066fbb302a4` 已正式通过：未改 `address-global-error.test.tsx` 转绿，与活动原失败夹具组合 2/2，exit 0，4.36 秒；地址完整专属回归 16/16，exit 0，4.09 秒，报告 `D:/Temp/issue401-fresh-review-keys/review-report.md` 保留原失败日志和夹具 SHA256。
- 真实 main.tsx 全局回调下 500/503 局部重试恢复、401/403 原错误、取消及卸载 signal 与原地址/复制/回退/键盘场景通过，本项 P2 闭环。401/403 只验证传输边界和拦截器静态链路，不是登录续期服务器验收；本轮 reviewer 另跑 Classic 地址 3/3，42.98 秒，仅覆盖原三项边界。
- 无新增生产、实库或完整 E2E 证据，不把历史浏览器与构建计为本次新执行。用户已授权提交推送 dev、部署 zzapi 并切换 Default，截至本段登记尚未执行；最新发布构建及保留线上 money MCP overlay 的组合产物证据见[部署专题](16_issue401_dev_zzapi_deployment.md)。
