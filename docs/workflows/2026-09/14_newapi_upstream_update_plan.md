# NewAPI 官方上游更新集成计划

> 当前交付范围：用户于 2026-09-15 明确授权将第一阶段 PR #210–#216 合入 `custom-main`。
> 下文保留调查和分发的历史状态；早期“不合并”约束已由此次授权替代，部署仍不在范围内。
> 七项兼容补丁不等于完整同步上游 `9fe0457ee`，未实现能力继续按文档待办跟进。

## 1. 基线与目标

- 本地仓库：`moeacgx/maolaonewapi`。
- 协调分支：`merge/newapi-upstream-latest`，固定功能基线：`origin/custom-main@45ab82100`，版本：`v1.0.0-rc.10.1.10.322`。
- 官方上游：`QuantumNous/new-api`，目标分支：`main`。
- 已确认上游最新提交：`9fe0457ee`（2026-09-14，Responses WebSocket）。
- 官方已发布版本参考：`v1.0.0-rc.37`，提交：`385d2dfd1`；发布说明仍不推荐生产使用，不将其视为稳定性保证。
- 当前差异：合入 rc.37 约 116 个提交、1,267 个文件；合入最新 main 约 131 个提交、1,302 个文件。
- 直接合并预演约有 185～187 个冲突文件，不能使用整文件 ours/theirs 解决。

本计划只定义调查、迁移、验证和集成边界。各 Agent 必须在独立 Paseo worktree 和专属分支工作，不得直接修改本协调工作区，不得部署或合并到 `custom-main`。

## 2. 上游更新分层

### A. 必须优先集成的基础修复

1. 数据库兼容与迁移：旧约束、`prefill_groups` 索引、SQLite WAL/写锁、PostgreSQL pooler 兼容。
2. 请求与计费安全：图片数量校验、缓存 token 分解、任务时长边界、Responses usage、预扣与退款共用逻辑。
3. Relay/relaykit：模型修饰符、推理参数、Kimi K3 工具消息、OpenAI Responses 兼容、JSON codec 注入。
4. 可靠性修复：上游响应头等待超时、SSE 识别、错误状态重试、日志敏感字段隔离。

### B. 需要保留二开契约的功能

1. 本地 AtlasCloud 任务适配器。
2. `relay_task.go` 的源任务多分组授权、分组别名和继承模式。
3. 上游响应模型采集、管理员可见/普通用户脱敏、渠道测试显示。
4. 渠道亲和性、失败驱逐、性能指标、客户端错误替换、通知前缀去重。
5. Default 与 Classic 双前端路由、支付、通知、福利、发票和自定义模块。
6. 本地钱包 `int64` 语义及 32 位安全边界，不能被上游钱包限制直接覆盖。

### C. 可单独开关或延后集成的高风险功能

1. 实验性 JS Task Plugin 系统：上游替换大量原生 Go 任务适配器，涉及沙箱、资源存储、签名访问和插件市场。
2. 统一账户安全：TOTP/Passkey、访问令牌、审计日志、敏感操作 proof、Argon2id。
3. Telegram 统一 OAuth：旧登录/绑定接口返回 HTTP 410，需要 `client_id/client_secret`，不能只依赖旧 Bot Token。
4. Responses WebSocket：新增 `/v1/responses` WebSocket、渠道显式开关，并重构 HTTP/WS 预扣费、重试和结算。
5. 模型供应商与价格后台重构：Default 前端改动集中，需确认 Classic 不被接口变化破坏。

## 3. Agent 分工与交付物

### Agent 1：后端基础与数据库兼容

范围：`common/`、`model/`、`middleware/`、数据库迁移、Go 依赖和 CI。

交付：逐文件冲突清单、保留的本地逻辑、可应用的补丁/提交、根模块测试结果、SQLite/MySQL/PostgreSQL 风险。

### Agent 2：Relay、计费与 relaykit

