# 通知类型独立开关

## 目标

将余额提醒、渠道状态通知和上游模型更新通知分开控制，同时保持旧客户端和历史用户设置兼容。

## 方案与边界

- 站点新增 `QuotaRemindEnabled`，默认开启；`QuotaRemindThreshold` 继续作为默认阈值。
- 用户设置新增 `quota_warning_enabled` 和管理员专用 `channel_notify_enabled`。
- 余额提醒只在余额跨过阈值时发送一次，充值回到阈值以上后重新触发。
- 历史设置未保存渠道开关时，Root 延续接收渠道通知，其他管理员默认不接收。
- 认证验证码、密码重置、邮箱绑定和账户安全邮件不受通知开关影响。
- 只修改 Default 前端；Classic 通知设置不在本次范围内。

## 验证

- `go test ./service ./controller ./model`
- `git diff --check`
- Default 前端需执行 `bun run typecheck`、受影响测试和 lint。
