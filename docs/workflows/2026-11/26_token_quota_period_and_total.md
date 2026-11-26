# 令牌周期额度保存与总额度上限修复

## 目标

每周 500 USD 的周期额度与 10,000 USD 的总额度组合可保存、回显并实际限制用量；界面区分「周期额度（每日/每周，重置后恢复）」与「总额度（不随周期重置）」，总额度可独立设为不限而不关闭周期额度。

## 反馈与复现

- 周期额度 500 USD、预期每周，下拉显示 daily；总额度 10000 USD 保存报「额度值超出有效范围，最大值为 2147483647」。
- 补充：总额度不限、选每日 500 USD，提示更新成功，列表只显示无限制，重新编辑后周期为 none、额度 0。

## 根因

1. 周期额度只存在于 Default 前端与文档（`docs/workflows/2026-08/21_token_period_quota.md`），Go 后端的 `model.Token` 没有 `quota_period*` 字段，创建/更新请求里的周期字段被静默丢弃：接口返回成功、读取接口缺字段、编辑表单回填默认值 none/0。
2. 总额度上限由控制器 `common.QuotaFromFloat(1000000000 * QuotaPerUnit)` 计算，该函数按单次请求计费的 int32 边界饱和，因此上限被压成 2147483647（约 4294 USD），10000 USD（5,000,000,000 内部额度）被拒。
3. Default 周期下拉的 `Select` 没有 `items` 标签映射，触发器显示内部值 `daily`/`weekly`。该项以源码判断为主，未在旧代码上做失败复现。

## 方案

- 令牌新增 `quota_period`、`quota_period_limit`、`quota_period_used`、`quota_period_window` 四列（AutoMigrate 新增列，默认 `none`/0，旧令牌不受影响，无需手工迁移）；`quota_period_used`/`window` 为服务端账本，客户端不可写；读取响应附带 `quota_period_reset_at`。
- 周期窗口按服务器时区的自然日或周一起始的自然周计算，惰性重置：窗口变化后首次预扣从 0 开始。
- 预扣时先对周期账本做条件 `UPDATE`（检查并累加原子完成），再预扣总额度；总额度失败时回滚周期预扣。无限总额度令牌同样受周期限制，带周期的令牌不走信任旁路。
- 结算差额、失败退款、异步任务、Midjourney 与图片任务退款同步调整周期账本，退款不低于 0。
- Redis 令牌缓存哈希加入周期模式与上限；周期账本只在数据库中维护，不依赖缓存。
- 总额度与周期额度的上限改由 `common.MaxTokenQuota()` 提供（1e9 个货币单位 × `QuotaPerUnit`，不超过平台 int），不再使用 int32 单次计费边界；`MaxQuota`、`QuotaFromFloat` 及全局计费换算不变。超限、负数、周期额度非正数、未知周期由后端返回错误，不截断。
- 更新请求省略 `quota_period` 时保持原周期（兼容 Classic 与旧客户端）；选择 `none` 清空周期额度并重置账本。
- Default：周期下拉增加标签映射；表单拆成「周期额度」与「总额度」两节，总额度开关改为「总额度不限」且不影响周期额度；表单校验在显示单位下给出上限提示；列表与手机卡片在总额度不限和有限时都显示周期额度与已用。
- Classic：`EditTokenModal` 新增独立「周期额度」卡片（额度周期下拉 + 周期额度金额输入）与「总额度」卡片，总额度开关不再关闭周期额度。提交时把周期金额换算为 `quota_period_limit`，`none` 时上限为 0；选择周期但金额不大于 0 时前端拦截。加载时把 `quota_period_limit` 回显为金额，并剔除只读账本字段 `quota_period_used`、`quota_period_reset_at`，不回传。列表额度列在总额度不限和有限时都追加周期额度已用/上限。

## 双模板评估

- Default：见上，已改并有测试。
- Classic：新增周期额度界面，见上。提交始终携带 `quota_period`，不再依赖后端「省略时保留原周期」的兼容分支；该兼容分支仍保留给旧客户端。Classic 词条已补齐 en/zh/zh-CN/zh-TW/fr/ja/ru/vi。

## 数据库兼容与回滚

新增列类型为 `varchar(16)` 与 `bigint`，SQLite、MySQL、PostgreSQL 通过 AutoMigrate 添加；账本更新使用 `CASE WHEN` 条件表达式，不使用方言特有函数。`remain_quota` 在 64 位构建下由 GORM 映射为 BIGINT；已有生产库列实际类型未核验。回滚旧版本后新增列保留，周期限制不再生效。

## 测试

- `model`：窗口边界（周一起始）、期间规范化、条件预扣在上限处停止、退款不为负、新窗口重置、无限总额度仍受限、无周期令牌不受限。
- `controller`：每周 500 + 总额度 10000 USD 创建与读取、每日/每周切换重开回显、总额度不限保留周期、列表读取、省略字段兼容、边界与非法输入、客户端不能写账本。
- `service`：真实预扣/结算路径中周期上限阻断（含无限总额度）、总额度失败回滚周期、结算与退款调整账本、无周期令牌保持旧行为。
- Default：表单载荷/回填/校验、抽屉标签与回填及提交载荷、列表周期行。
- Classic：`token-period-quota.compat.test.jsx` 覆盖每周 500 + 总额度 10000 回显与提交、总额度不限保留周期、未设周期提交 `none`/0、周期额度 0 拒绝提交、列表周期行。
- CI：PR #432 的 Frontend typecheck and test 失败于 `overview-dashboard.test.tsx` 第 205 行找不到 `Quick actions` 导航。该测试在 key 查询仍异步时同步断言导航，查询未返回时导航尚未渲染。改为 `findByRole` 等待导航出现后再断言引导消失，断言内容不变。
- 本机 Go 工具链为 windows/386，测试需 `GOARCH=amd64`；Node 26 下前端测试需 `--localstorage-file`，均放在项目 `.tmp/`。

## 未覆盖

真实 MySQL/PostgreSQL 迁移、Redis 多实例并发、真实浏览器渲染、构建与部署；未使用任何生产令牌、站点或数据。
