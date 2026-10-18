# Classic 第一阶段上游兼容

## 基线和范围

- 工作分支 `agent/upstream-classic-contract`，目标 `custom-main`。
- 功能基线 `45ab82100`；固定上游分析点 `9fe0457ee`，rc.37 参考 `385d2dfd1`。
- 对照 Default 已交付的 `e6ccc86e6`，仅修改 Classic 和本记录、开发索引。
- 初始工作区干净，旧 `353352428` 保存在 `backup/classic-contract-before-sync-20260915`。
  rebase 后与 `origin/custom-main` 完全一致，旧三个独有合并提交没有带入无关差异。

## 实现目标与契约

1. 保留现有多 Key 回填、追加/覆盖；策略编辑检查敏感写权限，沿用 `multi_key_mode`。
2. 统一供应商名称推断，修正 360gpt、Wan、Qwen embedding、StepFun TTS 分类；推断只用于图标，不替代响应模型证据。
3. 同日时间范围用 `&&`，跨午夜用 `||`；历史不一致逻辑保留原文，合法排他上界不裁剪。
4. 未知请求规则打开和编辑模式往返不能清空或重复乘；价格在 Tokens 额度显示下仍展示货币。
5. 日志查询使用文本字段及关闭自动填充；保留实际分组 ID 与日志敏感字段边界。
6. 额度详情可由键盘打开，负余额、零值和完整 Tokens 数值正确展示。
7. 创建兑换码后默认不下载，显式选择 TXT 或 Markdown；保留 `max_redeem_count` 与本地批量删除契约。
8. Passkey 能力只检测 `PublicKeyCredential`，允许 USB/NFC 和跨设备认证器。
9. 新文案按 Classic 的中文键、`translation` 对象和实际语言维护。

## 安全与兼容边界

沿用现有 API，不引入插件、供应商新接口或新安全中心。后端、Default 只读。
支付、福利、发票、通知、分组、主题及管理员上游响应模型不改变接口或权限。
后端认证切换（Telegram 410、统一验证流程、proof）必须另行双模板集成，不能将旧接口存在当作兼容证明。
本阶段不部署、不发版、不合并主线、不调用生产 API。

## 验证计划

先用可复现用例保护时间规则、Passkey、价格和导出边界，再验证实际组件交互。
执行 Classic 现有合同测试、修改文件 ESLint/格式检查、完整 lint 和生产 build。
测试进程限制 60 秒；完整 lint 的既有问题与本次新增问题分别记录。
交付前核验文档链接、`git diff --check`、最新基线和 PR 文件范围。

## 已实现与保留证据

以下均对照当前 Classic 实现及固定 Default 业务提交，不代表完整上游迁移。