范围：`relay/`、`relaykit/`、`pkg/billingexpr/`、`service/*billing*`、Responses HTTP 路径。

交付：计费不变量核对、模型修饰符和 usage 迁移方案、AtlasCloud/本地审计契约接入点、根模块与 `GOWORK=off` relaykit 验证结果。

### Agent 3：任务插件与任务链路

范围：`pkg/jsplugin/`、`plugins/`、任务适配器、`service/task*`、`relay/relay_task.go`。

交付：原生适配器到 JS 插件的映射表、AtlasCloud 保留方案、源任务授权保留方案、插件资源/迁移/回滚风险和测试计划。

### Agent 4：认证、Telegram、Passkey 与审计

范围：`controller/auth*`、`middleware/auth*`、`oauth/`、`service/auth*`、用户会话、审计日志和相关 API。

交付：旧接口兼容矩阵、配置迁移要求、二开会话安全差异、前后端改动列表、回滚与升级提示、定向测试结果。

### Agent 5：Default 前端

范围：`web/src/` 及 Default API 类型、渠道/模型/价格/插件/日志/系统设置页面。

交付：按功能域的冲突清单、与本地二开页面的保留规则、i18n 影响、Bun lint/typecheck/test/build 结果。

### Agent 6：Classic 与跨模板契约

范围：`web/classic/`、共享路由、支付/福利/发票/通知/主题切换、后端接口兼容。

交付：Classic 受影响页面清单、无需变更的证据、必须同步的接口类型、Classic lint/test/build 结果。

### Agent 7：Responses WebSocket 专项

范围：`9fe0457ee` 相关 WebSocket、渠道开关、路由、预扣/退款、usage、重试、审计和日志。

交付：协议与安全边界、与本地渠道选择/亲和性/指标的差异、独立开关和回滚方案、协议测试结果。默认不启用生产配置。

### Agent 8：集成审查与发布门禁

范围：只读审查，不直接改代码；汇总其他 Agent 的结果。

交付：冲突优先级、集成顺序、依赖图、必须阻断项、验证矩阵、是否可进入隔离集成分支的明确结论。

## 4. 集成顺序

1. 先完成 A 类基础修复和 B 类本地契约盘点。
2. 再单独评估 C 类功能；任何涉及数据迁移、认证方式或插件运行时的变更必须先产出升级/回滚说明。
3. 在新的隔离集成分支按“后端基础 → relay/计费 → 任务插件 → 认证 → Default → Classic → WebSocket”顺序逐层合入。
4. 每层合入后运行受影响测试；所有层完成后再运行根 Go 测试、独立 relaykit 构建、Default/Classic 前端检查。
5. 未完成全量验证前，不创建正式 PR、不部署、不修改 `custom-main`。

## 5. 统一验证要求

- `git diff --check`，确认没有意外生成物和未解释的本地改动。
- 根模块：`go test ./... -timeout=60s`。
- relaykit：`cd relaykit; $env:GOWORK='off'; go build ./...`，并运行其定向测试。
- Default：在 `web/` 使用 Bun 执行 lint、typecheck、test、build。
- Classic：按 `web/AGENTS.md` 执行对应 lint、test、build。
- 数据库：核对 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+ 的迁移和 SQL 差异。
- 运行时：只在明确授权后，先 zzapi 验证；本计划不包含生产部署。

## 6. 当前阻断项

