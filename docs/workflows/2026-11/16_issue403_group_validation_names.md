# Issue #403 H：分组校验错误显示当前名称

## 目标与范围

修复已解析到真实分组实体的用户可见校验错误：渠道与令牌绑定禁用分组、分组配置负倍率/负并发上限，以及高级选项将独立分组加入 Auto 的拒绝提示。

初始范围为 `model/group_bindings.go`、`model/group_identity.go`、本项专属模型测试及本记录；不提交、推送、部署或执行全量构建。共享开发文档索引由协调者整合，本工人不改其他工人拥有的文件。

协调者随后明确追加 `model/group_token_migration.go` 的已知实体倍率保护不一致错误，以及 `service/prompt_audit_config.go` 的已知停用分组错误；补充专属服务层测试，保持迁移拒绝、已有豁免保留和停用校验条件不变。

## 契约与方案

- 已知实体优先使用当前 `Group.Name`；配置保存使用经过 `normalizeGroupName` 规范化的本次显示名称。
- 缺失名称时保留原标识回退；未知 ID/code、非法标识、未解析到实体的迁移诊断保留原行为。
- alias 只用于解析稳定身份，名称变更不改变 ID/code、顺序、去重、状态校验、独立分组约束或事务原子性。
- 不修改 `GetGroupDisplayNameMap` 等共享帮助函数的语义；不新增名称查询循环。
- 不修改路由、鉴权、计费或数据库结构；GORM 查询继续兼容 SQLite/MySQL/PostgreSQL。
- 已检查本地分组专题 `docs/developer/group-tags.md`；README 中若干 2026-07 分组历史记录在当前工作区不存在，以当前源码为准。

## 双模板与国际化边界

Default 与 Classic 共用后端分组配置、选项更新、渠道及令牌绑定 API；本项修复共享错误 `message`，不改变提交值或响应字段。已核对：

- 分组配置：`PUT /api/group/details` → `controller.UpdateGroupDetails` → `SaveGroupConfigWithOptionsAndAutoConfigResult` → `SaveGroupConfigWithOptionsAndResult`；负倍率/负并发使用本次规范化名称。Default 的 `features/system-settings/api.ts`、`models/group-ratio-form.tsx` 与 Classic 的 `GroupRatioSettings.jsx` 共用此入口。
- 独立组 Auto 约束：`PUT /api/option/` → `controller.UpdateOption` → `model.UpdateOption` → `UpdateOptionsBulk` → `lockOwnedGroupReferenceOptionWrite`；仅在原实体查询中补选 `name`，锁和判定不变。
- 渠道/令牌写入：渠道创建与更新的模型准备路径、`controller/token.go` 的两处绑定验证共用 `resolveBindingGroupsWithPolicy`；已知禁用实体按 ID/code/alias 均显示名称。未知 ID 保留原 `ID 不存在`，缺名的已知 ID 保留旧回退文案。既有绑定允许保留禁用分组的策略不变。
- 迁移：`POST /api/group/token-migration/preview` 与 `POST /api/group/token-migration` → `controller/group.go` → `PreviewTokenGroupMigration` / `MigrateTokenGroup` → `buildTokenGroupMigrationPlans`；仅两处已知实体倍率镜像不一致提示使用名称。Default 的 `features/system-settings/api.ts` 与 Classic 的 `GroupTokenMigrationModal.jsx` 均消费原错误消息。
- 安全审计：`PUT /api/security-audit/builtin-policy` → `controller.UpdateSecurityAuditBuiltinPolicy` → `SaveSecurityAuditBuiltinPolicy` → `resolvePromptAuditGroupCodesForUsage`；普通配置保存也复用同一解析路径。Default `features/security-audit/api.ts`、Classic `pages/SecurityAudit/api.js` 均由 `unwrap` 提取服务端 `message`。
- 控制器使用 `common.ApiError` 原样返回 `HTTP 200`、`success:false`、`message:err.Error()`；本项没有新增 i18n key，也没有修改共享 locale、HTTP 状态或错误封装。

两模板覆盖依据为共享 API 和消费者源码核对；本项没有运行浏览器、真实鉴权或生产验收，不需要无意义的页面变更。

## 测试计划

