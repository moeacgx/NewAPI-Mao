# Responses WebSocket 专项集成审查

## 交付状态与基线

2026-09-14 至 2026-09-15，审查上游提交 `9fe0457ee1f4b9de407a254500d54f5a8f41ee29`。
本阶段 PR 基线为 `origin/custom-main@45ab82100a74b9e7c3866f1c173d16d32745527e`，
隔离分支为 `agent/upstream-responses-ws`。上游分析点固定不浮动；rc.37 参考点为 `385d2dfd1`。
首次工作区错误采用旧本地 `353352428`，同步前 behind=92、ahead=3，现已安全对齐。
三个旧独有提交 `353352428`、`99db8805c`、`5fb6a75ca` 均为合并提交；旧 HEAD 与共同祖先
`cc8fd2bbe` 的文件树相同，无需把旧合并历史带入本阶段 PR。
恢复点为本地分支 `backup/responses-ws-before-base-sync-20260915` 及 stash
`47d0fb5b46cffe6a3e9edff5a4c1c8586c6008db`；stash 保留，文档索引按块合并，未覆盖最新条目。

**结论：暂缓接入运行时。原提交不得直接 cherry-pick。**

本次只集成独立的 relaykit DTO 兼容字段并验证协议数据契约，不注册 GET
`/v1/responses`，不引入上游的 controller、relay、共享计费抽取或前端开关。
HTTP/SSE Responses 与 `/v1/realtime` 继续走原有入口。
`responses_websocket_enabled` 缺失时为 false；在此准备分支中，即使显式设为 true，
也不会产生可用的 Responses WebSocket 服务。

## 目标、范围与方案

- 目标：核对鉴权、选渠、逐轮计费、usage、重试、关闭、审计、亲和和错误展示契约，
  将可独立的 DTO 准备与尚不兼容的运行时实现明确分开。
- DTO：在原有 `ChannelSettings` 上追加默认 false 的渠道配置字段；在 `Usage` 上追加
  可空 `output_tokens_details`，保留本地输入/输出与缓存字段的 presence 标记。
- 安全边界：本次不增加网络入口，不改变权限、审计链或生产配置；仅提交、推送和草稿 PR，不部署、不合并。
- 兼容性：不替换完整 DTO 文件，不覆盖本地 `AdvancedCustom`、HTTP 传输配置、缓存别名、
  显式零值和 `BillingUsage`。字段解析不等于 usage 归一或计费集成已完成。
- 测试计划：relaykit 独立构建与 DTO 测试；本地 controller/middleware/Responses/计费
  契约回归；在另一个 detached worktree 上运行原上游的协议测试，分别记录结果。

## 尚未允许合入的部分

以下上游行号均对应 `9fe0457ee`，本地行号均对应 `353352428`，不能混用版本。
这些是源码确认的集成冲突；没有据此声称线上已发生资金损失或审计绕过。

2026-09-15 在新基线 `45ab82100` 复核：并发租约释放、选路后审计、usageError 返回、
亲和与失败排除契约仍存在；上游约束接口和累计器依赖仍未接入。以下旧行号保留为首次审查
证据，不能作为新基线行号。新基线另有必须保留的契约：

- `controller/relay.go` 的 relayErrorLogDisplayContent 将日志正文保存为客户端最终文案，
  `other.upstream_error` 仅为管理员保留脱敏原文；不能恢复为所有用户可见的上游原文。
- HTTP/SSE Responses 清理 previous_response_id 和非法 call_output；未来 WS 准备函数
  必须明确区分传输，不得把 HTTP 专属清理直接用于有状态 WebSocket。
- 新基线的上游响应模型采集必须随消费日志保留；DTO 增量不覆盖实际响应模型字段。

### P1：并发租约泄漏和后续轮次准入缺失

上游 `controller/responses_websocket.go:42` 的内部引擎仅运行清理、TokenAuth 和
ModelRequestRateLimit，不执行本地 Distribute。首轮选渠调用本地
`middleware/distributor.go:622` 的 SetupContextForSelectedChannel，会获取渠道租约，
`:664` 还获取分组用户租约；释放原本由 Distribute 的 `:324` 负责。
照搬上游 runner 会漏掉释放。后续轮次在上游 `relay/responses_websocket.go:241`
只恢复锁定上下文，反而不重新获取这两类租约。

接入要求：每个 create 独立获取和释放请求级租约，包含鉴权失败、预扣失败、超时、取消、
panic 与成功终态。连接级锁定不豁免请求级并发。不能直接补一个 Distribute 后再重复选渠。

### P1：内容审计和归档被旁路

