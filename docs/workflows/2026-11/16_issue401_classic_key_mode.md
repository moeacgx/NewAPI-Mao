# Issue #401：Classic 多密钥策略立即提交

## 目标与范围

核实迁移矩阵 B3 的 Classic 单密钥转多密钥现象：点击 Polling 后立即提交，
请求仍携带 `multi_key_mode=random`。只处理本仓库当前 Classic 渠道编辑组件，
核对 Default 相同行为；不读取上游、不连接生产、不读取真实密钥。

## 根因与修复前证据

- 复用 B3 的 mock API、完整分组夹具和本地 Classic 产物，去除固定等待。
  三次实际点击 `.semi-select-option-text` 的 Polling，38–39ms 后点击 Submit，
  三次 PUT 均携带 `multi_key_mode=random`；没有页面异常或未处理 API。
- D 盘隔离的 Vitest 夹具直接导入当前 `EditChannelModal.jsx`，
  点击真实 Semi 下拉选项后立即提交，稳定得到 `expected random to be polling`。
  因此不属于旧产物或只点到错误文本的问题。
- 当前安装的 Semi Select 在受控模式下，`onSelect` 立即触发，
  `onChange` 则放在 `popoverClose` 的回调中，等待退出动画结束。
  业务组件只在 `onChange` 更新 `multiKeyMode`，提交又从该状态构造请求，
  因而在动画窗口内使用旧值；`key_mode` 的追加/覆盖选择也沿用同样写法。

证据目录：`D:/Temp/newapi-issue401-key-mode/`。修复前浏览器记录为
`browser-original.json`，当前源码红测为 `component-red.log`，
复现脚本为 `reproduce.py`、`selection.compat.test.jsx` 和 `vitest.config.mjs`。

## 方案与契约

- 将密钥策略及更新模式的业务状态同步移到 Semi `onSelect`；
  同时通过既有 `handleInputChange` 同步表单值，不等待下拉退出动画。
- 保留 Form.Select 自身处理 `onChange` 的机制，不修改 Semi 依赖或全局动画。
- `PUT /api/channel/` 的 `convert_to_multi_key`、`key_mode`、
  `multi_key_mode` 字段和枚举不变，不新增后端接口。
- 空密钥继续不发送 key，不改变既有密钥；转换继续显式授权，
  敏感字段仍受 `ChannelSensitiveWrite` 门禁控制。
- 保留 Codex、Vertex、io.net 转换限制；关闭/重开重置来源与模式；
  不新增多密钥转单密钥，不调整后端权限。

## 测试计划与兼容性

- 当前真实组件回归：单密钥转换立即选择 Polling 后提交；已有多密钥切换策略；
  追加/覆盖立即提交、空密钥保留、取消重开及敏感写权限。
- 对照 Default 的选择器和提交链路，执行定向组件回归，不制造产品改动。
- 浏览器复核使用明确选项定位及请求断言，不添加固定等待；
  mock 仅验证前端事件和请求载荷，不证明真实数据库、缓存轮询与生产权限。
- 每个测试子进程限时 60 秒；TEMP/TMP 使用 D 盘，
  Node 26 使用进程级 `--no-experimental-webstorage`。
- 不重复全量构建；源码修改交协调者安排独立审查与必要重建。
  其他共享文档由协调者集成。

## 实现与验证结果

仅修改以下三个文件：

- `web/classic/src/components/table/channels/modals/EditChannelModal.jsx`：
  两个选择器使用 `onSelect(value, option)` 的第一个参数，同步模式状态及表单；
  不再依赖动画结束后的自定义 `onChange`。当前 Semi 的键盘 Enter 路径同样调用
  `onSelect`，已由真实组件键盘事件验证。
- `web/classic/src/components/table/channels/modals/__tests__/multi-key-immediate-submit.compat.test.jsx`：
  新增 8 项真实组件回归，覆盖鼠标和键盘双向切换随机/轮询、双向切换追加/覆盖、
  空密钥保留、取消重开、无敏感写权限。
- 本记录。

### 定向测试

以下命令在 `web/classic/` 执行，进程级环境为
`TEMP=TMP=D:/Temp/newapi-issue401-key-mode`，
`NODE_OPTIONS=--no-experimental-webstorage`：

```text
node scripts/run-compat-tests.mjs --config vitest.config.mjs src/components/table/channels/modals/__tests__/multi-key-immediate-submit.compat.test.jsx src/components/table/channels/modals/__tests__/multi-key-strategy.compat.test.jsx
node --test src/components/table/channels/modals/__tests__/multi-key-editing.test.mjs
node node_modules/eslint/bin/eslint.js src/components/table/channels/modals/EditChannelModal.jsx src/components/table/channels/modals/__tests__/multi-key-immediate-submit.compat.test.jsx
```

- 修复前 D 盘隔离组件红测：1 项失败，准确得到 `random` 而非 `polling`；
  整次运行 40.20 秒，日志 `component-red.log`。
- 修复后正式组件测试：2 文件 10/10 通过，11.61 秒，
  日志 `component-green-final.log`。断言前没有等待策略文本或动画结束，
  直接触发选项与真实表单提交；键盘按当前活动选项导航后立即 Enter 并提交。
- 原有纯契约测试 8/8 通过，涵盖转换资格、取消转换、空密钥和权限，
  日志 `native.log`；ESLint 通过，日志 `lint.log`。
- 组件测试仍有项目既有 Semi/React 18 警告，不影响断言结果。
- 新增测试通过项目 Prettier 检查，本记录通过 oxfmt 格式化与检查，
  受影响文件 `git diff --check` 通过；组件保留既有格式，仅改两个事件处理器，
  不执行整份大型组件的无关重排。集成阶段新增的总交付记录相对链接已校验。

