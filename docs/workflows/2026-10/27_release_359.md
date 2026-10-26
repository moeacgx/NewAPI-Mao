# .359 Skills 私有 Chat SSE 修复发布与部署

## 目标与范围

用户授权在 CI 通过后发版并更新 maolaoapi。本版纳入 PR #362：仅在私有 Chat
发现阶段聚合上游意外返回的 SSE，保留技能读取、授权及正式生成边界。
不修改数据库结构、模型 ID、鉴权、价格或用量语义；Default/Classic 页面不变。
修复与 197 项回归见[实现记录](27_gateway_skill_multiple_choices.md)。

## 发布门禁

- #362 修正后的 Backend、Frontend CI 通过，独立复核和本地 go vet 通过。
- 版本 PR 校验通过后，在合并提交创建 .359 附注标签，不移动旧标签。
- 等待 Linux Release 和双架构 Docker 发布完成；检查二进制校验和、镜像
  amd64/arm64 清单、version/revision，生产只使用固定镜像摘要。

## 生产目标与备份

2026-10-27 实时确认 CloudSSH「API中转站 / OVH兜底」，serverId 63、hostId 29。
项目 maolaoapi-ovh，Compose 为 /apps/maolaoapi/docker-compose.json。
三个应用当前均为 .358、healthy、restart=0，revision 为
ef27e266f456038706e3f4fbfa9fa5380b557c99。旧固定镜像为
ghcr.io/moeacgx/newapi-mao@sha256:8ba1baab62c569f64889750d3897d52084e02c9d6bf9f5489ebe3d70c08c4d47。

备份目录 /apps/maolaoapi/backups/pre-359-20261026T170925Z（目录时间为 UTC）。
包含 Compose、容器配置、数据库 custom dump、schema、TOC 和 data 归档；文件只保存在
远端私有目录，不打印环境变量或凭据。数据库 dump 1,538,177,633 字节，SHA-256 为
642c88b019f2f86e556ceb1c26dd4e7ff1ffb9c795e5614e41fdb0c4e0b6e00e；pg_restore --list 通过。
首次备份使用错误数据库角色失败，已按容器自身 POSTGRES_USER/POSTGRES_DB 重做成功；
失败目录不作为有效备份。

## 更新、验收与回滚

1. 候选镜像检查，保留旧镜像，预置新静态资源并保留旧哈希。
2. 串行更新 maolaoapi、maolaoapi-slave-1、maolaoapi-slave-2；使用 --no-deps --no-build。
3. 每节点验证 healthy、restart=0、准确版本/revision、端口/挂载/环境/资源限制保持，
   状态接口、匿名鉴权、静态资源及启动错误检查通过才进入下一节点。
4. 最后复核集群、公共入口、备份哈希，以及数据库/Redis 容器身份保持。

.358 到 .359 无新增迁移，无需修改发票入口。失败时停止后续节点，恢复备份 Compose 中
对应应用镜像并逐节点回滚 .358，不回滚数据库或业务数据。旧版本可能重新出现私有 SSE 502。
仅更新猫佬三个应用，不更新同机 zhishiapi、DouDi、PostgreSQL、Redis 或反向代理配置。

## 验收边界与跨项目影响

健康检查、版本与模拟回归不等同于用户真实付费请求验收。本次不重放付费请求，不执行客户端工具。
NewAPIForDouDi 与 NewAPIModifyByGang 仅在采用相同私有 Skills 发现链路时值得适配；
未核查目标实现，不自动发送 Issue 或 PR。对外请求接口和计费规则不变。

## 结果

- 修复 PR #362、版本 PR #364 均已通过 CI 并合并；.359 标签提交为
  2b08e92bd8b88611768ba3edb5013166918e0af1。
- Linux Release 工作流 40367475936、Docker 双架构工作流 40367475975 均成功。
  两个下载的二进制通过 SHA-256 与 ELF 架构核对：amd64 为
  3bd6f94438767409765c1972cf6ce06385a7f294d15ec163fcfa9929cfceac33，arm64 为
  7ea814880c21a878bdd95fc3ee7910851582ba77d607cc3c2e8724a959539df4。
- 双架构索引为 sha256:34b7514220c9babcfe3ac1284f6c060190077e62b86bcb3531115af305889146；
  生产固定 amd64 镜像为
  ghcr.io/moeacgx/newapi-mao@sha256:e7811dc2815f850a958de6c36404c474a7e60db87be58e32b6c74da6a97785de。
  两架构 version/revision、镜像默认配置和断网只读 --version 检查通过。
- 静态资源预置作业 61d5dfb1-36a6-4a0f-bef6-f1f076e9c8b5 成功：125 个文件、18,881,412 字节，
  本机 HTTPS 哈希逐项一致。两项既有 `_base*` 虚拟引用仍为 404，未改写为伪资源。
  预览使用独立 SQLite、network=none，未接入生产数据库，提取后已停止。
- 三节点依次更新并验收通过：主节点作业 9cc27817-8985-4e7e-942a-e4374b9e085d，
  第一从节点 1a3d4f0c-a614-463c-a078-62ca51da5075，第二从节点 55fc412f-10d4-43a7-af90-fe212501d757。
  均为准确 .359 镜像/revision、healthy、restart=0；有效配置保持，六项匿名保护接口均 401，
  JS/CSS 与预置哈希一致，缺失资源为 404/no-store，启动严重错误标记为零。
- 最终集群复核作业 41b09b0b-baf9-4c51-b5dc-1e3f140b40d7 通过：Compose 只改变三个应用镜像，
  数据库 schema 未变、PostgreSQL/Redis 身份及启动时间保持，全部备份文件重新校验通过。
  最终报告为 /apps/maolaoapi/releases/359/deployment.final.json，备份目录保留同名副本。
  未触发回滚，旧固定镜像与 Compose 可供回退。
- 独立外部 GET /api/status 确认 200、.359／Classic；用户、模型、提示词中心、工单统计的匿名
  请求均为 401。公网 JS/CSS 返回 200 且 SHA-256 与源站一致，缺失资源为 404/no-store。
  JS SHA-256 为 844d46df402461ec312d3a593056a333f6744593552a949b09272e8cbbcec0cb，
  CSS 为 d7d4163aee0dbfe13d36f2c9428fdf517b94445ccba8f97f9395805f05014d0e。
- 生产滚动更新及基础验收已完成。未调用真实付费模型，不宣称已复演原用户完整会话。
