# MAO-6：分组标签与多 Logo 分类

## 范围与结果

用户截图中的分组表与 Classic 的 `GroupRatioSettings` 对应，因此本次页面范围为 Classic。
Default 使用独立入口，不在本次截图任务范围内。公开模型广场的模型标签筛选不与分组标签混用。

- 后台「分组与模型定价设置」增加「分组标签」页签，可维护名称、说明、排序和分组绑定。
- 一个标签支持 0–6 个按顺序组合的内置图标或 HTTP/HTTPS 图片 URL，可先建空标签再绑定分组。
- 一个标签可绑定多个分组，一个分组可属于多个标签；令牌创建/编辑和后台分组表提供分类过滤。
- 保留所有分组、未分类、auto、独立分组、多选顺序与倍率保护。跨标签切换不会清空已经选择的分组。
- 管理表只过滤展示行，仍保存完整草稿；新建分组会切回全部，避免新行落入隐藏分类。
- 七种当前语言和兼容的 `zh.json` 补齐新文案，未改写原有翻译值。

长期数据与接口契约见[分组标签](../../developer/group-tags.md)。

## 实现与兼容边界

- `group_tags` / `group_tag_bindings` 使用 TEXT JSON 和稳定 Group ID；串行与并行迁移均登记。
- 管理 CRUD 位于 `/api/group/tags`，继承 AdminAuth；用户分组响应只增加顶层 `group_tags`，
  保留 `data` 字典、原 ID/code 和 `group_mode`。标签内容与关联都裁剪到当前可用分组。
- 标签和绑定同事务保存，复用 `lockForUpdate`，按 Group ID 顺序加锁。
  分组删除在同一事务清理标签绑定；旧分组保存接口不接管标签。
- 查询使用 GORM 子查询引用分组表，避免 MySQL 8 对裸 `groups` 保留字的语法问题。
- 不下载外部图片，不接收脚本、内联 SVG/HTML 或含凭据 URL；复用供应商图片失败兜底与无 Referer 加载。
- 目录刷新使用独立请求并忽略旧响应；订阅分组更新事件以展示最新名称和删除结果。
  保存中关闭、删除中新增等操作不会误关另一份草稿。
- 手机候选弹出列表使用视口约束宽度，避免继承的固定宽度产生横向溢出。

## 本地验证

验证日期：2026-09-27。初始工作区干净，`origin/custom-main...HEAD` 为 `0 0`。
未创建提交或 PR，未部署、未访问生产、未写任务看板。

后端以进程级 `GOARCH=amd64`、`CGO_ENABLED=0` 执行：

```powershell
go test ./model ./controller ./router -run 'Group|Token.*(Auto|Exclusive|Migration)' -timeout 60s -count=1
```

三个包通过。新测试覆盖多 Logo 与 ID 持久化、别名/改名、旧客户端保存保留关联、非法更新保留旧数据、
清空与删除、用户投影权限，以及经过真实 API 路由的匿名/普通用户拒绝和管理员 CRUD。

Classic 在 `web/classic` 执行：

```powershell
node scripts/run-compat-tests.mjs src/components/common/group-tags/__tests__/selection.compat.test.jsx src/pages/Setting/Ratio/__tests__/group-tags.compat.test.jsx src/helpers/__tests__/vendor-logo.compat.test.jsx
node --test src/helpers/groupDetails.test.mjs src/helpers/groupDetails.exclusive.test.js
bun run i18n:sync
bun run build
```

- 组件与供应商 Logo 回归：26 项通过，外层 60 秒超时；首次冷启动与并行构建竞争导致一次超时，之后独立重跑通过。
- 既有分组 helper 回归：24 项通过。
- 完整令牌创建/编辑组件读取真实形状的用户分组响应，在跨标签选组后向 API 提交原 code、ID 顺序和 explicit 模式。
- 已验证键盘 radio、空分类禁用、旧请求不覆盖新目录、改名实时刷新、保存失败保留编辑内容、保存期间 Esc 不退出。
- 修改过的 JS/JSX 通过 ESLint；JS/JSX/CSS/Markdown 通过 Prettier；`git diff --check` 通过。
- Classic 没有 typecheck 脚本，本次未改 TS/TSX，不以 Default 类型检查代替。
- 本机没有 PATH 中的 Bun，使用 `npm exec --package bun -- bun` 调用已缓存的 Bun；
  依赖按 frozen lockfile 安装，锁文件和依赖版本声明未修改。Vite 最终构建通过。
