# ALHE-114：官方新版模板溯源、历史漏项清单与切换计划（修订版）

> 原计划：计划 Agent；本轮文档返修：通用执行（ALHE-114）
> 父任务：ALHE-109
> 日期：2026-11-11（修订：2026-11-11，回应 changes_required）
> 范围：证据核查与计划；不涉及切换实现或合并发版

> ALHE-113 实施附记：以下矩阵保留已验收阶段一计划的静态审计结论。P1/B2 已在固定 head `e91adf6743fd29a7a907aae9fc47cc9720532744` 通过独立验收；实现与验证见[Default 单密钥转多密钥渠道](11_default_multi_key_conversion.md)。P2/B5 及组合版本 P1 回归已在固定 head `ffe87461b566dec6ea9119971d5d51f07b83a1f9` [独立通过](11_default_group_tags_qa.md)。主脑已核准 [P3 正式基线集成交付准备](11_default_integration_delivery.md)，新候选须独立交验；本项未切换生产主题、复制命名模板或满足发布门禁。

---

## 核心结论（三句话）

1. **Default（`web/src`）对应本次固定审查的官方新版模板。** 上游 `QuantumNous/new-api` 的历史固定树 `0409deccfff2b0e8babc28ca552af44d526375bf` 含 `web/src`、不含 `web/classic`；这不是本轮实时最新版本结论，也不能说明上游完整历史。Classic 在本仓库于 2026-04-28 通过 R100 从已有前端迁移（`a42b397607780bb0f2dbc851ae0c01043244691d`，含 `#4265`）；迁移前承继关系未决。
2. **代码默认值为 `default`，线上实际主题未验证。** `setting/system_setting/theme.go` 默认 `Frontend: "default"`；但这只证明未配置时的回退值，不能证明生产 `options` 表 `theme.frontend` 字段的实际值。P0 仅为代码层默认，不代表实际已完成切换。
3. **确认补齐候选只有 B2 的单 key 转多 key，以及 B5 的管理员标签管理和普通用户令牌标签筛选。** 新建多 key、已有多 key 编辑已覆盖；C2 福利活动①–④双模板已有，⑤流水可见入口由双模板一起移除，不恢复。A 类静态覆盖清单为 18 项；原 B4 和 C1 因完整链路不足改为未决，与 Classic 来源共 3 项，退出确定实施范围。先盘点，再在 Default 补齐确认缺口并独立验收，最后才复制、独立命名和大改；Classic 核查共享契约与回归影响，长期双模板策略未定。

---

## 1. 官方新版模板溯源

### 1.1 权威来源

| 字段                    | 值                                                                                                     |
| ----------------------- | ------------------------------------------------------------------------------------------------------ |
| 上游仓库                | `QuantumNous/new-api`（`https://github.com/QuantumNous/new-api`）                                      |
| 上游历史固定审查 commit | `0409deccfff2b0e8babc28ca552af44d526375bf`（本轮未刷新远端）                                           |
| 该 commit 日期          | 2026-11-10 21:44:17 +0800                                                                              |
| 该 commit 说明          | `fix(deepseek): map developer role to system for chat completions (#8287)`                             |
| 上游 `web/` 子目录      | 仅 `web/src/`，**无 `web/classic/`**                                                                   |
| 技术栈                  | React 19、TypeScript、Rsbuild、Base UI、Tailwind CSS、TanStack Router、TanStack Query、Zustand、Vitest |
| 构建脚本                | `bun run build`（Rsbuild）                                                                             |

**证据**：`git ls-tree --name-only 0409deccfff2b0e8babc28ca552af44d526375bf:web` 列出 `src`、`package.json`、`rsbuild.config.ts` 等，不含 `classic`；技术栈及构建依据该树的 `web/package.json` / `web/rsbuild.config.ts`。目录判断仅适用于这棵完整树，不外推完整上游历史。

### 1.2 本仓库模板来源

| 模板    | 目录           | 来源                                     | 引入时间   | 第一次提交                                                                                                                       |
| ------- | -------------- | ---------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Default | `web/src/`     | 官方 `QuantumNous/new-api` 的 `web/src/` | 随上游同步 | —                                                                                                                                |
| Classic | `web/classic/` | 从已有前端迁移而来（R100）               | 2026-04-28 | `a42b397607780bb0f2dbc851ae0c01043244691d`（`🚀 feat: launch v1.0 — next-generation frontend built from the ground up (#4265)`） |

Classic 来源证据：`git show --format= --name-status a42b397607780bb0f2dbc851ae0c01043244691d -- web/classic/package.json web/package.json` 显示 `R100 web/package.json → web/classic/package.json`，只证明一次重命名。**Classic 来源未决**：已有上游 shallow fetch（depth=1）记录不足以排除迁移前模板承继自官方旧版本；本轮不补抓远端历史，也不作“上游历史从未有 Classic”的断言。

Classic 技术栈：React（JSX，非 TypeScript）、Vite、Ant Design / Semi UI，与上游 Default 完全不同。

### 1.3 审计基线（完整 SHA）