### 浏览器证据

`source-browser.py` 加载本次修复后的 Vite 源码（`5197/src/index.jsx`），
而非旧的 `dist/assets/index-GrRcKNNE.js`。API 全部拦截为本地夹具，
对外请求拒绝，不读取真实密钥；每个脚本进程由外部 60 秒超时保护。

- `browser-source-mouse.json`：桌面鼠标选择 Polling 后立即提交。
- `browser-source-keyboard-mobile.json`：移动视口通过方向键和 Enter 选择 Polling，
  随即提交。
- `browser-source-replace.json`：桌面选择覆盖后立即提交，载荷为 `replace`。
- `browser-source-append.json`：先选择覆盖，再选择追加后立即提交，
  载荷为 `append`；两次展开之间断言前一个菜单已隐藏，
  最后一次选择与提交之间没有等待。

每组重复三次，共 12 次通过；请求均保留 `group_ids=[11]` 和显式转换字段，
策略为 `polling`，没有页面异常或未处理 API，相关 `source-*.png` 保存在同一证据目录。

夹具调试失败也保留：`browser-source-mouse-cold.json` 记录首次 Vite 冷编译导致的
导航超时；`browser-source-replace-selector-error.json` 记录英文单复数定位错误；
`browser-source-append-reopen-error.json` 记录首次菜单尚未关闭时重开的错误时序。
这些失败没有产品保存载荷，不能拿它们解释修复前 3/3 的真实错误请求。
修正为实际 DOM 文案及确定性菜单状态后重跑，未添加固定等待。

### Default 对照

Default 的 `ChannelMutateDrawer` 使用 Base UI Select 的
`onValueChange={field.onChange}` 直接更新 React Hook Form；提交 hook 使用当前表单值，
没有 Classic 这条 Semi 退出动画回调链。未修改 Default。

在 `web/` 执行：

```text
node node_modules/vitest/vitest.mjs run src/features/channels/components/__tests__/multi-key-strategy.test.tsx src/features/channels/hooks/__tests__/multi-key-conversion.test.tsx --maxWorkers=1
```

2 文件 27/27 通过，22.78 秒，日志 `default-tests.log`；
覆盖真实选择/提交以及转换、资格、权限和关闭重置契约。

## 交付边界与集成事项

- 修复的是用户完成选择后、下拉退出动画期间提交的真实窗口；
  不依赖延迟保存或固定等待，不扩大后端权限，不改变 API 请求字段。
- 真实后端、数据库、Redis 轮询执行、线上角色权限仍未验收；
  模拟前端通过不能代替这些验证。
- 实现工人阶段未运行全量构建或修改 Classic dist；后续集成已完成受影响 Classic 重建并更新共享索引、能力记录和迁移矩阵，结果见下节。独立审查由 `ctx_108ffd07f56c` 完成并通过，未发现 P1/P2。
- 没有新增用户文案，不需翻译更新；对外 API、推理、usage、计费契约没有变化。
- `NewAPIForDouDi` 与 `NewAPIModifyByGang`：若其 Classic 同样使用受控 Semi Select
  并在动画后的 `onChange` 中同步提交状态，则值得分享此修复与回归方法；
  未核实它们的实现，不建议直接复制补丁，本轮不向兄弟仓库发送消息。

## 追加集成门禁（2026-11-16）

- 在 `web/classic/` 执行 `bun run build --outDir D:/Temp/newapi-issue401-final/classic-dist-after-key-mode`，exit 0，进程耗时 33.51 秒，Vite 报告 32.39 秒；仅保留 chunk 体积警告。
- 本地执行时间 05:44:44 至 05:45:18；独立 `result.json` 记录 UTC 起止时间、完整命令、工作目录及退出码，和 `classic-build.log` 一起保存在 `D:/Temp/newapi-issue401-final/key-mode-build-ctxd41efb359a12/`。TEMP/TMP 只对构建子进程指向该 D 盘目录。
- 旧 `results.json`、旧独立产物和仓库 dist 保留；Default 源码未再变化，沿用先前 build/typecheck 结果，不重复执行。
- README、能力总览、迁移矩阵和[总交付记录](16_issue401_delivery.md)已登记此追加修复；原六子项独立审查不代替本项独审，`ctx_108ffd07f56c` 已正式回执通过。
- 构建通过不扩大前述 mock/组件覆盖；真实数据库、Redis 轮询和线上角色权限仍未验收，未提交、推送、部署或同步上游。

## 最终独立审查（2026-11-16）

- 非实现者 `ctx_108ffd07f56c` 独立复核两个处理器、Semi 鼠标/键盘及 Form 延迟回调链，确认功能通过，未发现 P1/P2。
- 独立复跑正式真实组件 10/10、原生契约 8/8；额外组件边界 5 项通过，另 1 项因 JSDOM 不完成 CSS 退出动画而未通过，不计入通过数。对应动画结束后再次选择与提交的边界，使用新构建产物在真实 Chromium 中补验 3/3 通过。
- 独立浏览器同时验证键盘立即覆盖、覆盖后切回追加并立即提交，最终请求字段正确；全部 API 与身份仍为本地 mock。
- 原始命令、日志、失败记录及浏览器产物见 `D:/Temp/issue401-key-mode-review-242d/report.md`。独立审查没有重跑构建，没有修改仓库。
- 审查发现的 P3 文档不一致已修正：本记录包含总交付记录链接，已删除没有新增链接的过时表述。
- 本项实现、独立审查及受影响 Classic 重建均已完成；真实后端、数据库、Redis 轮询与生产权限的未验收边界保持不变。