上游新 runner 缺少本地 `middleware/prompt_audit.go` 的请求归档、对话归档、原始安全快照、
会话阻断和 Qwen/屏蔽词门禁。上游 `relay/request_billing.go:25` 的旧式
CheckSensitiveText 不能替代本地 `controller/relay.go:606` 的最终渠道/分组过滤。

接入要求：先把 flat/wrapped create 规范化为 Responses 正文，更新 BodyStorage，再进入
本地 PromptAudit；选定或恢复真实渠道后从原始正文执行最终过滤，使用 mask 后重新解析的
DTO。上游闭包再次从原始 message 解码，因此仅增加一个审计中间件仍不能证明 mask 生效。
每轮归档与审计必须具有独立 request_id、渠道、稳定分组和用户快照；不得归档密钥或完整头。

### P1：零用量错误被吞，最终预扣可能未退款

上游 `relay/responses_handler.go:105` 抽取的 ConsumeResponsesQuota 返回 void；上游
WS 的 `relay/responses_websocket.go:350`、`:386`、`:412` 在结算后返回 nil。
本地 `service/text_quota.go:466`、`:519` 在零用量和无产出取消时返回 usageError，
并保持预扣会话，等待 controller 重试或最终退款。

触发：握手已成功，但上游无输出断开、超时或零用量结束。照搬后返回 nil，
RefundFailedRequestBilling 也因 nil 跳过退款。共享 HTTP 抽取同样会破坏现有 HTTP 路径。
本地 PostTextConsumeQuota 的 settlement 错误当前仅记日志，不能把它描述成此次新增的
“吞 settleErr”问题；这里确定被吞的是已经具有返回值契约的 usageError。

接入要求：保留 `*types.NewAPIError` 返回值；每轮使用独立 BillingSession；最终失败对未结算
预扣只退款一次，已结算会话不可再次退款。实际 partial output 可按现有规则计费，但不因此
把业务失败标为成功。余额、订阅、福利券及计费表达式的组快照必须沿用本地预扣/结算入口。

### P1：usage 归一与工具去重倒退

上游 `service/responses_usage.go:50` 仅累计 output_text.delta；`:57` 仅按 item.done
统计部分工具；`:79` 把数值零当成缺失。本地
`relay/channel/openai/relay_responses.go:22`、`:131`、`:159` 已识别终态 output、refusal、
reasoning、function arguments、custom tool input，并按工具 ID/内容去重。

触发：只有终态文本或推理/工具输出而没有 usage，会被新累计器算成零；重复的同一
output_item.done 会重复计工具费用。不能用更短的上游累计器覆盖本地 HTTP/SSE 实现。
本次 `Usage.OutputTokensDetails` 只补充解析载体，尚未将其接入 canonical usage 或计费。
后续归一必须保留 HasInputTokens、HasOutputTokens、缓存 presence 与缓存嵌套优先级，
显式零不应被当成缺失覆盖。

### P1：失败事件、错误替换与原始审计不一致

上游 `relay/responses_websocket.go:366` 只识别 accepted 之前的裸 error，`:386` 将
response.failed 与 completed 一起标为 Done 并返回 nil。accepted 后的 error/response.error
可被原样转发并一直等到超时；`:791` 构造错误时直接调用 ToOpenAIError。

本地 `relay/channel/openai/stream_error.go:47` 识别这三类错误；
`controller/relay.go:247` 记录上游策略错误；`:522` 以后的输出边界处理敏感词安全文案、
客户端错误替换和 request_id。照搬上游会漏记策略事件并把应替换的上游错误原文送给客户端。

接入要求：先保存原始内部错误供审计、策略、自动禁用与指标使用，再生成独立客户端视图。
仅真正上游返回错误允许进入 ReplaceClientErrorCandidates；本地额度、权限、审计拒绝不能
匹配成上游通用故障。错误事件的 status 是应用层状态，不是已完成升级的 HTTP 状态，也不是
WebSocket close code。敏感词、上游 URL、密钥和内部 panic 内容不得直接透传。

### P1/P2：重试、亲和与失败指标缺失

上游首轮 `relay/responses_websocket.go:263` 使用简单 RetryTimes 循环，缺少本地
`controller/relay.go:273` 的 ExcludeChannelID 和 RelayMaxRetries/顺序组边界。
上游 ShouldRetryRelayError 也不具备本地 `:710` 的完整取消、响应已提交、原始状态码、容量、
指定渠道和亲和失败优先级。写出 create 后不得因上游错误或客户端重连盲目重放请求。

