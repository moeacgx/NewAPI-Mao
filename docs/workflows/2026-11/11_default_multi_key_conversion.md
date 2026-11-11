# Default 单密钥转多密钥渠道

## 目标与范围

补齐历史模板审计确定的 B2 路径③：在 Default 渠道编辑中，允许有敏感写权限的管理员显式把符合条件的单密钥渠道转换为多密钥渠道。本项不改变 Classic 页面、不新增接口或数据字段，也不涉及主题切换、部署或 P2 标签筛选。

审计依据为 ALHE-114 已验收计划 `881ab6f0e977582a13cd8ee7bf4e2e4999fb16d5`，正式对照基线为 `a946865c9b74046d44db046b0fbb84a363551da5`。本任务起点 `95efd0ae40f57eb7d6df77c65ee827e1946896af` 与该基线的共同祖先为 `6da5e7fee3e29391e04fd95a1d7a95c08ae59726`；本次只在现有 Default 上增补经核准的转换能力。

## 行为与权限契约

- 入口只在编辑单密钥渠道、原始类型有效且来源与当前目标类型均不为 Vertex AI（41）或 Codex（57）、且渠道不是 io.net 管理渠道时显示。
- 转换必须由用户显式勾选。勾选后可选新密钥、追加或覆盖策略及随机或轮询策略；留空密钥不发送 `key` 和 `key_mode`，已有多密钥渠道仍可只更新调度策略。
- 更新请求继续使用 `PUT /api/channel/`。仅显式转换时增加 `convert_to_multi_key: true`；多密钥更新字段仅在 `ChannelSensitiveWrite` 权限下发送。常规更新仍受既有 `ChannelWrite` 路由权限保护，服务端继续校验转换资格。
- 转换取消时表单字段不会写入请求；关闭并重开或切换编辑渠道时，表单以新渠道数据重置，转换标记默认关闭。取回渠道密钥的 RootAuth 仅用于读取密钥，不替代更新权限。
- Classic 已有对应资格、载荷和权限处理，本项保持不变；Default 与 Classic 仍共享原有 API 契约。

## 验证计划与当前证据

应覆盖资格边界、显式转换载荷、追加/覆盖、留空密钥只改策略、取消残留、普通单密钥更新、新建多密钥，以及 Classic 请求契约和敏感权限回归。本次新增 Default 资格/载荷确定性测试和渠道编辑请求载荷测试；Classic 既有 `multi-key-editing` 与 `multi-key-strategy.compat` 测试覆盖其稳定契约。

本地使用 Go 1.25.5（初始工具链为 windows/386）、Bun 1.4.4、Node 26.13.0；项目 `.node-version` 为 24.10.0。两套模板均使用各自锁文件执行 `bun install --frozen-lockfile`，没有修改依赖或锁文件。Node 26 的原生 Web Storage 与 jsdom 冲突，测试进程设置 `NODE_OPTIONS=--no-experimental-webstorage` 后运行；没有修改生产代码或全局环境。

| 验证             | 命令或方法                                                                                                                  | 本地结果                                                                                                                      |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| 原始缺口复现     | 对起点的 drawer、hook、schema 运行新增 append/replace 转换交互用例，结束后恢复实现                                          | 两个用例因找不到转换复选框失败，确认保护实际缺口；这是在依赖安装完成后进行的基线复现，不追溯声明为实现前已运行                |
| Default 渠道回归 | `bun run test -- src/features/channels --maxWorkers=1 --testTimeout=15000`                                                  | 12 个文件、101 个用例通过；包含合法转换、来源/目标限制、io.net、普通管理员权限、取消、重开、切换渠道、空 key 策略和新建多 key |
| Default 类型     | `bun run typecheck`                                                                                                         | 通过                                                                                                                          |
| Default lint     | `bun x --no-install oxlint -c .oxlintrc.json` 后接全部七个改动 TS/TSX 文件                                                  | 通过，无 error                                                                                                                |
| Classic 纯契约   | `node --test src/components/table/channels/modals/__tests__/multi-key-editing.test.mjs`                                     | 8 个用例通过                                                                                                                  |
| Classic 权限组件 | `node node_modules/vitest/vitest.mjs run src/components/table/channels/modals/__tests__/multi-key-strategy.compat.test.jsx` | 2 个用例通过；项目 `bun run test:compat` 包装器在 60 秒固定进程时限终止，直接运行相同 Vitest 用例完成耗时约 100 秒            |

