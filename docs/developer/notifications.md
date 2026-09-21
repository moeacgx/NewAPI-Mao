# 通知中心与模块事件

通知中心只负责发送。业务模块产生事件，通知中心根据任务、Bot、接收目标和模板完成投递。模块不能读取 Bot Token，也不能覆盖接收人和模板；一个 Bot 可以被多个任务复用。

上游模型校验的 Codex 响应头异常由主程序产生，沿用现有
`extension.upstream-model-guard.channel_disabled` 事件、通知任务及模板。
其 `reason`、`comparison` 会明确标注头部来源与声明模型，旧 0.2.1 模块无需更新。
负载中的额外 `detection_source` 尚未在旧模块清单中声明为模板变量；自定义模板仍使用已有
`reason` 或 `comparison`。详见[模型校验扩展](upstream-model-guard.md)。

## 内置事件

内置事件定义必须同时提供事件值、显示名称、默认模板、变量白名单和示例负载。当前核心事件包括：

- `invoice_pending`：变量包括 `mention`、`invoice_id`、`source_type`、`source_id`、`user_id`、`title`、`total_amount`、`create_time`。
- `channel_disabled`：变量包括 `mention`、`channel_name`、`channel_id`、`status_code`、`error_code`、`error_message`、`reason`。
- `channel_enabled`：变量包括 `mention`、`channel_name`、`channel_id`。

`invoice_pending` 在发票记录事务内进入 `pending`（待开票）状态时入队，事件键为
`invoice:<invoice_id>`。单笔充值/订阅发票、零服务费的合并发票，以及
合并外部支付都会在创建时触发；服务费尚未支付的外部合并申请保持
`payment_pending`，不提前通知，待支付成功转为 `pending` 时触发一次。合并发票的
`source_type` 为 `batch`，`source_id` 为合并发票号，`total_amount` 为所有来源订单
开票金额与服务费的合计。

渠道事件的负载只提供渠道状态字段，不提供发票金额字段。模板校验和发送渲染都按事件负载执行，未知变量必须拒绝。

## 模板生命周期

模板使用 `{{variable}}` 语法，所有负载值在发送前转义为 Telegram HTML。创建和更新任务时，后端使用该事件的示例负载校验模板；异步发送时再次按实际事件负载渲染，避免绕过管理端校验的历史数据发送错误消息。

通知中心前端在切换事件类型时，如果当前模板为空或仍等于原事件默认模板，会替换为目标事件默认模板；用户已经自定义的模板会保留。

早期前端曾在从 `invoice_pending` 切换到渠道事件时保留发票默认模板。升级后的后端对核心渠道事件的空模板和这一个完整的历史默认模板做按事件归一化：渠道禁用和渠道启用分别替换为各自默认模板；发票事件仍使用包含 `total_amount` 的发票默认模板。该兼容处理覆盖保存、管理端加载和异步发送，但不会放行任意自定义的 `{{total_amount}}` 或其他未知变量。

## 渠道事件接入

渠道状态实际改变后，在业务代码中调用通知事件入队。`channel_disabled` 负载应包含渠道名称、渠道 ID、状态码、错误码、错误消息和禁用原因；`channel_enabled` 负载包含渠道名称和渠道 ID。通知表尚未迁移时，入队应静默跳过，不影响渠道状态变更。

投递由通知中心统一处理 Telegram 调用、429 重试、失败状态和历史清理。投递请求使用 `chat_id`、HTML `text`、`parse_mode: HTML` 和 `disable_web_page_preview: true`，Bot Token 不出现在响应、日志或事件负载中。

## 渠道禁用筛选与 Classic 编辑

`filter_config` 只对 `channel_disabled` 任务生效；其他事件类型的任务请求不得携带该字段。配置为空时不写入筛选 JSON。`status_codes` 是由服务端校验的 HTTP 状态码或范围字符串；`error_keywords` 会去除首尾空白并忽略空值，最多 64 项、每项最多 256 个 Unicode 字符。`prefix_dedup_seconds` 是可选的短时去重窗口，取值 1 到 86400；未设置或为 0 时不去重。

