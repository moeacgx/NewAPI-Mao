# 网站 SEO 设置与 Classic 日志筛选布局

## 目标与范围

- `SystemName` 驱动浏览器标签页标题，新增可清空的 `SystemDescription`（默认空、最多 200 个 Unicode 码点）供 SEO 与分享描述使用。
- Default 与 Classic 首屏 HTML 都在脚本执行前具有动态标题及唯一 description/Open Graph 元数据；`/`、`/index.html` 和 SPA HTML 路由一致，保存后下一次请求生效。
- Classic 使用日志筛选将渠道提前，并让用户名/用户 ID 模式与输入成组相邻，不进行其他表单重构。
- 提交 PR 供审查，本轮不部署发版、不改线上配置；最终 Git 提交、推送和 PR 由主协调者负责，不改旧工作区或兄弟项目。

## 方案与接口契约

- 新配置复用现有 Option 表、校验、Root 权限 `PUT /api/option/` 和多节点配置同步；公开 `GET /api/status` 增加 `system_description`，不增加匿名写接口。
- 描述按 Unicode 码点限制为 200，可传空字符串清除。旧库无此项时稳定读为空字符串。
- Go 在 Web Router 设置阶段预处理两主题静态 HTML，只在应用 HTML 请求时插入基于 `OptionMap` 读锁快照动态转义的名称/描述；不改资源、静态品牌归属、generator、SVG title、版权和统计脚本，响应设置 `no-cache`。
- Classic 与 Default 的运行时状态刷新均以最新 `/api/status` 更新 title/description 元数据；Classic 输入提示与模式同步，Default/Classic 表单保存同一个 Option。
- SEO 不新增 canonical、robots、keywords 或 sitemap。HTML 文本/属性值必须正确编码 `&<>\"'`，不能形成标签或脚本注入。

## 回归与验证计划

- Go：初始 HTML、更新和清空、恶意字符、唯一元数据、两模板 `/`、`/index.html`、`/console/log`、静态资源保留；Option Unicode 长度和 `/api/status` 字段契约。
- Default/Classic：真实 DOM 状态到达后更新名称/描述并可清空；站点设置保存/表单字段；Classic 用户筛选控件成组、placeholder 切换、reset 默认值及管理员可见性。
- 执行受影响 Go 测试（不超过 60 秒）、两模板受影响测试与构建、Default typecheck、相关文件 lint/格式和 `git diff --check`。

## 实际修改与验证

- 后端：新增 `SystemDescription` 默认值与 Unicode 长度校验，公开状态添加只读描述字段；两模板应用 HTML 使用读锁快照、转义后输出名称与描述，清空时移除描述标签。修复审查发现的注释占位符残留及根路由重复响应问题。
- Default 与 Classic：设置表单、状态同步与保存后 DOM 更新均已接入；缓存旧站名不会覆盖首屏标题。Classic 用户模式与输入框作为最后一个筛选组，保留管理员可见性。
- 七语翻译通过 `add-missing-keys.mjs` 写入；Default 新增 3 个键，Classic 新增 8 个键（含实际运行的 `zh-CN` 及历史 `zh` 副本）。两模板 `i18n:sync` 通过，既有翻译值全部保留，临时脚本已移除。
- `go test ./router ./controller ./model -count=1 -timeout 60s` 通过；增强的 router 用例解析实际 head 节点，验证描述可见、唯一、正确转义，覆盖两主题的 `/`、`/index.html`、`/console/log`。后端只读对抗复核通过。
- Default 两个定向测试文件共 6 项通过：真实路由、React Query 与表单，网络边界模拟成功、清空、业务失败和超长描述；`typecheck` 通过。
- Classic 两个定向 Vitest 文件共 4 项通过：元数据更新/清空、管理员用户筛选分组、初始模式提示、重置和普通用户隐藏行为。实际下拉切换另由浏览器交互验证。
- 本地 Chrome 使用真实 Classic 页面和模拟 API：1440 像素桌面并排、390 像素窄屏展开后上下相邻且无溢出，用户 ID 切换后输入 `42` 成功；设置页保存名称、保存/清空描述的请求字段与 DOM 更新均正确，无页面脚本错误。此验证不代表线上部署或真实数据库持久化验收。
- 包含全部翻译的 Default 与 Classic 生产构建通过（约 25 秒、76 秒）。Classic 仍有既有的大分块、Browserslist 数据和第三方 eval 提示。
- Classic 原有用户查询 Hook 回归 4 项通过，参数编码、400 恢复与统计失败时单次提示的行为保持。
- Default 相关文件 Oxlint、Classic 相关文件 ESLint/Prettier、翻译与 Markdown 格式检查及 `git diff --check` 通过；`types.ts` 沿用现有分号风格，避免整文件无关格式重排。新增文档链接已核对。
- 共享契约影响：`/api/status` 增加可选展示字段，既有请求和鉴权不变；不影响计费、分发或兄弟项目集成。

