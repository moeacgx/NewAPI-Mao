# Issue #405：双模板顶栏导航宽度与响应式布局

## 目标与范围

用户截图中，Default「扩展模块 → 渠道可观测性」的中文顶栏导航被逐字换行，
搜索占宽较大。修复全局宿主布局，保留所有导航、搜索、通知与用户操作的可达性，
不增加整页横向溢出。授权修改 `app-header.tsx`、`top-nav.tsx`、
Classic `PricingTemplateHeader.jsx` / `.css`、专属测试及本记录；
不改共享翻译、索引、MCP 或外置扩展。

## 调用链与待验证根因

- `AuthenticatedLayout` 在侧栏和页面内容之前挂载全宽 `AppHeader`；
  普通后台与 `/extensions/$moduleId/$pageKey` 共用该宿主顶栏。
- 原生扩展由 `ExtensionPage → NativeExtensionPage` 加载在内容区域，
  不生成宿主顶栏，因此修复不应下沉到渠道可观测模块。
- `AppHeader` 右侧横向排列导航、搜索和动作，缺少可收缩的宽度边界。
  `Search` 在桌面设置固定宽度；`TopNav` 链接既能收缩又没有禁止换行，
  中文允许在任意字间换行，可能突破固定顶栏高度。
- `TopNav` 已有小屏菜单，但 `AppHeader` 又用 `hidden lg:block` 包住整个组件，
  导致小屏菜单本身也被隐藏。
- 浏览器使用本地真实宿主、模拟 `/api/status` 中文自定义菜单、模拟原生扩展入口；
  不连接线上或读取真实业务数据。截图宽度和完整站点导航配置未知，
  以相近宽度及更长中文菜单明确记录复现边界。

## 方案

- 将导航区域设为可收缩的独立区域，横向链接禁止换行和压缩，
  仅在导航容器内部滚动，不扩张页面。
- 所有屏幕保留有可访问名称的导航菜单，窄屏通过菜单访问完整文本，
  桌面也能不用水平滚动找到长导航。菜单限制在视口宽度内，并允许正常文本换行。
- 搜索在不足宽度时使用原有图标按钮，宽屏给有上限的紧凑宽度；
  固定尺寸动作不被导航挤压。品牌在窄屏限制占宽，保留可访问名称。
- 复用现有 Base UI DropdownMenu、Button、Link，不新增依赖和全局样式。

## 契约、安全与兼容

- 不改变 HeaderNavModules 的解析、动态菜单优先级、链接 URL、路由鉴权或扩展 API。
- 保留禁用项不可操作、外链 `noopener noreferrer`、新窗口行为与内部链接导航。
- 保留自定义 `leftContent/rightContent` 及各显隐参数的既有行为。
- Classic 后台和模型广场经 `HeaderBar` 实际选择 `PricingTemplateHeader`，
  并非初步检查的 `Navigation.jsx`；两套模板不复用本次布局组件。
- 新增可访问文案使用英文 i18n 键，由协调者统一登记翻译。

## 测试计划

- 修前真实浏览器记录长中文换行和顶栏边界；修后桌面、窄桌面、平板、390px，
  深浅主题、侧栏展开/收起、原生扩展与常规后台页面。
- 浏览器断言整页无横向溢出、可见导航单行且位于顶栏内、搜索/通知/用户入口可点击；
  通过键盘打开导航菜单、访问长链接、Escape 关闭并返回焦点。
- 组件测试使用真实 TopNav/DropdownMenu，验证可访问名称、展开状态、键盘、禁用项、
  外链安全属性、空链接和动态长标题，不用 class 快照代替行为。
- 每个测试子进程不超过 60 秒，TEMP/TMP 指向 D 盘，
  Node 26 使用进程级 `--no-experimental-webstorage`。
- 执行定向测试、类型检查、受影响文件 lint/格式检查；
  全量构建及独立审查由协调者统一安排，不提交或部署。

## Classic 扩围依据与实施前方案

