# 令牌提示词中心

## 目标与范围

Classic 与 Default 提供相同能力的提示词中心：管理员发布文本模板并配置装载分组白名单，
用户安装到自己的令牌。令牌名称旁以「技能 · 数量」徽标标识已安装数量；全部停用时灰显，
点击查看安装项、启用状态和生效分组。未安装不显示徽标。

支持全文装载与渐进式披露两种模式。旧模板继续全文注入，新建与导入默认渐进式：
先向模型提供技能名称与用途，由网关完成目录发现、主文件及引用文本的按需读取，
再将实际读取内容用于正式生成。不会把技能下载到用户 Agent 的文件系统。
仅支持文本资料，不执行脚本、不安装客户端工具，不提供 Agent 运行环境。
支持 Chat Completions、Responses、Claude Messages、Gemini 四种 HTTP 对话请求；
其他端点保持原行为。渐进式模式要求上游模型支持工具调用。

渐进式能力已完成本地实现与模拟验证，并更新 zzapi 三节点；正式站仍为 `.352` 全文版本。
最终方案与本地验证范围见[渐进式披露记录](../workflows/2026-10/23_skill_progressive_disclosure.md)，
本轮备份、隔离 PostgreSQL 迁移、SQLite HTTP 模拟与公网验收见
[zzapi 更新记录](../workflows/2026-10/23_skill_progressive_zzapi.md)。

## 管理与装载契约

- `prompt_templates` 保存 `id/title/description/content/enabled/sort_order/version/load_mode`。
  白名单关联表 `prompt_template_groups` 使用稳定 `group_id`；显示当前分组名称。
  扩展字段 `kind/category_id/skills_json` 保存技能类型、稳定分类 ID 和合集明细；
  `skills_json` 使用 TEXT，API 只提供解析后的 `skills`，不暴露存储字段。
- `load_mode` 为空或 `eager` 时保持旧全文模式：`kind=prompt` 使用原正文，旧 `kind` 空值按该
  类型读取；`kind=bundle` 包含 2–16 个标题不重复的技能。合集正文由服务端按数组顺序生成
  `## 技能：标题\n正文` 并以空行拼接，忽略客户端派生正文。全文合集仍受单模板和令牌正文
  总量限制，JSON 转义后的 `skills_json` 也不得超过旧 TEXT 列容量。
- `load_mode=progressive` 时，单技能将模板标题、说明和正文镜像到 `skills[0]`，保留该项稳定
  `id/path/resources`；合集包含 2–128 个独立技能，不拼接正文。技能项包含
  `id/title/description/path/content/resources:[{path,content}]`。发布时每项用途说明必填，
  最多 512 字符；主文件和引用文本单文件最多 8 MiB，包内文本总量最多 64 MiB，
  每包最多 4096 个文件（含主文件）。这些内容不占用旧全文模式的正文额度。
- 合集作为一个安装项启停、排序、卸载，共用模板白名单；内部数量通过 `skill_count` 展示。
  分类只用于目录组织，不影响子技能选择或授权。
- 渐进式技能元数据保存在 `prompt_skill_records`，文件索引保存在 `prompt_skill_files`，
  正文按不超过 48 KiB 的 UTF-8 分块保存在 `prompt_skill_chunks`，避免 MySQL TEXT 单列限制。
  新增普通表，不对旧正文列做破坏性 ALTER；文件路径使用 SHA256 作为索引标识。
  技能 ID 全局唯一，同包编辑和重排保持身份，禁止把其他模板的技能 ID 移入本包。
- `prompt_categories` 保存分类 ID、名称和乐观锁版本，名称 1–40 字符且唯一。
  分类可以创建和改名，不提供删除；未分类为 ID 0。分类不授予分组或令牌权限。
- `token_prompt_bindings` 以 `(token_id, template_id)` 唯一关联，保存 `enabled/sort_order`。
  绑定只记录 ID，不读取或复制令牌密钥，不修改令牌额度、分组与其他限制。
- 模板标题 1–80 字符，说明最多 512 字符；上架时至少选择一个实体分组；
  下架草稿可以没有分组。最多 1000 个白名单分组，排序范围 -1000000 至 1000000。
  每个令牌最多安装 8 项。全文模式单模板正文最多 32768 UTF-8 字节，启用全文项合计最多
  65536 字节；渐进式模式使用上述文件和包限制，不提高一次请求的读取预算。
