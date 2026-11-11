# ALHE-113 P1 单密钥转换独立验收

结论：`verdict=pass`，仅针对 P1/B2 单密钥转多密钥。P2 标签管理和筛选尚未开始，须由主脑在本次验收后核准；ALHE-113 不能据此标记 done。本轮不合并、不发布、不部署，也不宣称生产主题已切换。

## 版本、范围与独立性

- 项目正式资源：`moeacgx/NewAPI-Mao`，基线分支 `custom-main`；平台正式本地资源为 `D:/脚本程序/开源程序二开/newapi`，本轮未在原目录修改文件或切换分支。
- 固定正式对照基线：`a946865c9b74046d44db046b0fbb84a363551da5`。未查询实时远端，不能视为远端最新状态。
- 被测提交：`e91adf6743fd29a7a907aae9fc47cc9720532744`，实现分支 `agent/agent/alhe-113`；直接父提交 `95efd0ae40f57eb7d6df77c65ee827e1946896af`。本次审查差异严格为该单提交，共 18 个文件。
- 实现者为通用执行 Agent `8acd4f4b-b0ed-4b94-9992-6f3a7a3a10dc`；验收者为验收 Agent `e80d1d97-c160-4c22-beec-940b9b31642a`，未参与该 head 实现。
- 验收分支保持 `agent/agent/alhe-113-745fc06d9dc2`，起点 `a3da5fcf52e61da1ca90f170d31f155c07a46053`；没有切换到实现分支或合入实现快照。使用 `git archive` 将完整被测提交导出至本轮任务临时目录，所有测试、依赖安装和构建均在导出目录运行。
- 全部 18 个交付文件的导出内容经 `git hash-object --no-filters` 与被测提交对象比较，差异为 0。两套前端均使用自己的锁文件执行 `bun install --frozen-lockfile`，未升级锁文件。
- 已读父任务 01:28 纠正、P1 核准线程、实施交付、正式项目说明和资源；查询项目全部 9 个已有工作项及父项子任务，没有创建重复任务、恢复已取消事项或提前实现 P2。

正式基线与实现起点在所查渠道路径（Default channels、Classic channels、`controller/channel.go`、`router/channel-router.go`）无差异。实现起点相对正式基线仍有 12 项继承快照差异，包括历史部署脚本、版本和文档等，不属于 P1；本次通过不授权将整段任务分支混入正式基线。主脑应使用实现者交付的单提交补丁集成，并在实际集成版本上完成相应检查。

## 逐项验收

| 标准                                  | 实际结果与依据                                                                                                                                                                        | 结论 |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---- |
| 显式单密钥转换，追加与覆盖            | Default 实际 drawer 交互测试经 API 边界捕获 `PUT /api/channel/` 载荷；勾选转换、输入新 key、选择 append/replace 及 polling 后携带明确转换标记和独立模式字段                           | 通过 |
| 编辑态、有效原始类型及特殊类型边界    | 资格、hook、组件测试覆盖新建、已有多密钥、缺失/0/NaN 原始类型、来源和目标 41/57；目标改为 Codex 再切回不会恢复转换选择                                                                | 通过 |
| io.net 锁和服务端拒绝                 | 前端读取详情的原始元数据；hook 拒绝请求。后端转换测试验证伪造清空 `other_info` 也无法绕过托管锁，失败后密钥和渠道状态未变                                                             | 通过 |
| ChannelWrite 与 ChannelSensitiveWrite | Default 组件及 hook 分别验证缺失写权限或敏感权限；非 Root 且两项权限齐全允许转换。后端控制器敏感校验及路由权限注册回归通过。RootAuth 仅保护取 key，未被当作更新权限                   | 通过 |
| 空新 key、已有多 key 策略             | 无新 key 省略 `key`、`key_mode`，仍可更新 `multi_key_mode`；后端验证保留旧密钥，转换默认追加、随机策略正确                                                                            | 通过 |
| 取消、重开、切换记录                  | 交互测试验证取消后普通更新省略转换及模式字段；重开或换渠道清空转换和策略选择；再启用默认 random                                                                                       | 通过 |
| 普通更新及新建多 key                  | 单 key 不显式转换不提交模式；新建沿用 `mode=multi_to_single` 和创建字段，不携带编辑转换标记；已有多 key 追加/覆盖及策略请求保持原契约                                                 | 通过 |
| 服务端持久化与密钥状态                | 隔离 SQLite 控制器测试验证追加去重、覆盖轮询、空覆盖/非法模式拒绝、缺省或 false 不转换、伪造状态无效、已有禁用 key 与轮询索引保留；响应不泄露旧 key                                   | 通过 |
| Classic 共享契约                      | 未修改 Classic 实现；8 个纯契约用例及 2 个实际编辑组件敏感权限用例通过，包含空新 key 不发送 key/key_mode                                                                              | 通过 |
| 文案、组件和文档                      | 使用已有 Checkbox/Form/Select，转换区有边框和背景；组件查询确认可访问名称及禁用状态。新增两条文案七语 14 个非空值；开发索引、能力登记和实施记录同步，保留 P2/来源未决和线上未验证边界 | 通过 |

前端请求载荷测试 mock API 网络边界，后端测试直接调用控制器并检查真实隔离 SQLite 数据；这两类证据分别验证请求和持久化契约，不冒充真实浏览器到完整服务的端到端联调。路由权限用例验证注册契约，本轮未运行完整认证中间件 HTTP 端到端流程。

