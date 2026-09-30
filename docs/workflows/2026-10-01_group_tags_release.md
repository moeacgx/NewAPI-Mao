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
