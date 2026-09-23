# NewAPI-Mao 二次开发能力

项目名称、默认仓库和镜像规则见[项目身份与发布兼容](project-identity.md)。本次更名无新增业务功能或数据迁移；旧版自更新资产名和配套仓库保持兼容，首次新镜像发布仍须验收。

## Classic 供应商自定义 Logo

- 在供应商图标字段填写 HTTP/HTTPS 图片直链，或继续使用 LobeHub 图标名和链式参数。
- 覆盖 Classic 供应商、模型、渠道选择和模型广场；图片加载失败时显示默认图标。
- 复用现有 `icon` 字段，最长 128 字符，无数据迁移；Default 未实现 URL 图标渲染。
- 实现与验证边界见 [Classic 供应商自定义 Logo URL](../workflows/2026-09/23_classic_vendor_logo_url.md)。

## xAI 渠道 Messages 兼容

- `/v1/messages` 经 xAI 渠道转为 Chat Completions，上游返回转换回 Anthropic JSON/SSE，保留 Grok 缓存用量和现有结算口径。
- 强制 Responses、请求体透传及原生端点沿用既有优先级；无需新增开关或数据库迁移。
- 实施与验证状态见[工作记录](../workflows/2026-09/21_xai_messages_compat.md)，实际供应商兼容性和上线需单独验收。

## 账号绑定身份保持

- 邮箱验证码绑定仅更新当前现存用户的 email，保留并发权限、状态、分组和资金修改；验证码匹配后一次性消费，不创建用户或签发新会话。
- Classic 社交绑定使用明确的 bind flow，个人资料回读保留同一用户的会话凭证；Default 邮箱补齐 Turnstile，Classic 邮箱使用参数编码与失败恢复。
- 实现、双模板差异、测试和现场核验边界见[工作记录](../workflows/2026-09/21_account_binding_identity.md)。不自动合并历史账号，不改变数据库 schema；验证码仍为进程内存储。

## 复制令牌密钥独立限流

- 单条和批量读取密钥按认证用户共用独立额度，默认每分钟 60 次请求，避免同出口登录等关键操作导致首次复制 429。
- 继续执行 GA、UserAuth、密钥归属和批量上限；Default、Classic 无需修改，无数据库迁移。配置与验证见[工作记录](../workflows/2026-09/21_token_key_copy_shared_limit.md)。

## 主程序 Codex 响应头模型校验

- 主程序读取成功上游 HTTP 响应的 `x-codex-safety-buffering-faster-model`，与正文模型独立校验；即使正文匹配，头部不匹配仍按现有白名单、连续容错和关渠规则处理。
- 来源记录与通知区分正文和头部声明；单独的缓冲标志、turn-state 长度和 Plan 不触发关渠。现有 0.2.1 模块无需升级，能力通过主程序更新生效。
- 记录：[Codex 响应头证据](../workflows/2026-09/21_model_guard_codex_header_evidence.md)。中间代理未透传头部时无法识别该信号；不能据此证明底层模型身份。

## 上游模型校验渠道 ID 白名单

- `upstream-model-guard` 模块 `0.2.1` 在 Default、Classic 支持直接填写渠道 ID，与现有勾选白名单同步；沿用 `excluded_channel_ids` 与 Root 权限。
- 记录：[白名单 ID 输入](../workflows/2026-09/21_model_guard_allowlist_ids.md)。外置模块需单独安装，宿主更新不会自动升级旧模块。

## 渠道强制使用 Responses 上游

- 文档：[渠道强制使用 Responses 上游](channel-force-responses.md)。
- 配置：`setting.force_responses` 默认关闭，要求渠道敏感写权限，Default 与 Classic 均提供入口。
- 能力：支持 OpenAI、Azure、xAI、Codex、Sub2API、NewAPI 渠道，将 Chat/Claude/Gemini 对话转为 Responses，客户端返回协议不变。
- 边界：开启后优先于透传与 Responses 转 Chat；其他业务端点不改路由，无数据库迁移。
- 稳定性：复用既有协议转换和结算链路；供应商实际兼容性与线上部署需单独验证。

## 认证上游兼容阶段

- 文档：[认证差异与兼容矩阵](../workflows/2026-09/15_auth_upstream_compatibility.md)。
- 已实现：Telegram/Passkey 在途流程固定会话版本、登录签发固定认证快照版本、Argon2id 有界读取、PAT POST 生成与 DELETE 撤销及指纹审计。
- 稳定性：后端兼容增量；保留本地 JWT、Session、刷新轮换和撤销栅栏。尚未切换 Telegram OAuth、统一多因子登录、操作上下文单次 proof 或独立审计存储；不能当作上游安全功能全部移植。

