# Default GitHub 技能来源与更新

状态：本地实现和验证完成，尚未部署；范围仅为 Default 模板 `web/src/features/prompt-center`。
后续增量“管理员删除技能”和“取消用途说明 512 字符限制”见文末独立章节。

## 目标与契约

管理员查看 GitHub 导入技能的来源 URL、引用、目录和提交，并通过一键更新获取仓库中的新副本。
普通用户目录不展示管理员来源信息和更新入口，不增加正文查看功能。

导入返回的 `repository_source` 对象完整带入首次保存；编辑已有模板同样带回来源，不经过表单 schema 丢弃。
更新调用 `POST /api/prompt-center/admin/:id/update-repository`，提交当前 `version` 和 `overwrite_local`。
返回 `data.template` 与 `data.updated`；本地修改冲突由 HTTP 409、`code=repository_local_changes` 标识。

更新保留分类、白名单、上下架、排序及站点顶层标题/说明；正文和引用文件会替换。
已知本地修改时先确认；服务端发现新的本地修改时进入同一确认流程。普通版本冲突与网络错误保留现有副本并显示错误。
请求期间禁止重复提交；成功更新模板、令牌安装和令牌列表相关查询。无来源的历史模板不推断仓库。

## 安全与样式

沿用现有浅蓝卡片和主题变量，来源区使用有边界的紧凑模块；仅对安全的 GitHub HTTPS URL 生成可点击链接。
不执行仓库内容、不写客户端密钥、不触发自动轮询。页面文案按项目国际化技能通过脚本维护七语。

## 验证计划

先补失败交互用例，覆盖来源导入后保存、普通用户隔离、管理员来源展示、更新成功/无变更、
异步失败/版本冲突、覆盖确认/取消、服务端本地修改冲突及请求忙碌状态。
执行受影响 Vitest、TypeScript 类型检查、lint、格式和生产构建，结果如下。

## 验证结果

- 新增来源/更新交互用例先出现预期失败，完成实现后通过；包含不安全 URL 不生成链接的回归。
- 提示词中心 8 个测试文件、69 项测试全部通过，其中本次来源/更新职责 11 项。
- Default 类型检查、7 个受影响源码与测试文件的 oxlint、生产构建通过。
- 七语各新增 13 个键，遵循国际化脚本流程并执行 `i18n:sync`；既有翻译值保持不变，
  同步工具同时对既有尾部键排序并保留受保护键的序列化格式。
- 所用依赖与锁文件未变；本地复用已存在的依赖目录，没有全局安装。
- 本记录对应独立前端契约模拟；真实 GitHub 拉取、后端事务、三数据库和生产部署由主工作项独立验收。

## 模板与共享边界

Classic 由独立实现任务处理，本记录不宣称修改 Classic。
来源和更新属于管理员控制台管理接口，不改变现有用户令牌、模型发现、转发或用量响应契约。
本前端依赖后端新增 `repository_source` 与版本保护更新接口，不能单独部署到旧后端宣称能力完成。

## 增量：管理员删除技能

范围：Default 模板 `web/src/features/prompt-center`；新增 `components/skill-delete-button.tsx`，
在 `api.ts` 增加 `deletePromptTemplate(id, version)`，技能卡片管理模式下在“编辑”旁挂载删除按钮。

接口契约（后端由主工作项并行实现，本记录只用模拟响应验证）：

- 请求：`DELETE /api/prompt-center/admin/:id?version=<当前模板 version>`，前端设置 `skipErrorHandler`。
- 成功：HTTP 200 `{ "success": true, "data": { "id": 12, "uninstalled_tokens": 3 } }`。
- 版本冲突：HTTP 409 `{ "success": false, "message": "提示词已被其他管理员更新，请刷新后重试" }`。
- 其他失败：HTTP 200 `{ "success": false, "message": "..." }`。
- 非 2xx 响应体与 200 业务失败统一抛出带服务端 `message` 的 Error；无消息时回退 `Failed to delete skill`。
- 假设：`success=true` 但缺少 `data` 时仍视为删除成功，卸载数量按 0 展示，避免把已生效的删除误报为失败。

交互：

- 仅管理员在管理模式下显示删除按钮（危险样式、垃圾桶图标），可访问名称为 `Delete skill <标题>`；
  普通用户和管理员浏览模式不显示。
- 点击后使用 `@/components/confirm-dialog` 二次确认，说明删除后所有已安装该技能的令牌会自动卸载且无法撤销。
  取消不发请求。
- 成功：关闭确认框，从所有 `['prompt-center', *]` 缓存列表移除该模板，失效 `['prompt-center']`、
  `['token-prompts']`、`['keys']`、`['prompt-center-tokens']`，并以 toast 提示已从多少个令牌卸载。
- 失败（409 或业务失败）：确认框保持打开并以 `role="alert"` 显示服务端消息，卡片保留，可取消或重试。
- 请求进行中：确认与取消按钮禁用，确认框不可关闭，重复点击不会再次提交。

## 增量：取消技能用途说明 512 字符限制

用户要求说明不能有 512 字符限制。前端删除 `components/prompt-template-editor.tsx` 顶层说明与
`components/bundle-skill-editor.tsx` 技能用途说明的 `maxLength={512}`，并删除 `lib/validation.ts`
中顶层 `description` 与 `skills[].description` 的 `.max(512)`，不新增任何字符数上限。
prompt-center 内未发现其他 512 相关限制或提示文案。

后端计划将每条说明上限设为 16 KiB（16384 个 UTF-8 字节），仅作为存储和目录安全边界，并由后端返回明确错误；
前端沿用编辑器现有保存失败提示展示该错误，不在前端重复该字节上限。渐进式发布仍要求说明非空，行为未变。

## 增量验证结果

先补失败用例：`__tests__/skill-delete.test.tsx`（7 项）与 `__tests__/progressive.test.tsx` 新增“技能用途说明长度”2 项。
实现前运行这两个文件，8 项按预期失败：6 项找不到删除按钮，2 项粘贴 900 字符说明后被截断为 512 字符；
“普通用户不显示删除按钮”作为负向守护在实现前即通过。实现后全部通过。

在 `web/` 下执行，结果均为本次实际输出：

- `node ./node_modules/vitest/vitest.mjs run src/features/prompt-center`：9 个测试文件、78 项测试全部通过。
- `node node_modules/@typescript/native-preview/lib/tsgo.js -b`：退出码 0。
- `node node_modules/oxlint/bin/oxlint -c .oxlintrc.json <8 个改动源码与测试文件>`：退出码 0。
- `node node_modules/oxfmt/bin/oxfmt -c .oxfmtrc.json --check <改动源码、测试与 7 个语言文件>`：
  首次检查新测试文件存在换行差异，仅对该文件执行 `--write` 后复查通过。
- 七语各新增 5 个键；以保持既有键序的方式插入后执行 `node scripts/sync-i18n.mjs`，
  语言文件无额外改动，新增键未出现在未翻译报告中。
- 本工作记录以 `oxfmt --check`（默认配置）检查 Markdown 格式通过；`git diff --check` 无问题。

未执行：生产构建、全量 `format:check`（按要求只检查改动文件）、真实后端联调与 Classic 模板验证。
删除和 16 KiB 说明上限的后端事务、令牌自动卸载及三数据库兼容由主工作项独立验收。
