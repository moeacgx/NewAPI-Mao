# 工单中心与工单管理

## 目标与范围

Default 和 Classic 均提供个人工单中心；管理员仍可提交和查看自己的工单，另有工单管理入口处理所有工单。登录后的控制台提供浮动创建按钮，点击后打开创建表单。通知复用现有通知中心的 Bot、任务、接收目标、模板、队列和重试。

本期包含文本会话、分类、优先级、列表筛选、分页、关闭和重新打开，以及管理员配置的
“用户提交后自动回复一次”、管理员批量改状态/软删除以及超时自动归档。不包含附件、自动分配、SLA、外部邮件收件或新增站内通知收件箱。

## 数据与权限

- `Ticket` 保存用户 ID、标题、分类、优先级、状态、版本号和创建/更新时间；`TicketMessage` 保存工单 ID、作者 ID、作者角色、正文、创建时间。
- 标题最多 120 个 Unicode 字符，消息最多 10000 个；分类为 `general/billing/technical/account`，优先级为 `low/normal/high`。
- 状态为 `open/processing/resolved/closed/archived`。新工单为 open；人工管理员回复进入 processing，用户回复进入 open；closed/archived 禁止追加，必须显式重新打开。创建时的自动确认消息保持 open，不代表人工接单。
- 用户接口只读取或更新当前会话用户所属工单，管理员个人入口也不例外。管理员接口要求 AdminAuth 和 `tickets:read/write` 权限。
- 用户可设置 closed 或 open；管理员可设置所有合法状态。状态变更不发送通知。
- 所有用户和管理员变更采用事务、`lockForUpdate` 和服务端版本条件更新；SQLite 不使用不支持的 FOR UPDATE。版本冲突返回 409，客户端刷新后重试，不自动重放写操作。
- 模型同时注册普通迁移和快速迁移；通过 GORM 保持 SQLite/MySQL/PostgreSQL 兼容，不修改计费和转发。
- 管理员批量软删除设置 `deleted_at`，列表/详情/回复/状态接口均不可再访问。工单、消息和事务内 `TicketActionLog` 保留，不提供物理清理和回收站恢复入口。

## API 契约

统一返回 `{success, message, data}`，失败返回适当 HTTP 状态码。

- `GET /api/tickets`：本人列表；支持 `p/page_size/status/keyword`。data 为 `{page,page_size,total,items}`。
- `POST /api/tickets`：`{title,category,priority,content}`，data 返回新建 Ticket。
- `GET /api/tickets/:id`：data 为 `{ticket,messages,has_more}`，messages 为最近 50 条按 ID 升序；可用 `before_message_id` 读取更早消息。
- `POST /api/tickets/:id/messages`：`{content}`，data 返回新增 TicketMessage。
- `PUT /api/tickets/:id/status`：`{status}`，data 返回更新后的 Ticket。
- 管理端使用同样数据契约，将 `/api/tickets` 换成 `/api/ticket-admin`；无管理端创建接口；列表额外支持 `user_id`。

管理员自动回复配置接口：

- `GET /api/ticket-admin/automation/auto-reply`：需要 `tickets:read`，返回 `{enabled,content}`。
- `PUT /api/ticket-admin/automation/auto-reply`：需要 `tickets:write`，请求为 `{enabled,content}`。
  启用时 `content` 必须为非空文本，最多 10000 个 Unicode 字符；关闭时可保留正文，便于稍后重新启用。

配置存储在主库 `Option` 的 `TicketAutoReplyConfig` 键中，默认关闭。用户创建工单时，服务端
在同一事务内追加一条 `author_role=admin`、`author_id=0` 的系统回复；创建事务回滚时不会留下
半条自动回复，后续修改配置也不会重新发送历史回复。自动回复不额外产生通知，
新建工单只入队一次 `ticket_user_message`（`action=created`）。

### 批量操作与超时归档