| 基准点                       | 完整 SHA                                   | 说明                                                                   |
| ---------------------------- | ------------------------------------------ | ---------------------------------------------------------------------- |
| custom-main 远端（审计时）   | `a946865c9b74046d44db046b0fbb84a363551da5` | `Merge pull request #395 from moeacgx/agent/alhe-102-image-task-cors`  |
| 本地用户快照（文档提交父级） | `0f4a9f2c1d86cd87fc15cc364dc2d1920e956e11` | `chore(agent): baseline — uncommitted work from the local directory`   |
| 原审计共同祖先               | `6da5e7fee3e29391e04fd95a1d7a95c08ae59726` | `docs: 记录 maolaoapi .366 逐节点部署与验收`                           |
| 文档初版                     | `074d928748fc733f84400d183fec3508ebb07603` | 仅作交付历史，不代表本轮 head                                          |
| 本轮被退回文档               | `e716fe0e3e0bf5a227e8500a5745ac9ed31bd9ec` | 仅提取指定两份文档作为返修输入                                         |
| 本轮实际工作树起点           | `3f2f818381b63634a4887469ffc044d6b0b31344` | 分支 `agent/agent/alhe-114-4bc7b7e061dd`；提交树与完整文档 base 无差异 |
| 父任务本轮 HEAD              | `c90d019fcc713beff8c899e951459a7e7b287b7c` | 包含用户成果快照，不等同远端 custom-main                               |
| 上游历史固定审查树           | `0409deccfff2b0e8babc28ca552af44d526375bf` | 旧审查 shallow fetch 的固定点；非本轮实时查询                          |

**本地快照 → 固定 custom-main 的差异方向**：运行 `git diff --name-status 0f4a9f2c1d86cd87fc15cc364dc2d1920e956e11 a946865c9b74046d44db046b0fbb84a363551da5` 得到 12 项。A 为目标树新增、M 为修改、D 为目标树删除，仅表示该比较方向，不能作为清理授权。

| 方向 | 路径                                                        |
| ---- | ----------------------------------------------------------- |
| D    | `.agents/skills/dsh-purge-redteam-repo`（子模块）           |
| D    | `.tmp-deploy-364/00_inspect.sh`                             |
| D    | `.tmp-deploy-364/01_check.sh`                               |
| D    | `.tmp-deploy-364/02_prepare.sh`                             |
| D    | `.tmp-deploy-364/02_pull_and_extract.sh`                    |
| D    | `.tmp-deploy-364/03_update_one.sh`                          |
| M    | `VERSION`                                                   |
| M    | `docs/developer/README.md`                                  |
| M    | `docs/workflows/2026-11/06_release_367.md`                  |
| A    | `docs/workflows/2026-11/06_release_368.md`                  |
| A    | `docs/workflows/2026-11/10_image_task_cors_verification.md` |
| A    | `router/image_task_cors_test.go`                            |

**隔离边界**：上述快照包含用户脚本和子模块，不是待发布补丁。本轮仅引入两份已审文档，不搬运原目录快照、整分支或这 12 项差异。保留工作树初始 `AGENTS.md` 修改与 `.multica/` 未跟踪内容，编辑器未保存状态不可观测。主脑后续只集成文档提交。

**共同祖先至固定 custom-main 的文件差异**：`6da5e7fee3e29391e04fd95a1d7a95c08ae59726..a946865c9b74046d44db046b0fbb84a363551da5` 为 53 个文件的历史差异。前述本地用户快照至固定 custom-main 的 12 项差异是另一比较范围；两组路径交集为 6 项，互不构成包含关系。快照差异只用于隔离用户本地成果，不代表共同祖先后的正式历史审计已覆盖这些路径，也不授权清理。

**上游 shallow fetch 限制**：原记录 `git fetch upstream main --depth=1` 仅抓取当时一条提交，不能据此完成上游历史归属或全量增量审计。固定树的目录可直接检查；更早历史须另行取得足够历史后核对。本轮未 fetch，不把历史基线称为实时最新。

### 1.4 主题选择机制

- 存储：`model/option.go`，key `theme.frontend`，值为字符串 `"default"` 或 `"classic"`
- 加载：`setting/system_setting/theme.go` → `common.SetTheme()`（原子写）
- 读取：`common.GetTheme()`（原子读）
- 路由分发：`router/web-router.go` `setThemeWebRouter()`，`serveSiteIndexForRoot()` 和 `NoRoute` 处理函数均按 `GetTheme() == "classic"` 决定返回哪个 `index.html`
- 静态资源：`common/embed-file-system.go` `ThemeAwareFS.Open()` 同样按主题选择文件系统
- 代码默认值：`themeSettings.Frontend = "default"`（`setting/system_setting/theme.go` 第 17 行）
- **API 暴露**：`/api/status` 返回 `theme` 字段，前端加载时读取并据此发请求

**线上实际值**：**未验证**。生产实际主题由后台数据库 `options` 表 `theme.frontend` 字段决定。若为空或未设置则回退代码默认值 `"default"`。读取须通过 cloudssh-agent 执行只读 SQL，本次未操作，标记为未验证。**不得把代码默认值等同生产实际状态。**

### 1.5 主题切换与回滚契约（已更正）

