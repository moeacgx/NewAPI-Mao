# 多分组并发满时继续回退

## 问题与根因

有序分组选择在首个分组返回渠道并发上限错误时直接终止，即使令牌后续已授权分组仍有可用渠道也会拒绝请求。
显式多分组与自动分组共用此路径。`cross_group_retry` 不是本次问题的控制开关。

## 方案与安全边界

- 仅渠道并发上限错误允许继续检查后续授权分组；其他选择错误立即返回。
- 所有授权分组耗尽且曾遇到并发不足时，保留并发限制错误；仅缺少候选时保留原有无候选语义。
- 保留分组顺序、组内优先级、重试预算、渠道排除及按组指定渠道的交集，不扩大令牌权限、不突破并发上限。
- 初选或后续重试中的原子抢租失败均在当前路由尝试内排除已失败渠道并继续授权分组选路，不消耗供应商重试预算；重选跨组时同步实际分组上下文，确保单用户分组并发及后续计费使用新组。单用户分组并发拒绝仍立即终止。
- 不改变接口、模型标识、数据库结构、缓存键或生产配置。Default 与 Classic 共用后端，无需前端改动。

## 验证计划与结果

先添加真实数据库和并发租约的确定性回归，确认旧实现失败后修改实现。
覆盖显式/自动分组、跨过空组、全组满、满与无候选混合、未授权分组、按组限定、单组及重试上下文；
检查分发重选后的实际分组与单用户并发边界。每次 Go 测试设置 `-timeout 60s`。

验证结果：

- 新增回归在旧实现上确定性失败：首组满时未选择后备组；重选后实际分组仍为首组，且后备组用户并发限制被绕过。
- 修复后，service、middleware、model 相关回归共 81 项通过，覆盖缓存与 SQLite 查询、真实内存并发租约、分组权限、优先级重试及请求结束释放。
- `go vet ./service ./middleware ./controller ./model`、Go 格式检查、Markdown 格式检查及 `git diff --check` 通过；开发索引中的新增链接存在且可解析。
- 独立复核未发现阻塞问题。新增 miniredis 双客户端竞争回归覆盖真实 Redis Lua 租约路径；未实测 MySQL/PostgreSQL、真实 Redis 集群拓扑及线上业务请求。

最终相关回归命令：

```powershell
go test -mod=readonly ./service ./middleware ./model -run 'Test(CacheGetRandomSatisfiedChannel|RelayMaxRetries|RetryParam|GetRequestTokenGroups|ParseTokenGroupList|TokenGroupChannel|.*ChannelConcurrency.*|.*GroupUserConcurrency.*|GetChannelWithSelectionExclusions.*|DistributeReselection.*|DistributeGroupChannelRestriction.*|SetupContextForSelectedChannel.*|SetupContextForTokenGroupChannels.*|ChannelSelectionConcurrencyError.*|ReleaseChannelConcurrency.*)' -timeout 60s -count=1
```

仅本地实现与验证，不包含合并、发布或生产部署，也未发起付费供应商调用。

## 复核结果

2026-10-28 再次检查全部差异，独立只读审查未发现本次修改引入的阻塞问题。

- 上述相关回归重新运行，81 项通过；新增 Redis 竞争回归 2 项通过；补充控制器重试与切组提示词/计费准备契约检查 18 项通过；`go vet ./service ./middleware ./controller ./model`、Go/Markdown 格式与 `git diff --check` 均通过。
- 合并前补查单组重试：已有兼容回退标记的渠道一旦原子抢租失败，硬排除同时撤销其回退标记，避免反复重新放开同一渠道；缓存和 SQLite 两条路径均验证先允许回选、再硬排除的行为。
- 补充硬排除修复后，受影响的 `service`、`middleware`、`controller`、`model` 四个 Go 包最终全量回归共 3211 项通过。
- 原有选路策略测试曾因未初始化 `model.DB` 崩溃；本次为其增加隔离 SQLite 分组夹具，相关 `PrepareSelectedRouteRequest` 测试连续运行 3 次通过。
- 新增 Redis 双客户端测试以真实生产 Lua 路径制造预检查与抢租之间的竞争，`RetryTimes=0` 时初始实现确定性返回 503，修复后初选与 `getChannel` 两项均通过；竞争租约按 token 保留并在测试清理。

```powershell
go test -mod=readonly ./controller -run 'Test(PrepareSelectedRouteRequest|GetChannelFallsBackAfterConcurrentSetupFailureWithoutRetryBudget|ShouldRetry|ShouldEvictAffinity|TokenPromptSelectedGroupRetry|TokenPromptBillingIncludesLoadedInputAndRestoresAfterGroupChange)' -timeout 60s -count=1
```

本地竞争回退不改变外层 `RetryTimes`；固定渠道或锁定渠道仍不跨组回退。

## 跨项目影响与分享建议

兼容 API 的并发错误仅在所有授权候选均无法使用时返回；请求身份、usage 和计费接口保持现有契约。

- `NewAPIForDouDi`：若沿用相同有序分组选路，可分享本修复，需核对自身并发错误与实际分组上下文。
- `NewAPIModifyByGang`：若存在同一提前终止分支，可分享；需先验证授权列表、渠道限定和重试语义，不能直接照搬。

不自动向兄弟仓库创建 Issue、PR 或发送消息。