真实浏览器对照发现独立的宽度缺陷：1100px、相同长中文导航时，链接虽保持单行，
用户区却位于 x=1343.58～1429.64，超过视口。导航可用宽度约947px，
内容溢出到1406px；整页 scrollWidth 仍为1100，所以仅检查整页宽度无法证明操作可达。
修前证据在 `D:/Temp/newapi-issue405-header/classic.json` 和 `classic-1100.png`。
该次对照使用本地02:51:52构建的 Classic 产物，源码同样将链接及所有操作放在一个
无局部滚动边界的 flex 容器内；后续以当前源码 Vite 服务重新验证，排除旧产物因素。
390px 原移动菜单可以展开完整长导航。

协调者已于本任务内授权独占 Classic 两个文件与专属测试。实施方案是将桌面链接包入
可收缩的横向滚动区域，固定语言、主题、通知和用户区；链接保持完整文字与正常 Tab
顺序，聚焦时由浏览器滚入可见区域。限定品牌宽度，避免较长站点名称挤占操作。
不改变1100px断点、登录重定向、外链安全属性、移动菜单流程及业务权限。
回归使用真实组件验证链接和移动菜单交互，真实浏览器验证1100px深浅色、代表性
宽桌面和390px移动菜单，不通过固定等待规避宽度缺陷。

## 2026-11-16 分工调整交接（历史，已被文末最终状态取代）

协调者因新的 AGENTS 分工要求，指示前端实现转交 Claude Code。本工人收到停止通知后
不再修改前端源码或测试，不回滚现有成果；以下七个文件的前端写权及本记录写权在正式
交接后释放，由协调者重新分配。任务不能视为已完成，尚需 Classic 修后浏览器验收、
最终定向检查与另一工人的独立审查。

### 精确文件与当前状态

1. `web/src/components/layout/components/app-header.tsx`：已限制品牌宽度、建立可收缩
   导航区域、恢复小屏菜单入口，搜索在窄宽度为图标、宽屏为160px；保留原有显隐参数。
2. `web/src/components/layout/components/top-nav.tsx`：已增加局部横向滚动、单行链接、
   所有尺寸可用的原生 DropdownMenu、可访问名称及 active 状态；完整长标题在菜单
   中可正常换行，菜单每项独占一行，视觉顺序与 Tab 顺序一致。
3. `web/src/components/layout/components/__tests__/top-nav-accessibility.test.tsx`：
   新增真实 Router/TopNav/DropdownMenu 测试，覆盖键盘、焦点返回、禁用、外链和空列表。
4. `web/classic/src/components/layout/headerbar/PricingTemplateHeader.jsx`：
   已将桌面导航链接放入独立容器，未改变链接解析与移动菜单逻辑。
5. `web/classic/src/components/layout/headerbar/PricingTemplateHeader.css`：
   已增加导航区域局部滚动、品牌上限和非导航操作不收缩、链接聚焦轮廓；
   **刚实施，尚未格式化与修后浏览器验证**。
6. `web/classic/src/components/layout/headerbar/__tests__/header-navigation-accessibility.compat.test.jsx`：
   新增三个真实组件测试，保护长链接/外链/Tab顺序/通知操作、访客登录重定向、
   移动菜单键盘展开和选项激活后关闭。尚未格式化。
7. 本文档：包含实施前方案、根因和交接，不改共享文档索引。

上述四个既有源码文件在本任务开始接手对应范围时无其他未提交修改；没有修改共享
翻译、MCP、routeTree、其他工人文件或外置扩展。Classic JSX 当前仅包一层链接容器；
Default 原有 `app-header-layout.test.tsx` 只运行未修改。

### 已取得的精确证据

所有本地证据位于 `D:/Temp/newapi-issue405-header/`。浏览器由 Python Playwright
运行真实 Chromium；API、用户和自定义菜单全部为本地夹具，非生产数据。

- Default 修前 `before-1100-light-extension.json`：固定48px顶栏内，主页40px高、
  控制台60px高、模型广场80px高（y=-16～64），侧栏收起后问题仍在。
  对应 PNG 保留修前可见效果。
