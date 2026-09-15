# 任务插件迁移

本文保留 PR #213 的原生任务修复和固定上游分析记录。后续官方插件后端已采用独立编号 62、
默认关闭、显式渠道绑定和持久版本 pin；当前接口、迁移及未验收项以
[官方 JS 任务插件后端](official-task-plugins.md) 为准。下文的“本阶段”均指 PR #213。

## 基线与交付范围

本地功能基线为 `origin/custom-main@45ab82100`；固定上游分析点为 `9fe0457ee`，
发布参考 rc.37 为 `385d2dfd1`，插件架构最初引入点为 `eb48396d5`（#7076）。
本分支交付原生任务计费修复、契约回归测试及本文迁移矩阵；在线流量继续使用原生任务适配器。
上游全量替换涉及路由、持久化、结算和管理页面，不能作为无冲突补丁直接应用。

## 设计与实施边界

- 本补丁不引入尚无在线调用者的 JS 引擎、十份插件源码、嵌入注册器、生产映射函数或 Sobek
  依赖。早期离线原型保留在本地恢复引用，避免给独立计费修复附带未接入的运行时维护负担。
- 本文映射采用明确的历史平台值到插件 key，AtlasCloud、Midjourney 和本地图片任务不映射。
  映射表示候选迁移对象，不能证明响应、计费或历史数据完全等价。
- 保持 `ChannelTypeAtlasCloud=61`、AtlasCloud 同步/任务适配器和现有路由。
  上游 `ChannelTypeTaskPlugin=61` 与其冲突；完整接入前必须另分配编号并同步导入规则。
- 保留 `ResolveOriginTask` 的用户归属、多分组 ID、别名解析、inherit/explicit/auto、
  当前渠道能力复查和完整渠道锁定。价格重建必须恢复经校验的源任务倍率，重试不得累乘。
- 保留本地轮询退款认领、失败恢复、补偿状态及日志，不覆盖导入上游同名文件。
- AtlasCloud 在预扣前校验 `duration`、`seconds`、`metadata.duration` 的整数范围
  （0 为未指定，正数不超过 3600）。保留 metadata 覆盖优先级，但把最终时长归一到
  同一个请求字段，确保预扣乘数与实际上游 `duration` 相同。

## 数据与安全契约

本阶段不迁移数据库、不修改已存平台值或渠道配置、不开放插件上传和资源签名入口。
上游引擎禁止模块导入并禁用 source map 文件读取；执行限时和并发限制不是内存硬隔离。
未来接入 JS 和上传源码前须独立复核日志脱敏、内存限制和版本锁定；本次 Go 定向测试
只验证原生任务行为，早期原型测试不作为当前 PR 的运行时验证证据。

## 测试计划

- AtlasCloud 原生提交/查询及所有既有原生平台分发。
- 源任务别名、多组令牌、继承、撤权、渠道能力和倍率继承/重试。
- 本地轮询终态 CAS、退款认领、失败恢复及钱包/订阅结算回归。
- Go 定向测试使用 `-timeout=60s`；不以离线测试声称已验证真实供应商或生产部署。

## 完整上线阻断

以下以固定上游 `9fe0457ee` 复核，`eb48396d5` 只作为架构来源，不跟随上游浮动。
这些是迁移依赖，不是本分支已经具备的在线功能。
未满足这些要求前，不切换在线任务到 JS。

### 迁移矩阵

