# 令牌提示词中心

## 目标与范围

Classic 与 Default 提供相同能力的提示词中心：管理员发布文本模板并配置装载分组白名单，
用户安装到自己的令牌。令牌名称旁以「技能 · 数量」徽标标识已安装数量；全部停用时灰显，
点击查看安装项、启用状态和生效分组。未安装不显示徽标。

首期仅支持文本提示词以及作为文本导入的 Markdown / SKILL.md；不执行脚本、不安装工具、
不提供附件或 Agent 运行环境。支持 Chat Completions、Responses、Claude Messages、Gemini
四种 HTTP 对话请求。其他端点保持原行为，页面明确列出范围。

## 管理与装载契约

- `prompt_templates` 保存 `id/title/description/content/enabled/sort_order/version`。
  白名单关联表 `prompt_template_groups` 使用稳定 `group_id`；显示当前分组名称。
- `token_prompt_bindings` 以 `(token_id, template_id)` 唯一关联，保存 `enabled/sort_order`。
  绑定只记录 ID，不读取或复制令牌密钥，不修改令牌额度、分组与其他限制。
- 模板标题 1–80 字符，说明最多 512 字符，正文最多 32768 UTF-8 字节，上架时至少选择一个实体分组；
  下架草稿可以没有分组。最多 1000 个白名单分组，排序范围 -1000000 至 1000000。
  每个令牌最多安装 8 项，启用项合计正文最多 65536 字节。
- 模板更新递增 `version`，安装引用当前版本，管理员保存后后续请求使用更新内容。
  下架和卸载从后续读取生效；不回写已发出的请求和历史日志。
- 修改正文或重新上架时，同时校验已有启用安装的正文总量；超限拒绝保存。
  分组删除后编辑器显示可移除的失效 ID；可以移除后重新选组，或下架并保留空白名单。
- 安装仅接受当前启用模板，且白名单与令牌当前可用分组至少有一个交集。
  已下架的安装项仍可查看、停用或卸载；不能通过替换列表重新启用已下架项。
- 持久化采用 GORM、普通关联表和 TEXT；兼容 SQLite、MySQL、PostgreSQL。
  运行时直接读取数据库，不使用进程内模板缓存，避免多节点撤销延迟。

## API

所有接口返回 `{success,message,data}`，需要登录。用户接口只操作自己的令牌；管理员管理接口
额外要求 `AdminAuth`，权限由后端校验。

- `GET /api/prompt-center/`：可浏览的启用模板数组。
- `GET /api/prompt-center/admin/`：全部模板数组；管理员。
- `POST /api/prompt-center/admin/`：创建模板；管理员。
- `PUT /api/prompt-center/admin/:id`：更新模板；管理员，提交当前 `version` 防止覆盖并发编辑。
  版本冲突返回 HTTP 409。
- `GET /api/token/:id/prompts`：安装数组，元素为 `{template,enabled,sort_order}`。
- `PUT /api/token/:id/prompts`：完整替换安装数组，请求为
  `{bindings:[{template_id,enabled,sort_order}]}`；必须显式提交数组，空数组卸载全部。
- `/api/token/`、`/api/token/search`、`/api/token/:id` 增加只读的 `prompt_count` 与
  `enabled_prompt_count`。前者包含下架和停用项，后者仅计模板与绑定均启用的项。
  列表批量汇总，避免逐行查询；徽标不表示当前每个分组都生效。

模板响应包括 `group_ids` 与 `group_details:[{id,name}]`，不把分组显示名称用作权限标识。
模板管理支持上下架，不提供物理删除接口，保留安装关系及历史可追溯性。

## 请求与计费

每次选定实际渠道和分组后，使用原始客户端正文重新构造该次请求；按稳定分组 ID 选出
白名单命中的启用模板，并依次注入。排序为绑定的 `sort_order`、模板 ID，保留客户端原有
system/developer/instructions、工具调用、多模态块、显式零值与未知协议字段。

Chat 使用适合模型的 system/developer 角色，Responses 使用 `instructions`，Claude 使用
`system`，Gemini 使用 `systemInstruction` 并兼容已有 `system_instruction`。重试重新从
客户端正文构造，不能重复添加，也不能把前一分组的模板带入新的非白名单分组。
非白名单请求正常调用，不装载该模板；安装不改变路由或扩大令牌分组权限。

