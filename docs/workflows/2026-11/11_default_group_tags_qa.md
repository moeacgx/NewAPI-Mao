# ALHE-113 P2 分组标签独立验收

独立验收结论：`verdict=pass`。本轮检查未发现阻断范围内交付的缺陷。范围仅 P2/B5 和新组合版本中的 P1 回归；P3 集成与发布评估由主脑负责，本记录不代表生产主题切换或整个 ALHE-113 完成。

## 版本与独立性

- 正式项目资源：`moeacgx/NewAPI-Mao`，目标 `custom-main`，正式本地目录 `D:/脚本程序/开源程序二开/newapi`。本轮未修改正式目录。
- P2 被测 head：`ffe87461b566dec6ea9119971d5d51f07b83a1f9`，直接 base/P1：`e91adf6743fd29a7a907aae9fc47cc9720532744`；实现分支 `agent/agent/alhe-113`。P2 单提交 32 文件，未改后端、Classic、P1 渠道实现或依赖锁文件。
- 固定历史正式对照：`a946865c9b74046d44db046b0fbb84a363551da5`，没有宣称它是实时远端最新版本。实现起点 `95efd0ae40f57eb7d6df77c65ee827e1946896af` 中的 12 项无关快照差异仍须隔离；主脑只集成 P1/P2 单提交任务补丁，不能整分支合入。
- 实现者为通用执行 Agent `8acd4f4b-b0ed-4b94-9992-6f3a7a3a10dc`，独立验收者为 `e80d1d97-c160-4c22-beec-940b9b31642a`。本 Agent 未参与该实现 head。
- 验收分支保持 `agent/agent/alhe-113-745fc06d9dc2`，起点 `e36f5b7f54580e8ab7e388b6ea5ede13ebac76b1`；仅添加验收文档，不集成业务代码。原有 `AGENTS.md` 和 `.multica/` 内容保留，编辑器未保存状态不可观测。
- 从固定 head `git archive` 导出完整源码至任务临时目录 `multica-task-2315852412/alhe113-ffe8746`。32 个交付文件逐项 hash-object 与 Git blob 相同；依赖分别按两份锁文件 frozen 安装。测试、构建和独立缓存均位于本轮任务临时目录。
- 已核对正式任务描述、父项 01:28 纠正、主脑 P2 核准、P2 交付线程、已有 P1 独立报告和项目正式资源，查询全部 10 个项目工作项。没有新建重复任务、恢复已取消事项或改变既有范围。

## 契约及验收矩阵

| 标准                    | 审查与运行证据                                                                                                                                                                                                                                                                    |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 管理员 CRUD 与权限      | `GroupTagSettings`/`GroupTagEditor` 使用既有 GET/POST `/api/group/tags`、PUT/DELETE `/api/group/tags/:id`。普通用户不渲染或请求管理操作。后端既有 `TestGroupTagRoutesRequireAdminAndSupportLifecycle` 通过真实 Gin 路由、认证头和隔离 SQLite 验证匿名/普通用户拒绝及管理员 CRUD。 |
| 实际页面门禁            | Default 入口 `/system-settings/billing/group-pricing` 沿用 `system-settings/route.tsx` 的 SUPER_ADMIN 门禁；组件与后端 API 的 AdminAuth 能力并不意味着普通管理员能访问系统设置页。本次未扩大页面权限，符合保留原门禁的实施范围。                                                  |
| CRUD 异常与草稿         | 组件用例覆盖业务保存失败保留字段、再次保存成功、删除必须确认、删除失败保留标签和确认框、加载失败禁用新增并可重试，以及非法 Logo 不发请求。保存状态禁用表单和取消，mutation 成功刷新管理及用户分组缓存。                                                                           |
| 管理表筛选及刷新        | 真实 `GroupRatioVisualEditor` 测试覆盖标签、全部、未分类、空结果；修改可见组仍回传完整分组集合，不丢隐藏行；删除活动标签后回到全部，标签重新出现不会恢复旧选择。                                                                                                                  |
| 普通用户标签来源        | 只消费 UserAuth `/api/user/self/groups` 顶层 `group_tags`。标签按 ID 与既有候选组取交集，不构造新授权候选；空标签隐藏，auto 保留可达。实际令牌创建/编辑测试断言没有请求管理员标签 API。                                                                                           |
| 已选值与提交契约        | 跨标签选择不会清除已有选择；最终请求仍使用 `group=stable_a,stable_b`、`group_ids=[11,12]` 和 `group_mode=explicit`，不提交 `group_tags`。编辑请求保留令牌 ID；普通用户不提交 `group_channel_ids`。                                                                                |
| 当前显示名、alias、删除 | 管理与令牌测试使用名称不同于 code 的数据；管理改名立即展示新名称，删除组仅在无法解析时回退 `Deleted (id)`，编辑移除已删除绑定。令牌历史 alias 由详情解析到当前名称与稳定 code，保留既有历史引用策略；后端名称/alias 与删除清理测试补充共享契约。                                  |
| 令牌初始化修复          | 新增实际抽屉测试保护服务端初选不被旧默认值清理 effect 覆盖；`isFormInitialized` 门禁仅改变清理时机，不改变授权集合或 ID/code 提交规则。                                                                                                                                           |
| 组件与可访问性          | 沿用 Card、Dialog、Button 等组件，有边框、背景、圆角和错误区；标签按钮测试使用 Enter/Space，断言 aria-pressed 与选择一致。Logo 为装饰图片，不发送 Referer，加载失败兜底且 URL 更换后恢复。                                                                                        |
| P1 与 Classic           | P2 未改 P1 渠道源码，仍在新 head 上运行渠道集成回归。Classic 维持原管理、用户标签及多密钥实现，运行标签管理、选组、布局与多密钥权限回归。                                                                                                                                         |

