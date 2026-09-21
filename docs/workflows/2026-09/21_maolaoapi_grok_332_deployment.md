# maolaoapi Grok Messages 兼容 .332 发布

## 固定版本与范围

用户授权发布并逐个更新 maolaoapi 应用。版本 `v1.0.0-rc.10.1.10.332` 固定到
`59484e8a3f5d1bbe2f5e7e2ee5ef97d558ee6b70`，包含 PR #256 的 xAI Messages 兼容及
PR #254 的项目身份和自更新兼容。新镜像位于 `ghcr.io/moeacgx/newapi-mao`，
容器、卷、数据库和运行标识保持原值。相对 .331 无数据库 schema 变更。

## 执行方案

CloudSSH serverId=38、hostId=11，目录 `/home/docker/maolaoapi`。
备份数据库、Compose、环境及模块文件，保留旧镜像；备份内容仅存主机私有目录。
镜像固定 tag/digest 并核对 OCI revision，结构化比较 Compose 只允许三个应用镜像变化。
按 `maolaoapi → maolaoapi-slave-1 → maolaoapi-slave-2` 逐个重建，
每次验证健康、版本、二进制摘要、端口/环境/挂载、未认证接口及启动日志。
最终检查公网状态和静态资源，数据库与 Redis 的身份和启动时间须保持。
单节点失败即停止后续更新，恢复该节点旧镜像；不自动恢复数据库。

## 发布及预检

PR #256 后端首次 CI 失败于既有 Realtime 测试的清理时序；本地定向通过后，
同一提交重跑 CI `35589322338` 全部通过，再合并。CodeRabbit 状态表示跳过审查，
不作为实质审查通过的证据。

Linux 发布 `35592209411`、多架构镜像 `35592210266` 均成功。
镜像 digest：`sha256:639af94bae1899bdb5b7bf5f4a2b4eac8e24821091e796ea2ec69c8e2527d364`。
预检作业 `71ccc751-5ea8-4b51-9281-e5acaa94c86b` 确认三节点 .331、healthy、restart=0，
端口 18095/18100/18101，数据库约 43.6 GB，磁盘余量约 885 GB。

此前首次备份脚本仍指向 `releases/331`，被已有备份断言拒绝，未修改应用或创建新备份。
已修正并固定 `releases/332`，继续执行本次发布。

## 验证结果

发布附件下载后与 `checksums-linux.txt` 一致：

- amd64：157,716,745 字节，SHA-256 `cea6695bd8c1ac77efaed0ea51c1bf806942a9eb6244a06fc11efdbb6f19f014`。
- arm64：152,633,609 字节，SHA-256 `6f0b4370fcb46c4af7b74a1e76fe52e595d69d4a3d636a4be0824ffe30c2ba9b`。
- 新镜像匿名 manifest HEAD 返回 200，digest 与发布工作流一致。

备份作业 `fc1242e7-d7a9-4676-b327-afcbb5d7c21e` 成功，私有备份目录
`/home/docker/maolaoapi/backups/pre-332-20260921T122530Z`：

- PostgreSQL：2,114,394,920 字节，SHA-256 `ca53da9308d322c565d6649bb9bb10dfc7a38ca59eac89efafb1ed8b9a4b153e`；`pg_restore --list` 通过，不等同完整恢复演练。
- Compose SHA-256 `d13d89112817d431e2226dc20c3ba7b2c9450ffd7213461d8f29761a4feb1978`。
- 29 个模块文件已备份，归档 SHA-256 `65f15cab90cfa743864873a60c2384163a56a87cdb032c685005c85da67b5584`。

镜像准备作业 `8dd59675-8f72-4b47-ac7b-33a1f37e885a` 成功，OCI revision 与发布提交一致；
amd64 image ID 为 `sha256:21961ce2b96394a25fbf5b7e602ad1f28ef149a862ebfeb4580d25e6aa937a96`。
Compose 结构化比较只改变三个应用的镜像引用，未改其他服务。

逐节点升级记录：

| 节点              | 端口  | 新容器 ID 前缀 | 作业                                 | 状态                     |
| ----------------- | ----- | -------------- | ------------------------------------ | ------------------------ |
| maolaoapi         | 18095 | 3d15046e5600   | 160e6660-b00b-4385-a48b-1d7deda77886 | .332、healthy、restart=0 |
| maolaoapi-slave-1 | 18100 | df2766b2d21e   | 419a3944-e501-4f79-a197-f092ad71c383 | .332、healthy、restart=0 |
| maolaoapi-slave-2 | 18101 | 9b1510ebe8ab   | 5d6804b6-4bf4-4936-8f0c-9a297e0ec5ca | .332、healthy、restart=0 |

最终验收作业 `3baf3be7-bc9b-410f-ae3c-2f6e377ec402` 成功：

- 三节点 `/proc/1/exe` 与 `/new-api` 摘要一致，均为 `29be681bcfaa384f4b86dc1a84ee6d81da40aa03995cced276183e0f56e1a128`。
- 环境、端口、挂载和运行参数摘要保持；数据库与 Redis 容器 ID、启动时间及重启计数保持。
- 29 个模块文件摘要保持，模型校验配置和 55 条原有记录完整保留。
- 未认证用户资料、扩展、模型列表接口均返回 401，Classic 页面为 200；启动以来最近 300 行日志未发现 panic、FATAL 或 `[ERROR]`。
- 主机访问公网状态仍返回 Cloudflare 403；操作端独立验证 `/api/status` 为 200、success=true、版本 .332。
- 公网 Classic 资源 `/assets/index-Ba4qM9gi.js` 返回 200，11,812,670 字节，SHA-256 为 `68221cfe63ee7d42ef85e3faefa5d5d8c46ba00dd81a81f565e3eb2559d0f2d2`，保留此前账号绑定修复。

现场脚本、镜像引用、备份路径、逐节点报告及最终报告位于 `/home/docker/maolaoapi/releases/332/`。
旧 .331 镜像及 `deploy.py rollback <应用名>` 单节点回滚动作保留；本次未触发回滚。
尚未以真实供应商请求验证 Grok 返回，本次部署验收未使用付费模型调用。