- 模板更新递增 `version`，安装引用当前版本，管理员保存后后续请求使用更新内容。
  下架和卸载从后续读取生效；不回写已发出的请求和历史日志。
- 修改全文正文或重新上架时，同时校验已有启用安装的全文正文总量；超限拒绝保存。
  分组删除后编辑器显示可移除的失效 ID；可以移除后重新选组，或下架并保留空白名单。
- 模板、版本、渐进式文件及分块替换在同一事务中完成，失败保留旧资料和版本。
  切回全文模式时清理该模板的渐进式元数据与文件，全文合集不保留引用资料。
  下架保留资料但拒绝后续读取；卸载或删除令牌只撤销安装，不删除管理员模板。
- 安装仅接受当前启用模板，且白名单与令牌当前可用分组至少有一个交集。
  已下架的安装项仍可查看、停用或卸载；不能通过替换列表重新启用已下架项。
- 持久化采用 GORM、普通关联表和 TEXT；兼容 SQLite、MySQL、PostgreSQL。
  运行时直接读取数据库，不使用进程内模板缓存，避免多节点撤销延迟。

## API

所有接口返回 `{success,message,data}`，需要登录。用户接口只操作自己的令牌；管理员管理接口
额外要求 `AdminAuth`，权限由后端校验。

- `GET /api/prompt-center/`：可浏览的启用模板摘要数组，不含正文 `content`。
- `GET /api/prompt-center/admin/`：全部模板数组；管理员。
- `POST /api/prompt-center/admin/`：创建模板；管理员。
- `PUT /api/prompt-center/admin/:id`：更新模板；管理员，提交当前 `version` 防止覆盖并发编辑。
  版本冲突返回 HTTP 409。
- `GET /api/prompt-center/categories/`：分类数组，登录用户可读。
- `POST /api/prompt-center/admin/categories/`：管理员创建分类 `{name}`。
- `PUT /api/prompt-center/admin/categories/:id`：管理员改名 `{name,version}`，版本冲突 HTTP 409。
- `POST /api/prompt-center/admin/import/archive`：管理员上传 multipart `file` ZIP，返回草稿。
- `POST /api/prompt-center/admin/import/repository`：管理员提交 `{url}` 公开 GitHub 链接，返回草稿。
- `GET /api/prompt-center/admin/store/`：管理员浏览内置商店摘要；`?source=online` 刷新官方仓库。
- `POST /api/prompt-center/admin/store/:id/import`：管理员读取商店条目为草稿，ID 为需 URL 编码的不透明字符串。
- `GET /api/token/:id/prompts`：安装数组，元素为 `{template,enabled,sort_order}`。
  `template` 仅含摘要，不含正文；PUT 保存安装的响应同样不含正文。
- `PUT /api/token/:id/prompts`：完整替换安装数组，请求为
  `{bindings:[{template_id,enabled,sort_order}]}`；必须显式提交数组，空数组卸载全部。
- `/api/token/`、`/api/token/search`、`/api/token/:id` 增加只读的 `prompt_count` 与
  `enabled_prompt_count`。前者包含下架和停用项，后者仅计模板与绑定均启用的项。
  列表批量汇总，避免逐行查询；徽标不表示当前每个分组都生效。

模板响应包括 `group_ids` 与 `group_details:[{id,name}]`，不把分组显示名称用作权限标识。
模板管理支持上下架，不提供物理删除接口，保留安装关系及历史可追溯性。
用户模板及安装摘要含 `kind/load_mode/category_id/category_name/skill_count` 和
`skills:[{id,title,description}]`，不返回主正文、子技能正文、主文件路径或引用资料。
渐进式公共目录只查询元数据，不读取文件与正文分块；管理员接口提供完整编辑内容。
模板保存 JSON 请求上限为 448 MiB，用于容纳单技能镜像字段和 Unicode 转义膨胀；
模型层仍按解码后的单文件 8 MiB、包内文本 64 MiB 校验，不能用编码绕过大小限制。
网关私有技能工具不是新增的公开 HTTP 或 MCP 接口。

## 导入与技能商店

Classic、Default 均提供 ZIP、公开 GitHub 仓库链接和商店导入。导入只返回
`id=0/version=0/enabled=false/category_id=0/group_ids=[]/load_mode=progressive` 的编辑草稿，
不写库、不自动上架。
管理员检查内容，选择站点分类与白名单后通过原保存接口生成站点副本。
导入失败保留原编辑内容；商店更新不会覆盖站点副本，站点修改不会回写仓库。

