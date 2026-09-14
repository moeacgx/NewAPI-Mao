# Default 前端上游集成

## 基线与范围

- 原任务分支：`agent/upstream-default-frontend`，起点 `353352428`。
- 本地主线：`origin/custom-main` 的 `45ab82100`，已通过合并保留主线增量。
- 评估上游：`upstream/main` 的 `9fe0457ee`，共同祖先 `e2c7aa7b`。
- 本 PR 仅处理 Default：`web/rsbuild.config.ts` 的入口为 `src/main.tsx`。整体任务已要求 Classic 实现，其等价功能由既有 Classic Agent 在独立工作树交付，不混入本 PR。
- 不整文件覆盖现有实现，按功能域移植兼容片段。

## 实现计划与契约

1. 渠道多 Key 策略编辑：已有敏感权限且为多 Key 渠道时更新 `multi_key_mode`，保持原密钥、供应商、分组与监控设置。
2. 渠道分类和使用日志共享供应商识别；保留请求模型、映射模型、管理员实际响应模型的独立语义。
3. 时间表达式：新建同日范围使用交集，跨午夜使用并集；只折叠与范围方向一致的既有条件。合法排他上界（如 `hour < 24`）不被丢弃；保留日志 `request_rules` 的权威追踪语义。
4. 价格展示使用价格币种格式，保留路线规格、分组名称、零倍率和本地综合折扣契约。
5. 使用日志筛选避免密码自动填充；额度展示只使用现有额度字段。
6. 兑换码可选导出，保留本地 `DELETE /api/redemption/batch` 与 `{deleted_ids, skipped}`、营销兑换码和加载隔离。
7. Passkey 能力检测不能因缺少平台认证器而禁用外置密钥或跨设备认证。
8. 新增文案通过脚本维护七种语言，并补齐同步发现的 101 项历史缺失翻译；不覆盖本地已存在的翻译。

供应商识别来自 `d92612038`，仅统一分类和图标；不会修改管理员的 `vendor_id` 绑定，也不把名称推断当作上游实际响应证明。Wan 图标随资源打包，不增加外部请求。

用户额度借鉴 `2bec37062`、`ea7cb0ba4`：以剩余额度和已用额度分别判断空态，支持键盘打开详情及币种切换。令牌周期额度保持原有独立实现，未移植上游普通总额结构。

新增修复沿用现有 HTTP 接口，不增加后端字段、生产调用或数据库迁移。历史价格配置不会自动迁移；已有错误的全天表达式需要管理员显式审阅、修改。

## 后端依赖与剩余冲突

- 任务插件：缺少 `/api/plugin/task`、`/api/task_plugin_options`、市场与版本管理以及插件使用量契约；不得替换本地 `/api/extensions` 和 `native v1`。
- 模型/供应商重构：缺少供应商操作预览、批量合并、版本化元数据同步、级联删除、模型广场状态筛选契约。
- 新表达式编辑器：`fixed()`、`img_cr`、插件价格及日志计费单位需要后端编译、预扣和结算共同接入。
- 新安全中心：本地已有会话和动作绑定的 `X-Security-Proof`，但缺少新令牌管理、`flow_token` 和单次授权契约。
- Passkey 多域：缺少专用域名更新接口、历史 RP ID、删除预览与二次确认响应。
- 新审计和任务附件：缺少 `/api/audit/self`、`/api/task/:id/artifacts` 对应数据与接口。
- Responses WebSocket 配置依赖对应后端能力，本工作不单独暴露开关。

上游统一渠道配置页、大型模型供应商编辑器、公共表格移动筛选与可搜索分组筛选均未整段导入。本地分组仍按稳定内部标识提交、按显示名展示；后续整合应连同共享 Combobox 的焦点和 Enter 选择行为分别验证，不能直接采用上游 `label = group`。

本地 ApiPanelWatch、TokensPro 并发配置、监控继承、通知筛选及模型实际响应字段通过先同步本地主线保留。与 `origin/custom-main` 的最终差异不包含 Classic、后端或通知模块实现变更。

## 验证计划

先运行缺陷回归用例确认失败，再移植实现；覆盖权限裁剪、密钥不变、时间边界、货币模式、供应商兜底、兑换码导出与外置认证器。

最终使用 Bun 运行 lint、typecheck、受影响测试和 Default build；测试每批最多运行 60 秒。检查格式、文档链接和 `git diff --check`。

## 最终验证与交付边界

