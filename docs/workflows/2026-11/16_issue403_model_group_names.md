# Issue #403：模型管理可用分组显示名称

## 目标与根因

Default 模型管理 `models-columns.tsx` 和 Classic 模型管理
`ModelsColumnDefs.jsx` 直接展示 `enable_groups` 的内部编码。管理接口
`/api/models/`、`/api/models/search`、`/api/models/:id` 尚未提供独立名称字段。

## 方案与契约

- 在模型响应增加 `enable_group_names`，类型为编码到当前名称的映射，标记
  `gorm:"-"`，不增加数据库列或迁移。
- 保留 `enable_groups` 的顺序、编码、筛选与提交契约。模型响应没有分组 ID，
  因此批量复用 `GetGroupDisplayNameMap` 的当前 code、历史 alias 解析规则。
- 每次管理响应只加载一次名称映射；每条模型仅附带其自身 `enable_groups`
  的映射，不将全站分组名称附加到响应中。解析失败、删除或未知标识回退原编码。
- 两套模型管理表格使用名称显示，颜色与业务值继续使用编码；原型属性名必须
  按自有属性处理，避免缺失映射时读到 JavaScript 原型。
- 管理接口保持 `AdminAuth`；不修改公开 `/api/pricing`、价格权限、计费或分组匹配。
- Classic 公开价格表已调用 `getGroupDisplayName`，详情已传递 `groupNames`
  至 `GroupPillList`；本轮保留其现有实现。

## 范围与兼容性

独占修改 `controller/model_meta.go`、`model/model_meta.go`、对应后端测试、
Default 模型类型/表格及专属测试、Classic 模型表格及专属测试。
`controller/model.go` 的 MCP 分页与模型列已有 Logo URL 修复保持原样。
新增可选响应字段对旧客户端兼容，旧后端缺失名称映射时前端回退原编码。
README 索引由协调者统一登记。无新增用户文案，无需翻译。

## 验证计划

- 先运行失败回归，再验证后端管理列表、搜索、详情名称响应与两套真实表格渲染。
- 使用名称不同于编码的夹具，覆盖改名、历史 alias、删除/未知、原型属性名与
  映射不包含无关分组；验证原有编码提交契约。
- 每项测试限时 60 秒，TEMP/TMP/GOTMPDIR 使用 D 盘，前端单 worker。
- 执行 Default 类型检查、定向 lint、格式与差异检查；不全量构建，不访问生产。

## 验证结果

实现已完成，定向回归通过；独立审查、冻结后统一类型检查与构建由协调者安排。

所有测试通过 `D:/Temp/issue403-model-groups/run.py` 限时运行：单次最长 60 秒，
TEMP/TMP/GOTMPDIR 指向该 D 盘目录，前端 `--maxWorkers=1`，
NODE_OPTIONS 为 `--max-old-space-size=4096 --no-experimental-webstorage`。
后一选项绕开 Node 26 原生 Web Storage 与 jsdom 的冲突，不修改产品配置。
实际工具链为 Go 1.25.5 windows/386、Bun 1.4.4、Node 26.13.0。

### 红绿证据

- `backend-red2.log`：管理列表、搜索、精确详情与规则详情在初始、改名、删除三个
  阶段均因缺少名称字段失败；原分组编码数组断言已通过。
- `backend-green.log`：`go test ./controller -run TestModelMetaGroupNamesResponses
-count=1 -timeout=60s` 通过，覆盖上述 12 个请求场景，以及别名、未知、
  `__proto__` 和不返回无关分组名称。
- `default-red.log` 与 `classic-red2.log`：两套名称渲染各 3 项失败，原编码传递
  各 1 项通过。Classic 首次失败含运行环境错误，修正测试进程环境后才采纳红测证据。
- `query-red.log`：价格解析 4 项失败、价格真实查询改名后失败；首次模型查询测试
  被图标 mock 干扰，不作为业务红测证据。更正夹具图标 URL 后，
  `model-query-red.log` 明确在第二次响应找不到“改名后”而失败。
- `model-cells-final.log`：Default 名称 4 项与原 Logo 2 项通过，19.24 秒。
- `model-query-green.log`：真实 ModelsTable + Router + QueryClient 请求与重取
  回归 1 项通过，11.41 秒。
- `pricing-green.log`：名称解析 5 项、真实价格 hook 请求与改名/删除重取 1 项
  通过，8.75 秒。业务 `enable_groups` 编码数组保持不变。
- `classic-final.log`：Classic 名称、编辑回调、真实 PUT 共 5 项与原 Logo 2 项
  通过，28.32 秒；保留 Semi UI 的既有 React 18 弃用提示。
- Default 共 13 项，Classic 共 7 项。一次合并五个 Default 文件的运行达到
  60 秒而被本任务 runner 结束，随后拆分上述确定场景通过；不将超时算通过。
  Go 初次冷编译、两次 Classic 收集也曾超时，后续红绿日志是正式证据。

### 静态检查与边界

- 后端已执行 `gofmt`；复用既有 GORM 查询，无新列、迁移或数据库专用语法。
  SQLite 回归通过，未运行真实 MySQL/PostgreSQL。
- Default 定向 oxlint 无 error，曾报告 2 条 `prefer-object-has-own` warning；
  已改为 `Object.hasOwn`，最终重跑交冻结检查。Classic 定向 ESLint 通过。
