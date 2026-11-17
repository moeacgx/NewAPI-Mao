# Issue #403：归档分组捕获与安全审计展示名称

## 目标与范围

修复请求归档把用户归属分组 ID 当成请求分组 ID、把内部 code 当成名称的问题。
独占修改 `service/request_archive_scope.go`、`model/prompt_audit.go` 与新专属测试；
不修改前端、共享文档、鉴权、计费或数据库结构，不提交或部署。
协调者另批准三个既有测试函数的断言适配：
`model/prompt_audit_test.go` 的 `TestPromptAuditEventChannelSnapshotRoundTripAndLegacyDefault`、
`service/prompt_audit_runtime_test.go` 的 `TestAuditPromptSnapshotBlockingStoresEncryptedEvent`、
`service/security_audit_events_test.go` 的 `TestResponseAuditEventUsesFinalRetryGroupMetadata`。
保留 ID/code/name、顺序和长度的完整相等比较，以及原始 JSON、密文、解密与重试组选取验证，
仅明确添加只读 DisplayName 预期；其他既有测试函数未修改。

## 捕获契约

- `group_id` 只代表捕获时可确定的请求分组实体；不得套用用户归属分组 ID。
- 优先使用审计上下文中的明确稳定 ID；无稳定 ID 时按请求 code/alias 解析实体。
- `group_code` 保留请求筛选使用的标识契约，`group_name` 保存捕获当时的实体名称。
  单一 code/alias 成功解析后，归档候选记录该实体当前 code，不修改认证或路由上下文。
- 普通 HTTP 归档发生在 PromptAudit/Distribute 之前；此时仅为请求意图，不能宣称已知最终路由组。
- Realtime 的审计分组上下文只在 Guard 生效时设置，缺失时不能假定已有审计实体。
- Realtime 已明确选中的不同分组优先于早先审计候选；相同标识仍保留已知稳定 ID。
- 缺失、删除或未知实体回退已有合法快照或原标识，不扫描全部分组。
- 历史归档、密文、AAD 及已经持久化的事件不回写；异步排队后保持捕获时快照。

## 安全审计响应契约（已获协调批准）

原 `group_name`、`channel_groups[].name`、`token_groups[].name` 保留事件快照。
新增只读 `group_display_name`、`channel_groups[].display_name`、
`token_groups[].display_name`，供 Default 与 Classic 展示当前名称。
当前名称按稳定 ID 优先，旧记录无 ID 时按 code/alias 批量解析；
不存在实体时回退快照名称，再回退原标识。新增字段不持久化，证据导出保留原字段语义。
列表一次批量加载，详情同样投影，不能在每个事件上遍历全分组或执行逐项查询。

协调批准回执：`msg_86483ed3eb90`；前端工人已确认接入新字段。全部新增模型字段
标记 `gorm:"-"`，嵌套展示字段在编码时从副本剥离，不修改原始数组。
有 ID 时只按 ID 查询；即使 code 被新组复用，也不允许重绑身份。仅无 ID 记录
复用 `GetGroupDisplayNameMap` 的 code/alias 兼容映射。分组名称查询失败只降级展示，
不得阻断原始审计证据的读取。列表补选原有 `group_code`，让历史 alias 能参与解析。

## 生命周期与实现边界

- HTTP：`TokenAuth` 写入 `UsingGroup`，`PromptAudit` 入口先保存原始请求，
  后执行审计范围判断，再进入 `Distribute`；不能把请求时意图描述成最终路由结果。
- Realtime：仅 `guardActive` 时设置 `ContextKeyPromptAuditGroupId/Code/Name`，
  名称可能是 `pre_allocation:*`。捕获时按实体读取真实名称；选渠后的帧可使用
  `ContextKeySelectedChannelGroup`，不继承不同候选组的 ID。
- 请求捕获按 `GetGroupsByIds` 精确 ID 查询或 `GetGroupByCodeOrAlias` 查询，
  不在热路径加载全部分组，也不引入会使改名快照滞后的新缓存。
- 审计列表汇集根分组、渠道分组、令牌分组，ID 去重并每批至多 500 个；
  老式无 ID 字符串每页至多构造一次名称映射。原事件字段、原 JSON 与正文保持不变。

## 双模板与共享契约

Default 与 Classic 共享原安全审计 List/Get 接口；前端工人 `ctx_e7accf56c9d9`
负责两套模板优先消费新字段，后端单元测试验证序列化字段和旧字段同时存在。
本工人不修改前端、不声称浏览器验收；前端运行结果由对应工人交付。
既有请求参数、路由、鉴权、计费、筛选标识均不变；只新增可选响应展示字段。

