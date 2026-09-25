# 福利券提前结束、有效期快照与随机拆分修复

## 问题

线上反馈包含三种现象：旧的 24 小时券在发布 168 小时活动后仍显示 24 小时；随机面额经常等于保底；结束旧活动并发布同组新活动后，部分请求没有扣福利券而使用钱包。

## 根因与契约

- `BenefitUserVoucher.expires_at` 是领取时快照，按 `min(claimed_at + personal_valid_seconds, ends_at)` 计算。后续活动不会修改历史券；领取新活动才会得到新活动的有效期。
- 提前结束活动时，原实现只结束活动和未领取份额；券虽然仍是 `active`，资金源查询却排除了 `ended` 活动，于是页面状态和实际扣费路径不一致。
- 随机拆分按顺序从剩余预算抽取，余数逐步变小，导致后续份额明显偏向保底值。

## 修改范围

- 活动结束只关闭领取入口，已领取券继续使用到自身 `expires_at`；同组多个活动的可用券会合并跨券预扣、结算和退款。
- 金额模式和 quota 模式共用受限均值随机分配，仍保证每份处于 `[min, max]`、总额严格相等，并在发布时固化结果。
- 补充随机分布和提前结束券状态的模型回归测试。

## 兼容性与边界

- 不修改已领取券的历史有效期，不自动把 24 小时券延长为后续活动的 168 小时。
- 活动自然结束和提前结束都只停止领取；已领取券仍由自身 `expires_at` 控制可用期。
- 数据库结构、API 路径和计费组合顺序不变，SQLite、MySQL、PostgreSQL 均使用现有 GORM 操作。

## 验证

- `go test ./model`
- `gofmt -w model/benefit_voucher.go model/benefit_voucher_test.go`
- `git diff --check`

## 线上盘点（只读、局部已确认）

2026-09-25 通过 CloudSSH 对 `serverId=38` 的 MaoLaoAPI PostgreSQL 做只读盘点，作业
`7d387b35-ffde-48d5-9b38-ad9d9129d47b`（`2026-09-25T07:28:10Z`）确认：`ended` 活动未删除、
仍为 `active` 且 `remaining_quota > 0`、`expires_at` 未来的已领取券为 48 张；`ended` 且已真实过期
的券为 79 张，已耗尽 21 张；`terminated` 且已过期 2 张。负数余额、`remaining_quota + used_quota >
original_quota` 和孤儿券均为 0。历史查询中 `expires_at` 未来但状态误为 `expired` 的券为 0；另有
81 张真实已过期但仍保留未用余额，不能复活或延长。本次不执行数据库改写，用户授权目标仅是恢复
48 张仍有效券的抵扣能力。

同日只读审计作业 `bd2f77e8-2012-4469-b478-186d61b8e2aa` 聚合出 15,624 个非空
`request_id`：19 个存在 `pre_consume` 但无 `settle_delta`/`refund` 且最后流水早于 1 小时；
`settled + refunded` 但无 rollback、缺少 pre 的终态流水、同请求多券均为 0。19 个历史未闭合请求
尚待关联请求日志确认，不能据此宣称全部历史账务无异常，也不自动退款。

流水类型计数为：`pre_consume=15617`、`refund=299`、`expire=81`、`settle_delta=15299`、
`settle_rollback=3`、`refund_additional=0`。实现必须兼容这些历史单券流水，不把现场统计推断为
线上已部署或历史账务完全正常。

## 第二轮账务闭合

- `preConsumedQuota=0` 的真实福利券会话在正差额结算时，先按差额追加目标预扣，再以
  `Settle(0)` 建立完整的 `pre_consume`/`settle_delta` 账务；组合结算失败时沿原结算补偿路径回滚。
- 同一请求已有 `settle_delta` 或 `settle_rollback` 后，Reserve 只接受与已有目标总预扣严格相等的
  只读重放；任何更大目标都明确拒绝，正常结算前的 Reserve 追加不受影响。
- 多券 breakdown 以请求流水计算每券 `pre_reserved + sum(-settle_delta)`，只保留正数分配；单券继续
  输出兼容的 activity/voucher 字段，多券将其置零并输出完整 `voucher_allocations`。流水查询失败不得
  静默伪造逐券归属。
