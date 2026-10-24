# GitHub 技能导入的超长元数据兼容

## 问题与根因

导入公开仓库 `ljagiello/ctf-skills` 时，`ctf-malware/SKILL.md` 的 `description`
为 520 字符。导入器将其直接映射到站内最多 512 字符的目录说明，导致整个导入失败。
该错误来自技能入口的元数据校验，与仓库中的其他文件是否被过滤无关。

## 方案调整

初版修复把超过 512 字符的说明截断为带省略号的摘要，并把完整说明前置到主文件正文。
用户随后明确要求不能有 512 字符限制，因此改为取消字符数上限：

- 模型层三处说明校验（全文单技能、全文合集子项、渐进式技能）不再按 512 字符拒绝，
  改为单条最多 16 KiB UTF-8 字节（`model.MaxPromptDescriptionBytes`）。该边界只保护 MySQL TEXT 列
  和网关目录分页，约为旧上限的 10 倍以上；超出时返回包含技能名称的明确错误。
- 导入器原样保存 YAML `description`，不截断，也不再向正文追加说明。
- 显示名称仍受站内 80 字符限制（`title` 列宽 320 字节）。名称超长时缩短为带省略号的摘要，
  完整名称前置到主文件正文；元数据不能替代原本为空的正文，合并后仍受单文件 8 MiB 校验。
- 内置商店索引的说明同样改用 16 KiB 字节边界。
- 网关目录分页每页最多 32 KiB；单条说明在 JSON 转义后可能超过该预算，分页至少返回一条，
  保证 `newapi_skills_list` 总能前进。
- 两套前端去掉说明输入框的 `maxLength=512` 与表单校验上限，由后端返回超限错误。

不执行仓库或压缩包中的任何代码。

## 验证结果

- `TestSkillArchiveKeepsLongSourceDescriptionVerbatim`：520 字符真实说明与 4500 字符（14400 字节）
  中文/表情说明原样进入草稿，正文不变。
- `TestSkillArchiveRejectsDescriptionBeyondStorageBoundary`：超过 16 KiB 时返回明确错误。
- `TestSkillArchiveShortensUnicodeNameWithoutBreakingUTF8`、空正文和 8 MiB 文件预算回归保留。
- `TestPromptDescriptionsAcceptLongTextUpToStorageBoundary`：三种模板保存 3600 字符说明后原样读回，
  超过 16 KiB 拒绝。
- `TestGatewaySkillCataloguePagesAdvanceWithLongDescriptions`：转义后超过 32 KiB 的条目每页返回一条；
  修复前该用例返回空页。
- `go test ./model ./service ./controller ./router -count=1 -timeout=60s` 全部通过。
- 真实公开仓库只读取入口元数据确认根因；未向生产站点导入、修改或执行任何技能。
