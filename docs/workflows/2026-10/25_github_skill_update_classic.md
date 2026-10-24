# Classic GitHub 技能来源、更新、删除与说明长度

状态：本地实现与验证完成，未部署。基于 `origin/custom-main` 的 `8d0f1874db8b8a34984fdfc7380ca51bf9e7c056`。
前端依赖后端同批新增的来源字段、更新接口、删除接口和 16 KiB 说明上限，不能单独部署到旧后端宣称能力完成。

## 范围

仅修改 Classic 提示词中心（`web/classic/src/pages/PromptCenter/**`）与 Classic 语言资源，保持现有浅蓝卡片布局。
Default 模板 `web/src` 由独立工作项处理，本记录不宣称修改 Default；后端 Go 代码不在本工作项范围内。

1. GitHub 来源与一键更新：管理员管理模式显示来源 URL、引用、目录和短提交号，并提供更新操作。
2. 管理员删除技能：管理模式下每张卡片在编辑按钮旁提供危险样式删除按钮，二次确认后删除。
3. 取消用途说明 512 字符限制：单技能和合集内技能说明输入不再设置字符数上限。

## 契约

### 来源与更新

- 管理员模板 JSON 携带 `repository_source: { url, ref, path, commit, imported_at, content_hash, locally_modified }`；
  导入草稿通过该字段保留来源，保存时原样提交。普通用户目录不返回来源，界面也不显示来源或更新入口。
- 仅当 URL 为不带端口、凭据、查询和片段的 `https://github.com/...` 时生成外链，其余来源仅以纯文本显示。
- `POST /api/prompt-center/admin/:id/update-repository`，请求 `{ version, overwrite_local }`，
  成功返回 `{ success: true, data: { template, updated } }`，界面以服务端模板替换该卡片并提示已更新或已最新。
- 已知本地修改先确认覆盖；服务端返回 HTTP 409 且 `code=repository_local_changes` 时进入同一确认流程。
  普通版本冲突（HTTP 409，仅 `message`）与其他失败保留原卡片并显示错误，可以重试；进行中禁止重复提交。
- 更新由后端保留模板 ID、分类、白名单分组、上下架、排序及站点顶层标题/说明，替换正文和引用文件。
- 编辑器内重新导入（压缩包、GitHub、商店）沿用原有契约：生成未发布的新副本，不复用已有模板 ID/版本，
  来源只取本次导入结果，不继承原模板的 `repository_source`。保留已有模板身份的刷新只通过一键更新完成。

### 删除

- `DELETE /api/prompt-center/admin/:id?version=<当前模板 version>`，通过 `promptRequest` 发送。
- 成功：HTTP 200 `{ success: true, data: { id, uninstalled_tokens } }`；前端从列表移除该卡片，
  并提示“已删除并从 N 个令牌中自动卸载”。
- 版本冲突：HTTP 409 `{ success: false, message }`；其他失败：HTTP 200 `{ success: false, message }`。
  两者都保留卡片，在确认框内以 `role="alert"` 显示服务端消息，按钮恢复可用。
- 确认框使用 Semi `Modal`，提示删除后所有已安装该技能的令牌会自动卸载且无法撤销；
  请求进行中禁用确认、取消、关闭和卡片删除按钮，并以同步引用防止重复提交。
- 仅在管理员“管理模板”模式出现；普通用户、管理员浏览模式及令牌装载列表不显示删除按钮。

### 用途说明长度

- 删除 `PromptTemplateEditor.jsx` 与 `BundleSkillsEditor.jsx` 中说明输入的 `maxLength={512}`；
  `validation.js` 与 PromptCenter 文案中没有其他 512 字符限制。前端不新增字符数上限，
  后端以每条说明 16 KiB（16384 UTF-8 字节）作为存储/目录安全边界，超限由后端返回明确错误。

## 对前任实现的修正

- 前任提交将“编辑已有模板时重新导入”改为保留模板 ID、版本和站点设置，并在导入结果缺少来源时继承旧来源，
  同时改写了两项既有回归用例。这改变了既有“导入生成未发布副本”的契约，与 Default 实现和后端工作记录不一致，
  且会让压缩包副本误带 GitHub 来源、被后续“一键更新”覆盖。现已恢复原行为与原用例，并新增来源不继承回归用例。
- 前任遗留的 `add-missing-keys.mjs` 已不存在于工作区；`web/classic/*.log` 测试日志为被忽略的临时文件，已删除。

## 新增国际化键

`Delete {{title}}`、`Delete skill?`、
`Deleting “{{title}}” automatically uninstalls it from every token that installed it. This action cannot be undone.`、
`Deleted {{title}} and uninstalled it from {{tokens}} tokens`，8 份语言文件（en、zh、zh-CN、zh-TW、fr、ru、ja、vi）均补齐真实翻译，
只在文件开头新增键，不重排既有键。`Delete`、`Cancel` 复用已有键。

## 验证结果

在 `web/classic` 下执行（Windows Git Bash，Node v24.11.0，复用已有依赖目录，未安装或更新依赖）：

- 先写用例再实现：新增/恢复的 11 项用例在实现前全部按预期失败（导入保留身份与继承来源、说明被截断为 512 字、
  缺少删除按钮），实现后通过。
- `node ./node_modules/vitest/vitest.mjs run src/pages/PromptCenter/__tests__/`：5 个文件、62 项全部通过
  （`center` 16、`bundles` 12、`progressive` 16、`repository-update` 11、`delete` 7）。
- `node --test src/pages/PromptCenter/__tests__/i18n.test.mjs`：通过，已纳入 `TemplateDelete.jsx`。
- 受影响 JS/JSX 的 `node node_modules/eslint/bin/eslint.js <files>` 通过；受影响 JS/JSX/CSS/测试与 8 份语言文件的
  `node node_modules/prettier/bin/prettier.cjs --check <files>` 通过。
- `node node_modules/vite/bin/vite.js build` 成功（约 58 秒），仅有既有大 chunk 警告。
- 未执行：`bun run i18n:sync`（便携 Bun 无 `bunx` 入口，且本次仅手工新增 4 个键，由 i18n 覆盖测试校验）；
  真实后端联调、GitHub 拉取与生产写入（接口由 Axios adapter 在边界模拟，后端契约由主工作项验证）。

## 已知限制

- fr、ru、ja、vi 中既有 `Cancel` 键的译文仍为占位值（如 `Cancel (fr)`），属于既有问题，本次未改动。
