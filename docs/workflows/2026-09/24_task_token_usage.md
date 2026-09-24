# 原生同步任务 Token 统计修复

## 问题与范围

Jev 成功返回用量后，任务消费日志没有填写输入和输出 Token，渠道指标也收到空用量。
Cloudflare 插件 0.2.2 仅在完成钩子上报 input_tokens，output_tokens 只存在于响应正文。
按次收费不应阻止统计实际 Token。当前官方上游 d04c118c8 同样未写入这两列。

## 契约与方案

只在原生插件路由成功完成、完成用量 Hook 整体验证通过后，从未经计费饱和处理的
原始 facts 读取 input_tokens/output_tokens，放入仅进程内的 TaskSubmitResponse.ActualTokenUsage。
计费用的 Immediate.UsageFacts 和原有额度饱和逻辑保持不变。
两字段作为 Token 统计字段，必须是有限、非负整数，不大于 int32 上限；
非法字段忽略并告警，不影响另一合法字段。两字段总和超过 int32 上限时，整组
统计用量忽略并告警，避免 32 位宿主的统计导出相加溢出；不截断或伪造实际数量。
不从预扣快照推算实际用量，不把 upstreamUnits 或 legacy totalTokens 当 Token。
消费日志与渠道指标使用同一份实际用量，日志附加 task_token_usage 保留字段存在性。

费用继续由原按次价格或用量表达式计算；输出价格为零不代表输出 Token 为零。
Cloudflare 插件需升级至 0.2.3 才上报输出 Token；旧插件在新宿主仅恢复其已上报输入量。
插件 0.2.3 配旧 .335 宿主可继续调用与计费，但不能修复旧宿主的日志统计缺口。

## 边界与兼容

只调整后端，不修改 Default 或 Classic 页面；两模板读取相同的日志 API 数值。
不修改请求、认证、选渠、模型别名或响应正文，不改变数据库结构。
异步提交、非成功终态和上游用量缺失不记估算 Token；轮询统计不在本次范围。
历史 retainResult=false 结果已清除，不能凭空补回旧日志。修复影响升级后的新请求。

## 验证计划

先复现消费日志 0，验证输入/输出存在性、显式零值、非法与越界数值，
完整路由验证按次/表达式扣费不变、Token 正确落库、上游失败退款和旧插件兼容。
使用本机模拟上游，不发起生产收费请求。测试超时 60 秒。

## 验证结果

- 原始宿主的官方 Jev 路由测试复现输入期望 1000、实际 0；相同 Cloudflare 0.2.3
  在旧 .335 上开启日志断言时复现 486/70 均为 0，在修复宿主上全部通过。
- `go test -mod=readonly -timeout=60s ./service ./controller ./router ./relay/channel/task/jsplugin ./relay -count=1`
  全部通过。包含转换边界、日志真实落库、渠道 collector、固定费用 0.001 美元、
  输入单价 0.5 美元/百万且输出免费、显式零值、官方旧插件输入统计及失败退款。
- 插件 123 项 Node 协议测试、5 个不可变版本索引校验、Go 宿主 JS 引擎回放通过。
  新插件与新宿主联测使用 `CLOUDFLARE_JEV_EXPECT_TOKEN_LOGS=1` 运行
  `node scripts/verify-cloudflare-host.mjs <修复宿主目录>`，覆盖规范名、别名、
  按次和输入表达式，验证消费日志实际 Token 与钱包/令牌扣款。
- `gofmt`、修改文档 Prettier、新增链接检查和 `git diff --check` 通过。
  开发文档索引全量链接检查发现基线已缺少 `15_topup_payment_fee_included_hint.md`，
  与本次修复无关，未扩大范围修改。
- 无真实付费模型调用；未部署到生产。历史日志不作补填。
