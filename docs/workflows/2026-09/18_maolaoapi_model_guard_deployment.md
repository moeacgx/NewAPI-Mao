# maolaoapi 逐节点更新与模型校验插件安装

## 授权与范围

用户明确要求逐个更新 maolaoapi 容器并上传上游模型校验插件。
目标为 CloudSSH `API中转站 / 美国netcup rs4000 猫佬API`，serverId=38、hostId=11，
仅更新 `maolaoapi`、`maolaoapi-slave-1`、`maolaoapi-slave-2` 三个应用。
使用 zzapi 已验证的 `.326.guard.1` 配套宿主和 `upstream-model-guard-0.1.0.zip`，检测保持关闭。

## 更新前证据

三个应用当前均为 `.321`、healthy、restart=0，镜像 ID 为
`sha256:1f57ba28821909ec0bef1d514d25cb6556078ac19986098bb730cc48fe5b1de3`。
实际运行二进制 SHA-256 为 `4544236f7863af39f74e2d177c5a931b3f34d6b7df8315f230899b3a34100b8e`。
端口为 18095/18100/18101，沿用原绑定；模块目录共享 `/home/docker/maolaoapi/data/data/modules`。

备份已保存到 `/home/docker/maolaoapi/backups/pre-model-guard-20260918T052510Z`，
数据库自定义格式备份 1,993,721,223 字节，SHA-256 为
`d25eeb228cc65d33e028e5173e53cbc8944609e2d2054bdea7bfccb28af7661f`，`pg_restore --list` 通过。
另保存 Compose、环境文件、模块目录、容器快照和渠道配置摘要。

## 执行方案

固定 `.326` 基础镜像摘要并验证二进制、插件 ZIP 和传输包摘要。
先在隔离容器验证真实上传，再将插件安装到共享目录且保持关闭。
Compose 结构比较必须确认只改变三个应用的镜像，然后按主节点、从节点 1、从节点 2 串行更新。
每台都检查健康、版本、重启数、实际管理接口的模块根目录、插件版本、加载错误与关闭状态。

`.321` 到 `.326` 包含预填分组唯一约束迁移和任务插件表。预检已确认旧 name 全局约束、目标部分索引均存在，
没有外键依赖、长事务或锁等待；迁移先进行限制锁等待的事务回滚演练。
主节点完成迁移后再更新从节点。已有数据库、Redis 容器不重建。

最终在实际 maolaoapi 上传接口重传同一 ZIP，检查三个端口与公网版本，并比较渠道配置摘要。
失败时停止后续更新，恢复原 Compose 和应用镜像；不自动回写业务数据库或删除新增表。

## 执行结果

2026-09-18 完成更新，三个应用均为 `v1.0.0-rc.10.1.10.326.guard.1`、healthy、restart=0。
这是已在 zzapi 验证的本地补丁构建，尚未正式发布到 GitHub Release。

关键作业：

- 现场预检：`76dcbe70-464a-4bf7-a6e0-683cbe96b583`。
- 数据库与配置备份：`cbda7617-2b23-4fb8-917d-37a88dbee48e`。
- 固定基础镜像构建：`fbe5a589-95c8-42e2-a6d8-1e3104547c0f`。
- 隔离容器真实 ZIP 上传与双模板资源验证：`165cef49-dd3d-45bf-8922-6e6df6dfac79`。
- 预填分组迁移事务演练并回滚：`28315724-01c3-4514-9f75-eaf20304761f`。
- Compose 仅三个镜像差异、模块安装并关闭：`fb6eb6e2-3bfd-4280-90ae-038cb7a9a123`。
- 主节点：`e9ef3365-434c-4f8f-a6d0-d6420ad37436`。
- 从节点 1：`30f69f57-c6e9-40f4-b87c-c66cbc4b863f`。
- 从节点 2：`c9f9beff-90b2-4074-bea7-a7eab588e596`。
- 实际上传和三节点最终验收：`c84d5e7e-1dc9-4dab-ba9f-6ca3fd5484c7`。

镜像标记为 `maolaonewapi:maolaoapi-326.guard.1-20260918`，实际 image ID：
`sha256:9d810f5e9eca6384cdb2c72c1b04b70d130f0303cd9b0167ee3f7a34968e5c71`。
二进制、基础镜像、插件 ZIP 与 [zzapi 同一工作项](18_upstream_model_guard.md) 的校验值一致。

