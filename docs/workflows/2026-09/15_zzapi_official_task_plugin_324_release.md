# zzapi 官方任务插件阶段版本 `.324`

## 范围

本版本基于 `custom-main@2a4dc4a58`，tag 为 `v1.0.0-rc.10.1.10.324`，包含官方内置 Task Plugin 的管理、类型 62 渠道绑定、版本 pin、轮询脱敏、公开状态与认证内联资源阶段能力。插件总开关默认关闭；现有 Extensions、原生任务和 AtlasCloud 61 保留。

发布前已完成 Default/Classic 真实前端构建、根 Go 二进制构建和后端插件定向回归。上传、市场、S3/匿名签名、真实远程媒体、完整 TokenAuth 端到端、三库现场迁移、订阅故障恢复和完整 UsageFacts 仍未纳入本次启用范围。

## 目标

仅部署 CloudSSH `API中转站 / RS2000 德国建站`，serverId=52，目录 `/home/docker/zzapi`，三应用端口 18097、18098、18099。只更新三个应用镜像；不更新 maolaoapi、zhishiapi，不重建 PostgreSQL 或 Redis。

## 回滚

滚动更新前备份 Compose 和数据库迁移前信息，保留 `.323` 镜像。健康检查或启动迁移异常时停止后续节点，并按旧镜像逐节点恢复；不删除插件归档，不覆盖业务数据库。

## 验证记录

发布、镜像构建和迁移已完成。Linux Release `34988533531`、GHCR 构建 `34988533381` 均成功；发布资产包含 amd64/arm64 二进制，GHCR 多架构 manifest 已生成。

CloudSSH server 52 作业 `6f0df8b0-41d8-430b-a990-97350d650900` 完成 Compose 备份、PostgreSQL 备份和镜像拉取；备份位于 `/home/docker/zzapi/backups/`，Compose 备份为 `docker-compose.yml.pre-324-20260915T153956Z`，数据库备份为 `postgres.pre-324-20260915T153956Z.sql.gz`（149 MiB）。

滚动更新作业 `b4803013-da51-4168-91c3-7cce636731fe` 按主节点、slave-1、slave-2 顺序完成。三个容器均运行 `ghcr.io/moeacgx/maolaonewapi:v1.0.0-rc.10.1.10.324`，健康检查通过，重启次数为 0；18097、18098、18099 的 `/api/status` 均返回 `.324`。PostgreSQL 和 Redis 容器未重建，`prefill_groups.uk_prefill_name` 索引计数为 1。

验证作业 `8b10f034-457e-4156-ba9f-b171461ccb2f` 完成三端口版本、容器、Compose 和备份检查。此次没有调用真实推理、修改账户/渠道或发送 Telegram 消息；完整 TokenAuth、真实供应商媒体和下一阶段资源能力仍按接入计划保留。
