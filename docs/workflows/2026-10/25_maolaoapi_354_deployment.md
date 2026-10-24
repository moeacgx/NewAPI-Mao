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

### 发布

- #334 合并提交 `01d689e968369865a69651a06e9693eb746b2b91`，附注标签
  `v1.0.0-rc.10.1.10.354`。Release（Linux）与多架构镜像工作流均成功。
- Release 非草稿、非预发布，且为仓库 Latest；amd64 与 arm64 二进制均匹配
  `checksums-linux.txt`。`go version -m` 确认 amd64 二进制为 linux/amd64，版本注入 `.354`，
  `vcs.revision` 为合并提交。
- GHCR 标签 `v1.0.0-rc.10.1.10.354` 与 `latest` 均指向索引
  `sha256:098296645ebe5b24f03e7ffd38b36c27fc0c222ac82fe23d3898195e53e176fb`；amd64 镜像
  标签 version/revision 与发布一致。

### 正式站

- 只读预检：三应用为 `.353`、healthy、restart=0，PostgreSQL/Redis 运行，`releases/354`
  不存在。
- 备份 `/apps/maolaoapi/backups/pre-354-20261024T220805Z`：数据库 custom dump
  1,554,265,776 字节，TOC 可读，SHA256
  `e73bb7620232779efc2c5cd97760a65cec95c32563972f2fc4b90a3d57fea3a6`；持久数据归档
  88,640,019 字节，SHA256 `ac72e7a85f6ae9f431eef89e7efbbad291ec224b109b94ed77a33c020c4ce2ec`；
  表结构 1501 列。
- 候选镜像：匿名拉取固定索引摘要，amd64、`.354`、revision 一致，运行时默认配置与 `.353`
  相同，断网只读 `--version` 输出 `.354`。
- 静态资源：Classic 新旧资源 176 个、33,331,122 字节按原哈希并存，经本机 HTTPS 反代
  逐项哈希校验通过；未知资源 404 / `no-store`；预览实例已停止，nginx 未修改。
  与 `.352`、`.353` 相同，四个历史 `_base*` 虚拟引用（新旧版本各两个）返回 404，
  不计为真实资源成功。
- 主节点：新容器 `c1303d7efdb8`，`.354`、healthy、restart=0，配置与依赖不变，12 个匿名
  保护接口均 401，样式版本参数 `.354`，启动日志无严重错误。
- 迁移：原有 1501 列全部保留且类型不变，仅新增 `prompt_templates.repository_source_json`
  （text、可空、无默认值）；users/tokens/channels/groups/prompt_templates 按备份前最大 ID
  限定的行数、最大 ID、ID 总和一致；从节点当时仍为 `.353`。
- 第一从节点：新容器 `f172a8bf8488`；第二从节点：新容器 `abbd7a48533e`。均 `.354`、
  healthy、restart=0，部署前复核前序节点，配置、依赖、鉴权与启动日志通过。
- 最终验收（UTC 22:32）：三节点同一固定镜像，healthy、restart=0；公网
  `https://maolaoapi.com/api/status` 返回 `.354`、Classic，JS/CSS 可读，缺失资源 404 /
  `no-store`；备份文件大小、SHA-256 与 0600 权限复核通过。结果保存在备份目录的
  `deployment.verified.json`，未执行回滚。

### 未覆盖

- 未使用管理员会话实际执行 GitHub 导入、一键更新或删除，未发送模型请求。
- #333（Default 编辑弹窗）未先部署 zzapi；正式站当前主题为 Classic，该改动不影响正式站
  默认页面。
- 本轮只发布与部署，不改变接口、鉴权、模型或计费契约，不产生跨项目适配要求。
