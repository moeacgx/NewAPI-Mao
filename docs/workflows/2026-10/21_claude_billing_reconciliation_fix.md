# Claude 流式计费对账修复

首次修复：2026-10-21；断流结算与限时读取补全：2026-10-22。

## 问题

952 渠道对账发现两类相反差异：上游返回错误或空流时，猫佬可能用本地估算输入写入消费；上游已经返回可计费用量的重试或断流请求，猫佬又可能因为客户端方向取消而没有完成结算。

## 修改范围

- Claude Messages 流尾只有在收到实际输出证据时，才允许用本地文本计数补齐缺失的输入或输出。
- 没有输出、没有缓存用量、没有上游 usage 的错误流或空 EOF 保持零用量，保留重试或退款路径。
- Claude 缓存读取和缓存写入用量即使普通输入和输出为零，也继续进入计费摘要。
- 流尾重建 `BillingUsage`，确保补齐后的输出和缓存字段与最终结算快照一致。
- `/v1/responses` 经 Claude Messages 上游转换时，在下游写入取消、扫描期间取消和最终事件写出取消三个出口保留上游已报告的 usage，让 `ResponsesHelper` 继续进入统一结算。
- 取消分支使用本地估算前的 usage 快照，深拷贝 `BillingUsage`；有文本但上游无 usage 的取消流不会补入预估输入或输出。
- 客户端取消不记录为上游流错误；上游读错误、超时等已经发生时，后续客户端取消保留原 `EndReason/EndError`，不掩盖真实渠道故障。
- 仅 Claude Messages 转 Responses 的成功流，客户端断开后继续限时读取上游 usage；停止下游写入，也不再累积文本。收到 `message_stop` 就关闭上游，不等待额外 EOF。
- 读取窗口上限为断开后 180 秒；已有入口 deadline、HTTP 客户端超时和流空闲超时仍有效，以先到者为准。新帧不会延长固定窗口。
- 上游响应头返回前取消、非 200 响应及其他协议的取消行为仍及时终止；关闭响应体会释放独立上下文和定时器。

## 计费边界

本地估算只代表“已观察到输出但上游没有完整 usage”的补计数，不代表上游账单。本次 Claude Responses 转换链路中，客户端取消且没有上游已报告 usage 时不估算 token 费用，预扣仍交外层退款；上游已报告的普通输入、输出、缓存读写按现有费率结算，已有工具附加费用契约不变。上下文长度错误本身不生成本地估算收费，后续换渠道成功请求由成功尝试单独结算。

本次为共享扫描器新增显式启用的限时排空入口，默认扫描入口仍在取消时停止。HTTP context 仅对上述 Claude Responses 成功流脱离客户端取消，并保留原入口截止时间。没有改变自动重试策略或计费表达式。真正的上游 API/解析/转换错误仍返回错误，不转换为成功。

客户端断开后，上游可能在窗口内继续生成并产生费用；网关只按实际收到的 usage 结算，既可能包含断开前用量，也可能包含窗口内新增用量。窗口结束仍没有 usage 时退款；只有部分 usage 时只结算已收到的字段，不凭输入长度补足缺失部分。

## 根因与修复出口

`ClaudeMessagesToResponsesStreamHandler` 已解析 usage 后，最终 `ResponseChunkData` 仍可能返回 `context canceled`。旧代码返回 `nil, error`，`ResponsesHelper` 因此在结算前退出。只在最终写入前检查一次取消也不够：取消可能发生于检查之后。

现在客户端写取消后先限时收集上游终态，再返回已观察的上游 usage；最终事件写出取消则返回写入前冻结的快照，由 `PostTextConsumeQuota` 使用原有 `BillingSession` 结算一次。无有效用量继续返回内部取消错误，由外层退款；不将本地估算快照混入该结算出口。先于客户端取消发生的上游读取失败继续保留，用量与故障分类分开处理。