- Default 修后 `after-{1874,1100,820,390}-{light,dark}-{extension,ordinary}.json`：
  **16/16 passed，0 pageerror**。每种记录侧栏展开/收起与菜单截图，断言可见导航
  单行、必需动作位于视口内、整页 scrollWidth 等于视口宽度、长菜单单项独占一行、
  Enter/End/Escape及焦点返回、搜索对话框、通知和用户菜单可打开，桌面最后链接聚焦
  会滚入可见区域。移动主题设置按钮原来就隐藏，夹具修正后不将其误判为必须可见。
- Default 浏览器入口是真实 `/extensions/channel-observability/overview` 和
  `/usage-logs/common`；前者扩展内部内容由最小 native SDK 夹具渲染，宿主布局、
  侧栏和顶栏真实。**没有验证完整渠道可观测性业务或线上页面**。
- Default 组件测试曾红3/3（原组件缺少本次菜单可达契约），
  `green-final.log` 记录新测试3项加既有布局测试1项 **4/4通过，26.68秒**。
  之后菜单链接改为块级独占行并调整DOM顺序，16场景浏览器已覆盖最终变化，
  **组件测试尚需在最终源码重跑一次**。
- Classic 旧构建与源码服务均复现同一溢出。`classic-source-before.json` 最新成功
  结果为1100px用户区 right=1429.640625；不是单纯旧产物问题。
  `classic-source-before-1100.png` 为当前源码修前截图，390px对照也保留。
  第一次源码浏览器冷编译超过默认10秒导航超时，调为30秒导航上限后成功，
  子进程总上限始终60秒；没有固定等待。
- Classic 已启动的组件测试在停止通知后收取：`classic-component.log` 显示
  **1文件3测试通过，40.77秒**，有既有 Semi findDOMNode 等警告；
  PowerShell 这次执行返回1但 Vitest 日志明确3项通过，需继任者复跑确认进程退出码。
  该测试在 Classic JSX 包裹修改前启动，不能代替最终源码与 CSS 布局验收。
- Default 修改已做局部 oxfmt（保留版权头）及 oxlint；Classic 新变化尚未格式化/lint。
  最后四个源码文件 `git diff --check -- <文件>` 无输出通过。
- 曾运行一次 `node node_modules/@typescript/native-preview/bin/tsgo -b --pretty false`，
  `tsgo.log` 有11项来自 keys/models/profile/subscriptions 并行改动的类型错误，
  没有本任务文件报错，**不能宣称全量类型检查通过**。协调者已接管集成，
  明确不再重复全量 typecheck/build。

### 继任者需要继续的事项与命令

所有测试子进程最大60秒，进程级环境：

```powershell
$env:TEMP = 'D:/Temp/newapi-issue405-header'
$env:TMP = $env:TEMP
$env:NODE_OPTIONS = '--no-experimental-webstorage'
```

1. 在当前源码上完成 Classic 1100px深浅色、宽桌面代表场景和390px移动菜单浏览器：
   操作区必须在视口内、长导航完整可滚动、Tab聚焦最后链接可见、主题/语言/通知/用户
   操作可用、移动菜单/登录权限不回退。需要编写修后断言脚本，现有
   `classic-source-before.py` 只有记录并不校验用户区边界，不可直接当绿测。
2. 定向重跑下面测试；不再全量类型检查或构建。

```text
# cwd web/，外层子进程 timeout=60
node node_modules/vitest/vitest.mjs run src/components/layout/components/__tests__/top-nav-accessibility.test.tsx src/components/layout/components/__tests__/app-header-layout.test.tsx --maxWorkers=1
# cwd web/classic/，脚本内置60秒超时
node scripts/run-compat-tests.mjs --config vitest.config.mjs src/components/layout/headerbar/__tests__/header-navigation-accessibility.compat.test.jsx
# Default 已完成浏览器矩阵的可重复入口（每个子进程60秒）
python D:/Temp/newapi-issue405-header/matrix.py
```

3. 格式化 Classic 两源码与新测试，保留版权头；定向 lint、diff检查，
   更新最终证据后提交协调者安排独立审查。不能由本实现者自审替代独审。4.（历史）当时需协调者补 Default 英文翻译键 `Navigation`、`Navigation menu`，现已集成
   （建议中文“导航”“导航菜单”），并将本工作记录加入开发索引；本工人没有共享写权。
   Classic 本次没有新增文案。

