# zzapi 扩展在线安装 `.327` 发布

## 目标与范围

用户授权发版并更新 zzapi。版本 `v1.0.0-rc.10.1.10.327` 汇总已合并的
PR #232 至 #236：外置上游模型校验、渠道白名单与连续容错、扩展公开仓库、
两个配套仓库的根目录 Git 子模块，以及 Default、Classic 在线模块弹窗。
本次版本提交只调整版本和部署文档，业务实现保持上述已验证提交。

目标为 CloudSSH `API中转站 / RS2000 德国建站`，serverId=52、hostId=17，
工作目录 `/home/docker/zzapi`。只更新 `zzapi`、`zzapi-slave-1`、`zzapi-slave-2`，
端口依次为 18097、18098、18099。maolaoapi、zhishiapi 不在本次范围内。

## 发布与升级方案

版本 PR 经当前基线检查和 CI 后合入 `custom-main`，标签固定到合并提交。
由既有 `Release (Linux)` 和 `Publish Docker image (Multi-arch)` 工作流发布
amd64/arm64 二进制、校验文件及 GHCR 多架构镜像。

预检确认三节点均为 `.326.guard.1` 且健康。升级前备份 PostgreSQL、Compose、
环境文件和模块目录，保留旧镜像，验证数据库备份目录并记录校验值。
只替换三个应用服务的镜像，结构化比较保证其他 Compose 配置不变。
逐台重建并检查健康、实际镜像、版本、模块状态、错误日志，再继续下一台。
PostgreSQL 和 Redis 容器不得重建。

## 兼容与回滚

模型校验新增配置字段、请求观测记录和渠道计数表，由宿主负责迁移。
旧配置没有阈值时，新宿主采用默认连续两次；旧宿主仍为单次关渠，滚动期间存在
短暂行为差异。因此滚动前通过模块启停 API 暂停检测并逐节点刷新，配置表不修改；
全部节点和 guard 0.2.0 ZIP 更新一致后恢复原模块启用状态。暂停期间不执行模型检测。
保留已有分组规则、模块启用状态和业务配置，不用真实不匹配请求触发关渠或通知。

任一节点检查失败立即停止后续更新，使用备份 Compose 对应的旧镜像恢复该应用。
回滚旧宿主前同时恢复备份中的 0.1.0 模块并逐节点刷新，旧宿主不识别 0.2.0 的容错能力。
新增数据库结构保留，不直接回写业务数据库；完整数据库恢复必须另行评估更新后写入。
数据库目录校验不等于完整恢复演练，Redis 快照不在本次备份范围内。

## 验证计划与已有证据

PR #236 的双模板组件、i18n、安全、类型检查与构建已经通过。
隔离 SQLite 上真实 Chromium 验证过桌面和手机弹窗、焦点恢复、重新打开、
公开目录无凭据请求，以及 guard 0.2.0 与 OKX 0.3.0 的真实下载上传回读。
本次检查发布产物校验值、三节点及公网 `/api/status`、无身份管理接口拒绝、
当前模板静态资源、模块配置保留、任务插件状态，以及 PostgreSQL/Redis 身份与启动时间。
没有后台会话时，不把管理 PAT 接口验证表述为线上登录态页面验收。

本次不改变外部模型 ID、API 请求格式、计费和令牌契约；渠道健康策略按已配置规则
支持白名单与渠道级连续计数，匹配清零，无模型名跳过。

## 执行记录

