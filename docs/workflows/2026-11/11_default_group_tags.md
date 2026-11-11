# Default 分组标签补齐（ALHE-113 P2/B5）

## 授权与版本

主脑已接受 P1 固定 head `e91adf6743fd29a7a907aae9fc47cc9720532744` 的独立验收，并在 ALHE-113 核准 P2/B5。当前分支 `agent/agent/alhe-113` 从该提交继续，不重写历史；原有 `AGENTS.md` 修改与 `.multica/` 文件保留，不纳入功能提交。正式对照仍为 `a946865c9b74046d44db046b0fbb84a363551da5`，起点包含的 12 项无关快照差异不属于交付。

## 实施方案与契约

- Default 实际入口为系统设置的分组定价页（`GroupRatioForm` / `GroupRatioVisualEditor`）和本人令牌抽屉（`ApiKeysMutateDrawer`）。管理模块提供标签名称、说明、排序、多 Logo、匹配词与分组绑定；复用有边框、背景、圆角的 Card 和 Dialog。
- 管理端复用 AdminAuth 的 `GET/POST /api/group/tags`、`PUT/DELETE /api/group/tags/:id`，按当前管理员角色加载和显示操作，保留请求层认证、服务端校验与错误提示。CRUD 完成后刷新标签和用户分组缓存；分组保存后刷新标签，以获取关键词匹配的新绑定及删除清理结果。
- 用户端只读 UserAuth 的 `GET /api/user/self/groups` 顶层 `group_tags`，在现有选项内本地筛选，不请求管理接口。筛选不修改当前选择、数量限制、互斥、auto、继承、分组渠道限定及提交语义；auto 入口保持可达，清除或标签失效回到全部。
- 新增真实编辑测试复现了既有初始化顺序缺陷：`form.reset` 刚写入服务端选组后，同一轮持有旧默认值的清理 effect 会把新选择清空。清理现在等待 `isFormInitialized`，避免旧渲染覆盖初选；保留加载后名称、历史 alias 的当前稳定引用以及跨标签追加的实际载荷断言。此修正仅涉及 Default 抽屉时序，不改变选组或后端授权规则。
- 管理绑定使用稳定分组 ID；令牌提交仍为既有 ID/code。展示采用当前 `name`，仅缺失名称或无法解析的历史记录回退标识。标签不授权、不改变用户可用分组集合；删除标签不删除分组或令牌。
- 本轮不改数据库、后端 API、鉴权或计费。回退前端保留已有标签及绑定，不逆转用户数据。
- Classic 已有管理模块、分组表和令牌标签筛选，只核对共享契约并运行现有回归，不反向添加无关功能。模板复制、命名、大改及 B4/C1 未决项不在本轮范围。

## 验证计划

新增确定性交互测试覆盖管理 CRUD 请求、失败保留草稿与确认、管理员/普通用户边界、显示名与 code 不同、绑定 ID、刷新后的改名/删除、标签筛选/清除/空结果、令牌跨标签已选值及实际提交。复用后端现有标签、分组名称/alias/删除测试和 Classic 现有管理、选组测试。实际执行结果与未执行边界见下文。

## 本地执行证据（2026-11-11）

工具：Bun 1.4.4、Node 26.13.0、Go 1.25.5。仓库 CI Bun 为 1.4.1，`web/.node-version` 为 24.10.0，未宣称工具版本完全一致。双模板均使用各自 `bun install --frozen-lockfile`，未修改依赖或锁文件。Windows 沙箱无法启动进程，命令通过已审批的终端执行；Git 进程设置 `GIT_CONFIG_COUNT=0` 避免继承缺失 KEY 的无效配置，未修改 Git 配置。

Default 测试从 `web/` 执行，并设置 `NODE_OPTIONS=--no-experimental-webstorage`，避开 Node 26 原生 WebStorage 对 jsdom 的影响。

