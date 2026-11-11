# ALHE-113 P3 正式基线集成交付

## 目标与边界

P1/P2 已分别通过固定版本独立验收。本轮把已审差异应用到固定正式基线
`a946865c9b74046d44db046b0fbb84a363551da5` 的独立导出目录，重跑组合验证，
交出父项可采用的补丁、逐文件清单与验收报告，不新增业务功能。

平台指定分支为 `agent/agent/alhe-113`，起点为
`ffe87461b566dec6ea9119971d5d51f07b83a1f9`。该历史包含 12 项与正式基线不同的快照；
禁止切分支、重写起点或把整分支推作发布候选。因此本轮采用任务明确允许的可移植补丁交付：
保持 HEAD 祖先不变，在无 Git checkout 的临时源码导出目录验证，使用独立 Git index 固定候选树。
候选树对象不是分支 head；最终树 ID、补丁 SHA256 和逐文件映射随交付清单提供。
没有创建另一分支或 detached HEAD，也不以未发布的树对象冒充 PR/CI。

## 集成输入与方案

| 输入        | 固定提交                                   | 搬运范围                                                             |
| ----------- | ------------------------------------------ | -------------------------------------------------------------------- |
| P1/B2       | `e91adf6743fd29a7a907aae9fc47cc9720532744` | 对直接父提交 `95efd0ae40f57eb7d6df77c65ee827e1946896af` 的单提交差异 |
| P2/B5       | `ffe87461b566dec6ea9119971d5d51f07b83a1f9` | 对 P1 的单提交差异                                                   |
| 已审计划    | `881ab6f0e977582a13cd8ee7bf4e2e4999fb16d5` | 保留 P1/P2 已纳入的计划和后续状态增量，不用旧计划覆盖                |
| P1 独立报告 | `e36f5b7f54580e8ab7e388b6ea5ede13ebac76b1` | 仅报告正文与新增索引                                                 |
| P2 独立报告 | `90bbccf254ca7c8c76f696a0400095cce7fa71e4` | 仅报告正文与新增索引                                                 |

README 按正式基线已有内容进行增量插入，保留目标已有发布和跨域索引；
不使用实现快照的 README 整文件覆盖。其他任务文件逐项核对适用性与 blob；
任何冲突或内容调整均在最终证据中说明。

## 契约、安全与验证计划

Default 使用既有渠道转换请求和标签 API；系统设置页面仍为 SUPER_ADMIN，
标签管理 API 为 AdminAuth，普通用户仅使用 UserAuth `self/groups.group_tags`。
保持授权候选集合、当前分组名称展示和稳定 ID/code 提交语义。
Classic 实现不变，核对共享契约并运行已有回归。无后端/API/数据库迁移或计费变更。

在正式基线候选目录运行 P1/P2 综合回归、Default 类型/lint、双模板构建、
model/controller/router 指定共享契约回归、七语、格式、链接和差异检查。
测试结果与源码固定信息随实施完成更新；旧版本独立 pass 不直接覆盖新候选。

本轮不合并、发布、部署、改生产配置/数据或更改自动化。不复制或命名新模板，
不把 Classic 定为 legacy。PR 发布需要从正式基线创建干净历史；当前平台分支无法安全提供，
由父项在允许的独立交付环境应用附件补丁、固定新 head 后继续 PR/CI 和发布门禁。

## 实际隔离结果

正式基线于 2026-11-11 通过 GitHub API 只读复核，仍为上述完整 SHA，
`protected=false`；开放 PR 列表和 ALHE-113 关联表均为空。
无分支保护不等于免验收或 CI 已通过。当前 Git 身份为历史核心作者之一，
只读比较后未修改 Git 配置。`multi-agent-git-pr-coordination` 未在本轮技能目录中找到，
实际依据平台分支约束、任务 P3 条款、multica-platform 及仓库 PR 模板处理交付。

- P1/P2 补丁应用无代码冲突。开发索引按正式基线增量插入 6 个任务条目，
  保留其全部 407 条既有索引，包括实现快照缺失的发布与跨域文档条目。
- 37 个未作 P3 状态调整的任务文件与 P2 最终 Git blob 相同。
  两份独立验收报告只复制原提交正文，不改写旧 head 的结论；计划仅更新顶部实施状态，
  能力登记仅更新两项的验收与集成交接状态。
- 其余 4,178 个正式基线 blob 内容一致，两项子模块 gitlink 沿用正式基线。
  VERSION 保持 `.368`，正式 CORS 文档及 `router/image_task_cors_test.go` 保留。
- 起点 12 项继承差异逐项在附件 manifest 中列出；除 README 接受明确任务增量外，
  其他项候选 blob/缺失状态严格等于正式基线。未删除工作树已有部署脚本或用户文件。
- Windows `core.autocrlf` 使 archive 和 apply 的少数文本出现 CRLF；
  首轮字节检查据此停止，确认只有换行差异后规范化任务文件为原始 Git blob。
  141 个未改动归档文本按 LF 内容比对通过；不把换行差异当作功能冲突或产品修改。

本轮交付补丁名为 `alhe113-formal-integration.patch`；
`alhe113-formal-integration.manifest.json` 记录完整候选树、补丁 SHA256、
每个文件的正式旧 blob、实现来源提交/source blob、新 blob 和 12 项排除证明。
二者属于交付包，补丁自身和 manifest 不递归包含于候选树。
平台分支上的交付提交和候选树分别固定，不把前者冒充干净集成 head。
仓库内保存同内容的 `.patch.gz`，任务评论另附可直接应用的原始 `.patch`。
使用第二份独立 index 从正式基线执行 `git apply --cached --check`、
`git apply --cached` 和 `git write-tree`，重放结果与 manifest 候选树相同；
重放未改变当前 HEAD、工作树或用户 index。