### 进程归属、接口与限制

- 本工人启动并仍保留：PID **55364**，`node node_modules/vite/bin/vite.js --host
127.0.0.1 --port 5197 --strictPort`，cwd `web/classic`；
  日志为本证据目录 `vite.log` / `vite.err`。可交继任者继续验证，当前未停止。
- Default `127.0.0.1:5174` 是任务前已有服务（记录PID52188），不属于本工人，
  不应停止。所有本轮浏览器/测试子进程已结束，没有轮询或后台验证任务。
- 未启动后端、未读取密钥、未连接真实数据库或生产API、未提交推送、未部署；
  本次只影响宿主导航展示，无跨项目API/鉴权/计费契约变化。
- NewAPIForDouDi、NewAPIModifyByGang 均可在确认存在相同长导航容器结构后参考
  “导航局部滚动、动作保留宽度”的方案；当前未核对两库结构，不建议直接照搬组件，
  不发送任何跨仓库通知。

## 2026-11-16 最终验收与事故后复验（Claude Code 接续）

### Classic 追加修复

- `PricingTemplateHeader.jsx` 的桌面链接 `renderLink` 增加 `onFocus`：非移动端时调用 `event.currentTarget.scrollIntoView({ block: 'nearest', inline: 'nearest' })`。原因：Chromium 不会滚动「部分可见」的聚焦元素，1600px 下最后一个链接右缘 1333.5 超出容器 1267.9，scrollLeft 仍为 0。
- 链接容器 `.classic-pricing-template-links` 保持 `overflow-x: auto`，语言、主题、通知和用户区不被压缩。

### 事故记录（本工人造成）

本工人曾在 `web/` 运行 `node scripts/format-with-protected-headers.mjs --check`。该脚本忽略文件参数，会剥离 `web/` 下全部 ts/tsx/js/jsx/mjs/cjs/mts/cts 的 AGPL 头，随后在 `restoreSnapshot` 崩溃（errno -4094，`src/features/models/components/drawers/model-mutate-drawer.tsx`），导致 782 个文件丢失头部。本工人未做任何修复写入并已升级上报；协调者安排的独立审查核验 783 个文件已恢复且正文不变。此后本工人不再调用该脚本的任何模式，只使用直接的 prettier、eslint、oxlint、oxfmt。

### 事故后复验证据（均在恢复后重新执行）

- Default 组件测试：`top-nav-accessibility.test.tsx` 加 `app-header-layout.test.tsx`，2 文件 4 测试通过，exit 0。
- Classic 兼容测试：`header-navigation-accessibility.compat.test.jsx` 3 测试通过，exit 0（有既有 findDOMNode 警告）。
- Classic prettier --check 与 eslint 目标文件 exit 0；Default oxlint 与 oxfmt --check 目标文件 exit 0；`git diff --check` exit 0。
- Classic 修后浏览器（自有 Vite PID 55364，127.0.0.1:5197，Playwright，脚本 `D:/Temp/newapi-issue405-header/classic-after.py`）：4/4 通过，0 pageerror，exit 0。
  - 1100 浅色与深色：scrollWidth 1100，用户区 x=985.94、宽 86.06，位于视口内；链接区 client 681、scroll 1086、overflow-x auto；Tab 聚焦最后链接后其右缘 848.5，在容器内（容器右缘 847.94，容差 1px）。
  - 1600 浅色：用户区 x=1405.94；链接区 client 1021、scroll 1086；最后链接聚焦后在容器内。
  - 390 深色：移动菜单展开，最后一个链接可见。
- 首次复验的 1100 浅色失败是 `Page.evaluate: Execution context was destroyed`，为 Vite 重载与测量的竞态，属于夹具问题而非布局缺陷；仅在 D 盘临时脚本中改为等待 networkidle 与 `wait_for_function` 后重跑，源码未因此改动。
- Default 16/16 浏览器场景为事故前取得的证据，源码此后未再改动，未在事故后重跑浏览器矩阵；组件测试已在最终源码复跑。

