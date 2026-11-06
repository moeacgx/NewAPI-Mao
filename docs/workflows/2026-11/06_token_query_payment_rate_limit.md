# 令牌查询与充值限流隔离

## 问题与证据

2026-11-06 线上用户的 `/api/user/pay` 返回 HTTP 429。该账号当前登录出口
先后密集调用 `/api/usage/token/`，两个路由均经过共享的按 IP 关键操作限流，
导致只读余额查询耗尽额度并阻塞充值。GET `/api/log/token` 也使用相同限流。
本次仅清除该用户当前出口的 CT Redis 计数键以恢复充值；未更改余额、订单或全局限流。

## 修改方案与契约

- `/api/usage/token/` 与 `/api/log/token` 在 Bearer 只读令牌认证后按令牌所有者限流。
- 两类查询使用独立作用域，不共享彼此计数，也不再消耗充值、登录等按 IP 的 CT 桶。
- 默认每用户每类查询 20 次 / 1200 秒，可通过
  `TOKEN_QUERY_RATE_LIMIT_ENABLE`、`TOKEN_QUERY_RATE_LIMIT`、
  `TOKEN_QUERY_RATE_LIMIT_DURATION` 调整。Redis 模式多节点共享用户计数；内存模式
  仍按进程隔离。超限继续返回 HTTP 429 与 `Retry-After`。
- 全局 API 限流 GA、认证、计费、响应格式和支付接口保持原样；无需数据库迁移。
- 两套钱包页面不调用这两个只读令牌端点。Default 与 Classic 的钱包及订阅支付请求
  均抑制 Axios 全局错误提示，由支付界面单独处理；同一次 429 只展示一次，并提示按
  `Retry-After` 等待。
- 两套模板的历史订单重新支付采用相同处理；不自动重试付款。Default 同时支持 AxiosHeaders
  的大小写无关读取，缺少有效等待秒数时退回“稍后重试”。

## 兼容与安全边界

- 原请求和成功响应字段不变，登录态、API Key 与计费语义不变。查询从 IP 计数改为用户计数，
  同用户所有 Key 和出口合计受限，不同用户共享出口时独立；余额与日志各有独立额度。
- 默认仍保留每类查询 20 次/20 分钟的保守频率。原来只配置 `CRITICAL_RATE_LIMIT*` 的站点，
  升级后应按查询需要单独配置 `TOKEN_QUERY_RATE_LIMIT*`；关闭 CT 不会关闭查询保护。
- 无效或禁用凭证在查询桶之前拒绝，仍受 GA 防护；禁止用户不可查询，过期或额度耗尽的令牌
  保留原有只读查询能力。Redis 故障维持失败关闭，内存模式不提供跨节点合计。
- 无数据库迁移，无需清除旧 Redis 桶。回滚会恢复查询误伤充值的旧行为；本修复不改变其他
  关键操作共用 CT 的现有策略，因此其他关键操作仍可能触发充值限流。

## 验证计划

- 通过真实 API 路由验证 Redis/内存下查询耗尽后支付仍进入控制器参数校验、反向 CT
  耗尽不阻塞查询；同用户多 Key/跨 IP 共享查询限额、同 IP 不同用户隔离，以及独立作用域。
- 覆盖缺失/无效/禁用令牌、禁用用户、耗尽/过期令牌只读语义、GA、开关和 Redis 窗口恢复。
- 运行 `go test ./middleware ./router ./common -count=1 -timeout 60s`、Default
  lint/typecheck/test/build、Classic lint/test/build、Markdown 检查及 `git diff --check`。
- 不调用生产支付接口，不部署生产服务。生产修复在发布并部署后生效。

## 验证结果

- `go test ./middleware ./router ./common -count=1 -timeout 60s` 通过，719 项测试通过；
  相关包 `go vet` 和根模块 `go build ./...` 通过。
- Default 受影响文件 Oxlint、TypeScript 类型检查、钱包支付错误与真实请求测试（13 项）及 Rsbuild
  构建通过。
- Classic 支付错误与历史订单重试测试（4 项）、ESLint、Prettier 与 Vite 构建通过；无 TypeScript 检查脚本。
- 构建出现既有 Browserslist 数据、`lottie-web` eval 和大分块警告，不影响构建结果。
- `git diff --check` 通过；未部署生产。
