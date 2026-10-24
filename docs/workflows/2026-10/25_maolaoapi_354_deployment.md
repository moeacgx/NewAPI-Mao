# maolaoapi .354 逐节点生产部署

## 目标与范围

按用户要求将 [.354 发布](25_skill_updates_354_release.md) 逐个更新到正式站三个应用容器。
目标为 CloudSSH serverId 63，根目录 `/apps/maolaoapi`，Compose 项目 `maolaoapi-ovh`，
主节点 `maolaoapi`（18095）、第一从节点 `maolaoapi-slave-1`（18100）、
第二从节点 `maolaoapi-slave-2`（18101）。不操作 zzapi 测试站或同机其他站点。

部署前三节点均为 `.353` 固定镜像
`ghcr.io/moeacgx/newapi-mao@sha256:4bce6ec322e8dde4eb026cabaec20266815a99623ca59757129029d79a3c9836`，
revision `69eef59e5982645201e1e75b8b886995228d4c0c`，healthy、restart=0。

## 方案

沿用 [.352](22_maolaoapi_352_deployment.md) 与 `.353` 的逐节点运维脚本，仅替换版本常量：

1. 只读预检：容器、Compose、依赖身份、各节点状态接口。
1. 备份：Compose、容器 inspect、表结构、关键表身份、PostgreSQL custom dump 与 TOC、
   持久数据归档，生成 SHA-256 清单；私有文件仅留主机，权限 0600/0700。
1. 候选镜像：匿名拉取固定摘要，核对架构、版本与 revision 标签及运行时默认配置不变，
   断网只读运行 `--version`。
1. 静态资源预置：空 SQLite 预览实例提取 Classic 新旧资源，按原哈希并存到
   `/var/www/maolaoapi-static/assets`，经本机 HTTPS 反代逐项校验；不改、不 reload nginx。
1. 经用户二次确认后：主节点更新并验收 → 校验迁移（既有列保留，新增
   `prompt_templates.repository_source_json` 可空 TEXT，关键表身份一致）→ 第一从节点 →
   第二从节点 → 公网与备份复核。

每个节点只替换应用 image，不改环境、端口、挂载、资源限制和启动命令，不重建
PostgreSQL/Redis。节点验收包括 healthy、restart=0、revision、配置与依赖不变、匿名接口
401、静态资源与启动日志。

## 回滚

失败即停止，后续节点保持 `.353`。已更新节点回退到上述 `.353` 固定镜像；新增列兼容
`.353`，不删除列、不恢复在线数据库。回滚后已在 `.354` 删除的技能不会恢复，需要时从
本次备份单独提取。

## 结果

待执行。
