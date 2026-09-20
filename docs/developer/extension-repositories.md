# 扩展模块与任务插件配套仓库

| 仓库                                                                          | 职责                                           | 发布入口                                                                                       |
| ----------------------------------------------------------------------------- | ---------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| [maolaonewapi](https://github.com/moeacgx/maolaonewapi)                       | 宿主、业务接口、权限、数据库迁移与两套后台     | 主程序版本                                                                                     |
| [maolaonewapi-extensions](https://github.com/moeacgx/maolaonewapi-extensions) | 扩展模块源码、版本 ZIP、SHA-256 和在线安装目录 | [模块目录](https://github.com/moeacgx/maolaonewapi-extensions/blob/main/MODULES.md)            |
| [maolaonewapi-plugins](https://github.com/moeacgx/maolaonewapi-plugins)       | Task Plugin 源码及任务插件市场源               | [任务插件索引](https://raw.githubusercontent.com/moeacgx/maolaonewapi-plugins/main/index.json) |

扩展模块仓库收录渠道可观测性中心 `0.4.1`、对话归档 `0.1.1`、OKX 支付宝汇率 `0.3.0`、上游模型校验 `0.2.1`（保留旧版 `0.2.0`）。
前面三个是主程序内置资源的版本快照，宿主启动仍可能按内嵌版本刷新；上游模型校验是外置模块，
`0.2.0` 需要白名单与容错宿主能力及数据库迁移。仅看清单最低版本不能判断全部业务兼容性。
`0.2.1` 在双模板增加渠道白名单 ID 直接输入，沿用同一宿主能力，无新增迁移；已发布到在线目录，安装由 Root 在后台执行，多节点须分别刷新模块注册表。

模块源码在扩展仓库 `modules/`，版本安装包在 `published/<id>/<version>/`，
固定目录地址为 `https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json`。
支持在线安装的新宿主可读取目录并安装；旧宿主由 Root 下载 ZIP 后在模块管理上传。
目录存在不会自动升级宿主，也不代表当前站点已部署在线安装功能。
SHA-256 只用于完整性验证，不能当成发布者签名。模块的后台业务仍由对应宿主提供。

任务插件继续使用已有 `/api/plugin/task` 体系与 `indexVersion:1` 插件源；扩展 ZIP 不能导入任务插件入口。
任务插件安装后依照既有版本激活和启停规则，扩展首次安装保持关闭。具体契约见
[扩展开发](extensions.md)与[任务插件源发布](../workflows/2026-09/16_task_plugin_sources_plan.md)。

## 主仓库中的子模块入口

主仓库根目录通过 Git submodule 关联两个独立仓库：

- `maolaonewapi-extensions/`：扩展模块仓库。
- `maolaonewapi-plugins/`：任务插件仓库。

GitHub 文件列表会显示两个可点击的 `仓库名 @ 提交` 入口。`.gitmodules` 使用公开 HTTPS 地址，
主仓库的 gitlink 固定各子仓库的具体提交；`branch = main` 只指定主动更新时的目标分支，
不会在克隆、构建或启动宿主时自动跟进新版本。

首次克隆且需要同时获取模块源码时：

```sh
git clone --recurse-submodules https://github.com/moeacgx/maolaonewapi.git
```

已有主仓库克隆时，在根目录执行：

```sh
git submodule update --init --recursive
```

维护子仓库指针时，先在对应独立仓库提交并发布；确认其工作区干净后，按需更新主仓库的指针：

```sh
git submodule update --init --remote maolaonewapi-extensions
git diff --submodule=log
git add maolaonewapi-extensions
```

更新任务插件仓库时，将命令中的路径替换为 `maolaonewapi-plugins`。核对版本和差异后，
把 gitlink 变更作为主仓库 PR 提交；其他克隆拉取主仓库后，再运行初始化命令同步到固定提交。
回滚指针同样通过主仓库 PR 恢复先前提交，不会改写子仓库历史。

普通克隆可以不初始化子模块，宿主构建与运行不依赖这两个目录；`.dockerignore` 排除它们，
避免递归克隆后将模块源码和安装包带入宿主 Docker 构建上下文。
在线安装仍读取独立仓库 `main` 上的目录，和主仓库固定的 gitlink 提交相互独立。
本变更不修改 Default、Classic 的页面、安装接口或生产部署。

验证入口时，检查 `git submodule status`、`.gitmodules` 地址及 `git ls-files --stage` 中两个
`160000` 条目；合并后核对 GitHub 根目录入口和其目标提交。

现有宿主内置资源继续保留；扩展仓库版本不可覆盖，修订须提升版本并重新验证。
不得发布运行状态文件、数据库、渠道密钥、Bot Token 或生产配置。
