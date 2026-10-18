# 渠道可观测性中心渲染失败排查与修复边界

日期：2026-10-18

## 线上主因

线上宿主 `/api/status` 返回版本为 `dev-320e44fcfd0b`。`channel-quality`
扩展版本 `0.4.1` 要求宿主至少为 `v1.0.0-rc.10.1.10.204`；宿主版本解析
只接受合法 SemVer，`dev-*` 不会被当作兼容版本。因此
`/api/extensions/?all=true` 返回 `enabled: false` 和
`module is incompatible with current host version`，扩展页面随后显示为
不存在，用户看到渠道可观测性中心渲染失败。

线上修复应发布包含扩展门禁修复和新内置快照的宿主镜像，逐节点更新并刷新扩展注册表，再
验证模块启用状态、原生资源和 `/api/channel-analytics/*` 请求。`dev-*` 版本不再作为模块版本兼容失败条件。

截至 2026-10-18，扩展仓库 `moeacgx/maolaonewapi-extensions` 的 main
（`475a1be3192b6ca64b920154e66fe21d726c0cbe`）仍只有
`channel-quality` `0.4.1`，没有可替换的新模块版本；该包 SHA-256 为
`5e62e49b78cfefaa4895173e376bf1cc5a8071db82e3f788ce5eede1a7379675`。
因此不能通过重新安装现有扩展绕过宿主版本不兼容。

## 另一条 Default 路由缺陷

在宿主版本已经兼容时，Default 前端侧栏会向 `role >= admin` 的管理员展示
`channel-quality` 扩展的渠道可观测性入口。该扩展的 manifest 也声明
`permissions.roles: ["admin"]`，后端原生扩展资源路由允许管理员访问。但
Default 深层扩展路由
`/_authenticated/extensions/$moduleId/$pageKey/` 的前置守卫只接受
`SUPER_ADMIN`，普通管理员点击入口会被错误重定向到 403，页面表现为渠道观测
中心无法渲染。这是独立的前端权限缺陷，不能替代线上版本兼容性修复。

## 变更范围

- 生产部署先更新宿主扩展门禁和内置模块快照，再核对扩展启用状态。
- Default 模板深层扩展路由的访问门槛与扩展 manifest、后端资源权限保持一致：
  `ADMIN` 及以上可打开已授权扩展页面。
- 扩展管理页、安装、启停和卸载权限仍只授予 `SUPER_ADMIN`。
- 增加路由守卫回归测试，覆盖普通用户拒绝、管理员允许和 Root 允许三条路径。
- Classic 模板保持现有 `PrivateRoute` 与扩展页面实现，不在本次范围内。

## 安全与兼容性

该修复只调整已登录管理员访问扩展页面的前端路由门槛；后端仍通过会话、扩展
manifest 角色和原生资源白名单校验。不会扩大扩展管理接口权限，也不改变渠道
统计接口、数据存储或计费行为。

## 验证计划

- 检查线上 `/api/status` 与 `/api/extensions/?all=true`，确认宿主版本、扩展版本、
  清单错误和启用状态符合预期。
- 运行深层扩展路由测试，确认 `ROLE.USER` 重定向 `/403`，`ROLE.ADMIN` 和
  `ROLE.SUPER_ADMIN` 可通过前置守卫。
- 运行 Default 前端类型检查、受影响文件 lint 和构建。
- 使用管理员会话打开 `/extensions/channel-quality/index`，确认原生渠道可观测
  页面及其 `/api/channel-analytics/*` 请求正常；再确认 Root 扩展管理页仍受保护。
- 执行 `git diff --check`。

## 验证结果

- 扩展仓库 `main@475a1be3192b6ca64b920154e66fe21d726c0cbe` 的
  `channel-quality` 仍只有 `0.4.1`；没有可直接替换的模块更新包。
- 在公开扩展仓库隔离副本执行 `python scripts/validate.py`，结果为
  `PASS: 5 个版本的 ZIP、原生资源、清单与 SHA-256`；其中
  `channel-quality-0.4.1.zip` 的 SHA-256 为
  `5e62e49b78cfefaa4895173e376bf1cc5a8071db82e3f788ce5eede1a7379675`。
- 主仓库执行 `go test ./extension -count=1 -timeout 60s`，通过；Default
  深层扩展路由回归已覆盖普通用户拒绝、管理员放行和 Root 放行。
- Default/Classic 原生入口依赖与宿主 SDK 模块清单已核对，现有模块资源没有
  可由模块包单独修复的缺失项；Classic 不在路由修复范围内。
- 本轮没有安装、发布或修改生产模块。要恢复线上页面，必须先部署包含门禁修复和新
  内置模块快照的宿主版本并逐节点刷新扩展注册表，再进行管理员会话验收。

## 后续决策：移除宿主版本门禁

后续核对确认 `dev-*` 版本是开发宿主的正常版本标识，不能作为模块不可用的理由。
因此宿主已改为不再使用 `host.min/max` 阻断扫描、安装、启用和代理；清单字段仅保留
格式校验和目录展示。Native SDK、后端接口、权限与能力声明仍继续检查。生产恢复应改为
发布包含新内置模块快照和该门禁修复的宿主镜像，再按节点健康门禁滚动更新。
