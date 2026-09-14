# Responses HTTP/SSE 计费详情归一化

## 目标与范围

固定本地基线 `45ab82100`、上游分析点 `9fe0457ee`（rc.37 参考 `385d2dfd1`）。
分支为 `agent/upstream-relay-billing`，与 WS DTO 准备 PR #211 独立交付。
旧干净工作区 `353352428` 的三个独有提交均为历史 merge，其文件树与共同祖先一致；
保留 `backup/relay-billing-before-base-sync-20260915` 后对齐最新 origin/custom-main。

原生 Responses HTTP/SSE 只复制输入/输出总量和 cached_tokens，图片、音频、推理详情与
字段存在性未进入本地计费 usage。表达式使用 img、ai、ao、img_o 时会得到错误的分类数量。
本阶段沿用上游 NormalizeResponsesUsage 的职责划分，适配本地 DTO 与缓存字段，
不整体移植上游累计器、HTTP/WS 共享预扣抽取或任务插件。

## 接口与行为契约

- 新增 relaykit 公共 `NormalizeResponsesUsage(*dto.Usage) *dto.Usage`：将原生 Responses
  总量及已支持的详情映射到结算字段，复制 input_tokens_details 和已有 BillingUsage，
  保留输入/输出/缓存 presence；没有快照时不凭空创建转换侧计费快照。
- `UsageFromResponsesUsage` 继续为协议转换保留/创建 BillingUsage，复用同一字段映射。
- 原生 HTTP 与 SSE 的终态 usage 使用前者。保留原始响应字节、实际上游响应模型采集、
  缺失/零值 usage 的既有文本回退，以及工具去重、空响应错误和未提交流重试边界。
- 原生网络响应边界继续忽略 wire 中的 billing_usage、usage_semantic、usage_source、cost 和
  Claude 专用统计，不让内部转换元数据覆盖本地回退或改变原生计费语义；转换 API 仍可保留快照。
- SSE 后续 usage 缺少缓存或整个详情时保留已观测的详情，显式 cached/cache-write 零值覆盖；
  completion_tokens_details 按原始帧的字段存在性区分缺失与显式零对象。
- 输入图片/音频从 input_tokens_details 读取；输出侧仅覆盖当前基线已有的
  completion_tokens_details。标准 output_tokens_details 的端到端映射依赖 #211 的字段准备，
  留待后续显式组合，不在此 PR 重复增加 DTO 或伪称已支持。
- 表达式继续按实际使用的变量排除 p/c 子类别，len 仍为完整输入；不更改表达式语法、
  系数、QuotaPerUnit、钱包上限或预扣/退款状态机。

## 安全、兼容与回滚

不修改任务适配器、relay_task.go、relay/common/relay_info.go、路由、鉴权、审计、亲和性和
失败指标。无数据库/配置迁移，无生产 API、发版或部署。Default/Classic 使用日志 API 结构
不变，无 UI 改动；两套模板的 WS 入口仍属于后续完整功能范围。

回滚本补丁会恢复原生 usage 详情丢失行为，不需要改数据库；历史消费日志与账务不得删除。
新增归一接口只依赖 relaykit 内部包，必须独立 build。#211 不是本 PR 的编译依赖。

## 测试计划

1. 原生 HTTP 与 SSE 返回相同 input/output detail，表达式 p/c 排除后的分类和成本一致。
2. 转换路径保留计费快照，原生路径不新建；显式零与缺失 presence、缓存创建优先级不丢失。
3. 本地空响应、文本回退、错误替换、实际响应模型、预扣退款、亲和与安全重试定向回归。
4. `GOWORK=off go build ./...`、relaykit 定向测试、根模块定向测试，均使用 60 秒测试超时。

## 验证与剩余工作

2026-09-15，在上述新基线 windows/amd64 完成红绿验证：

