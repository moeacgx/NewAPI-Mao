# 网站 SEO 设置

## 范围

为站点管理员提供浏览器标题和一句话网站描述配置。`SystemName` 继续作为站点名称；新增 `SystemDescription` 仅承载公开站点描述，不包含关键字、canonical、robots、sitemap 等 SEO 功能。Default 与 Classic 各自提供设置表单，并共用后端 Option 与状态 API。

## 配置与接口

- `SystemName`：现有选项，作为 `<title>`、`meta[name="title"]`、`og:title` 和 `og:site_name` 的值。
- `SystemDescription`：新持久化 Option，默认空字符串，允许清空，最多 200 个 Unicode 码点。
- `GET /api/status`：公开只读返回 `system_name` 与 `system_description`，供前端状态刷新使用。
- `GET /api/option/` 与 `PUT /api/option/`：继续由现有 Root 权限保护；新字段只通过通用 Option 写入，不添加匿名写接口。
- 写入沿用 Option 数据库事务、校验、内存 OptionMap 更新和现有多节点配置同步。未配置的旧数据库按空描述处理。

## HTML 与安全

Go Web Router 在每次应用 HTML 请求中，从 `OptionMap` 的读锁快照获取最新名称和描述，输出动态 `<title>`、唯一 `meta[name="title"]`、唯一 `meta[name="description"]`、`og:title`、`og:site_name` 和 `og:description`。`/`、`/index.html` 与 SPA 路由都返回当前主题的已更新 HTML，并设置 `Cache-Control: no-cache`。真实静态资源、主题资源兜底、静态项目版权和 generator、SVG 内部标题及已有脚本（含统计脚本）保持原样。

标题和元数据按 HTML 文本/属性上下文转义，不将设置值作为脚本或标签注入。描述为空时移除 `meta[name="description"]` 与 `meta[property="og:description"]`，不采用模板里的静态描述；Option 和状态 API 仍保留空字符串，后续 HTML 请求直接体现保存、清空后的值。根页中间件完成响应后终止处理链，避免后续静态服务追加第二份 HTML。

## 前端行为

- Classic `PageLayout` 在 `/api/status` 成功后以服务端最新 `system_name` 更新 `document.title` 与相关 title meta，使用 `system_description` 更新 description/Open Graph 描述；空描述清空旧值。旧 `localStorage` 名称不得覆盖服务端 HTML 初始标题。
- Default 的启动品牌同步对服务端首屏标题、title meta 和描述元数据执行相同更新；缓存的 Logo 可先显示，缓存的名称不覆盖服务端首屏标题。
- 两套个性化/站点设置表单写入同一个 `SystemDescription` Option，允许空字符串保存；不把站点实例名称或描述硬编码进代码。
- 保存成功后立即更新当前页对应元数据；业务失败响应不更新元数据。其他节点仍按既有 `SyncOptions` 周期同步；本改动不提供跨节点即时一致性。
- 名称/描述保存被业务拒绝时仅显示错误，不显示保存成功；保留输入以便原值重试。Default 表单回调以 `false` 表示业务未保存，保留 dirty 与原 baseline；现有返回 `void` 的表单仍按原成功路径处理。
- 保存成功后，Default 同步扁平 `['status']` 查询缓存、本地 `status` 和名称配置 store；Classic 同步 `StatusContext` 与 `setStatusData`，让页头与元数据使用相同新值。仅替换本次保存字段，保留其他状态；不额外请求 `/api/status`，避免异步读取未同步节点的旧值覆盖本次保存。

## 验证

后端回归覆盖自定义名称和中文描述、更新/清空、恶意字符转义、元数据唯一性，以及 Default/Classic 在 `/`、`/index.html`、`/console/log` 下的动态输出、no-cache 和静态资源边界。前端使用两模板既有 Vitest/RTL 测试设施验证服务端状态更新 DOM 与清空语义、表单保存行为及 Classic 用户筛选布局。涉及 Go 包执行有界测试；涉及 Default TypeScript 执行 typecheck；运行两模板受影响测试、构建、lint/格式检查。工作记录见 [网站 SEO 与 Classic 日志筛选工作记录](../workflows/2026-09/23_site_seo_and_classic_log_filters.md)。
