# Classic 会话刷新限流与误退登修复

## 目标与证据

Classic 在业务请求返回 401 后尝试刷新会话，但拦截器的空 catch 丢弃刷新错误，
继续把原 401 交给 `showError`，导致清除本地用户并跳转登录页。
刷新接口返回 429、503 或网络异常时都能触发这一问题。
Default 已通过 `auth-session.ts` 将此类错误归为临时故障，本次不修改 Default。

2026-09-19 线上只读日志显示刷新 429，随后退出、OAuth 状态与登录请求也返回 429。
三节点 GA 为 12000 次 / 60 秒，CT 没有环境覆盖，使用默认 20 次 / 1200 秒。
观测流量远低于 GA 阈值，与共享 CT 桶冲突高度吻合；没有直接读取 Redis 桶计数，
也不能仅凭相同 IP 将日志唯一归属于某位用户。生产容器未执行变更。

## 范围与方案

- Classic 拦截器将真实刷新错误传递给展示层与调用方，保留原业务请求的
  `skipErrorHandler`；网络错误缺少 response 时仍可正常展示。
- 刷新返回 429、5xx 或网络错误时保留本地用户，不跳转；明确的刷新 401 仍按现有
  认证失效规则处理。明确的 `409 AUTH_SESSION_MISMATCH` 清理旧会话并要求重新登录，
  普通 409 与 `AUTH_REFRESH_RACE` 不清理用户；调用者仍收到真实的 409。
  刷新成功仍只重放一次原请求。
- `POST /api/user/auth/refresh` 使用独立的按 IP 刷新限流桶，沿用
  `CRITICAL_RATE_LIMIT_ENABLE`、`CRITICAL_RATE_LIMIT`、`CRITICAL_RATE_LIMIT_DURATION`。
  普通关键接口的 CT 消耗不再阻塞刷新，刷新也不再消耗 CT。
- GA、刷新 Cookie 校验、来源检查、会话撤销、权限与其他敏感接口的防护保持有效。
  刷新独立桶耗尽仍返回 429 与 `Retry-After`；不提供管理员全局免限流。

## 兼容性与安全边界

无数据库迁移、无新配置、无模型调用或计费契约变化。独立刷新桶仍支持共享 Redis
和进程内模式。回滚会恢复共享 CT 计数以及 Classic 的旧错误处理，应避免单独回滚
前端修复。现有 Cookie、SID、Bearer 与会话期限不变。

## 验证计划

- 使用真实 Axios adapter、实际 `showError` 与本地存储，覆盖业务 401 后刷新
  429、503、网络异常、明确 401、会话身份冲突、普通冲突、刷新竞争、成功重放、
  直接 429 和原请求静默错误配置。
- 通过真实 API 路由验证普通 CT 耗尽不阻塞刷新，刷新仍有限流，GA 仍生效。
- 覆盖 Redis 和进程内模式的刷新隔离、429 响应，以及 Redis 限流窗口恢复。
- 执行受影响测试、Classic lint / 构建、Markdown 格式化和 `git diff --check`。

## 验证结果

- Classic 新增 10 项真实 Axios 回归全部通过；修复前 429、503、网络错误、静默错误
  透传四项失败，身份冲突边界补测也先复现失败再修正。
- 既有 Classic 认证兼容测试 9 项通过，修改文件 ESLint 通过。
- Classic `bun run build` 通过；构建仍提示 Browserslist 数据较旧及部分分块大于 500 kB。
- `go test ./router ./middleware -count=1 -timeout 60s` 通过。
- `go test ./router -run TestAuthRefresh -count=2 -timeout 60s` 通过；内存测试使用
  独立文档 IPv6 来源隔离进程级计数，避免重复执行受到上一轮影响。
- Markdown 与新测试 Prettier 检查、文档入口链接存在性、`git diff --check` 通过。
- 两轮只读审查指出的身份冲突处理和测试计数隔离均已修正并复核。
- 仅修改 Classic 和共享后端刷新限流，不修改 Default 的既有临时错误处理。
  未改动模型 ID、API Key、Relay 请求/响应或计费；未执行部署或生产数据操作。
