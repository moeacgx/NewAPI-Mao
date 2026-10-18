# maolaoapi .345 分组标签自动匹配部署

## 发布与范围

用户授权提交 PR、合并并更新线上容器。功能 PR [#296](https://github.com/moeacgx/NewAPI-Mao/pull/296)
已合并到 `custom-main`，提交 `b64ab5e7813596e288ba71d65340506493a227c3`。
版本为 `v1.0.0-rc.10.1.10.345`；Classic 令牌隐藏未分类和空标签、后台紧凑筛选，
标签支持多个显示名称关键词，保存及后续新增/改名分组自动追加匹配绑定。
Default 页面不在本次范围内；分组 ID/code、路由、鉴权、计费和令牌提交契约保持。

通过 CloudSSH「API中转站」项目 `serverId=38`、`hostId=11` 操作
`/home/docker/maolaoapi`。更新顺序为 `maolaoapi:18095`、
`maolaoapi-slave-1:18100`、`maolaoapi-slave-2:18101`，每节点验证通过后才继续。

## 验证与镜像

- 本地 `go test ./model ./controller -timeout 60s -count=1` 和
  `go test ./router -timeout 60s -count=1` 通过；独立连接并发回归通过。
- Classic 管理/令牌兼容测试共 16 项、涉及文件 lint/格式化、i18n 同步和构建通过。
- 首次 CI 暴露路由 fixture 缺少 `options` 表，补齐后
  [CI 36842711065](https://github.com/moeacgx/NewAPI-Mao/actions/runs/36842711065) 前后端均成功。
- [Linux Release](https://github.com/moeacgx/NewAPI-Mao/actions/runs/36843242421) 与
  [多架构 Docker 构建](https://github.com/moeacgx/NewAPI-Mao/actions/runs/36843242365) 均成功。
- 固定镜像为
  `ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.345@sha256:cd951f910dff1c701b7036953879063f92decd7ae13fee612a13612295d105e1`，
  本机 amd64 镜像 ID 为 `sha256:82d6f8f33e2f46af2734af5632118848090f4dffa6bd1101ffe48bd91b139d3e`。
  OCI revision 与上述合并提交一致。

## 备份与回退

备份目录为 `/home/docker/maolaoapi/backups/pre-345-20261001T091303Z`，
包含 Compose、五个容器 inspect、应用数据归档和 PostgreSQL 自定义格式 dump。
数据库归档 1,805,190,496 字节，SHA-256：
`3fae5da607544a6499806b02ddd7cad5ff558878005d10b991cc3f957920c301`。
`pg_restore --list` 校验通过，未做完整恢复演练。

备份时用户 9855、令牌 15056、分组 25、渠道 197、配置项 198、标签 7、标签绑定 17。
保留旧 `.344` 固定镜像及 Compose；应用回退可逐节点恢复 `.344` 镜像，
保留新增 `match_keywords` TEXT 列及内部锁行。旧应用不会自动匹配；混合版本期间不具备
所有节点共同参与的事务锁保证。数据库恢复会覆盖后续业务数据，不属于普通应用回退。

远端部署脚本、状态和验证报告位于 `/home/docker/maolaoapi/releases/345/`。

## 逐节点执行记录

- 镜像准备作业 `54ae36d1-7de2-4ed2-b231-28dd82359510` 成功；仅替换 Compose 中三个应用镜像，
  `docker compose config --quiet` 通过，保留其既有 `version is obsolete` 警告。
- 主节点作业 `31ee5272-e87e-492f-bd1d-0b120b7ab7bf` 成功。
- 从节点 1 首次作业 `02c19a3f-28f1-43b0-92f6-9ce8f3f49f9b` 已完成更新，
  但验证脚本直接比较挂载数组顺序导致断言失败。只读诊断确认两个挂载的路径和权限完全相同，
  仅 Docker 返回次序不同；改为排序后比较，未再次重建容器。
  复验作业 `a31c97a8-ec2a-4aad-a39b-5668b39502ab` 成功后才继续下一节点。
- 从节点 2 作业 `44dbdb48-315d-4faf-898a-342c2d4b305b` 成功。

## 最终验证

最终作业 `972d15ed-a15f-4294-8550-67b9aa4fbef1` 成功、退出码 0。
三个应用均为固定 `.345` 镜像、healthy、restart=0，实际 `/proc/1/exe --version`
与 `/api/status` 一致。各节点首页引用的 8 个 JS/CSS 资源返回 200；匿名访问
`/api/user/self`、`/api/group/tags`、`/api/token/` 均返回 401。
环境变量、启动参数、端口和挂载与备份保持一致；最近 300 行日志无 panic、fatal 或迁移失败。

PostgreSQL 与 Redis 容器 ID 均保持，未重建。新增 `group_tags.match_keywords` 为 TEXT。
上述七项数据库计数与备份完全相同，7 个既有标签及 17 条绑定的内容指纹保持一致。
公网 `https://maolaoapi.com/api/status` 独立请求返回 `success=true` 和 `.345`。

生产核验覆盖 PostgreSQL 字段迁移和既有数据保持，未执行 MySQL 实跑，
未用管理员身份写入演示数据或发起付费模型请求，未声称生产并发压力验证。
管理员需编辑标签保存匹配词后启用规则；旧标签保持原绑定，不凭名称猜测并自动填充规则。

## 分享评估

- NewAPIForDouDi：大量分组场景可复用多关键词自动归类，需先评估其标签模型和前端模板。
- NewAPIModifyByGang：可参考后台批量绑定及紧凑筛选，需适配 UI 组件和数据契约。
  两项目均未在本轮发送通知；本轮没有跨项目鉴权、计费或路由接口变化。
