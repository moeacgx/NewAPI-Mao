# Default 官方任务插件管理

## 范围与权限

固定参考上游 `9fe0457ee`，基于 `custom-main@5d90556e2` 新增
`/task-plugins`。现有 `/extensions`、原生扩展 SDK 与二开入口保留。
本 PR 覆盖 Default 及其开发记录；Classic 与 Go 后端由独立 PR 实现。

Admin 可读取列表、详情、版本、计数和运行状态；Root 才可启停、激活版本、
修改总开关、查看源码与源码比较。后端仍需独立实施相同权限校验。

## 接口合同

响应采用 `{success,message,data}`，业务失败必须显示错误并允许重试。

| 操作     | 接口                                      | 合同                                           |
| -------- | ----------------------------------------- | ---------------------------------------------- |
| 列表     | GET `/api/plugin/task`                    | 每 key 一项，包含 meta、状态和真实计数         |
| 详情     | GET `/api/plugin/task/:key?version=...`   | 无 active 时仍可查看最新归档，源码仅 Root 返回 |
| 历史版本 | GET `/api/plugin/task/:key/versions`      | 无源码，包含 active/enabled                    |
| 激活     | POST `/api/plugin/task/:key/activate`     | `{version}`，激活并启用                        |
| 启停     | POST `/api/plugin/task/:key/status`       | `{enabled}`，修改活动版本                      |
| 总开关   | GET/PUT `/api/plugin/task/runtime/status` | PUT `{enabled}`，默认 false                    |
| 渠道选项 | GET `/api/task_plugin_options`            | key/name/version/models/channel_type           |

首次安装 `active=false,enabled=false`，Root 需先在版本页激活。
关闭总开关只禁止新提交，历史任务继续使用固定版本处理。
启停或激活后同时刷新列表、详情、版本和渠道选项缓存。
上传、市场、删除与沙盒控件禁用，不发送这些尚未开放的请求；
不通过通用 `/api/option/` 修改插件开关，不发送 force/cascade。

## 渠道与展示

`TaskPlugin=62`、`AtlasCloud=61`。创建和编辑渠道使用 JSON 字符串
`setting.task_plugin_key`；插件渠道必须选择 key。切换其他类型不携带此键。
已有绑定不在返回选项中时保留原值；敏感写权限不足时不能修改绑定。

元数据包含声明的模型、协议、路由与用量 schema，属于插件声明；
展示这些字段不表示宿主已经开放对应协议或用量表达式计费。
源码使用 `@codemirror/lang-javascript` 只读高亮。源码比较限制二维表大小，
超限时展示整体增删，避免大文件占用过多内存。

## 编码回归与验证

PowerShell 向 Python 管道传递中文时曾把六种非英文翻译、注释与工作记录写成
字面问号。现通过 UTF-8 补丁创建翻译脚本，由 `add-missing-keys.mjs` 写入七语
资源并运行 i18n sync；开发记录整体重写，PR 正文按仓库模板独立整理。

新增测试读取真实的七语 JSON，用实际 i18next 实例验证阶段限制、关闭行为、
未激活状态和选择提示；修复前六种语言失败，英文通过。另用真实渠道绑定组件
验证启停及激活后，已打开的选项列表立即刷新，避免新鲜缓存保留旧版本。

原实现者报告 80 项测试、typecheck、定向 lint 与 Default 构建通过。
协调修复后将重跑插件测试、typecheck、定向 lint 与构建，结果在最终提交中更新。
未验证浏览器真实后端登录、供应商视频、数据库迁移或 zzapi 部署。

## 依赖与回滚

依赖官方后端 PR 的管理接口及渠道 62 支持。合并顺序为后端、双模板，再组合验证；
协调计划见 PR #219。当前仅完成内置管理与渠道接线，不能称作完整插件体验。
远程媒体、市场、源码上传、S3、匿名签名和 UsageFacts 表达式不在本 PR 实现范围。

回滚前端只移除本 PR 页面和接线；不要删除插件归档、历史任务或原生扩展。
共享 API 请求及计费行为未由此 PR 改动；未向其他项目发送通知。
