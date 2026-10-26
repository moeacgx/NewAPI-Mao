# 使用日志技能筛选后端

## 契约与方案

四个列表/统计入口接受可选 `skill=used`，空值保持原行为，其他值返回 400。个人接口保持当前用户隔离。
已使用指非 discovery 且始终装载模板包含数字 id，或 generation 已读文件包含数字 template_id 与非空字符串 skill_id。
坏 JSON、错误字段类型、未读取和 discovery 不匹配。

日志 TEXT 不强转数据库 JSON：以参数化 LIKE 缩小候选，保留包含 Unicode 转义的候选，随后 Go 严格解析。
所有数据库沿用原有排序，流式处理全部匹配记录后返回准确总数和目标页；统计以相同判定处理消费记录。
技能查询继承请求上下文并限制 15 秒；取消、超时或读取错误不返回部分结果。默认路径继续数据库分页聚合。
不增加表字段、索引、迁移或历史回填。大时间范围可能超时，应组合用户、分组和时间限制。

## 验证计划

SQLite 行为测试覆盖跨页、总数、权限范围、费用/RPM/TPM、异常 JSON、Unicode 转义与取消；控制器覆盖非法参数。
MySQL 5.7、PostgreSQL 9.6、ClickHouse 使用普通参数化 LIKE 和流式读取，没有 JSON 方言依赖；实库验收另行记录。

## 已完成验证

- `GOARCH=amd64 go test ./model ./controller -run 'TestLogSkill|TestLogUserID|TestParseLogUserID' -count=1 -timeout 60s` 通过。
- `GOARCH=amd64 go test ./model ./controller -run 'Test.*Log|TestParseLog' -count=1 -timeout 60s` 通过（model 2.292 秒，controller 1.508 秒）。
- `GOARCH=amd64 go vet ./model ./controller`、`gofmt`、`git diff --check` 通过。
- 数据库行为测试实际使用 SQLite；未执行 MySQL/PostgreSQL/ClickHouse 实库测试，未部署。
- 统计累加保留整数精度，溢出返回错误，不返回回绕或部分统计。
