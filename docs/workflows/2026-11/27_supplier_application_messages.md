# 供货商申请对话

## 问题

供货商申请只能提交报价和内部备注。议价、优惠和测试额度没有可追溯记录，申请人也看不到管理员的回复。内部备注必须继续只对管理员可见。

## 方案

- 新增 `supplier_application_messages`，只追加。类型为普通消息、议价、优惠、测试额度。
- 议价保存分组名称、建议倍率和说明。优惠保存方案说明。测试额度保存数量、单位、说明，以及人工状态 `pending` / `issued`。
- 这些记录不增加余额、不生成兑换码、不创建渠道，也不修改分组或计费。
- 个人接口 `POST /api/supplier-applications/:id/messages` 只允许申请本人。管理接口 `POST /api/supplier-application-admin/:id/messages` 需要 `supplier_applications:write`。
- 明显的 API 密钥或接入凭据沿用现有模式拒绝。审计 `supplier_application.message` 只记标识、类型、角色和正文长度。
- Default 与 Classic 都在我的提交和管理详情展示时间顺序对话。无写权限的管理员不显示发送表单。Classic 详情窄屏使用 100% 宽度。

## 验证

- `go test -count=1 -timeout 60s ./router -run TestSupplierApplicationMessages`
- `bun run test src/features/supplier-applications`（在 `web/`）

未覆盖：MySQL/PostgreSQL 实库、Classic 组件测试、浏览器窄屏实播、生产站。

## 清理

测试使用隔离 SQLite，随测试进程结束。不保留临时库、临时进程或额外分支。