- `PUT /api/ticket-admin/batch/status`：`{tickets:[{id,version}],status}`，返回 `data:{updated_count}`。
- `POST /api/ticket-admin/batch/delete`：`{tickets:[{id,version}]}`，返回 `data:{deleted_count}`。两者仅管理员且拥有 `tickets:write` 可操作。
- 每批 1–100 个不同正整数 ID 和正版本；非法/重复输入 400，任一不存在或已删除 404，任一版本陈旧 409。先锁定并检查全部记录，整批提交或回滚；状态变更不发送通知，但仍验证版本。前端只勾选当前页，操作前确认，翻页/筛选/切换账号清空选择。
- `GET/PUT /api/ticket-admin/automation/auto-archive`：`{enabled,inactive_days,statuses}`，读取 `tickets:read`、保存 `tickets:write`。默认关闭、30 天、resolved/closed；天数整数 1–3650；状态非空、无重复且只允许 open/processing/resolved/closed。
- 超时从最近的创建、回复或状态变化（`updated_at`）起算；达到截止时间即符合条件。启用后包括既有工单，管理员可显式选择归档长期未处理工单；新回复与重开会重新计时。
- `ticket_auto_archive` 复用系统任务主节点调度和数据库租约，每分钟执行、每轮至多 100 条、30 秒执行期限；逐笔事务重读配置并锁定工单、再次核对状态和时间，避免覆盖新回复或处理已删除记录。归档后不重复命中，失败不会留下半笔状态/审计；已完成的其他工单保留成功结果。任务历史保存 `archived_count` 和脱敏错误。
- 批量改状态、软删除、自动归档在主库事务内保存 `TicketActionLog`（工单、操作者、动作、前后状态、版本和时间）。自动归档操作人 ID=0；批量改状态、软删除和自动归档都只留审计，不发送外部通知。

Ticket 字段为 `id,user_id,title,category,priority,status,version,created_at,updated_at`；TicketMessage 字段为 `id,ticket_id,author_id,author_role,content,created_at`，author_role 为 `user/admin`。列表不返回正文，不接受客户端指定归属和作者。

列表按 updated_at DESC、id DESC 稳定排序；页大小默认 20，超过 100 时截为 100，页码限制 1 至 1000000。负值、非数字、无效状态返回 400。消息历史使用严格 id < before_message_id，始终限制消息量。

写入请求完整读取且限制 64 KiB，超限返回 413，尾随第二段 JSON 返回 400。个人写入按用户限流，管理接口沿用既有管理员豁免规则；归属错误和不存在均返回 404，避免泄露他人工单。

## 通知契约

### 后台待处理提醒

Classic 与 Default 数据看板展示待处理、进行中、已解决、已关闭四项统计，每项点击进入对应状态列表。顶部入口进入完整工单管理列表。
先验证 `tickets:read`，再请求 `GET /api/ticket-admin/stats`，返回 `{success:true,data:{open:0,processing:0,resolved:0,closed:0,archived:0}}`。
接口仅管理员可访问，使用一次状态分组聚合；排除软删除工单，空状态返回零。归档独立计数，不计入已关闭。没有数据库迁移。
后台右上角浮动提醒仅使用同一响应的 `open`，不把进行中数量合并进提醒。
人工管理员回复后的 `processing` 不属于待处理；自动确认消息保持 `open`。
数量为零不浮动提示，手动关闭后当前连续后台访问期间不重复提示；无遮罩、不抢焦点。
点击入口进入 `/console/ticket-management?status=open`。可见页面每 60 秒刷新，工单操作后及返回标签页时刷新，
失败不继续显示旧数量，权限撤销和账号切换清理状态。Classic 路由为 `/console/ticket-management?status=open`，Default 为 `/ticket-management?status=open`。详见
[Classic 实现](../workflows/2026-10/26_classic_pending_tickets.md)与[Default 实现](../workflows/2026-10/26_default_pending_tickets.md)。
状态概览的当前接口、布局与验证见[工作记录](../workflows/2026-10/26_ticket_status_overview.md)；此前工作记录中的列表计数接口为历史实现。

### 外部通知

两套模板共用侧栏设置键 personal.tickets 与 admin.ticket_management，新用户默认启用。
已有用户未指定这两个新键时，前端按默认值补齐；明确关闭的设置继续保留。

工单只提供一个通知事件 `ticket_user_message`：用户新建工单（`action=created`，
`action_label=新工单`）或用户追加回复（`action=replied`，`action_label=用户追加回复`）。
管理员回复、显式状态修改、批量改状态和自动归档都由管理员或系统发起，不发送通知。

