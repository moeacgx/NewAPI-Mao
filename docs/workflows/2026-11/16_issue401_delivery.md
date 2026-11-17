# Issue #401 / #398 / #399 / #400 / #402 双模板集成总交付记录

## 范围与交付门禁

本记录汇总本仓库 Classic 与 Default 的历史能力对照和本轮修复，不是向上游同步。
原六子项冻结后，追加修复 Classic 多密钥模式立即提交问题；当时阶段没有提交、推送、评论、部署或关闭 Issue，并保留并行 MCP 未提交改动；后续授权与新审查状态见下节。
六个子项的实现、独立复审和最终门禁分别保留在专题记录中，本记录只汇总证据与限制。

六个子项如下：

1. [#398 供应商 Logo URL 补齐](16_issue398_logo.md)。
2. [#399/#400 令牌状态与实时活动](16_issue399_400_token_activity.md)。
3. [#402 令牌页 API 地址入口](16_issue402_api_addresses.md)。
4. [F1 普通日志分组名称选择](16_issue401_log_group_selection.md)。
5. [F2 普通管理员返佣业务入口](16_issue401_affiliate_admin_access.md)。
6. [F3 令牌列表分组名称与迁移矩阵](16_issue401_migration_matrix.md)。

追加项：[Classic 多密钥模式立即提交](16_issue401_classic_key_mode.md)；原六子项独立审查结论保留，追加修复独立审查 `ctx_108ffd07f56c` 已通过，未发现 P1/P2。

## 2026-11-16 新一轮独立审查与交付授权

本轮新审查确认的两项 P2 和一项 P3 均已由其他工人返修，再由独立 reviewer 复验闭环：

- 令牌活动首次列表在途时丢失刷新（P2）：按完整主体/查询 scope 保留 pending 并在请求结束后合并补查；reviewer `ctx_b066fbb302a4` 的原失败活动夹具未改动转绿，活动四文件 37/37 通过（28.81 秒）。
- API 地址 HTTP 500 全页跳转（P2）：局部归一化非认证 Axios 故障，保留 401/403、取消原错误和请求 signal；同 reviewer 的原失败地址夹具未改动转绿，地址 16/16 通过（4.09 秒），两份原失败夹具组合 2/2 通过（4.36 秒）。报告：`D:/Temp/issue401-fresh-review-keys/review-report.md`。
- 日志合法 `__proto__` alias 改名重取后丢名称（P3）：仅对该组选项查询关闭结构共享；`ctx_434aefa8fec0` 正式复验原未改 edge 4/4（6.17 秒）、四文件 34/34（24.31 秒）均 exit 0，无新授权、请求循环或 Hide 隐私回归。报告：`D:/Temp/issue401-fresh-review-groups/recheck-report.md`。
- 新 UI reviewer `ctx_855cca0b2af7` 对返佣、Logo、Classic 即时提交独立验证 44/44，通过且无可证实新增 P1/P2/P3：Default 23、Classic 即时提交 10、Classic 返佣 3、原生契约 8。报告：`D:/Temp/issue401-fresh-review-ui/report.md`；未重新执行浏览器，不能把历史截图计为本次执行。

上述真实 hooks/组件/QueryClient 测试使用模拟网络；401/403 为原错误保留及拦截器静态核查，非真实登录续期验收。原生 Node 合同断言也不等于浏览器验收。未因此签署真实认证、数据库、Redis 多实例、支付、供应商或完整 15 类 E2E 通过，未扩大 MCP 审查范围。

用户已授权 review 后提交推送 dev、部署 zzapi 并切换 Default；截至本段登记，提交、推送、此次发布构建、部署和模板切换均尚未执行。dev 提交仅包含本轮经审查的修复，部署镜像另需保留线上 money 基线的 MCP overlay，二者不是同一份纯 dev 产物；冻结、摘要、构建与发布证据由[部署专题](16_issue401_dev_zzapi_deployment.md)维护。本轮是本仓库 Classic/Default 能力对照，不同步上游。

## 子项状态与证据（此前轮次）

- **#398 Logo**：独立 reviewer `ctx_84181ce7d92b` 确认两个 P2 与 fallback 收缩闭环；
  Default 13 项、Classic 14 项、仓库外边界 7 项通过，并核对五张 mock 截图。证据是
  组件/边界测试和本地 mock 视觉检查，不能推导真实认证、生产图片或跨模板保存验收。
- **#399/#400 令牌状态与活动**：原 reviewer `ctx_9682f8bd635f` 复审 Reset、
  名称和历史恢复两个 P2，通过原失败 1 项、实施 39 项及额外 `history.go`/批量 fallback
  3 项。F3 fixture 通过桌面单/多组、改名 alias 按 ID、保存刷新稳定提交；手机编辑器
  通过，列表没有分组字段不适用，自定义 tooltip 浮层未观察到。证据以组件、fixture 和
  API mock 为主，真实认证、Redis 多实例、数据库和完整 E2E 未验收。
- **#402 API 地址**：独立 reviewer `ctx_21255e4335ff` 确认 Default 10/10 通过；
  六种视口/主题 mock 浏览器证据通过。Classic 动态回归 3/3 通过，覆盖线路名称、原地址、
  描述和两级回退；不覆盖 Classic 复制、失败重试、异常形状、手机主题或生产环境。
- **F1 日志分组**：原 reviewer `ctx_608ae81b5cfc` 确认缓存失效和原型属性两项 P2 闭环；
  独立 6 文件 40 项耗时 46.21 秒通过，原失败 edge fixture 现 2 项通过（含 `__proto__`）。
  覆盖真实 GroupRatioForm/QueryClient 保存、活动观察者和普通用户 self 范围，但 GET/PUT
  网络为 mock。
- **F2 返佣入口**：独立 reviewer `ctx_cbdb5e0bca8c` 复跑 Default 10 项、Classic 3 项，
  全部通过且无 P1/P2。覆盖 role 10/100 路由与组件边界、AdminAuth 和提现事务调用；
  真实认证、生产角色权限、交易链和 MySQL/PostgreSQL 实库仍未验收。
- **F3 迁移矩阵**：已完成 24 个 Chromium/mock 代表场景，覆盖 B1–B6 的双模板日志、令牌、
  渠道、F1 分组、F2 角色和 F3 复验。Classic 即时提交问题另有下述追加修复证据；逐项保留真实认证、数据库、多实例、
  自定义 tooltip 视觉和完整 15 类 E2E 的未验收边界。

- **B3 追加修复**：Classic 的两个选择器改在 `onSelect` 同步策略及更新模式，避免退出动画前提交旧值；实现工人的真实组件 10/10、契约 8/8、Default 对照 27/27 通过，修后四组共 12 次浏览器 mock 请求通过（桌面鼠标、移动视口键盘、覆盖及追加），保留修前组件红测及 3/3 错误请求证据。独立审查 `ctx_108ffd07f56c` 已通过：正式组件 10/10、契约 8/8、额外组件 5 项及真实 Chromium/mock 补验 3/3 通过；另 1 项 JSDOM 动画等待未通过，保留日志并由浏览器覆盖对应边界，不计入组件通过数。未发现 P1/P2，真实后端仍未验收。

## 共享接口与安全边界

共享 API 仅在既有两组 GET 响应中增加可选 `group_names`；其余请求/响应、稳定 ID/code
提交、鉴权、计费、路由和分组匹配契约不变。没有数据库 schema 或迁移变更，也不读取完整
令牌密钥；显示名称只用于展示，不能替代业务标识或扩大授权范围。该兼容影响可分别供
`NewAPIForDouDi` 和 `NewAPIModifyByGang` 评估，未自动创建 Issue、PR 或发送消息。

逐项分享建议如下：

- **#398**：只有目标项目也存在 Logo URL 被当作图标名的缺口时值得参考，需按其图标加载器
  和真实图片策略单独复核。
- **#399/#400**：只有目标项目也有同一 Default 令牌页和本人活动摘要/服务端筛选需求时
  值得参考，不能直接复制 UI 或假设其 Redis 生命周期相同。
- **#402**：只有目标项目缺少 API 地址入口时值得参考，需核对其公开状态字段和组件体系。
- **F1/F3**：只有目标项目也需要稳定 ID、当前名称、历史 alias 和按用户作用域的分组选项
  合同才值得参考，显示名称不能替代授权标识。
- **F2**：只有目标项目把返佣业务错误挂在 Root 设置父路由时值得参考，不能照搬并放宽
  配置权限。

- **B3 追加修复**：NewAPIForDouDi 若也在 Semi 动画结束后的 `onChange` 更新提交状态，值得参考即时提交回归；NewAPIModifyByGang 需先确认相同受控选择器链路再考虑采纳。未核对两仓库源码，不自动发送消息。

## 历史构建与类型门禁（早于本次三项返修）

协调者在 `D:/Temp/newapi-issue401-final/results.json` 留存原六子项的门禁结果（早于 Classic 追加修复，原文件保留）：

- Default build：exit 0，10.61 秒。
- Classic build：exit 0，46.72 秒。
- Default typecheck：exit 0，12.24 秒。

构建产物写入独立 D 盘目录，没有覆盖原有 `web/dist` 或 `web/classic/dist`。Classic 没有
单独的 typecheck 结果，仅保留既有 chunk 体积警告；随后 `git diff --check` 通过。

追加修复后的 Classic 构建于本地时间 2026-11-16 05:44:44 至 05:45:18 执行，exit 0，进程耗时 33.51 秒（Vite 报告 32.39 秒），保留 chunk 体积警告。命令为 `bun run build --outDir D:/Temp/newapi-issue401-final/classic-dist-after-key-mode`；子进程 TEMP/TMP、完整日志及独立 `result.json` 位于 `D:/Temp/newapi-issue401-final/key-mode-build-ctxd41efb359a12/`，旧结果和原产物未覆盖。Default 源码未再变化，沿用上述 build/typecheck，不重复执行；追加独立审查已通过。

## 证据边界与后续限制

记录中的 Go/SQLite 结果仅以对应专题明确的真实本地夹具为准；原生 Node 源码断言不等于
浏览器组件验证；受控 Chromium/mock 网络不等于真实认证、后端权限或生产服务。MySQL、
PostgreSQL 实库、Redis 多实例、真实供应商/支付/Bot/OAuth、完整账号权限和完整 15 类
业务 E2E 均未在本轮签署通过。矩阵中的逐项未验收范围保持有效，不能因最终 build/typecheck
通过而缩小。

此前文档收尾阶段未发布或部署；当前已获 dev/zzapi/Default 授权但尚未执行，发布构建不得沿用旧快照结论。MCP 工作流及其未提交源码不属于本轮六个子项的审查结论，部署中须保留线上 money overlay。
