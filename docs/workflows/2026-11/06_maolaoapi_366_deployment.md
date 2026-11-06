# maolaoapi .366 逐节点更新

## 目标与现场

用户授权逐个更新 maolaoapi 容器。2026-11-06 经 CloudSSH 核对目标为
「API中转站 / OVH | NEWAPI」，serverId 63、hostId 29。
Compose 项目为 `maolaoapi-ovh`，配置为 `/apps/maolaoapi/docker-compose.json`。
三个应用节点均为 `.362`、healthy、restart=0。

目标为已发行的 `v1.0.0-rc.10.1.10.366`，固定镜像摘要：
`ghcr.io/moeacgx/newapi-mao@sha256:0d80892c22b7e1edf191b3e884118a774f787759418d9dc5840879c1916f716f`。
发行和双模板验证见 [.366 发布记录](06_release_366.md)。

## 实施与验收计划

1. 保存 Compose、容器配置、数据库 custom dump、schema 和持久数据归档，验证备份。
2. 核对镜像 revision、架构和版本；预置共享静态资源，保留旧哈希。
3. 依次更新 `maolaoapi`、`maolaoapi-slave-1`、`maolaoapi-slave-2`，
   使用 `--no-deps --no-build`，每节点健康与匿名 HTTP 验收通过后才继续。
4. 验证公共入口、静态资源、数据库新增表和配置兼容；
   PostgreSQL、Redis 与同机其他业务容器不得重建。

`.362` 到 `.366` 新增 `ticket_notification_messages` 表，由主节点执行迁移。
`MaxTokenAutoGroups` 未保存时默认 0（不限制），已有正整数配置保持。
Default 与 Classic 均包含在发行镜像中；生产当前使用 Classic。
不执行真实付费模型请求，不创建生产测试用户，不自动导入或启用技能。

## 回滚边界

旧镜像摘要为 `sha256:bfb4ed9e3da167e0b1f86499b28418858d46c3ce9a05654c191b08e66ca7c264`，
保留旧镜像与原 Compose。失败时停止后续节点并逐个恢复应用镜像，
不整库恢复覆盖在线业务写入。回退旧版前需处理已显式保存的分组上限 0，
并评估新版允许的超出旧版限制的令牌；新增通知关联表可保留。

## 执行结果

备份目录 `/apps/maolaoapi/backups/pre-366-20261106`，0700 私有权限。
包含 Compose、容器配置、数据库 custom dump、schema、TOC 与持久数据归档。
数据库 dump 为 1,644,501,634 字节，SHA256 为
`ac3febbced09798a92fc694802bd9eb9d930a79fd75775b62ec71988f0efb876`；
数据归档为 88,640,019 字节。`pg_restore --list`、归档可读性与校验和检查通过，
未进行数据库恢复演练。备份启动作业 `4ebe00bf-f55c-4425-8dee-1397a33c932f`。

使用独立匿名 Docker 配置成功拉取固定摘要；最初使用主机默认配置时仓库返回 denied，
未改写主机已有登录配置。该主机使用 containerd 存储，`docker inspect` 的镜像 ID
显示清单摘要，与 zzapi 传统存储显示的配置摘要不同；RepoDigest、amd64 架构、
版本与 revision 均匹配已发行镜像。

候选镜像在 `network=none` 的独立 SQLite 容器中验证两套模板，
未接入生产数据库。Default 的 7 个入口资源正常；Classic 递归提取 125 个资源，
共 18,918,065 字节，预置于原共享静态目录且保留全部旧资源。
125 个文件均通过本机 HTTPS SHA256 检查，作业
`bcbb0201-9bde-40d6-b7bb-5192f1920680`。预览容器已停止。
生产继续使用 Classic，未修改主题选项或反向代理。

主节点更新作业 `e14d2988-9c39-4d18-8994-32bf39eccd87` 已通过，
准确版本 `.366`、healthy、restart=0，新增通知关联表已存在。
环境、命令、网络、端口、挂载、资源限制和日志配置保持；
四项匿名保护接口均为 401，8 个入口资源与预置摘要一致。

滚动期间旧版的默认 5 组限制与新版默认 0 暂时并存，未在此窗口创建超限生产令牌；
旧节点处理的工单通知可能缺少新的回复标记关联，不保证事后补标记。
两个从节点随后依次完成相同验收：

- slave-1：`aba93708-a6ab-47ac-b22e-73bcd36700b8`。
- slave-2：`33b45890-0b12-4b89-9eec-4326ef998daa`。

最终复核作业 `6fef08c9-f151-4fde-89fd-1a677aabd5af` 成功：
三个节点均为 `.366`、healthy、restart=0；Compose 仅三个应用镜像发生变化。
基线内其余 13 个容器的 ID 与启动时间均保持，包括 PostgreSQL、Redis、
DouDi 与 zhishiapi。备份文件 SHA256 再次校验通过。
数据库仍未保存 `MaxTokenAutoGroups`，三个节点统一使用默认 0；
没有修改历史令牌、主题或其他选项。

源站 HTTPS 与独立外部请求均确认 `.366`／Classic，8 个入口 JS/CSS
逐项 SHA256 一致；页面包含新上限配置及默认 0 的说明。
四项匿名接口均返回 401，缺失资源返回 404/no-store。
服务器直接访问公网曾收到 403，随后分别使用源站域名 TLS 检查和本地外部访问验证，
没有将该次请求失败当作应用故障，也没有调整代理或防护规则。

最终报告保存在 `/apps/maolaoapi/releases/366/deployment.final.json`，
备份目录中保留副本。未触发回滚。上述为部署与基础 HTTP 冒烟验收，
未执行登录后的真实浏览器交互、真实付费推理或完整用户会话复演。