上游 `:326` 在握手和发送成功后 RecordChannelAffinity，却没有本地
`:241`、`:251`、`:256` 的业务成功/失败/驱逐处理；连接成功不等于请求成功。
每轮还需接入 Begin/Bind/FinishChannelMetricRequest、每次尝试的指标和最终
perfmetrics.RecordRelayFailure。客户端取消不算上游故障，不触发自动禁用；有 partial usage
的失败也不能进入成功缓存命中率分母。不能用普通 Close/Done 模糊这些结果。
上游 ProcessChannelError 还采用 DisableChannel；本地 DisableChannelWithError 携带的
结构化错误码/状态码通知必须保留。

### P1/P2：关闭开关不是完整回滚，关闭码尚不完备

上游渠道过滤要求 OpenAI 或 Codex 且显式开启，缺失开关为 false。但 GET 路由始终注册，
禁用全部渠道仍可能先升级再在首帧报无可用渠道；上游没有全局握手 gate。
restoreConnectionContext 只在下一次 create 重查开关；普通 UpdateChannel 的 true→false
不广播关闭存量连接，活动生成和空闲连接可能继续存在。

上游 `relay/responses_websocket.go:611` 普遍直接 Close；只有管理策略关闭 `:629` 发 1008。
正常结束、超时、鉴权撤销及上游失败容易在客户端合并为异常 EOF/1006。
后续需明确并测试：1000 正常会话关闭、1008 权限/策略撤销、1011 服务故障；1006 只能观测，
不能作为线上 close frame 发送。上游 close reason 不能未经脱敏直接返回。

上游 wsmanager 的广播事件只含 channel_ids，没有 kind 过滤，渠道关闭会同时关闭 Realtime。
Responses 功能回滚不能直接调用这种“关闭渠道全部 WS”的广播而影响 `/v1/realtime`。
Redis Pub/Sub 按逻辑 DB 隔离但不是持久队列；无 Redis 时只有本进程关闭，不能承诺跨节点
即时撤销。需要连接类型过滤、逐轮重检和明确的节点级兜底。

## 已核对的协议设计与前置依赖

- 入口：GET `/v1/responses` 升级；握手 TokenAuth；每个 response.create 重新 TokenAuth、
  ModelRequestRateLimit 和 request_id，捕获原始凭据并删除握手专用头，避免复用池化 Gin 上下文。
- 消息：支持平铺 create 与兼容 response 包装；向上游输出平铺 create；删除 event_id、
  background、stream、stream_options，保留 generate；max_output_tokens 在计费前受上限约束。
- 串行：一条连接同时只允许一个 create，忙时 409 错误事件；cancel 隶属于当前请求，接受前
  暂存控制帧；不能把所有裸 error 都视为当前请求终止，否则会把 cancel 拒绝误判为生成失败。
- 选渠：首轮过滤 opt-in 渠道，支持 pin、亲和、普通组/auto；连接建立后锁定 model、channel、
  group、key/index。后续检查令牌、组/模型资格、渠道开关和 key 状态；物理连接 URL、代理或头
  改变要求重连。上述设计须与本地租约、路由审计、顺序组结合后才能启用。
- 上游 `ChannelConstraints/ChannelFilter/ResolvedPin/ChannelSatisfiesFilters` 当前基线没有。
  新累计器所需的 NormalizeResponsesUsage、MergeUsageNonZero、
  CloneBillingUsageWithEstimatedCompletion 也不是本地现有公共 API。
- 上游子协议鉴权修复可单独评估：只有合法、非空的 `openai-insecure-api-key.` 条目才覆盖
  Authorization；普通子协议不能破坏已有 Bearer。它影响 Realtime，未在本次修改。
- 上游 controller/channel.go、多 key/标签禁用、Realtime 注册、main subscriber、MJ/task
  测试和两套前端差异均不纳入本次 DTO 准备，不能为了通过编译整片覆盖。

## 配置迁移与回滚契约

### 当前准备分支

仅在现有渠道 setting JSON 中追加可选键，不新增数据库列、表、索引或 AutoMigrate。
旧 JSON、缺失/null/false 都不启用；新建零值配置关闭。已有的本地传输和高级自定义设置保留。
本分支没有配置写入脚本，不批量更新线上渠道。Default、Classic 均未新增界面，因为运行时未就绪。

本分支无需操作服务即可保持关闭：GET Responses WS 未注册。撤回本次字段与文档准备时，应
通过独立反向提交回滚代码；不要清空整个 setting 或删除历史审计、用量和计费流水。
若外部系统已写入该键，旧程序可能在重新序列化 setting 时丢弃未知键，不能保证旧版编辑后保留。

