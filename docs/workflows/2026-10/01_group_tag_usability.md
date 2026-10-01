# MAO-6 标签易用性修正

## 范围与方案

- 本轮仅修改 Classic：实际入口是 `EditTokenModal`、`GroupTable` 和 `GroupTagEditor`。
  Default 不属于本轮截图和需求范围，不修改。
- 令牌创建和编辑隐藏「未分类」与没有可用分组的标签；保留「所有分组」入口，
  未绑定分组与虚拟 auto 仍可选。当前标签删除或变为空时，筛选状态同步回到全部。
- 后台分组管理保留未分类和空标签，筛选改为约 68px 高的紧凑卡片，最多六个 Logo 分两行。
- 绑定分组新增可保存的多关键词输入，去除首尾空白、不区分大小写、按任一词的包含关系匹配。
  有名称时不匹配内部 code；名称缺失才回退 code。保存时按 ID 去重追加，保留已有绑定顺序。
  后端会在新建或改名分组的保存事务中自动应用规则。既有绑定只增不减，移除关键词不会解绑。
- 用户确认 Gemini Logo 已恢复；本轮不更改 Logo 映射或渲染器。

## 契约与边界

新增 `group_tags.match_keywords` 文本 JSON 字段并由迁移自动创建；API 读写 `match_keywords` 数组。
API、权限、路由、计费及令牌提交契约均不改变。绑定继续使用稳定 Group.Id，
显示使用当前名称。只新增 Classic 七种语言翻译键，无新增依赖。
详见 [分组标签契约](../../developer/group-tags.md)。

## 验证结果

- 两个相关兼容测试文件共 16 项通过，覆盖完整创建/编辑令牌提交、隐藏空标签、失效回退、
  未分类管理筛选、显示名称匹配、缺失名称回退、空关键词、无匹配、去重与顺序。
- 后端分组标签匹配词、分组新增/改名自动绑定、规则移除保留既有关联的模型和控制器测试通过。
- 合并前修正保存响应缺少自动绑定的问题，POST/PUT 返回事务内最终关联；
  标签保存、删除和分组保存共用数据库事务门闩，防止多节点新增漏绑定和删除后的孤儿关联。
  独立 SQLite 连接并发回归、单独运行标签用例均通过；MySQL/PostgreSQL 尚未进行并发实跑。
- 首次 CI 发现路由标签集成测试未创建事务锁所需的 `options` 表；补齐 fixture 迁移，
  保持生产锁协议不变，并重新验证路由用例。
- 默认线程池两次触及 60 秒进程上限。改用 forks 后分别在 27.78 秒、23.18 秒完成；
  保留测试包装器的 60 秒上限，没有修改项目测试配置。
- 涉及 JSX 的 ESLint、Prettier、七语言 i18n 同步、Classic 生产构建通过。
  Classic 无 typecheck 脚本，本轮无 TypeScript 修改。
- Playwright 检查 1440px 桌面与 390px 手机端实际组件：令牌无未分类，后台保留未分类；
  六 Logo、长名称卡片对齐；Codex 匹配并加入两个分组。手机端页面宽 390px、弹窗宽 366px。
- 截图位于本地 `output/playwright/mao6-{token,groups,keyword}-{desktop,mobile}.png`。
  演示复用本地 5179 端口 Vite，数据由 fixture API 提供，不代表生产接口验收。
- 构建仍提示既有大 chunk、Browserslist 数据过旧和 lottie eval；浏览器/测试有 Semi UI
  的 React 弃用提示。没有修改这些无关依赖。
- 本轮按用户授权随 `.345` 发布并逐节点更新 maolaoapi；部署结果另行记录。
  稳定分组 ID/code、鉴权、路由和计费契约不变。
- 文档新增链接与 `git diff --check` 通过；开发索引中仍有基线已存在的历史失效链接，
  本轮不修改无关归档路径。

复现测试（在 `web/classic`）：

```powershell
node scripts/run-compat-tests.mjs src/pages/Setting/Ratio/__tests__/group-tags.compat.test.jsx --pool=forks --poolOptions.forks.singleFork
node scripts/run-compat-tests.mjs src/components/common/group-tags/__tests__/selection.compat.test.jsx --pool=forks --poolOptions.forks.singleFork
bun run build
```

## 分享评估

- NewAPIForDouDi：若也有大量分组，显示名称批量绑定与紧凑筛选值得分享；需先确认标签模型及模板。
- NewAPIModifyByGang：同样适用于多平台分组管理，但 Semi UI 样式与当前标签契约不能直接假定兼容。
  本轮未向兄弟仓库发送消息或创建 Issue。
