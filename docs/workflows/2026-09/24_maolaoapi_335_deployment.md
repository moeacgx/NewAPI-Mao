# maolaoapi .335 逐节点更新

## 授权与固定目标

用户要求逐个更新 maolaoapi 容器。本次仅操作 CloudSSH「API中转站」项目
serverId=38、hostId=11，目录 `/home/docker/maolaoapi`，不操作 zzapi。
现场三个应用均为 `.333`、healthy、restart=0；端口为 18095/18100/18101。
目标为已发布并在 zzapi 验证的 `v1.0.0-rc.10.1.10.335`。

固定镜像 `ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.335`，摘要
`sha256:3ebcd65ca90e33ede0da7a5892c3edda90c1261d0468572a4c6568ef43df6753`，
OCI revision `e94fb60a4edccb729a7a1dc6e254526a5972cbc7`。
Linux 与多架构镜像发布工作流均成功，升级不包含未发布的本地渠道测试提示修复。

Cloudflare Jev 0.2.1 是独立插件版本，本次不自动安装、激活、修改渠道或价格。
容器升级不等于插件升级，也不执行真实付费推理验收。

## 执行与回滚方案

1. 备份 PostgreSQL、Compose、环境文件与外置模块，记录容器、运行参数及模型校验配置摘要。
2. 校验备份可读及固定镜像 revision/架构/摘要，保留 `.333` 旧镜像。
3. 仅替换三个应用的 Compose 镜像引用，结构化检查其他配置不变。
4. 严格按 `maolaoapi → maolaoapi-slave-1 → maolaoapi-slave-2` 重建，
   每个节点健康、版本、运行配置及匿名权限边界通过后才开始下一个。
5. 核验实际进程二进制、静态资源、启动日志、公网状态；数据库和 Redis 不重建、不重启。

某节点失败则停止后续更新，恢复该节点 `.333` 固定镜像与原运行配置，
不自动恢复在线数据库。本次已检查 `.333 → .335` 模型差异，新增任务结果丢弃标记存于既有 JSON，
没有发现这批变更新增数据库列的要求；正常启动仍会执行宿主既有迁移逻辑。

## 验证记录

预检 CloudSSH 作业 `bd7c1afa-5c89-441f-9b80-e7659d16e739` 成功，三节点 healthy，
旧镜像摘要 `sha256:e61a8d79627230baf2e4791ae0dc28a341830f2d3aa201c0c750e3f3f2e7d990`。

备份作业 `cbab0a09-17c0-4f20-be1c-aadd6dc826d0` 成功、exitCode=0。
私有目录 `/home/docker/maolaoapi/backups/pre-335-20260923T195718Z`（路径使用 UTC）：

- PostgreSQL 自定义归档 2,180,274,495 字节，SHA-256
  `a4bb2c8ac5be7b72ffa813f23eeff53fcfb3f9fa5b4a610480aa92d8052b9866`；`pg_restore --list` 通过，未做完整恢复演练。
- Compose SHA-256 `ebf543cd7b1e4984403a4773e290f41e471cf965ed19e5ef4b8eb8829815aa7c`。
- 29 个模块文件已备份，归档 SHA-256 `65f15cab90cfa743864873a60c2384163a56a87cdb032c685005c85da67b5584`。
- 环境文件和容器运行参数摘要已保留，未输出凭据。

镜像准备作业 `5f83d290-91a6-49eb-872c-bd6023da063d` 成功。
OCI revision、amd64 架构、固定摘要一致；image ID 为
`sha256:cc92e6a81da7947ea6003b1f50a01bcac33b4e48c79dde3935df2cca85ed5eac`。
结构化比较确认 Compose 仅三个应用镜像引用发生变化。

## 逐节点执行

- 主节点作业 `49c721c0-6637-4fb0-9059-a3955d9d5fed` 成功，容器 `413c588b784e`，端口 18095。
- 从节点 1 作业 `4ffc222c-3549-463d-ab49-02ebc6da023a` 成功，容器 `38d7f9cbd197`，端口 18100。
- 从节点 2 作业 `2ade34eb-ed05-4f9e-bd91-af08e8327cc9` 成功，容器 `4b536915fad8`，端口 18101。

已完成节点均为 `.335`、healthy、restart=0；环境、端口、挂载、启动参数与备份摘要一致。
`/proc/1/exe` 与 `/new-api` 均与目标镜像二进制一致，SHA-256 为
`4352e8f69bf0776517e0c4b06498972727f133ec128adf6bfad1d272f2d45f51`。
未认证 user/extensions/plugin/models 接口为 401，Classic 页面为 200。
模型校验配置、字段结构及 245 条原有记录保持。

## 最终验收

最终作业 `0b268c76-12a6-4ca7-8144-edf1bd5b0f60` 成功、exitCode=0，三个节点验证全部通过。

- 三节点实际进程版本与镜像均为 `.335`，健康且重启计数为 0，启动日志未出现 panic 或 fatal。
- 环境、端口、挂载、启动参数保持；PostgreSQL/Redis 容器 ID、启动时间及重启计数保持。
- 29 个模块文件 hash 不变，模型校验配置与 245 条原有记录保持。
- 远端主机访问公网仍被 Cloudflare 返回 403；操作端独立请求确认公网 `/api/status`
  HTTP 200、success=true、版本 `.335`，Classic 页面 HTTP 200。
- 公网页面引用 `/assets/index-BpjaYuOg.js`，实际资源 HTTP 200、11,823,659 字节，SHA-256
  `3fc78dfc6fc7851fc81ec6f865e8969f44dba97786adb624f3512408fa5ece4d`。

没有执行回滚，没有改动 zzapi，没有安装/激活插件或发起付费模型调用。
部署脚本与逐节点报告保存在 `/home/docker/maolaoapi/releases/335/`，可按
`python3 /home/docker/maolaoapi/releases/335/deploy.py rollback <应用名>` 单节点回退到 `.333`；
回退不恢复数据库。备份与旧镜像均保留。

本次验收覆盖应用进程、权限门禁、配置和静态资源，不代表登录态页面流程或真实供应商推理已验收。
共享 API 与插件合同变化来源于已发布 `.335`，本次部署不新增跨项目接口变更。
