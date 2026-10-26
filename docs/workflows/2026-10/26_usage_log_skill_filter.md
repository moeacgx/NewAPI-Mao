# 使用日志技能筛选

## 目标与范围

Classic 使用日志新增“技能”下拉筛选：全部、已使用技能。管理员可与用户名或用户 ID 组合查询，普通用户仍只能查询本人日志。分组右上角技能徽标由独立 PR 交付；Default 本次不增加界面控件。

## 接口契约

`GET /api/log/`、`GET /api/log/self/`、`GET /api/log/stat`、`GET /api/log/self/stat` 增加可选 `skill=used`；省略或空字符串不筛选，其他非空值返回 400。旧调用保持兼容。

“已使用”对应徽标的 used 状态：非 discovery 阶段且 `prompt_templates` 数组有数字 id 的条目，或 `gateway_skills.stage=generation` 且 files 数组存在数字 template_id 与非空字符串 skill_id 的条目。discovery 轮次始终排除；仅有目录、未读取、格式错误及无技能信息均不匹配。费用只汇总匹配的消费日志，RPM/TPM 保持原有最近 60 秒口径并叠加技能条件。

## 实现与安全边界

- 筛选必须在分页之前执行，列表总数与统计均使用实际匹配结果，不能只过滤当前页。
- 保留原有用户归属、管理员权限、分组、时间和其他条件。
- 日志 other 为 TEXT，PostgreSQL 9.6 无安全 JSON 有效性函数。为兼容坏历史数据，不直接对任意 TEXT 强转 JSON；仅技能筛选路径使用 SQL 条件缩小候选后逐行解析 JSON，复用 `common.Unmarshal`。候选条件只排除不可能包含技能字段的记录，不以模糊字符串判断最终匹配。
- 列表仅保留所需页结果，统计仅保留累加器；查询继承 HTTP 取消并设置超时，失败不返回部分统计或假总数。非技能筛选保持原数据库分页和聚合路径。
- 不增加日志表字段、索引、迁移或后台回填。技能筛选仍需扫描时间范围内候选记录，建议结合时间、用户和分组缩小范围；超大范围可能超时，长期可另做派生索引字段。

## 验证计划

覆盖始终装载、正式读取、发现优先、未读取、坏 JSON 和字段类型错误；跨页总数、用户隔离、费用/RPM/TPM一致性、取消与非法参数。前端覆盖表单选择、分页保留、清除筛选和列表/统计请求参数。执行 Go 受影响测试（60 秒超时）、Classic 测试、lint、格式与构建。MySQL/PostgreSQL/ClickHouse 实库验收需另记，不能由 SQLite 测试推定。

## 验证结果

- 整合后 `GOARCH=amd64 go test ./model ./controller -run 'Test.*Log|TestParseLog' -count=1 -timeout 60s`：126 项通过；`go vet ./model ./controller` 通过。
- Classic 筛选、角标、详情组合：4 文件 20 项通过；请求参数与用户筛选原生测试 15 项通过；语言与编码 2 项通过。
- 相关 ESLint、Prettier、gofmt、文档链接、`git diff --check` 通过。Classic 本次为 JSX 改动，无 TS 类型变更。
- Classic 生产构建通过（1 分 5 秒）；环境未安装 Bun，使用现有 Node 与已安装依赖运行等价命令，保留既有大资源包及 Browserslist 警告。
- Chrome 独立组件预览已验证“技能 → 已使用技能”选中后提交 `used`，重置后恢复全部；实际筛选语义由后端测试验证，未连接生产日志数据。
- 数据库行为测试仅 SQLite；MySQL/PostgreSQL/ClickHouse 实库未验收。无需迁移，但大时间范围仍可能达到 15 秒超时。
- 15 秒为单次查询期限，不是并发准入；本次不新增技能筛选专用并发或频率限制，多个大范围查询仍可能增加数据库和 CPU 负载。
- 后端与前端链路只读复核无阻断项。未提交 PR、未部署。跨项目影响仅新增可选查询参数 `skill=used`，省略时兼容旧请求；计费写入、日志归属与权限不变。
