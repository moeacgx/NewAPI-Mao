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
