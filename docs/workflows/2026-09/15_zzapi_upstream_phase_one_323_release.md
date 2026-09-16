# zzapi 上游兼容第一阶段 .323 发布

## 目标与范围

发布 `v1.0.0-rc.10.1.10.323`，将 `custom-main@84395db6a` 已合入的
#210–#216 第一阶段兼容补丁部署至 zzapi。覆盖认证安全兼容、WS DTO 准备、
Default/Classic 后台、任务计费、数据库兼容和 Responses HTTP/SSE usage。
完整 JS 插件、统一 OAuth 切换和 WS 运行时仍未启用。

发布目标为 CloudSSH `API中转站 / RS2000 德国建站`，`serverId=52`、`hostId=17`，
目录 `/home/docker/zzapi`。仅滚动更新 `zzapi`、`zzapi-slave-1`、`zzapi-slave-2`，
端口分别为 `18097/18098/18099`；不更新 maolaoapi、zhishiapi，不重建 PostgreSQL 或 Redis。

## 现场核对与迁移

- 更新前三应用镜像均为 `ghcr.io/moeacgx/maolaonewapi:v1.0.0-rc.10.1.10.319`，
  实际 `/api/status` 版本一致，running/healthy，重启次数 0。
- 现场两个 slave 的 `NODE_TYPE=slave`，主应用未设该值；按当前代码由主应用执行迁移。
  本次以现场配置为准，不沿用旧文档的角色等价假设。
- PostgreSQL 15.17，钱包 quota 为 bigint；预填分组表为空，无传入外键。
  旧 `idx_prefill_groups_name` 全局唯一约束与 `uk_prefill_name` 软删除部分索引并存。
  本版本启动迁移应移除前者、保留后者。不得执行业务数据清空或强制 CASCADE。
- 数据库约 15.3 GB，可用磁盘约 718 GB。先备份 Compose、数据库及迁移前索引定义。

## 发版与部署步骤

1. VERSION 与本文经发布 PR 合入最新 `custom-main`，发布 `.323` tag。
2. 等待 Linux Release 与 Docker amd64/arm64 构建、版本 manifest 完成；核对源码 revision。
3. 备份保存在 zzapi 私有目录，验证数据库备份可列目录，保留旧 `.319` 镜像。
4. 只修改三个应用的 Compose image，解析后确认其他所有配置相同。
5. 逐节点更新，优先完成主应用迁移及健康检查，再更新两个 slave；每步确认版本、
   Docker health、重启次数及其他应用/数据库/缓存容器身份。
6. 核对三端口与公网 `/api/status`，检查前端入口、定价 API、近期启动错误和预填索引。

## 回滚与验证边界

节点异常时停止滚动，使用备份旧镜像逐个恢复已更新应用。保留备份和历史账务，
不自动恢复全库覆盖新数据。删除旧全局索引是兼容软删除的迁移，旧应用仍可使用保留的
部分唯一索引；只有需要精确数据回退时另行制定恢复方案。

合并前 7 项 PR 的 CI、组合 Go/relaykit 测试和双模板构建已通过。
本次上线验证涵盖产物来源、迁移、版本、健康和公开只读入口；不制造付费推理请求、
真实账户/渠道修改或 Telegram 消息。硬件 Passkey、完整业务流程仍需后续验收。

## 执行结果

发布和滚动验证完成后，在本工作项补充 tag/source、镜像 digest、CloudSSH 作业、备份位置、
前后容器身份与 PostgreSQL 索引结果；准备状态不能视为部署成功。

## 已执行结果

- 源码 tag：`v1.0.0-rc.10.1.10.323`，合并提交 `5d90556e217576ce17fa5f27ddc2c7260ad16b50`。
- Docker 多架构 workflow `34952221388` 成功，manifest digest：
  `sha256:687b240aee7a3c7e587c39399c223f41ba125dd570724142c8e0b1f816579c7a`。
  amd64 节点实际镜像 ID：`sha256:6e6014fc7b236793a3fdf7e7b87772341a5fdef9f56c4476fe633396afd4c267`。
- Linux Release workflow `34952221452` 首次因 GitHub Release API 500 失败，重跑后构建成功；资产接口偶发 500 已记录，未影响 GHCR 镜像。
- 更新前备份 CloudSSH 作业：`6a27de29-796b-47e4-b977-49b9ca752e5c`；备份目录：
  `/home/docker/zzapi/backups/release-323-20260915`；Compose SHA-256：
  `3f8444d2fdf823b705f2387e01031b5432bb60d38f55703d7481e4a671f94189`；PostgreSQL dump 约 161,564,981 字节，982 个 TOC 条目。
- 镜像准备和 Compose 仅替换三个应用 image 的 CloudSSH 作业：`f603e274-f95b-47f7-bf8b-437a8928ea26`；更新前 PostgreSQL 15.17、预填表 0 行、无入向外键，钱包 quota 为 bigint。
- 三节点滚动作业按 `zzapi` → `zzapi-slave-1` → `zzapi-slave-2`：
  `32493caa-5feb-45e8-a7c9-03afe1346166`、`23bbcddf-659b-469f-ba1f-d20c4ab0a60f`、
  `924d4ff8-ad94-4b12-9d72-44b5948e52df`。每一步都确认新镜像、healthy、running、重启 0，
  其他应用和 PostgreSQL/Redis 容器 ID 未被替换。
- 最终验证作业：`a5458be1-841c-4c6b-9e28-231721a952f5`。三个本地端口 `18097/18098/18099`、
  公网 `/api/status`、`/api/pricing`、`/`、`/console` 均成功；启动日志无 panic、fatal、数据库初始化失败或迁移异常。
- PostgreSQL 最终状态：旧 `idx_prefill_groups_name` 约束 0 个，`uk_prefill_name` 部分唯一索引 1 个，
  预填表 0 行、入向外键 0 个，users.quota 类型 bigint。迁移前 dump 和 Compose 备份保留。
- 迁移实现未自动完成旧约束删除，现场在备份和无外键前提下执行一次目标约束删除作业：
  `a8bedc70-30ab-49a0-8824-2d1c6a53ddf9`；该手工步骤属于本次 zzapi 部署审计，后续应修复代码并在测试库复现。
- maolaoapi、zhishiapi、PostgreSQL、Redis 未重建；未创建真实渠道、未发起付费推理请求、未使用或记录任何凭据。