## 独立运行证据

验收日期：2026-11-11。工具：Bun 1.4.4、Node 26.13.0、Go 1.25.5；项目 `web/.node-version` 为 24.10.0、`go.mod` 为 1.25.1。Node 测试进程使用 `NODE_OPTIONS=--no-experimental-webstorage`，避免 Node 26 原生 Web Storage 与 jsdom 冲突。后端设置进程级 `GOARCH=amd64` 和本任务独立 `GOCACHE`，未修改 Git 配置或系统环境。

| 工作目录       | 实际命令/检查                                                                                                                                                 | 结果                                                                        |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `web/`         | `bun run test -- src/features/channels --maxWorkers=1 --testTimeout=15000`                                                                                    | exit 0；12 文件、102 用例通过，282.14 秒，包含最终新增的目标类型切换用例    |
| `web/`         | `bun run typecheck`                                                                                                                                           | exit 0                                                                      |
| `web/`         | `bun x --no-install oxlint -c .oxlintrc.json` 加全部七个变更 TS/TSX 文件                                                                                      | exit 0                                                                      |
| `web/`         | `bun run build`                                                                                                                                               | exit 0，Rsbuild 2.1.6 实际产物生成                                          |
| `web/classic/` | `node --test src/components/table/channels/modals/__tests__/multi-key-editing.test.mjs`                                                                       | exit 0；8 用例通过                                                          |
| `web/classic/` | `node node_modules/vitest/vitest.mjs run src/components/table/channels/modals/__tests__/multi-key-strategy.compat.test.jsx`                                   | exit 0；2 用例通过，147.09 秒；直接运行已有 Vitest 配置，未使用 60 秒包装器 |
| `web/classic/` | `bun run build`                                                                                                                                               | exit 0；Vite 5.4.11 构建通过                                                |
| 导出仓库根     | `go test ./controller ./router -run 'Test(UpdateChannelMultiKeyConversion\|.*Channel.*Permission)' -count=1`（正则中的竖线直接传给 Go，不包含 Markdown 转义） | exit 0；controller 0.888 秒，router 0.675 秒                                |
| `web/`         | `bun run i18n:sync` 与 JSON 实际值检查                                                                                                                        | exit 0；七语 missing/extras 均 0，新增 14 个值齐全                          |
| 验收工作树     | `oxfmt --check` 加实现提交全部 18 个文件                                                                                                                      | exit 0；均已格式化                                                          |
| 验收工作树     | `git diff e91adf6^ e91adf6 --check`                                                                                                                           | 通过                                                                        |
| 验收工作树     | 对补丁新增 Markdown 引用解析相对路径                                                                                                                          | 新增 4 处，缺失 0                                                           |

测试日志保存在本轮任务临时目录 `multica-task-215300746`，分别为 `default-tests.log`、`classic-tests.log`、`default-build.log`、`classic-build.log` 和 `i18n.log`；平台验收评论及本记录保存命令和最终结果，不依赖临时目录长期保留。后端首次编译前按项目要求仅在导出目录创建了双模板嵌入占位文件，随后两套真实构建成功；未把占位文件视为构建通过证据。

## 限制、交接与跨项目影响

- 没有发现阻止 P1 验收的缺陷。Default 测试仍有 jsdom `scrollTo` 提示；Classic 有既有 React/Semi API、Browserslist、lottie eval 和大 chunk 提示；通过不表示零警告或性能基准通过。
- 七语同步报告仍有既有未译项：fr 12、ja 37、ru 37、vi 13、zh 5。新增文案完整，不据此声称全库翻译完成。
- 扩大扫描开发索引时发现历史链接 `../workflows/2026-08/15_topup_payment_fee_included_hint.md` 目标缺失；该链接已存在于实现父提交，非本补丁引入。新增链接检查通过，不宣称全库文档链接有效。
- 没有运行全仓测试、真实浏览器视觉/键盘/响应式验收、真实供应商调用、MySQL/PostgreSQL 实库、生产验收或多实例传播检查；未修改后端或数据结构。当前 pass 限于已授权 P1 及上述自动化、静态审查范围。
- 按工作流交 `qa` 给主脑确认 P1 收尾及是否核准 P2，不额外 assign、@ 或创建重复工作项。P2/P3 未完成；B4 Logo URL、C1 DeepSeek UI、Classic 历史来源仍未决，C2⑤不恢复，不复制或命名新模板。
- 当前验收工作树原有 `AGENTS.md` 修改和 `.multica/` 内容保留；编辑器未保存状态不可观测。仅提交验收记录和开发索引，不包含实现快照或原有用户文件；本轮没有业务代码修改，无需新建代码 PR。
- 已验证共享契约影响：Default 新消费已有 `convert_to_multi_key`、`key_mode`、`multi_key_mode`；没有新增 API、鉴权/密钥语义、模型 ID、计费、用量、分组或部署契约。
- NewAPIForDouDi：存在相同 Default 缺口且后端已有转换与敏感权限支持时值得分享；先核对版本和权限结构。
- NewAPIModifyByGang：多密钥管理有参考价值，但 UI 和权限模型可能不同，应确认缺口及后端能力后移植，不建议直接照搬。未向两个项目创建 Issue、PR 或发送消息。