- 上游任务插件是架构替换，不能直接覆盖本地 AtlasCloud 和源任务授权。
- **P0 渠道编号冲突**：本地 `ChannelTypeAtlasCloud = 61`，上游 rc.37 将 `61` 用作 `ChannelTypeTaskPlugin`。必须先设计新编号或显式数据迁移，并覆盖历史渠道、任务平台、模型发现、图片/视频适配器和双前端枚举。
- **P0 钱包契约冲突**：本地钱包使用 `int64/BIGINT`，上游使用 JavaScript 安全整数上限 `2^53-1`。必须整体保留钱包迁移、单请求 int32 边界、任务退款与崩溃恢复，不得只复制上游上限常量。
- **P0 基线分叉**：集成前必须以最新 `origin/custom-main` 重新固定本地功能基线，解释本地独有提交后再重新计算冲突；旧审查分支或旧计划的“已覆盖”状态不能直接复用。
- **P0 会话准入**：新登录 flow 必须继续接入本地事务内会话限额、最旧会话淘汰、AuthVersion 重验和 Redis deny fence，不能仅采用上游新建会话函数。
- **P0 双前端认证切换**：Telegram 410、`require_verification`、一分钟 session-bound proof、Passkey/访问令牌/敏感操作必须与 Classic 同批适配；Telegram 通知 Bot 投递单独保留。
- **P0/P1 审计与授权**：插件、任务和 Responses 路由必须接入本地实际选渠后的提示词审计、重试、亲和性、失败指标、日志脱敏和 Root 门禁；上游独立 audit 表不能替代既有契约。
- Telegram 旧登录接口存在明确 410 兼容断点。
- WebSocket 同时修改 HTTP 公共计费/usage 链路，不能作为孤立功能合入。
- 上游大规模 Default 前端变更与本地二开文件重叠，必须按功能域逐段解决。

## 7. 审查结论与新的分层门禁

集成审查结论为 **blocked**：当前不适合整体合并。建议以 rc.37 为主体，选择性补入 main 的数据库、授权和 Passkey 修复；Responses WebSocket 独立放在最后。关闭 WebSocket 开关不能替代 HTTP/SSE usage、重试、结算回归。

推荐顺序：

```text
L0 重新固定 origin/custom-main 功能基线
 → L1 渠道编号、钱包、Task 字段、JSON/relaykit、三库迁移
 → L2 授权资源、审计、原子会话准入、公共安全与计费边界
    ├→ L3 认证/Telegram/Passkey/proof + Default/Classic
    └→ L4 JS Task Plugin + AtlasCloud/历史任务 + Default/Classic
 → L5 表达式、usage、HTTP/SSE 错误与结算
 → L6 Responses WebSocket 完整生命周期
 → 全量集成验证
```

共享的 `controller/relay.go`、`model/task.go`、路由、计费和迁移文件必须由集成责任人串行收口；L3 与 L4 只能在不触碰同一共享文件的前提下并行准备。

## 8. 记录格式

每个 Agent 返回：基线 SHA、Paseo workspace/分支、修改文件、保留的本地契约、测试命令与结果、未验证风险、是否建议进入下一层集成。协调者独立复核 Agent 报告，不把 Agent 的“完成”视为合并证明。

## 9. 首轮分发结果（2026-09-15）

八个 Paseo Agent 均以 `codex/gpt-6-astra`、`thinkingOptionId=high` 启动。
本轮结束不代表上游更新完成；目前仅 Default 有已推送的可审查提交。
下面测试结果来自各 Agent 交付记录，协调者尚未重新执行集成测试。