分享建议：NewAPIForDouDi 若沿用同样的归档与审计模型，适合参考稳定 ID 与名称快照
分离方案；NewAPIModifyByGang 需先确认是否存在此自定义审计模块与相同响应消费者，
不建议直接移植。未向任何兄弟仓库发送 Issue、PR 或消息。

## 验证计划与边界

使用真实本地 SQLite fixture，名称与 code 不同，覆盖 alias、用户归属分组不同、
稳定 ID 优先、改名后旧事件不变、未知/删除回退，以及历史 AAD 解密行为。
只使用 GORM 与 common JSON；不添加方言 SQL。MySQL/PostgreSQL 未实连时明确记录，
不能把 SQLite 或 SQL mock 当作三数据库运行验收。每个测试进程限时 60 秒，临时文件使用 D 盘。

## 验证结果

首次专属红测真实复现请求分组 ID 错用、code 名称、缺少展示投影；修复后
service/model 专属测试分别于 1.422 秒与 1.771 秒内通过。中间件定向回归
`RequestArchive|PromptAudit.*Group|PromptAuditRealtime` 于 6.553 秒内通过。

首次完整包级回归：model 36.260 秒、service 32.990 秒，均在 60 秒进程上限内结束。
三处本任务导致的旧结构比较失败已按协调授权适配；另观察到并行 H 工人的
`TestPromptAuditGroupValidationNames` 与 `TestGroupValidationMigrationRatioNames`
红阶段失败，未修改其文件。协调者随后确认 H 已转绿，最终全包门禁由协调统一执行。

第二次完整包级复跑均被外围 60 秒上限终止：model 已记录 871 个通过、16 个跳过，
service 已记录 1448 个通过、1 个跳过；这只是中途计数，不能作为包级通过结论。
随后并发定向复跑在编译阶段同样触及上限，没有产生任何测试结果。改为逐包串行后，
最终定向测试全部通过，计数包含子用例：

- model：14 项，1.298 秒，覆盖 5 个新专属测试和获授权的既有快照测试。
- service：20 项，1.325 秒，覆盖 3 个新专属测试、2 个获授权既有测试，
  以及原请求归档 JSON 防篡改、ra2/ra3 密文兼容与认证测试。
- middleware：34 项，3.697 秒，覆盖请求归档、审计分组和 Realtime 生命周期。

最终命令（外围 Python 子进程另设 `timeout=60`）：

```powershell
go test ./model -json -run 'Issue403PromptAudit|PromptAuditEventChannelSnapshotRoundTripAndLegacyDefault' -count=1 -timeout=55s
go test ./service -json -run 'Issue403RequestArchive|RequestArchive.*(Authenticat|Tamper|V2|V3)|AuditPromptSnapshotBlockingStoresEncryptedEvent|ResponseAuditEventUsesFinalRetryGroupMetadata' -count=1 -timeout=55s
go test ./middleware -json -run 'RequestArchive|PromptAudit.*Group|PromptAuditRealtime' -count=1 -timeout=55s
```

日志分别为 `model-targeted-serial.jsonl`、`service-targeted-serial.jsonl`、
`middleware-targeted-serial.jsonl`；历史失败、超时与红绿日志均保留在临时目录。
补充边界红测 `service-edge-red.log` 真实复现相同 code 重建误绑以及只有稳定 ID
时拼接另一使用组的错误，最终修复后均转绿。

已执行 Go 格式化、仓库现有 `oxfmt` 单文件 Markdown 格式化与检查、
本工作记录相对链接校验，以及限定改动文件的 `git diff --check`。

真实 SQLite fixture 验证 alias、稳定 ID、名称与 code 不同、改名、实际删除后复用 code、
数据库名称来源缺失回退、批量查询，以及历史归档正文解密与名称篡改的 AAD 拒绝。
只在本机 SQLite 运行，没有使用 SQL mock，也未连接真实 MySQL/PostgreSQL。
Go 实际版本 `go1.25.5 windows/386`；临时目录和完整日志位于 `D:/Temp/issue403-archive`。

安全审计专题入口 `docs/developer/prompt-security-audit.md`
在当前检出缺失，已改读现存归档与安全审计工作记录及实际源码；共享索引由协调者整合。

## 相关文档

- [开发文档索引](../../developer/README.md)
- [对话归档扩展与采集时序](../2026-09/02_conversation_archive_extension.md)
- [安全审计与归档运行时加载](../2026-09/03_security_audit_archive_runtime_load.md)
