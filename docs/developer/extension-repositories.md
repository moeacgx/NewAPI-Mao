# 扩展模块与任务插件配套仓库

| 仓库                                                                          | 职责                                           | 发布入口                                                                                       |
| ----------------------------------------------------------------------------- | ---------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| [maolaonewapi](https://github.com/moeacgx/maolaonewapi)                       | 宿主、业务接口、权限、数据库迁移与两套后台     | 主程序版本                                                                                     |
| [maolaonewapi-extensions](https://github.com/moeacgx/maolaonewapi-extensions) | 扩展模块源码、版本 ZIP、SHA-256 和在线安装目录 | [模块目录](https://github.com/moeacgx/maolaonewapi-extensions/blob/main/MODULES.md)            |
| [maolaonewapi-plugins](https://github.com/moeacgx/maolaonewapi-plugins)       | Task Plugin 源码及任务插件市场源               | [任务插件索引](https://raw.githubusercontent.com/moeacgx/maolaonewapi-plugins/main/index.json) |

扩展模块仓库收录渠道可观测性中心 `0.4.1`、对话归档 `0.1.1`、OKX 支付宝汇率 `0.3.0`、上游模型校验 `0.2.0`。
前面三个是主程序内置资源的版本快照，宿主启动仍可能按内嵌版本刷新；上游模型校验是外置模块，
`0.2.0` 需要白名单与容错宿主能力及数据库迁移。仅看清单最低版本不能判断全部业务兼容性。

模块源码在扩展仓库 `modules/`，版本安装包在 `published/<id>/<version>/`，
固定目录地址为 `https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json`。
支持在线安装的新宿主可读取目录并安装；旧宿主由 Root 下载 ZIP 后在模块管理上传。
目录存在不会自动升级宿主，也不代表当前站点已部署在线安装功能。
SHA-256 只用于完整性验证，不能当成发布者签名。模块的后台业务仍由对应宿主提供。

任务插件继续使用已有 `/api/plugin/task` 体系与 `indexVersion:1` 插件源；扩展 ZIP 不能导入任务插件入口。
任务插件安装后依照既有版本激活和启停规则，扩展首次安装保持关闭。具体契约见
[扩展开发](extensions.md)与[任务插件源发布](../workflows/2026-09/16_task_plugin_sources_plan.md)。

主仓库以文档链接关联两个独立仓库，不使用 Git submodule，不改变普通克隆、构建或部署命令。
现有宿主内置资源继续保留；扩展仓库版本不可覆盖，修订须提升版本并重新验证。
不得发布运行状态文件、数据库、渠道密钥、Bot Token 或生产配置。
