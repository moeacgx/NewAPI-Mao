# zzapi 自定义插件与多源版本 `.325`

## 发布范围

用户授权合并并部署到 zzapi。功能 PR #225 已于 2026-09-16 合并到 `custom-main`，
合并提交 `470ee522612d92b338569dd13e392064d8c660b3`，功能 HEAD `9151cee04` 的前后端 CI 均成功。
本版本加入 Root 临时源码上传、Default/Classic 多源市场、版本预览与 hash 校验安装；
自维护源使用独立公开仓库 `moeacgx/maolaonewapi-plugins`，初始索引为空。

安装默认未激活、禁用；部署不自动开启插件总开关或安装第三方代码。
保留原生任务、Extensions、AtlasCloud=61 和 TaskPlugin=62。S3、远程媒体、在线试运行等阶段边界见多源工作记录。

## 部署目标和步骤

CloudSSH 项目 `API中转站`、主机 `RS2000 德国建站`，serverId=52、hostId=17，
工作目录 `/home/docker/zzapi`，应用端口 18097/18098/18099。
先核对实际 Compose 服务、容器身份和旧版本，再备份 Compose 及 PostgreSQL；
拉取固定镜像 `ghcr.io/moeacgx/maolaonewapi:v1.0.0-rc.10.1.10.325`，逐个重建应用服务。
每个节点通过健康检查与 `/api/status` 版本验证后再继续下一个。
PostgreSQL 和 Redis 不重建，主项目 maolaoapi、zhishiapi 不在部署范围。

## 回滚

升级前保留 `.324` 镜像和 Compose 备份。若任何节点迁移或健康检查失败，停止后续节点，
将对应应用镜像恢复到旧版本；不删除插件表或回写业务数据库。
迁移新增来源字段应兼容已有数据，历史任务 pin 与退款记录必须保留。

## 发布前证据

Default 158 项插件测试、Classic 21 项管理/市场组件测试及 5 项字节/语言检查通过。
两套真实前端生产构建、根应用 Go 构建、相关 vet 与后端定向测试通过；
CI `35055704742` 成功。发布镜像及部署现场验证记录如下。

## 发布与备份记录

版本准备 PR #226 的 CI `35056760042` 成功，合并提交为
`5d95c6994184af6b3ad6cd8856ff85d74bbb681c`，`.325` 标签固定指向此提交。
Linux Release `35057170449` 成功，发布资产包含 amd64、arm64 二进制和
`checksums-linux.txt`。GHCR 发布工作流 `35057170456` 的两架构构建、签名及多架构 manifest 均成功。

CloudSSH 备份作业 `299e4514-81b3-4d59-a760-cfc2a2ff4122` 成功。
备份目录为 `/home/docker/zzapi/backups/pre-325-20260916T044402Z`，保存 Compose、
已有环境文件和不含凭据的容器身份快照；数据库为约 136 MiB 的 PostgreSQL 自定义格式逻辑备份。
`pg_restore --list` 已通过，不代表执行过完整恢复演练，也不包含 Redis 数据快照。

- `database.dump` SHA-256：`a653287c491dca6c36fa857abde4d042cf0f0c822099d7629c4a548bca32ed03`。
- `docker-compose.yml` SHA-256：`294f1cf7a8c4130f5ab5033921a75de795e4f0301ec43dc6cf407be220bcbaf7`。

部署前作业 `e07e9675-4756-4893-a8a0-03740523a78a` 核实 `TaskPluginEnabled=true`，
共有 10 个插件版本。本次保持现有总开关与每个版本的激活、启用状态，不自动安装插件。

## 现场验收

2026-09-16 完成镜像拉取与三个应用滚动更新。镜像准备作业为
`7167c5ad-780a-46e1-a914-b0b846bf8aed`，仅替换 Compose 的三个应用镜像。

- 多架构 digest：`sha256:834515597e60057254df27320ed9aec537282169e4190c19d4e7b180b46fe968`。
- 现场 amd64 image ID：`sha256:ccd4fd168e19bac0c247e6518b1c09c0833cf582b53ab28e21aa81d22d34e74a`。
- `zzapi` / 18097：作业 `73fd2c87-89c9-4c8e-a51b-fba0dec30578`。
- `zzapi-slave-1` / 18098：作业 `f0f74c19-5203-46c6-96f2-3cd4cf553e7c`。
- `zzapi-slave-2` / 18099：作业 `9a311c40-3775-44f9-bcae-f456b8359e26`。

最终验证作业 `7f137a25-28e7-47a3-a383-6345a7aa71fd` 成功：

- 三应用均为 `.325`，实际 image ID 一致，健康状态为 `healthy`，重启次数为 0。
- 三端口 `/api/status` 均返回 200 和 `.325`；公网 `https://zzapi.maolaoapi.com/api/status`
  独立返回 200、`success=true` 和 `.325`。
- PostgreSQL、Redis 的容器 ID 和启动时间与备份前相同，均运行中，重启次数为 0。
- PostgreSQL 的 `task_plugins.source_kind` 为 `character varying`，`marketplace` 为 `text`；
  本次部署确认此实例的插件来源字段迁移，不代表全部历史迁移或 MySQL 现场验收。
- `TaskPluginEnabled` 仍为 `true`，10 个插件版本的激活、启用状态与部署前逐项一致。
- 三节点的插件管理、市场源及 runtime 接口无身份请求均返回 401；
  `/console/task-plugins` HTML 与其本地入口脚本均可读取。此项不替代已登录浏览器操作验收。
- 各应用启动后最后 300 行日志未发现 panic/fatal 或标准 ERROR 标记。

首次单节点脚本因 Windows CRLF 在 `set -euo pipefail` 处退出，未执行到容器重建；
改为明确 LF 字节后按上述作业完成。Compose 提示既有 `version` 字段过时，不影响本次更新，未夹带配置清理。

本次未调用真实供应商或执行付费任务，未新增、安装或激活插件，未修改用户、渠道或现有扩展配置。
完整订阅退款及故障恢复、真实远程媒体交付等边界仍按专题文档执行。跨项目接口与计费契约无新增调整。
