# 通知类型独立开关

## 目标

将余额提醒、全渠道测试通知和上游模型更新通知分开控制，同时保持旧客户端和历史用户设置兼容。

## 方案与边界

- 站点新增 `QuotaRemindEnabled`，默认开启；`QuotaRemindThreshold` 继续作为默认阈值。
- 用户设置新增 `quota_warning_enabled` 和管理员专用 `channel_notify_enabled`；渠道状态自动禁用/恢复继续由通知中心 Telegram 任务控制，不读取该个人开关。
- 钱包和有限额订阅在本次结算已知的剩余额度低于阈值时，都可产生提醒候选，继续由既有用户通知频控限制发送次数；不依赖不可靠的请求旧快照判断唯一跨线。
- 并发请求或批量延迟写入可能使本次快照尚未反映合并扣减；后续消费看到低余额时仍能提醒。没有新增批次刷新触发器，不保证最后一次扣费立即触发或恰好投递一次。
- 充值回到阈值以上后不再产生低余额提醒候选；后续再次低于阈值时仍受原频控窗口约束，充值不会清空频控。
- 历史设置未保存渠道开关时，Root 延续接收全渠道测试完成通知，其他管理员默认不接收。
- 管理员通知继续使用业务调用方传入的标题和正文，个人语言设置不会自动翻译投递内容。
- 认证验证码、密码重置、邮箱绑定和账户安全邮件不受通知开关影响。
- 只修改 Default 前端；Classic 通知设置不在本次范围内。

## 验证

- 回归覆盖多个旧快照未反映合并扣减、后续已低于阈值的余额、站点/个人关闭、有限额订阅与既有发送频控；使用显式余额和 HTTP 通知捕获，不使用 sleep。
- 通过：`go test ./service -run 'Test(WalletQuotaReminder|QuotaReminder|SubscriptionQuotaReminder|ChannelTestNotification|RenderNotification|DispatchChannelNotification)' -count=1 -timeout=60s`（1.279 秒）。
- `go test ./service ./controller -count=1 -timeout=60s` 中 controller 全量通过（9.500 秒）；service 全量因累计 SQLite 建表耗时在 `TestSecurityAuditEventSnapshotsAutoAndInheritedTokenGroups` 处达到 60 秒上限，不能记为全量通过。
- Default `tsgo -b`、修改文件的 oxlint/oxfmt 检查、通知页 2 项测试通过；个人设置相邻测试另有 16 项通过。未运行生产构建或真实浏览器验收。
- 七国语言均通过同步，缺失和多余键均为 0；新增文案均已翻译，修复原简中问号值。同步报告仍有其他既存未翻译项，本次不扩展处理。
- 本机 Bun 不在 PATH 且常见安装路径未发现可执行文件，使用 Node 执行 `package.json` 指向的同一 `scripts/sync-i18n.mjs`，类型检查、lint、测试和格式化均直接调用仓库现有依赖。没有安装或升级依赖。
- Markdown 已格式化，本次新增文档链接存在性与 `git diff --check` 通过。README 全量链接检查发现既存失效链接 `../workflows/2026-08/15_topup_payment_fee_included_hint.md`，未纳入本次修复范围。改动位于本地独占工作树，未推送、部署或调用生产 API。
