# 供货商申请与分组报价（#408）工作记录

## 目标

实现 Issue #408：登录用户提交站点资料与 1–20 个分组报价，管理员分页搜索、查看并记录处理状态与内部备注。设计、API 与校验规则见 [供货商申请与分组报价](../../developer/supplier-applications.md)。

基线：`origin/custom-main`（`8c69094b2`）。独立分支 `Worktree/issue408-supplier-applications`。不推送、不合并、不部署，不改动令牌限额，不触及生产数据库。

## 复用与新增

- 复用：UserAuth/AdminAuth、`authz` 权限注册与 `RequirePermission`、`UserCriticalRateLimit` 与 `AdminRateLimitBypass(CriticalRateLimit)`、`recordManageAudit`、`common.PageInfo`、`lockForUpdate`、Default 的 `SectionPageLayout`/Card/Sheet 与现有侧栏、顶栏、侧栏设置体系。
- 新增：两张表、权限资源 `supplier_applications`、个人与管理路由、Default 页面、七语翻译。

## 关键决策

- 申请不可修改：不提供更新路由；重复内容用 `(user_id, fingerprint)` 唯一索引拒绝，并发安全。
- 内部备注隔离：个人视图是独立的类型，结构上没有内部字段，而不是序列化时过滤。
- 倍率口径：保留原始填写值，不换算；范围 0.0001–1000，4 位小数。
- 审计不含备注与联系方式，仅记录 ID、前后状态与备注长度。
- 凭据拦截为服务端模式匹配，前端提示加后端拦截。
- Classic 本期不实现入口（任务范围限定 Default），后端契约共享。

## 验证（隔离环境）

| 检查                                                                                                                 | 结果                                                                 |
| -------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `go test ./router -run SupplierApplication`                                                                          | 33 例通过                                                            |
| `go test ./router`（全包）                                                                                           | 841 例通过                                                           |
| `go test ./service/authz`；`go test ./service` 与 `go test ./controller` 的 MCP/目录、审计、侧栏、工单、选项相关用例 | 通过（8 / 672 / 73 例）                                              |
| Default `bunx vitest run`（供货商 17 例 + 工单、侧栏、设置、布局、i18n、hooks、lib 相邻套件）                        | 43 文件 243 例通过（需 `NODE_OPTIONS=--no-experimental-webstorage`） |
| `bun run typecheck`、`bun run build`                                                                                 | 通过                                                                 |
| oxlint（全部改动文件）、`format-with-protected-headers.mjs --check`（22 文件）                                       | 通过                                                                 |
| Chromium 真实路径（隔离 SQLite、端口 3107）                                                                          | 33/33 通过                                                           |

浏览器覆盖：桌面 1280px 与手机 390px 的空表单校验、填写 2 个分组和超长模型名、提交跳转“我的提交”、详情展开、重复提交拒绝并保留内容、他人 404、普通用户管理接口 403、申请人自行审批 403、无更新路由、管理员列表/搜索/详情/保存、重新加载后备注与处理人保留、申请人看到状态但页面与接口均无内部备注、无横向溢出。

## 未覆盖与缺口

- Classic 无页面、导航与设置入口（见开发文档“模板范围”）。
- 未在 MySQL/PostgreSQL 上运行；只使用 GORM 方法与跨库写法，仅 SQLite 实测。
- 未执行 Redis 限流路径；限流沿用既有中间件，测试里为避免干扰关闭了全局限流，24 小时上限由数据库层测试覆盖。
- `system-info-section.test.tsx` 与 `group-ratio-log-cache.test.tsx` 共 9 例在未改动的基线上同样失败（本机 Node 26 缺少 `localStorage` 的环境问题），与本变更无关。
- 后端用户默认侧栏配置（`controller/user.go`、`model/user.go`）未加入新键，前端对缺失键默认开启。
- 没有 20 个分组的浏览器全量填写，20 组上限由 Go 与组件测试覆盖。
