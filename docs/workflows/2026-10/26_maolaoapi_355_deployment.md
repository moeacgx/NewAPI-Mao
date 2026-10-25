# maolaoapi .355 逐节点生产部署

## 目标与现场基线

将 `.355` 正式固定镜像滚动更新到 MaoLaoAPI 生产应用。目标为 CloudSSH 项目“API中转站”中
精确名称“OVH兜底”的主机；部署前重新核对 `serverId`、`hostId`、Compose 项目、目录、当前镜像、
健康状态和依赖身份，不以本记录中的历史数据代替现场检查。

当前 CloudSSH 清单对应“API中转站 / OVH兜底”（`serverId=63`、`hostId=29`）。
预期路径为 `/apps/maolaoapi`，Compose 项目为 `maolaoapi-ovh`；仅更新
`maolaoapi`、`maolaoapi-slave-1`、`maolaoapi-slave-2` 三个应用，不改 nginx、PostgreSQL、Redis、
端口、挂载、环境变量、资源限制或重启策略。

## 更新前门禁与备份

1. 只读检查三应用、Compose、PostgreSQL/Redis、启动日志与外部 `/api/status`；确认三个节点健康且运行同一 `.354` 固定镜像。
2. 在 `/apps/maolaoapi/releases/355/` 准备固定版本运维脚本，校验语法、目标服务白名单和固定 digest；
   备份前后均禁止读取、输出或复制容器环境变量值及凭据。
3. 备份 Compose、三应用与数据库/Redis inspect、表结构、关键表身份、持久数据归档、
   PostgreSQL custom dump、`pg_restore --list` 与 SHA-256 清单，私有文件仅存主机并设最小权限。
4. 只读核对 `all_groups` 列不存在或状态符合预期、工单通知任务与待处理通知数量，并记录
   `.354` 的版本、revision 与镜像摘要。现有生产配置开关维持原值。
5. 匿名拉取候选固定 digest，确认 `linux/amd64`、`.355`、OCI revision 与发布提交一致，
   默认配置与当前生产镜像一致；在断网、只读容器中检查 `--version`。
6. 按 `.354` 流程预置 Classic 新旧哈希资源到静态资源目录；经本机 HTTPS 反代核验哈希，
   不修改或 reload nginx。

## 逐节点顺序与迁移

更新顺序固定为：

1. `maolaoapi`（主节点，执行数据库自动迁移）
2. `maolaoapi-slave-1`
3. `maolaoapi-slave-2`

每个节点仅替换应用镜像。主节点通过健康、`.355`、revision、端口/挂载/环境配置、匿名鉴权、
静态资源和启动日志验收后，再更新下一节点；每次操作前确认先前节点仍健康。任一门禁失败即停止，
不同时重建多个应用，不重建数据库或 Redis。

主节点迁移仅预期新增可空 `prompt_templates.all_groups` 列；既有表列及关键表身份保持。
通知任务迁移为幂等更新，不删除任务记录；部署前若发现旧通知任务，按变更记录确认合并和停用结果。
在全部节点完成前不得创建不限分组模板、删除分类或编辑引用目标分类的模板。

## 回滚

任一节点失败即停止后续节点。使用 `.354` 已验证固定 digest 回滚已更新应用；保留 `all_groups` 列与已迁移通知任务，
不删列、不整库恢复、不修改 Redis/数据库容器。回滚后新 `ticket_user_message` 任务不能由旧版发送，
需按现场决定恢复旧任务配置；不限分组模板在旧版不装载。已删除分类不会因程序回滚恢复。
回滚应用脚本和备份目录均保留至发布验收完成。

## 验收与结果

每节点核验 `.355`、固定镜像 digest、准确 revision、healthy、restart=0、有效配置未变、
PostgreSQL/Redis 容器身份未变，`/api/status`、静态 JS/CSS、匿名保护接口与启动日志正常。
完成三节点后独立核验公网入口及备份哈希。除非另有明确授权，不发起真实收费模型请求；
部署和健康检查不代表 Gateway Skills 的实际上游行为已由生产付费请求验证。

实际作业 ID、更新前后容器标识、备份路径/校验值、迁移结果、公网结果、验收限制和回滚状态，
仅在全部验证完成后补入本节。
