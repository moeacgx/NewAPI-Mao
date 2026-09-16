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
CI `35055704742` 成功。发布镜像及部署现场验证记录在完成后补充。
