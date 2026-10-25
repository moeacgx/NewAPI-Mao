# 工单通知合并

## 目标

通知中心的工单事件只保留管理员需要处理的用户动作：

- “新工单已创建”与“用户追加回复”合并为一个事件 `ticket_user_message`（新工单或用户追加回复）。
- 不再提供“管理员回复工单”和“工单状态已变更”通知。

通知只投递到 Root 在通知中心配置的目标，不发送给工单所有者；管理员回复和状态变更都是管理员
或系统自己发起的动作，通知给管理员没有意义。

## 修改

- 后端
  - `model/ticket_notification.go`：事件常量改为 `ticket_user_message`，默认模板
    `{{mention}} {{action_label}} #{{ticket_id}}：{{title}}` 加用户、分类、优先级；负载新增
    `action`（created/replied）与 `action_label`（新工单/用户追加回复），`actor_role` 固定为 user。
  - `model/ticket.go`：新建工单与用户回复入队该事件；管理入口回复与 `ChangeTicketStatus` 不再入队。
  - `model/ticket_bulk.go`、`model/ticket_archive.go`：批量改状态与自动归档不再入队，仍保留操作审计。
  - `controller/ticket_notification.go`：event-types 只返回合并事件，变量白名单加入 `action`、`action_label`。
    旧事件值不再在定义中，新建/更新任务时返回“暂不支持该通知事件”。
  - `model/main.go`：主节点普通与快速迁移都调用 `migrateTicketNotificationTasks`，可重复执行。
- Default：`EVENT_LABEL_KEYS`、`EVENT_DESCRIPTION_KEYS` 改为合并事件，新增两个变量说明与示例，
  `static-keys.ts` 与七个语言包同步（移除旧事件的 4 个名称与 4 个说明键）。
- Classic：事件名称和说明从接口读取后经 `t()` 翻译，代码无改动；七个语言包同步替换键
  （`zh.json` 原本没有这些键）。

## 迁移与兼容

- 迁移：`ticket_created`、`ticket_user_replied` 任务改为 `ticket_user_message`；模板为空或仍是旧默认
  模板时换成新默认模板，自定义模板保留。`ticket_admin_replied`、`ticket_status_changed` 任务停用并保留，
  任务列表显示原事件值，供管理员删除。
- 同时订阅了新建与追问的部署迁移后，两个任务都会接收合并事件，需删除其中一个。
- 滚动更新期间旧节点仍产生旧事件；主节点完成迁移后这些任务不再接收旧事件，短暂期间可能漏发。
  旧节点不认识 `ticket_user_message`，在旧节点编辑该任务会被拒绝。
- 回滚到旧版本：合并事件任务不会被旧节点触发，需要按旧事件重新建任务；停用的旧任务可手动启用。
- 部署前只读核对：maolaoapi 与 zzapi 均没有工单通知任务（maolaoapi 工单数为 0），迁移对两站无实际数据变更。

## 验证

- 后端测试先改为新契约后实现：
  - 生命周期：新建、管理员回复、管理员改状态、用户回复、用户关闭/重开中只产生 2 个事件，
    `action/action_label` 依次为 created/新工单、replied/用户追加回复；同时订阅的旧事件不再入队。
  - 新建与用户回复在通知写入失败时整体回滚；管理员回复与状态变更在通知写入失败时照常成功。
  - 批量改状态、自动归档不产生通知；自动归档在通知写入失败时照常完成。
  - 并发关闭与追问：版本计入每次变更，事件只计新建与成功的追问。
  - 迁移：两类用户任务并入、旧默认模板替换、自定义模板保留、两类管理员任务停用、无关任务不变、重复执行结果相同。
  - 控制器：旧事件不在定义中且新建任务被拒绝；合并事件的默认模板渲染出“用户追加回复”，`{{content}}` 被拒绝。
- 工单相关模型测试连续 3 次通过；`go build ./...`、`go vet` 与 model/service/controller/router 全量测试通过。
- Default：通知中心、工单与 i18n 测试 12 个文件 58 项通过，`tsgo -b`、oxlint、oxfmt 通过，
  `sync-i18n` 未改变语言包。
- Classic：通知中心与工单测试 7 个文件 41 项通过（通知中心测试改为只列出合并事件、切换后使用其默认模板），
  `classic-i18n` 检查、Prettier、ESLint 通过。
- 未运行真实 Telegram 投递。
