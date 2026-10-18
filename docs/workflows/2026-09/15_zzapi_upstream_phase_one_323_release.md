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
