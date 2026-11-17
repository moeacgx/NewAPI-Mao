# 订阅降级分组删除引用保护

## 目标与范围

独立审查在 HEAD 发现 P2：分组删除门禁没有统计套餐和用户订阅的
`DowngradeGroup`，订阅到期可能将用户切换到已经删除的分组。
本项以真实 SQLite 和业务保存入口复现，再做最小修复。

允许修改 `model/group_identity.go` 的引用检查和必要的同契约引用路径、
专属模型测试以及本记录；保留已经审查通过的 #403 H 差异。
`model/subscription.go` 只读，不改购买、计费、schema、权限、全局解析器。
不碰 web、locale 或共享文档，不执行全量构建、提交、推送和部署。

## 已核实调用链与契约

- `PUT /api/group/details` → `UpdateGroupDetails` →
  `SaveGroupConfigWithOptionsAndAutoConfigResult` → 事务内
  `groupBusinessReferenceCount`；`SaveGroupConfig` 复用同一个保存事务。
- 现有门禁覆盖套餐升级组、订阅升级组及购买前分组，遗漏两表的降级组。
  两表都保存字符串 code；用户订阅保留套餐购买时的降级快照。
- `ExpireDueSubscriptions` → `expireDueSubscriptionsForUserTx` →
  `reconcileUserGroupAfterSubscriptionChangeTx`，显式降级时直接使用快照字段，
  因此有引用的目标必须保留。
- 沿用现有 code / alias 身份解析、精确标识匹配及当前 code 优先于 alias 的规则；
  显示名称不参与业务匹配，未知标识或仅包含相同子串不形成引用。
- 删除失败必须回滚同次保存的令牌迁移、其他组删除、名称修改和配置修改；
  没有实际引用的组仍可删除。不会自动清空订阅字段或替换为 default。
- 分组显示改名保持 ID/code 不变，既有订阅字段无需因展示改名写回。
  静态核对 `model/group_code_migration.go`：旧 code 迁移后会登记历史 alias，
  当前只归一化套餐升级组及订阅升级组、购买前分组，没有归一化两表降级字段。
  本次删除门禁复用历史 alias 解析即可保护目标，没有修改该迁移流程或缓存策略，
  也不以本次测试证明降级字段已经改写为新 code。

## 测试计划（编码前）

1. 新建专属测试文件，复用现有隔离 SQLite fixture，显式迁移订阅两表。
2. 通过业务删除入口分别验证套餐/订阅仅 downgrade 引用、code/alias：
   修复前应复现“错误允许删除”，修复后拒绝并保留目标。
3. 验证同一保存内的令牌迁移、批量删除、显示改名和配置全部回滚。
4. 验证未知标识、子串、被当前 code 遮蔽的 alias 不阻止非引用组删除，
   且原订阅标识不被改写；验证目标显示改名不改变稳定标识。
5. 通过实际到期入口验证被保护的目标仍可解析，用户按原 code 降级，
   状态正常过期；不模拟真实购买或修改到期实现。
6. 顺序执行定向测试，每个子进程上限 60 秒，Go 内部超时 55 秒，
   TEMP/TMP/GOTMPDIR 指向 D 盘，保留命令、日志和数值退出码。
   新测试使用 testify，JSON 操作如有需要只用 common 包装器。

## 验证边界与交接

Default 与 Classic 共享后端 `/api/group/details` 删除契约，错误由既有 API
返回，两套模板的提交值、响应结构和名称展示契约不变，不制造页面改动。
Default 依据 `web/src/features/system-settings/api.ts` 的 GET/PUT；Classic
依据 `web/classic/src/pages/Setting/Ratio/GroupRatioSettings.jsx` 的 GET/PUT。
路由 `router/api-router.go` 与 `controller/group.go` 确认共用同一保存入口；
这里只读核对代码，没有执行浏览器或前端测试。
本项只实跑 SQLite；MySQL/PostgreSQL 仅审查复用现有跨库 GORM 路径，
不宣称三库实测或生产验收。独立审查由另一工人执行。

本记录索引需协调者统一登记至 `docs/developer/README.md`，本工人遵守共享文档写入冻结。

## 最小实现与实际回归

本项生产代码仅在 `groupBusinessReferenceCount` 的现有旧标识检查中增加
`SubscriptionPlan.DowngradeGroup` 与 `UserSubscription.DowngradeGroup` 两项。
缺表/缺列兼容、SQL 候选查询与精确匹配、事务边界、拒绝删除错误文案均沿用原实现；
不重写共享 helper，不增加 schema、权限或购买计费行为。
与本项开始时保留 H 差异的文件快照比对，`group_identity.go` 仅新增上述两行。