## 保存失败回归修复（审查追加）

- Classic：`updateOption` 在业务 `success=false` 时只提示错误但正常返回，名称/描述提交链仍无条件显示成功。限定让该方法返回成功布尔值，并仅让 `SystemName`、`SystemDescription` 的提交处理成功信号；其他设置提交行为不扩改。
- Default：`SystemInfoSection` 虽已阻止失败响应更新元数据，但仍正常返回，`useSettingsForm` 随后更新 baseline 并清除 dirty，相同输入再次保存无法重试。已让表单回调可显式返回 `false` 保留草稿与 dirty；原有 `void` 成功语义不变，不抛出业务失败异常、不重复提示。
- 成功状态同步追加：名称保存成功时，只有 DOM 元数据更新而共享状态未更新，导致页头仍显示旧名称。现已直接修补 Default 扁平 `['status']` 查询缓存、本地缓存和名称配置 store；Classic 同步 `StatusContext` 与 `setStatusData`，并使用当前状态引用保留请求期间的其他状态更新。保留其他字段，不额外发起可能读到旧节点配置的状态 GET；失败响应不更改这些状态。
- 按先失败后修复验证：Classic 三个场景（名称、描述、清空描述）先实际复现错误成功 toast；Default 三个场景先实际复现 dirty 被清除。追加真实 `SystemBrand`/`useStatus`、Classic `useHeaderBar` 状态消费者后，再次实际复现成功保存后名称/描述仍旧的缺陷，修复后同一组用例通过。测试没有 mock 被测表单、状态 hook 或更新 hook，仅在 API 边界模拟响应，并观察真实 toast 调用。

### 本轮最终实测（2026-09-23）

- Default：在 `web/` 运行 `npm exec --yes --package=bun -- bun x vitest run src/features/system-settings/general/__tests__/system-info-section.test.tsx src/lib/site-metadata.test.ts --testTimeout 60000`，2 文件 **8/8 通过**，总耗时 7.02 秒；覆盖成功、清空、三个业务拒绝后原值重试、Unicode 长度与元数据行为。
- Classic：在 `web/classic/` 运行 `npm exec --yes --package=bun -- bun x vitest run src/components/settings/__tests__/site-metadata-save.compat.test.jsx src/helpers/__tests__/site-metadata.compat.test.jsx src/components/table/usage-logs/__tests__/admin-user-filter-layout.compat.test.jsx --testTimeout 60000`，3 文件 **7/7 通过**，总耗时 16.73 秒。
- Classic 查询 Hook：`node --test --test-timeout=60000 src/components/table/usage-logs/__tests__/user-filter.test.mjs`，**4/4 通过**，总耗时约 0.38 秒。
- 两套新增失败回归均断言：只提示一次错误、不提示成功、DOM/页头/本地状态不变、输入保留、原值再次点击保存确实重新发出正确请求；成功后名称/描述状态立即更新，描述清空后两个 meta 节点不存在，其他状态字段保留，且不额外请求 `/api/status`。
- Default 最新 `bun run typecheck` 退出码 0；本轮四个 TS/TSX 文件 Oxlint/oxfmt 及 Classic 两个 JSX 文件 ESLint/Prettier 检查通过。没有新增翻译键或更改 locale、依赖、锁文件、后端。
- Classic 测试有既有 Semi/React 弃用、React Router future flag、Browserslist 数据过旧提示，以及预期拒绝场景的错误日志；没有失败测试或未处理 Promise。未以这些模拟测试冒称线上验收。
- 主协调者最终生产构建回报：包含品牌同步的最新 Default 构建 **8.29 秒通过**、Classic 构建 **59.81 秒通过**。此项证据来自主协调者，本 writer 未重复构建；此前已启动的 Classic 重复构建已停止。最终提交、推送和 PR 由主协调者负责；本 writer 未执行 Git 写入、发版或部署。
