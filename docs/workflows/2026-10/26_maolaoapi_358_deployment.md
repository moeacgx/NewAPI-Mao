# maolaoapi .358 逐节点更新

## 授权与现场边界

用户明确授权：#359 合并后发版，再逐个更新 maolaoapi。使用 CloudSSH 技能，实时确认“API中转站 / OVH兜底”（预期 serverId 63、hostId 29）及 `/apps/maolaoapi` 的 maolaoapi-ovh 项目。只更新 maolaoapi、maolaoapi-slave-1、maolaoapi-slave-2，主脑串行执行生产写入。

不修改数据库／Redis容器、应用环境变量、挂载、端口、资源限制和现有主题。不调用收费供应商或真实支付接口。为控制跨版发票兼容风险，仅在滚动窗口对现有反向代理添加临时发票路径维护保护，先备份并测试配置；全部节点一致后恢复原配置，不改变长期路由。

## 更新前门禁

- 现场读取当前版本、准确镜像 digest、三个节点健康与重启次数、数据库和 Redis 身份、Compose 路径、静态资源目录及可用空间，不依赖旧版本记录。
- 备份 Compose、必要容器配置、数据库 custom dump、表结构和持久化数据；敏感配置只保存在远端私有备份，不打印环境变量或凭据。验证 pg_restore 目录、字节数及 SHA-256。
- 从已完成校验的正式发行拉取固定镜像，断网只读运行 `--version`，核对 OCI revision、架构与当前默认运行配置。
- 沿用现有静态资源机制，预置新资源且保留旧哈希，避免滚动更新期间前端请求打到不同版本节点。

## 逐节点顺序与验证

1. maolaoapi 主节点（由既有启动流程执行必要迁移）。
2. maolaoapi-slave-1。
3. maolaoapi-slave-2。

每次仅替换一个应用镜像，使用 `--no-deps --no-build`；确认上一节点 healthy、版本和 revision 正确、restart=0、配置未变、匿名鉴权正常、资源可读及启动无严重错误后才推进。任一项失败即停止下一节点并评估应用回滚。更新前暂时保护发票查询、申请、支付与管理入口，覆盖可能触发历史释放的订单读取；等待既有请求排空后再进入新旧混合窗口，三节点验收通过后恢复。

## 迁移与回滚

`.357` 到 `.358` 无数据库迁移；若现场仍为 `.355`，跨版包含 `.356` 的可空 TEXT 字段 `invoice_records.closed_orders_snapshot`。部署前备份，主节点迁移后核对新增列，不删列、不整库覆盖。回滚前先停止新的开票操作，并阻止可能触发历史占用释放的订单查询；评估已释放来源及历史快照，确认后再回滚应用，不能仅以没有新增申请判断安全。关闭不自动退款。

回滚使用现场确认并保留的旧固定镜像，只恢复应用配置并逐节点重建，不回滚数据库、主题或业务数据。旧版会恢复头部模型检测并可能隐藏新充值日志 IP，必须在回滚说明中明确。

## 结果状态

部署于 2026-10-27（Asia/Singapore）完成。三应用已逐个从 `.355` 更新到 `.358`；发行 revision 为 `ef27e266f456038706e3f4fbfa9fa5380b557c99`。

### 备份与候选

- 现场身份与计划一致：CloudSSH 63／host 29，Compose 项目 maolaoapi-ovh。旧三应用均 healthy、restart=0，旧镜像 `ghcr.io/moeacgx/newapi-mao@sha256:62fdd34becf5d62ff2444b21bbaa797081be4bc891b78271248cef9d174f5e12`。
- 备份目录 `/apps/maolaoapi/backups/pre-358-20261026T160806Z`，共 11 个清单文件。数据库 dump 1,527,494,837 字节，SHA-256 为 `83f57349d70231ec90220c72e549a50f443506ca5098dc3b825d259d099df1c8`；数据归档 88,640,019 字节。`pg_restore --list` 和最终全量备份哈希检查通过，私有文件仅保留主机。
- 候选固定 amd64 镜像 `ghcr.io/moeacgx/newapi-mao@sha256:8ba1baab62c569f64889750d3897d52084e02c9d6bf9f5489ebe3d70c08c4d47`。现场 Docker containerd 的 Image ID 使用 manifest 摘要；已核对 Descriptor、RepoDigests、架构、version、revision，未混用 OCI config digest。
- 镜像默认配置比较与断网只读 `--version` 通过；新旧 Classic 资源共 176 个、33,408,131 字节，经本机 HTTPS 逐项哈希校验。隔离 SQLite 预览未接入生产数据库，预置完成后已停止。

### 逐节点与临时保护

- 发票入口保护作业 `0c329577-9e12-40a9-ae99-d403ebfa3e42` 成功，旧 nginx worker 自然退出后才更新主节点。
- 主节点首次更新作业 `216c11c4-b71b-467e-baa7-32cf8d970019` 在验收时因 Binds 列表顺序变化停止；只读确认路径、权限和实际挂载完全一致。仅修正无序列表比较，使用 `--verify-only` 完成验收（`05568997-34f9-4d87-972b-5ea7885200ed`），未重复重建。
- 第一从节点作业 `d9ef7c92-70a2-4028-b522-1b253148337f`、第二从节点 `67fd8558-9457-4f03-9526-f608c6618b75` 均成功；每个节点通过后才推进下一节点。
- 三节点均为固定 `.358` 镜像、准确 revision、healthy、restart=0；每节点 13 个匿名保护接口返回 401，静态资源正常，缺失资源为 404/no-store，启动严重错误标记为零。
- `invoice_records.closed_orders_snapshot` 为可空 TEXT；既有列定义、关键表身份、主题 Classic、有效应用配置及 PostgreSQL/Redis 容器身份和启动时间均未变化。
- 集群验收及解除保护作业 `8052ace4-92e5-4c0f-9e85-59dcb4ffc631` 成功。nginx 配置字节级恢复到备份，发票入口恢复原鉴权；未执行回滚。

### 最终验收

- 解除保护后再次核验作业 `e0a71e84-a994-4637-ae69-7046f431c273` 成功。最终报告为备份目录中的 `deployment.final.json`。
- 生产机本地 HTTPS 与外部独立请求均确认 `/api/status` 为 200、`.358`／Classic；发票、工单统计、用户和模型匿名入口为 401。公网 JS/CSS 与缺失资源缓存策略通过。
- 外部 JS SHA-256：`844d46df402461ec312d3a593056a333f6744593552a949b09272e8cbbcec0cb`；CSS SHA-256：`d7d4163aee0dbfe13d36f2c9428fdf517b94445ccba8f97f9395805f05014d0e`。
- 初次生产机默认 User-Agent 公网请求曾为 403；随后本机 HTTPS 及带浏览器标识的公网请求通过，未修改 WAF。
- 本轮没有真实支付、登录态浏览器或收费模型调用验收；上述结果证明发行、滚动更新、数据结构与基础接口正常，不替代完整业务验收。