### 后续完整集成的准入条件（尚未实现）

1. 增加独立的全局默认关闭 gate，并明确在升级前拒绝；不复活历史
   `ResponsesWebsocketEnabled` 或客户端 `supports_websockets` 的旧值，不自动迁移为启用。
2. 需要全局 gate 和渠道 `responses_websocket_enabled=true` 同时满足；仅已验证支持的
   OpenAI/Codex 渠道进入候选，pin 不能绕过过滤。切换本地 API 配置不等于上游已支持协议。
3. true→false 必须禁止新握手/新 create，并仅关闭 Responses 存量连接，保留 Realtime。
   活跃轮次按照实际已交付 output 结算或退款一次，不删除账务数据。
4. 多节点先使所有节点 gate 关闭，再验证活动连接已排空；Pub/Sub 故障需节点级关闭兜底。
   之后才能回滚二进制。此处是未来运维契约，不是本次部署指令或已实现的配置项。
5. 在最新协调集成基线上补齐下述测试，再审查实际差异；本专项分支不直接合入 custom-main。

## 协议与本地治理验收矩阵

- 路由：默认关闭、显式开启、无可用渠道、普通 HTTP/SSE 与 Realtime 不受影响。
- 身份：每轮撤销 token、改组、model/pin/key 失效、真实代理 IP、子协议凭据解析。
- 准入：每轮 RPM/成功请求限制、渠道并发、分组用户并发；每个失败阶段租约都归零。
- 账务：两轮独立预扣/结算；首包拒绝、零用量终态、无产出断连、超时、cancel、panic、
  终态写失败和 partial output 均不遗留预扣、不双退款，覆盖钱包/订阅/福利券。
- usage：cached/reasoning/audio/image detail、缺失与显式零、仅终态文本、推理/拒绝/
  function/custom tool、多次重复 item.done 与 terminal output 去重。
- 重试：仅确认 create 未发送时换渠；顺序组/优先级/失败排除/亲和驱逐符合本地规则，
  create 已发送后不因 error、断线或 previous_response_id 自动重放。
- 治理：每轮审计与归档、最终选路 mask 真正进入上游正文、上游策略事件、管理员原始错误、
  客户端替换后的事件及 status、请求/尝试失败指标、取消抑制和缓存统计逐项断言。
- 关闭：1000/1008/1011、大小限制、忙连接、长连接权限变化、开关撤销和 Redis 跨节点范围。

## 可复现反例

[`usage_contract_test.go`](../../testdata/responses_ws_upstream/usage_contract_test.go) 是仅用于原上游
`9fe0457ee` 的实验测试，放在 testdata 中不会进入本地正常 Go 包扫描。将其复制到原上游
detached worktree 的 `service/responses_ws_local_contract_test.go`，执行：

```powershell
$env:GOARCH = 'amd64'
go test ./service -run '^TestAuditResponses' -count=1 -timeout=60s
```

断言终态实际文本仍应计费、同一工具 ID 只收费一次。它们是接入门禁，禁止把期望值改成
零用量或重复收费以让上游测试变绿；也不得直接复制进缺少累计器 API 的本地 service 包。

## 验证记录

### PR #211 快照隔离复审

复审确认新增 OutputTokensDetails 指针经 cloneOpenAIUsage 的浅拷贝发生共享。
修改源对象可污染 NewOpenAIResponsesBillingUsage/CloneBillingUsage 的输出详情快照。
本次仅在同一深复制入口复制非 nil 的 OutputTokensDetails，不注册路由，也不改变计费字段值。
覆盖 Chat/Responses 两种构造、再次克隆及源/两份快照的双向修改隔离；缺失/null 保持 nil，
显式零对象保持存在。Relay 运行时补丁保留在独立分支，本修复不混入该分支。
红灯已复现：修改源 ReasoningTokens 为 99 后两份快照同步变成 99，反向修改音频/图片字段
也相互污染。修复后双向隔离与缺省/null/零对象用例通过；windows/amd64 的独立
`GOWORK=off go build ./...` 和 `go test ./dto ./relayconvert/... -count=1 -timeout=60s` 通过。

2026-09-15 同步到 `45ab82100` 后重新验证：windows/amd64 下独立 relaykit build、
`./dto ./relayconvert/...` 测试均通过；根模块 controller、middleware、relay、
relay/channel/openai、service 聚焦测试通过，新增包含管理错误日志展示和 HTTP 输入清理。
测试使用 `-timeout=60s`。旧配置/null/false 默认关闭以及显式零 detail 的 DTO 测试通过。
本次复核保留 testdata 的必要性：它只含合成 JSON，用来重现固定上游的两个计费反例，
不作为生产代码，也不在本地正常测试中制造故意失败。

