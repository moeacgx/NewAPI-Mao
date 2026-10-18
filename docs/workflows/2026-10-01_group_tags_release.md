# v1.0.0-rc.10.1.10.344 分组标签发布准备

## 范围与基线

本次发布包含已合并的 MAO-6 分组标签功能（PR #291，合并提交 `3bd3200ad04fbbd2a199bb79669f47fe6e5cc3f4`）。
Classic 后台可维护分组标签、组合 Logo 和稳定 Group ID 绑定；前台创建/编辑令牌按标签筛选分组，保留原 ID/code 提交契约。

本版本仅更新 `VERSION` 与发布记录，不修改业务代码、数据库、渠道配置或生产配置。

## 验证与发布边界

- 后端 `go test ./model ./controller ./router -run 'Group|Token.*(Auto|Exclusive|Migration)' -timeout 60s -count=1` 通过。
- Classic 28 项兼容测试、ESLint、Prettier、`git diff --check` 和 `bun run build` 通过。
- 本地 Playwright 已验证 1440/390 宽度、前后台页面边界、长名称/六 Logo 对齐和跨标签令牌提交。
- PR #291 的后端 CI、前端 typecheck/test、PR quality 均通过。
- 版本 PR 合并后由 tag `v1.0.0-rc.10.1.10.344` 触发 Linux Release 与 GHCR 多架构构建；构建成功后再更新 zzapi。
- 生产更新沿用 zzapi 三节点逐一健康门禁；不修改 maolaoapi，不重启 PostgreSQL/Redis，不修改渠道价格或令牌。

## 实际 zzapi 更新结果

- PR #292 已合并到 `custom-main`，Release `v1.0.0-rc.10.1.10.344` 已发布；GHCR 镜像为
  `ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.344`，多架构 manifest 摘要为
  `sha256:df04ffd3a8757488f424e81c24952db873dff4f8b7248768c0ee0e3c3439c033`。
- 通过 CloudSSH 项目「API中转站」的 `serverId=52` 更新 `/home/docker/zzapi`，按
  `zzapi:18097`、`zzapi-slave-1:18098`、`zzapi-slave-2:18099` 顺序逐一重建。
- 三个应用容器均使用镜像层 `sha256:74cc65560778353cc3916b0c51a683fa39e045c9f7d773727a92e64d2abce0f0`，
  健康状态为 `healthy`、重启次数为 `0`，`/api/status` 返回 `.344`；匿名访问
  `/api/user/self`、`/api/group/tags`、`/api/token/` 均返回 `401`。
- PostgreSQL 自定义格式备份和 `pg_restore --list` 均通过。备份目录为
  `/home/docker/zzapi/backups/pre-344-20260930T162553Z`，dump SHA-256 为
  `824713c44fd9312e61bbe5129081308a8226e42be6ed580a49331da1b6028a84`。
- 最终数据库计数为 users 62、tokens 122、groups 21、channels 130、options 196；
  `options` 哈希保持不变，`group_tags` 与 `group_tag_bindings` 已创建。渠道哈希的差异仅来自
  channel 601 的 TokensPro 同步运行时间字段，JSON 键集合未变化；其余渠道字段无差异。
- PostgreSQL 与 Redis 容器 ID、启动时间均未变化；未发现 panic、fatal、迁移失败或缺表日志。