变量为 `mention,ticket_id,title,user_id,category,priority,status,actor_id,actor_role,create_time,action,action_label,content`，
`actor_role` 固定为 `user`。`content` 在保存本次用户消息的事务内冻结，不取自动回复或后续消息。
正文摘要限制为 HTML 转义后 2000 个字符，超长附截断说明；工单正文仍完整保留。
默认模板增加正文，精确旧默认自动升级，自定义模板保持不变。历史事件缺少正文时显示
“历史通知未保存正文，请查看工单”，不查询或重写历史内容。

用户正文会随通知发送给 Root 配置的外部接收目标，并进入 Root 可访问的通知历史。
接收范围由配置者确认，系统不校验 Telegram 群成员身份。详细契约、验证与限制见
[工单正文通知](../workflows/2026-10/28_ticket_notification_content.md)。

事件与业务写入同事务，幂等键使用工单 ID 和版本号。无订阅者时工单照常保存；通知表尚未迁移时沿用可选通知语义，跳过且不补发历史通知，其余入队失败回滚新建或用户回复。管理员回复和状态变更不再依赖通知写入。不自动建立 Bot 或发送测试通知，通知配置保持已有 Root 权限。

create_time 表示本次事件时间。通知发送给 Root 在任务中配置的目标，不自动发送给工单所有者。管理员个人入口的发言按用户身份处理，同样产生 `replied` 事件；管理入口回复不通知。

旧事件 `ticket_created`、`ticket_user_replied` 已合并，`ticket_admin_replied`、`ticket_status_changed` 已下线。
主节点迁移时把前两类任务改为 `ticket_user_message`（仍为旧默认模板的换成新默认模板，自定义模板保留），
停用后两类任务但保留记录，供管理员在通知中心删除。同时订阅了新建与追问的部署迁移后会收到重复通知，
需删除其中一个任务。滚动更新期间旧节点仍产生旧事件，迁移后的任务不再接收，短暂期间可能漏发工单通知。

## 实施与验证计划

### 页面布局约定

Classic 的个人与管理入口统一使用已有 `classic-console-page` 及其容器，
为固定导航保留顶部空间；列表、权限加载与错误状态均遵守同一外壳。
工单弹窗整体限制在视口内并保留四周留白，正文独立滚动，提交与回复按钮必须可达。
Default 详情使用可收缩的单列网格，并允许连续长标题换行，
避免合法长度的标题或正文撑开网格和弹窗。两套模板均需验证窄屏与连续长文本。
管理员工单管理页优先展示工单列表；自动回复与超时归档集中放入默认折叠的“工单设置”区域，分别保留原有配置接口、权限和保存操作。

### 数据保留与回滚

批量删除采用软删除；工单与消息保留在主数据库，账号停用不会改变归属。没有自动物理清理任务。
通知历史继续使用通知中心既有保留策略。升级时由主节点迁移工单、消息和操作审计三张表及软删除字段。
回滚到不识别软删除的工单版本会重新暴露删除记录；回滚前须关闭工单入口或保留软删除过滤修复，不能仅替换旧二进制。归档状态也需要旧前端兼容。
回滚旧程序前应先关闭自动归档并停用工单通知任务，旧版本不识别新增事件及合并后的 `ticket_user_message`；保留新增表以便后续恢复，
不执行自动删表或删除用户记录。

### 验证顺序

1. 后端：先编写权限、生命周期、分页、输入边界、通知回滚与去重测试，再实现模型、控制器、权限注册与路由。
2. Default：工单列表/详情/创建表单、个人与管理员路由、侧栏配置、控制台浮窗、通知变量说明和七语翻译；覆盖可见交互与失败保留输入。
3. Classic：独立实现相同契约，复用 Semi UI 容器；补兼容交互测试和七语翻译。
4. 整合：运行后端测试（60 秒超时）、前端受影响测试、类型检查、lint 和两套构建；检查文档链接、格式和 git diff --check。
5. 验收边界：本地 SQLite 测试不代表 MySQL/PostgreSQL 实库验收；本地构建不代表生产部署或真实 Telegram 投递。

目前状态：后端与双模板已实现，本地五个受影响后端包全量测试、79 项前端相关测试、
Default 类型检查、两套 lint/构建及完整 Go 构建通过。具体命令和边界见
[实施记录](../workflows/2026-10/22_ticket_center.md)。