- Bun 1.4.2 通过临时工具缓存调用；`bun install --frozen-lockfile` 成功，未修改包清单与锁文件。
- `bun run typecheck` 通过；`bun run build` 通过，Default 生产构建约 5.3 秒。
- 完整 Default 测试拆为 `bun run test --maxWorkers=3 --shard=1/2` 与 `--shard=2/2`，每批设置 60 秒超时；分别 36.48 秒、39.33 秒，共 108 个文件、496 项测试通过。
- `bun run lint` 已执行，但现有脚本扫描 Classic 并报错。排除 Classic 后仍有 293 条既有 lint 错误；本次修改的 25 个 TS/TSX 文件定向 oxlint 全部通过。价格编辑器本次触及的 11 条既有错误已解决，未把全量 lint 失败表述为通过。
- 使用仓库 oxfmt 配置格式化变更，保留原版权头；`git diff --check` 通过。
- 七种语言各 7079 个键，缺键为 0；新增 11 个业务文案键，共 77 项翻译，并补齐 101 项历史缺失翻译。逐键比较确认没有修改任何原已存在的翻译值。同步报告仍有历史英文同值候选，未声称全仓所有文案已完成翻译。
- 新增文档链接检查通过；开发索引和既有文档仍有历史失效引用，本次未扩大修复范围。
- 无 Git 未合并索引项；剩余项是前文列出的接口、数据契约和成套 UI 重构依赖，并非已完成整仓上游合并。
- 未部署、未调用生产接口；未进行真实 Passkey 硬件或线上浏览器业务流程验证。

目标审查基线为 `origin/custom-main@45ab82100`。分支仅提供可审查实现，由协调者决定后端能力与其余前端域的集成顺序；不自动合并主线。

跨项目影响：没有新增请求/响应协议、模型 ID、认证或计费数据交换字段；多 Key 策略继续使用既有 `multi_key_mode`。供应商推断只用于展示，日志实际响应模型继续按既有管理员权限隔离。未向兄弟仓库发送通知。

## PR 续办与 lint 基线复核

2026-09-15 重新执行 `git fetch origin --no-tags`，目标仍为 `45ab82100`，`behind=0`。上游分析点固定为 `9fe0457ee`，rc.37 参考点为 `385d2dfd1`，未随上游更新。

原有三个分支独有合并提交为 `353352428`、`99db8805c`、`5fb6a75ca`。同步后提交 `35e61b999` 与 `45ab82100` 的树均为 `87d1c9fd3443b2e506281b312a340f6495a016d9`；`git diff --exit-code origin/custom-main 35e61b999` 返回 0。这证明它们没有给本 PR 带入无关文件差异，无需重写已推送历史。原分支完整历史已通过本地 `git bundle create` 保存并验证，备份位于本工作树依赖缓存。

新增工具 [compare-lint-baseline.mjs](../../../web/scripts/compare-lint-baseline.mjs) 只在当前工作树的 `.local-tests/` 内解包已提交的基线和目标 `web/` 快照，复用当前 `web/node_modules`；不切分支、不改源码、不安装依赖。不能把快照放进 `node_modules`，否则 oxlint 的循环导入解析会漏报四条错误。

在仓库根目录执行以下命令（临时 Bun 路径失效时替换为本机 Bun 绝对路径）：

```powershell
$taskBun = 'C:/Users/Administrator/AppData/Local/npm-cache/_npx/b22965130bfded9d/node_modules/bun/bin/bun.exe'
git fetch origin --no-tags
node web/scripts/compare-lint-baseline.mjs origin/custom-main HEAD $taskBun
```

脚本为两个快照分别执行 `bun run lint --format json` 和 `bun run lint --ignore-pattern classic --format json`。按文件、规则和诊断文案比较多重集合，忽略增删行引起的位置变化，同时单独统计变更文件里的错误。输出目录包含原始 JSON、stderr、差异和版本摘要。脚本退出 0 仅表示没有新增错误及变更文件错误，**不代表完整 lint 通过**。

本轮实际工具版本为 Bun **1.3.13**、oxlint **1.74.0**；临时工具缓存已变化，上轮记载的 Bun 1.4.2 不作为当前版本依据。

| 扫描范围                | 基线错误 | 本补丁错误 | 新增 | 消除 | 变更文件错误 |
| ----------------------- | -------: | ---------: | ---: | ---: | -----------: |
| 完整 web（含 Classic）  |     1470 |       1454 |    0 |   16 |            0 |
| Default（排除 Classic） |      309 |        293 |    0 |   16 |            0 |

两边完整 lint 均退出 1。消除的 16 条为价格编辑器 11 条与 Passkey 5 条；剩余 293 条全部位于相对基线未修改的文件，不要求用户相信仅定向 lint 的结论。

已向既有 Classic Agent `5bc35f88-9053-406b-8de2-155326defffe` 发送九个等价功能域、相关回归测试、Bun 实际路径与版本提醒。Classic 需单独提交实现和验证；本 PR 不声称完成双模板或完整上游集成，也不启用尚缺后端支持的页面。
