# 扩展模块独立仓库与在线安装

用户明确授权建立公开扩展模块仓库、上传现有四个模块、关联主仓库和任务插件仓库，并同时实现 Default、Classic 的站内在线安装。
前一项模型校验 0.2.0 的已验证工作区改动保留，不能被此项覆盖；线上部署不在本项范围。

## 仓库与发布

- 主程序：`https://github.com/moeacgx/maolaonewapi`。
- 扩展模块：`https://github.com/moeacgx/maolaonewapi-extensions`，公开源码、版本 ZIP、SHA-256、下载清单。
- 任务插件：`https://github.com/moeacgx/maolaonewapi-plugins`，保持现有 Task Plugin 安装机制。
- 主仓库通过 README 和开发文档关联两个仓库，不引入 Git submodule 或改变构建依赖。
- 本期固定官方维护源，不新增自定义源管理、远程执行、自动升级或通知能力。

## 在线安装契约

Root `GET /api/extension-admin/marketplace` 返回 `data`：

```json
{
  "catalog_url": "https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json",
  "repository_url": "https://github.com/moeacgx/maolaonewapi-extensions",
  "host_version": "宿主当前版本",
  "max_archive_bytes": 104857600
}
```

浏览器直接下载清单及 ZIP，使用 `credentials: omit`、`referrerPolicy: no-referrer`、`redirect: error`，
清单上限 2 MiB、ZIP 上限 100 MiB，读取响应流时限制实际字节。路径只能在清单同源同目录的
`published/<id>/<version>/<id>-<version>.zip` 下，不允许绝对路径、目录穿越、编码逃逸、查询或片段。
下载完成以 Web Crypto 核对 SHA-256，再提交宿主；下载与解析不得执行模块 JavaScript。

清单结构：`{catalogVersion:1,purpose:"extension-catalog",name,modules:[...]}`。
`modules` 每项是一个明确版本，必填 `id/name/version/path/sha256/size/host`，另有
`description/capabilities/compatibility/source/status` 描述兼容边界。版本不能重复，SHA-256 为 64 位十六进制。
页面显示模块名称、版本、简介、宿主要求/开发状态以及安装操作；安装前确认模块和版本，安装后刷新列表。

复用 Root `POST /api/extension-admin/upload` multipart：

- 原有 `file` ZIP。
- 在线安装额外提交 `archiveSha256`、`expectedId`、`expectedVersion`、`catalogUrl`、`archivePath`。
- 五个字段全填或全省略，部分元数据拒绝。`catalogUrl` 必须等于当前固定源，`archivePath` 必须匹配预期 ID/版本。
- 宿主不访问上述 URL，只读取上传字节，复核 hash、实际清单身份/版本、宿主兼容性及现有安全解包规则后安装。
- 失败不得替换现有模块及其启用状态；成功沿用现有首次安装关闭、更新保留已有启用状态的规则。
- SHA-256 仅为完整性校验，Root 安装仍意味着信任该模块；不表述为发布者签名。

## 验证

后端：Root 权限、固定源、合法真实 ZIP、hash/ID/version/path 不匹配、字段缺失、上限和失败保持原安装。
双模板：列表加载/失败/空态、明确版本确认、下载完整性与路径边界、安装请求及刷新、Root 入口和七语。
仓库：四个版本原生资源齐全、源码归属、可复现 ZIP、哈希、已有发布不可改写、GitHub Raw 可读与 CORS。
集成：隔离宿主从实际公开仓库下载后上传安装，分别核对两套页面；不执行生产安装。

## 发布与验收结果

扩展仓库已公开，首个提交 `8fe51f4`，仓库 CI `35357601468` 通过。
四个版本为 `channel-quality 0.4.1`、`conversation-archive 0.1.1`、`okx-alipay-rate 0.3.0`、
`upstream-model-guard 0.2.0`。保留来源代码与 AGPL 许可，发布包不可覆盖；三个内置模块注明宿主快照边界。
目录及四个 ZIP 的 GitHub Raw 均 HTTP 200、CORS `*`，实际下载字节与目录 hash 一致。
主仓库关联 PR [#233](https://github.com/moeacgx/maolaonewapi/pull/233) 已合并，提交 `c2173d294`。

- 后端完整相关包、市场定向矩阵及 Go 构建通过。四个真实公开包分别通过本地安装器和 multipart 上传，
  错误 SHA 更新均保留原版本、文件和启用状态。非文本/重复/部分来源元数据拒绝，不能静默退回手工上传。
- Default 下载、组件与 URL 测试 40/40；Classic 组件 7/7、安全与 i18n 8/8；类型、lint、格式检查通过。
  两套新文案均覆盖七语；Classic 的旧中文别名同步保留。
- Classic 标准生产构建通过。Default 构建仍遇已记录的 Windows C/D 盘依赖 Junction 字体路径限制，
  使用忽略目录中的临时路径适配脚本完成本地生产构建，没有改动仓库构建配置。
- 使用最终两套生产前端与当前 Go 宿主，隔离端口 3191 的真实 Chromium 完成：
  Default 确认并从公开 Raw 下载模型校验 0.2.0，再上传安装；Classic 从同一公开源安装 OKX 0.3.0。
  断言外部请求无 Cookie/Authorization/Referer，首次安装关闭、更新保留开启状态，安装后列表刷新。
  两次在线安装均返回成功，四个外部请求满足凭据隔离，页面异常为 0；1440px/390px 无页面横向溢出。
  截图位于忽略目录 `.local-tests/marketplace-browser/`。
- Windows 本地首次启动曾遇内置资源目录 rename 被占用，复制相同内置资源后重启通过；
  测试使用独立旧 SQLite 副本和本地账号，不读取生产凭据、不修改生产数据。

源码变更含上一工作项已验证的容错宿主支持，以便当前公开 0.2.0 包有对应实现。
本次不部署线上实例；他站需升级包含本项的宿主，旧版本继续下载 ZIP 手工安装。
