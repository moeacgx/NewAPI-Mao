# WorkBuddy 经强制 Responses 的缓存结算与请求参数修复

## 目标与范围

基线 `9cfa20df9`。修复 WorkBuddy → Chat Completions → 强制 Responses → GPT
链路中，客户端显示缓存命中但实际结算丢失缓存折扣，以及缓存请求参数在协议转换中丢失的问题。
仅修改相关 Go 用量归一化、请求 DTO 和转换映射、直接行为回归测试及开发文档。

## 根因与方案

- Responses 转 Chat 已将 `input_tokens_details.cached_tokens` 转成客户端的
  `prompt_tokens_details.cached_tokens`，同时保存未经归一化的原生 `BillingUsage`。
  结算优先读取快照，但 `usageFromOpenAIBillingUsage` 只补齐总 token 数，未补齐缓存读取详情。
  因此原生输入 100、输出 20、缓存读取 70，客户端缓存为 70，实际结算缓存为 0。
- 在结算边界仅补齐缺失的缓存读取详情：Prompt 侧显式值（含零）优先；旧 Go 构造的非零值即使
  无 presence 标记也保留；缺失才从 Input 侧复制值和 presence。保留其他已有字段与总量规则。
- 用量结果、原始计费快照及结果中携带的快照必须彼此独立，避免嵌套指针造成原始依据污染。
- Chat DTO 增加 `prompt_cache_options` RawMessage，Chat ↔ Responses 显式保留
  `prompt_cache_options`、`prompt_cache_retention` 和既有 `prompt_cache_key` 契约。
  未提供的字段继续省略，已有 RawMessage 保持原始 JSON 语义，不替客户端生成缓存策略。
- Responses→Chat SSE 的元数据读取会替换 usage；标准流在创建时 usage 为空，在唯一终态报告
  usage。目前复现仅需一个终态快照，没有多帧覆盖的关联证据；本次不为额外或重复终态扩展累计器。

## 接口、安全与兼容

不改变路由、强制 Responses 开关、system/developer 消息顺序、鉴权、模型名、倍率或表达式语法。
结算摘要、`cache_tokens` 日志和表达式 `cr` 均使用同一个 effective usage。
没有 BillingUsage 的历史路径继续使用原 usage；不从客户端展示字段反向填充可信计费快照。
不涉及数据库、生产环境、密钥、提交、推送、PR、发布或版本号。Default/Classic 无前端改动。
缓存策略是否生效与真实命中取决于上游支持，本地仅验证字段保留和已报告用量的结算。
回滚代码会恢复原有缓存折扣遗漏；不重写历史账务或消费日志，无数据迁移。

## 测试计划

1. 原生 Responses 快照经非流式及标准 SSE 转换后，验证客户端缓存、实际 effective usage、
   结算摘要、日志字段、倍率折扣和表达式 `cr` 一致；原生 100/20/70 使用同一固定 fixture。
2. 验证缓存缺失、Input 显式零、Prompt/Input 冲突、Prompt 显式零、旧构造非零及无快照兼容，
   并以双向修改证明归一化结果及快照隔离。
3. 验证三个缓存请求字段在流/非流公共请求转换中保留，反向转换及缺省/显式空对象语义正确。
4. `go test` 均使用 `-timeout=60s`，执行受影响回归、gofmt、
   `cd relaykit; GOWORK=off go build ./...`、仓库 Markdown 格式化、链接检查与 `git diff --check`。

## 验证结果

2026-10-30，本地 Windows/amd64 完成先失败再修复验证：

- 旧代码 JSON/SSE 均显示缓存 70，但 effective usage、摘要、日志缓存字段和表达式 `cr` 为 0。
  输入 100、输出 20、模型倍率 1、输出倍率 2、缓存倍率 0.1 时，旧倍率额度为 140，修复后为 77。
  表达式 `p*2+c*4+cr*0.2` 经每百万 token 换算及 `QuotaPerUnit=500000` 后同样得到 77；
  `p=30`、`cr=70`、`len=100`，上下文长度没有被缓存减小。
- 原始快照仍保留 Input 缓存 70 和 Prompt 缓存 0，证明补齐发生在结算边界，没有改写原始依据。
  Input/Output 详情分别深复制；修改结果不污染原始快照或结果携带的快照，修改原快照也不影响结果。