**切换接口**：受 `RootAuth` 中间件保护（`router/api-router.go:329-333`）：

```
PUT /api/option/
Authorization: Bearer <root-token>
Content-Type: application/json

{"key":"theme.frontend","value":"classic"}
```

切换回 Default：

```
{"key":"theme.frontend","value":"default"}
```

**处理链**：`controller.UpdateOption` → `model.UpdateOption` → `system_setting.UpdateAndSyncTheme()` → `common.SetTheme()` 原子更新当前进程内存，当前进程无需重启即可感知；**多实例传播未验证**（`common.SetTheme` 仅证明本进程即时生效，不能承诺全站所有实例实时一致）。`key` 按 `.` 分割为 `configName=theme`、`configKey=frontend`，通过 `config.GlobalConfig.Get("theme")` 匹配 `ThemeSettings` 结构体后更新。

前端（Default `web/src`）和 Classic（`web/classic`）的系统设置页均调用此接口，经路由核查 `router/api-router.go:329-343` 确认。

---

## 2. 历史漏项清单（三类能力矩阵）

> 两种范围分别记录：① 固定差异审计 `6da5e7fee3e29391e04fd95a1d7a95c08ae59726..a946865c9b74046d44db046b0fbb84a363551da5` 为 11 个提交、53 个文件；② 能力来源追溯跨越更早历史（如 2026-08/09 的 B2/B5/C3），不声称都属于这 11 个提交。本轮源码静态核查以被退回 head 为参照；`git diff --name-only e716fe0e3e0bf5a227e8500a5745ac9ed31bd9ec 3f2f818381b63634a4887469ffc044d6b0b31344 -- web router controller service setting` 无差异。A 类表示已找到静态实现，不代表本轮运行或端到端验收通过；原有提交短号已展开为完整 SHA，重点返修项另补实际链路。

### 类别 A：已同步双模板，能力对齐

| 功能                                  | 核查证据                                                                                                                                                                                                                                                                                           | Default 路径                                           | Classic 路径                                                            |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------ | ----------------------------------------------------------------------- |
| 工单中心与工单管理                    | `8cfd88156fed88492fbeb85f35aeb32a15222698`（后端）/ Default+Classic 各有提交                                                                                                                                                                                                                       | `web/src/features/tickets/`                            | `web/classic/src/pages/Tickets/`                                        |
| 待处理工单看板与浮动提醒              | `c463581d4d4b8343234dcd6cfef6014d1539136e`（Default）/ `cd73e59a73329c371e495ca167caa2573660220e`（Classic）                                                                                                                                                                                       | `web/src/features/dashboard/`                          | `web/classic/src/pages/Dashboard/`                                      |
| 工单四状态概览                        | `2495573e51f4c3a40199297fa495769d5842b6ac`（Default）/ `e7fb55817d9d08f403eb8a7f205d621753a592b5`（Classic）                                                                                                                                                                                       | 同上                                                   | 同上                                                                    |
| 工单通知格式编辑器                    | `513fbb255a57742d55eb59105971c7fe3db7a27c`（双模板）                                                                                                                                                                                                                                               | `web/src/features/notification-center/`                | `web/classic/src/pages/NotificationCenter/`                             |
| 技能合集/分类/导入                    | 双模板各有提交                                                                                                                                                                                                                                                                                     | `web/src/features/prompt-center/`                      | `web/classic/src/pages/PromptCenter/`                                   |
| GitHub 技能来源与更新                 | `8c10f295385fffdc40c2beb9517523a944d95ab7`（Default）/ `feff3c3b8a1e04784be451068c36ccd8beb28ee8`（Classic）                                                                                                                                                                                       | 同上                                                   | 同上                                                                    |
| 令牌分组数量上限                      | `f2624befe9bbaf4e54f1d6bd144413b483045446`（后端）/ `be42e85d694ec75a92a18c986863986561c698b8`（双模板）                                                                                                                                                                                           | `web/src/features/keys/`                               | `web/classic/src/pages/Token/`                                          |
| 令牌分组默认不限制                    | `5884ec2b229a1b74bdf58b62b752ad02c7abd403`（双模板）                                                                                                                                                                                                                                               | 同上                                                   | 同上                                                                    |
| 令牌分组渠道限定                      | `225fd34773d351f02cb97b70ca2a0568b860ed76`（双模板）                                                                                                                                                                                                                                               | 同上                                                   | 同上                                                                    |
| 技能白名单分组选择                    | `e19385c7015ac27fe9a1b2be38c35411c6744a84`（双模板）                                                                                                                                                                                                                                               | `web/src/features/prompt-center/`                      | `web/classic/src/pages/PromptCenter/`                                   |
| 技能分类删除                          | `805e852bf562bae1315dfdc8c073277973179586`（双模板）                                                                                                                                                                                                                                               | 同上                                                   | 同上                                                                    |
| 使用日志技能筛选与角标                | `6b00cc1db21acfb8a963e0d6f14f927ff5a09e34`（双模板）                                                                                                                                                                                                                                               | `web/src/features/usage-logs/`                         | `web/classic/src/pages/Log/`                                            |
| 当前连接轮询布局稳定性                | `c362df9179f717765795555abb50f2646362e588`（Default）/ `3ee546f7dda3a7a9b9a7d03ed89aca6ee3feca08`（Classic）                                                                                                                                                                                       | `web/src/features/dashboard/`                          | `web/classic/src/pages/Dashboard/`                                      |
| 充值支付限流提示                      | `a2bd8a02791a31338d481055675fb9a41a1365b4`（双模板）                                                                                                                                                                                                                                               | `web/src/features/wallet/`                             | `web/classic/src/pages/TopUp/`                                          |
| 令牌查询限流隔离                      | `2686ec90d9c7848a9f872ddcf40d6749999f3eef`（双模板）                                                                                                                                                                                                                                               | 后端 `middleware/rate-limit.go`                        | 同上                                                                    |
| **系统负载均衡实例展示**（原 B7）     | 实际挂载与请求见下方 B7 链路；不以文件存在或体积作覆盖证据                                                                                                                                                                                                                                         | `/system-info` → `SystemInfo` → `SystemInstancesPanel` | `/console/setting` → 性能设置 → `SystemInstancesPanel`                  |
| **模型广场分组动态价格展示**（原 B6） | Default `web/src/routes/pricing/$modelId/index.tsx:22` 导入 `ModelDetails`；`web/src/features/pricing/components/model-details.tsx:1051-1110` 实现 `isDynamicPricingModel` 判断、`getDynamicPricingTiers` 阶梯、`getConfiguredGroupRatio` 分组倍率、`groupRatioMultiplier: ratio` 传入动态价格计算 | `web/src/features/pricing/`                            | Classic commit `8b0f8eb4f3d3f572a544bff0a1e35c570874417b`（2026-09-08） |
| **渠道通知按上游错误过滤**（原 C3）   | 两套表单均提交 `filter_config.error_keywords`，后端统一匹配；完整链路和 SHA 见下文                                                                                                                                                                                                                 | `/notification-center` → 任务编辑表单                  | `/notification-center` → `NotificationCenter` 任务表单                  |

