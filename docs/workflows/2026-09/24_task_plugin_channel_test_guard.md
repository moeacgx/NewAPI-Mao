# 任务插件渠道测试能力判断

## 目标与根因

Classic 的 Cloudflare Jev 渠道配置完成后，通用模型测试返回“渠道与官方插件入口不匹配”。
自动检测把 `typesafe/jev` 送入 `/v1/chat/completions`，在渠道隔离检查时失败，尚未请求 Cloudflare。
这条结果不能用于判断账户地址、密钥、额度或供应商服务是否可用。

## 方案与范围

同步官方 `QuantumNous/new-api` 固定提交 `d04c118c8803f49e0c9bab74dcf5b5efeab9464a`
的 `controller/channel-test.go`：把 `ChannelTypeTaskPlugin` 加入通用测试不支持列表。
返回既有的本地“不支持测试”错误，不产生上游错误分类，不新增插件测试协议。
不放宽 `TaskPluginChannelMatchesPath`，不为模型名硬编码聊天转换。

本次只修改共用 Go 测试入口；截图来自 Classic，但 Default 使用同一后端判断，均无需前端改动。
插件源码、版本、渠道配置、密钥、数据库结构、价格和真实推理接口保持原合同。
无数据迁移；回滚该判断只会恢复旧的误导性错误，不影响已安装插件。

## 使用与安全边界

Cloudflare Jev 必须使用 `POST /v1/systemone`，请求包含 `model/state/questions`。
客户端使用本站令牌；Cloudflare 令牌仅保存在渠道密钥中。
专用接口仍执行插件激活、模型与分组授权、渠道隔离、预扣、结算和失败退款。
通用渠道测试下拉框中的标准端点均不能替代此接口；同步“不支持测试”判断并不新增在线试运行能力。

## 验证结果

- 新增回归测试先在未修复代码上运行：自动检测与显式 Responses 两个场景均复现“渠道与官方插件入口不匹配”。
- 补回官方判断后，两场景返回本地不支持结果，`newAPIError` 为空，用户 quota 保持不变。
- `go test -mod=readonly ./controller ./middleware -run 'TestChannelProbe|TestNormalizeChannelTestEndpoint|TestNativeTaskPlugin|TestTaskPlugin' -count=1 -timeout=60s` 通过。
- `go vet -mod=readonly ./controller ./middleware` 通过。
- Go 格式、三份修改文档的 Prettier 检查、新增相对链接和 `git diff --check` 通过。
- 文档 HTTP 示例的 JSON 经 Cloudflare Jev 实际解码及请求构造函数校验通过，全程零网络调用。
- 没有部署，没有真实 Cloudflare 推理验收；专用接口示例仅离线核对了解码与上游请求构造。

## 已有监控限制

现有批量测试统计只根据 `newAPIError` 计算成功数，未单独统计“不支持测试”。
本次同步官方本地错误语义，不扩展该统计合同；批量汇总里的成功数不能证明这类插件调用了供应商。
通用测试与真实插件调用的模型、认证、请求响应和计费合同均未改变，无跨项目接口变更。