新增 `model/subscription_downgrade_group_reference_test.go`，4 个主测试、
13 个叶用例覆盖：

- 两表分别仅有降级引用时，通过 `SaveGroupConfig` 拒绝删除；同时覆盖当前 code、
  历史 alias、禁用套餐与已过期订阅快照，使用显示名称与 code 不同的实体。
- 通过 `SaveGroupConfigWithOptionsAndResult` 验证失败后令牌分组迁移及 ID 绑定、
  批量组删除、名称和配置均保持原值。
- 空标识、未知标识、仅子串匹配、显示名称及被其他当前 code 遮蔽的 alias
  均不阻止无引用目标删除，两表原值不被改写。
- 通过 `ExpireDueSubscriptions` 实际到期入口验证显示改名后的目标仍存在，
  原 code/alias 快照与用户降级值保留，订阅状态正常过期，目标仍解析到原 ID。

## 执行命令与证据

证据目录为 `D:/Temp/subscription-downgrade-group-reference/`。每次测试均顺序执行，
独立 runner 将 TEMP/TMP/GOTMPDIR 设置到该 D 盘目录，60 秒超时终止整个进程树，
Go 内部使用 `-timeout=55s`；相应 `.json` 保存完整命令、数值退出码和耗时，
同名 `.log` 保存输出。Go 版本为 `go1.25.5 windows/386`。

1. `red.json`：修复前首次编译超过外部预算，退出码 **124**，60.42 秒；
   没有断言输出，不计红测复现。
2. `red-retry.json`：修复前暖缓存重跑，退出码 **1**，8.51 秒；
   两表 code/alias 引用均被错误允许删除，批量保存及到期目标测试均在预期
   “应拒绝删除”处失败，确认通过真实业务入口复现。
3. `green.json`：相同新增测试修复后退出码 **0**，7.24 秒，13 个叶用例通过。
4. `related-regression.json`：退出码 **0**，3.93 秒，49 个主测试、97 个叶用例通过，
   覆盖既有保存/删除/改名/原子配置、旧 code 迁移、H 名称提示及新增测试。
5. `expiry-regression.json`：退出码 **0**，3.50 秒，9 个主测试、18 个叶用例通过，
   覆盖既有续期回退、到期缓存提交顺序、管理员结束订阅、认证会话与新增测试。

```powershell
go test ./model -run '^TestSubscriptionDowngrade' -count=1 -timeout=55s -p=1
go test ./model -run '^(TestSaveGroupConfig|TestGroupValidation|TestMigrateLegacyGroupCodesToIDs|TestGroupCodeMigration|TestPreviewGroupCodeMigration|TestSubscriptionDowngrade)' -count=1 -timeout=55s -p=1 -json
go test ./model -run '^(TestExpireDueSubscriptions|TestScheduledExpiry|TestAdminDeleteActiveSubscription|TestSubscriptionGroupTransitions|TestSubscriptionGroupCacheRefreshFailure|TestSubscriptionDowngrade)' -count=1 -timeout=55s -p=1 -json
```

上述命令由证据目录内 `run.py` 启动，不能直接执行后声称具有外部 60 秒限制。
未使用 `-vet=off`；只做所列定向测试，没有全量构建。
格式检查只针对本项三个文件，禁止使用会触及全局前端的格式化包装脚本。
已完成两个 Go 文件的定向 gofmt、本记录的直接 Prettier write/check，
以及三个文件的 `git diff --check`；未跟踪新文件另行检查尾随空白与文件末尾换行。

## 已知边界与交付

- 精确文件：本记录、新专属模型测试、`model/group_identity.go` 的两行引用补全；
  `model/subscription.go`、旧 code 迁移实现和原 H 其他文件未写入。
- 不修复已经存在的未知/已删除目标，不校验所有历史脏数据，不重建分组；
  正常业务删除入口从此会拒绝仍被任一降级字段引用的实体。
- 旧 code 迁移的降级字段归一化与对应套餐缓存失效覆盖没有在本项扩展；
  当前仅验证显式构造的历史 alias 能被删除保护和到期解析正确使用。
- 跨项目 API 影响仅是恢复应有的引用删除保护，失败消息结构与现有名称提示一致；
  没有计费、配额、认证、渠道健康或外部传输契约变化。
- `NewAPIForDouDi`：若也保存独立降级字段并提供分组删除，建议分享该保护用例和
  两字段补全思路；未检查兄弟仓库，不能直接建议照搬代码。
- `NewAPIModifyByGang`：同样建议先核对降级快照和删除门禁后再采纳，
  该修复无新依赖；未创建 Issue、PR 或其他消息。
- 实现者不兼任独立审查者；交回协调者安排另一人复核，未提交、推送或部署。
