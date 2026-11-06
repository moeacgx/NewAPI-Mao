# Classic 支付限流提示修复

## 目标

Classic 钱包充值或订阅支付被服务端限流时，只显示一条准确的等待提示，并使用
响应 `Retry-After` 告知剩余秒数。普通支付错误继续显示原有通用提示。

## 范围与方案

- Classic 充值的 Stripe、Epay、Bepusdt、OKPay、Creem、Waffo 与 Waffo Pancake
  支付请求，以及 Classic 订阅的 Stripe、Creem、Epay、Bepusdt、OKPay 和余额支付。
- 历史账单弹窗的重新支付同样抑制全局错误提示并显示局部等待提示。Classic 没有
  Waffo Pancake 订阅创建入口，本次不新增支付方式；Default 的已有对应入口同步修复。
- 支付请求通过 Axios 已有 `skipErrorHandler` 配置抑制全局 Toast；各自 catch
  统一调用局部支付错误处理，根据 HTTP 429 与 `Retry-After` 展示等待提示，其他
  异常仍展示支付失败。
- `Retry-After` 缺失或无效时使用“请稍后再试”，不改变请求重试流程。
- 不改变后端限流策略、接口请求字段、支付权限或支付订单生命周期。
- 新提示同步维护 Classic 全部八种语言包。

## 审查补充：会话失效与跨域等待时间

- `skipErrorHandler` 仅抑制通用错误提示，不能跳过认证失效处理。Classic 在拦截器中
  先处理明确的 401 或 `409 AUTH_SESSION_MISMATCH`，清除旧登录态并跳转登录页，
  再决定是否显示通用提示；`showError` 复用相同认证处理，保留直接调用的既有行为。
- 刷新返回 429、5xx、网络错误、普通 409 或 `AUTH_REFRESH_RACE` 时保留用户状态，
  调用方继续收到实际刷新错误；成功刷新仍仅重放一次原请求。
- 共享 API CORS 显式暴露 `Retry-After`，允许受信任跨域部署的两套模板读取等待秒数；
  不扩大允许来源，不更改凭据规则、支付限流或自动重试策略。
- 回归使用真实 Axios 拦截器覆盖充值、订阅、订单重试的静默错误配置，并通过 Gin
  验证可信跨域 429 响应的暴露头。Default 既有认证分支独立于提示抑制，复验其支付与认证测试。

## 验证结果

- Classic 源码中的中文键为“请求过于频繁，请在 {{seconds}} 秒后重试”及
  “请求过于频繁，请稍后再试”，已补齐英文、简体中文、繁体中文、法语、俄语、
  日语、越南语和 zh-CN 语言包。
- `topupError.compat.test.jsx` 和 `retry-payment.compat.test.jsx` 共 4 个用例通过，
  后者渲染实际弹窗并使用真实 Axios 拦截器验证局部提示一次、全局提示零次；修改文件的 Prettier 检查与 ESLint
  检查通过；Classic Vite 生产构建通过。
- Classic 为独立 JavaScript/Vite 应用，没有 TypeScript 类型检查脚本。
- 构建报告既有 Browserslist 数据过期、`lottie-web` 使用 eval 和大 chunk 提示，
  均未阻断构建。本轮未部署。
- Default 对应修复与双模板共享契约记在
  [令牌查询与充值限流隔离](06_token_query_payment_rate_limit.md)。

### 审查修复验收

- 新增会话回归先复现 6 项失败：充值、订阅和订单重试请求抑制通用提示后，刷新
  401 或会话不匹配 409 都未清理登录态。修复后相关 4 个文件共 44 项测试通过，
  包含 23 项会话刷新、17 项缓存兼容和 4 项支付提示测试。
- 验证了静默支付的成功刷新仅重放一次且保留付款参数；直接会话不匹配会重新登录；
  刷新限流、服务故障、网络失败和非身份冲突均不误退登。支付 429 仍只提示一次。
- CORS 回归先复现缺少暴露头，修复后 6 个顶层 CORS 测试通过，新增场景确认可信来源
  的 429 响应暴露等待秒数，未授权来源仍返回 403 且不进入支付处理器。
- Default 无需改变认证实现：其会话清理及跳转本已独立于 `skipErrorHandler`；
  支付与认证相关 4 个文件共 23 项测试复验通过，共享 CORS 修复同时覆盖两套模板。
- Classic 修改文件的 ESLint、Prettier 与生产构建通过；Markdown 格式化及
  `git diff --check` 通过。未改依赖、版本号，未提交或部署本轮修复。
- 扩展检查的 `oauth-binding.compat.test.jsx` 存在 11 项本地存储断言失败；
  将 `api.js`、`utils.jsx` 在测试加载阶段替换为 HEAD 原文后，同样 11 项失败，
  已确认非本轮改动引入。本轮未扩大到修改这组既有测试或 OAuth 流程。
