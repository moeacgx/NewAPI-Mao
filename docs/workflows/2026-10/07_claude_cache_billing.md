# Claude 显式零用量与缓存计费修正

## 目标

修正 Claude Messages 流式响应在以下边界下的用量和结算：

- `input_tokens` 为 0，但缓存读取或缓存写入仍有数量时，保留缓存计费；
- `message_delta` 携带显式 0 时，不覆盖 `message_start` 已收到的有效输入或缓存用量；
- 流中断时，仅在用量缺失时使用本地文本估算；正常拒绝且没有输出时保留上游的 0；
- 最终计费快照与流结束阶段修正后的输出用量一致。

## 变更范围

- `relay/channel/claude/relay-claude.go`：按输入、缓存和已输出文本判断是否缺少用量，避免显式零值覆盖有效数据；最终生成与结算一致的 Claude 计费快照。
- `service/text_quota.go`：Claude 语义下缓存读取和缓存写入可单独使请求进入计费路径，即使普通输入输出 token 都为 0。
- 对应测试覆盖缓存只读、缓存写入、拒绝响应、流中断、显式零值和结算快照。

该修复不改变非 Claude 请求的零用量规则，也不把客户端断开转换为上游成功。缓存是否实际命中仍由上游返回的用量决定。

## 验证

```text
go test -timeout 60s ./relay/channel/claude ./service
git diff --check
```
