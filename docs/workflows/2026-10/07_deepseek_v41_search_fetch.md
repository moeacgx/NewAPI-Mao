# DeepSeek V4.1 与 Search/Fetch 接入

## 目标

接入 DeepSeek V4.1 Flash/Pro 模型标识、`-none`/`-low`/`-max` 思考后缀，
以及 DeepSeek Search/Fetch API。

## 范围

- DeepSeek 渠道模型列表增加 `deepseek-flash`、`deepseek-v4-pro` 及思考后缀。
- 增加 `/v1/tools/search` 与 `/v1/tools/fetch` 路由，保留请求体字段并原样转发。
- Search/Fetch 使用独立路由模型名和按次价格；只有返回搜索结果或页面内容时结算。
- DeepSeek 自定义上游和 New API 兼容渠道支持这两个端点。
- DeepSeek 的 `@thinking`、`@effort`、`@temperature` 控制在适配器内转换为上游字段。

## 计费与安全边界

Search/Fetch 请求按每次 0.00075 美元的默认价格预扣，空结果退还预扣；上游错误不结算。
端点请求体只在本地读取后重新使用原始 JSON，未知字段不会丢失。模型名仍通过现有分组、
渠道能力和模型映射校验。

## 验证

- `go test ./relay/channel/deepseek ./relay ./relay/common ./relay/helper ./middleware`
- `cd relaykit && GOWORK=off go test ./...`
- `git diff --check`

当前实现保留本地现有计费与协议转换链；官方完整的 hosted-tool 计数重构仍需单独评估。
