# 流式性能统计与上游响应头等待优化

## 目标

修正两个独立问题：

- 流已经收到 `DONE` 或 EOF，随后 HTTP 上下文才取消时，不再把它计为客户端取消；
- 为上游响应头等待增加可配置上限，避免上游连接建立后长期不返回响应头而占用请求资源。

## 变更范围

- `service/channel_metrics_lifecycle.go`：正常结束且无错误的流优先按成功统计，即使请求上下文在收尾阶段已取消。
- `common/constants.go`、`common/init.go`、`service/http_client.go`：新增
  `RELAY_RESPONSE_HEADER_TIMEOUT`，默认 `1800` 秒，设置为 `0` 禁用；仅限制等待响应头，
  不限制响应头返回后的流式读取。
- `.env.example`：记录配置和语义。

客户端中途断开、流扫描错误、超时和真实上游错误仍按原有失败/取消口径处理。
该变更不修改 Nginx 配置，也不会把客户端断开转换成 499。

## 验证计划

- `go test -timeout 60s ./service -run 'TestClassifyChannelMetricAttemptKeepsCompletedStreamSuccessfulAfterCancellation|TestRelayHTTPTransportResponseHeaderTimeout' -count=1`
- `go test -timeout 60s ./pkg/perf_metrics`
- `git diff --check`
