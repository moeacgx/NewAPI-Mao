# 官方插件模式与本地宿主边界

## 固定来源，按需更新

本次核对日期为 2026-09-24。官方插件仓库固定提交
`b42cc99a6bd1998d0cc1581270bd46798ce00ad6`；下表路径均相对此仓库。
这些是已核对的参考点，不是永远有效的最新版本。后续开发先用 `gh` 获取实际 HEAD，
固定提交后读源码，不凭记忆选择版本；不要自动拉取或改写工作中的用户分支。

- [官方 API v1](https://github.com/QuantumNous/new-api-plugins/blob/b42cc99a6bd1998d0cc1581270bd46798ce00ad6/docs/plugin-api/v1.md)
- [官方发布规范](https://github.com/QuantumNous/new-api-plugins/blob/b42cc99a6bd1998d0cc1581270bd46798ce00ad6/AGENTS.md)
- [官方插件目录](https://github.com/QuantumNous/new-api-plugins/tree/b42cc99a6bd1998d0cc1581270bd46798ce00ad6/plugins/tasks)

需要本地源码时，先查已有克隆；没有则用 `gh repo clone QuantumNous/new-api-plugins <独立参考目录>`。
查看 `index.json` 的 latest 后读取对应源码，不向参考克隆写入测试或变更。

## 选择接近目标生命周期的例子

| 官方路径                                  | 优先阅读的符号                                                                          | 可以复用的思路                                                                | 不能机械复制的部分                                                                      |
| ----------------------------------------- | --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `plugins/tasks/typesafe/1.0.0/plugin.js`  | `native`、`parseSubmitResponse`、`extractUsage`、`extractUsageOnComplete`               | 同步决策、公开 ID、`retainResult:false`、输入预扣与实际覆盖                   | 原生前缀、65536 预算、仅上报输入；不能据此认定输出免费就不统计                          |
| `plugins/tasks/alibaba/1.4.1/plugin.js`   | `usageProfiles`、`convertImage`、`parseSubmitResponse`、三个 `extractUsage*`            | 同一规范化参数服务于请求与用量；图片/视频模型 profile；估算和实际结果分别取数 | 部分接口同步、部分异步；`billing_ratios` 与 facts 不同；SSE 和图片协议需宿主独立支持    |
| `plugins/tasks/doubao/1.2.0/plugin.js`    | `decodeNativeSubmit`、`parseSubmitResponse`、`extractUsageOnComplete`                   | 图片即时返回、视频轮询；源任务先授权；模型相关用量                            | 视频计费 tokens 不能凭名称拆成输入/输出；图片按尺寸/数量而非 Token                      |
| `plugins/tasks/sora/1.1.0/plugin.js`      | `buildSubmitRequest`、`parseTaskResult`、`extractUsage*`、`buildContentRequest`         | 单任务轮询、实际秒数覆盖预估、宿主认证下载                                    | 秒数与尺寸不是 Token；下载需要渠道认证，不能给任意外站附 Bearer                         |
| `plugins/tasks/kling/1.1.0/plugin.js`     | `tokenFor`、`viaGateway`、`extractUsage*`                                               | 供应商认证与网关转接不同；按上游模型选能力；最终资源单位覆盖                  | `units/final_unit_deduction` 可为小数，是资源包单位，不是 Token；旧密钥前缀推断不能泛化 |
| `plugins/tasks/sunoapi/1.1.0/plugin.js`   | `buildBatchQueryRequest`、`parseBatchResult`、`extractUsageOnComplete`、`listArtifacts` | 批量 ID 关联、按实际 clips 数量结算、稳定资源 key                             | 音乐/歌词预估数量不同；只验证 batch 主路径不能证明 per-task 后备正确                    |
| `plugins/tasks/google/1.0.2/plugin.js`    | `parseSubmitResponse`、`parseTaskResult`、`buildContentRequest`                         | operation `done` 缺省表示运行中，无法识别对象表示 UNKNOWN；支持即时结果       | 完成 hook 返回 null 是该协议的用量策略；`x-goog-api-key` 和资源策略不是通用模板         |
| `plugins/tasks/vertex-ai/1.0.2/plugin.js` | `meta.auth`、`buildSubmitRequest`、`buildQueryRequest`、`extractUsage`                  | 使用宿主 OAuth 凭据，轮询还原项目/区域；保留 `generateAudio:false`            | OAuth 授权、区域路由及音频价格档位不适用于其他供应商                                    |

上表是源码归纳，不是给所有供应商添加所有能力的清单。
官方示例也要验证：例如固定版本 Suno 的 per-task 后备给 batch parser 传 `code:200`，
但 batch parser 要求字符串 `success`。这说明需要测试实际调用路径，不能据静态后备问题
断言其线上 batch 主路径失败，也不能原样复制问题分支作为新插件模板。

## 本地能力以代码为准

本次宿主基线为 `b6476c6c8c6fcc5c3b7725545fb3b16ba117a625`（custom-main，.335 时期）。
本地 `docs/developer/official-task-plugins.md` 明确记录了原生 JSON submit 和同步计费范围；
query/dynamic、部分协议外观、远程资源和异步 UsageFacts 等有未开放或未验收边界。
官方 API 文档不能代替这些本地证据。

逐能力分别记录三个状态：编译器是否接受、实际入口是否开放、完整组合是否有验收证据。
明确返回 501 是实现未开放，缺少集成测试是尚未验收，两者不能合写成“功能已支持”。

该基线的编译器已识别 query/dynamic 和 usageProfiles，但原生非 submit 入口在中间件返回 501。
因此安装校验通过与功能可用是两道门；`plugin source failed validation` 仍需原字节离线编译定位，
不能用运行时尚未接线来解释所有安装失败。

从仓库根目录检索以下入口；行号可能随版本变化，使用符号定位：

| 层           | 本地入口                                                                   | 核对重点                                                                           |
| ------------ | -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| 编译与元数据 | `pkg/jsplugin/engine.go`、`registry.go`、`routing.go`                      | 严格字段校验、模型大小写折叠冲突、路由冲突与能力声明                               |
| 解码与选渠   | `router/plugin-router.go`、`middleware` 下插件相关文件                     | decoder 模型先验、TokenAuth、type=62/key 隔离、别名与价格身份                      |
| 适配器       | `relay/channel/task/jsplugin/adaptor.go`                                   | `ParseResponse`、query context、hook 参数顺序、facts 与 ratios、原始数值与饱和数值 |
| 预扣与结算   | `relay/relay_task.go`、`service/task_billing.go`                           | `RelayTaskSubmit`、冻结表达式、`EvaluateTaskCompletionUsage`、最终金额与退款       |
| 交付与持久化 | `controller/task_plugin_relay.go`、`controller/relay.go`                   | renderer、持久化边界、retainResult、同步终态、`LogTaskConsumption` 调用            |
| 统计         | `model/log.go`、`service/channel_metrics_lifecycle.go`、`model/usedata.go` | 两个 Token 列、统计导出、渠道指标、32 位累加边界                                   |
| 安装与发布   | `service/task_plugin_runtime.go`、`cmd/task-plugin-index/main.go`          | 活动版本、历史 hash pin、多节点运行时、源码与索引一致                              |

2026-09-24 创建技能时，Token 日志补齐仍在主程序 PR #270，Cloudflare 完整用量在插件 PR #7。
不要假定任意 checkout 已有 `ActualTokenUsage`，也不要将“PR 已通过”写成“线上已更新”。
使用前核对分支、合并提交、发布资产和目标节点版本。

## 发布仓库的差异

- 官方目录为 `plugins/tasks/<key>/<version>/plugin.js`，索引由 `tools/pluginindex` 生成；
  必须附英文 `CHANGELOG.md`，可另加译文。官方仓库禁止新增插件测试，回放放宿主或外部临时目录。
- 本地独立插件仓库目录为 `published/<key>/<version>/plugin.js`，版本说明与验证入口以其 README 为准。
  现有 `tests/` 和 `scripts/verify-cloudflare-host.mjs` 可复用，不把官方禁止测试规则搬来删除它们。
- 本地索引含 `retired` 历史记录。旧宿主生成器不能保留该扩展，不可直接覆盖正式索引。
  先在隔离副本生成、对照并保留这些记录，按插件仓库既有校验验证；不能只看生成命令成功。
- 根仓库子模块 gitlink 与插件市场 main 是两条独立分发路径。只改 gitlink 不会上架，
  上架也不代表站点已安装或激活。相同 key/version 的源码和 hash 不可覆盖。