- 构建仍有既有依赖 `lottie-web` eval、Browserslist 数据过期和大包体积警告，没有因此更新无关依赖。

## 浏览器验证

Playwright 使用本地 Vite 与实际组件，API 为独立 fixture，不代表线上端到端验证。

- 桌面 1280 宽度检查组合 Logo、选中态、说明、管理卡片。
- 手机 390×844 检查两列分类、分组候选弹出列表及深色标签编辑。
- 打开分组候选列表和编辑器时，实测 `documentElement.scrollWidth === innerWidth === 390`。
- 检查从 OpenAI 切换至国产标签后，已选 Codex 分组仍保留。
- 本地截图保存在 `.local-tests/output/playwright/`，不进入业务源码。
- 控制台存在 Semi UI 的 React 弃用/更新提示；未作为本次功能故障扩大重构。

## 2026-09-30 桌面对齐与完整令牌预览补验

- 用户指出原演示把窄选组组件和宽管理页上下拼接，边界不一致，且未展示完整创建令牌流程。
- 已将前台令牌抽屉与后台标签管理拆为独立本地入口，直接加载实际 Classic 组件；模拟 API 仅在演示入口生效。
  正式管理入口仍为「分组与模型定价设置 → 分组标签」，不挂载到前台令牌页面。
- 默认图标与供应商 Logo 使用相同尺寸；标签名预留两行，超过三 Logo 时所有卡片统一预留两行图标槽；
  桌面三列、手机两列；管理卡片标题槽与底部操作区对齐。
- Playwright 已验证 1440/390 宽度，长名称与六 Logo 的名称和计数纵向偏移一致，无横向溢出；
  不等长后台内容的操作区底线一致。完整令牌抽屉跨标签选择后仍提交 `group_ids: [11,31]` 和原 code 顺序。
- 本地地址：`http://127.0.0.1:5179/.local-tests/token-create.html` 为前台，
  `http://127.0.0.1:5179/.local-tests/group-tags.html` 为后台；加 `?stress=1` 可检查长名称和六 Logo。
  模拟保存仅存当前页面内存；刷新恢复演示数据，不访问生产。Vite 绑定 `127.0.0.1`。
- 截图分别为 `.local-tests/output/playwright/mao6-token-create.png`、`mao6-token-mobile.png`、
  `mao6-admin-group-tags.png`；布局和跨标签提交脚本为 `.local-tests/verify-group-tag-layout.cjs`。
- 本轮 28 项兼容测试通过（含 2 项新布局回归），ESLint、Prettier 与差异检查通过。
  完整测试首次冷启动超过 60 秒后按模块定位；新测试的 CSS 读取适配 Node URL 后重跑全套通过。

## 未验证与共享影响（原验收）

- 已运行 SQLite 用例；MySQL/PostgreSQL 采用兼容 GORM 设计并完成源码审查，但未进行实库测试。
- 未执行真实远程图片可用性、生产数据库迁移或线上部署。
- tokens-pro/sub2api：仅用户分组响应追加展示元数据，鉴权、模型 ID、计费和中继接口无已知契约改变，
  不需要下游同步发布。未自动发送跨项目通知。
- NewAPIForDouDi：建议在分组数量多时分享交互方案，移植前确认其稳定分组 ID 与模板契约。
- NewAPIModifyByGang：同样具备可复用价值，但应按其选组组件和分组存储适配，不建议直接照搬源码。
  本次未向两个仓库发送消息或创建 Issue/PR。