1. 通过真实渠道/令牌绑定准备入口验证名称与 code 不同、历史 alias、当前改名、禁用实体和缺失名称回退；成功路径仍产生原 ID/code。
2. 通过分组配置保存入口验证负倍率、负并发上限错误显示规范化名称，失败不持久化配置。
3. 通过 `UpdateOption("AutoGroups", ...)` 验证独立分组经 code/alias 定位后显示当前名称，选项写入失败保持原值。
4. 保留未知 ID/code、非法标识的拒绝及回退，验证已有禁用绑定编辑策略不被更改。
5. 先运行失败回归，再最小修复，最后运行专属和相关既有模型测试；每个测试进程硬上限 60 秒，TEMP/TMP/GOTMPDIR 指向 D 盘专属目录。
6. 使用 testify `require`/`assert`、项目 `common.Marshal`；定向 gofmt、Markdown 格式与链接检查、`git diff --check`。
7. 迁移分别覆盖无稳定倍率、稳定倍率与 JSON 不同两条分支，经 alias 解析后预览和执行都拒绝，且令牌、JSON 镜像与绑定完全不改。
8. 安全审计通过 `SaveSecurityAuditBuiltinPolicy` 分别验证官方风控范围和新增自动封禁豁免组，覆盖 code/alias、改名、空名称、未知回退，失败不改配置版本或作用范围。

## 验证结果

环境实测：Go `1.25.5 windows/386`、Bun `1.4.4`。测试使用本地 SQLite 夹具，`TEMP`、`TMP`、`GOTMPDIR` 均为 `D:/Temp/issue403-group-validation`，每个 Go 测试子进程由专属 `run.py` 设置 60 秒墙钟上限并超时终止进程树，Go 内部另设 `-timeout=45s`。

- 初始失败回归第一次编译达到 60 秒上限，记为超时而非通过；第二次在 10.10 秒内完成并准确复现名称错误，日志 `red-second.log`。
- 原范围修复后 `^TestGroupValidation` 通过，用时 10.00 秒，日志 `green-base.log`。
- 追加迁移与审计回归分别在 11.63 秒、17.66 秒内复现旧 code/alias 提示，日志 `red-migration.log`、`red-service.log`。
- 最终模型相关回归通过，命令如下，墙钟 13.20 秒、包测试 1.230 秒，日志 `green-model.log`。

```powershell
go test ./model -run '^Test(GroupValidation.*|LegacyFallbackAllowsExistingDisabledBindings|TokenGroupBindingsPreserveOrderAndModes|SaveGroupConfigChangesDisplayNameOnly|SaveGroupConfigSupportsAtomicDisplayNameReassignment|SaveGroupConfig.*Exclusive.*|UpdateOption.*|MigrateTokenGroup.*|TokenGroupMigration.*|GroupIdentifierResolutionAndDisplayMapUseStablePrecedence)$' -count=1 -timeout=45s
```

- 最终服务相关回归通过，命令如下，墙钟 25.93 秒、包测试 3.221 秒，日志 `green-service.log`；包含已有白名单停用保留、历史 alias 和自动封禁语义。

```powershell
go test ./service -run '^Test(PromptAuditGroupValidationNames|SaveSecurityAuditBuiltinPolicy.*|UpstreamPolicyGroupScopeAcceptsHistoricalAliasDuringMigration|CyberPolicyAutoBan.*)$' -count=1 -timeout=45s
```

- 定向 `gofmt -l` 无输出，Prettier `--check` 通过，作用域内 `git diff --check` 通过；7 个交付文件无行尾空白，工作记录 6 个相对链接全部存在。
- 生产变更为 4 个文件的错误展示参数和一次实体查询投影；没有新增 SQL 方言、数据库迁移、共享 helper 语义、公共接口字段、权限或计费变更。
- 未运行 MySQL/PostgreSQL 集成、控制器 HTTP 集成、前端构建、浏览器或生产环境验证；SQLite 结果不代表三库实测。
- 不涉及 `relaykit/`；未执行提交、推送、部署和全量构建。

## 精确交付文件

- [分组绑定](../../../model/group_bindings.go)
- [分组身份与配置](../../../model/group_identity.go)
- [令牌迁移](../../../model/group_token_migration.go)
- [安全审计配置](../../../service/prompt_audit_config.go)
- [模型回归](../../../model/group_validation_names_test.go)
- [服务回归](../../../service/prompt_audit_group_validation_names_test.go)
- 本工作记录。

## 集成与分享边界

共享 `docs/developer/README.md` 的本记录索引由协调者整合；本工人遵守文件独占范围，没有直接修改该索引。独立审查也交协调者安排，未将实现者自查冒充独立审查。

跨项目影响仅为已知实体的可读错误文本变化；请求、响应结构、ID/code/alias 识别、权限、计费、状态码与迁移原子性保持兼容。`NewAPIForDouDi`、`NewAPIModifyByGang` 均建议仅在具有相同分组身份及可改显示名能力时分享此修复思路；尚未核对兄弟仓库实现，不建议直接套补丁，也未创建 Issue/PR 或发送消息。
