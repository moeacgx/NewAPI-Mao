# 工单管理列表优先与设置折叠

## 目标与范围

Classic 管理员进入工单管理时，两个完整设置面板原先排在工单列表之前，占据首屏；本次将列表前置。Default 原本已先展示列表，本次保留该顺序。两套模板均把自动回复与超时自动归档统一收纳到默认折叠的“工单设置”区域。

## 行为与兼容性

- 工单列表排在设置入口之前。
- 设置入口默认折叠，展开后显示自动回复和超时归档两个独立面板。
- 两项配置仍使用现有接口、权限和独立保存操作；没有改动后端数据或调度行为。
- 新增“Ticket settings”七语翻译。
- 设置保持挂载，收起再展开保留未保存草稿；Default 切换账号时重置为折叠状态并清理旧草稿，Classic 继续沿用既有按账号/角色重挂载的隔离行为。
- 设置入口支持键盘聚焦与原生展开语义；Classic 设置内保存按钮限制宽度并允许长译文换行，避免法语在 320px 窄屏向左溢出。

## 验证

本工作树没有 Bun，使用现有 Node 和主仓已安装依赖执行等价脚本，没有安装或更新依赖。

- Default：`vitest run` 执行 `layout.test.tsx`、`batch-actions.test.tsx`、`auto-reply-settings.test.tsx`、`auto-archive-settings.test.tsx`，4 文件共 22 项通过；单工作进程并设置 60 秒最大执行期限。
- Classic：`node scripts/run-compat-tests.mjs` 执行 `settings-layout.compat.test.jsx`、`localization.compat.test.jsx`、`automation.compat.test.jsx`、`archive.compat.test.jsx`，4 文件共 16 项通过；`node --test src/pages/Tickets/__tests__/layout.test.mjs` 的 2 项布局约束通过。
- Default `tsgo -b`、所改文件的 Oxlint/ESLint 和 Oxfmt/Prettier 均通过；两套生产构建通过。Classic 保留既有大资源包和 Browserslist 数据过期警告。
- Default i18n 同步完成，七语均没有缺失键；逐项比对两套语言包，本次仅新增 `Ticket settings` 的值，没有改动既有翻译。
- 修改过的 Markdown 已格式化；新增文档链接和 `git diff --check` 通过。开发文档索引全量扫描仍有本次修改前就存在的断链，本次没有引入新断链。

浏览器使用本地生产构建与模拟会话/工单 API，不代表生产环境验收；记录与截图位于忽略目录 `.local-tests/ticket-settings/`。两套模板的中文、法语、俄语及 1440/390/320px 共 18 组均通过：初始设置折叠，Enter 可展开，收起后表单不可见，面板内部没有横向溢出，保存按钮均在视口内，页面没有 JavaScript 异常。独立只读审查无阻断问题。

## 跨项目影响

没有改变 API、认证、计费、用量或跨项目集成契约。

- NewAPIForDouDi：如果工单规则设置也挤占处理列表，值得参考列表优先和默认折叠的交互；需要按其实际工单组件适配。
- NewAPIModifyByGang：如果同样提供自动回复和归档配置，值得参考统一设置入口；组件结构和样式需按该项目实现核对。

本次没有核实兄弟项目是否存在相同页面，也没有向它们创建 Issue、PR 或发送消息。本工作项在专用 `task/8` 分支完成；未直接写入 `custom-main` 或部署。
