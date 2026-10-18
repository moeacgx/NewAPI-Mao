# maolaoapi 渠道余额不足前缀去重与 .322 发布

## 目标与范围

发布 `v1.0.0-rc.10.1.10.322`，把渠道禁用通知的余额不足前缀去重合入
`custom-main`，并生成 GitHub Release / GHCR 镜像。

**本次不更新任何线上容器。** maolaoapi、zhishiapi、zzapi 生产仍继续运行
`.321`，不改 Compose、不 `docker compose up`、不滚动重启。

## 变更

- `channel_disabled` 任务新增 `filter_config.prefix_dedup_seconds`。
- 渠道名按第一个 `/` 取供应商前缀；同一任务同一前缀在窗口内只投递第一条。
- Default / Classic 通知任务编辑器提供「余额不足去重」预设：
  关键词 `预扣费额度失败`、`余额不足`，窗口 300 秒。
- 禁用 / 启用仍是两个独立事件类型。

## 发布步骤

1. 功能与 `VERSION` 经 PR 合入 `custom-main`。
2. 在合并提交上打标签 `v1.0.0-rc.10.1.10.322`。
3. 等待 Linux Release 与 Docker 多架构镜像完成。
4. 停止。不拉取镜像到生产，不替换容器。

## 回滚

未改线上容器，生产无需回滚。若要撤销仓库标签或 Release，另开任务处理。