#### B7 实例展示与 C3 通知过滤的实际链路

- **B7**：Default `web/src/routes/_authenticated/system-info/index.tsx` 校验 SUPER_ADMIN，挂 `features/system-info/index.tsx`，后者挂 `SystemInstancesPanel`；面板 `queryFn` 调用 `listSystemInstances` → `features/system-info/api.ts` → `GET /api/system-info/instances`。Classic `/console/setting` → `components/settings/PerformanceSetting.jsx` → `pages/Setting/Performance/SettingsPerformance.jsx:533` 挂同名面板，面板 `:200` 请求同一 API。`router/api-router.go` 的 `systemInfoRoute.Use(RootAuth())` 保护实例查询和删除。两者均展示实例及 CPU/内存等资源信息，入口位置不同，不能因 Default 不在设置页而定性缺失。Classic 历史来源 `3f9f2771e3df756771d1340b9c4004b4f0f8db92`；不据此承诺每个指标、刷新时序和视觉布局完全相同，未做浏览器/多实例运行验证。
- **C3**：Default 路由 `web/src/routes/_authenticated/notification-center/index.tsx` → `features/notification-center/index.tsx` 的任务表单，`CHANNEL_DISABLED_EVENT` 时归一化 `filter_config` → `createNotificationTask` / `updateNotificationTask`（同目录 `api.ts`）；Classic `App.jsx` 的 `/notification-center` → `pages/NotificationCenter/index.jsx`，`:283-300` 归一化并保存。最终请求均为 `POST /api/notification/tasks` 或 `PUT /api/notification/tasks/:id`，载荷包含 `filter_config.error_keywords` / `status_codes`；`router/feature-router.go:78-96` 统一 `RootAuth` 和写操作限流。后端 `controller/notification.go` 校验去空关键词，`model/notification.go` 的 `NotificationTaskFilterConfig.Matches` 对 `error_message` / `reason` 忽略大小写匹配。双模板均有状态码、错误关键词输入，布局不同而契约相同。历史 `5fad5573044737e7af55de27349224d67072c862` **同时修改后端和 Default 表单/类型/翻译**（`git show --stat`），旧文“仅后端”错误；Classic 恢复来源 `de98d05c527d04c5cc65f1cb9bdc744667764aab`。本轮仅静态核查，未发送通知或验证真实投递。

### 类别 B：确认缺口——Default 缺失，Classic 已实现