| 本地适配器   | 历史 Task.Platform   | 渠道类型 | 上游插件 key | 本次处理                        |
| ------------ | -------------------- | -------- | ------------ | ------------------------------- |
| Suno         | suno                 | 36       | sunoapi      | 候选迁移映射，保留原生          |
| Ali          | 17                   | 17       | alibaba      | 候选迁移映射，保留原生          |
| Gemini       | 24                   | 24       | google       | 候选迁移映射，保留原生          |
| Hailuo       | 35                   | 35       | hailuo       | 候选迁移映射，保留原生          |
| Vertex       | 41                   | 41       | vertex-ai    | 候选迁移映射，保留原生          |
| Doubao       | 45 / 54              | 45 / 54  | doubao       | 候选迁移映射，保留原生          |
| Kling        | 50                   | 50       | kling        | 候选迁移映射，保留原生          |
| Jimeng       | 51                   | 51       | jimeng       | 候选迁移映射，保留原生          |
| Vidu         | 52                   | 52       | vidu         | 候选迁移映射，保留原生          |
| Sora         | 1 / 55               | 1 / 55   | sora         | 恢复 Remix 源任务倍率，保留原生 |
| AtlasCloud   | 61                   | 61       | 无           | 保留原生，修复时长预扣边界      |
| Midjourney   | mj                   | 2 / 5    | 无           | 独立链路，不迁移                |
| 本地异步图片 | canvas_image / image | 随原渠道 | 无           | 保留本地任务、退款与补偿链      |

映射依据本地 `relay/relay_adaptor.go` 和上游同文件的 `taskPluginKeys`。Suno 的渠道类型为 36，
但原生及上游历史映射都使用 `suno`；不得把数字平台 `36` 自动扩展为有效平台。
JS key 也不自动回退到原生，防止未来插件停用被隐式原生回落绕过。

### 源任务与提交链

上游 `middleware/task_plugin.go` 的 `applyOriginTaskIntent` 只检查归属、平台、
同渠道和启用状态。新 `originTaskIds` 入口必须复用本地完整分组授权和渠道能力复查，
不能只在旧 `ResolveOriginTask` 保留校验。多源任务还必须明确规范分组一致性及计费组规则。

上游把 `TaskAdaptor.DoResponse` 替换为纯解析 `ParseResponse`，由控制器在落库与
结算后返回。本地 AtlasCloud 及其他原生适配器仍在 `DoResponse` 写响应；本地
`controller/relay.go` 当前先结算、记录消费再插入任务，插入失败只记录错误。
该持久化时序必须作为独立兼容迁移处理，不能仅把 AtlasCloud factory 分支加回来。

旧 action 为 `generate/textGenerate/firstTailGenerate/referenceGenerate/remixGenerate`，
上游改用 `image_to_video/text_to_video/first_tail_to_video/reference_to_video/remix`。
未来应在适配边界规范化，避免改写历史后令旧二进制无法回滚。

### 插件源码、资源存储与签名

- 上游 `model/task_plugin.go` 保存 key/version/source/hash/active/enabled；同版本不同
  源码被拒绝。SHA-256 只用于完整性比对，不是发布者签名或来源信任证明。
- `service/task_artifact_store.go` 仅返回 disabled store；S3 配置回退至 upstream，
  不能承诺对象归档、保留期限或供应商 URL 过期后的读取能力。
- `service/task_artifact_access.go` 用 HMAC-SHA256 绑定 taskID 与 artifactKey，
  未包含有效期，也无逐对象撤销。用户禁用会阻断访问；撤销 API token 不撤销已签发 URL。
  未来须确定短期签名/撤销策略，再连同整个资源代理链接入。
- `controller/video_proxy.go` 有 SSRF、重定向与凭据清理；不能只引入签名 URL 而省略这些保护。
  外层反向代理也须避免记录 `access` 查询参数。
- 多实例 `CryptoSecret` 必须一致且稳定；轮换会撤销所有旧签名。
  `TaskPublicAddress` / `ServerAddress` 必须使用客户端可达地址，本阶段没有修改这些配置。

### 审计、轮询与计费

- `service/task_plugin_audit.go` 记录 key/version/API version/generation/request ID，
  并区分管理员与 Root 视图。这是执行归因，不能替代插件管理操作审计。
- 插件可收到渠道凭据，`console.log` 可原样进入 debug 日志；上传虽为 Root 权限，
  仍须校验可信来源并处理凭据泄露。引擎默认每插件并发 8、执行 5 秒，限制按进程累加，
  没有内存硬上限；运行时池允许模块状态跨调用保留。
