# 兑换码多人兑换次数修复

## 目标与根因

兑换码表单已提交 `max_redeem_count`，列表也显示 `redeemed_count/max_redeem_count`，但后端 `Redemption` 模型没有这些持久化字段，创建接口也丢弃兑换上限。兑换逻辑仍将状态从启用直接改为已使用，因此一个兑换码只能被兑换一次。

## 范围与方案

- 将每码最大兑换次数和已兑换次数保存到 `redemptions`；旧数据默认最大次数为 1，维持历史行为。
- 新增兑换使用记录，以兑换码 ID 和用户 ID 唯一约束，避免同一用户通过同一码重复领取。
- 兑换在事务内验证状态/过期时间、原子占用一个名额、写入用户使用记录、发放额度与返佣；只有最后一个名额被占用后状态才改为已使用。
- 更新名额时同时比较事务读取的已兑换次数；首位兑换人与时间从事务快照确定，不引用同一 UPDATE 内刚递增的字段，避免 MySQL 赋值顺序与 PostgreSQL/SQLite 的差异。
- 新建码与编辑接口持久化每码上限。上限必须为 1 至 100000；编辑时不能调低到已兑换次数以下。
- `used_user_id` 保留首位兑换用户的历史字段兼容；后台进度使用持久化计数。

## 接口、数据和兼容性

API 字段沿用现有前端契约：`max_redeem_count` 表示每个码可兑换的最大用户数，响应增加 `max_redeem_count` 和 `redeemed_count`。批量创建的 `count` 仍只表示生成多少个不同兑换码。

`redemptions.max_redeem_count` 默认 1，`redemptions.redeemed_count` 默认 0。新增 `redemption_usages` 表记录 `(redemption_id, user_id)`，并对该组合建立唯一索引。使用 GORM AutoMigrate，以兼容 SQLite、MySQL 和 PostgreSQL。

历史已使用码仍保持已使用，历史启用码最多可再兑换一次。已有 `used_user_id` 无法可靠回填完整使用记录，因此不将历史用户重新开放给已用码。

## 权限与安全边界

兑换入口继续由现有用户认证路由保护，名额占用和额度发放共用数据库事务；同一用户重复兑换不得增加额度或返佣。上限更新只通过管理端现有接口执行并校验。

## 测试计划与结果

- 后端回归覆盖多用户达到上限、重复用户拒绝、历史码一次性兼容、并发最后名额只发放一次，以及旧表迁移默认值。
- 运行兑换相关 Go 测试与 `git diff --check`。
- 验证结果：`go test ./model -run 'Test(DeleteRedemptionsByIDsArchivesSelectedCodes|SearchRedemptionsFiltersAndPaginates|Redeem)' -count=1 -timeout=60s` 通过；`git diff --check` 通过。