| #      | 功能                                                    | Classic 证据                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | Default 现状                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | 优先级 |
| ------ | ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| **B2** | **渠道单 key 转多 key**（仅路径③缺口）                  | 渠道页编辑 → `EditChannelModal.jsx:2200-2210` → `multiKeyEditing.js` → `PUT /api/channel/`。`canConvertChannelToMultiKey` 同时检查 `isEdit && !isMultiKeyChannel && !isIonetLocked`、有效 `originalType`，且来源/目标 `type` 均非 41/57；`buildChannelKeyUpdateFields` 按敏感权限和转换状态生成字段。历史 `11c8c06bbc58a696cfbb686f2f73f388d3be1070`，后续兼容 `c2394314a42b5249cfad465eff3c7c58b8ab050b`                                                                                                                              | 渠道页 → `channel-mutate-drawer.tsx` → `use-channel-mutate-form.ts` → `channels/api.ts`：①新建多 key 已有；②已有多 key 编辑已有，`key_mode` 条件是 `canEditSensitive && isMultiKeyChannel && data.key?.trim() && data.key_mode`，而 `multi_key_mode` 仅需前两项，值取 `data.multi_key_type`，留空新 key 仍可改策略；③单 key 编辑的 transform/hook 均未生成 `convert_to_multi_key`，转换入口缺失。更新统一为 `AdminAuth + RequirePermission(ChannelWrite)`，实际敏感变更另验 `ChannelSensitiveWrite`（`router/channel-router.go:20-35,55-56`、`controller/channel.go:1135-1138`）；取 key 的 RootAuth 不用于替代更新权限。P1 仅补③ | 高     |
| **B5** | **管理员标签管理 + 普通用户令牌标签筛选**（两条权限链） | 管理：`/console/setting` → `components/settings/RatioSetting.jsx:111` → `pages/Setting/Ratio/GroupTagSettings.jsx:199` 挂 `GroupTagEditor` → `GET/POST /api/group/tags`、`PUT/DELETE /api/group/tags/:id`（`AdminAuth`）；`GroupRatioSettings` → `components/GroupTable.jsx` 挂标签筛选。用户：令牌页 → `EditTokenModal.jsx:326` 挂 `GroupTagFilter`，`:715-721` 从 `GET /api/user/self/groups` 的顶层 `group_tags` 读取可用标签 → 本地过滤分组 → `POST/PUT /api/token/` 提交选中分组。来源 `5e2c3c78b19ed192736963d131367d75b9e5698b` | `features/keys/index.tsx` → `ApiKeysDialogs` → `api-keys-mutate-drawer.tsx:144` → `lib/api.ts:getUserGroups` 请求同一 `/api/user/self/groups`，但返回类型与消费仅用 `data`，未消费 `group_tags`；prompt-center 的管理员标签 GET 不能代替令牌筛选或标签 CRUD。`router/api-router.go:151-156` 对用户组查询用 `UserAuth`；`controller/group.go:GetUserGroups` 仅为可用 group ID 取标签，标签不扩大授权。P2 分开补管理与用户选择；标签只筛选，分组展示用当前名称，提交保持既有稳定 ID/code                                                                                                                                            | 高     |

### 类别 C：经五路径逐项核查无确认缺口

原列 C2（福利活动管理）①–④双模板均有实际入口和请求；⑤流水可见入口由双模板一起移除，组件/API 保留。无确认缺口，不恢复入口。

#### C2 福利活动管理逐路径核查

| 路径              | Default 证据                                                                                                                                                                                                                                                                                                       | Classic 证据                                                                                                                  | 结论                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| ① 管理员活动 CRUD | `/redemption-codes` 路由 → `features/redemption-codes/index.tsx:60` 的 Tab 挂 `BenefitActivitiesPanel` → `:563` 挂 `BenefitActivityForm`；提交 `createAdminBenefitActivity` / `updateAdminBenefitActivity`，同目录 `api.ts` 最终 GET/POST `/api/benefit/admin/activities`、PUT `/api/benefit/admin/activities/:id` | `/console/redemption` → `pages/Redemption/index.jsx:56` 第三 Tab 挂 `BenefitActivitiesPanel`，`:214,396-400` 请求相同活动接口 | 双模板静态覆盖；管理接口 `AdminAuth`                                                                          |
| ② 批量删除        | 同一面板的选择操作 → `deleteAdminBenefitActivities(ids)`（`:223`）→ `api.ts:272-275`，`DELETE /api/benefit/admin/activities/batch`，body `{ids}`                                                                                                                                                                   | 面板 `:664` 挂 `BenefitActivityBatchActions`，`:52-53` 调用同一 DELETE                                                        | 双模板静态覆盖；管理员+关键操作限流，服务端判可删除状态                                                       |
| ③ 券管理          | 面板 `:518` 选活动，`:587` 挂 `BenefitVouchersSheet`；sheet `:111` 查询、`:153` 作废 → `api.ts` GET `/api/benefit/admin/activities/:id/vouchers`、POST `/api/benefit/admin/vouchers/batch-void`                                                                                                                    | 面板 `:745` 挂 `BenefitVoucherTable`；`:77` 查询同一券列表，`:126-132` 批量/单券作废 API                                      | 双模板静态覆盖；`AdminAuth`，批量作废另有限流；Default 使用批量接口处理选择，Classic 可选单券接口             |
| ④ 用户领取        | `/benefits` 路由挂 `UserBenefits`（`features/benefits/index.tsx`）→ query 加载活动/券，挂 `BenefitSummary` / `UserVoucherCard` / `ClaimableActivityCard` → `claimBenefitActivity`；`api.ts` GET `/api/benefit/activities`、GET `/api/benefit/vouchers`、POST `/api/benefit/activities/:id/claim`                   | `App.jsx` 的 `/console/benefits` → `pages/Benefits` → `useBenefitsData`，`:34-35` 读活动/券，`:61` 领取，挂同类用户卡片       | 双模板静态覆盖；`UserAuth`，服务端重新判断领取资格及所有权                                                    |
| ⑤ 流水视图        | `AdminVoucherLedger` 仅定义在 `admin-voucher-ledger.tsx`，`rg -n AdminVoucherLedger web/src` 无外部挂载；用户卡片和管理员券列表均无可见流水入口                                                                                                                                                                    | 用户福利页和管理员券列表同样移除入口，流水组件仍保留                                                                          | `87f481805fe950320903cb18a8d5a547dad8a6e6` 与 `docs/developer/benefit-vouchers.md` 明确双模板移除；不恢复入口 |

