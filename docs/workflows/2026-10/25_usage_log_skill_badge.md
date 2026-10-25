# 使用日志技能徽标

## 目标

在使用日志中标出每条记录是否使用了技能，方便管理员和用户判断某次请求有没有用到技能。
范围：Default（`web/src`，zzapi 使用）与 Classic（`web/classic`，正式站使用）两套模板的使用日志表格与
详情。纯前端改动，不改后端、接口或日志写入。

## 数据来源（现有字段，不新增）

消费/错误日志 `other` 顶层已有以下字段，不在 `admin_info` 下，管理员与日志所属用户都能读到：

- `prompt_templates`：始终装载（全文）模式实际注入的模板 `[{id,title,version}]`。
- `gateway_skills`：按需读取（渐进式）审计
  `{stage,round,catalogue_count,catalogue_queries,files:[{template_id,skill_id,title,version,path,offset,bytes}]}`。
  令牌在实际分组有可用渐进式技能时，每个私有发现轮次写一条 `stage=discovery`（`round` 从 1 起，
  `files` 为截至该轮已读取的页面），最终生成写一条 `stage=generation`（`files` 为正式生成装载的已读页面）。
  `catalogue_count` 为当次可用技能数。

## 显示规则

按以下优先级为一条日志生成至多一个徽标：

1. `gateway_skills.stage == "discovery"`：灰色“技能发现 · 第 N 轮”，提示这是私有技能发现轮次、单独计费；
   若已有读取文件，提示中列出已读技能名称。
1. 否则汇总“已使用技能”：`prompt_templates` 的标题（始终装载）与 `stage=generation` 时 `files` 中按
   `template_id + skill_id` 去重的技能标题（按需读取）。非空时显示醒目的“技能 N”，提示中分组列出名称。
1. 否则 `gateway_skills.stage == "generation"` 且未读取文件：灰色“未用技能”，提示“可用 N 个技能，本次未读取”。
1. 其余不显示。

名称只用于展示；字段缺失、非数组或解析失败时视为无数据，不报错。详情视图同时以文本列出同样的信息，
便于移动端查看。

## 位置

- Default：时间列中日志类型徽标旁（与现有 WebSocket “WS” 徽标同一行），详情对话框增加技能一行。
- Classic：类型列中类型标签旁，展开详情增加技能一行。

## 测试计划

- 解析函数：四种情形、去重、字段缺失或格式错误。
- 两套表格：渲染对应徽标与提示文本；无技能数据的日志不显示。
- 多语言：新增文案在两套模板全部语言包中存在，并通过编码检查（中文不能是问号）。

## 实现

- Default：`web/src/features/usage-logs/lib/skill-usage.ts` 中 `getLogSkillUsage` 解析、`describeLogSkillUsage`
  生成徽标与详情共用文案；`components/skill-usage-badge.tsx` 在时间列显示（使用为绿色，发现轮次与未用为灰色，
  悬停列出名称，`aria-label` 含完整文本）；详情对话框增加“技能”一行。
- Classic：`web/classic/src/helpers/log.js` 增加同名两个函数；`UsageLogsColumnDefs.jsx` 在类型列显示 Semi `Tag`
  与 `Tooltip`（使用为紫色，其余灰色）；展开详情（`hooks/usage-logs/useUsageLogsData.jsx`）增加“技能”一项。
- 补充约定：发现轮次缺少有效 `round`、未用缺少有效 `catalogue_count`、文件缺少数字 `template_id` 或
  非空 `skill_id` 时不显示；技能标题为空时显示 `skill_id`，模板标题为空时显示 `#id`；消费与错误日志均按数据显示。
- 未用文案的英文与法语改为“Available skills: {{count}}”形式，避免单数时出现“1 skills”。

## 验证

- Default：使用日志测试 19 个文件 91 项通过（新增解析、徽标、详情、语言包测试）；`tsgo -b`、oxlint、oxfmt、
  `rsbuild build` 通过；`sync-i18n` 不改变语言包。
- Classic：新增解析、标签、展开详情、语言包测试通过（解析 9 项、组件 7 项）；`classic-i18n`、Prettier、ESLint、
  `vite build` 通过；既有 `user-filter.test.mjs` 的手写模拟补充两个新函数。
- Classic 使用日志目录中 `filter-autofill.compat.test.jsx` 失败：在未含本改动的主分支代码上同样失败，
  原因是 `bffe91161` 将分组筛选改为下拉选择而测试仍按文本框查找；CI 不运行 Classic 测试，未在本 PR 修改。
- 未在真实站点截图验收。
