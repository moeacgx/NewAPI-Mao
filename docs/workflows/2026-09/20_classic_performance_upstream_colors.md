# Classic 模型广场成功率配色对齐官方

日期：2026-09-20

## 问题与目标

Classic 模型详情中的成功率数字在达到 90% 时显示绿色，但成功率柱需要达到 99% 才显示绿色，低于 90% 即为红色。同一数值在数字、柱和趋势点之间产生不同的视觉判断。

本次对齐官方 `QuantumNous/new-api` 的固定版本 `9a0be8750a6d736d9692535ed2cd68f8eec46529`，规则来源为 [performance-metrics/lib/format.ts](https://github.com/QuantumNous/new-api/blob/9a0be8750a6d736d9692535ed2cd68f8eec46529/web/src/features/performance-metrics/lib/format.ts)。

## 范围与方案

- 仅修改 `web/classic` 模型广场成功率展示；Default 的 `web/src/features/performance-metrics/lib/format.ts` 已采用官方阈值，不需修改。
- 成功率柱和可用率趋势点使用官方色值：100% 为 `#10b981`，90% 至不足 100% 为 `#34d399`，70% 至不足 90% 为 `#f59e0b`，不足 70% 为 `#ef4444`；非有限数值使用灰色 `#9ca3af`。
- 数字继续使用 Classic 的 Semi 主题语义色，健康 / 提醒 / 异常的边界统一为 90% / 70%。
- 保留柱高的延迟含义、时间桶选择、分组等权汇总、异常桶计数和实际成功率数值。
- 卡片的四段可用性状态仍使用独立的 `status_rate`、最佳分组和 95% 可用阈值，不将这次详情配色变更扩展到卡片状态规则。

## 接口、安全与兼容性

`GET /api/perf-metrics` 和 `GET /api/perf-metrics/summary` 的路径、参数、响应与权限不变。没有后端采样、失败过滤、计费、数据库、配置或数据生命周期变更，不需要迁移；回滚仅需恢复对应前端版本。

## 验证计划

- 更新既有 `performance/utils.test.mjs`，覆盖 100%、90%、70% 及其相邻边界，以及无效数值的颜色规则。
- 运行模型详情视觉契约与卡片状态回归，确认卡片可用性和分组汇总保持原有行为。
- 对修改文件执行 ESLint、Prettier、文档链接检查及 `git diff --check`，完成 Classic 构建。
- 使用本地模拟数据检查性能页面，明确区分本地验证和生产发布。

## 验证结果

- `node --test --test-timeout=60000` 执行 `performance/utils.test.mjs`、`model-pricing-visual-contract.test.mjs` 和 `view/card/ModelPerformanceBadge.test.mjs`，14 项通过；旧实现先在 99.99% 的官方配色断言上失败，修改后通过。
- 修改的 `utils.js` 通过 ESLint，代码、测试、工作记录和开发文档索引通过 Prettier；本地文档链接及 `git diff --check` 通过。
- Classic Vite 生产构建通过。本机 Bun 不在 PATH，使用 Node 调用现有 Vite；首次复用旧依赖时缺少 `@fontsource-variable/public-sans`，改用本机同项目完整依赖目录后构建成功，依赖清单和锁文件未修改。
- Playwright 在 1440 × 1080 与 390 × 844 视口检查真实 Classic 页面，使用本地模拟接口验证 100%、99.99%、98.6%、90%、89.99%、70%、69.99%、0% 的实际柱颜色，并确认每行仍有 24 个历史点；移动端保留表格横向滚动。
- 截图和模拟数据保留在工作区被忽略的 `.local-tests/` 下，未请求生产数据、未部署。构建仍提示既有 Browserslist 数据陈旧、Lottie 的 `eval` 与大体积 chunk；开发页面存在 Semi 的 `findDOMNode` 弃用警告。
- 共享契约影响：无。模型、认证、usage、quota、价格与跨项目接口均未修改。