**C2 权限/API 依据**：`router/api-router.go:446-472` 分开 `UserAuth` 和 `AdminAuth` 路由；流水 GET `/api/benefit/vouchers/:id/ledger` 与 `/api/benefit/admin/vouchers/:id/ledger` 继续保留。五路径均不列入补齐范围；本轮没有运行 CRUD、领取或计费测试，不将静态路径核对表述为运行验收。

### 未决审计项（需进一步查清，不列入实现范围）

| #   | 未决项                                                | 未决原因                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | 下一步                                                                                                                                                                                                                                                                                    |
| --- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Classic 迁移前模板历史**                            | shallow 历史不完整；`a42b397607780bb0f2dbc851ae0c01043244691d` R100 只证明迁移                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | 如需明确归属，另取足够上游历史追溯 `#4265`；不阻塞已有确认缺口                                                                                                                                                                                                                            |
| B4  | **供应商自定义 Logo URL 的端到端等价性**              | Classic 模型管理 `components/table/models/index.jsx:119` 挂 `EditVendorModal` → POST/PUT `/api/vendors/` 提交 `icon`；`router/api-router.go:548-556` 为 `AdminAuth`。`7d3df6b93587e33eabfcc4b11f19fa25b2e82a8d`（合并 PR #265：`b221905e18e41bf38ca806a49e84d3213b299919`）新增 HTTP/HTTPS 图片语义，`helpers/render.jsx:getLobeHubIcon` 处理 URL。Default `features/models/components/models-dialogs.tsx` 挂 `VendorMutateDialog` → `models/api.ts:createVendor/updateVendor` 请求相同 API，`icon` 输入仍提示内置图标名。渠道 `providerLogo` prop 不能证明供应商 URL 保存后的最终渲染相同 | 已有专题 `docs/workflows/2026-09/23_classic_vendor_logo_url.md` 记录当时 Default URL 为首字母兜底；本轮尚未完成当前 Default 所有供应商/模型/渠道展示面的消费链核对，因此撤回“已覆盖”，暂不定为全局确认缺口、不安排实现。后续用相同 URL 记录核对保存、刷新、失败兜底及各挂载处，再固定范围 |
| C1  | **DeepSeek V4.1 Flash / Search / Fetch 的模板操作链** | 历史 `383524f22845f2d79576d350bff259457d33df57` 适配本地 relay；`router/relay-router.go:90-144` 可追 `TokenAuth` / 限流 / `Distribute` → POST `/v1/tools/search`、`/v1/tools/fetch` → `controller.Relay(RelayFormatDeepSeekTools)` → `relay/channel/deepseek/adaptor.go` 的上游 `/v1/tools/*`。但这仅证明共享协议入口；Classic 渠道常量和提交标题不能证明双模板模型选择、操练场最终请求及可见结果                                                                                                                                                                                          | 撤回“两套前端透明、已覆盖”结论；Default/Classic 的实际 UI 挂载、模型选择到最终请求、可见差异仍未闭环。另行核对两套操练场和独立原生 API 调用，包含模型别名与工具结果，不进入本轮确定实施范围                                                                                               |

---

## 3. 实施方向与切换计划

### 3.1 方向确认（已无需拍板）

依据 2026-11-11 01:28 大哥纠正（父任务评论 `01a1c681-c5a0-773a-91a3-6b22dc4517b2`）：

**正确顺序**：

1. 整理历史漏掉的功能（本阶段，ALHE-114）
2. 在官方 Default（`web/src`）补齐确认缺口并独立验收（ALHE-113）
3. 最后由大哥确定名字后复制一份独立命名，再做大改

**当前阶段禁止**：复制、命名或开始模板大改。Classic 核查共享契约与回归影响，不自动扩大反向补功能范围，也不主动降为 legacy。

> 注：2026-11-11 01:27 的"Classic 降 legacy"拍板已被 01:28 纠正明确作废，不作为计划或实现依据。

### 3.2 分批切换顺序

