# 渠道禁用通知：余额不足前缀去重

日期: 2026-09-13

## 目标

同一上游常被拆成「供应商 / 分组 / 倍率」多条渠道。上游余额不足时这些渠道会在短时间内连续自动禁用，Telegram 通知刷屏。新增可配置的渠道名前缀短时去重，并提供余额不足预设。

## 实现

- `filter_config.prefix_dedup_seconds` 仅对 `channel_disabled` 任务生效。
- 前缀取渠道显示名第一个 `/` 之前的文本；窗口内同一任务同一前缀只投递第一条。
- 去重状态写在通知 receipt 表，键与事件幂等键隔离；入队已有序列锁，窗口判断与投递创建在同一事务。
- 未开启前缀去重的任务仍接收全部匹配事件。
- Default / Classic 通知任务编辑器增加窗口输入，以及「填入余额不足去重预设」：关键词 `预扣费额度失败`、`余额不足`，窗口 300 秒。

## 未做

禁用和启用仍是两个事件类型。两者模板和变量不同；余额不足刷屏只发生在禁用路径。若仍需合成单一「渠道事件」，另开改动。

`.322` 只发 GitHub Release / GHCR 镜像，不滚动更新线上容器。

## 验证

- `go test ./model ./controller ./service -run 'Notification|ChannelNotification' -count=1 -timeout 180s`
- Default `filter-config` Vitest 与 Classic `filter-config` Node 测试
