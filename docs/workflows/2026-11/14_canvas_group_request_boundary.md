# Canvas 分组参数与上游请求隔离

## 问题与范围

Canvas 通过 URL 的 `group` 参数选择授权分组。入口完成会话、来源和分组校验后，
已经把分组保存到可信上下文及临时令牌，但仍将它注入 JSON 或 multipart 正文。
OpenAI 兼容图片渠道重建表单或直接透传正文时，会把该内部字段一起发给上游。
用户实测 Gemini 图片渠道返回 `Gemini image channel does not support parameter group`。
本地代码可确认字段泄漏路径；尚未调用用户的真实渠道验证。

## 方案与契约

- 保留 URL `group` 的必填、权限校验、上下文和临时令牌设置。
- Canvas 入口改为从 JSON 和 multipart 正文移除 `group`，包括客户端重复提交的值。
- 继续沿用正文存储、大小限制、multipart 文件类型、文件名和分段数量校验。
- 同步请求和异步任务重放均使用清理后的正文；所选分组仍由可信上下文承担路由和计费。
- 渠道分发前，从可信 Canvas 上下文恢复内部模型选择对象的分组并重新校验权限，
  防止审核等待或异步重放期间撤权后继续调用；该对象不写回请求正文。
- 普通令牌 API 不经过该入口，不改变其请求契约。
- 不改变认证方式、数据库、图片结果及任务查询接口；无需数据迁移。
- Default 和 Classic 共用后端 Canvas 契约，无页面或配置变更。

## 验证计划

使用本机模拟上游，覆盖 JSON 同步/异步图片请求、双参考图编辑、转换及透传路径。
验证上游不再收到 `group`，其他参数和文件内容保留，任务、日志及路由上下文仍使用
已授权分组；保留非法文件、超限正文、来源与分组权限拒绝测试。
Go 单测设置 `-timeout 60s`，不调用生产 API，不执行发布或部署。

## 验证结果

- 修改前，双参考图异步编辑在透传开启与关闭时都因上游收到 `group=[default]` 而失败，
  复现字段泄漏，其他图片内容和计费断言未失败。
- 修改后，`go test ./controller -run '^TestCanvas' -count=1 -timeout 60s` 通过，
  测试输出汇总为 21 项通过。覆盖 JSON 同步/异步生图、双图编辑的透传开关组合、
  提示词审核重放、会话与分组限制、钱包和订阅结算，以及正文大小和上传安全边界。
- 双图测试使用前端实际发送的两个同名 `image` 文件字段；转换后的 `image[]`
  与直接透传的 `image` 均保留原图片字节和顺序，正文中重复提交的 `group` 全部移除。
- Markdown 格式化及 `git diff --check` 通过。未执行构建、发布、部署或生产 API 调用。
- 首轮审查后，进一步复审发现固定分组 JSON 请求在移除正文 `group` 后丢失分发前的权限复查。
  临时隔离测试确认撤销分组倍率后，带内部字段的原流程返回 403，过滤后曾继续选中渠道。
  已补回基于可信上下文的分组复查。新增 JSON/multipart × 撤销可用分组/倍率四个场景，
  均验证重放 403、任务失败、不选渠道、不请求上游、无消费日志且余额不变。
- 最终目标回归共 57 项通过：
  `go test ./controller ./middleware ./router -run 'TestCanvas|TestGetModelRequest|TestApplyRequestedGroup|TestPlayground.*Group|TestStandardChatCompletionsGroup|TestImageTaskBrowser' -count=1 -timeout 60s`。
- 真实 Gemini 渠道仍需更新服务后验收，不能把本地模拟上游通过视为线上已修复。