关键词匹配 `error_message` 或 `reason` 字段，多个关键词之间是 OR 关系；状态码和关键词同时填写时是 AND 关系。服务端匹配使用不区分大小写的 `strings.ToLower` 语义。

`prefix_dedup_seconds` 在入队阶段按任务生效：渠道名按第一个 `/` 分段并去掉首尾空白，取前缀（例如 `DragAPI / Codex-Plus / 0.15x` 的前缀是 `DragAPI`）。同一任务、同一前缀在窗口内只创建第一次投递，后续事件直接跳过；没有该筛选的其他任务不受影响。窗口到期后下一次匹配会重新通知。Default 与 Classic 编辑器提供「余额不足去重」预设，会填入 `预扣费额度失败`、`余额不足` 两个关键词和 300 秒前缀窗口。

Classic 通知任务编辑器使用 TextArea，每行一个报错关键词；打开已有任务时将 `error_keywords` 数组按换行回显，输入使用 `split(/\r?\n/)` 处理 LF 和 CRLF。保存时由 `normalizeNotificationFilterConfig` 去空行、trim 并按非 locale 的小写身份去重，保留首次出现的原始拼写。任务 Modal 的 class 挂在 Portal 外层，窄屏宽度规则直接约束其内层 `.semi-modal`，并对 `.semi-modal-content`、`.semi-modal-body-wrapper`、`.semi-modal-body` 和任务 body 设置盒模型收缩边界；目标卡片和输入宽度规则也均限定在该 class 下，footer 保持可达。

## 模块事件

扩展模块通过宿主声明变量白名单和默认模板，再由受信任的服务端事件入口发布事件。模块事件变量必须与声明的负载字段一致，事件 ID 使用小写字母、数字、短横线和下划线，完整事件名最多 64 个字符。模块不得把 Bot Token、Access Token 或密码放入负载。

### 上游模型不匹配

`extension.upstream-model-guard.channel_disabled` 在上游模型校验关闭整条渠道时触发。
Root 在通知任务中选择「上游模型不匹配，渠道已关闭」，复用已有 Bot、Chat ID、提及对象和模板。
这与通用 `channel_disabled` 是两个独立订阅，避免通用任务的状态码或错误关键词筛选误丢模型校验通知。

变量包括 `channel_id`、`channel_name`、`group`、`requested_model`、
`expected_upstream_models`、`actual_upstream_model`、`comparison`、`reason`、
`request_id` 和 `create_time`，以及所有模块事件共有的 `mention` 等宿主变量。
默认模板使用渠道名称、ID 和有长度上限的 `comparison`，摘要包含触发分组与三项模型对比。
完整值保存在模块触发记录中。多条允许模型过长时，通知摘要允许截断。
`0.2.0` 起增加可选变量 `consecutive_mismatches`、`failure_threshold`，比较摘要显示连续次数/阈值。
同渠道跨模型、分组共用累计，达到阈值才发通知；未达阈值只保存异常记录，白名单渠道不触发事件。

`group` 和 `comparison` 中的分组均为事件产生时的显示名称，按稳定分组 ID 在事务中解析；
名称缺失或分组删除时才回退原标识。模块触发记录的 `group` 仍保留业务代码，
展示使用记录 API 的 `group_name`，不可直接将代码填入面向用户的通知。

渠道启用状态的条件更新、触发记录和通知事件在同一事务中写入；同一轮并发命中只发送一次。
人工恢复后再次命中会产生新的事件。Telegram 发送由已有通知任务队列执行，重试不会再次修改渠道。
没有启用的任务或 Bot 时，仍保存禁用状态与触发记录；不会自行创建 Bot 或发送测试通知。

## 变更与验证

涉及事件变量、默认模板、请求字段或发送行为的改动，必须同时更新本专题文档和 `docs/workflows/YYYY-MM/` 工作记录，并覆盖保存校验、历史模板兼容、未知变量拒绝及 Telegram 请求负载测试。
