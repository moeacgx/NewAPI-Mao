# 撤销 faster-model 响应头识别

日期：2026-09-21

## 目标与范围

按用户要求撤销 PR #247（`6d2f7bffdb5c742083fbce22ed512752e69d2c2c`）引入的 faster-model 识别代码。第三方中转站通常不提供该响应头，主程序无法据此稳定校验实际链路；本次恢复仅按响应正文声明模型进行校验。

基于 `origin/custom-main@59484e8a3`，提交前同步部署文档至 `eade57f2f`，工作分支 `revert/codex-faster-model-evidence`。撤销头部采集、证据回调及来源常量、头部异常关渠分支、管理员新增日志字段和通知来源字段；恢复原正文模型回调。保留原规则、渠道白名单、连续异常计数、并发去重及正文异常通知。

## 接口、数据与兼容性

- 检测入口仍为正式转发请求；渠道测试窗口仍显示原正文模型。Default、Classic 和外置模块包均无改动。
- `GET /api/extensions/upstream-model-guard/records` 恢复原字段集合，不再投影 `detection_source`。新消费日志不再写入 `other.admin_info.codex_faster_model`；通知恢复原 `reason`、`comparison` 和负载。
- 移除模型结构中的来源字段，不执行 DROP COLUMN，不清理历史日志、通知或检测记录。已升级数据库中可空的 `detection_source` 列作为额外列保留，旧值不回写，后续插入可继续使用原字段集合。
- 不修改生产规则、渠道状态或数据库，不自动恢复被禁用渠道。代码交付与线上部署分开；当前任务不发布或重启服务。
- 保留原实现及 .330 发布历史记录，并标注其能力已撤销，避免旧链接失效或误写历史。

## 验证计划

- 真实本地 HTTP → Responses JSON/SSE → 守卫验证：正文匹配但 faster-model 不匹配时不关渠；正文不匹配仍按原规则处理；仅头部有模型时不产生检测记录。
- 验证已有来源列和历史记录的数据库仍能读取、插入正文检测记录，且原历史值保持。
- 执行 `relay/common`、`relay/channel` 以及 `service/model` 守卫和日志回归；每组测试设置 60 秒超时。
- 执行受影响包静态检查、根模块构建、文档格式与链接检查、`git diff --check`，并复核原提交之后的改动均保留。

## 验证结果

- 新 HTTP 回归在撤销前按预期失败，撤销后 JSON/SSE 的 6 个场景全部通过，验证头部不触发关渠、不写管理员模型字段，正文异常仍产生检测记录。
- `go test ./relay/common ./relay/channel -count=1 -timeout 60s`：154 项顶层测试通过。
- `go test ./service ./model -run 'Test(UpstreamModelGuard|GenerateTextOtherInfoIncludesUpstream|FormatUserLogs)' -count=1 -timeout 60s`：43 项顶层测试通过，涵盖正文守卫、白名单、连续容错、人工禁用、通知队列及数据库回退。
- SQLite 实际执行旧额外列保留、AutoMigrate、历史记录读取及新正文异常插入；历史列值不变，新记录来源列为 NULL，接口恢复原字段集合。MySQL/PostgreSQL 未联网验证；未引入数据库专用操作或生产删列迁移。
- `go vet ./relay/common ./relay/channel ./service ./model` 与 `go build ./...` 通过。使用 Go 1.25.5、`GOARCH=amd64`、`GOWORK=off`；根构建使用忽略目录中的前端嵌入占位文件，不代表前端生产构建。
- Go 格式、修改 Markdown 的 Prettier、变更文档链接及 `git diff --check` 通过。Default、Classic、模块包及 relaykit 独立模块均未修改。
- 独立只读审查通过；生产代码与 PR #247 之前的对应文件一致，后续 xAI、项目身份及部署文档改动保留。
- 未发布、未部署，未修改任何线上配置、记录或渠道状态。共享契约影响限于撤回 #247 新增的记录来源、管理员日志和通知来源字段，原模型、认证、用量及计费接口保持。