本页登记可复用的二次开发能力及其稳定性边界。长期专题文档负责接口和行为契约，
`docs/workflows/` 负责单次问题的根因、变更和验证记录。

## Default 上游兼容增强

- 文档：[Default 上游集成](../workflows/2026-09/15_default_upstream_integration.md)。
- 稳定性：按功能片段复用现有接口，新增渠道多 Key 策略编辑、统一供应商识别、用户额度详情和兑换码可选导出；修复时间规则、价格单位和 Passkey 能力检测。
- 权限和生命周期：多 Key 策略要求敏感写权限；兑换码导出只使用本次创建响应，在浏览器内生成文件，默认不下载，关闭后清空导出状态。
- 兼容边界：保留 ApiPanelWatch、通知、实际响应模型、稳定分组标识和本地定价展示。Classic 不在范围内；新任务插件、供应商版本化管理和安全中心需后端协同。
- 已知限制：不迁移既有计费表达式；真实 Passkey 硬件和生产页面交互未验证。

## TokensPro overview 渠道并发对齐

- 文档：[TokensPro overview 渠道并发对齐](tokenspro-overview-concurrency.md)
- 稳定性：复用现有渠道 Key、代理、系统任务租约和自动禁用状态；不新增协议类型。
- 权限与默认值：渠道编辑保存 `settings.tokenspro_overview_sync_enabled`；旧渠道默认关闭。
- 边界：只写 `concurrency.allowed`；失败不改并发和状态；`allowed=0` 因 NewAPI `0=不限制` 会自动禁用，恢复时不打开人工禁用渠道。
- 关闭任务：`TOKENSPRO_OVERVIEW_SYNC_TASK_ENABLED=false`。

## Classic 渠道编辑密钥聚合

- 文档：[Classic 编辑渠道启用密钥聚合](../workflows/2026-09/09_classic_channel_multikey_conversion.md)
- 稳定性：复用现有多密钥调度、管理和持久化；通过显式转换字段原位升级单密钥渠道。
- 权限与默认值：要求渠道敏感写权限；默认保留旧密钥并追加，支持显式覆盖及随机/轮询。
- 边界：本期只提供 Classic 编辑入口；Codex、Vertex 与 io.net 托管渠道不提供转换。
  不支持反向转换；并发编辑沿用现有渠道更新语义，应避免同时编辑同一渠道。

## 任务插件迁移

- 文档：[任务插件迁移](task-plugin-migration.md)。
- 稳定性：原生任务计费修复与迁移分析阶段；在线任务继续使用原生适配器，本补丁不引入 JS 运行时或依赖。
- 契约：AtlasCloud 渠道类型 61、源任务多分组授权与别名/继承、退款认领及补偿状态保留。
- 限制：完整 JS 路由、插件管理、资源签名和插件用量计费尚未接入，禁止直接全量切换。

## 官方 Task Plugin 并存接入

- 文档：[官方 Task Plugin 接入计划](../workflows/2026-09/15_official_task_plugin_integration_plan.md)。
- 范围：接入官方 JS 插件运行时、内置插件、插件管理和市场；现有 `/extensions` 二开扩展与 Go 原生任务适配器继续保留。
- 默认值：插件总开关默认关闭；体验阶段只在 zzapi 显式启用，不触碰 maolaoapi。
- 编号：AtlasCloud 的渠道类型 61 保留，官方插件使用 62，渠道绑定为 `setting.task_plugin_key`。
- 当前合同：内置与自定义插件列表、详情、版本、Root 上传/激活/启停/删除；专用 runtime/status 管理总开关。支持多个插件源，默认官方源与独立公开的 `moeacgx/maolaonewapi-plugins`。安装后需显式激活，关闭只阻新提交，历史任务仍按版本和 hash 轮询。
- 稳定性：内置 JSON/per_call/认证内联资源阶段已通过组合验证。多源安装与发布契约见[工作记录](../workflows/2026-09/16_task_plugin_sources_plan.md)，其部署状态以最终交付为准；S3、匿名资源签名、在线试运行、UsageFacts、真实远程媒体与完整 TokenAuth 端到端仍待验收。

## Responses WebSocket 集成准备

- 文档：[Responses WebSocket 专项集成审查](responses-websocket-integration.md)。
- 稳定性：仅 relaykit DTO 兼容准备，运行时未接入，不是可用功能或发布候选。
- 开关：渠道 `setting.responses_websocket_enabled` 缺失为 false；本分支设为 true 仍不启用入口。
- 限制：上游审计、计费错误传播、失败指标和错误替换尚未满足本地契约，禁止直接摘取整个提交。
- 范围：本阶段交付 DTO 与协议夹具，不增加界面；完整运行时启用前必须分别实现 Default 与 Classic 的开关、保存恢复和权限验收。无数据库迁移，无部署。