ZIP 最多 16 MiB、4096 个条目，所有条目声明解压总量最多 64 MiB；被忽略文件也占用此预算。
有 `SKILL.md` 时只将这些文件作为独立技能入口，否则读取 Markdown，按路径排序后导入单项或
合集，最多 128 个技能。读取 YAML front matter 的 name/description，保存用途说明和主正文。
技能根目录内的 Markdown、TXT、JSON、YAML、CSV、TSV、XML、RST、TOML、INI、CFG、
HTML、HTM 和 TEX 保存为引用文本；嵌套技能归最深技能根目录，父技能不能读取子技能资料。
仓库根 README 不作为额外技能，脚本和二进制附件不导入；其他文件存在不会直接导致报错。

主文件路径可保留包内入口位置，如 `skills/review/SKILL.md`；运行时用空路径或 `SKILL.md` 读取。
引用路径相对于各自技能根目录，最多 1024 字节，不能使用大小写任意的 `SKILL.md`。
拒绝绝对路径、反斜杠、越界路径、符号链接、损坏 ZIP、非 UTF-8 和 NUL 文本。
内容仅在内存处理，不落盘、不执行。正文错误分别显示空正文、无效 UTF-8 或实际字节数超限，
ZIP 与仓库导入附带失败文件相对路径；失败不返回部分草稿，也不截断或跳过问题技能。
诊断修复与验证见[技能导入正文错误定位](../workflows/2026-10/23_skill_import_diagnostics.md)。

商店原件位于 [skill-store](../../skill-store/README.md)，编译内嵌可离线使用。
显式在线刷新仅读取本项目 `custom-main/skill-store/`，不跟随重定向，主文件和引用资料均逐文件
校验清单 SHA256。
该目录发布到目标分支后在线内容才可用；本地实现不包含仓库推送和商店发布。
仓库链接导入则使用固定 GitHub API 与原始文件主机，固定提交读取公开技能文件，
不接受私有仓库凭据、任意下载地址或自定义 Git 主机。细节、边界和测试见
[合集与导入记录](../workflows/2026-10/22_skill_bundles_categories.md)。

## 请求与计费

每次选定实际渠道和分组后，使用原始客户端正文重新构造该次请求；按稳定分组 ID 选出
白名单命中的启用模板。全文模式依次注入，渐进式模式先读取目录元数据。
排序为绑定的 `sort_order`、模板 ID，保留客户端原有
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

### 渐进式读取与正式生成

网关先执行独立的私有非流式发现阶段，只向上游提供技能目录及三个内部工具：

- `newapi_skills_list`：按名称、用途或包名搜索目录，支持 `query/offset`，每页最多 32 项和
  32 KiB。初始目录不超过 64 KiB 时完整提供，超过则只提供一页并由模型继续查询。
- `newapi_skills_read`：按 `template_id/skill_id/version/path/offset` 读取，空路径或 `SKILL.md`
  为主文件。读取主文件后才能读取该技能的已披露引用路径；每页正文最多 24 KiB。
- `newapi_skills_files`：读取主文件后按 `query/offset` 搜索引用路径目录，每页最多 32 项和
  8 KiB，不把整包路径一次塞入上下文。

每个客户端请求最多进行 8 轮私有模型调用，每轮最多 8 次工具调用；实际读取正文合计最多
192 KiB。大文件可按 UTF-8 字节分页，起点不得位于字符中间，终点收拢至字符边界。
目录和正文均受预算约束，文件存储上限不代表一次请求会加载整个包。
每次读取及正式生成前重新检查令牌、安装、发布状态、模板版本与当前实际分组。
auto 或跨组重试重新选定实际分组后，从原客户端请求重新开始，不能携带上一组目录和已读资料。

私有阶段只执行这三个读取工具，不执行客户端工具，也不向客户端输出私有回复、工具结果或思考。
正式生成恢复客户端原有 `tools/tool_choice`、`stream` 和输出格式约束，仅注入实际读取的文件页；
最终 SSE 直接流式返回。客户端工具调用仍交给客户端 Agent 处理。
因此这是文本型技能的渐进式披露，不是远程脚本执行或客户端本地技能安装。