| 任务          | Agent ID                               | 实际交付与剩余项                                                                                                                             |
| ------------- | -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| 后端数据库    | `8e079750-1181-4920-bdce-ce930f2e6c0d` | 报告会话无工具，未实现、未验证，需重新派发                                                                                                   |
| Relay 计费    | `5b6b8efc-ade0-4799-99a9-5ebfcd71085c` | 报告会话无工具，未实现、未验证，需重新派发                                                                                                   |
| 任务插件      | `80643d44-b794-4161-afa8-39a76aeaa62a` | 未提交的迁移基础与 Remix 倍率、AtlasCloud 时长修复；定向测试及 vet 通过；在线仍走原生适配器，JS 全链路及实库未验证                           |
| 认证 Telegram | `5f7d74ba-b4cb-4733-bbf5-c79df2fbb23f` | 未提交的会话版本、AuthVersion、Argon2id 读取及 PAT 兼容补丁；六包定向测试、vet、路由编译通过；未切 OAuth/410 与新 proof 协议                 |
| Default 前端  | `d5b79b15-3ae1-431a-8f6b-6bafbf1b79d8` | 已推送 `agent/upstream-default-frontend@e6ccc86e6`；496 项测试、typecheck、build、25 文件 lint 通过；完整 lint 未通过，报告有 293 条既有错误 |
| Classic 契约  | `5bc35f88-9053-406b-8de2-155326defffe` | 仅源码审查回报，缺少可验收的逐项契约产物与构建验证；不得标为 Classic 适配完成                                                                |
| Responses WS  | `c75d5f9c-57d7-4ad6-99fa-2da4300031b5` | 未提交的 DTO 准备；独立 relaykit 构建与定向测试通过；未注册 WS 入口，运行时仍被审计、租约、usage、退款与重试差异阻断                         |
| 集成审查      | `bd250950-0f5f-4f2b-96fc-770dcef9cc04` | 只读结论 blocked；其部分比较基于旧本地 `353352428`，具体结论需在 `45ab82100` 复核，不能替代协调者的 185/187 文件合并预演                     |

### Default 已核对的交付边界

协调者已核对提交、工作区和远端分支：`e6ccc86e65c6d5e186f5610f58969cead2eacbaa`。
相对 `45ab82100` 共 40 个文件，仅涉及 `web/src/` 和文档，未包含后端、Classic 或部署变更。
Agent 工作区干净。测试记录已读取，尚未由协调者独立重跑。

- 包含：多 Key 策略编辑、供应商展示识别、时间价格规则保护、日志防自动填充、额度详情、兑换码导出、Passkey 能力检测和七语种文案。
- 不包含：完整插件后台、模型供应商管理重构、新安全中心、多域 Passkey 后端、审计接口、任务附件和 WS 配置入口。
- 继续使用现有 API；没有新增跨项目模型 ID 或计费数据字段。多 Key 策略沿用 `multi_key_mode`。
- 交付记录位于该分支的 `docs/workflows/2026-09/15_default_upstream_integration.md`。

### 下一轮必须完成

1. 给数据库与 Relay 任务恢复可用工具并重派，先落实 L1/L2 契约。
2. 复核任务插件、认证与 WS 分支的未提交差异及实际基线，明确共享文件唯一所有者后再组合。
3. Classic 必须产出逐 API/页面兼容矩阵并完成实际适配；其他 Agent 已通过临时工具缓存使用 Bun，不能将 PATH 中没有 Bun 当作永久阻断。
4. 对 Default 提交做独立语义审查；完整 lint 失败需核对基线，不能以定向 lint 代替全量通过。
5. 收齐后端契约后再推进依赖的前端功能，最后在同一集成结果上运行测试。各分支单独通过不等于组合通过。

## 10. 续办与提交责任（2026-09-15）

用户要求继续跟进原 Agent 至提交、推送、PR，并明确 Classic 也必须实现。
已向全部八个原 Agent 发出续办任务，统一保持 `gpt-6-astra/high`。
实现 Agent 以本阶段可独立验收的范围提 PR；完整上游迁移未完成不得宣称完成。
未达到就绪标准的阶段标为草稿，记录阻断，不自动合并或部署。

- Default：对已推送的 `e6ccc86e6` 补齐独立审查、基线核对及 PR。
- Classic：从只读审查改为实际实现；第一阶段逐项对齐 Default 已交付兼容功能，保留 Classic 原有契约，完成定向验证和 PR。
- 任务插件、认证、WS：保留现有未提交改动，先同步基线、排除无关历史，再提交、推送并创建范围准确的 PR。
- 集成审查：保持只读，先复核 Default 提交与更新后的基线，再审查各 PR 和共享文件重叠。

### 无工具会话的任务交接