| 阶段                   | 内容                                                                 | 前置条件                                                          | 风险                                                                                                                                                                                                                                                                                                                               |
| ---------------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **P0（已有，代码层）** | Default 作为代码默认主题（`"default"`），切换 API 已就绪             | —                                                                 | 线上实际值未验证，不代表已完成切换                                                                                                                                                                                                                                                                                                 |
| **P1**                 | Default 仅补 B2 路径③单 key 转多 key                                 | ALHE-114 独立复审通过，主脑核准 ALHE-113 范围；实现后另行独立验收 | 高：沿用 Classic 的 `originalType` / 目标 `type` 双重排除、`isIonetLocked`、编辑态/单 key 状态及敏感权限资格；仅显式转换提交 `convert_to_multi_key`，保留已有多 key 的 hook 逻辑，分别处理新 key 的 `key_mode` 和策略字段。取消转换、关闭重开或切换渠道时清理转换/模式残留；后端继续拒绝无权或非法转换，不把取消后字段带入普通更新 |
| **P2**                 | Default 补 B5 管理员标签 CRUD 与普通用户令牌标签筛选                 | P1 完成并独立验收，主脑核准 P2                                    | 中：管理 API 用 AdminAuth，用户选择只从 UserAuth 的 `/api/user/self/groups` 消费 `group_tags`；标签不授权、不替代分组稳定标识，显示名称与 code 不同时验证展示/提交                                                                                                                                                                 |
| **P3**                 | 汇总 Default 确认缺口验收，检查 Classic 共享契约回归；未决项另行取证 | P1/P2 独立验收及主脑确认                                          | B4/C1 与来源历史不预设为实现；C2 双模板①–④已有，⑤共同移除，不恢复流水入口。Classic 不自动反向补功能，不以长期双模板决策阻塞当前范围                                                                                                                                                                                                |
| **P4**                 | 最后复制独立命名并开始大改                                           | Default 补齐且独立验收通过，大哥在需要时确定名字及后续范围        | 当前不复制、不命名、不大改                                                                                                                                                                                                                                                                                                         |

### 3.3 回滚方案

- 主题切换通过 `PUT /api/option/` 写入数据库并调用 `common.SetTheme()` 原子更新当前进程内存，当前进程无需重启即可感知；**多实例传播未验证**（`common.SetTheme` 仅证明本进程即时生效，不能承诺全站所有实例实时一致）。若原始状态为 Classic，回滚只需管理后台操作或发以下请求：
  ```
  PUT /api/option/   (Authorization: Bearer <root-token>)
  {"key":"theme.frontend","value":"classic"}
  ```
  若原始状态为 Default，回滚同理改为 `"default"`。**执行前须记录切换前实际主题值，不得以代码默认值代替。**
- 功能补丁通过独立 PR 合入，每个 PR 独立可回滚
- 前端补丁回退须保留渠道 ID、多 key 服务端状态和标签关联；UI 回退不会把已转换渠道逆转为单 key。
- B5 复用现有标签 API，不计划数据库迁移；若实施发现必须改后端契约，先重新固定范围及兼容性。B6 已有静态展示，不在补齐范围。

### 3.4 验收矩阵

| 验收项                            | Default         | Classic        | 说明                                                                                                              |
| --------------------------------- | --------------- | -------------- | ----------------------------------------------------------------------------------------------------------------- |
| 管理员登录 / 认证                 | 已有            | 已有           | 双模板 `/api/status` theme 字段正确                                                                               |
| 渠道管理基础 CRUD + 多 key        | 仅路径③待 P1 补 | 静态已有       | B2 验证来源/目标特殊类型、io.net 锁、敏感权限、追加/覆盖、留空新 key 改策略、取消/关闭重开残留；①②保留既有行为    |
| 供应商自定义 Logo URL             | 未决            | 历史实现已定位 | B4 从 A 类退出；需完成最终消费链，`providerLogo` 不能证明等价，不安排确定实施                                     |
| 令牌分组上限与默认不限制          | 已有            | 已有           |                                                                                                                   |
| 令牌分组标签管理及用户选择        | 待 P2 补        | 静态已有       | B5 分别验 AdminAuth CRUD 与 UserAuth 的 `group_tags` 本地筛选；不得把用户链改接管理员 API，展示名称不替换提交标识 |
| 技能中心（合集/更新/删除）        | 已有            | 已有           |                                                                                                                   |
| 工单全流程                        | 已有            | 已有           |                                                                                                                   |
| 充值/支付限流提示                 | 已有            | 已有           |                                                                                                                   |
| 福利活动管理（新 UI）             | ①–④静态已有     | ①–④静态已有    | C2 双模板均移除⑤可见入口，组件/API 留存；Default AdminVoucherLedger 未挂载，不恢复入口                            |
| 模型广场分组动态价格              | 已有            | 已有           | B6 核查确认 Default 已覆盖（`model-details.tsx:1051-1110`）                                                       |
| 系统实例展示                      | 静态已有        | 静态已有       | B7 实际页面挂载到 RootAuth 的相同 API；运行刷新和多实例未验证                                                     |
| 渠道通知上游错误筛选              | 静态已有        | 静态已有       | C3 表单到 RootAuth 保存及后端匹配链已核对，真实投递未验证                                                         |
| DeepSeek V4.1 / Search / Fetch UI | 未决            | 未决           | C1 共享 relay 入口不能代替双模板 UI 到最终请求的证据，不列确定实施                                                |
| i18n 完整性                       | 后续按改动验证  | 后续按回归验证 | 历史修复不代表本轮全量通过；B2/B5 新文案需七语同步                                                                |

---

## 4. 发版自动化与生产部署核查

### 4.1 Release 自动化触发条件