## 福利营销时效额度券

- 文档：[福利营销时效额度券](benefit-vouchers.md)
- 稳定性：已实现后端模型、接口、组合计费、流水和 Default/Classic 页面；流水接口与数据表
  继续用于审计，但两套福利页面不提供查看流水入口；两套模板共享 API、权限、状态和金额语义，
  组件实现与构建链路分别维护。
- 金额边界：内部 quota 是计费真值；页面按当前 USD/CNY/CUSTOM/TOKENS 展示，货币精度
  为 0.01，Tokens 为整数。领取门槛始终是 CNY 实付快照，API/表单仅按当前展示单位回显。
- 删除边界：福利活动、兑换码、优惠码均使用管理员鉴权、关键操作限流和 GORM 软删除；
  单条/批量接口返回 `deleted_ids` 与逐项 `skipped`，保留券、流水、充值、支付和审计。
  优惠码已有 payment reservation 允许通过 `Unscoped` 回调结算，删除后禁止新 reservation。
- 关键契约：福利券独立余额；显式分组才抵扣；福利券 -> 订阅 -> 钱包；请求总价仍受
  token 上限约束；所有差额和回滚按 `request_id` 幂等。组合结算补偿使用独立
  `settle_rollback` 流水和原 `request_id`/类型组合键，日志 breakdown 关联
  `activity_id`/`voucher_id`/`request_id`/`log_id`。
- 已知限制：分组并发为单实例进程内限制；活动不自动复制渠道/分组；本期不覆盖图像和
  视频异步任务；Default 前端时间输入显式按 `Asia/Shanghai` 转换为 Unix 秒。管理 HTTP
  接口使用服务端时间，忽略请求体 `now`；个人券有效期对外按小时传输，数据库内部仍按
  秒保存并兼容旧秒字段；模型层时间参数仅供内部测试注入。
- 历史清理：福利活动批量删除仅归档可安全删除的 `draft`/`ended`/`terminated` 状态；
  `published`、`paused` 或仍有 active 券的活动会跳过。兑换码、优惠码批量删除和失效清理
  沿用软删除并保留账务关联。三类资源接口均限制最多 500 个 ID，仅管理员可调用，返回
  实际删除和跳过原因；重复请求幂等。
- 迁移/回滚：额度配置迁移仅填充空字段，按历史 share quota 或旧人民币语义转换一次，
  异常即停止且不补零；回滚代码保留新表、新列和历史审计，不物理删除数据。本次 Task 10
  未部署、未推送、未开 PR，发布前仍需重新确认目标实例与回滚方案。

## 扩展模块

- 在线分发：独立公开 `maolaonewapi-extensions` 仓库维护模块源码、版本 ZIP 和完整性目录；主仓库关联扩展与任务插件仓库。
- 安装入口：Default/Classic Root 点击“在线模块”打开弹窗后读取固定目录、下载并核对包，宿主复核 hash/ID/版本后安装；保留手工 ZIP，不增加服务端远程下载或自动升级。实现与边界见[工作记录](../workflows/2026-09/18_extension_repository_marketplace.md)。

### 上游模型校验

- 增量能力：渠道白名单、同渠道跨模型/分组共享连续计数（默认 2 次）、匹配清零、每请求去重与异常计数记录。`0.2.0` 需配套宿主及数据库迁移，尚未部署。

- 文档：[上游模型校验扩展](upstream-model-guard.md)
- 能力：可上传 ZIP 独立安装的外置扩展，配置分组和模型对应关系，根据响应声明模型关闭整条出错渠道，并通过通知中心 Bot 通知；支持 Default 与 Classic。
- 展示契约：页面与通知使用分组显示名称，记录 API 以 `group_name` 提供当前名称；ID/code 保留原有匹配与提交语义。名称修复尚未重新部署。
- 稳定性：已完成本地后端与双模板验收，并在 zzapi、maolaoapi 各三节点部署 `.326.guard.1` 配套补丁、验证真实 ZIP 上传和插件表；默认关闭，尚未正式发布。
- 限制：需要包含 `channel.upstream-model-guard` 的宿主支持，旧版不能单独上传即用。只覆盖正常转发中已采集响应模型的协议，模型声明不构成真实性证明；不撤回当前响应或在途请求。多节点须逐节点开启模块。