每轮私有模型调用与最终生成分别预扣、按实际用量结算并写消费日志和渠道指标，会增加延迟和费用。
已成功结算的读取轮次不会因为后续生成失败而退款；失败轮沿用现有失败与退款逻辑。
正常读取轮次不计为渠道重试，只有外层实际渠道重试计入重试数。
日志 `gateway_skills` 记录阶段、轮次、模板与技能 ID、版本、路径、偏移和字节数，不额外记录正文。

以下状态组合仅在实际命中渐进式技能时返回 HTTP 400，在上游调用前拒绝，
退回入口已完成的预扣，不产生消费：

- Responses 的非 `null` `conversation`：服务端会话可能持久化私有工具结果，`store=false` 也不能
  提供隔离。请提交完整 `input`。现有 Responses 转发会清除 `previous_response_id`，本轮未改变，
  不承诺依赖该字段的服务端历史续接。
- Gemini 的非空 `cachedContent` 或类型不合法值：缓存与私有阶段的系统指令和工具配置互斥。
  请提交完整 `contents`。

旧全文模板及未命中渐进式技能的请求不受上述两项新增限制。

## 页面与验证

### 操练场 API 令牌切换

Classic 与 Default 操练场支持默认账号模式和本人 API 令牌模式。默认模式使用临时令牌，
不会装载真实令牌的提示词；选择已安装模板的 API 令牌后，使用其分组、模型权限、IP 限制、
额度与限流设置，按实际命中的白名单分组装载提示词。普通与流式请求均记录所选令牌并扣减额度。
选择不修改历史消息，仅在当前页面内存保存，生成期间禁止切换。

- `POST /pg/chat/completions`：可选 `X-Playground-Token-Id` 正整数头；不传保留账号模式。
  令牌模式前端不提交 `group`；自定义请求提交的分组仍受令牌绑定校验。
- `GET /pg/token/models`：必须携带同一头，返回所选令牌可用模型列表，采用正式模型发现契约。
- 两个端点均要求登录会话，并按当前用户校验令牌归属；完整密钥仅在服务端使用。
  显式无效、禁用、过期、耗尽或他人令牌会拒绝，不静默回退账号模式。
- 模型发现失败会清空可选模型并阻止发送；切换后移除新列表不包含的模型。
  本人令牌摘要沿用分页/搜索接口，显示分组当前名称和技能数量，不读取完整密钥或提示词正文。

实现和验证记录见[操练场令牌切换](../workflows/2026-10/22_playground_token_switch.md)。

Classic 操练场采用与 Default 相同的居中对话、空状态和底部输入布局，令牌切换位于输入框上方。
模型及账号分组放在底部控件，参数和调试使用侧面板；自定义请求、附件及导入导出继续可用。
该布局调整不替换 Classic 请求实现，也不引入另一套前端整包；见[布局对齐记录](../workflows/2026-10/22_classic_playground_alignment.md)。

两套页面均提供模板摘要浏览、选择令牌、安装/卸载及保存，以及管理员编辑和
白名单管理。两套管理员编辑器均支持模式切换、逐技能用途说明和引用文本编辑；
切回全文模式会提示引用资料丢失。普通用户继续只浏览摘要与安装、卸载，不提供正文查看。
渐进式合集总说明可空，每个已发布子技能仍须填写用途；不同主路径的技能允许同名。
Classic 保存按钮与错误提示固定在同一操作区，失败聚焦提示并保留草稿；未执行导入的仓库
地址不参与模板保存校验。Default 单技能在表单校验前用可见字段同步技能元数据，避免旧模板
切换按需模式后被隐藏字段阻断。验证见[模板保存修复记录](../workflows/2026-10/23_skill_template_save_feedback.md)。
直接 Markdown 导入在浏览器读取文本；ZIP、仓库和商店导入由后端限量读取，
保存主正文及支持的引用文本，不执行脚本或工具。
页面业务模块使用现有带边框和背景的 Card/Panel；徽标包含可访问名称，键盘可打开详情。
两套侧栏均使用 `console.prompt_center` 独立开关，与 `console.token` 分离。

后端验证模板与安装边界、归属、越权、版本冲突、显示名称与 code 不同、分组改名、历史 alias、
下架失效、批量徽标计数；四协议实际出站正文、透传、流式、换组重试、防重复和计费估算。
前端验证空状态、安装/停用/卸载、失败恢复、管理员权限、徽标详情、手机布局和国际化。
Go 测试设置 `-timeout=60s`，另执行 relaykit 独立构建、双模板检查与构建、文档格式和链接检查。