同一 ZIP 已通过 maolaoapi 实际上传接口安装，三节点均返回插件 `0.1.0`、无加载错误、`enabled=false`，
资源摘要为 `3b42f1f1a5d3739114c3467a65297c4f9327e150f39fd225de5c6676321952e5`。
数据库配置行 1 条、启用配置 0 条、禁用记录 0 条，未配置规则或发送真实 Telegram 消息。
PostgreSQL 与 Redis 的容器 ID、启动时间保持不变。

服务器通过公网入口请求 `/api/status` 被 Cloudflare 返回 403；从操作端独立请求
`https://maolaoapi.com/api/status` 返回 `success=true` 和 `.326.guard.1`。
三个本机端口均独立验证版本成功。没有调整 Cloudflare 或任何访问控制。

## 验收差异与限制

从备份只读提取渠道数据，与现场逐项比较：备份及现场都是 194 条渠道，`models`、`group`、`model_mapping` 全部一致。
运行期间有 10 条渠道 `status` 发生变化：861、885、899、900、901、902、903、904、905、933。
所有变化渠道均无 `upstream_model_guard` 标记，插件保持关闭且禁用记录为 0。
这些运行期状态没有回写恢复；不能据此宣称所有渠道状态完全不变。
差异报告作业为 `b93afba5-416c-42da-8a42-a0cb543746de`，状态标记核验为 `de965f2d-54f2-41b8-930d-bb4ccbb26826`。

主节点日志确认执行数据库迁移并正常启动，`task_plugins` 和模型校验两表已存在。
但额外检查发现旧 `idx_prefill_groups_name` 全局唯一约束仍存在，`uk_prefill_name` 部分唯一索引也仍有效，
与预检状态一致；预期“旧约束消失”的断言未通过。未将此项标为迁移成功，也未在生产上直接删除旧约束。
具体成因尚未在本部署任务中确认，不影响已验证的插件安装和配置接口，应另行调查预填分组历史兼容逻辑。

现场验收文件：`/home/docker/maolaoapi/releases/326.guard.1/verification.json` 和 `channel-diff.json`。
备份和原 `.321` 镜像保留。回滚需恢复备份 Compose，再逐台重建应用；新增表可保留，业务数据库不自动回写。

## 启用后的原生页面加载故障

用户启用插件后，Classic 页面报告动态导入 `native/index/classic/entry` 失败。
2026-09-18 现场对照发现，共享 `state.json` 已保存 `enabled=true`，但 maolaoapi
主节点内存中的模块状态仍为关闭，两从节点为开启。三节点版本、资源摘要与清单校验均一致。
同时间 zzapi 主节点开启、两从节点关闭，因此测试实例某次打开成功不能证明所有节点已同步。

maolaoapi 日志在北京时间 14:02:14 和 14:02:21 记录主节点的 Classic entry 请求返回 403；
对应样式分别由两从节点返回 200。`OpenNativeAsset` 会拒绝本节点认为未开启的模块，
负载均衡将样式和入口分发到不同进程时即可出现该错误。此处提取仅保留时间、状态和资源路径。

通过每个节点的 Root 管理接口 `POST /api/extension-admin/refresh` 重新加载磁盘状态，
不重启容器、不重传 ZIP、不修改启用状态文件或规则。执行作业：

- maolaoapi 三节点刷新：`312f8408-06e1-4cc5-83fb-650662a98087`。
- zzapi 三节点刷新：`81087e4d-6368-48e5-a9fd-dc9f47c9439d`。
- maolaoapi 逐节点复查及历史资源日志：`795f7ae5-8d83-4518-8a66-5cd4cadb260c`。

刷新后六节点均为模块开启、版本 `0.1.0`、无清单错误、资源摘要一致、healthy、restart=0。
逐节点对比检测配置 SHA-256 与共享状态文件原始字节，确认操作前后完全一致：
maolaoapi 检测关闭、0 条规则；zzapi 沿用用户已有的检测开启、1 条规则。
各实例的 `releases/326.guard.1/registry-refresh-verification.json` 保存脱敏验收结果。

本次恢复了节点状态一致性，未修改宿主的跨进程自动同步机制。
后续安装、启停或卸载模块仍必须对所有节点执行刷新并逐节点核验；仅刷新负载均衡域名不能保证覆盖所有进程。
真实资源请求要求活跃后台会话，管理 PAT 不能替代资源 Cookie。
2026-09-18 用户在实际 maolaoapi 插件页面刷新或重试后明确确认“能正常打开”，页面恢复验收完成。
此结论来自用户的实际浏览器确认；Agent 未直接操作该后台会话，也未验证真实异常渠道关闭或 Telegram 投递。