- 请求 pin 只固定单次请求的 generation。上游 `relay.GetTaskAdaptor`、轮询和资源读取
  按当前 registry 再选插件，审计版本号没有参与旧版本执行解析。
  新 JS 任务必须保存 key/version/source hash 并按其读取、轮询、结算；版本策略未完成前阻断切换。
- `9fe0457ee` 的 `pkg/jsplugin/registry.go` 已将有效 key 限制为 30 字符，
  与 `Task.Platform varchar(30)` 对齐；早期 128/30 不一致风险已修复，不再列为当前阻断。
  后续引入模型和注册器时必须一并保留该校验，不能只复制仍声明较长列的模型。
- 上游 `usageSchema` / UsageFacts 表达式计费与本地 OtherRatios/按次价格不是等价配置。
  不迁移现有价格表，不改变 `pkg/billingexpr`。后续必须验证用量单位、非有限值、上限、
  预扣、补扣/退款、表达式快照和饱和审计；不能把旧倍率直接当新用量事实。
- `9fe0457ee` 的 Alibaba `convert` / `convertImage` 已按 `!== undefined` 复制
  `seed` / `prompt_extend`，不再删除 falsy 值；早期 `seed=0` / 显式 `false` 丢失问题
  已修复。正式接入仍需保留这些边界测试，不能据单个插件推断全部供应商等价。
- 禁止覆盖本地 `sweepUnrefundedFailedTasks`、`refundTaskQuotaWithClaim`、
  `ClaimQuotaForRefund`、`RestoreQuotaAfterFailedRefund`、图片资金退款事务及
  `RefundReconciliationState`/索引。上游同名文件会删除这些恢复契约。

### 分阶段推进与回滚

1. 本阶段：固定上游分析点、文档迁移映射与原生契约测试；在线默认原生，修复已复现的计费缺口。
2. 下一阶段：独立实现原生/插件显式选择，给插件分配不冲突编号；统一每个源任务授权，
   处理纯解析与持久化响应时序，持久化插件版本并保留本地退款链。
3. 接入阶段：完成安全资源代理、签名生命周期、审计、用量计费和三数据库验证后，
   再逐供应商比对请求、响应、历史任务和零值语义。Default/Classic 管理入口分别验收。
4. 切换阶段：先停止新 JS 提交并排空对应任务，再切回原生。上游关闭 override 只回到
   factory；关闭 master 也不自动恢复原生，且可能阻断旧 JS 任务轮询和资源读取。
   不能据活跃插件删除保护或仅 type=TaskPlugin 的启用渠道计数判断所有任务已排空。

本阶段回滚只需撤销本工作项的代码改动，无数据库逆迁移或配置恢复；但撤销计费修复会
重新引入 Remix 倍率丢失和 AtlasCloud 时长错配。未来产生 JS key 任务后，旧二进制不识别
这些平台，必须保留按版本执行的兼容轮询器，或在排空后回退，保留审计/源码和资源数据。

## 验证与交付记录

见 [本次实施记录](../workflows/2026-09/15_task_plugin_migration.md)。

### Classic 与 Default 必做依赖

本阶段没有新增字段、action、路由、插件编号或管理接口，因此不改两套前端。
Classic 的 `constants/channel.constants.js` 已将 61 映射为 AtlasCloud；
`hooks/task-logs/useTaskLogsData.js` 使用 `/api/task/` 与 `/api/task/self`，
`TaskLogsColumnDefs.jsx` 继续展示原生 action、组名称和公开任务 ID。
此次 `quota` 数值修复由既有后端接口返回，不需要新增前端计算或配置。

完整切换时 Classic 必须实现插件列表/上传/启停/版本、渠道绑定、任务附件和用量详情，
与 Default 共同对接已确定的后端权限、编号和版本合同，不能只移植 Default。
按协调计划，两套 UI 由各自前端负责人实施，本任务负责提供后端合同；当前阶段未实现
这些 UI，不可标为 Classic 或全量插件集成完成。完整迁移保持未就绪。
