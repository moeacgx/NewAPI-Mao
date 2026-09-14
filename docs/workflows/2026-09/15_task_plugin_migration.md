# 原生任务计费修复与插件迁移准备

## 基线、范围与恢复点

- 分支：`agent/upstream-task-plugins`；PR 目标为 `custom-main`。
- 当前基线：`origin/custom-main@45ab82100a74b9e7c3866f1c173d16d32745527e`。
- 固定上游：`9fe0457ee1f4b9de407a254500d54f5a8f41ee29`；rc.37 参考
  `385d2dfd10d821b25c8a6766bd16eea248cb1652`；`eb48396d5` 仅作为插件架构引入来源。
- 目标、迁移矩阵和后续风险见 [任务插件迁移](../../developer/task-plugin-migration.md)。
- 本阶段提交、推送并创建草稿 PR；完整插件架构和 Classic 插件后台仍未就绪。
  不合并、不发版、不部署、不调用生产 API；不修改协调工作区。

原分支错误采用旧本地 `353352428`，同步前为 behind=92、ahead=3。
三条独有记录 `5fb6a75ca`、`99db8805c`、`353352428` 均是合并提交，
与共同祖先的文件差异为空，未承载独有文件改动。因此保存后将本分支基线重建到远端最新提交，
没有携带这三条冗余历史。

恢复点：

- 原提交：`backup/upstream-task-plugins-before-sync-20260915`。
- 全部已修改和未跟踪文件：`refs/backup/upstream-task-plugins-work-20260915`，
  对应 stash 对象 `f2356869e11a7187c155810f8ce2e75054d18f54`；未删除 stash。
- 用 `git stash push --include-untracked` 完整保存，`git rebase --onto origin/custom-main HEAD`
  同步，再用 `git apply --3way` 恢复精确文件补丁。只有两份文档索引的插入点冲突，
  保留远端新条目和本任务条目。恢复新测试及专题文件时检查目标不存在。
- 未使用 `reset --hard`、`clean`、整文件 ours/theirs 或线上数据迁移。

## 本 PR 实现

- Remix 在源任务解析后保存独立倍率快照，价格重建时恢复。原实现丢失倍率：
  8 秒/高分辨率预期 quota 为 66666 或 80000，实际仅 5000。两次尝试均由快照恢复，
  不继承上次尝试的计算结果。
- AtlasCloud 在预扣前校验原始 `duration`、`seconds` 与 `metadata.duration`。
  后者原先仅在构建上游请求时覆盖，使预扣 4 秒却发出 8 秒。现在有效正值按
  `seconds < duration < metadata.duration` 优先级归一；0 为未指定，不覆盖已有正值；
  非法或超过 `MaxTaskDurationSeconds=3600` 的值返回 400。
- 保留 AtlasCloud 类型 61、原生适配器、公开任务 ID 和失败输出语义；保留源任务多组
  ID/别名/继承授权、当前组/模型/渠道能力复查，以及轮询退款认领和 reconciliation。
- 迁移基础仅包含文档中的明确映射和原生契约测试。早期离线 JS 引擎、十份插件源码、
  未使用的映射函数及 Sobek 依赖已保存在恢复点，不纳入本 PR。

## 固定上游重新复核

`9fe0457ee` 仍存在渠道 61 冲突、新插件源任务入口缺少本地分组授权、
当前版本轮询、签名无期限、资源存储占位和覆盖本地退款恢复链的迁移风险。
上游 key 已限长 30，Alibaba 已保留 seed=0/显式 false；撤回早期对此两项的阻断判断。

Classic 的 `constants/channel.constants.js` 已声明 61=AtlasCloud；任务页使用
`/api/task/`、`/api/task/self` 和旧 action。本阶段没有新增页面字段或 API，
因此不改前端、不声称 Classic 插件管理完成。后续 Classic 的插件管理、绑定、资源、
用量 UI 必须与 Default 同步实施，依赖明确编号、版本和权限合同，按协调计划由前端负责人交付。

## 验证

使用 Windows amd64（本机 Go 默认 windows/386），Go 测试传 `-timeout=60s`。
以下是同步到 `45ab82100` 后重新执行的结果，不复用旧分支通过记录：

```powershell
$env:GOARCH = 'amd64'
go test ./relay ./relay/channel/task/... -count=1 -timeout=60s
go test ./service -run 'TestUpdateSunoTasksStalePollsRefundExactlyOnce|TestSweepUnrefundedFailedTasks|TestSweepTimedOutTasksHonorsRefundRolloutBoundary|TestRefundImageTaskQuota|TestRefundTaskQuota|TestRecalculate_|TestCASGuarded|TestSettle_PerCallBilling|TestSettle_NonPerCallBilling' -count=1 -timeout=60s
go vet ./relay ./relay/channel/task/atlascloud
```

- 原生任务包测试通过；部分既有适配器无独立测试，不代表已实测全部供应商行为。
- service 定向测试通过，覆盖钱包/订阅退款、CAS、失败恢复、崩溃后补偿及按次结算。
- `go vet` 通过；提交前核对 Go 格式、四份相关 Markdown 的 oxfmt、新增本地链接及
  `git diff --check`。
- 新增测试覆盖实际多组与别名、显式令牌无 ID 回退、撤权、Remix 倍率继承/两次尝试，
  AtlasCloud 时长边界、倍率/请求一致性、公开 ID、原生查询路径和无输出终态失败。
- 源码初审后先以测试复现两项计费缺口，再加入修复；同步后对最终变更重新验证。
- 本阶段不引入 JS 运行时；早期离线原型测试不作为本 PR 当前测试。
- 无真实供应商调用、MySQL/PostgreSQL 实库验证、完整 Go 全库测试或前端构建；本阶段无
  数据库/前端变更。未修改 `relaykit/` 及其公开 API，未额外运行独立模块构建。

## 共享契约与回滚

AtlasCloud 非法/超界时长由原先可能接受改为 HTTP 400；合法 metadata 时长覆盖现在同步
参与预扣。Remix 恢复按源任务时长/规格计费，实际收费可能高于原先丢失倍率的错误值。
API 路径、模型 ID、认证、action 和原生平台值不变；没有向兄弟项目发送通知。

本阶段回滚无需数据库逆迁移，但回滚计费代码会重现上述缺陷。JS 全量切换需要独立实现
按任务版本读取/轮询、完整源任务授权、持久化响应屏障、签名与审计、计费及双模板页面；
在途任务未排空前不得回退到不识别 JS key 的旧二进制。
