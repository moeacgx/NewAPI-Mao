# Issue #398 Default 供应商 Logo 工作记录

## 目标与范围

补齐 Default 模板对供应商 `icon` 字段中 HTTP/HTTPS 图片直链的渲染支持，复用既有保存契约，核查供应商编辑表单的字段保存、模型列表、渠道供应商选择、模型广场卡片/列表/详情/筛选与排行榜等共享调用点；保留 LobeHub 名称、点号链式参数、Sub2API/Wan 自定义图标。仅修改 `web/src/lib/lobe-icon.tsx`、其专属回归测试及必要 Logo 使用点；不触及 `keys` 目录、共享 locale、开发总览/能力总览文档和 Classic 源码。

## 已核查根因

Default `getLobeIcon` 在 URL 判断前直接以 `.` 拆分 `iconName`，HTTPS URL 被当成 LobeHub 属性路径，最终只显示首字母 fallback。Classic 的 `getLobeHubIcon` 已先识别合法 HTTP/HTTPS URL，并以固定尺寸、`object-fit: contain`、`referrerPolicy=no-referrer`、加载失败头像兜底渲染；地址变化通过 key 重置失败状态。

## 实施方案

在 Default 共享图标加载器中沿用 Classic 契约：先验证 HTTP/HTTPS 绝对 URL（拒绝带凭据地址和非法地址），使用固定 `width/height` 的 `img`，`loading=lazy`、`referrerPolicy=no-referrer`、`object-fit: contain`，图片失败后显示问号占位；URL 变更时重置失败状态。模型列表在供应商图标路径遇到 URL 时保留原值，跳过会改写图标路径的 `.Avatar` 链参数。以测试覆盖实际共享渲染路径所需的 URL、失败、空值、LobeHub 链式参数和自定义图标行为。

## 验收计划

- Default：专属 Vitest/RTL 回归、受影响文件 lint、typecheck、生产构建。
- Classic：核对既有渲染和契约测试；不改 Classic 源码。
- 浏览器：若当前可用本地页面与数据，提供桌面/手机及浅色/深色证据；否则记录实际阻碍。
- 文档：本记录登记 Default 已支持 URL，并在交付中说明需由协调者串行校正 `docs/developer/custom-development.md` 第 243-248 行的双模板 Logo 能力与验收状态；按任务约束不直接改该文件。

## 新翻译需求

本修复不新增用户可见文案，不需要 locale 变更。

## 实施与核查结果

- 根因已在 Default `getLobeIcon` 确认：此前 URL 在点号切分之后只会命中首字母兜底。
- 在共享图标加载器新增 HTTP/HTTPS 图片分支，使用浏览器原生图片加载、固定请求尺寸、
  等比完整展示、懒加载和 `no-referrer`。无效/带凭据地址不发图请求；加载失败显示现有
  问号占位；URL 改变通过 key 重新初始化失败状态。空值、LobeHub 和自定义图标仍走既有路径。
- 调用点核查：供应商编辑表单保存原 `icon` 值，但没有独立的 Logo 预览；ProviderBadge
  实际调用点为模型表格供应商列（显示供应商 Logo）和渠道表格类型列（显示渠道类型图标）。
  模型名称单元格从 `model.icon` 或供应商 `icon` 获取紧凑 Logo；渠道供应商筛选及编辑选项目前仅显示名称，不是供应商 Logo 消费点。
  模型广场的筛选、表格、卡片、详情以及榜单/Pulse 均使用共享 `getLobeIcon`。
- Classic 对照：`getLobeHubIcon` 已支持相同 URL 加载契约；既有 Classic 渲染逻辑及供应商、
  模型、渠道、模型广场入口未改。Node 26 默认未启用 Web Storage，compat 全局清理阶段
  `localStorage.clear()` 抛出 `TypeError: Cannot read properties of undefined (reading 'clear')`。
  两组测试首次运行报告的 14 项均在清理钩子失败，退出码为 1，并非断言失败。仅在 Classic
  测试子进程设置 `NODE_OPTIONS=--no-experimental-webstorage` 后，两个指定文件 14/14 通过；
  未修改系统环境或依赖。
- Default RTL 回归覆盖 HTTP/HTTPS 固定尺寸和安全属性、失败问号占位（含 `flex-shrink: 0`）、
  URL 切换、空值、非 HTTP 协议、带凭据地址、LobeHub 链式参数及 Sub2API 图标；另有模型列表
  单元回归验证供应商及模型自定义 URL 的原值不会拼接 `.Avatar` 参数。

## 验证记录

- `NODE_OPTIONS=--no-experimental-webstorage node node_modules/vitest/vitest.mjs run src/lib/__tests__/lobe-icon.test.tsx src/features/models/components/__tests__/model-logo.test.tsx --maxWorkers=1`（在 `web`）：通过，2 个文件 13/13，8.15 秒；包含用户名、密码及 URL authority userinfo 回归。
- `NODE_OPTIONS=--no-experimental-webstorage node scripts/run-compat-tests.mjs src/helpers/__tests__/vendor-logo.compat.test.jsx src/components/table/models/__tests__/vendor-logo.compat.test.jsx --maxWorkers=1 --reporter=verbose --reporter=json --outputFile=<临时报告>`（在 `web/classic`）：通过，2 个文件 14/14，47.48 秒（Vite 收集约 44 秒）。首次不带 `NODE_OPTIONS` 的 compat 运行退出 1，stderr 指向 `scripts/setup-compat-tests.mjs:22` 的上述 `localStorage.clear()` TypeError。
- `bunx oxlint -c .oxlintrc.json src/lib/lobe-icon.tsx src/lib/__tests__/lobe-icon.test.tsx src/features/models/components/models-columns.tsx src/features/models/components/__tests__/model-logo.test.tsx`：通过。
- `bun run typecheck`：上轮实现验证记录为通过；本轮遵照协调者要求未重复运行全量 typecheck。
- `bun run build`：上轮实现验证记录为通过，Rsbuild 15.3 秒且有既有超大异步分块提示；本轮未重复全量构建。
- `bunx oxfmt --check src/lib/lobe-icon.tsx src/lib/__tests__/lobe-icon.test.tsx src/features/models/components/models-columns.tsx src/features/models/components/__tests__/model-logo.test.tsx`：通过。
- 仓库 `format:check` 因检查全仓及 Classic 中缺失的
  `web/classic/node_modules/tailwindcss/theme.css` 失败；改用 Oxfmt 对本次四个前端文件检查通过。
