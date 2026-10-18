# 渠道强制使用 Responses 上游

## 目标与范围

Default 与 Classic 的渠道编辑「额外设置」提供默认关闭的「强制使用 Responses 请求上游」开关。
用于上游只接受 Responses、客户端仍使用 Chat Completions、Claude Messages 或 Gemini 对话接口的渠道。

## 配置与接口

复用渠道创建、更新接口中的 `setting` JSON 字符串，新增布尔字段 `force_responses`，缺失等同于 `false`。
例如：`{"force_responses":true}`。沿用渠道敏感写权限，不增加接口、数据库列或迁移。

支持 OpenAI（1）、Azure（3）、xAI（48）、Codex（57）、Sub2API（59）和 NewAPI（60）渠道；
其他渠道保存开启配置时拒绝，避免原生适配器把请求再次转换为其他上游协议。
上游地址、密钥和模型仍须实际支持 Responses；此开关不提供上游能力探测。

## 转换及优先级

- Chat Completions、Claude Messages、Gemini generateContent/streamGenerateContent 转为 Responses 请求。
- 客户端继续接收原协议的 JSON 或 SSE；沿用现有转换器、usage 归一化及结算路径。
- 原生 Responses 继续使用 Responses；`responses_to_chat_enabled` 不再把它转回 Chat。
- 强制开关优先于全局与渠道请求体透传，不要求全局 Chat→Responses 策略或模型正则匹配。
- 关闭开关后恢复原全局策略及透传语义。强制转换仍应用模型映射、系统提示词、参数覆盖和禁用字段过滤。
- Chat/Claude/Gemini 的桥接沿用现有 Chat 字段参数覆盖规则，先覆盖再转 Responses；原生 Responses 仍按 Responses 字段规则覆盖。
- 上游路径使用适配器原有 Responses 地址规则：普通渠道为 `/v1/responses`，Azure/Codex 保留专用路径。
- 图片、音频、向量、重排、Realtime 和 Responses compact 不新增协议转换。
- 渠道测试按钮的对话探测同样走 Responses，图片、向量与 compact 探测保留原端点。
- 转换或上游报错走既有错误与重试流程，不静默退回 Chat。
- 换渠重试会重新记录本次协议转换链；上游 SSE 错误不会改变客户端原本的流式标志。

## 兼容性与验证计划

旧渠道不受影响；回滚时关闭开关即可，旧版本会忽略未知字段。
跨项目只影响显式开启渠道的上游对话协议，客户端鉴权、模型标识及返回协议不变。

验证覆盖配置保存/回显/关闭、敏感权限、三种客户端协议的上游路径与请求体、透传与反向转换优先级、
同步/流式返回及 usage、未开启与非对话接口回归、relaykit 独立构建、两套前端检查。
真实供应商请求和线上部署需在单独发布任务中验证。

## 验证结果

- 后端定向回归通过：配置保存与校验、Chat/Claude/Gemini/Responses 真实入口的上游请求、三协议 JSON/SSE/缓冲返回及 usage、渠道测试按钮和换渠协议清理。
- `relay/common` 与 `relay/channel/openai` 全包测试通过；后端测试均设置 `-timeout 60s`。
- `relaykit` 已使用 `GOWORK=off go build ./...` 验证独立构建。
- Default 类型检查、4 项开关交互回归、受影响文件 lint 与生产构建通过。
- Classic 的 4 项开关交互回归、受影响文件 lint 与生产构建通过。类型切换测试沿用现有 Semi 测试方式，显式补发 jsdom 缺失的 CSS 动画结束事件。
- 已有渠道相关回归通过：Default 的表单契约、NewAPI 渠道和密钥策略共 17 项；Classic 的密钥策略及插件绑定共 6 项。
- 根模块构建通过。本地预览的两套页面与独立 SQLite 后端返回正常，首次访问需要初始化；浏览器工具受认证限制，未完成真实浏览器页面检查。
- 本地验证使用模拟上游和独立 SQLite 数据，不包含真实供应商兼容性、生产请求或部署验收。