## 候选验证证据（2026-11-11）

工具为 Bun 1.4.4、Node 26.13.0、Go 1.25.5，与 CI Bun 1.4.1、
`web/.node-version` 的 24.10.0 有偏差；两模板均 frozen 安装，锁文件不变。
Node 测试设置 `NODE_OPTIONS=--no-experimental-webstorage`，
Go 设置 `GOARCH=amd64` 和本任务独立 `GOCACHE`。Git 进程仅用
`GIT_CONFIG_COUNT=0` 避免继承缺失 KEY 的无效配置。
所有命令在本任务临时目录的正式基线候选执行，非旧实现工作树。

| 范围                   | 命令/结果                                                                                                                                                                       |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Default 综合           | 下方命令 exit 0，31 文件 / 206 用例全部通过，507.18 秒；P1/P2 均在正式基线候选上重跑                                                                                            |
| Default 类型/构建      | `bun run typecheck`、`bun run build` 均 exit 0，Rsbuild 2.1.6，真实 dist 入口生成                                                                                               |
| 任务源码 lint          | 27 个 TS/TSX 用 `oxlint -c web/.oxlintrc.json <文件列表>`，exit 0；保留原分组编辑器 3 条 prefer-number-properties 警告                                                          |
| Classic 组件           | 下方 Vitest 命令 4 文件 / 37 用例全部通过，164.65 秒                                                                                                                            |
| Classic 契约           | `node --test src/components/table/channels/modals/__tests__/multi-key-editing.test.mjs`，8 通过                                                                                 |
| Classic 构建           | `bun run build` exit 0，Vite 5.4.11，真实 dist 入口生成                                                                                                                         |
| Go 共享契约及正式 CORS | 下方 Go 命令 exit 0，model 1.118 秒、controller 1.204 秒、router 1.659 秒                                                                                                       |
| 七语                   | `bun run i18n:sync` exit 0，七语 missing/extras 均 0；相对正式基线新增 15 键 / 105 个非空值                                                                                     |
| 格式/链接/差异         | oxfmt 0.57.0 受保护副本检查 43 文件 / 27 个源码版权头无差异；14 个新增相对链接缺失 0。81 处原 README 历史坏链单独记录，未当作新增失败；候选 diff 与平台交付 diff 的空白检查通过 |

Default 综合命令（从 `web/`）：

```powershell
bun run test -- src/features/keys src/features/channels src/features/group-tags src/features/system-settings/models src/features/prompt-center/__tests__/group-selector.test.tsx --maxWorkers=1 --testTimeout=15000
```

Classic 组件命令（从 `web/classic/`）：

```powershell
node node_modules/vitest/vitest.mjs run src/pages/Setting/Ratio/__tests__/group-tags.compat.test.jsx src/components/common/group-tags/__tests__/selection.compat.test.jsx src/components/common/group-tags/__tests__/layout.compat.test.jsx src/components/table/channels/modals/__tests__/multi-key-strategy.compat.test.jsx
```

Go 命令（从候选根目录）：

```powershell
go test ./model ./controller ./router -run 'Test(GroupTag|GetGroupDisplayName|FormatGroupDisplayNames|SaveGroupConfigChangesDisplayNameOnly|UpdateChannelMultiKeyConversion|.*Channel.*Permission|.*ImageTask.*CORS)' -count=1
```

复现候选时，在另行获准的干净正式基线 checkout 运行以下命令；本轮未执行 checkout、
创建另一分支或产生以正式基线为父的提交。此为父项后续操作说明，不授予合并/发布权限。

```powershell
git apply --check alhe113-formal-integration.patch
git apply --index alhe113-formal-integration.patch
git write-tree
```

输出须与 manifest 的 `candidateTree` 相同，再由获准执行者固定实际提交、开草稿 PR，
记录真实 CI 链接并交准确新 head 验收。回退范围仅前端与开发文档，不回滚任何数据库；
已转换的多 key 渠道及标签数据由既有后端持久化，不因回退 UI 自动撤销。

## 限制与共享影响

本地结果不等于独立验收或 CI。前端 API mock、后端隔离 SQLite 与路由测试
不等于真实浏览器到完整服务的 E2E；没有运行 MySQL/PostgreSQL 实库、
全仓测试、浏览器视觉/完整响应式、真实供应商、生产或多实例传播验收。
Default jsdom scrollTo、Classic React/Semi、Browserslist、lottie eval 和大 chunk
告警保留。七语历史未译仍为 fr 12、ja 37、ru 37、vi 13、zh 5；
原自定义倍率说明键缺失与开发索引的历史坏链未扩为本轮修复。

仓库 CI 响应 PR 事件；正式 Release 和镜像 workflow 由 tag/手动触发。
本轮未推送、开 PR 或触发 workflow，无 CI 结论；GHCR latest 的外部自动拉取
风险仍未知，不以工作流静态审查代替生产部署风险核实。

Default 新消费既有转换字段、标签管理 API 和 `self/groups.group_tags`；
没有新增 API、数据库、模型、密钥认证、用量计费或部署契约。
Classic 既有链路仅回归，无反向新功能。

- NewAPIForDouDi：有相同 Default 缺口且具备转换敏感权限、稳定分组 ID、
  管理 CRUD 和 UserAuth 裁剪时值得参考，须成套核对宿主版本与权限。
- NewAPIModifyByGang：多密钥及分类选组有价值，先核对模板、响应结构、
  ID/code 语义和生命周期，不能照搬 UI。仅记录分享建议，未外发。