数据库和 Relay 的两个原会话续办后仍报告没有工具。
新建接替会话 `81584a32-e678-499a-80f2-3802a9ea4242`（数据库）和
`692f3224-e2d4-4989-874d-ceaaff4ea36b`（Relay）也失败，均未产生代码。
Paseo 状态显示 full-access、无待审批，但没有证据证明工具已实际传入模型，根因尚未确定。
停止反复重建，将任务交给已有真实工具调用与测试记录的会话串行接手：

| 后续任务            | 唯一接手 Agent                         | 前置任务                    | 后续专属工作区           |
| ------------------- | -------------------------------------- | --------------------------- | ------------------------ |
| 数据库兼容          | `5f7d74ba-b4cb-4733-bbf5-c79df2fbb23f` | 认证兼容补丁提交、推送与 PR | `upstream-backend-db`    |
| HTTP/SSE Relay 计费 | `c75d5f9c-57d7-4ad6-99fa-2da4300031b5` | WS DTO 准备提交、推送与 PR  | `upstream-relay-billing` |

两份交接均已由 Paseo 接受；接手 Agent 当时处于 running。
后续任务使用独立工作区与分支，不混入前置 PR；保留旧会话记录，不再唤醒其写入。
创建/更新 PR 前重新 fetch 并验证基线；旧 `353352428` 的分叉与本地独有提交必须解释清楚。

## 11. 独立审查与 PR 实际状态（2026-09-15）

协调者通过 `gh pr view` 实时核实：