| 工作流文件                           | 触发条件                                                   | 产物                                                              |
| ------------------------------------ | ---------------------------------------------------------- | ----------------------------------------------------------------- |
| `.github/workflows/release.yml`      | `push: tags: '*'`（排除 `*-alpha*`）或 `workflow_dispatch` | Linux amd64/arm64 二进制 + `checksums-linux.txt` → GitHub Release |
| `.github/workflows/docker-build.yml` | `push: tags: '*'`（排除 `nightly*`）或 `workflow_dispatch` | GHCR 多架构镜像（`ghcr.io/moeacgx/newapi-mao:TAG`、`:latest`）    |

### 4.2 是否触发生产部署

**代码层证据**：两个工作流末步均为上传产物（`softprops/action-gh-release`）或推送 GHCR manifest（`docker buildx imagetools create`），**不包含** SSH 远端执行、`docker pull`、`docker compose up`、Kubernetes rollout 或外部 webhook 调用。代码层确认发版不直接触发生产部署。

**未验证项**：是否存在外部 Watchtower、Portainer 或运营商 CD 钩子订阅 GHCR `:latest` 自动拉取。此类配置在代码库中不可见，需通过 cloudssh-agent 检查生产节点容器运行配置确认。本次未操作，**不宣称生产部署风险已排除**。

---

## 5. 文档变更登记

本文档为 ALHE-114 阶段一交付件，已更新 `docs/developer/README.md` 索引。

### 5.1 格式与链接核查说明

| 项目             | 命令                                                                                                                                        | 结果                                                                                                                               |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| 完整交付空白检查 | `git diff --check 0f4a9f2c1d86cd87fc15cc364dc2d1920e956e11 HEAD`                                                                            | 提交后对确定 head 重跑；实际完整 SHA 与 exit code 记录于本轮交付评论，不循环将自身 SHA 写进自身提交                                |
| 工作区/暂存区    | `git diff --check` / `git diff --cached --check`                                                                                            | 返修文件编辑后两项 exit 0；暂存及提交后再查，结果绑定交付评论。保留初始 AGENTS.md 与 .multica/，不提交无关成果                     |
| 仓库格式器       | `bun x --no-install oxfmt -c web/.oxfmtrc.json --check docs/workflows/2026-11/11_template_audit_switching_plan.md docs/developer/README.md` | Bun 1.4.4，exit 1：`Could not find an existing 'oxfmt' binary`。格式未验证，未安装依赖                                             |
| 新增相对链接     | Python 提取两份文档 Markdown 链接，按 base 与候选文本的多重集差集识别新增，按文档父目录解析目标并检查存在                                   | 419 个相对引用中新增 1 个计划索引链接，目标存在，新增缺失 0；81 个缺失目标均继承于 base README，不扩为返修。提交后对固定 head 重跑 |

本轮只修改文档，不运行业务构建、单元测试、浏览器、HTTP、生产或多实例验证，无 PR/CI 通过声明。三类矩阵的“已有”均指静态证据；执行自检不等于独立验收通过。ALHE-113 在本项复审通过并经主脑核准前继续 backlog。

---

## 附：关键路径文件索引

| 文件                                                                  | 作用                                                           |
| --------------------------------------------------------------------- | -------------------------------------------------------------- |
| `router/web-router.go`                                                | 主题路由分发逻辑                                               |
| `router/api-router.go:329-333`                                        | `PUT /api/option/` RootAuth 保护                               |
| `controller/option.go:116-121`                                        | `OptionUpdateRequest{Key, Value}` 结构                         |
| `model/option.go:960-999`                                             | `theme.frontend` key 分层配置处理                              |
| `common/constants.go`                                                 | `GetTheme()` / `SetTheme()` 原子操作                           |
| `common/embed-file-system.go`                                         | `ThemeAwareFS` 静态资源按主题分发                              |
| `setting/system_setting/theme.go`                                     | 主题设置读写与同步（默认值 `"default"` 在第 17 行）            |
| `controller/misc.go:101`                                              | `/api/status` 暴露 `theme` 字段                                |
| `web/src/`                                                            | Default 模板（官方上游）                                       |
| `web/classic/`                                                        | Classic 模板（来源未决；仅固定上游树不含该目录，历史不能外推） |
| `web/src/features/channels/hooks/use-channel-mutate-form.ts:107-122`  | B2 最终更新载荷条件                                            |
| `router/channel-router.go` / `controller/channel.go:1135-1138`        | B2 更新与敏感权限分层                                          |
| `web/src/features/models/components/dialogs/vendor-mutate-dialog.tsx` | B4 供应商管理入口；最终 Logo 展示链未决                        |
| `web/src/features/system-info/components/system-instances-panel.tsx`  | B7 核查依据                                                    |
| `web/src/features/notification-center/index.tsx:757`                  | C3 核查依据                                                    |
| `web/classic/src/components/common/group-tags/GroupTagFilter.jsx`     | B5 Classic 实现                                                |
| `.github/workflows/release.yml`                                       | Release 自动化（无生产部署）                                   |
| `.github/workflows/docker-build.yml`                                  | Docker 镜像推送（无生产部署）                                  |