| 检查                       | 实际命令 / 结果                                                                                                                                                                                                                                                                                                                             |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 缺口复现                   | 实现前新增选组测试因缺少标签按钮失败；管理员模块尚不存在时测试导入失败，不把导入失败冒称 CRUD 行为红灯。                                                                                                                                                                                                                                    |
| 受影响综合回归             | `bun run test -- src/features/keys src/features/channels src/features/group-tags src/features/system-settings/models src/features/prompt-center/__tests__/group-selector.test.tsx --maxWorkers=1 --testTimeout=15000`：31 文件、206 用例，首轮 205 通过、1 个编辑令牌初选用例失败。该失败定位并修复了上文初始化顺序缺陷；未把首轮称为全绿。 |
| 修复后令牌及公共标签回归   | `bun run test -- src/features/keys src/features/group-tags --maxWorkers=1 --testTimeout=15000`：10 文件 / 62 用例全部通过，退出 0，128.80 秒。                                                                                                                                                                                              |
| P1 集成及其余 Default 范围 | 综合回归中其余 30 文件通过，包含 P1 渠道 12 文件 / 102 用例、分组设置、标签 CRUD/管理表筛选及提示词分组选择。初始化修复只改令牌抽屉，对应模块另行完整复跑；P1 渠道文件保持 P1 提交内容。                                                                                                                                                    |
| 类型与构建                 | 最后一次初始化修复后重新执行 `bun run typecheck`、`bun run build`，均退出 0；Rsbuild 2.1.6。                                                                                                                                                                                                                                                |
| lint                       | 本轮 20 个 TS/TSX 执行 `web/node_modules/.bin/oxlint.exe -c web/.oxlintrc.json <文件列表>`，无错误；原分组编辑器 3 条 `prefer-number-properties` 警告保留。                                                                                                                                                                                 |
| Classic 标签回归           | 从 `web/classic/` 运行 `node node_modules/vitest/vitest.mjs run src/pages/Setting/Ratio/__tests__/group-tags.compat.test.jsx src/components/common/group-tags/__tests__/selection.compat.test.jsx src/components/common/group-tags/__tests__/layout.compat.test.jsx`，3 文件 / 35 用例通过。                                                |
| Classic P1 回归            | `node --test src/components/table/channels/modals/__tests__/multi-key-editing.test.mjs`：8 通过；`node node_modules/vitest/vitest.mjs run src/components/table/channels/modals/__tests__/multi-key-strategy.compat.test.jsx`：2 通过。                                                                                                      |
| Classic 构建               | 从 `web/classic/` 执行 `bun run build`，Vite 5.4.11，退出 0；Classic 实现未改。                                                                                                                                                                                                                                                             |
| 后端共享契约               | 设置 `GOARCH=amd64` 和任务独立 `GOCACHE`，执行下方命令，model/controller/router 三包通过。复用隔离 SQLite 测试，无后端改动。                                                                                                                                                                                                                |
| 七语                       | 按 i18n 技能通过临时 `add-missing-keys.mjs` 添加 13 键 / 91 个值，再执行 `bun run i18n:sync`；七语 missing/extras 均 0。新增值非空，调用点差异扫描通过。                                                                                                                                                                                    |
| 文档与差异                 | 使用 oxfmt 0.57.0 和仓库脚本相同的版权头保护流程，仅格式化本轮 32 个文件；格式复查、20 个源码版权头、4 个新增相对链接及 `git diff --check` 均通过。临时脚本不交付。                                                                                                                                                                         |

后端回归命令：

```powershell
go test ./model ./controller ./router -run 'Test(GroupTag|GetGroupDisplayName|FormatGroupDisplayNames|SaveGroupConfigChangesDisplayNameOnly|UpdateChannelMultiKeyConversion|.*Channel.*Permission)' -count=1
```

新增测试使用 API 边界 mock 与真实组件、hook、表单转换；后端测试另行验证 AdminAuth 拒绝、用户标签交集裁剪、稳定关联、名称/alias、删除清理、规则匹配和最终返回绑定。二者不等于浏览器到完整后端的 E2E。键盘按钮与 URL 图片失败兜底有确定性组件测试；未做真实浏览器视觉、完整响应式或真实图片网络验收。

保留限制：未跑全仓测试、MySQL/PostgreSQL 实库、真实供应商或生产验收。Default 有 jsdom `scrollTo` 提示；Classic 有既有 React/Semi 弃用与属性警告、陈旧 Browserslist、lottie `eval` 和大 chunk 警告。七语历史未译项仍为 fr 12、ja 37、ru 37、vi 13、zh 5；代码扫描另确认原有 `Configure a custom ratio for "{{userGroup}}" users when using a specific token group.` 在七语均缺键，原基线已存在，本项未顺带补译；不能称全站语言包或全库链接完全通过。

## 隔离、交接与共享影响

P1 仍是独立提交 `e91adf6743fd29a7a907aae9fc47cc9720532744`；P2 在其上另作提交。正式集成只应用两批任务补丁，不能整分支携带起点的 12 项无关快照差异。没有创建 PR、推送或改远端分支，父项主脑负责隔离集成及后续发布评估；本轮无可复用的 CI 结论。未合并、发布、部署、切换主题、复制或命名模板。独立验收审新固定 head，本地自测不替代验收；ALHE-113 进入 `in_review`，P3 不标完成。

共享契约只增加 Default 对现有管理标签 API 与用户分组响应 `group_tags` 的消费，无新增模型、鉴权密钥、用量、计费或部署契约，无数据库迁移。

- `NewAPIForDouDi`：若同样缺少 Default 标签入口且已具备标签 CRUD、稳定分组 ID 和 UserAuth 裁剪出口，值得参考；只有类似页面而缺少后端权限边界时不宜照搬。
- `NewAPIModifyByGang`：分类选组有价值，但先核对其前端模板、用户分组响应和令牌 ID/code 语义；需要配套版本、权限和生命周期回归。仅记录分享建议，未向兄弟仓库外发。
