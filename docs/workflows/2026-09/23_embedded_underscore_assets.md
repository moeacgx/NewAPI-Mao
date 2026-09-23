# 下划线静态资产嵌入与 404 缓存修复

## 现象与根因

v333 三个源站的入口 `index-Ba4qM9gi.js` 均可正常返回，而构建清单中出现的
`_baseUniq-leTQd-Cl.js` 和 `_basePickBy-BDFb_vAM.js` 均返回 404。Go 的
`//go:embed web/.../dist` 默认忽略名称以 `.` 或 `_` 开头的文件和目录；前端构建
成功并不代表这些合法分块进入最终 Go 二进制。

Web 缓存中间件还会在处理结果未知时为除根路径外的请求预置
`Cache-Control: max-age=604800`。缺失分块进入 `/assets` 的 404 分支后没有覆盖该
响应头，浏览器或 CDN 可以把缺包结果负缓存一周。已经确定的是二进制遗漏合法
构建产物且 404 可被长期负缓存；仅出现在 Vite `__vite__mapDeps` 清单不能证明
启动时必然执行，是否直接触发本次白屏仍需用静态 import 链或浏览器 DOM/控制台
证据闭环。

## 修改范围

- Default 与 Classic 的构建目录统一改为 `//go:embed all:<dist>`，允许合法的
  下划线和点前缀构建产物进入二进制；独立的 `index.html` 嵌入保持不变。
- Web 路由对 `/assets` 的未找到响应在写出 404 前覆盖为
  `Cache-Control: no-store`，禁止沿用一周缓存。
- 已存在的哈希静态资产继续返回原有一周缓存，不关闭全局网页限流，也不让缺失
  资产绕过 GW。真实存在的静态资产仍按现有规则绕过 GW。
- 不修改 Default 或 Classic 的前端源码、分块命名和业务行为。

## 回归合同

- 后端测试构建前在两套 dist 中生成下划线 fixture，根包测试直接从实际
  `defaultBuildFS` 和 `classicBuildFS` 读取，防止未来把 `all:` 改回默认模式。
- Web 路由用真实静态文件系统验证：存在的下划线哈希资产返回 200、原内容和
  `max-age=604800`；不存在的资产返回 404 且实际 HTTP 响应头不含长期缓存。
- 路由仍把缺失资产交给现有 404 响应，GW 和主题缓存分块兜底的边界不改变。

## 验证与发布边界

实现后执行根包嵌入测试、Router 测试、相关 Go 测试和 `git diff --check`。发布
镜像还必须在构建后的二进制或源站实际读取已知下划线分块；仅入口 200、健康检查
或源码存在不能证明修复完成。本工作项不执行生产部署。

本地验证结果：

- 修改前，实际根包 `embed.FS` 无法读取 Default 和 Classic 的下划线 fixture；修改
  后两者均可读取。
- 修改前，真实 Web 路由的缺失资产最终响应仍携带 `max-age=604800`；修改后为
  HTTP 404 与 `Cache-Control: no-store`，存在的下划线资产仍为 HTTP 200 与一周
  缓存。
- `go test . -run TestFrontendBuildEmbedsUnderscoreAssets -count=1 -timeout 60s` 通过。
- `go test ./router ./middleware -count=1 -timeout 60s` 通过。
- `git diff --check` 通过。
- 当前 Windows 环境未安装 Bun，无法本地重新生成两套真实前端 dist；固定版本
  CI/Release 构建仍需验证真实 `_baseUniq-*`、`_basePickBy-*` 文件进入最终二进制。