- Default 定向格式化使用 `.oxfmtrc.json` 并保护原版权头；Classic 使用本项目
  Prettier。曾误用默认 Prettier 格式化 Default 文件，已用正确 oxfmt 恢复，
  最终差异仅保留本项修改与原 Logo URL 三行。
- `typecheck.log` 记录 `bun run typecheck` 的真实失败：
  `features/keys/components/__tests__/api-key-ratio-protection.test.tsx:82:66`
  的 `ByRoleOptions.exact` 类型错误属于 #404，已报协调者。协调者明确要求
  不在多工人修改期间重复全量类型检查，冻结后统一执行；不宣称类型检查通过。
- 未执行完整构建、生产访问、提交或推送；未更改路由鉴权、分组匹配或定价权限。
- 交接前专属文档 Prettier 检查与限定修改范围的 `git diff --check` 通过；
  文档列举的 14 个源码/测试路径全部存在，原 Logo URL 三行分支核对通过。
- 原审计 `pricing-prototype.cjs/json` 未改写。该夹具直接调用默认 QueryClient，
  仍会揭示全局默认结构共享行为；本项通过真实局部查询测试证明修复，不修改全局缓存。

## 追加范围：原型键与提交边界

协调者授权追加 Default `pricing/lib/group-names.ts`、
`pricing/hooks/use-pricing-data.ts`、模型列表 `models-table.tsx` 及专属测试。
审计原始夹具位于 `D:/Temp/issue403-group-audit/pricing-prototype.cjs`，保持原样。
`__proto__` 自有名称在 React Query 默认结构共享的第二次响应中丢失；未命中的
`constructor`、`toString` 会被名称解析器误当字符串读取并抛异常。
修复仅在上述两个查询局部禁用结构共享，名称解析仅接受自有字符串属性。
不改变全局 QueryCache、公开接口权限或任何业务提交值。

另获准核对 Classic `EditModelModal.jsx`：静态阅读发现完整响应进入表单，
但真实组件测试证明 Semi Form 提交只保留注册字段，既不提交原只读
`enable_groups`，也不提交新 `enable_group_names`。因此保留原编辑器实现，
新增实际 PUT 回归，并另测编辑回调收到的原编码数组不变；不为测试制造新的提交字段。
Classic 公开价格解析器由另一工人修复，本子项不修改。

## 2026-11-16 前端写权释放与交接

收到协调者“前端交 Claude Code”指令后立即停止所有前端代码与测试修改，仅收取
此前启动的验证。前端实现已经完成，无已知待实现项，不需要重新实现。
全部前端写权已通过 Orca 回执 `msg_816c9d129c6a` 释放；Go 实现也已完成。
以下为本任务实际修改清单，原有并行改动不归本项：

1. `controller/model_meta.go`
2. `model/model_meta.go`
3. `controller/model_meta_group_names_test.go`
4. `web/src/features/models/types.ts`
5. `web/src/features/models/components/models-columns.tsx`
6. `web/src/features/models/components/models-table.tsx`
7. `web/src/features/models/components/__tests__/model-group-names.test.tsx`
8. `web/src/features/models/components/__tests__/model-group-query.test.tsx`
9. `web/src/features/pricing/lib/group-names.ts`
10. `web/src/features/pricing/hooks/use-pricing-data.ts`
11. `web/src/features/pricing/lib/__tests__/group-names.test.ts`
12. `web/src/features/pricing/hooks/__tests__/group-names-query.test.tsx`
13. `web/classic/src/components/table/models/ModelsColumnDefs.jsx`
14. `web/classic/src/components/table/models/__tests__/group-names.compat.test.jsx`
15. 本工作记录。

未修改获准的 `EditModelModal.jsx`，原因见真实 PUT 验证；未修改
`controller/model.go`、lobe-icon、原 Logo 测试、共享解析器、locales 或公共价格接口。
`controller/model.go` 仍为原 MCP 15 行差异，模型列原 Logo 三行完整保留。

交接后待办：另一人独立审查；协调者登记 README 索引并执行冻结后的统一
typecheck、最终 lint/格式检查及构建。Classic 公开价格 `groupDetails.js` 同类修复
由 D 工人处理，与本项只共享响应语义，不共享写文件。无新增文案需要翻译。
后端响应只是新增可选 `enable_group_names`；没有新的分组 ID 持久化或授权语义。

本任务启动的验证进程已全部退出，包括 exec 会话 10581、51938、13204、89057；
没有服务、浏览器、子工人或其他后台任务需要接管，没有强杀共享进程。
外部产物仅位于 `D:/Temp/issue403-model-groups`，含 runner、格式化脚本及红绿日志。

## 跨项目影响与分享建议

- 共享契约：管理模型列表、搜索和详情新增可选只读映射，原编码数组与写入契约
  保持兼容；公开价格 API 数据、鉴权、计费和路由没有改变。
- `NewAPIForDouDi`：若存在独立分组显示名和相同模型管理表，建议评估名称投影
  修复；需先核对其分组身份/别名解析方式，不能直接复制本仓库辅助函数。
- `NewAPIModifyByGang`：若同样通过 React Query 缓存动态键名称映射，建议复用
  原型键改名的回归思路；局部禁用结构共享只适用于实际受影响的查询。
- 未向任何兄弟仓库发送消息或创建 Issue/PR。
