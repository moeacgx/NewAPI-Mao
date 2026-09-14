# 数据库上游兼容阶段

## 基线、范围和方案

工作区为 `upstream-backend-db`，分支 `agent/upstream-backend-db`，PR base 为 `custom-main`。固定基线 `45ab82100a74b9e7c3866f1c173d16d32745527e`，上游分析点 `9fe0457ee`，rc.37 为 `385d2dfd1`。本任务与认证 PR 独立，不包含其改动。

旧分支 `353352428` 干净但落后 92 个提交，独有 `353352428`、`99db8805c`、`5fb6a75ca` 三个合并提交，无额外文件差异。保留 `backup/upstream-db-before-sync-20260915` 后将当前分支基于最新远端；未 reset、clean 或覆盖既有文件。

本阶段实现：

1. 移植 `1751f43ee` 的 SQLite 默认 DSN：驱动可识别的 busy_timeout、WAL 和 BEGIN IMMEDIATE。自定义 SQLITE_PATH 仍由操作者控制，文档给出完整示例。
2. 根据 `007d69942` / 固定上游最终实现，迁移 PostgreSQL 预填分组旧全局 name 唯一约束和索引，按真实定义匹配，在排他锁事务内替换为现有软删除部分唯一索引。保留无关复合/表达式/部分索引和外键；遇到依赖失败回滚，不用 CASCADE。
3. 保留 SQLite/MySQL 的现行预填分组约束语义、所有钱包 int64/BIGINT 迁移及单请求计费边界，不引入认证、任务或 relaykit 实现。

## 验证计划和安全边界

- SQLite 用真实临时文件、两个连接验证忙等待参数、WAL 读写与立即事务写锁；测试无睡眠或吞错。
- PostgreSQL 用专用 TEST_POSTGRES_DSN 中独立事务/schema 验证导入改名约束、软删除后重建同名、无关索引保留、外键依赖回滚及重复迁移。未配置实库时明确跳过，不将跳过计为实库通过。
- SQLite/MySQL 验证迁移无副作用及既有索引；钱包回归覆盖 5,000,000,000 额度保持。
- 所有 Go 测试使用 `-timeout=60s`。本机 Docker Desktop Linux 引擎管道不存在，不能据此声称 PostgreSQL/MySQL 已运行；不启用桌面服务或访问生产数据库。

## 待整体迁移项目

PostgreSQL pooler 修复必须同时处理 JSON Valuer 的 string/[]byte 语义，其中涉及被本任务排除的任务模型；本阶段不单独关闭 PrepareStmt。GORM/SQLite 驱动升级、MySQL decimal 默认值和 PostgreSQL bpchar 元数据归一化、通用旧约束处理、options 主键重建、token 历史约束迁移需分别验证，不在本补丁中宣称完成。上游钱包 `2^53-1` 边界不得覆盖本地 int64/BIGINT。

## 三数据库兼容矩阵

| 数据库          | 本阶段行为                                                                                                        | 未覆盖或不变的边界                                                                                   |
| --------------- | ----------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| SQLite          | 默认启用 WAL、30000ms busy_timeout、BEGIN IMMEDIATE；预填分组迁移函数不发 DDL                                     | 自定义 SQLITE_PATH 不自动补参数；WAL 不能替代写入限流，仍只有一个写者；预填分组现有部分唯一索引保留  |
| MySQL 5.7.8+    | 不改变连接参数，不修改预填分组表/索引                                                                             | MySQL 不支持 PostgreSQL 风格部分索引，现有全局唯一限制仍可能阻止软删除后同名重建；不宣称该差异已修复 |
| PostgreSQL 9.6+ | 用 pg_catalog 读取单列全局 name 唯一对象的真实名称，在事务排他锁内替换为 name WHERE deleted_at IS NULL 的唯一索引 | 保留其他索引、外键；目标索引定义不符或依赖无法移除时终止迁移；实库验收待完成                         |

迁移加入普通 `migrateDB` 和并行 `migrateDBFast` 的 AutoMigrate 前，位于已有钱包迁移之后。钱包迁移、定价/返佣/日志表、后台任务租约、认证安全表没有替换。本补丁不新增 API、配置字段、前端 DTO；Default 和 Classic 共享现有预填分组 API，无需为本阶段协议改动修改页面。软删除重用同名在 PostgreSQL 生效，MySQL 现有限制须作为独立功能处理。

## PostgreSQL 迁移保证与失败处理