版本 PR [#237](https://github.com/moeacgx/maolaonewapi/pull/237) 的 CI `35371277412`
和 PR 质量检查 `35371277602` 成功后合并。标签 `.327` 固定指向合并提交
`1a97f6ff387c92f15391fb35b47a2659cd3c9386`，未夹带业务代码修改。

Linux 工作流 `35372189175`、GHCR 多架构工作流 `35372189059` 均成功。
[正式 Release](https://github.com/moeacgx/maolaonewapi/releases/tag/v1.0.0-rc.10.1.10.327)
已补齐中文说明，amd64/arm64 二进制及 `checksums-linux.txt` 已上传。
本机二进制下载连接未完成，停止该下载后改在目标主机下载，作业
`7e377f15-e019-4a5e-adcf-39004766dfbb` 对两份二进制执行 SHA-256 校验均为 OK。

- amd64 二进制：`a954dca4e1a709f1a450baa89f635111c20d34021156e531a48a3abcd6bdf72d`。
- arm64 二进制：`bf6b524b346482f3dcfedcdc005e906881d0df5e9d744fdda04a134a96addc7e`。
- GHCR 多架构摘要：`sha256:cffb5d5337013a671696bb0e816322f6fde852f700df6596be4637251af5d71a`。
- 实际运行 amd64 image ID：`sha256:7aa25a606026e8c876f2f943f2912e20813d9ca215405d92c650b06c08c0addd`。

### 备份与现场配置

预检作业 `59270e05-0b42-4012-96f8-13c7438ba89c` 确认当前 `zzapi` 未设置 `NODE_TYPE`，
另外两节点实际设置为 `slave`。本次以现场为准，保留角色与挂载，不根据历史名称推断，
由 `zzapi` 首先执行数据库迁移。

备份作业 `9ad7ed8c-550b-422b-8c7c-8d2d3fce2f30` 完成，目录：
`/home/docker/zzapi/backups/pre-327-20260918T170454Z`。
保存 PostgreSQL 自定义格式备份、Compose、环境文件、模块目录及脱敏配置快照。
数据库备份 60,958,754 字节，`pg_restore --list` 通过；未执行完整恢复演练。

- 数据库 SHA-256：`fdea44fc54891699bd38e67a8069b6c758bf6b9e56085734c6ec9e8b1c468a70`。
- Compose SHA-256：`86f0670aef5257c18040b0b486f82789e9427ed97c71a1cfe3715ca96453d1ca`。
- 模块归档 SHA-256：`aa233ac65d308de90de47a7cf4d07b89c89c6872e23122a03d4efe81dd116fc4`。

### 滚动更新

镜像拉取作业 `2f101d02-cc45-4f3e-b620-493de6488805` 的摘要与发布工作流一致。
准备作业 `dd7ce347-ead8-4bde-87d4-a5845741f3a6` 结构化比较 Compose，
确认仅三个应用镜像改变；通过模块启停 API 暂停 guard 并刷新全部节点。

以下作业依次完成，每台均验证 `.327`、healthy、restart=0、角色保持及迁移成功：

- `zzapi` / 18097：`db4194aa-3cb3-46fc-93b1-689fe331e365`。
- `zzapi-slave-1` / 18098：`b1850d0b-f6fc-4a0e-84e5-5cba42797c37`。
- `zzapi-slave-2` / 18099：`b67fd9ea-6c07-4bb3-a3d9-afd76b96e944`。

迁移核对新增配置字段、记录字段、`upstream_model_guard_streaks` 表及
`ux_upstream_guard_observation` 唯一索引通过，各节点启动日志无 panic/FATAL/ERROR 标记。

全部宿主健康后，作业 `088b7709-59e6-49f9-9a3c-3374ecc7c9c1` 通过真实上传接口
安装 `upstream-model-guard` 0.2.0，逐节点刷新并核对相同资源修订号，最后恢复原启用状态。
原检测配置未写入：`enabled=true`、1 条规则、`config_version=3` 及更新时间均保留；
新宿主读回默认阈值为 2，白名单为空。检测暂停时间为 UTC 17:15:16 至 17:19:32，约 4 分 16 秒。

### 最终验收

最终作业 `8912be17-6665-457f-8532-d790374289fe` 成功：

- 三节点实际 image ID、版本、健康和零重启均通过，公网 `/api/status` 返回 `.327`。
- 三节点 Root 在线目录配置正确；无身份扩展管理、在线目录及 `/v1/models` 请求均返回 401。
- 当前 Classic 扩展管理 HTML 及入口脚本返回 200；线上未使用后台会话执行双模板弹窗交互。
- 四个扩展均启用且无清单错误；guard 为 0.2.0，另外三个扩展版本和启用状态保持。
- 三个内置模块的 Default/Classic 入口资源修订号变化。与备份逐文件比对证明，仅 CRLF/LF
  换行符不同，文件集合及规范化字节相同；这是 Windows 补丁切换正式 Linux 构建的预期差异。
- 10 个任务插件的版本、启用、激活、来源及源码 hash 完全一致，`TaskPluginEnabled=true` 保留。
- PostgreSQL/Redis 的容器 ID、启动时间、运行状态和零重启均保持。

未触发真实不匹配请求、渠道关闭或 Bot 通知；检测语义由既有隔离回归保证。
证据保存在目标主机 `/home/docker/zzapi/releases/327/verification.json`，旧镜像和备份保留。
本次未操作 maolaoapi 或 zhishiapi。
