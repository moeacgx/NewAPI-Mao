# maolaoapi .344 分组标签部署

## 范围与版本

用户要求将 MAO-6 分组标签更新到 maolaoapi。本次仅通过 CloudSSH「API中转站」
项目的 `serverId=38`（`hostId=11`）操作 `/home/docker/maolaoapi` 中的三个应用服务，
顺序为 `maolaoapi:18095`、`maolaoapi-slave-1:18100`、`maolaoapi-slave-2:18101`。
PostgreSQL、Redis 和同机其他服务不在重建范围内。

目标版本为 `v1.0.0-rc.10.1.10.344`，固定镜像为
`ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.344@sha256:df04ffd3a8757488f424e81c24952db873dff4f8b7248768c0ee0e3c3439c033`。
OCI revision 为 `0910e2549c05f17a71f293d16c088fa43afefd50`，本机镜像 ID 为
`sha256:74cc65560778353cc3916b0c51a683fa39e045c9f7d773727a92e64d2abce0f0`。
更新前三个应用均运行 `.343`，状态 healthy，重启计数为 0。

## 备份与执行

更新前备份位于 `/home/docker/maolaoapi/backups/pre-344-20260930T181000Z`，
包含 Compose、五个容器的 inspect 信息、应用数据归档和 PostgreSQL 自定义归档。
数据库归档为 2,219,872,361 字节，SHA-256 为
`da3ec7dbc75037c38565b7a28370dd648dfe67dbe9de4c411d6e4fdae61796bd`；
文件稳定且 `pg_restore --list` 成功，未执行完整恢复演练。

仅修改 Compose 中三个应用的镜像引用，`docker compose config --quiet` 通过。
按既定顺序逐个运行 `docker compose up -d --no-deps <服务名>`，前一节点通过
健康、实际进程版本、镜像 ID、接口权限、运行配置及日志检查后才更新下一节点。
CloudSSH 三个作业均为 `SUCCEEDED`、`exitCode=0`：

- 主节点：更新为 `.344`，端口 18095，healthy，restart=0。
- 从节点 1：更新为 `.344`，端口 18100，healthy，restart=0。
- 从节点 2：更新为 `.344`，端口 18101，healthy，restart=0。

三个节点的环境变量、启动命令、入口点、工作目录、端口绑定和挂载均与备份一致；
`/proc/1/exe --version` 与 `/api/status` 均返回 `.344`。匿名访问
`/api/user/self`、`/api/group/tags` 和 `/api/token/` 均返回 401；最近 300 行日志
未发现 panic、fatal 或迁移失败。

## 部署后核验

三个应用均运行固定镜像且健康、restart=0。PostgreSQL 容器 ID
`bee76d26e45d` 和 Redis 容器 ID `4b5b4ae42b6f` 与备份一致，未重建。
数据库计数与备份基线一致：用户 9844、令牌 15042、分组 25、渠道 197、
配置项 197。新表 `group_tags` 和 `group_tag_bindings` 均已创建，当前均为 0 行；
没有向生产环境填充演示标签。

三个端口的首页 HTML、所引用的 JS 和 CSS 资源均返回 200，三个
`/api/status` 均报告 `.344`。操作端独立访问公网 `https://maolaoapi.com/api/status`，
确认 `success=true`、版本 `.344`。检查覆盖了应用启动、静态资源、匿名权限和数据库迁移；
未使用管理员登录态操作标签页面，也未发起付费模型请求。

## 回退边界

保留原 `.343` 镜像、更新前 Compose 和数据库备份。若需要应用回退，
应将三个应用的镜像引用恢复到备份中的固定 `.343` 摘要，再按单节点健康门禁
逐个重建；新建标签表可保留，旧版本不使用它们。数据库恢复会覆盖更新后的业务数据，
不属于普通应用回退步骤。未执行回退。

分组标签仅影响 Classic 的管理和令牌选组界面，稳定分组 ID/code、鉴权、路由和
计费契约未改变；Default 页面不在 MAO-6 范围内。具体接口及数据契约见
[分组标签开发文档](../../developer/group-tags.md)。
