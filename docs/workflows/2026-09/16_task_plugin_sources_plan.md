# 任务插件多源与仓库发布

## 目标与分工

临时插件继续由 Root 上传；审核稳定的版本进入独立公开仓库 `moeacgx/maolaonewapi-plugins`，通过 index.json 供实例按版本安装。
默认提供 NewAPI 官方源和 MaoLao 自维护源，支持添加、修改和移除多个源，保留独立的 Extensions。
后端、Default、Classic 在各自工作树实现；协调者维护发布目录、契约、组合验证和 PR。

## 固定接口

- Admin `GET /api/plugin/task/marketplace/sources`：`{success,message,data:[{name,index_url}]}`。
- Root `PUT` 同路径：请求裸数组，最多 16 项，显式 `[]` 表示没有源。仅配置不存在时加载默认值。
- 源名最多 128 字符，索引地址最多 2048 字符，使用 HTTPS 且不得包含凭据、查询参数或片段。
- Root `POST /api/plugin/task`：保留手动 `{source}`，市场安装传入
  `{source,sourceSha256,expectedKey,expectedVersion,remark,marketplace:{name,index_url,path}}`。
- 后端复核原始源码 hash、编译后的 key/version 以及所选源当前仍在配置中；保存来源仅作追溯，
  不证明发布者身份。不会由服务器请求上传的 URL。
- 版本冲突返回 409，源码超过 1 MiB 返回 413，非法请求/校验失败返回 400，响应不携带源码错误片段。
- 新版本保持未激活、禁用；同 hash 重复安装不改变既有状态或来源。删除源不影响已装版本与历史 pin。

## 索引与读取

兼容官方 `indexVersion:1`、`plugins[]`，条目包含 key/name/latest/versions。
每个版本包含 version/path/sha256，可声明 kind/minApiVersion。市场安装必须有 SHA-256，且支持 task/API 1。
浏览器按需读取单个源，索引上限 2 MiB，源码上限 1 MiB；请求不带 cookie、Authorization 或 Referer，
拒绝重定向。源码路径限定在索引目录内，拒绝跨源、绝对路径和编码后的目录逃逸。
按原始字节校验 hash、严格 UTF-8 解码，预览源码并确认后才上传，不在浏览器执行插件。

## 自维护源

默认地址为 `https://raw.githubusercontent.com/moeacgx/maolaonewapi-plugins/main/index.json`。
插件仓库单独公开，不依赖主程序仓库可见性，也不需要分发 GitHub 凭据。稳定插件使用独立发布目录，
不能为了让索引有内容把未实测插件标成稳定。首次索引可为空，页面显示准确空状态。
后续按插件仓库 README 添加 `published/<key>/<version>/plugin.js` 和验证记录，在主程序仓库运行
`go run ./cmd/task-plugin-index -root <插件仓库目录>` 编译并生成校验和，经 PR 审查后发布；
已发布的同版本源码不可覆盖。生成器保留旧 latest；维护者审查后显式指定要升级的版本。
插件仓库的 Node 校验 CI 检查目录、原始字节 hash、UTF-8、大小和相对上一提交的历史不可变性，
不依赖主程序仓库访问权限。主程序中的生成器不把插件源码提交到主程序仓库。

独立仓库已创建，初始发布 `3f2057b553e84d51d9718f31e56b8b5f00be00e8`；索引 HTTP 200、
`Access-Control-Allow-Origin: *` 已实测，仓库 CI `35053704216` 通过。索引为空，未将示例发布为稳定插件。

## 门禁

后端验证权限、空配置、数据库读取一致性、hash/Meta/来源与幂等。两套前端分别验证多源配置、
读取失败与空态区分、目录逃逸、字节上限、安装确认、Admin 只读和真实多语资源。
组合时使用两套真实前端构建产物完成根 Go 构建；不把占位 HTML 的 CI 当作完整二进制验证。
本次不部署，不自动安装/激活远程插件，不开放远程媒体代理、S3、匿名签名或 dry-run。

## 组合验证

- 整合 Classic `b9a3d7b89`、Default `4cb59b352` 及后端工作树提交后，由协调者补齐合法安装和跨节点源配置读取测试。
- Default 158 项插件测试通过，另增的损坏索引与编码路径回归包含在该总数内；类型检查与定向 lint 通过。
- Classic 21 项管理/市场组件测试、5 项索引字节/语言资源检查通过；作者额外渠道绑定检查独立记录，不累计为同一测试集。
- 两套前端均通过实际生产构建（Bun 1.3.14），Classic 保留既有 chunk 与浏览器数据库告警。
- controller/service/model/router/setting/cmd/task-plugin-index 定向测试通过；setting 在该筛选下没有独立用例，校验器由路由/模型用例覆盖。相关 vet 与根 Go 构建通过。
- 插件仓库的空索引、生成器 `-check` 与 Node 校验/测试通过，公开 Raw 索引 200/CORS 可读；官方站点本机 TLS 探测失败，未宣称官方站点当前网络可用。

本次没有生产调用或部署。未用真实供应商做市场安装后的付费任务测试，未完成 MySQL/PostgreSQL 实库迁移。
