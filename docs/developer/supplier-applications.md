# 供货商申请与分组报价（#408）

## 目标与范围

让有上游资源的登录用户提交站点资料和 1–20 个分组报价，平台统一比价后联系合作；管理员可分页、搜索、查看并记录处理结果。流程：**提交报价 → 平台比价 → 联系合作**。收集阶段不要求、也不接受 API 密钥。

- 本期只覆盖申请、报价展示和管理员处理闭环。处理状态**不会**创建渠道、启用资源或修改渠道、分组、计费配置；正式接入仍走渠道管理流程。
- 站点地址仅作为资料保存，服务端和前端都不会请求该地址。
- 前端只做 Default（`web/src`）。Classic 的适配范围见[模板范围](#模板范围)。

## 数据与权限

- `SupplierApplication`（表 `supplier_applications`）：一次不可修改的提交。字段：`user_id`、`site_name`、`site_url`、`contact_type`、`contact`、`remark`、`status`、`admin_note`、`handled_by`、`handled_at`、`version`、`fingerprint`（不对外输出）、`created_at`、`updated_at`。
- `SupplierApplicationQuote`（表 `supplier_application_quotes`）：每个分组一行，`application_id`、`sort_order`、`group_name`、`platform`、`ratio`、`models`（JSON 文本数组，`TEXT` 兼容三种数据库）、`description`。创建申请与全部报价在同一事务内提交。
- 两个模型同时注册在 `migrateDB` 与 `migrateDBFast`，仅用 GORM 方法和显式转义的 `LIKE`，SQLite/MySQL/PostgreSQL 通用；没有新增外部依赖，也未触及现有表。
- 权限资源 `supplier_applications`（`service/authz/resources_supplier_applications.go`）：`read`、`write`，内置 admin 默认拥有，超级管理员始终拥有，可在现有“管理员权限”编辑器中单独撤销。
- 用户接口只读取会话用户自己的申请；管理接口要求 AdminAuth 加对应权限。管理范围只由路由构造，不能由请求参数或用户角色自动切换（管理员自己的申请同样走个人接口）。

## API 契约

统一返回 `{success, message, data}`。所有接口 `Cache-Control: no-store`。

个人接口（UserAuth）：

- `GET /api/supplier-applications`：本人列表。`p`、`page_size`（默认 20，最大 100）、`status`。搜索词和 `user_id` 被忽略。data 为 `{page,page_size,total,items}`。
- `POST /api/supplier-applications`：提交申请，写入按用户限流（`UserCriticalRateLimit("supplier-application-write")`）。请求体限 64 KiB，尾随第二段 JSON 返回 400，超限 413。
- `GET /api/supplier-applications/:id`：本人申请详情；他人或不存在均返回 404。
- 没有更新或删除接口：提交后不可修改，更正需重新提交，历史记录保留。

个人视图字段（唯一形态）：`id,site_name,site_url,contact_type,contact,remark,status,created_at,groups[]`。**不含** `admin_note`、`handled_by`、`handled_at`、`user_id`、`version`。分组项：`id,group_name,platform,ratio,models[],description`。

管理接口（`/api/supplier-application-admin`，AdminAuth）：

- `GET /`：需要 `supplier_applications:read`。支持 `p/page_size/status/keyword/user_id`。`keyword` 不区分大小写，匹配站点名称、站点地址、联系方式和分组名称；纯数字还匹配申请 ID 与用户 ID。`%`、`_` 按字面量搜索（`ESCAPE '!'`）。
- `GET /:id`：需要 `read`，返回完整资料。
- `PUT /:id`：需要 `write`，请求 `{status, admin_note, version}`，返回更新后的管理视图。
- 管理视图额外字段：`user_id,username,admin_note,handled_by,handled_by_name,handled_at,version,updated_at`。用户名在每页内批量解析，无逐行查询。

### 写入与校验（后端为准，前端同步）

| 字段                   | 规则                                                                                |
| ---------------------- | ----------------------------------------------------------------------------------- |
| `site_name`            | 必填，≤100 字符，单行                                                               |
| `site_url`             | 选填，≤500 字符；填写时必须是带主机名的 `http`/`https` 地址，禁止内嵌账号密码和空白 |
| `contact_type`         | `wechat/qq/telegram/email/phone/other`                                              |
| `contact`              | 必填，≤200 字符，单行                                                               |
| `remark`               | 选填，≤1000 字符                                                                    |
| `groups`               | 1–20 个                                                                             |
| `groups[].group_name`  | 必填，≤64 字符，单行                                                                |
| `groups[].platform`    | `openai/anthropic/gemini/grok/deepseek/other`                                       |
| `groups[].ratio`       | 必填，有限正数，0.0001–1000，至多 4 位小数；缺失（含显式缺字段）拒绝                |
| `groups[].models`      | 选填，≤20 个，每个 ≤100 字符，不可为空、不可重复                                    |
| `groups[].description` | 选填，≤500 字符                                                                     |
| `admin_note`           | 选填，≤2000 字符                                                                    |

- 所有文本按 UTF-8 校验、禁止零字符和控制字符（备注类多行字段允许换行/制表）。
- **不收集凭据**：除上述字段外，包含明显接入凭据形态的文本（`sk-` 长串、`Bearer` 令牌、`password/secret/api_key/access_token/authorization/密码/密钥` 后接 `=`、`:`）一律 400。这是尽力而为的提示，不是密钥扫描器。
- 校验错误返回 400，消息只带字段路径（如 `invalid supplier application input: groups[0].ratio`），不回显提交内容。
- 倍率保留供应商原始填写口径，不与本站分组倍率或计费换算；页面有文字解释。
- 状态：`pending`（默认）、`contacted`、`cooperating`、`rejected`。

### 防重复与限额

- 对规范化（去首尾空白）后的整份提交计算 SHA-256 指纹，`(user_id, fingerprint)` 唯一索引。相同用户相同内容再次提交（双击、网络重试、并发）返回 409，且不会留下孤立报价。
- 同一用户 24 小时内最多提交 20 次，超出返回 429。该限额按用户独立计算。
- 前端提交期间同步锁定并禁用按钮，失败后保留已填写内容。

### 管理处理与审计

- 处理使用 `version` 乐观并发：版本陈旧返回 409，前端刷新后重试。行锁走 `lockForUpdate`，SQLite 自动跳过。
- 状态与备注无变化时不写入、不产生审计。有变化时更新 `handled_by`（当前管理员）与 `handled_at`，递增版本。
- 审计复用 `recordManageAudit`，动作 `supplier_application.review`，参数为申请 ID、前后状态、备注长度和申请人 ID；**不写入内部备注和联系方式**。

## Default 前端

- 路由：个人 `/supplier-applications`，管理 `/supplier-application-management`（`beforeLoad` 校验角色与 `supplier_applications:read`，否则 `/403`）。
- 导航：顶部导航“供货商申请”（`HeaderNavModules.supplier_application`，默认开启，未登录时按现有“需要登录”逻辑提示）；个人侧栏 `personal.supplier_application`；管理侧栏 `admin.supplier_application_admin`（需 `read` 权限）。三个开关均出现在系统设置的导航与侧栏模块表单中，缺失的旧配置按默认开启补齐。
- 申请页：流程说明、醒目提示“请勿在此填写 API 密钥”和“提交后无法修改”；表单含站点信息、1–20 个动态分组（数量/上限提示）、主要模型标签（回车添加、可移除，忽略输入法组合态）；“我的提交”展示时间、状态、站点与分组报价详情，空列表显示“暂无提交”。
- 管理页：搜索、状态与用户 ID 筛选、分页表格；详情抽屉显示完整资料、全部分组报价、处理人和处理时间，并含状态与内部备注表单；无 `write` 权限时只读。
- 长分组名、长模型名使用 `overflow-wrap:anywhere` 换行，桌面与 390px 手机宽度下无横向溢出。
- 七语翻译已写入 `web/src/i18n/locales/*.json` 的 `translation` 命名空间；新增权限在权限编辑器中的标签复用同一批键。

## 模板范围

- **Default**：本变更完整覆盖（入口、表单、我的提交、管理页、导航与设置、国际化）。
- **Classic**（`web/classic`）：**本期未实现页面、导航或设置入口**。后端接口和权限与模板无关，Classic 可直接复用同一契约；Classic 的侧栏设置合并（`mergeAdminConfig`）和顶栏设置加载后整体保存，都会保留已存在的新增配置键；它们的“重置”会恢复默认，而缺失键在 Default 中按开启处理，不会隐藏入口。该保留行为只做了源码核对，未在 Classic 浏览器中实测。需要在 Classic 提供入口时，另行增加页面、侧栏模块与翻译即可，无后端改动。

## 已知限制

- 平台与联系方式枚举由本功能确定，新增类型需同时改后端校验与前端选项。
- 不发送外部通知，不进入 MCP 目录，不提供导出。
- 凭据检测为模式匹配，无法识别任意字符串形式的密钥；页面提示与服务端拦截共同降低风险。
- 内部备注只存在于管理接口响应与审计之外的数据库字段，没有加密存储。

## 测试

- 后端（`router/supplier_application_router_test.go`，真实路由与 SQLite）：个人隔离与 404；注入的归属/状态/备注被忽略；重复与并发只保存一份；24 小时限额按用户独立；30 项非法输入（缺失、超长、控制字符、协议、内嵌密码、凭据形态、分组数、平台、倍率边界与精度、模型数/重复/长度）均 400 且不落库；边界合法值（20 组、20 模型、倍率 0.0001/1000）通过；管理员完整视图、搜索、筛选；处理记录处理人与时间、审计不含备注与联系方式；版本冲突 409；个人响应不含任何内部字段；单独撤销读/写权限；不创建渠道、不改动分组。
- 权限目录：`service/authz/authz_test.go` 的期望含新资源。
- 前端（`web/src/features/supplier-applications/__tests__`，17 例）：空表单与字段错误、URL/倍率校验、分组 1–20 与计数、标签回车/去重/移除、提交载荷与重置、重复点击只发一次请求、失败/重复保留内容、我的提交空态与不显示内部字段、管理页权限、搜索详情与按版本保存、只读模式、侧栏默认补齐与显式关闭。
- 浏览器：隔离 SQLite 的真实服务加 Chromium，桌面 1280px 与手机 390px 走完 33 项真实路径检查，见[工作记录](../workflows/2026-11/26_supplier_applications.md)。