本地模拟上游不能代替真实供应商、三种数据库实跑或生产验收。回滚前备份数据库并保留新表；
旧程序不理解渐进式文件，仅停用或卸载不足以保证旧管理目录和安装详情兼容。
仅在尚无渐进式业务模板时回滚旧版；已有数据时优先向前修复，不能把空的旧正文列视为已完成装载。
未安装技能的令牌保留原调用行为。

## 2026-10-22 本地验证结果

以下为旧全文能力的历史基线，渐进式新增验证见后文。

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

## 2026-10-22 zzapi 全文版本测试部署

以下为此前全文能力的部署历史；后续渐进式更新见本页末尾。

用户后续授权安装到 zzapi，已从现场基线整合本功能并完成三节点更新。同镜像在隔离
PostgreSQL 数据库完成管理 API、徽标计数、四协议非流式白名单命中/不命中及扣费审计验收。
线上保持 Classic 主题，双模板资源在隔离实例验证；未调用真实供应商。
源码、备份、迁移、回滚与验收边界见[部署记录](../workflows/2026-10/22_token_prompt_center_zzapi.md)。

操练场发送准备期间也保持身份一致：Classic 图片读取后检查身份代次，切换身份即取消旧发送；普通模式未选择模型禁止发送，自定义请求可在正文指定模型。

## 2026-10-23 渐进式本地验证

- 完整后端回归及四协议本地 HTTP 模拟端到端验证通过，覆盖私有读取、正式输出、透传、最终流式、
  客户端工具恢复、钱包/令牌及逐轮日志。具体资金来源、价格与协议组合以
  [主工作记录](../workflows/2026-10/23_skill_progressive_disclosure.md) 为准，不等于真实供应商验收。
- auto 跨组回归验证首组已结算读取费用保留，最终生成失败后切到非白名单组，恢复原始正文和工具，
  不携带前一组技能；正常内部读取不计渠道重试。见
  [计费与指标记录](../workflows/2026-10/23_skill_progressive_metrics.md)。
- SQLite 验证大中文文件、分块重载、UTF-8 分页、稳定 ID、事务回滚、模式切换清理、版本撤销、
  历史分组别名及公共接口不读取或泄露正文。导入和商店使用本地固定 HTTP 模拟，见
  [存储与导入记录](../workflows/2026-10/23_skill_progressive_storage.md)。
- [Default](../workflows/2026-10/23_skill_progressive_default.md) 和
  [Classic](../workflows/2026-10/23_skill_progressive_classic.md) 相关交互回归、国际化、受影响静态检查
  与生产构建通过；根模块使用本地便携 Go 1.25.5 amd64 携带双前端资源整包构建通过，未改全局 Go。
- Responses `conversation` 与 Gemini `cachedContent` 拒绝场景先验证失败再修复，确认 HTTP 400、
  无上游调用且钱包、令牌和消费日志不变。相关回归及独立 relaykit 构建通过。
- 此本地实现阶段未部署、未调用真实供应商，也未连接 MySQL/PostgreSQL 实库；后续部署证据另列如下。

## 2026-10-23 zzapi 渐进式更新

三应用节点已更新到 `zzapi-progressive-db7e411577fe`，健康且重启次数为零，环境、挂载、端口及
Classic 主题保持一致，PostgreSQL 与 Redis 未重启。既有提示词表的原列、行数与内容哈希保持一致，
旧模板继续 `eager`，不会自动转换；新建或重新导入默认为 `progressive`。

同镜像隔离 SQLite HTTP 验收覆盖超过 32 KiB 的主文与引用资料、普通用户摘要隐藏、内置商店、
Chat 透传普通/流式按需读取及逐轮钱包、令牌和日志结算；隔离 PostgreSQL 15 使用备份恢复应用数据
并验证迁移。公网版本、Classic 两页面入口及渐进式编辑器资源哈希一致，未登录接口仍返回 401。
双模板静态资源通过隔离验收，本轮未做浏览器视觉或真实供应商验收，MySQL 实库未测。

详细镜像、备份、迁移、回滚及 PostgreSQL 维护扩展恢复边界见
[部署记录](../workflows/2026-10/23_skill_progressive_zzapi.md)。
