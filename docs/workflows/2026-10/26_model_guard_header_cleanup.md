# 上游模型检测旧响应头路径清理

## 目标与范围

复核旧 PR #258 与现有上游模型检测功能的重叠。当前 faster-model 头仍在通用 HTTP 链路提前触发同一守卫，正文模型正常时也可能累计异常。按现有需求撤掉该头作为自动关渠依据，继续以响应正文声明的模型匹配规则。

## 实现与兼容

- 移除 faster-model 头采集、请求状态字段及新增消费日志字段，保留正文模型证据回调。
- 保留白名单、连续阈值、请求去重、人工禁用保护、通知及多节点数据库检查。
- 历史 detection_source 列、记录 API 字段及旧头部来源展示继续兼容，不删除历史记录、不恢复已关闭渠道、不变更配置。
- 两套模板使用同一后端守卫，本次无需修改页面或模块包。

## 验证计划

- 真实本地 HTTP JSON/SSE：正文正常而头异常不关渠；正文异常仍按规则关渠；仅有头部模型不检测。
- 阈值、去重、白名单、通知与历史来源字段回归；消费日志不再新增头部模型字段。
- 受影响 Go 测试使用 60 秒超时，go vet、格式与差异检查。

## 验证结果

- 先在清理前代码运行新增的真实 HTTP JSON/SSE 回归，复现正文匹配仍因头部异常关闭渠道；清理后相关回归通过。
- `go test -mod=readonly -timeout=60s ./service ./model ./relay/common ./relay/channel -run 'UpstreamModelGuard|UpstreamResponseModel|InitChannelMeta|GenerateTextOtherInfoIncludesUpstream|FormatUserLogs' -count=1`：四个包共 77 项通过。
- `go vet ./service ./model ./relay/common ./relay/channel` 通过；变更 Go 文件经 `gofmt` 格式化，`git diff --check` 通过。
- `go test -mod=readonly -timeout=60s ./relay/common ./relay/channel -count=1`：两个包完整回归共 240 项通过。
- 独立只读代码审查未发现阻断，独立重跑四包定向回归通过；已修正审查发现的索引历史表述。
- 五份修改 Markdown 的 `oxfmt --check` 通过，新增相对链接目标存在。
- 未调用真实上游或生产接口，未做 MySQL/PostgreSQL 实库验收；本次不修改数据库模型或迁移。

## 状态

本地实现、回归与独立代码审查完成，未合并、未部署。

## .357 主干合并前同步

- 同步基线 `69502cd4bdef0a78451c4b7b2b1f40d34dfcd4ae`，唯一冲突为开发索引，逐条保留 `.357` 发行、充值客户端 IP、Classic 提示词界面和本次清理记录。
- 模型守卫代码与同步前的 `b42dcb5e1` 一致，`model/log.go`、Classic 前端和 `VERSION` 与最新主干一致，不撤销 #357/#358。
- 最新基线的守卫与日志定向回归、`relay/common` 和 `relay/channel` 完整回归、`go vet`、五份 Markdown 格式及差异检查全部通过。
- 本轮仅合并源码，不改已发布 `.357` 标签，不触发新版本或生产部署。