扩展模块的宿主、权限和通知契约见 [扩展模块开发](extensions.md)。新增扩展页面时，
同时登记入口、所需角色、API 和回滚方式。

### 对话归档扩展

- 文档：[扩展模块开发](extensions.md#对话归档扩展)
- 用途：按多个分组和指定用户 ID 采集对话，清洗后在线预览，用于定位蒸馏或异常调用来源。
- 稳定性：宿主 API 与 `native v1` 页面同版本发布；未识别协议仅保存可识别的有限文本，不保证还原原始请求。
- 权限：仅 Root 可读写配置和查看详情，普通用户及普通管理员不能通过扩展路由绕过安全审计权限。
- 数据契约：筛选字段为空表示不限制；用户与分组筛选同时存在时使用 AND。分组使用稳定 `GroupCode`，用户 ID 必须为正整数并受数量上限约束。
- 安全边界：不存储 Authorization、Cookie、请求头、URL 查询、媒体和工具 schema；详情仅返回纯文本消息。配置 `CRYPTO_SECRET` 后正文以 AES-GCM 密文存储，服务端解密后再返回。
- 生命周期：正文在筛选命中后才物化，进入持久队列后按保留期和全局最近会话数清理；默认最多保留 1000 条，设置降低后立即裁剪最旧记录。Root 可二次确认清空全部归档，采集未关闭时后续匹配请求会重新写入。
- 已知限制：普通流式请求只采集认证阶段可见的请求正文；OpenAI Realtime 会在连接结束时合并文本增量，但不保存音频、工具定义或 `*.done` 汇总正文；不支持的协议不会保存完整原始 JSON。

## 安全审计上游策略来源

- 文档：[安全审计上游策略来源泛化](../workflows/2026-09/03_security_audit_policy_sources.md)
- 稳定性：内置安全审计支持将 `cyber_policy` 与 `biological_risk` 分别记录为审计来源，
  并通过 `policy_action_sources` 选择哪些来源参与当前会话屏蔽和阈值自动禁用；缺失该字段的
  旧配置按仅启用 `cyber_policy` 兼容。
- 边界：生物风险仅接受状态码 500 且包含固定上游提示的响应，普通 500 不触发；来源选择不
  改变审计事件记录开关及渠道/分组作用范围。Default 与 Classic 分别维护页面，但共享同一
  管理 API 契约。

## 通知中心

通知事件、模板变量和投递边界见 [通知中心与模块事件](notifications.md)。新增通知事件
时必须补充事件类型、权限、失败重试和敏感字段处理。

## 发票中心

发票中心支持用户选择近 30 天内符合条件的充值或订阅订单申请发票。开票服务费必须
使用已配置的外部支付方式；零服务费申请不产生实际支付。

### 开票服务费支付

- 管理员在支付设置的 `PayMethods` 中配置易支付方式。默认配置包括 `alipay`（支付宝）
  和 `wxpay`（微信）；发票中心不提供账户余额支付。
- 易支付地址、商户 ID、商户密钥和支付合规确认均满足条件时，发票配置接口返回已配置
  的易支付方式。
- Default 与 Classic 发票中心读取 `/api/user/invoice/config` 的 `pay_methods`，按
  配置展示支付选项，并将所选类型提交到 `/api/user/invoice/payment`。
- 外部支付订单先进入 `payment_pending`；易支付回调验签、金额和商户快照校验通过后
  才转为 `pending` 待开票状态。
- 事件键、支付订单号和回调处理保持幂等。未完成支付的申请不会触发待开票通知。

### 相关接口

- `GET /api/user/invoice/config`：返回发票配置、可用支付方式和支付链信息。
- `POST /api/user/invoice/preview`：计算所选订单的开票服务费。
- `POST /api/user/invoice/request`：仅用于零服务费时提交申请；正服务费请求会被拒绝。
- `POST /api/user/invoice/payment`：创建外部支付申请并返回易支付收银台参数。
- `GET|POST /api/invoice/epay/notify`：易支付异步回调。
- `GET|POST /api/invoice/epay/return`：易支付同步回跳。

### 模板边界

Default 使用 `web/src/features/invoices`，Classic 使用
`web/classic/src/components/invoice`。两套模板分别读取相同的后端配置契约，修改一套
不会自动改变另一套。Classic `/console/invoice` 使用全宽透明玻璃业务卡片，表格按容器宽度
铺开，右侧操作列固定在边缘。

## 官方 JS Task Plugin 后端

见 [官方任务插件后端](official-task-plugins.md)：内置与 Root 上传的自定义插件并存，默认关闭；
生产资源签名和 S3 尚未验收。
