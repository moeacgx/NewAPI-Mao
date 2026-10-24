# 工单中心 .353 合并验证

## 目标与范围

将工单中心 PR #325 的 `8cfd88156fed88492fbeb85f35aeb32a15222698` 与当前 `custom-main` 合并，供 .353 组合发版使用。使用独立工作树和本地集成分支，原始工单会话、管理、批量操作、软删除、自动回复及自动归档契约见[工单专题](../../developer/tickets.md)和[批量与归档记录](24_ticket_bulk_archive.md)。

## 合并方案与兼容边界

- 首轮主分支基线为 `d0124985f795a8093042ec6a1cebc885a9c1f182`，已包含 PR #323 的令牌按分组限定渠道；后续对齐 PR #324 的最终主分支提交。
- 仅开发文档首页和能力总览产生文本冲突。按功能段落同时保留工单和令牌限定能力、链接与既有主分支记录；双模板语言文件由 Git 自动合并后再做键值保留核验。
- 工单权限、纯文本会话、事务、通知正文隔离、默认关闭的自动化以及三库迁移契约保持现有实现。令牌渠道限定和当前连接行为按主分支实现保留。
- 本次仅在本地整合及提交；远端 PR 合并、正式构建、发布和部署由主协调者负责。没有生产操作或外部通知。

## 验证计划

- 使用指定 Go 1.25.5 amd64 工具，工单模型、控制器、路由、权限和后台归档相关测试均设置 `-timeout=60s`。
- 对比合并结果与两个父提交的双模板语言 JSON，确认双方新增键和值被保留；检查冲突标记、Markdown 格式、本次新增与合并链接及 `git diff --check`。
- 完成 PR #324 对齐后复核差异及受影响检查。MySQL/PostgreSQL 实库、真实 Telegram 和生产验收不包含在本地验证中。

## 验证结果

- 首轮合并 `d0124985f795a8093042ec6a1cebc885a9c1f182`：已逐段解决两处开发文档冲突；双方索引完整保留。本次新增与合并的 5 条本地链接均可解析。
- 双模板共 15 份语言 JSON 的 101,944 个叶值通过三方比对，双方变更均保留，没有冲突值或丢失键。
- 指定 Go 1.25.5 amd64 运行 `go test -p=2 -mod=readonly -count=1 -timeout=60s ./model ./controller ./router ./service ./service/authz -run 'Test(Ticket|CreateTicket|.*TokenGroupChannel.*|SetUserPermissionsStoresOnlyOverrides|CapabilitiesUseCatalogShape)'`，5 个包全部通过；涵盖工单、归档、相关令牌限定及权限契约。
- 合并带入的 Go 文件通过 `gofmt -l` 检查，手工合并及新增 Markdown 通过仓库 `oxfmt` 格式化，工作区和暂存区 `git diff --check` 均通过。
- PR #324 的最终主分支提交尚未对齐；完成第二轮合并后继续记录最终基线与验证结果。MySQL/PostgreSQL 实库、真实 Telegram 和生产验收未执行。