| 能力           | Classic 结果与代码证据                                                                                                                                                                                    | 接口边界                                                                                                                                         |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| 多 Key         | `EditChannelModal.jsx` 已回填 random/polling 与追加/覆盖；本次用 `useUserPermissions` 的服务器能力矩阵禁用无权限控件，`multiKeyEditing.js` 覆盖表单残留字段；无新密钥不发 `key_mode`                      | `GET /api/user/self` 的 `permissions.admin_permissions.channel.sensitive_write`；`PUT /api/channel/` 的 `multi_key_mode`；权限仍由服务器最终校验 |
| 供应商         | 渠道模型选择和日志 `renderModelTag` 已共用分类；新增 `helpers/modelProvider.js` 修正 360gpt、Wan、Qwen embedding、StepFun TTS，保留 Classic Cloudflare/embedding 等别名及已有中文标签；Wan 图片随构建打包 | 只推断显示图标，不修改 `vendor_id` 或模型 ID                                                                                                     |
| 时间与未知规则 | `requestRuleExpr.js` 同日交集/跨夜并集、合法排他上界与历史运算符保真；`TieredPricingEditor.jsx` 仅用户修改时回写，不在初始化 effect 清空价格，原始未知规则模式往返保留一次                                | 继续存储现有 `ModelBillingExpr`；不引入 fixed、插件变量或后端新编译器                                                                            |
| 价格           | `helpers/utils.jsx` 在 Tokens 额度模式下仍生成货币单价；保留零倍率、K/M 单位、路线规格、按秒价格和充值折扣；工具栏允许充值及币种切换                                                                      | `/api/pricing` 响应及价格字段不变，钱包额度显示方式不变                                                                                          |
| 日志防自动填充 | `UsageLogsFilters.jsx` 已是普通文本输入且 form `autoComplete=off`，不存在 password 字段，无须改实现；新增真实表单测试核对查询保留稳定分组 ID                                                              | 查询参数及现有敏感显示逻辑不变；没有新增敏感数据暴露                                                                                             |
| 额度详情       | `UserQuotaCell.jsx` 保留已用、剩余、总额，负余额不按总额误判为空；按钮支持 Tab/Enter；详情同时显示六位小数货币与完整 Tokens，进度条限制在 0～100                                                          | 只使用 `quota`、`used_quota`，不推断或修改后端钱包类型                                                                                           |
| 兑换码         | `RedemptionExportModal.jsx` 创建后显式确认，默认不下载；TXT/Markdown 可选名称和额度，转义制表符、换行、Markdown 链接及实体，表头使用当前语言                                                              | 原 `POST/PUT /api/redemption/`、`max_redeem_count` 和营销流程保留；不改本地 `DELETE /api/redemption/batch` 的 `deleted_ids/skipped`              |
| Passkey        | `helpers/passkey.js` 只检测 `PublicKeyCredential`，平台认证器缺失或探测异常不排除外置/跨设备认证器                                                                                                        | 原登录、绑定、验证路径不变，未接入新安全协议                                                                                                     |
| i18n           | Classic 八个实际 locale 文件增加六个业务文案键；保留中文键与 `translation` 嵌套结构                                                                                                                       | 新增 48 项翻译，逐键比较确认原有翻译值修改数为 0                                                                                                 |

## 不需修改的本地契约

- 福利：`hooks/benefits/useBenefitsData.jsx` 继续调用 `/api/benefit/activities`、`/api/benefit/vouchers`；福利金额、分组、兑换和删除合同测试通过。
- 发票：`InvoiceBatchRequestModal.jsx` 保留 `/api/user/invoice/config|orders|preview|request|payment`，易支付请求仍由 `paymentRequest.js` 组装；相关发票请求、手续费及合并账单测试通过。
- 支付：`components/topup/index.jsx` 的支付请求及充值金额预览未改；本次只调整模型价格展示，不改变订单金额、币种或手续费。
- 通知：`pages/NotificationCenter/index.jsx` 保留 `/api/notification/bots|tasks|event-types|deliveries`；关键词、状态码、前缀去重合同测试通过。
- 主题：`context/Theme/index.jsx` 仍读写 `theme-mode`；顶栏、移动菜单、VChart 生命周期测试通过，没有移植 Default 的主题框架。
- 分组：提交仍使用 `buildGroupSelectionPayload` 的稳定分组 ID/内部编码；分组显示和复制合同测试通过。
- 上游响应模型：`UsageLogsColumnDefs.jsx` 仍先检查 `isAdminUser` 才显示 `other.upstream_response_model_name`；对应管理员显示合同测试通过。渠道测试及后端普通用户脱敏代码不变，供应商推断不能替代响应模型证据。
- 原生扩展：`/api/extensions` 与 native v1 未改；归档、安全审计及会话刷新后的 SDK API 获取测试通过。

## 验证结果

