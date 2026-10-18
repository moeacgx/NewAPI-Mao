# Default 安全与兑换码上游集成

## 目标与范围

基于 `origin/custom-main@45ab82100`，按功能片段移植上游 `6f2333990` 的
Passkey 能力检测修复及 `524455fac` 的兑换码可选文件导出。仅覆盖 `web/src`
Default 模板；Classic 不在本次范围内。

## 方案与契约

- 浏览器提供 `PublicKeyCredential` 时允许进入 Passkey 流程，避免平台认证器不可用
  时误排除 USB/NFC 密钥和跨设备验证；不修改后端验证规则。
- 兑换码创建成功后使用现有响应 `string[]` 展示导出选择，默认不下载文件。
  可选 TXT 或 Markdown，允许包含名称和按当前站点货币展示的额度。
- 保留本地编辑加载 `loadOutcome`、金额转换、营销优惠码和
  `DELETE /api/redemption/batch` 的 `{deleted_ids, skipped}` 契约。
- 导出仅在浏览器处理本次返回的兑换码，不增加网络请求或持久化。
  TXT 规范化单元格内制表符和换行；Markdown 转义分隔符与标记字符。
- 暂不引入需要后端配套的任务插件、安全中心、多 RP ID Passkey 设置。

## 测试计划

先为外部认证器可用性和创建后导出流程补充失败测试，再实现修复。验证默认不下载、
TXT/Markdown 选项、特殊字符、创建成功金额展示、创建失败不弹出导出及既有编辑加载行为。
使用 Bun 运行受影响测试、lint、typecheck，并由集成分支完成全量测试与构建。

## 验证结果

- 已先运行回归测试并复现两项失败：外部认证器被误判不支持，以及创建成功后缺少导出入口。
- Bun 定向测试通过：Passkey 与完整兑换码功能域共 11 个文件、62 项测试；覆盖本地
  编辑加载、营销优惠码和批删，新增导出测试同时验证 `max_redeem_count` 原样发送。
- `bun run typecheck` 通过；修改的 5 个 TypeScript 文件定向 oxlint 通过。
  `passkey.ts` 原有 5 项 `prefer-string-replace-all` 错误按上游等价写法修正，后续
  Passkey 回归测试 3 项通过。
- 修改的 TypeScript 与本工作记录已用 oxfmt 格式化，保留版权头；`git diff --check` 通过。
- 全量 lint、test、build 及 locale 同步由主集成分支统一执行。本工作树未提交、推送或部署，
  未进行真实浏览器 WebAuthn 硬件验证。
