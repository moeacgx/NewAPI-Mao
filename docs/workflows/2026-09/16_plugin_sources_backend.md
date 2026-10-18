# 多插件源后端阶段工作记录

日期：2026-09-16
基线：`origin/feat/custom-task-plugins`（PR #225 续作）

## 目标与范围

本阶段为 Task Plugin 增加可配置的市场索引源管理接口，并为后续浏览器侧市场浏览/安装保留稳定路由边界。后端仅保存和返回源配置，不下载索引或源码，也不把来源 URL 当作签名凭据。实现范围限定为 Go 后端路由与配套契约；Default、Classic 和 plugins 仓库索引由主协调者处理。

## HTTP 契约

- `GET /api/plugin/task/marketplace/sources`：Admin 可读，响应沿用 `common.ApiSuccess`，`data` 为裸数组，每项 `{name,index_url}`。
- `PUT /api/plugin/task/marketplace/sources`：仅 Root，叠加 `CriticalRateLimit`，请求体为裸数组，允许显式 `[]`，最多 16 项。名称最长 128 字符；URL 必须为 HTTPS，禁止 userinfo、query、fragment，长度不超过 2048；空名称、重复名称或重复 URL 均拒绝。
- 数据库配置不存在时由服务层提供两个默认源：`https://www.newapi.ai/api/v1/plugins/index.json` 与独立公开仓库 `https://raw.githubusercontent.com/moeacgx/maolaonewapi-plugins/main/index.json`；不使用私密主仓库的 `plugins/index.json`。显式空数组表示清空，不回退默认值。每次读取走数据库，保证多节点即时可见。

## 安全与兼容边界

市场源管理不改变 `POST /api/plugin/task` 的手动 `{source}` 入口。市场安装必须由服务层校验完整 provenance、SHA-256、插件 Meta、当前来源配置及路径安全规则；网关不代浏览器下载 URL。安装版本默认 `inactive/disabled`，总开关默认关闭。源被替换或移除时，已安装版本及历史 pin 保持不变。

路由注册保持已有 Admin/Root 中间件层级：市场 GET 继承 `DisableCache`、全局限流和 Admin 鉴权；PUT 继承 Root 鉴权与 `CriticalRateLimit`。市场路由显式置于 `/:key` 参数路由之前，避免被当作插件 key 解析。

## 验证记录

- 已检查工作树基线及现有任务插件路由/权限测试。
- 新增市场 GET/PUT 路由注册，控制器函数为 `GetTaskPluginMarketplaceSources` 与 `SetTaskPluginMarketplaceSources`。
- 协调工作树已组合控制器、模型、路由及两个模板。`TestTaskPluginMarketplaceInstallContract` 通过真实登录会话与 Admin/Root 路由，验证配置写权限、合法插件安装、禁用/未激活默认值、幂等安装保留状态/来源/备注，以及 hash、Meta、来源和编码路径拒绝。
- 源配置读取测试通过数据库更新后立即可见的断言，本地 OptionMap 保持旧值；空数组不回退默认源，非法 URL 不改写已有配置。
- `TestTaskPluginUploadBodyLimit` 验证 8 MiB 请求体 413；合法安装用例另验证 1 MiB 源码上限、手动上传兼容、编译错误脱敏、版本冲突 409，以及移除源后历史 pin 可读。
- 修正过路径字符集合误拒正常版本、编码分隔符遗漏及测试 LOG_DB 未初始化。最终六包定向测试、相关 `go vet` 和包含两套真实前端产物的根应用构建通过。

## 已知限制

网关不代浏览器远程抓取。源码编译和身份复核已在控制器测试中验证；SQLite 测试不代表 MySQL/PostgreSQL 实库迁移验收。来源由 Root 声明并按配置与 hash 复核，不代表发布者签名或生产环境可用性。运行时原有资源、计费和故障恢复阶段边界保持。
