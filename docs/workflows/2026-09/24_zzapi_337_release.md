# zzapi 同步任务性能 .337 发布

## 范围与基线

用户明确授权合并 PR #274 到 custom-main，发布 v1.0.0-rc.10.1.10.337 并更新 zzapi。
合并前基线为 9dfdafce8a981047c8cd1623ca0ae9722b9f7e7f，PR 原提交 bb64a3a36。
发布包含同步任务模型广场采样、消费日志耗时、Classic 结果未保留说明、回归测试及技能文档。
合并前另补齐纯文本及 Cloudflare 错误包裹的性能过滤，避免脱敏后丢失过滤依据。

## 部署边界与步骤

CloudSSH 项目「API中转站」，serverId=52，hostId=17，「RS2000 德国建站」。
现场核实 /home/docker/zzapi 的三个应用均为 .335、healthy、restart=0；
端口分别为 zzapi=18097、zzapi-slave-1=18098、zzapi-slave-2=18099。

1. 最新基线复核、回归与 CI 通过后合并；固定标签并等待 Linux Release 和多架构镜像构建成功。
2. 保留 Compose、环境配置、容器快照、数据目录归档与 PostgreSQL 自定义格式备份，检查备份可读。
3. 拉取固定 .337 镜像，验证架构、摘要及 revision 与标签一致。
4. 仅替换三个应用的 image；结构化比较确认其他配置一致。
5. 按 zzapi-slave-1 → zzapi-slave-2 → zzapi 顺序重建，每节点健康、实际进程版本、API、
   未认证访问门禁与启动日志检查通过后继续。
6. 核对公网版本、Classic 静态资源及三节点配置；PostgreSQL、Redis 容器保持原身份与启动时间。

不修改 maolaoapi，不重启数据库/Redis，不修改渠道价格或令牌。
不发起收费模型推理；本地真实 Gin 路由与模拟上游验证行为，部署验证宿主与资源。
现场发现 cloudflare-jev 活动版本为 0.2.1，尚未提供实际输入/输出 Token；
为完成本轮 Token/TPS 升级，一并安装和逐节点激活已发布的 0.2.3，保留旧版本供回退。
源码来自插件仓库固定提交 ad210868a2fa95b070b63bf9d46deb4b0afef639，
11,344 字节，SHA-256 为
`fbb6beadfea0f4d6573abdb654ba61ae22a03465f004111b535da5b5ca8da311`。
安装使用既有 Root 管理接口，凭据仅在目标主机内存读取和本机回环访问，不输出、不修改。

## 回退与验收

出现失败立即停止后续节点，使用备份 Compose 和保留的 .335 镜像恢复失败应用。
不自动覆盖在线数据库。保留任务账务行与旧插件版本，不补造历史性能或 Token 数据。
新请求在性能收集开启且符合过滤条件时开始采样。
发布成功、节点升级成功与真实供应商调用成功分别记录，不混为一次验收。

以下记录为本轮实际执行结果，不替代真实供应商推理验收。

部署前备份作业 `3c7b14f4-b45d-42e1-94bd-bfeceab1a66a` 已成功，目录
`/home/docker/zzapi/backups/pre-337-20260924`。数据库备份 20,073,109 字节，
`pg_restore --list` 通过；未执行恢复演练。数据库 SHA-256：
`5dc2ee73816ab7b3197430cfe63592ac83afd83d780a1277e62b7c68371c36e8`。

## 合并与发布

PR #274 已合并，标签提交为 `33aad9aec758e2556e53cf6a060c9118d2972459`。
最终 head 的 CI `35989534845` 前后端均通过；Linux Release `35990153506` 成功。
两架构二进制下载后均与发布的 checksums-linux.txt 校验一致：

- AMD64：`bb3d3956c81755e0b973c5fd9d7f8a91465e3dbe49ce20f8e97679d84acd2aa2`。
- ARM64：`59caea0fee951245e154f3ffa6dc917d1177825164c8f0b279207dfeded5cbc3`。

合并前复核补充了失败过滤红绿回归、数字错误码与 SSE 异常回归，并精确检查成功率。
相关五个 Go 包全量测试及 vet 通过，未改 relaykit 公共接口。

Docker 多架构工作流 `35990153426` 成功，AMD64/ARM64 及 manifest 均已发布。
现场镜像 revision 与标签提交一致：

- manifest 摘要：`sha256:689959811eae298cb7a4f7fd422a7186c1a650aeb07d415bdfedb25f9b61993d`。
- AMD64 image ID：`sha256:b0d59d697b1ca871fe0f09ff8cf06975246c098595eb294c604569420d6ea1ed`。

## 逐节点部署结果

三个节点均完成实际进程版本、/api/status、healthy、restart=0、镜像身份及配置一致性验证。
无身份访问 user/log/models/plugin 管理接口均为 401，启动日志无 panic 或 fatal。

1. zzapi-slave-1 / 18098：作业 `8deb6123-b4bf-4097-90f7-d92dc7be2ad3`。
2. zzapi-slave-2 / 18099：作业 `68b1e085-5b35-4219-a95b-2a0056bbdc05`。
3. zzapi / 18097：作业 `df0f81e4-e209-4d30-8342-20d0c43c8a0c`。

以上作业均 SUCCEEDED、exitCode=0，实际运行版本为 .337。
插件更新作业 `54016574-746d-4944-b7bc-c875ebd111f2` 成功：
三个节点的 cloudflare-jev 均为 0.2.3、active/enabled=true，hash 一致，保留旧 0.2.1。
安装前后渠道和 options 配置指纹一致。

最终作业 `a7abea07-48ad-4e6e-baa7-9a289a383d4c` 成功：

- 三节点 Classic 入口脚本 `/assets/index-CmuccVAg.js` 均为 11,825,570 字节，包含新结果说明。
- 环境变量、端口、挂载、命令与原备份一致；数据文件 SHA-256 校验通过。
- PostgreSQL/Redis 容器 ID、启动时间、重启次数均未变。
- 远端及本机独立访问公网 /api/status 均为 success=true、.337。

准备脚本最初把 `docker compose config --format json` 输出当 JSON 解析失败；
现场该版本仍返回 YAML。重新读取实际文件，使用 YAML 解析后结构化比对通过，仅三个应用 image 改变。
失败发生时没有重建任何容器；之后依照节点门禁完成部署，无回退。
本次未请求真实供应商推理，不宣称已产生新 Jev 性能样本；历史数据不回填。