## 官方实现参考

- NewAPI 官方 `QuantumNous/new-api`，核验提交 `7d8b91268dab69d7150f345c448b5e250163146a`：参考协议与用量转换，没有照搬空 EOF 估算输入或写取消丢弃 usage 的行为。
- Sub2API 官方 `Wei-Shaw/sub2api`，核验提交 `7da09cab4e875463baa79fc150adde700c5438df`：参考实际 usage 才部分结算、客户端断开后有界读取与独立账务上下文；本补丁只覆盖 Claude Responses 路径，不等同于全部实现。

## 验证

- Claude 空失败流不会设置 `local_count_tokens`，输入和输出保持零。
- Claude 缓存-only usage 仍被识别为可计费用量。
- `TestClaudeResponsesDisconnectReturnsOnlyObservedUsage`：首帧、文本分片、最终事件写入取消，覆盖已有输入/输出/缓存、无 usage 文本及本地估算隔离。补测前复现终态丢 usage 和错误归因问题，修复后通过。
- `TestClaudeResponsesUpstreamErrorsRemainErrors`：限流和损坏事件保留真实上游错误。
- `TestClaudeResponsesCancellationAfterCompletionKeepsCompletedUsage`：正常完成后取消不倒推为传输失败。
- `TestClaudeResponsesUpstreamReadFailureSurvivesClientCancellation`：读取失败后客户端取消，保留原上游读取错误。
- `TestCancelledClaudeUsageSettlesWalletOrRefundsReservation`：SQLite 实际钱包预扣、精确金额结算、无用量异步退款和会话幂等，逐项核对余额与单条消费/错误日志。

```powershell
go test ./relay/channel ./relay/channel/claude ./service ./relay/helper ./relay -count=1 -timeout=60s
go vet ./relay/channel ./relay/channel/claude ./relay/helper ./service
git diff --check
```

新增真实 HTTP 本地模拟上游测试，贯通 Claude Adaptor、HTTP context、共享扫描器及 Responses 桥接；客户端断开后返回终态 usage，验证普通输入、输出与缓存字段并主动关闭连接。另覆盖响应头前取消、HTTP 错误、入口 deadline、固定排空上限及断开后真正 SSE 错误。桥接层与 SQLite 钱包结算测试分别覆盖协议和账务边界，没有调用真实收费上游。本地验证不等于线上账务验证。

## 已知限制

供应商最终账单仍需以供应商正式账单或逐条 usage 为准；猫佬的失败事件只能证明本地请求生命周期，不能单独证明上游是否收费。

如果限时读取结束仍未收到 usage，或上游以真正 API/解析/转换错误结束，本补丁不会凭供应商消费日志或请求输入长度补扣用户，也不保证追回所有上游成本。历史四笔有上游消费的请求，现有证据不能证明网关当时已收到 usage，不能据此声称已全部解决。

本次只修改后端，无数据库迁移、历史账务补扣或退款。Default 与 Classic 前端均未改动；API 字段、鉴权、模型映射及费率保持不变。尚未发版或部署。当前 Windows Go 环境为 `386`、`CGO_ENABLED=0`，未执行 race 检查；未进行 MySQL/PostgreSQL 实库验证。

## PR 整合基线

2026-10-22 基于 `origin/custom-main` 的 `2088c0216` 整合。该基线已合入 PR #307 的显式零用量、缓存计费及计费快照修复，本次保留这些实现与回归测试，只补充空流估算边界和 Claude Responses 取消后的真实用量收集及结算。保留基线新增的独立工具请求计费与 HTTP 请求体元数据行为，不包含原工作区其他任务改动。

整合回归另发现限时读取时取消与 EOF 同时就绪的状态竞争：扫描器现直接核对入口上下文，避免主循环尚未观察取消时先记为正常 EOF。默认扫描入口仍保持原有结束语义；重复的缓存计费测试沿用基线版本。
