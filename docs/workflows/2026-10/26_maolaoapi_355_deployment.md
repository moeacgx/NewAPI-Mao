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
4. 只读核对 `all_groups` 列状态、工单通知任务与待处理通知数量，并记录
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

### 发布与备份

- `.355` Release 与镜像发布记录见[发布说明](26_gateway_skills_355_release.md)。候选固定镜像为
  `ghcr.io/moeacgx/newapi-mao@sha256:62fdd34becf5d62ff2444b21bbaa797081be4bc891b78271248cef9d174f5e12`，
  amd64、版本、OCI revision 和默认运行配置均符合预期。
- 部署前备份目录：`/apps/maolaoapi/backups/pre-355-20261025T205533Z`。部署脚本最终复核所有备份
  文件字节数、SHA-256 与权限。PostgreSQL custom dump 为 1,558,756,790 字节，
  `pg_restore --list` 成功（TOC 68,198 字节）；数据归档 88,640,019 字节。
- 数据库 dump SHA-256：`6622145015e303de059857b0b87fdcdb48a7ebbd82272205f7deac6f08d6a580`；
  TOC SHA-256：`f0744eb744caa12b99e4e308bfa1a33668cfdcf04cbfad0401947ba4b88cd64a`；
  data 归档 SHA-256：`ac72e7a85f6ae9f431eef89e7efbbad291ec224b109b94ed77a33c020c4ce2ec`。
  Compose、容器 inspect、schema 与关键表状态也纳入清单；脱敏报告留在主机备份目录。
- 主节点应用迁移新增可空 `prompt_templates.all_groups` 列，SQL 读取验证该列可查询；存量数据按白名单语义处理。
  工单通知任务现场共有 3 个，均为发票、渠道和扩展事件，没有工单通知任务，因此本次通知迁移未更改任务数据。

### 逐节点更新

- 更新脚本候选预检作业 `159a5645-d90f-468c-980a-b01065867fed`、静态资源预置作业
  `c5efd26b-096f-4332-b146-2ead4c54da5f` 均成功；匿名 pull 使用独立 Docker 配置，原 Docker 凭据配置未改。
- 主节点更新与验收作业 `f2e4a398-df71-4907-888d-7d14ae276e44` 成功；第一从节点
  `9b579d03-3d02-4162-b4e3-5f4c62837f18` 成功；第二从节点
  `8b92c926-89ab-41a8-98dc-e4825a5e2dfc` 成功。最终集群验收作业
  `fa5af771-2958-4af3-98fc-831fed9838a0` 成功、exit code 0。
- 三节点均为 `.355` 固定 digest 与 revision，healthy、restart=0；有效环境变量、端口、挂载和资源配置不变。
  PostgreSQL 与 Redis 容器身份及运行状态保持，启动日志未命中 panic、fatal、迁移失败或数据库连接失败标记。
- 每节点 12 个匿名鉴权接口均返回 401；Classic JS/CSS 返回 200，缺失资源为 404 且 `Cache-Control: no-store`。
  静态资源预置共 176 个文件、33,363,713 字节，内容哈希逐项校验；四个历史 `_base*` 虚拟引用仍为 404，
  与 `.354` 一致，不计作真实资源失败。
- 外部 `https://maolaoapi.com/api/status` 返回 `.355` / Classic；公网静态资源路径及缓存策略通过最终脚本复核。
  最终部署报告为 `/apps/maolaoapi/backups/pre-355-20261025T205533Z/deployment.verified.json`。
- 本次不调用收费供应商、不登录管理界面、不验证开关开启后的真实 Responses/Skills 请求；生产设置开关沿用原值。
  当前报告证明 Release、滚动更新、基础鉴权和静态资源完成，不替代真实用户网关 Skills 业务请求验收。
- 未执行回滚。若后续需要回退应用，`.354` 固定镜像为
  `ghcr.io/moeacgx/newapi-mao@sha256:098296645ebe5b24f03e7ffd38b36c27fc0c222ac82fe23d3898195e53e176fb`；
  保留新增列和数据库数据，不做整库恢复。回滚后“不限分组”模板按旧白名单语义不可用，已删除分类不会恢复。
