# maolaoapi 复制密钥与账号绑定修复 .331 发布

## 固定版本与范围

用户明确授权发版并逐个更新 maolaoapi 应用容器。发布标签为
`v1.0.0-rc.10.1.10.331`，固定到已通过 CI 的主分支提交
`691b1ac81f8864b21189c837901039e2872d9e82`，包含 PR #250 复制密钥独立限流和
PR #252 账号绑定修复。实现和验证见[复制密钥](21_token_key_copy_shared_limit.md)及
[账号绑定](21_account_binding_identity.md)。不把纯邮箱新账号现场描述为已复现。

通过 CloudSSH `API中转站 / 美国netcup rs4000 猫佬API`（serverId=38、hostId=11）
操作 `/home/docker/maolaoapi`，按 `maolaoapi → maolaoapi-slave-1 → maolaoapi-slave-2`
逐个重建，保留角色、端口、挂载、环境和代理配置。数据库、Redis 和其他项目容器不重建。
本版本相对 .330 没有数据库 schema 变更，也不升级外置模块。

## 发布、备份及逐节点门禁

先确认 Linux 发布和多架构镜像构建成功，核对标签提交、发布校验文件、镜像 digest 和 OCI revision。
CloudSSH 预检确认现场版本和应用身份后，备份数据库、Compose、环境文件及模块文件，记录
运行配置摘要、数据库/Redis 身份及启动时间；敏感配置只保留在主机私有备份目录，不输出明文。

镜像使用固定 `tag@sha256`，配置结构化比较只允许三个应用的镜像引用变化。
每次仅对一个应用执行 Compose `up -d --no-deps --force-recreate`，随后验证健康、版本、
运行二进制、端口、配置摘要、未认证接口和启动日志；通过后才进行下一节点。
最终检查公网版本、当前启用模板的入口资源、模块文件摘要和数据库/Redis 未重启。

## 回滚和验证边界

保留旧镜像及备份。任何节点失败即停止后续升级，使用单节点回滚动作恢复该应用的旧镜像，
核对旧版本、健康和运行配置；不自动恢复数据库。`pg_restore --list` 仅验证备份目录可解析，
不能当作完整恢复演练。本次不发真实邮件、不进行真实账号绑定、不调用付费上游模型。

## 执行结果

CloudSSH 预检作业 `052c7d9f-112f-4ed1-a000-6292980ca259` 确认三应用均为 .330、
healthy、restart=0，端口为 18095、18100、18101；数据库约 43.5 GB，磁盘余量约 888 GB。

Linux 发布工作流 `35571992405`、多架构镜像工作流 `35571992401` 均成功。
发布附件下载后与校验文件一致，且包含本次修复的特征：

- amd64：157,655,305 字节，SHA-256 `8beb5581fa40c36671fbf9979808ba990152a70ad0039df6616ca0893c94f9f2`。
- arm64：152,568,073 字节，SHA-256 `d71f20948f8cacb671440f9184c4d3017ff28ed0d9da6a2a599af45fe597c45a`。
- 多架构镜像摘要：`sha256:dc3f5308ea5394a117ae64ed8abbc9242371e5851353912862789b2522f5eddb`。

备份作业 `16791b36-2139-4553-82b8-17eb6aa25244` 成功，目录为
`/home/docker/maolaoapi/backups/pre-331-20260921T071617Z`：

- PostgreSQL：2,143,248,938 字节，SHA-256 `d9b812a415c203605199bc234c52eaf8082d345cbfadcdb528576d0aca1ee6bc`，`pg_restore --list` 通过。
- Compose SHA-256：`db6d530baa3dd3394b1d990030b433270ef606cab4df223b3b4868a8ce8899e0`。
- 29 个模块文件已记录摘要，模块归档 SHA-256 为 `65f15cab90cfa743864873a60c2384163a56a87cdb032c685005c85da67b5584`。

镜像准备作业 `09f211b6-3bf9-4895-9d2f-0f029456b198` 已核实 OCI revision 与发布提交一致，
实际 amd64 image ID 为 `sha256:deec2e5e0ab838213e0fe735d7db2580b5ed417fb261ed7d5004c588c0a887fc`。
结构化 Compose 比较确认只更改三个应用的镜像引用，随后按顺序完成：

| 节点              | 端口  | 新容器 ID 前缀 | 更新作业                             | 结果                     |
| ----------------- | ----- | -------------- | ------------------------------------ | ------------------------ |
| maolaoapi         | 18095 | 498acadda063   | 65a426ec-156e-43a5-a9a4-dcda12a52d59 | .331、healthy、restart=0 |
| maolaoapi-slave-1 | 18100 | aa24450e3eed   | 45b2f30d-b83a-4e3f-8173-578fcb2ee8f0 | .331、healthy、restart=0 |
| maolaoapi-slave-2 | 18101 | c00b3c7e73bd   | bcfc0e0e-997b-4862-bed9-c22f81e46c7b | .331、healthy、restart=0 |

最终验收作业 `66b8f764-5a11-4484-964d-9c73e3e7feb5` 和鉴权检查
`089f749a-3b22-45f4-a0fd-7da943dbc961` 均成功：

- 三节点 `/proc/1/exe` 与镜像内 `/new-api` 相同，SHA-256 均为 `25336a39af8c6b62ff77adf72ae2cd1e3261aee02d7af0cd5a7ca8c88829296d`。
- 各节点环境、端口、挂载及运行参数摘要保持；PostgreSQL/Redis 的容器 ID、启动时间与重启计数保持。
- 29 个模块文件摘要保持；模型检测配置及原有 49 条记录摘要保持，最终共有 51 条记录，业务继续产生记录，本次未插入测试记录。
- 未认证访问用户资料、扩展、模型列表均为 401；未认证 POST 邮箱绑定、单条与批量取密钥也均为 401；Classic 页面为 200。启动以来最近 300 行日志无 panic、FATAL 或 `[ERROR]` 标记。
- 主机访问公网状态仍被 Cloudflare 返回 403；操作端独立验证公网 `/api/status` 为 200、success=true、版本 .331，没有绕过该访问策略。
- 公网 Classic 入口 `/assets/index-B15tpQjD.js` 返回 200，11,796,035 字节，SHA-256 为 `e4474cd861bdeff6ced051ebaae1d64d141a0d5ce9f88fbc16f63a4d4d4b29e6`；确认包含新的绑定意图保护及 Telegram bind/start 调用。

完整现场证据保存在 `/home/docker/maolaoapi/releases/331/`，包括部署脚本、固定镜像引用、
备份位置、三个节点报告、`verification.json`、`auth-boundary-verification.json` 和
`public-verification.json`。旧 .330 镜像和单节点回滚动作保留，无需恢复数据库。
