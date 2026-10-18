# 任务插件性能失败过滤钩子

## 目标与范围

使用日志照常保留。宿主提供通用可选 shouldRecordPerformanceFailure(ctx, failure) 钩子，
允许任务插件仅排除模型广场的失败样本；不改变 HTTP 响应、计费、退款、渠道指标和成功样本。
移除先前未合并的 Cloudflare Jev 特定日志抑制逻辑，不硬编码供应商或模型名。
截图中错用 /v1/responses 的请求在首次分发时终止，本来不进入性能采样，也不执行任何插件钩子。
对此只验证“保留日志、不产生性能样本”，不做候选扫描或把插件挂载到其他协议入口。

## 接口和兼容

导出钩子需声明 requiredCapabilities: ["task-performance-filter@1"]；声明能力却缺少函数、
导出非函数或没有声明能力均拒绝安装。旧插件不声明能力且不导出钩子时保持原行为。
依赖能力的新插件会被不支持该能力的旧宿主拒绝，避免静默忽略。

ctx 仅含 pluginKey、pluginVersion、model、upstreamModel、requestPath、method；
模型保持客户端别名与映射后的上游模型分离，路径无 query，不传 key、headers、body、用户信息。
failure 仅含 stage、httpStatus、errorCode，不传错误正文。
stage 为 transport、http、parse、immediate。httpStatus 为宿主用于该失败的状态，
HTTP失败为供应商响应状态，解析/即时业务失败通常为 502，不能一律解释为上游响应码。

返回 false 表示排除该失败样本；true/null/undefined 表示继续宿主原过滤。
其他返回值、异常、超时都继续原统计，稳定日志仅记录失败原因类别，不输出钩子异常正文。
钩子只允许降低采样范围，不能将失败变成成功或绕过既有策略/管理员过滤。
错误码只接受128字节以内的ASCII标识符，其他值归为 invalid_error_code；不把异常正文当作错误码。
钩子必须只依赖参数，不依赖不同JS运行实例间的全局变量。

## 生命周期与资源边界

仅原生同步 retainResult:false、实际已发上游、非测试、非客户端取消的失败调用钩子，
复用请求开始时固定的插件对象，绝不临时按模型选择插件或使用后来激活的版本。
钩子排队加执行最多 100ms，沿用 JS 引擎并发限制；执行结束再次检查客户端取消。
成功、本地预扣/鉴权/持久化/结算失败、pending 和异步轮询不新增调用或样本。
这里“日志保留”意为保持原有日志边界，不补造此前未写入的日志。

## 验证计划

先验证显式false仍被统计的红灯，再覆盖false、true、空值、非法返回、异常、无限循环、
实际上下文白名单、固定版本、HTTP/parse/immediate、成功用量和退款、日志不消失。
独立编译验证能力声明，真实 Gin 原生请求验证样本与日志及余额；
分发回归验证错路径日志存在而性能摘要不变。相关Go测试使用60秒超时。
仅宿主实现和文档/示例，不自动发布、升级或安装插件；两套前端共用后端性能API，不改界面。

## 已完成验证

JS引擎覆盖能力与导出校验、旧插件、返回值矩阵、异常、无限循环终止及引擎后续可用，
实际占满并发槽的排队超时、输入字段白名单、错误码清洗和注册表版本替换。
真实 Gin 原生路由先复现钩子返回false仍计样本的红灯，接线后验证HTTP、解析、传输和即时业务失败、
原有错误日志保留、余额/令牌退款、成功486/70 Token及处理中固定版本、钩子触发客户端取消。
测试插件key为 performance-gate，与Jev无关，证明没有供应商硬编码。
另有16种分发场景在缓存开关两种模式执行，错路径日志保留、无性能样本、余额不变。
错误日志“保留”按既有边界验收；即时业务FAILURE原本不落使用错误日志，不在本次补造。
最终复审复现 Boolean 对象被通用引擎解包成 false 的问题；增加专用严格布尔调用入口，
在导出前拒绝所有对象，覆盖 boxed false/true 和带 getter 的非法对象，其他钩子调用行为保持原样。

已通过：

- `go test -mod=readonly -timeout=60s ./pkg/jsplugin ./controller ./middleware ./model ./router ./service ./pkg/perf_metrics -count=1`。
- `go vet -mod=readonly ./pkg/jsplugin ./controller ./middleware ./model ./router ./service ./pkg/perf_metrics`。
- oxfmt@0.57.0 格式化、选定文档链接及 git diff --check。

未修改 relaykit 公共API、前端界面或插件发布仓库。新增能力为本发行版宿主扩展，
未来依赖此能力的插件应带 requiredCapabilities 声明；不能据API版本1判断所有宿主均支持。