最终补充结果：

- Default 组件文件新增“目标类型改为 Codex 后清理转换，切回普通类型也不恢复”的回归，首次使用错误的 placeholder，按实际搜索输入修正后执行 `bun run test -- src/features/channels/components/__tests__/multi-key-strategy.test.tsx --maxWorkers=1 --testTimeout=15000`，17 个组件用例全部通过。这是在上表 101 个用例之后追加的一个用例。
- 分别在 `web/` 和 `web/classic/` 执行 `bun run build`，Default Rsbuild 与 Classic Vite 均通过。Classic 保留旧 Browserslist、lottie 的 eval 与大 chunk 警告。
- 设置 `GOARCH=amd64` 和本任务临时 GOCACHE 后，执行 `go test ./controller ./router -run 'Test(UpdateChannelMultiKeyConversion|.*Channel.*Permission)' -count=1`，两个包均通过。转换测试使用隔离 SQLite，未修改生产或持久配置；没有修改后端实现或数据库契约，未跑 MySQL/PostgreSQL 实库。
- `bun run i18n:sync` 已执行，七语 `missingCount`/`extrasCount` 均为 0，两条新增文案均有实际译文。同步报告仍有既有未译项（fr 12、ja 37、ru 37、vi 13、zh 5），本项不扩为全库翻译。
- 改动 Markdown 与 TS/TSX 使用仓库 `oxfmt` 格式化。相对链接按新增引用检查，空白检查按本项实际提交范围执行。

最后一次类型检查发现补充测试给 `getByRole` 传入不支持的 `exact` 选项；移除后重新执行 `bun run typecheck`、该测试文件 lint 和完整 17 个多密钥组件用例，均通过。格式检查覆盖全部 18 个交付文件；新增相对链接 4 处、缺失 0，七语新增译文 14 个值齐全。本轮只执行受影响渠道模块与所列后端/Classic 回归，没有宣称整个仓库所有测试通过。

首次测试因依赖未安装而未启动；第二次组件测试被 Node Web Storage 阻塞；一次既有组件用例在本机默认 5 秒时限超时，改为上述 15 秒测试时限后通过。hook 测试的必填 `key` fixture 已修正。最终通过的 Default 输出含 jsdom `scrollTo` 未实现提示；Classic 输出含既有 Semi/React 属性与废弃 API 警告、旧 Browserslist 提示，不据此宣称日志零警告。基线复现与单独组件运行中也观察到既有 `<p>` 内嵌 `<div>` 警告，未扩为通用布局修复。未做真实浏览器视觉验收、真实供应商请求或生产验收。

## 状态边界

本记录只说明 P1/B2 的实现范围。P1 完成后交独立验收；P2/B5 尚未开始，须等待 P1 验收通过及主脑核准。Default 代码默认主题不代表线上主题已切换；本项未访问线上环境，也未部署、发版或改变生产配置。

本轮起点已有 `AGENTS.md` 修改和 `.multica/` 未跟踪内容，均保留且不提交；编辑器未保存状态不可观测。只交付本项提交，不把起点用户快照或整个任务分支并入正式基线，父分支集成由主脑负责。

起点相对正式基线的 12 项继承差异包含部署脚本与历史文档，不是 P1 补丁。为遵守本任务的快照隔离要求，本轮提供仅含实现提交的补丁，由主脑集成到父分支；不直接对 `custom-main` 创建含整段任务快照的 PR，也不推送这些继承内容。当前 Git 身份与历史核心作者一致，未修改 Git 配置；本项实现由 AI 辅助完成。

共享契约影响：Default 新增对既有 `convert_to_multi_key`、`key_mode`、`multi_key_mode` 的消费，没有新增 API、模型、密钥语义、计费或分组契约。NewAPIForDouDi 可在确认存在相同 Default 缺口及后端转换、敏感权限能力后参考本补丁；NewAPIModifyByGang 亦有多密钥管理价值，但需核对其编辑组件和权限数据结构，不能直接照搬。没有向兄弟项目创建 Issue、PR 或发送消息。
