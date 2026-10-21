# maolaoapi .349 当前连接发布

## 目标与基线

将 Classic 当前连接、令牌实时使用状态及最终固定卡位布局发布为
`v1.0.0-rc.10.1.10.349`，并保留 maolaoapi 已运行的 dev 能力。功能仅为 Classic
新增页面呈现；活动请求接口由共用后端提供，Default 本次不增加入口。

- 生产 dev 基线：`bbb3180d59a75cd12a7a2854a66c02e48551df3c`。
- 功能 PR：[#312](https://github.com/moeacgx/NewAPI-Mao/pull/312)。
- 功能合并提交：`7785c19e98b908d7edbad983a89831a5b2932e83`。
- 整合分支从上述 dev 基线创建，再合入 `origin/custom-main`；业务代码自动合并，
  `docs/developer/README.md` 与 `custom-development.md` 按段保留两边内容。
- 整合提交、PR、发布提交、标签、Release 和镜像摘要在实际完成后登记，不能以计划值
  代替远端证据。

## 兼容与安全边界

- 当前连接不新增数据库表、迁移、环境变量或持久业务数据；多节点通过现有 Redis
  短租约汇总，旧键会自动过期。
- 请求注册、首响应观察和完成清理不改变路由、重试、用量与计费；生产更新期间只有已
  升级节点可发布活动快照，全节点完成后再验收集群汇总。
- 只更新 `maolaoapi`、`maolaoapi-slave-1`、`maolaoapi-slave-2` 三个应用镜像，
  不重建或修改 PostgreSQL、Redis，不更改端口、挂载、命令和环境变量内容。
- 本次只执行状态、静态资源、鉴权和日志等基础验收，不调用收费模型，也不把基础部署
  验收表述为真实供应商业务验收。

## 发布门禁

1. 整合分支同步最新 `origin/custom-main`，确认 `behind=0`，运行活动模块测试、相关
   Go 回归、独立 relaykit 构建、Classic 生产构建与差异检查。
2. 整合 PR 的后端、前端和审查门禁全部通过后才合并；标签必须指向准确的
   `custom-main` 合并提交。
3. 验证 Release 工作流、Docker 工作流、三个 Release 资产、amd64/arm64 manifest，
   并确认 GHCR OCI revision 等于发布提交。
4. 生产 Compose 固定到已验证的正式镜像摘要，不使用浮动 `latest`。

## 生产备份、顺序与回滚

目标为 CloudSSH 项目“API中转站”的 OVH 兜底主机；执行前重新核对 serverId、hostId、
根目录和三应用实际镜像。备份至少包括 Compose、三应用与 PostgreSQL/Redis inspect、
应用数据目录、PostgreSQL custom dump、`pg_restore --list` 结果和 SHA-256 清单。

节点顺序固定为：

1. `maolaoapi-slave-1`
2. `maolaoapi-slave-2`
3. `maolaoapi`

每个节点必须在 healthy、restart=0、版本/revision/镜像摘要、配置、401 鉴权、Classic
资源标记和启动日志检查通过后，才更新下一节点。任一节点失败即停止，使用备份 Compose
和部署前固定 dev 镜像回滚已更新应用；在线数据库不随应用回滚恢复。

## 实际证据

本节在发布和生产更新完成后填写准确提交、工作流、资产、镜像摘要、备份目录、CloudSSH
作业、逐节点结果及公网状态。未填写前只表示已完成发布设计和回滚准备，不表示 `.349`
已经发布或部署。