- [认证兼容草稿 PR #210](https://github.com/moeacgx/maolaonewapi/pull/210)：OPEN、draft，head `e26facb03a57670945f866771b22ece69d6c3671`，base `custom-main`。
- [WS DTO 准备草稿 PR #211](https://github.com/moeacgx/maolaonewapi/pull/211)：OPEN、draft，head `d06cf6e901274f987b37ffbf56d5d8579683f5a0`，base `custom-main`。
- Default `e6ccc86e6` 独立审查为 approve，没有新增 P0/P1；当次查询仍无对应 PR，已通知原 Agent 补齐验证记录后创建。

Default 审查者独立核对时间表达式往返和旧 OR 规则拒绝、翻译并集、实际差异及 lint 基线。
Default lint 为 309→293，完整扫描为 1470→1454，新增错误 0、消除 16。
审查工具是 Bun 1.3.13 与 oxlint 1.74.0，与作者 Bun 1.4.2 的 496 项测试记录分开标注。
审查者未重跑作者全部测试，也未验证真实浏览器和 Passkey 硬件。

审查者确认旧三个独有合并提交的文件树与 `cc8fd2bbe` 相同，Default 最终差异没有夹带无关业务变化。
这项证据仅说明已检查范围，不免除其他分支同步后的差异复核。
原集成审查 Agent 已获派固定提交的 #210/#211 独立评审；两项草稿不能因有 PR 就标为已验收。
整体上游迁移和 WS 运行时仍未完成，所有 PR 均未获本任务主线合并或部署授权。

### Default 正式 PR 已交付

协调者已通过 GitHub 核实 [PR #212](https://github.com/moeacgx/maolaonewapi/pull/212)：OPEN、非草稿，
目标 `custom-main`，head `8271c6ba2f5230fe07f24166927d0a363465965f`。
作者报告业务代码保持已审查的 `e6ccc86e6`，后续仅补验证脚本和文档；本轮 typecheck/build 通过，
496 项测试为上一轮结果，本轮复跑未全部完成，不重复宣称完整测试通过。
协调者查询时 `pr-quality` 通过，后端及前端 CI 均运行中；当前等待 CI 与最终验收，未合并。
作者已将 Classic 等价清单和 Bun 路径交给原 Classic Agent，Classic 仍为单独实现与验证任务。

### 认证、WS DTO 与任务修复审查进展

- #210 `e26facb03a576`：独立审查 approve，无必须修复的 P0/P1/P2；协调者实时核实前后端 CI 和 PR 质量检查通过。保留旧认证接口是本阶段边界，不是完整 OAuth 迁移完成。
- #211 `d06cf6e901274`：独立审查 request_changes，P2 为 `cloneOpenAIUsage` 未深拷贝新增的 `OutputTokensDetails`，源对象修改会污染 BillingUsage 快照。旧 head CI 虽通过，但没有覆盖此隔离契约。已要求原 WS Agent 最小修复并增加构造、再次克隆的双向修改隔离测试，更新原 PR 后复审。
- [任务修复草稿 PR #213](https://github.com/moeacgx/maolaonewapi/pull/213)：协调者核实 OPEN、draft，head `987502c1f6a515fb8fb66ecc5cb2519617075c91`，base `custom-main`。仅 9 个文件，包含 Remix 倍率恢复、AtlasCloud 时长校验、测试和文档；未引入 JS 引擎、插件包或依赖。查询时 PR 质量通过，前后端 CI 运行中。已交独立审查 Agent 评审这两项修复。

数据库阶段允许以 SQLite 实测与三库源码核对交付独立草稿 PR，必须标明 MySQL/PostgreSQL 实库未验收和补验步骤；不因此调用生产环境或要求提供生产凭据。

### 任务修复 #213 独立验收结论

独立审查对 `987502c1f6a515fb8fb66ecc5cb2519617075c91` 给出 approve，无必须修复的 P0/P1/P2。
协调者再次核对 head 未变化，前后端 CI、PR 质量检查均成功；PR 仍为 OPEN、draft，未合并。

- Remix 的倍率使用独立 map 快照，在重建价格后按键恢复；连续提交测试与调用链核对未发现重试累乘。
- AtlasCloud 在预扣前统一时长，优先级为 `metadata.duration > duration > seconds`；零不覆盖正值，非法、负数、非整数及超过 3600 秒返回 400，构建上游请求时不再被 metadata 覆盖。
- 授权、渠道能力检查先于快照保存；保留 AtlasCloud 61、既有退款与轮询实现。
- 审查者未本地重跑测试、未验证真实供应商和实库；两次提交测试不等于完整控制器失败重试演练。

本次验收只覆盖 9 文件的独立修复，不代表 JS 插件运行时、持久化版本或 Default/Classic 插件后台已实现。

### WS DTO 修复与 Relay 交付续办

#211 已推送修复后的 `4679c18c4a89dbc6a1715221eb97170a98c9b7fc`。
作者报告已补齐输出明细深拷贝、构造和克隆双向隔离、缺省/null 测试；协调者通过 GitHub 核实新 head 与前后端 CI 成功，已派独立聚焦复审，尚未收到复审结论。
WS 路由仍未注册，不把 DTO 测试通过等同于运行时可用。

同一 Agent 接手的 Relay 阶段继续独立提交、推送和创建草稿 PR；该操作已有用户授权，无需重复确认。
需保留原生响应本地文本回退、稀疏 usage 已有缓存详情及显式零语义，具体结果以其后续提交和测试证据为准。

### 数据库 PR 与 WS 复核证据

- [数据库草稿 PR #214](https://github.com/moeacgx/maolaonewapi/pull/214) 已由协调者通过 GitHub 核实，head `fd52b3402d9c9d777e960540c9d738e4d1487eb8`、base `custom-main`。8 文件涉及 SQLite DSN、PostgreSQL 旧预填分组索引迁移、点号标识符引用防误删及测试文档；查询时 PR 质量通过、前后端 CI 运行中。作者报告 common/model 测试、vet、根编译通过；PG/MySQL 实库仍未验收。已交任务 Agent 独立只读评审。
- #211 的复审 Agent 回报 approve，但其回报中的命令是文本、没有工具结果，不能单独作为执行证据。协调者实际读取固定 `d06cf6e901..4679c18c4` 差异，确认新增非 nil 分支复制输出明细值、赋予独立指针；该明细只有整数成员。测试同时覆盖 Responses/Chat 构造、源/快照/克隆的双向修改隔离和缺省/null/零对象保留；本次仅 3 文件变动、差异检查通过。GitHub 当前 head 的前后端 CI 成功。

因此 #211 此前快照别名问题已在源码和回归覆盖层面解决，DTO 准备范围静态复核通过；协调者没有本地重跑测试，不宣称验证 WS 运行时。

### #214 由协调者完成实际审查

被委派的任务 Agent 因续办会话缺少工具返回 blocked，没有形成审查证据。
协调者直接读取固定 `fd52b3402d9c9d777e960540c9d738e4d1487eb8` 的实现、测试与迁移文档，
并在该提交且干净的数据库工作区执行：

```powershell
$env:GOWORK='off'
go test ./model -run '^(TestPrefillMigrationQuotesLiteralObjectNames|TestMigratePrefillGroupUniquenessSQLite|TestSQLiteDefaultConnectionWriteIsolation)$' -count=1 -timeout=60s
```

实际结果通过（model 包 0.207 秒），覆盖 SQL 名称引用、SQLite 迁移无副作用、WAL 读写与立即事务。
协调者同时核实固定提交的前后端 CI 和 PR 质量检查全部成功，差异检查通过。

审查要点：

- PostgreSQL 专用分支在其他数据库直接返回，不替换 MySQL/SQLite 既有预填分组结构。
- 按目标表 OID 与唯一对象定义识别旧约束；排除主键、复合、表达式及部分索引。
- DROP 名称按目录解析出的实际 schema 限定并逐段转义，字面点号不成为跨 schema 路径；不使用 CASCADE。
- 排他事务内二次读取冲突，新增软删除列、删除旧对象、建立并核验目标部分索引；返回错误使事务回滚。
- WAL/忙等待/立即事务只改变默认 DSN，自定义路径仍显式配置，钱包实现没有改变。

结论：限定阶段的代码审查通过，未发现必须修复项；PostgreSQL/MySQL 实库未运行，
实际 DDL、并发锁与回滚验收仍待补，不能把引用 DryRun 或 CI 通过视为三库实测。
保留 #214 草稿及实库验收说明，不据此进行主线合并或生产部署。

### Relay #215 已交付及协调者验证

[Relay 草稿 PR #215](https://github.com/moeacgx/maolaonewapi/pull/215) 已提交推送，协调者核实
head `06b78e2fd764b6e9c656909c9ddba11ae80624d7`、base `custom-main`，仅 7 个文件。
本阶段归一 HTTP/SSE 已支持的 usage 分类详情，保留本地文本回退、缓存稀疏更新及显式零值，
不引入 WS 路由或 #211 的 DTO 字段。原生网络响应不采用内部 BillingUsage 等转换元数据，
转换 API 继续保留或创建独立计费快照。

协调者直接读取固定差异与新增测试，并在干净的该提交工作区独立执行：

```powershell
$env:GOWORK='off'
$env:GOARCH='amd64'
# 仓库根目录：HTTP/SSE 详情、文本回退与稀疏缓存回归
go test ./relay/channel/openai -run '^(TestOaiResponsesPreservesDetailsForExpressionBilling|TestOaiResponsesNativeUsageIgnoresWireBillingMetadata|TestOaiResponsesSparseTerminalRetainsCacheDetails)$' -count=1 -timeout=60s
# relaykit 目录：原生/转换快照边界和显式零存在性
go test ./relayconvert -run '^TestNormalizeResponsesUsage' -count=1 -timeout=60s
go build ./...
```

三项均实际通过，根定向包 0.163 秒、relayconvert 0.591 秒，固定差异检查通过。
限定范围未发现必须修复项；查询时前后端 CI 仍在运行、PR 质量检查成功，不能提前记为 CI 全绿。
标准 `output_tokens_details` 完整映射仍需与 #211 显式组合，模型修饰符、完整多块归并及其余上游迁移尚未完成。
本地定向验证不代表真实供应商账务、完整控制器重试或 Default/Classic 最终组合验收。

## 12. 第一阶段合并验收（2026-09-15）

用户已明确授权合并 #210–#216。此次只合入各 PR 已交付的独立兼容补丁，
不以此宣称官方上游 main 已完整合入；不发 tag、不发布镜像、不部署容器。
本文随最后的 Classic PR 一起提交，使更新计划、分发记录及实际验收进入仓库。

| PR                                                       | 本阶段内容                       | 验收结论                                                            |
| -------------------------------------------------------- | -------------------------------- | ------------------------------------------------------------------- |
| [#210](https://github.com/moeacgx/maolaonewapi/pull/210) | 会话版本、Passkey/PAT 与密码兼容 | 独立审查、CI 通过                                                   |
| [#211](https://github.com/moeacgx/maolaonewapi/pull/211) | WS DTO 与输出详情快照深拷贝      | 问题修复后复核、独立 relaykit 验证通过，未启用运行时                |
| [#212](https://github.com/moeacgx/maolaonewapi/pull/212) | Default 兼容功能                 | 独立审查通过，合并阶段修复两个测试逐字输入超时                      |
| [#213](https://github.com/moeacgx/maolaonewapi/pull/213) | Remix 倍率与 AtlasCloud 时长     | 独立审查和任务回归通过，不含 JS 插件切换                            |
| [#214](https://github.com/moeacgx/maolaonewapi/pull/214) | SQLite 连接与 PG 旧索引迁移      | 协调者审查及 SQLite 实测通过，PG/MySQL 实库未验收                   |
| [#215](https://github.com/moeacgx/maolaonewapi/pull/215) | 原生 Responses HTTP/SSE usage    | 协调者审查、定向回归、独立 relaykit 验证通过                        |
| [#216](https://github.com/moeacgx/maolaonewapi/pull/216) | Classic 等价兼容及更新记录       | 协调者核对权限、价格、导出与模式切换，321 合同测试及 9 组件测试通过 |

### 合并方法与组合验证

- 按 #210→#211→#212→#213→#214→#215→#216 顺序推进；每项合并前以最新远端 `custom-main` 同步，保留双方开发文档条目，不整文件取一边。
- Git HTTPS 一度发生 DNS/连接故障，GitHub API 可用；使用 Git Data API 上传已验证的文件树。逐次校验远端树 SHA 与本地一致，分支只快进，保留旧本地 HEAD 的备份指针；不是绕过 CI 或改写远端历史。
- 七项程序补丁在 Classic 工作区组合验证，`GOWORK=off`、`GOARCH=amd64` 的根 `go test ./... -p 2 -count=1 -timeout=60s` 通过。
- 独立 `relaykit` 的 `go build ./...` 和 `go test ./... -count=1 -timeout=60s` 通过，未依赖根工作区构建。
- Default 与 Classic 生产构建均通过；Classic 11 项原生扩展测试通过。
- Classic 321 项合同测试通过。组件测试在线程模式达到 60 秒进程上限，切为 `--pool=forks --poolOptions.forks.singleFork=true` 后 6 文件、9 项通过（20.78 秒）；没有通过提高超时掩盖失败。
- Default 多 Key 用例 CI 曾有 2 项逐字输入超过 5 秒，改为点击并粘贴测试密钥，继续检查真实策略控件和 PUT 载荷。定向 6 项、lint、类型检查通过；更新后的 #212 CI 全量 496 项通过。
- 每项变更后的 PR 重新运行 CI。最终主分支及对应合并提交状态以各 PR 链接为准，旧 head 的成功不能替代新 head。

未验收边界仍包括真实供应商、Telegram/WebAuthn 硬件、PG/MySQL 实库和生产流量。
完整任务插件、统一认证/OAuth 切换、WS 生命周期、标准输出详情完整映射及其余上游功能留在后续阶段。
历史全量 lint 问题保留已核实的基线边界，不宣称全仓 lint 零错误。