前端测试 mock 网络边界，使用真实组件、表单和请求转换；后端独立使用 Gin/控制器与 SQLite 验证接口和持久化。标签管理员路由包含认证中间件测试，但上述证据整体不等于真实浏览器到完整后端的端到端联调。

## 本轮运行证据

日期 2026-11-11。Bun 1.4.4、Node 26.13.0、Go 1.25.5（本机初始 windows/386，后端命令设 `GOARCH=amd64`）；与仓库 Bun 1.4.1、`web/.node-version` 24.10.0 有差异。测试进程使用 `NODE_OPTIONS=--no-experimental-webstorage`；Go 使用本任务独立 `GOCACHE`。没有修改系统环境或 Git 配置；Git 子进程用 `GIT_CONFIG_COUNT=0` 避开运行环境注入缺失 KEY 的配置。

| 检查                 | 实际命令/结果                                                                                                                                                                                                                                                  |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Default 综合回归     | `bun run test -- src/features/keys src/features/channels src/features/group-tags src/features/system-settings/models src/features/prompt-center/__tests__/group-selector.test.tsx --maxWorkers=1 --testTimeout=15000` exit 0；31 文件、206 用例通过，429.15 秒 |
| Default 类型/lint    | `bun run typecheck` 和全部 20 个变更 TS/TSX 的 `oxlint -c .oxlintrc.json` 均 exit 0；3 条 prefer-number-properties 警告已在父提交确认存在                                                                                                                      |
| Default 构建         | `bun run build` exit 0，Rsbuild 2.1.6                                                                                                                                                                                                                          |
| Classic 多密钥纯契约 | `node --test src/components/table/channels/modals/__tests__/multi-key-editing.test.mjs` exit 0，8 用例通过                                                                                                                                                     |
| Classic 组件回归     | 直接 Vitest 运行标签管理、selection、layout 及 multi-key-strategy 四文件，exit 0；4 文件、37 用例通过，204.15 秒                                                                                                                                               |
| Classic 构建         | `bun run build` exit 0，Vite 5.4.11，3 分 22 秒                                                                                                                                                                                                                |
| 后端共享契约         | 下方 Go 命令 exit 0；model、controller、router 三包分别用时 1.241、0.833、0.708 秒                                                                                                                                                                             |
| 国际化               | `bun run i18n:sync` exit 0；七语 missing/extras 均 0，新增 13 键共 91 个非空值                                                                                                                                                                                 |
| 格式                 | 使用仓库 oxfmt 0.57.0、`.oxfmtrc.json` 和受保护版权头流程，在独立副本检查 32 文件，20 个源码版权头保留，格式差异 0，不影响被测源码                                                                                                                             |
| 差异与链接           | `git diff e91adf6 ffe8746 --check` 通过；对新增 Markdown 行解析相对路径，本次扫描 7 处引用、缺失 0；不宣称全库历史链接有效                                                                                                                                     |

后端命令：

```powershell
go test ./model ./controller ./router -run 'Test(GroupTag|GetGroupDisplayName|FormatGroupDisplayNames|SaveGroupConfigChangesDisplayNameOnly|UpdateChannelMultiKeyConversion|.*Channel.*Permission)' -count=1
```

Classic 组件命令（从 `web/classic/`）：

```powershell
node node_modules/vitest/vitest.mjs run src/pages/Setting/Ratio/__tests__/group-tags.compat.test.jsx src/components/common/group-tags/__tests__/selection.compat.test.jsx src/components/common/group-tags/__tests__/layout.compat.test.jsx src/components/table/channels/modals/__tests__/multi-key-strategy.compat.test.jsx
```

本轮日志位于任务临时目录中的 `default-tests.log`、`classic-tests.log`、`default-build.log`、`classic-build.log`，报告和任务评论记录最终结果，不依赖临时目录长期保留。后端首次编译前仅在新导出目录创建双模板嵌入占位；真实构建单独验证，不能用占位冒充构建。

## 限制与后续

- 未运行全仓测试、真实浏览器视觉/完整响应式手验、性能基准、MySQL/PostgreSQL 实库、真实供应商或生产验收。未合并、发布、部署或复制/命名模板；线上主题、多实例传播、外部自动拉取风险仍未验证。
- 保留 jsdom scrollTo、Classic React/Semi、Browserslist、lottie eval、大 chunk 等提示；不声称零警告。七语历史未译统计仍为 fr 12、ja 37、ru 37、vi 13、zh 5；同步键集无缺失不等于全站所有调用点均有翻译。
- P1 已有独立通过记录；P2 本轮结论只绑定上述完整 head。P3 的父分支集成和发布评估由主脑负责，集成后按实际差异复核，不能把本地验收扩写成已满足发布条件。
- 共享影响：Default 新消费既有标签管理 API 和 self/groups 的 `group_tags`；未新增后端/数据库、模型、密钥认证、用量计费或部署契约。
- NewAPIForDouDi：有相同 Default 缺口且具备稳定分组 ID、标签 CRUD 和 UserAuth 裁剪时值得分享，须确认配套权限与版本。
- NewAPIModifyByGang：分类选组有参考价值，先核对模板、响应结构和令牌 ID/code 语义，不能直接照搬。未向两个项目创建 Issue、PR 或发送消息。
