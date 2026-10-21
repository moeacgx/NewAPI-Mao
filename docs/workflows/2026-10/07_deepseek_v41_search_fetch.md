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

### 2026-10-18 独立端点结算回归修复

独立 Search/Fetch 的预扣使用管理员配置的 `deepseek-search` / `deepseek-fetch`
按次价格，但此前成功结算把调用写入聊天内置工具计数，并将模型按次价清零，导致结算
改用 `deepseek_search` / `deepseek_fetch` 附加费价格。两套价格必须保持独立。

修复方案与验收契约：

- 独立端点的预扣和结算统一使用本次请求取得的 `PriceData.ModelPrice` 与分组倍率；
  管理员的自定义按次价和显式零价均生效，聊天内置工具价格不参与独立端点结算。
- 仅独立 Search/Fetch 模式中，由返回非空结果或页面内容的成功路径传入非空用量对象，
  表示一次成功调用；输入、输出 token 仍为零。缺失用量的普通聊天不能使用此例外。
- 成功调用（包括免费调用）复用消费日志、用户请求次数、用户/渠道已用额度及渠道指标
  链路；空结果退款，上游错误和读取失败不进入成功结算。
- 按次价格特判以独立 RelayMode 为边界，同名聊天模型继续遵守其阶梯表达式配置。
- 不改变路由、鉴权、模型映射、数据库结构和上游协议；前端不在本次修复范围。
- 回归覆盖自定义价格、零价、非单位分组倍率、失败不收费、审计统计，以及 Claude
  Search / Responses 工具计数去重；仅本地验证，不代表生产付费调用已验证。

验证结果：

- 修复前，真实独立 Search 入口在每次模型价为 0.012 美元、聊天工具价为 999 美元/千次时，
  结算错误收取 499500 额度，而正确值为 6000；同名聊天模型的阶梯快照也未建立。
- 修复后，Search/Fetch 自定义价、显式零价、空结果、上游错误与截断回包测试通过。
  SQLite 验证钱包/令牌余额、用户/渠道已用额度、请求次数、零 token 消费日志及渠道成功指标。
- Claude 搜索累计次数、Responses 非流式供应商计数，以及流式事件与终态的替换去重通过，
  未同时收取通用 Web Search 与 DeepSeek Search 费用。
- 完整受影响包回归：
  `go test ./relay ./relay/helper ./service ./relay/common ./relay/channel/deepseek ./relay/channel/claude ./relay/channel/openai -count=1 -timeout=60s`
  通过；Markdown 使用本机已有 Prettier 格式化，`git diff --check` 通过。
- 未修改 relaykit 模块、未执行 MySQL/PostgreSQL 实库测试、未部署或访问生产环境。

- `go test ./relay/channel/deepseek ./relay ./relay/common ./relay/helper ./middleware`
- `cd relaykit && GOWORK=off go test ./...`
- `git diff --check`

当前实现保留本地现有计费与协议转换链；官方完整的 hosted-tool 计数重构仍需单独评估。
