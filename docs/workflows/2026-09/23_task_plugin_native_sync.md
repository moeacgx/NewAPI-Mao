# 官方任务插件原生同步链路对齐

## 目标与基线

复用官方宿主自定义原生路由与同步结果呈现能力，使官方 TypeSafe Jev 插件可被调用。
用户要求优先同步上游实现，避免另建专用 Jev 控制器或第二套插件框架。

- 本地：`origin/custom-main@b2370db118cfd36da58b1074ba62cecdd4c01c13`。
- 固定官方：`QuantumNous/new-api@d04c118c8803f49e0c9bab74dcf5b5efeab9464a`。
- 插件来源：`QuantumNous/new-api-plugins/plugins/tasks/typesafe/1.0.0/plugin.js`。

历史提交 `583755592` 有意仅开放通用 JSON 任务、显式按次计费和认证内联资源，
并非官方没有原生路由能力。运行时能力声明不代表宿主路由已接线。

## 范围与方案

复用官方 `router/plugin-router.go` 的路由代际、冲突检查和 NoRoute 分发，
`middleware/task_plugin.go` 的 native decode，以及 `controller/relay.go` 的
同步任务视图与 native render 契约。按真实依赖接入用量计费，不为 Jev 创建专有 Go 路由。
官方 `controller/plugin_protocol.go` 是 Responses/图片标准协议桥，不因 Jev 需求全量迁移。

本地仅保留必要边界适配：AtlasCloud=61、Task Plugin=62、Root 版本管理、默认关闭、
渠道绑定与分组/用户/模型权限、源码 hash 历史 pin、原有资金来源和日志脱敏。
两套前端页面不改动；新路由属于后端插件能力。

## 协议与安全

官方 TypeSafe 插件公开入口为 `/typesafe/v1/systemone`，供应商端点为 `/v1/systemone`。
保留插件声明路径，客户端按安装的插件元数据调用。请求为 `model/state/questions`，
同步返回 `model/answers/usage`。不以任务 ID 代替结构化结果。
同步成功、错误退款与用量经过原有鉴权、渠道分配和资金链路。
插件激活、停用与路由代际保持一致，未激活版本不得意外开放。

## 验证计划与状态

回放官方插件编译与 Hook 合同；模拟上游覆盖混合题型、同步输出、认证与绑定渠道隔离、
总开关/版本禁用、预扣和实际用量结算、上游错误退款、不重复扣费、路由冲突及原生渠道回归。
Go 测试单次超时 60 秒；涉及 relaykit 时独立执行 `GOWORK=off go build ./...`。

不调用生产或付费供应商 API，不部署；真实 TypeSafe 与真实 MySQL/PostgreSQL 验收另行记录。

## 实现边界

- 仅补齐 Jev 所需的原生 JSON submit；原生 query/dynamic、multipart、decoder 源任务 intent
  尚未开放，不能将此次交付称为完整官方插件系统全量同步。
- 同步官方 `retainResult` 和 `upstreams` 元数据，原样 TypeSafe 插件保持作者、模型与端点。
  允许声明 `new_api` 不代表本地开放官方 60 类渠道；当前仍使用绑定目标 key 的 62 类供应商渠道。
- 路由代际在启动与本节点插件管理变更时刷新；多实例中其他节点需要刷新或重启。
  命中的旧代际仍检查数据库活动版本与总开关，不会绕过已撤销状态。
- 先提交数据库管理变更，再刷新路由。路由冲突错误不回滚已提交的 active/enabled 状态，
  管理审计仍保留；管理员需修复冲突或重新激活兼容版本。
- 原生同步成功保留任务账务记录；不保留结果的路由不存答案与插件状态，查询返回 404。
  官方 `retainResult:false` 只对即时终态生效，按次异步任务继续保留轮询所需数据。
- 同步上游持久化前补足预留及 durable 边界。持久化后结算故障返回 500、保留预扣等待核对，
  不进行全额自动退款；本次没有新增跨系统账务恢复队列。
- Default 与 Classic 页面不在修改范围。管理员通过既有表达式原始配置设置用量价格，
  本次没有迁移官方插件专属价格编辑器。API、模型 ID、Bearer 凭据语义与上游保持一致。

## 插件文件

官方插件库固定提交 `b42cc99a6bd1998d0cc1581270bd46798ce00ad6`，文件
`plugins/tasks/typesafe/1.0.0/plugin.js`，SHA-256 为
`80585e402c8e6709f976e6be1f0308a95d3b991370383ab984d21fe968d8e912`。
测试中原样保存在 `router/testdata/typesafe-1.0.0.js`，保留 Apache-2.0 许可证。

客户端示例：

```sh
curl https://YOUR_GATEWAY/typesafe/v1/systemone \
  -H 'Authorization: Bearer YOUR_GATEWAY_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"model":"jev-1.13.0","state":"重复扣款","questions":{"urgent":{"type":"noul","instructions":"是否需要优先处理","criteria":{"true":"涉及资金损失"}}}}'
```

上游地址为 `https://api.typesafe.ai`，渠道密钥使用 TypeSafe Key。
以文档中的 $0.042/百万输入 token 为例，管理员可以设置任务表达式
`u("input_tokens") * 0.042 / 1000000`，并选择 `tiered_expr`。
不自动设置价格；不把 Cloudflare Workers AI 地址或令牌用于此原生插件。

## 已执行验证

- `go test -mod=readonly -timeout=60s ./pkg/jsplugin ./pkg/billingexpr ./setting/billing_setting ./middleware -count=1`：通过，覆盖运行时、原生门禁、元数据和原有聊天计费回归。
- router、controller、relay、service、model 的插件定向回归：通过；包含官方分发器的路由冲突、代际固定、405、未命中响应隔离和原有任务授权/退款。
- `TestJevNativeIntegration`：使用真实 TokenAuth、SQLite 与本地 HTTP 模拟上游，6 个场景通过。
  高优先级的错误插件渠道及 AtlasCloud 不会被选中；noul 零值、Choice、结构化 Score legend 原样返回。
  测试售价 $0.042/百万输入 token、分组倍率 1 时，1000 输入 token 最终扣 21 quota；钱包与令牌一致。
- 上游 502：异步退回预扣，不保存成功任务且不泄露供应商错误正文。
- 持久化后结算故障注入：返回 500，等待退款线程池结束后仍保留 1376 quota 预扣，未误全退。
  这是待对账状态，不表示具备自动恢复队列。
- `retainResult:false`：任务账务/版本归因保留，响应、插件状态与渠道密钥不落任务记录，公开查询和资源接口为 404。
- 原生插件 CORS：允许无 Cookie 的 Bearer 客户端预检，不改变管理 API 的严格 CORS 分类。
- `go vet -mod=readonly ./router ./controller ./relay ./service ./pkg/jsplugin ./pkg/billingexpr ./middleware ./setting/billing_setting`：通过。
- Markdown 使用 `oxfmt@0.57.0`；检查新增文档链接、示例 JSON、插件原始字节 hash 与 `git diff --check`。

没有修改 `relaykit` 或其公共 API，未重跑其独立构建；未构建完整前端或应用二进制。
未提交、推送、合并、发布或部署，未访问真实供应商；多实例、真实 MySQL/PostgreSQL、订阅与生产故障恢复仍需单独验收。
