# zzapi 官方任务插件阶段版本 `.324`

## 范围

本版本基于 `custom-main@de1579c33`，包含官方内置 Task Plugin 的管理、类型 62 渠道绑定、版本 pin、轮询脱敏、公开状态与认证内联资源阶段能力。插件总开关默认关闭；现有 Extensions、原生任务和 AtlasCloud 61 保留。

发布前已完成 Default/Classic 真实前端构建、根 Go 二进制构建和后端插件定向回归。上传、市场、S3/匿名签名、真实远程媒体、完整 TokenAuth 端到端、三库现场迁移、订阅故障恢复和完整 UsageFacts 仍未纳入本次启用范围。

## 目标

仅部署 CloudSSH `API中转站 / RS2000 德国建站`，serverId=52，目录 `/home/docker/zzapi`，三应用端口 18097、18098、18099。只更新三个应用镜像；不更新 maolaoapi、zhishiapi，不重建 PostgreSQL 或 Redis。

## 回滚

滚动更新前备份 Compose 和数据库迁移前信息，保留 `.323` 镜像。健康检查或启动迁移异常时停止后续节点，并按旧镜像逐节点恢复；不删除插件归档，不覆盖业务数据库。

## 验证记录

发布、镜像构建、迁移和三节点健康检查完成后补充 tag/source、镜像 digest、CloudSSH 作业编号、备份位置、容器版本和 PostgreSQL 索引结果。准备文档本身不代表部署成功。
