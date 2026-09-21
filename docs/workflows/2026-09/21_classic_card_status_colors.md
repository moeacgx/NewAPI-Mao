# Classic 模型卡片状态柱配色对齐官方

日期：2026-09-21

交付分支：`fix/classic-card-status-colors`；目标：`custom-main`。提交前同步 `336544aa7`，重新运行相关测试与构建。

## 问题与依据

截图中的“吞吐量”和三根信号柱对应 Classic 模板。此前的详情页配色修复没有覆盖卡片；卡片仍通过 `availabilityTone` 使用 95% / 0% 三档规则，且无历史序列时改用 Semi 主题色。

本次核对官方 `QuantumNous/new-api@9c293e8c02371bda844af79e3500ff2d516d1dda` 的 [model-perf-badge.tsx](https://github.com/QuantumNous/new-api/blob/9c293e8c02371bda844af79e3500ff2d516d1dda/web/src/features/pricing/components/model-perf-badge.tsx) 和 [format.ts](https://github.com/QuantumNous/new-api/blob/9c293e8c02371bda844af79e3500ff2d516d1dda/web/src/features/performance-metrics/lib/format.ts)。官方卡片使用统一成功率色阶：100% 深绿 `#10b981`、90% 至不足 100% 浅绿 `#34d399`、70% 至不足 90% 黄色 `#f59e0b`、不足 70% 红色 `#ef4444`。

原卡片绿色已经是深绿色值，缺少的是深浅两档区分。浏览器进一步确认，历史柱命中了性能文字的通用样式而被设为 `opacity: 0.8`，摘要回退柱则为 `1`；这会淡化深绿并使同一数值呈色不同。官方当前卡片已改为 24 根小时柱，本次仅对齐色阶。

## 范围、契约与兼容性

- 仅修改截图对应的 `web/classic`，Default 模板不在本次范围内。
- 卡片历史柱、摘要回退柱和状态标签复用现有 `getSuccessRateHex`，移除不再使用的旧可用性颜色函数。
- 通用性能文字样式排除状态信号组件，历史柱保持完整不透明度，延迟与吞吐文字仍沿用原样式。
- 保留 `status_rate` 优先、缺失时回退 `success_rate`：后端 `bestGroupStatusRate` 取有请求分组中的最高成功率，并非官方请求加权成功率。
- 保留最近 24 小时聚合为三个 8 小时段、段内等权平均、空段灰色及 8/10/12 像素递增柱高。阈值只影响颜色，不修改后端统计或渠道健康判断。
- 保留价格、折扣、延迟、吞吐量和详情趋势。无接口、权限、配置、数据生命周期或数据库迁移变更；回滚恢复前端版本即可。共享契约影响：无。

## 验证计划

- 用真实卡片组件覆盖深绿与浅绿边界、黄红边界、摘要回退、空时间段和 `status_rate` 优先语义。
- 运行相关既有测试、修改文件 ESLint 与 Prettier、文档链接及 `git diff --check`，完成 Classic 生产构建。
- 使用本地模拟数据核对柱子实际呈色；不将本地验证视为线上发布。

## 验证结果

- 真实卡片组件的 17 项 Vitest 测试通过；旧实现先出现 14 项预期失败，修复后全部通过。相关 Node 回归 13 项通过。测试进程设置 60 秒超时。
- ESLint、修改的 JS/JSX/MJS 与 Markdown 的 Prettier、开发文档链接和 `git diff --check` 通过。修改的 CSS 规则格式检查通过；全量 `index.css` 在当前本地 Prettier 下存在四处既有格式差异，已对 `HEAD` 复现并保留原样，未混入本次变更。
- Classic Vite 生产构建通过；本机 Bun 不在 PATH，使用 Node 调用工作区既有依赖，未修改依赖清单或锁文件。Classic 无独立 typecheck 脚本，本次未修改 TS/TSX。
- Playwright 对真实构建页面注入 8 张模拟卡片，在 1440×1080 与 390×844 视口验证四档背景色、摘要回退、空段灰色、三根柱高 8/10/12 像素及不透明度 `1`；历史柱 `0.8` 的问题先在浏览器中复现，修复后的检查通过。
- 浏览器数据与截图位于本工作区忽略目录 `.local-tests/browser-colors.json`、`cards-desktop.png`、`cards-mobile.png`。独立只读审查通过；未部署。
- 构建保留既有 Browserslist 数据陈旧、Lottie `eval` 和较大 chunk 提示。
