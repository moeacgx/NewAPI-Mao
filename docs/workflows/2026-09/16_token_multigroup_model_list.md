# 多分组令牌模型列表回归

## 问题与目标

用户反馈令牌绑定多个分组后获取模型列表失败。修复前 TokenAuth 已解析并授权显式分组列表，
但 `controller/model.go` 的模型列表分组解析将逗号连接的 `token.Group` 当成一个分组。
例如绑定 `vip,default` 时会查询名称为 `vip,default` 的能力，而非两个分组的模型并集。
真实注册路由和测试令牌已复现：HTTP 200，OpenAI 返回 `success=true,data=[]`，
Anthropic 返回空 `data`，Gemini 返回空 `models`。单分组、继承和自定义 auto 路径正常。
因此客户端可能显示“模型列表获取失败”，根因是显式多分组未被模型列表展开。

## 范围与契约

- 模型列表读取鉴权上下文已经解析的显式分组，按绑定顺序合并启用模型并去重。
- 保留继承用户分组、单分组、自动分组和模型白名单、定价过滤。
- TokenAuth 原有权限、弃用分组和独立分组冲突拒绝规则保持，不扩大可见模型范围。
- OpenAI、Anthropic 与 Gemini 模型列表共用此解析，因此分别回归响应结构。
- 不修改数据库结构、令牌绑定、实际选渠或计费；两套前端无需更改保存格式。

## 验证计划

使用隔离 SQLite、假令牌、真实 TokenAuth 和 SetRelayRouter，验证多分组并集、去重、白名单、
禁用能力、未绑定分组不可见、单分组和继承兼容、分组撤权拒绝；测试超时设置为 60 秒。
执行相关 controller/router/middleware/service 测试、vet 和差异检查。
本记录不代表已使用线上令牌复现，也不包含发布或部署。

## 验证结果

基于 `custom-main@6fcf4bbb746bfb06cc52a7ca701cfae4adfc9abd`。
修复仅在 `getModelListGroups` 使用 `service.GetRequestTokenGroups` 读取 TokenAuth 已校验的
分组数组，复用现有模型并集与去重逻辑，未增加绕过鉴权的字符串拆分路径。

- `TestModelListExplicitTokenGroups`：修复前 6 个多分组子项因空列表失败，修复后 10 个子项全部通过。
  使用真实 `SetRelayRouter` 与 `TokenAuth`，覆盖 Group ID 绑定、旧逗号令牌、OpenAI/Anthropic/Gemini
  和 Gemini OpenAI 兼容路径、模型限制、禁用能力、未定价及未绑定模型过滤、单组/继承/auto 和撤权 403。
- `go test -mod=readonly ./router ./controller -run 'Test(ModelList|ListModels|GetUserModels)' -timeout 60s -count=1` 通过。
- `go test -mod=readonly ./middleware ./service -run 'Test(SetupContextForToken|ParseTokenGroupList|GetRequestTokenGroups|.*TokenAutoGroups|.*ExplicitGroup|.*GroupBinding)' -timeout 60s -count=1` 通过。
- `go test -mod=readonly ./router -timeout 60s -count=1` 全包通过。
- `go vet -mod=readonly ./router ./controller` 通过；根应用 `go build -mod=readonly` 通过，
  使用工作树现有真实双模板前端产物，未为本次后端修复重新构建前端。
- Go 格式、Markdown 格式、开发索引链接和 `git diff --check` 通过。
- 独立只读审查确认未扩大权限；按审查意见补全测试结束后的 `model.DB/LOG_DB` 指针恢复，
  随后完整 router 测试及相关 vet 再次通过。

没有数据库迁移或新的配置。未修改 `relaykit` 公共接口；Default/Classic 保存契约不变，
本次不涉及前端组件。外部客户端的模型发现行为恢复为令牌已授权分组的模型并集；
不改变实际转发、模型 ID、价格或额度。后台令牌编辑器通过 UserAuth 获取 `/api/user/models`
候选模型，属于另一条入口，不将本次 TokenAuth 复现等同于编辑器故障。
未使用线上令牌做现场复现，也未合并发布或部署。