- Prompt/Input 缓存冲突、Prompt 显式零、Input 显式零、详情缺失、旧构造非零和无快照回归通过。
  其他 Prompt 详情和原有总量规则保持，客户端伪造的展示缓存值不用于覆盖快照。
- 请求公共转换器覆盖流/非流及双向转换，保留三个字段及选项中的 `false`/`0`、空对象和显式 `null`。
  强制 Responses 的实际 `TextHelper` 模拟上游入口覆盖流/非流，以及透传、反向转换同时开启的优先级。
- SSE 检查确认 `applyResponseMetadata` 整体替换 usage，但复现使用标准 `response.created`（usage
  为 null）→文本 delta→唯一 `response.completed`，不存在前一份正缓存详情被后帧擦除。
  该替换行为不参与本次结算丢失；没有修改流状态机、重复终态、非标准序列或消息顺序。

实际执行命令如下。Go 子进程使用 `GOARCH=amd64`，不修改全局环境；relaykit 子进程额外设置
`GOWORK=off`。所有测试均使用 `-timeout=60s`，外层 Python `subprocess.wait(timeout=60)`
超时后只结束本次命令进程树。

```text
# 根目录：新回归在旧代码先失败
go test ./service -run TestResponsesCacheBillingMatchesClientUsage|TestEffectiveBillingUsageCachePresenceAndSnapshotIsolation -count=1 -timeout=60s
go test ./relay -run TestForceResponsesRoutesConversationRequests/chat -count=1 -timeout=60s
# relaykit 目录：新回归在旧代码先失败
go test ./relayconvert -run TestPromptCacheRequestConversion -count=1 -timeout=60s

# 根目录：修复后全部通过
go test ./service ./relay ./relay/channel/openai ./pkg/billingexpr -run Test.*(Responses|EffectiveBillingUsage|BillingUsage|CalculateTextQuotaSummary|BuildTieredTokenParams|TryTieredSettle|UsageBillingPath|AppendUsageBillingPath) -count=1 -timeout=60s
go test ./relay/channel/openai ./pkg/billingexpr -count=1 -timeout=60s
go vet ./service ./relay ./relay/channel/openai

# relaykit 目录：修复后全部通过
go test ./dto ./relayconvert/... -count=1 -timeout=60s
go build ./...
```

关键输出：`service 1.641s`、`relay 1.269s` 定向通过；OpenAI relay 全包 `5.692s`、
表达式全包 `2.247s`、relaykit DTO `0.671s`、公共转换器 `1.647s` 及其子包均通过；
独立构建和 `go vet` 退出码 0。首次根模块冷编译与首次 `go vet` 各触及外层 60 秒上限，
缓存预热后的重跑均完成；超时未被记录为测试通过。

`gofmt -l` 无输出；仓库版本 `oxfmt@0.57.0` 格式化与 `--check` 通过（3 个 Markdown 文件）；
新增相对链接检查 `checked=2, missing=0`，`git diff --check` 无输出，均退出码 0。

## 验证边界与共享契约

本次为本地 Go 修复与模拟上游验证，没有真实 WorkBuddy 客户端、GPT 供应商、生产消费、
历史账务重算、CI 或发布验收。没有执行根模块全量测试或三库实机检查；本补丁无数据库变更。
Default/Classic 均无页面修改，使用既有 `cache_tokens` 日志契约，不需要前端构建。

共享影响仅为 Chat/Responses 的可选缓存请求字段保留，以及原生快照的缓存用量进入统一结算、
日志和表达式；不改鉴权、模型 ID、响应协议和日志结构，不发送跨项目通知。

- `NewAPIForDouDi`：若仍使用相同快照优先结算及 Chat→Responses 桥接，建议分享这组修复与回归；
  先核对 DTO presence 与快照克隆实现，不能只复制字段赋值。
- `NewAPIModifyByGang`：若提供强制 Responses 或双向桥接，同样值得分享；依赖其结算入口、
  缓存详情存在性与 RawMessage 支持。未核验兄弟仓库当前源码，不建议未经对照直接照搬。