- 新 HTTP/SSE 测试在修改前均失败：输入 100、输出 20，缓存 30、缓存创建 4、图片输入 10、
  音频输入 5、图片输出 3、音频输出 2 时，详情丢失导致 P=66、C=20、Img/AI/ImgO/AO=0。
- 修复后两种传输均得到 P=51、C=15、Len=100、CR=30、CC=4、Img=10、AI=5、ImgO=3、AO=2。
  表达式 `p*2+cr*0.5+cc*3+img*4+ai*5+c*6+ao*7+img_o*8` 的原始计算值由错误的 279
  恢复为 322（尚未执行美元/百万 token 与 quota 换算，不是直接扣费 322）。
- 断言原始响应内容与实际响应模型保持；原生无快照、转换创建快照、已有快照独立复制、
  嵌套显式零不被顶层缓存创建别名覆盖均通过。
- 只读复审补齐两组红绿回归：网络响应中的零快照不能压掉实际文本回退，正快照不能伪造
  零用量响应的产出；稀疏终态不擦除已有缓存/输出详情，显式缓存和输出零值覆盖。
  原生入口保留旧有内部元数据边界；SSE 零用量仍由外层结算判定，未改变该调用层级。
- 后续仅有顶层 cache_write_tokens 的 0/5 更新，必须覆盖先前嵌套值 3；此组合也已先复现
  失败再修复验证。恢复缺失详情后重新应用当前帧的缓存创建规范值，避免旧嵌套值遮蔽更新。
- 独立 relaykit 全量 build/test，OpenAI relay 与 billingexpr 全包测试通过。
- controller、middleware、relay、service 定向测试覆盖 Responses、本地错误替换、管理错误
  日志、提示词审计、零用量预扣保持、BillingSession、阶梯结算和分组重试、渠道亲和。
- `go vet ./relay/channel/openai`、gofmt、仓库版本 oxfmt 格式化和 `git diff --check` 通过。

验证命令：

```powershell
$env:GOARCH = 'amd64'
# 在 relaykit 目录，禁用任何根工作区依赖
$env:GOWORK = 'off'
go build ./...
go test ./... -count=1 -timeout=60s
# 在仓库根目录
go test ./relay/channel/openai ./pkg/billingexpr -count=1 -timeout=60s
go test ./controller ./middleware ./relay ./service -run 'Test.*(Responses|RelayError|ClientErrorReplacement|ShouldRetry|PromptAudit|PostTextConsumeQuota|BillingSession|TryTieredSettle|BuildTieredTokenParams|PrepareTieredBilling|ChannelAffinity)' -count=1 -timeout=60s
go vet ./relay/channel/openai
```

本阶段没有执行根模块全量测试、生产账务、真实上游调用、race、双前端构建或三库实机验证，
不能宣称全量上游集成就绪。按用户授权交付独立补丁和草稿 PR，不等待全部上游功能迁移，
不合并 custom-main、不发版、不部署。

后续依赖与阻断：

- 标准 output_tokens_details 的 HTTP/SSE/转换端到端支持需要显式衔接 WS DTO PR #211，
  并复核输出子类；其新增指针的快照深复制已由 #211 的 `4679c18c4` 修复。
  本 PR 不包含 #211 的文件或提交，独立基线仍为 `45ab82100`。
- 模型修饰符、Kimi K3、完整 usage 多块合并、图片缓存/数量表达式扩展仍需独立验收。
- 上游共享预扣和 void 结算抽取不能覆盖本地 usageError 返回、退款及实际选路审计。
- WS 运行时的请求级并发租约、错误/关闭/回滚契约仍未实现；Default 和 Classic 的 WS 功能
  必须分别实施和验证，不在此 HTTP/SSE 补丁中声明完成。
- 任务适配器、relay_task.go、relay/common/relay_info.go 由任务 Agent 所有，本阶段未修改。

共享契约影响：原生 Responses 的既有分类 usage 可以进入现有表达式计费和日志，客户端响应
JSON/SSE、模型 ID 与身份验证契约不变；未发送跨项目通知。
