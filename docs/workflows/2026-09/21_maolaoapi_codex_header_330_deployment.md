# maolaoapi Codex 响应头校验 .330 发布与逐节点更新

## 范围与固定版本

用户明确授权发版后逐个更新 maolaoapi。发布标签为 `v1.0.0-rc.10.1.10.330`，
固定到 PR #247 合并提交 `6d2f7bffdb5c742083fbce22ed512752e69d2c2c`。
实现与验证见[主程序 Codex 响应头证据](21_model_guard_codex_header_evidence.md)。
标签同时包含此前已合并的模块 0.2.1 源码快照，外置模块文件不会因本次宿主更新而替换。

使用 CloudSSH 的 `API中转站 / 美国netcup rs4000 猫佬API` 入口，serverId=38、hostId=11。
目标为 `/home/docker/maolaoapi/docker-compose.yml` 的三个应用，顺序为
`maolaoapi → maolaoapi-slave-1 → maolaoapi-slave-2`，逐台健康及版本校验后继续。
不更新 zzapi、zhishiapi、数据库或 Redis 容器，不修改模块、检测规则和白名单。

## 现场预检

CloudSSH 作业 `2a895e6c-c848-4666-8278-5e5be984b092` 核实：

- 三应用为 .329、healthy、restart=0，端口分别为 18095、18100、18101。
- 固定旧镜像摘要为 `sha256:3c42b83f1b0ec7e31056cff4b034123e6c875379557a54c759885deac4aecc5d`，
  旧 image ID 为 `sha256:2385226a9cd73f2b7311199e8bbb628188be17788a378cd719a5ce99e8cc861e`。
- 主应用未设置 NODE_TYPE，两个从应用为 slave，保留现场角色、端口、挂载和环境。
- 上游模型校验模块已安装 0.2.1，检测开启，预检有 41 条历史记录；新增 detection_source 列尚不存在。
- PostgreSQL 约 43 GB，磁盘可用约 890 GB，数据库与 Redis 正常运行。

短作业连接随后返回 ECONNRESET，未开始重建；切换 CloudSSH platform 持久会话
`9419083e-14b9-4564-ac9f-daabebb82516`，核实没有开始的备份后再执行。
未使用直连 SSH 或读取 SSH 凭据。

## 备份与迁移边界

备份 PostgreSQL 自定义格式数据、Compose、环境文件、模块归档及文件摘要，
记录容器身份、运行配置摘要、检测配置摘要和历史记录摘要。备份生成期间服务保持运行。
`pg_restore --list` 仅验证备份目录，不代表完整恢复演练；本次不创建 Redis 快照。

首个主节点启动执行可空 TEXT 列 detection_source 的迁移，后续节点复用新 schema。
逐节点检查字段类型/可空/无默认值、历史记录和检测配置保留。部署期间新旧节点检测能力不同，
新增响应头判定将从已升级节点开始按原规则生效。

## 回滚与验收

保留 .329 镜像与备份。任一节点失败停止后续更新，使用现场 deploy.py 的 rollback 动作
只回滚对应应用镜像并核对健康、配置和旧版本，不自动恢复数据库。
新增可空列可保留，旧程序仍执行正文模型检测。

最终核对全部节点的固定镜像、版本、健康、重启次数、未认证接口边界和启动错误，
检查公网状态及静态资源、模块文件摘要、数据库/Redis 身份与启动时间。
不发起真实上游付费请求，不把静态资源检查描述为登录态交互或真实模型替换验收。

## 执行结果

Linux 发布工作流 `35560127408` 和多架构镜像工作流 `35560127407` 均成功。
两份二进制已下载并按发布校验文件核验：

- amd64：157,647,113 字节，SHA-256 `406d5b538ade9e1835733f4788dbe46a7d47b7e6ef4ee0d00cc4c8f0d8935d7e`。
- arm64：152,568,073 字节，SHA-256 `e6c235fa8cbe7fb9cf86e92daab58506499848fabfec178ecc29ff7c71ddc06e`。
- 目标镜像摘要：`sha256:150c5a18d6604313b27684a2e3b75d4e7216b22d52e9bd2ca49c0292089674ce`。
- 实际 amd64 image ID：`sha256:a11c5a98e757da56340cf3735f569a94a2a0566fe426dd3875e75a9060ab1861`，revision 与发布提交一致。

备份目录为 `/home/docker/maolaoapi/backups/pre-330-20260921T041932Z`：

- PostgreSQL 备份 2,095,269,596 字节，SHA-256 `5770ac12c8223fbbc8273ba63f5d95e48053d6ad8e7dd1e38462aaf65cd2787d`，`pg_restore --list` 通过。
- Compose SHA-256 `11d120bb5b5e378a475366f389475f44503d5bf155413acc65c37ad6b413d68e`。
- 29 个模块文件已记录摘要，模块归档 125,319 字节，SHA-256 `65f15cab90cfa743864873a60c2384163a56a87cdb032c685005c85da67b5584`。

配置准备已完成，结构化比较确认仅三个应用镜像引用发生变化，使用 `tag@sha256` 固定镜像。

三个节点已按指定顺序更新，每节点验证完成后再执行下一台：

| 节点              | 端口  | 新容器 ID 前缀 | 验收                     |
| ----------------- | ----- | -------------- | ------------------------ |
| maolaoapi         | 18095 | e72a5456eba7   | .330、healthy、restart=0 |
| maolaoapi-slave-1 | 18100 | 7d3894d6a1fa   | .330、healthy、restart=0 |
| maolaoapi-slave-2 | 18101 | e3395b74c9f2   | .330、healthy、restart=0 |

每台均核对运行配置摘要保持，`/api/user/self`、`/api/extensions/`、`/v1/models` 未认证返回 401，
Classic 页面返回 200。启动以来最后 300 行日志未发现 panic、FATAL 或 `[ERROR]` 标记。

最终验收确认：

- 三台实际镜像均为上述固定摘要及 image ID，`/proc/1/exe` 与容器 `/new-api` 字节相同；
  三节点运行二进制 SHA-256 均为 `a7980ba5db2137d6bf74f007a268f4a82e4595631ea547576df08da7f1f741b3`。
- PostgreSQL 新列为 TEXT、允许 NULL、无默认值，41 条原记录摘要保持；最终记录总数为 42，
  运行期间业务继续处理，本次没有写入测试检测记录。检测配置摘要保持。
- 29 个模块文件集合和摘要保持，包含外置上游模型校验 0.2.1；没有替换模块包。
- PostgreSQL、Redis 的容器 ID、启动时间和重启计数保持。
- 服务器访问公网状态页仍被 Cloudflare 返回 403，未绕过该策略；操作端独立请求公网
  `/api/status` 返回 200、success=true、版本 .330。
- 公网 Classic 入口 `/assets/index-DeCaGX23.js` 返回 200、11,791,967 字节，
  SHA-256 `4137b4142f8fff3ef9ef74c2cb024c7d52f3e1925681b10250199c9b16b18c8e`。
  该前端资源与 .329 相同，符合本次主程序检测变更范围。

完整现场证据在 `/home/docker/maolaoapi/releases/330/` 的 `verification.json`、
`runtime-verification.json`、`public-verification.json` 及三个节点各自的 JSON。
保留备份、旧镜像和单节点 rollback 动作。没有更新 zzapi/zhishiapi，没有执行真实付费模型验证，
不将上线健康检查描述为已观察到真实 Codex 替换事件。