正文改写必须覆盖透传路径，并在敏感词过滤前完成。装载后重新计算输入 Token、请求计费快照，
必要时补足预扣，再发上游；上游 usage 仍为最终结算依据。Gemini 系统文本计入本地估算。
消费日志只追加实际装载模板的 ID、标题和版本，不额外记录模板正文。
追加预扣使用独立 `ReserveRequestQuota`：钱包原子校验剩余额度，令牌限额失败回滚新增钱包扣款；
既有结算 `Reserve` 语义不变。阶梯计费保留首次表达式版本，只刷新装载正文与输入估算。

## 页面与验证

两套页面均提供模板浏览、详情、选择令牌、安装列表的排序/启停/卸载，以及管理员编辑和
白名单管理。导入仅在浏览器读取文本，明确不支持包含脚本的完整 Skills。
页面业务模块使用现有带边框和背景的 Card/Panel；徽标包含可访问名称，键盘可打开详情。
两套侧栏均使用 `console.prompt_center` 独立开关，与 `console.token` 分离。

后端验证模板与安装边界、归属、越权、版本冲突、显示名称与 code 不同、分组改名、历史 alias、
下架失效、批量徽标计数；四协议实际出站正文、透传、流式、换组重试、防重复和计费估算。
前端验证空状态、安装/停用/卸载、失败恢复、管理员权限、徽标详情、手机布局和国际化。
Go 测试设置 `-timeout=60s`，另执行 relaykit 独立构建、双模板检查与构建、文档格式和链接检查。

本地模拟上游不能代替真实供应商、三种数据库实跑或生产验收。回滚保留新表，旧版本忽略
新接口和装载关系；无装载的令牌保留原调用行为。

## 2026-10-22 本地验证结果

- 功能分支同步 `origin/custom-main@fdbe81d3a` 后重新验证；主线新增转发和错误脱敏逻辑自动合并，
  无冲突。以下回归覆盖同步后的代码。
- 后端 `model/controller/service/router` 全包测试通过（每包 `-timeout=60s`）；新增四协议
  普通/透传、流式请求出站验证，真实 TokenAuth→Relay 的 Chat 成功响应与钱包/令牌/消费日志
  结算验证覆盖普通/流式×透传/非透传四种组合。后者使用按次价格，不宣称四协议逐 Token 全覆盖。
- `relaykit` 的 Gemini 系统计数回归通过，`GOWORK=off go build ./...` 独立构建通过。
- Default、Classic 页面与徽标行为回归、Default 类型检查、受影响 lint、新增模块格式和双模板构建通过；
  具体子任务结果见 [Default](../workflows/2026-10/22_token_prompt_center_default.md) 与
  [Classic](../workflows/2026-10/22_token_prompt_center_classic.md)。
  Default 全目录 lint 有两处未修改文件的既有错误；Classic `helpers/render.jsx` 有既有混合
  分号格式问题，本次仅增加图标映射并沿用相邻格式，未重写全文件。
- 修改文档已格式化，新增 11 处相对链接验证通过；完整索引检查发现既有历史文档缺失链接，
  本项未扩大修复历史索引。
- 本地独立 SQLite 实例通过管理 API 创建模板并安装到测试令牌；浏览器检查两套实际页面的
  徽标与详情。原工作区未提交修改保持原样。该本地实现阶段未推送、创建 PR、合并到主线或部署。
- 未实跑 MySQL/PostgreSQL 并发、真实供应商或生产环境；组合资金源严格扩额与 Redis 专项
  仍沿用既有回归和代码审查边界。
- 最终 Default 10 个测试文件共 61 项通过；Classic 12 项交互与 1 项国际化检查通过。
  Classic 在并行构建期间两次触及 60 秒限制；停止并行构建后同一测试在 31 秒内通过，
  模块收集耗时 27.44 秒、实际用例耗时 2.21 秒，未放宽超时限制或修改业务断言。

## zzapi 测试部署

用户后续授权安装到 zzapi，已从现场基线整合本功能并完成三节点更新。同镜像在隔离
PostgreSQL 数据库完成管理 API、徽标计数、四协议非流式白名单命中/不命中及扣费审计验收。
线上保持 Classic 主题，双模板资源在隔离实例验证；未调用真实供应商。
源码、备份、迁移、回滚与验收边界见[部署记录](../workflows/2026-10/22_token_prompt_center_zzapi.md)。
