# 模型广场详情「按分组定价」显示分组倍率

## 目标

模型广场 → 模型详情 → 「按分组定价」中，每个分组单独显示配置的分组倍率 `{ratio}x`，参考 Classic 已有样式。只改展示，不改计费、接口、权限，也不提交、推送或部署。

## 范围与现状

- Default（`web/src`）：`web/src/features/pricing/components/model-details.tsx` 的 `GroupPricingSection` 修改前只在分组名旁显示折扣徽标 `BillingAdjustmentBadge`，没有倍率。
- Classic（`web/classic`）：`ModelPricingTable.jsx` 已在动态表达式卡片标题右端（`.classic-pricing-detail-group-card-ratio`）显示 `{row.ratio}x`，常规定价表格的「分组」列后也有独立「倍率」列。行为符合目标，**不改 Classic 生产代码**，只补回归测试。

## 方案（Default）

- 倍率取值沿用实际定价使用的 `getConfiguredGroupRatio(groupRatio, group)`：配置为 0 显示 `0x`，为 1 显示 `1x`，缺失回退 `1x`。只显示分组倍率，不乘充值比例或汇率。
- 可解析表达式（分档卡片）：卡片标题改为 `justify-between`，分组名与折扣徽标保持在左侧，右侧增加独立 `{ratio}x`。
- 常规定价（按 Token、按次、按秒共用同一张表）：在「Group」列后新增「Multiplier」列（复用已有 i18n 键 `Multiplier`，七语均已有翻译），右对齐，对齐 Classic 的倍率列。
- 倍率文本使用 `shrink-0 whitespace-nowrap tabular-nums font-mono`，长分组名由左侧 `min-w-0` 截断，数字不换行、不被挤压。
- 折扣徽标、分组显示名（`getGroupDisplayName`）、价格计算均未改动。
- 不可解析表达式仍走原「Special billing expression」提示分支，不生成分组卡片和倍率，不伪造定价。

## 测试

分组显示名与 code 不同（`vip_code` → `VIP 显示名` 等），倍率覆盖 2.4、0、1 及缺失回退 1。

- Default：新增 `web/src/features/pricing/components/__tests__/model-details-group-ratio.test.tsx`，真实渲染 `ModelDetailsContent`，共 5 例：按 Token / 按次倍率列；充值比例 0.5 时折扣徽标（`5 fold`）与倍率列独立，价格 `$2` / `$4.8` 不变；分档卡片标题右侧倍率不在表格内且不换行；不可解析表达式保留提示且无倍率。
  - 先红：生产代码未改时 4 例失败、1 例（不可解析表达式保留提示，属于保护性用例）通过。
  - 后绿：5/5 通过。
- Classic：新增 `web/classic/src/components/table/model-pricing/modal/components/__tests__/model-pricing-table-group-ratio.compat.test.jsx`，真实渲染 `ModelPricingTable`，共 4 例，覆盖同样的倍率、折扣独立性、卡片与不可解析表达式。Classic 生产代码未改，因此没有红阶段；用临时变更（把 `{row.ratio}x` 改为 `{row.ratio}`）做反向验证，4 例全部失败，恢复后 4/4 通过。

## 验证结果

| 检查                | 命令（各自目录）                                                                                                                                             | 结果             |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------- |
| Default 定价测试    | `web/`：`NODE_OPTIONS=--no-experimental-webstorage bunx vitest run src/features/pricing`                                                                     | 9 文件 49 例通过 |
| Default 类型检查    | `web/`：`bun run typecheck`                                                                                                                                  | 通过             |
| Default lint        | `web/`：`bunx oxlint -c .oxlintrc.json <改动文件>`                                                                                                           | 0 问题           |
| Default 格式        | `web/`：`node scripts/format-with-protected-headers.mjs --check <改动文件>`                                                                                  | 通过             |
| Classic compat 测试 | `web/classic/`：`NODE_OPTIONS=--no-experimental-webstorage node scripts/run-compat-tests.mjs src/components/table/model-pricing/modal/components/__tests__/` | 2 文件 6 例通过  |
| Classic 格式        | `web/classic/`：`prettier --check <新测试> ModelPricingTable.jsx`                                                                                            | 通过             |

环境说明：本机 Node v26.13.0 内置 Web Storage 会遮蔽 jsdom 的 `localStorage`。不加 `--no-experimental-webstorage` 时，Default `price-currency.test.ts` 的 3 例在未改动基线上同样失败，Classic compat 测试也全部失败，与本次改动无关。所有测试均在 60 秒内完成。

## 未验证

- 未启动后端和前端开发服务器，未在真实浏览器检查桌面和手机宽度的渲染，长分组名与窄屏布局只靠类名和测试断言（`whitespace-nowrap`、`shrink-0`）保证。
- 未运行 Default / Classic 全量生产构建，也未运行全量测试。

## 兼容性

纯前端展示变更，不影响接口、计费、权限和数据。