- 只识别 `prefill_groups.name` 的全局单列唯一约束，或不隶属约束的全局单列唯一索引；排除主键、复合、表达式和已有部分索引。
- 初次读取目录发现冲突才进入事务；取得 ACCESS EXCLUSIVE 锁后再次读取，防止另一迁移进程先完成。
- 历史表缺 deleted_at 时在同一事务添加，再删除冲突对象并创建/核验现有 `uk_prefill_name` 部分唯一索引。名称引用使用 GORM 标识符，不拼接未转义的数据库对象名。
- 不使用 CASCADE。外键依赖会导致 DROP CONSTRAINT 失败，事务恢复旧约束、索引和新增列；启动报错后由管理员检查依赖设计，不能强制放宽。
- 本地旧 PostgreSQL 驱动会拆分目录对象名中的点号；不能直接用 DropIndex/DropConstraint 引用。补丁查询目标表实际 schema，并对 schema、表、约束、索引各段双引号转义后执行限定 DDL，保留字面点号和双引号，防止 search_path 选中其他 schema 的非目标索引。DryRun 已复现错误 SQL 并验证修正；实库用例额外保护 pg_temp 中的同名索引。
- 新库或没有冲突的现有库不执行替换；AutoMigrate 仍负责普通建表。错误返回沿原启动链上抛，不忽略迁移失败。

## 升级与回滚

1. 升级前备份数据库，PostgreSQL 预留取得表排他锁的维护窗口；本补丁只交付代码，不自动访问生产数据库。
2. 未设置 SQLITE_PATH 时采用新默认；已有自定义路径按需要显式增加参数：

   ```env
   SQLITE_PATH=/path/to/sqlite.db?_pragma=busy_timeout(30000)&_pragma=journal_mode(WAL)&_txlock=immediate
   ```

3. SQLite 目录必须可写以创建 `-wal`/`-shm` 文件；在线备份使用 SQLite 备份机制，不能只复制正在写入的主文件。WAL 需要适合 SQLite 共享内存/锁的本地文件系统，已有网络存储部署需单独验证或显式选择旧日志模式。
4. 回滚应用不会自动将 SQLite journal_mode 从 WAL 改回 DELETE；确需切换时须停止写入、完成 checkpoint 并按 SQLite 维护流程处理。不要删除 WAL 文件来回滚。
5. PostgreSQL 保留迁移后的部分索引及历史行即可回滚到本地基线代码；若想恢复全局唯一约束，必须先处理软删除历史同名，不得直接收紧导致启动失败。失败迁移本身会回滚，但此前成功的其他迁移不在本事务内。
6. 钱包继续 int64/BIGINT，不执行降列；5,000,000,000 等已有额度不能因回滚被截断。未引入新 Go 依赖或修改版本号。

## 实库验收方法

测试只接受操作者明确提供的专用 `TEST_POSTGRES_DSN` / `TEST_MYSQL_DSN`，不得指向生产库。PostgreSQL 测试用随机 schema 和事务回滚隔离；MySQL 使用随机测试表并清理。测试覆盖改名约束、多重索引、目标名占用、外键回滚、真实行保留、重复启动和软删除后重建。

```sh
go test ./model -run 'TestMigratePrefillGroupUniqueness' -count=1 -timeout=60s -v
go test ./common ./model -run 'Test.*(Database|SQLite|Migration|Migrate|Wallet|QuotaCapacity|Column|Index|Prefill)' -count=1 -timeout=60s
go vet ./common ./model
```

SQLite 已复现旧参数无效：旧日志模式 delete，busy_timeout 实际 5000 而非预期 30000；新配置下真实双连接回归通过。迁移定向回归已通过 SQLite 部分，PostgreSQL/MySQL 明确跳过；不能引用上游发布者的实库结果代替本分支验证。完整实库验证前以草稿 PR 交付，不标为全量数据库集成就绪。

## 本地验证结果（2026-09-15）

- `go test ./common ./model -count=1 -timeout=60s`：引用修正后最终全部通过，common 3.702s、model 19.879s；外部数据库可选测试无 DSN 时跳过。
- 数据库/迁移/索引/钱包定向模式通过，model 2.183s；对应模式下 common 无匹配测试，未把它算作定向覆盖。
- `go vet ./common ./model`、gofmt、修改 Markdown 的 oxfmt 格式化和 `git diff --check` 通过。
- 根模块 `go build ./...` 通过；初次因缺少两套前端 dist 嵌入资源失败，按 CI 方式仅在忽略目录创建编译占位后重跑。占位未提交，不能作为可发布前端产物。
- SQLite 运行版本为 3.50.4，实际驱动沿用仓库既有版本，未升级依赖。50 亿额度的迁移/读写回归通过；没有把上游 int32 废弃或钱包上限改动移入。
- PostgreSQL 迁移及测试来自固定 `9fe0457ee`，以当前连接预处理策略适配测试；本地只做编译/静态审查，不能声明 PostgreSQL 9.6/16 或 MySQL 5.7/8 实库通过。
- 名称引用修正有本地 PostgreSQL DryRun 精确 DDL 回归，覆盖点号、双引号及包含 SQL 标点的合法标识符；不连接服务器，不能代替实际 DDL/回滚验收。
- 本阶段不影响 relaykit 公共接口，未改其实现；未执行生产 API、合并、发布、部署或远端数据库迁移。

## 跨项目影响

无请求/响应、模型 ID、额度交换或生产运维变更；没有向兄弟项目发送通知。认证 PR 独立，数据库阶段未修改认证、任务、relaykit 或钱包实现。