- 工具实测：Bun `1.3.13`，Vite `5.4.11`；锁定安装成功。新增 Vitest `2.1.9`、Testing Library 与 jsdom 开发依赖用于真实 Semi 组件测试，未升级运行时依赖。
- 先运行时间规则和 Passkey 新回归，7 项中 6 项失败；修复后全部通过。
- `bun test --timeout 60000 <全部 src/**/*.test.mjs>`：63 文件、321 项通过。纯 Node 运行时有一个既有省略扩展名的 JSX 导入加载失败；Bun 支持该既有文件，完整 Bun 结果为准。
- `bun run test:compat`：6 文件、9 项通过；覆盖真实渠道表单权限与 PUT 载荷、未知规则往返、货币/零倍率/路线规格、日志自动填充、额度键盘和导出取消/下载。脚本对整个测试进程设置 60 秒上限，测试清理 Semi 的全局 Toast，避免跨文件污染。
- `bun run test:native`：11 项通过。
- 修改的 26 个 JS/JSX 文件定向 ESLint 通过。
- 完整 ESLint 与 `45ab82100` 归档使用同一 node_modules 比较：5 → 5 条错误，新增 0；均为未改文件的既有版权头格式错误（`paymentRequest.js`、`consoleHeaderBehavior.js`、`table-view-options.js`、`manage-log-presenter.js`、`topupError.js`）。
- `bun run lint`（Prettier）已执行，未通过；同依赖、同配置基线比较为 115 → 115 个文件，失败文件集合完全相同。新增及原格式合规的修改文件已格式化，5 个原有不合规修改文件保留原排版，避免整文件无关重排。`.prettierignore/.eslintignore` 排除生成的 dist 和 node_modules。
- Classic Vite 生产构建通过；保留浏览器数据过旧、第三方 lottie eval、大包体积警告。最终构建耗时 34.32 秒。
- Classic 没有 typecheck 脚本；本次仅 JS/JSX，没有新增 TS/TSX，不将 Default 类型检查冒充 Classic 验证。
- 独立只读审查发现 Markdown 链接/实体转义不足，已修复并增加精确内容回归。
- 尚未验证真实 WebAuthn 硬件、线上 API 和真实浏览器端到端操作；jsdom 交互与生产构建不等于生产业务验收。

### 可复跑命令

在 `web/classic` 下运行，Bun 可使用用户提供的临时缓存绝对路径：

```powershell
bun install --frozen-lockfile
$classicContractTests = rg --files src -g '*.test.mjs'
bun test --timeout 60000 $classicContractTests
bun run test:compat
bun run test:native
bun run lint
bun run eslint
bun run build
```

完整 lint 的基线复核应从 `git archive 45ab82100 web/classic` 解出源码，并使用同一依赖目录执行相同 Prettier/ESLint；不要全局安装或修改其他工作区。构建产物、测试临时缓存均被 Git 忽略，不进入 PR。

## 后续单独集成

插件管理、供应商新接口、新安全中心、Telegram OAuth/410、统一登录验证、Passkey 多域、审计接口、任务附件及 Responses WebSocket 依赖新后端，未在本 PR 暴露入口。日志 request_rules 逐条新追踪显示也未新增，本阶段不改变既有日志请求倍率字段。后续必须一起评估后端与双模板，不能拿当前前端测试宣称完整上游集成完成。

跨项目影响：没有新增模型 ID、认证协议、usage/计费交换字段或生产配置；多 Key 继续使用既有管理 API 字段。未向兄弟仓库发送通知。

## 提交前核验

- 2026-09-15 再次 `git fetch origin --no-tags`，`origin/custom-main` 仍为 `45ab82100a74b9e7c3866f1c173d16d32745527e`，提交前 behind=0。
- 原分支没有已有 PR；本阶段独立提交，不依赖 Default PR #212 的实现文件。
- Git 身份为 `meoacgx <a1478882500@gmail.com>`，最近 100 次提交中有 55 次来自同一作者；不修改 Git 配置。PR 仍透明说明 AI 辅助实现与验证，人工确认栏留给维护者。
- `git diff --check` 通过；本记录索引链接可解析，八个 locale JSON 可解析，原有翻译修改数为 0。
- 文件范围仅 `web/classic/`、本记录与开发索引；没有 Default、后端、relaykit、版本或部署改动。
