# Classic 任务插件多源管理

## 目标与范围

Classic `/console/task-plugins` 增加插件市场和多插件源配置。默认源由后端提供，包含 NewAPI 官方源与 MaoLao 维护源。临时源码继续使用原有上传入口，稳定版本从维护仓库的索引安装。Default 与后端由各自工作树负责，本工作项不修改它们。

## 接口与安全边界

- Admin 可读取 `/api/plugin/task/marketplace/sources`，Root 可 PUT 同一路径保存 `{name,index_url}[]`。允许空数组，最多 16 个唯一地址；地址只接受不含凭据、查询参数和片段的 HTTPS URL。默认源来自后端配置，前端不在请求失败时伪造默认列表。
- 浏览器读取选择源的 `index.json`，仅接受 `indexVersion=1` 的任务插件版本，源码必须使用位于索引目录下的相对路径。跨源、绝对路径、目录穿越、查询参数、片段及编码绕过均拒绝。
- 远端请求不携带 Cookie、Authorization 或 Referer，不跟随重定向；索引最多 2 MiB、源码最多 1 MiB 原始字节，每次下载限时 15 秒。损坏 UTF-8 拒绝，BOM 字节保留用于哈希。
- Root 安装前查看源码预览并确认 key、version、源地址和 SHA-256。浏览器先校验源码哈希，再 POST `/api/plugin/task`，请求包含 `source`、`sourceSha256`、`expectedKey`、`expectedVersion` 和 `marketplace:{name,index_url,path}`；后端仍负责重新编译、身份及完整性校验。
- 安装后保持禁用、未激活，需在已安装列表显式激活。SHA-256 是完整性校验，不是发布者签名。
- 不新增任意 URL 下载、后端网络代理、远程资源访问或自动激活，不删除现有扩展和原生渠道。

## 验证计划

覆盖 Admin 只读、Root 源增删保存、跨源路径与重定向拒绝、哈希不匹配时不上传、正确安装后刷新、切源时过期结果丢弃和真实语言资源。执行 Classic 定向兼容测试、ESLint、Prettier 和构建。

## 继承问题修复

本轮同时修复自定义上传阶段的 Classic 问题：Semi UI 使用独立 `TextArea` 导出；上传错误转换传入 `t` 函数；删除版本使用点击作用域内的编码路径并按具体版本的 `source_kind` 判断。源码长度按 UTF-8 字节限制，上传加互斥并在权限拒绝后撤下写入口。上传、权限撤销和实际删除 API 路径均有组件回归。

本轮新增及修复 44 个页面文案，Classic 实际七语种及历史 `zh` 别名同时写入。通过 `add-missing-keys.mjs` 脚本后执行 `bun run i18n:sync`，临时脚本已移除；Locale diff 中包含自动排序，已有其他键值不做业务修改。

## 交付状态

2026-09-16，基于 `origin/feat/custom-task-plugins@6c346504a`：

- `node --test src/pages/TaskPlugins/__tests__/i18n.test.mjs src/pages/TaskPlugins/__tests__/marketplace.test.mjs`：5 项通过。
- `node scripts/run-compat-tests.mjs --no-cache` 指定管理、市场及渠道绑定三个组件文件：25 项通过，包含源切换迟到响应丢弃、配置权限失败、源码上传与拒绝、删除路径、AtlasCloud=61 和 TaskPlugin=62 真实保存载荷。
- 涉及 7 个 JavaScript/JSX 文件的 ESLint 通过。
- `npx --yes bun@1.4.2 run build` 通过，最终源码快照的真实 Classic dist 构建耗时 33.03 秒。保留既有 Browserslist 过期、Semi UI `findDOMNode` 和大包体积警告。
- Locale 语义对比：每语种新增 36 个键；非英文修复 8 个旧占位值，无键删除。排序由脚本和 sync 完成；7 个相关 JS/JSX 文件的 ESLint、全部任务插件文件的 Prettier 及差异检查通过。

本工作项仅交付 Classic，实现依赖后端多源与 provenance 安装契约，不宣称完成真实远程源浏览器 CORS 验收、真实供应商任务执行或部署。组合构建与后端测试由协调者收口；本地生成产物不提交。