### 事故后 SHA256（仅记录当时状态）

- app-header.tsx 23EAFFFFCCAA6C9730F91F18E69F8DE938308D009C04D7DA2EA6C0A178867727
- top-nav.tsx 007D4F5E44C86BDF521D577A036DC3DA06DC9E59DBF1A59112F32BF63D4E2D93
- top-nav-accessibility.test.tsx 26176A465BC11F22BEC56C7286EE8F85009368DA737C30193373692DA5895519
- PricingTemplateHeader.jsx 18492CF8CAEEB971507278C8FCC740FC948974416569C507F7275E12B0741BB3
- PricingTemplateHeader.css 26207EC479F3F8E3ECB9333A4A87FB0028C661AA24B26D6780746761810FFB22
- header-navigation-accessibility.compat.test.jsx E42D7B344D0B3FD21DAC14336009A937F94331400D6FE613ADDCF9F4D0932801

tsx 与 jsx 文件的 QuantumNous 版权头均在位。本文档自身 hash 随本次编辑变化，不在此记录。

### 未验证项与边界

- 未在浏览器逐项验证 Classic 未登录访问链接的登录重定向，以及 1100px 下主题、语言、通知、用户菜单的点击操作；这些仅由组件测试覆盖（访客登录重定向、通知入口、移动菜单键盘），不能视为浏览器已验证。
- 未执行全量 typecheck 或 build；之前的 tsgo 错误来自其他工人的并行改动，与本任务文件无关。
- 历史交接说明：当时需协调者补 Default 英文翻译键 `Navigation`、`Navigation menu`；这两键现已集成到七语言包，并已通过翻译最终文档独审。
- 未提交、未推送、未部署；Vite PID 55364 为本工人所有，5174 不属于本工人。（历史）当时仍需另一名 Codex 独立审查，该审查现已完成，见下节。

## 最终状态（独立审查后，2026-11-17）

本节为当前有效状态；上文各节均为历史过程记录，其中「尚未完成」「仍需独审」等表述已被本节取代。

### 独立审查结论

- 当前代码已由另一名 Codex 独立审查（`D:/Temp/issue405-final-review/report.md`），未发现 P1/P2。
- 翻译键 `Navigation`、`Navigation menu` 已集成到七个语言包，并通过翻译最终文档独审（`D:/Temp/issue403-405-final-doc-review/final-doc-review.md`）。

### 证据口径

- 作者原始四件证据被后续运行覆盖且不可恢复：`after-1100-dark-ordinary.json`，以及展开、菜单、折叠三张截图（目录 `D:/Temp/newapi-issue405-header`）。作者历史 16 场景矩阵因此不再作为完整通过证据。
- 当前有效证据以独立复核的代表场景为准：Default 1100 深色复跑 exit 0，Classic 1100、Classic 390 深色及相关组件测试。独审中 `classic-current-390` 截图显示加载状态，目标页面不计入通过。

### 模板边界

- Default（`web/src`）：`app-header.tsx`、`top-nav.tsx` 及组件测试，覆盖 1100 深色独立复跑与组件测试。
- Classic（`web/classic`）：`PricingTemplateHeader.jsx`、`PricingTemplateHeader.css` 及兼容测试，覆盖 1100 与 390 深色代表场景；`onFocus` 滚动修复仅针对桌面链接区。

### 仍未验证

- 访客未登录链接的登录重定向、真实后端、1100px 下主题/语言/通知/用户菜单点击，以及全部视口与主题组合，均未在浏览器中验证。
- 未执行全量 typecheck 或 build；未提交、未推送、未部署 zzapi。

### 兄弟项目建议

- `NewAPIForDouDi`：仅建议评估顶栏窄屏链接区横向滚动与聚焦滚入的思路；需其自行核对模板结构，不建议直接照搬。
- `NewAPIModifyByGang`：同上，仅供参考。
- 以上仅为建议，未向任何兄弟仓库创建 Issue、PR 或消息。
