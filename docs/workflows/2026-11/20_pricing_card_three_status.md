# 模型广场卡片状态恢复三格信号

## 问题

`26b8fac81` 把模型广场卡片的状态从三格信号改成了 24 根逐小时长条。用户要求卡片恢复三格，计费、延迟、吞吐和状态仍放在同一行；详情页性能区保持现状。

## 修改范围

- Default：`web/src/features/pricing/components/model-perf-badge.tsx` 改回共享组件 `StatusSegments`（`segmentCount=3`、`shape='signal'`、`tone='availability'`）。网格改为 `auto auto auto` 并靠右对齐，卡片页脚 `flex` 同行布局不变。
- Classic：`ModelPerformanceBadge.jsx` 改回 `maxPoints={3}` + `signalStyle`，并移除 `compact`；`index.css` 删除拉伸 24 根长条的规则，状态列改为 `auto`。
- 不改共享详情代码：Default 的 `model-details-*`、`status-segments` 库，以及 Classic 的 `SuccessRateSparkline`、`utils.buildStatusSegments`、`ModelPerformancePanel` 都没有改动。

## 语义

- 时间窗仍是最近 24 小时，平均分为 3 个 8 小时时段，每格取时段内样本可用率的平均值。
- 没有样本的时段显示空格，使用中性灰（Default 为 `bg-muted-foreground/15`，Classic 为 `--semi-color-fill-1`），不会被判为成功或失败。
- 颜色沿用可用率分级：≥95% 为成功，>0% 为降级，0% 为故障。
- 只有聚合状态率时：Default 用聚合率生成三格；Classic 保留原有的三格语义色回退。
- 没有性能数据时不渲染性能区。

## 验证

- Default：`bunx vitest run src/features/pricing/components/__tests__/model-perf-badge.test.tsx`，8/8 通过。
- Classic：`node --test .../ModelPerformanceBadge.test.mjs .../performance/utils.test.mjs`，13/13 通过。
- 用本机 Chrome 加上合成 API 数据，截取了双模板 1440 桌面端和 390 手机端截图。每张卡都是 3 格信号，页脚只有一行，没有横向滚动，也没有页面错误。
