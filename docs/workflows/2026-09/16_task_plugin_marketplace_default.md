# Default 多插件源与插件市场

本阶段基于自定义上传分支，新增插件源列表管理和按源浏览，不改变插件运行时与原生渠道。

- 管理员只读浏览配置及索引；Root 可保存多个源、预览下载源码并安装。
- 源配置由服务端提供，默认官方源与 MaoLao 源；客户端只请求当前选中的源。
- MaoLao 稳定源使用独立公开仓库 `https://raw.githubusercontent.com/moeacgx/maolaonewapi-plugins/main/index.json`。每个插件可选择索引中的具体版本，预览下载、hash 校验与安装身份始终绑定同一个版本快照。
- 浏览器使用无凭据 HTTPS 请求读取索引与同源相对源码路径，禁止重定向。
- 索引限制 2 MiB，源码限制 1 MiB；安装必须有合法 SHA-256 且匹配下载的原始字节，解码要求合法 UTF-8，保留 BOM。
- 上传请求带 source、sourceSha256、expectedKey、expectedVersion、remark 及 marketplace（name/index_url/path），经既有 Root 上传编译与准入，保存后仍禁用未激活。
- 临时手工上传保留；市场安装不自动启用版本、总开关或配置渠道。
- 索引元数据仅用于展示，SHA-256 仅证明与索引一致，并非发布者签名。
- 临时上传按 UTF-8 字节计数限制大小；版本删除逐条按 source_kind 判断，内置当前版本不会遮蔽历史自定义版本的删除入口，拒删时保留列表及确认框。

接口：GET/PUT /api/plugin/task/marketplace/sources，PUT 请求为 {name,index_url} 数组；安装 POST /api/plugin/task。

## 验证

- 基线 `origin/feat/custom-task-plugins@6c346504a`，复用固定上游 `upstream/main@9fe0457ee` 的索引解析和源编辑结构，并收紧路径与下载边界。
- 插件 feature 测试 156 项通过：含索引解析、路径隔离、响应大小、重定向、hash 与 UTF-8、Admin 浏览与 Root 安装载荷、选择历史版本时 path/hash/version 一致、读取失败不能空写配置、中文上传字节限制及逐版本删除拒绝后的状态。
- 七语言真实 i18next 资源、禁用 fallback 的显示验证通过；翻译通过脚本和 `i18n:sync` 写入。
- Default 类型检查、feature 定向 oxlint、生产构建通过（Bun 1.4.2，首次构建 11.1 秒，后续增量构建通过）。
- 仅 Default 与本工作记录；Classic 和后端分别由其责任 Agent 实现。公共开发索引与能力总览由协调者更新，最终组合门禁仍需执行。

本地测试使用受控 fetch/API 夹具，未调用真实供应商，也未完成远端源 CORS 或真实浏览器交互验收。市场来源不授予额外运行时能力；远程媒体、签名和沙箱仍不开放。