以下为首次审查时在旧本地基线和固定上游点执行的历史证据，不代替上述新基线验证：

1. DTO 红灯：新增测试在增加字段前因缺少 ResponsesWebSocketEnabled 和
   OutputTokensDetails 编译失败；字段级补齐后测试通过。
2. 本地 `relaykit`：`GOWORK=off go build ./...` 在 windows/386 和 windows/amd64 均通过。
   386 下 `go test ./dto ./relayconvert/... -count=1 -timeout=60s` 全通过；amd64 下新 DTO
   测试同样通过。没有修改 go.mod/go.sum 或增加根模块依赖。
3. 本地 controller、middleware、relay/channel/openai、service 聚焦回归全部通过：
   覆盖错误替换、本地额度错误不替换、Responses 输出/usage、审计、重试、零用量预扣与亲和统计。
4. 原上游 detached worktree 的 controller、relay、service、middleware、wsmanager
   专项测试在 windows/amd64 下全部通过；涵盖原有 WS 多轮计费、断连结算、cancel 错误、
   首包拒绝退款、消息/选渠/限流/子协议及关闭广播。**这些是上游自身测试，不是本地集成测试。**
5. 在同一原上游 worktree 运行新增反例，两条均按预期失败：终态只有 `hello` 输出且无 usage
   得到 PromptTokens=0、CompletionTokens=0（预期输入估算 10、输出正数）；同一搜索工具
   `ws_search_1` 重复完成得到 CallCount=2（预期 1）。测试保留在 testdata 作为阻断证据。
6. 上游原提交在本机默认 windows/386 编译失败：common/quota_math.go 的
   MaxWalletQuota=9007199254740991 溢出 int。切换子进程 GOARCH=amd64 后通过；本次未改写
   上游额度实现或系统环境变量。首次依赖下载/编译超过 60 秒，后续缓存完成后重跑成功。
7. 修改的 Markdown 使用仓库版本 `oxfmt@0.57.0` 和 `web/.oxfmtrc.json` 格式化/检查通过，
   通过 npm exec 临时调用，未安装全局包。新专题链接、索引入口、gofmt 与 git diff --check 通过。

本地聚焦回归命令：

```powershell
go test ./controller ./middleware ./relay/channel/openai ./service -run 'TestWriteRelayErrorResponse|TestClientErrorReplacementIgnoresInternalQuotaErrors|TestRealtimeClientErrorView|TestShouldRetry|TestOaiResponses|TestResponses|TestPromptAudit|TestPostTextConsumeQuota' -count=1 -timeout=60s
```

原上游验证命令（在 `9fe0457ee` 的 detached worktree 中执行）：

```powershell
$env:GOARCH = 'amd64'
go test ./controller ./relay ./service ./middleware ./pkg/wsmanager -run 'TestResponsesW|TestNormalizeResponsesWS|TestSelectResponsesWS|TestBuildResponsesWS|TestToWebSocketURL|TestHTTPResponsesRequestDoesNotMarshalGenerate|TestApplyResponsesUsage|TestResponsesUsage|TestCloseChannel|TestUnregister|TestRegisteredClose|TestPublishClose|TestRedisChannelClose|Test.*WebSocketSubprotocol' -count=1 -timeout=60s
```

所有测试批次外层均设置 60 秒执行上限。未执行 race、真实上游 WebSocket、真实 Redis 集群、
生产钱包/订阅/福利券全链路、三种真实数据库迁移或前端构建；本次没有数据库/前端程序变更。
不能把这份结果视为上述完整运行时验收矩阵已完成。

## 交付与跨项目边界

本阶段仅交付 DTO、测试与审查文档，通过专属分支提交和推送，并以草稿 PR 对接 custom-main。
草稿不表示 WS 运行时就绪；不得直接 cherry-pick 整个上游提交。提交和 PR 前重新 fetch，
确认 behind=0。未合并、未发版、未部署。

对 tokens-pro/Sub2API 已核实的共享影响仅是将来可能新增 GET WebSocket 传输与逐轮错误、
usage 契约；此准备分支没有启用入口，也没有改变 HTTP API、模型 ID、鉴权或计费语义。
不发送跨项目 Issue。本阶段不修改 Default/Classic；完整集成必须分别实现两套模板的
渠道开关、保存恢复、权限与提示、i18n 和测试，不能以 Default 完成代替 Classic。
HTTP/SSE 计费与归一补丁随后由同一实现者在独立 Relay 分支交付，必要时显式依赖此 DTO PR。
