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

发布及生产更新进行中；实际标签、工作流、镜像摘要与逐节点结果完成后补齐。