- `git diff --check`：通过。
- 中间检查及处理：Classic 首次 compat 退出 1 的环境根因和最终复验见上文；Default 首轮
  lint 指出列表单测的单子节点 fragment，移除后通过。增加带 user-info 的 URL 回归时，RTL 的
  图标库 mock 缺少相应 URL 形态导出，扩展 mock 后最终专属测试 13/13 通过。尝试从 `web`
  目录以 `../docs/...` 调用 Oxfmt 被工具拒绝（禁止 `..` 路径）；改从仓库根目录调用
  `web/node_modules/.bin/oxfmt.exe --check` 后工作记录格式检查通过。
- 使用 Playwright CLI 在 Default 本地开发服务器打开模型广场；`/api/status`、
  `/api/pricing`、供应商图片均为受控 fixture，未登录生产环境。模型卡片实测桌面 1440px
  与手机 390px、浅/深色主题、成功/失败图片、长模型名、实际安装的 OpenAI 链式图标和
  Wan 自定义图标；成功图片 `naturalWidth=64`，卡片显示 28×28，失败显示固定尺寸问号，
  手机 `scrollWidth=390`，无横向页面溢出。模型广场表格中的失败问号、成功 Logo 与供应商
  Badge 也在桌面浏览器实测；截图保存在 `D:\issue398-desktop-light.png`、
  `D:\issue398-desktop-dark.png`、`D:\issue398-mobile-light-clean.png`、
  `D:\issue398-mobile-dark-clean.png` 和 `D:\issue398-desktop-table-light.png`。这些是本地
  mock 视觉证据，不是生产验收。供应商编辑及需认证的管理模型列表未作真实浏览器流程验收；模型列表单元
  测试实际渲染 TanStack 模型名单元格，覆盖供应商继承 URL 和带 query 的自定义 URL 原值。

## 独立审查反馈处理

- P2 凭据绕过：URL 解析后检查 `URL.username`，并检查原始 authority 是否包含 `@`，
  覆盖同主机前缀用户名 `cdn.example.com@cdn.example.com` 及空 userinfo 变体；URL 表驱动
  回归确认这些输入不会产生图片元素。
- P2 模型列拼接：模型列仅对非 URL 的品牌名追加 `.Avatar.type={'platform'}`；URL 直接传入
  共享加载器。新增列表级测试以 TanStack 实际单元格渲染覆盖供应商继承 URL 和带 query 的
  模型自定义 URL。
- P3 失败占位压缩：FallbackIcon 设置 `flexShrink: 0`；回归断言固定宽高和不收缩，手机卡片
  长名称截图显示问号占位完整。
- 实际品牌图标：本地浏览器使用已安装图标包渲染 OpenAI 链式图标及 Wan 自定义图标；RTL
  测试仍以 mock 隔离图标库。浏览器为受控 API fixture 下的视觉回归。

## Classic 与共享文档交接

Classic 无源码改动，原本已实现 URL 渲染及固定尺寸、失败回退契约。首次运行暴露 Node 26
测试子进程未启用 Web Storage 的环境问题；随后按项目 `run-compat-tests.mjs` compat 入口执行，
仅给测试子进程设置 `NODE_OPTIONS=--no-experimental-webstorage`，相关两个测试文件 14/14
通过。没有修改依赖或系统环境。

共享文档已登记“双模板供应商自定义 Logo”字段、URL 和加载失败契约，保留 HTTPS 页面可能
阻止 HTTP 图片、无迁移、最长 128 字符限制及历史链接；Default 模型广场受控 mock 的桌面/
手机、深浅色、正常/失败图、链式/自定义图标和表格视觉证据仍明确不是生产验收。Classic
compat 脚本在 Node 26 测试子进程加 `NODE_OPTIONS=--no-experimental-webstorage` 后两个
文件 14/14 通过。

## 最终独立复审与集成门禁（2026-11-16）

- 独立 reviewer `ctx_84181ce7d92b` 复审确认两个 P2 与 fallback 收缩闭环，无新增 P1/P2
  或阻断；Default 13 项、Classic 14 项和仓库外边界 7 项通过，并人工核对五张 mock 截图。
- 以上是 RTL、组件/边界测试与本地受控 mock 视觉证据，不能推出真实认证、生产图片、
  跨模板保存或上游服务验收通过。渠道供应商筛选和编辑选项仍仅显示文字，不计作 Logo
  展示入口。
- 统一门禁结果见[总交付记录](16_issue401_delivery.md)：Default build、Classic build、
  Default typecheck 均 exit 0；Classic 仅保留既有 chunk 体积警告，`git diff --check`
  通过。未提交、推送、部署或关闭 Issue。
