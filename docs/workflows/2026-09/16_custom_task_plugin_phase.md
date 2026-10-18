# 自定义 Task Plugin 第一阶段

## 目标

在官方内置 JS Task Plugin 之外，允许 Root 将经过宿主编译校验的 JavaScript
插件源码保存为自定义版本，并在 Task Plugin=62 渠道中绑定使用。现有内置插件、
AtlasCloud=61、Go 原生任务适配器和 `/extensions` 二开扩展继续保留。

## 已实现

- `POST`/`PUT /api/plugin/task` 接受 JSON `source`，源码上限 1 MiB，编译失败只返回稳定分类错误。
- 自定义版本保存为 `source_kind=custom`、`active=false`、`enabled=false`；重复相同 hash 的上传幂等，
  不会意外停用已经激活的版本，不同 hash 的相同 key/version 返回冲突。
- `DELETE /api/plugin/task/:key/versions/:version` 只允许删除非活动且没有历史任务 pin 的版本；
  活动版本必须先切换，历史任务按 key/version/source hash 继续读取。
- `/api/task_plugin_options` 从持久化版本生成，内置和自定义插件都可供 62 类渠道绑定。
- Default 和 Classic 的 Root 管理页提供源码上传、来源展示和非活动自定义版本删除；Admin 只读，
  本节记录上传阶段，多源市场续作见[多源工作记录](16_task_plugin_sources_plan.md)。远程资源、S3、匿名签名和 dry-run 仍不开放。

## 安全与兼容边界

上传沿用 RootAuth、全局/关键操作限流和管理审计。源码 hash 只用于完整性校验，不能作为发布者签名。
总开关默认关闭；关闭只阻止新提交，不影响已保存任务的版本 pin、轮询、结算或退款。

## 验证

- `go test -mod=readonly ./controller ./service ./model ./router -run 'TaskPlugin|PluginTask|CustomTaskPlugin'`
- `go vet -mod=readonly ./controller ./service ./model ./router`
- `relaykit` 使用 `GOWORK=off` 独立 build/test。
- Default 定向 oxlint；Classic 页面使用 Prettier 检查。当前环境没有 Bun、前端依赖目录，未运行两套
  前端的 typecheck、完整测试或生产构建。

## 后续工作

真实 MySQL/PostgreSQL 迁移、订阅退款与故障恢复组合、完整 TokenAuth 端到端、供应商媒体成功交付、
S3/匿名签名和 UsageFacts 表达式计费仍需单独验收，不因本阶段管理能力开放而宣称生产就绪。
