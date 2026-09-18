# 扩展模块开发

## 目标

扩展模块用于把可信的一方后台能力挂载到宿主控制台。模块清单、资源安装、
权限校验和宿主 API 调用都必须由 NewAPI 统一收口，避免扩展页面自行保存或读取
用户访问令牌。

## 模块清单

模块源码和版本安装包集中发布在 [maolaonewapi-extensions](https://github.com/moeacgx/maolaonewapi-extensions)，
Task Plugin 的独立 [maolaonewapi-plugins](https://github.com/moeacgx/maolaonewapi-plugins) 仓库保持原安装方式。
两个包格式不互通，宿主业务、权限和数据库迁移仍在主程序中维护。

内置和外部模块都通过 `manifest.json` 声明：

- `id` 使用小写字母、数字、`-` 或 `_`。
- `runtime.type` 支持 `static` 或 `http`；原生 UI 只允许 `static`。
- `ui.pages[].path` 必须是模块内绝对路径，不能包含跳转或路径穿越。
- `ui.pages[].render.type = native` 时，必须声明 `sdk = v1`。
- 使用原生 UI 时，`permissions.capabilities` 必须包含 `ui.native`。

## 在线安装

Default 与 Classic 的 Root 扩展管理页在操作区提供“在线模块”按钮，点击后用弹窗展示固定维护源的目录，保留原 ZIP 上传。
未打开弹窗时不请求市场配置或外部目录；弹窗包含加载、错误重试、空态和可滚动模块列表。
安装仍需确认具体模块与版本，下载或上传过程中禁止关闭弹窗，成功后刷新已安装列表。
普通关闭会清除未提交的安装选择，再次打开可重新选择；关闭后焦点返回入口按钮。
`GET /api/extension-admin/marketplace` 返回固定 `catalog_url`、`repository_url`、`host_version`、`max_archive_bytes`。
浏览器下载 `catalogVersion:1`、`purpose:extension-catalog` 的版本目录，每条带
`id/name/version/path/sha256/size/host`，显示名称、版本和兼容要求，确认后安装所选版本。
外部下载不发送登录凭据或 Referer，不跟随重定向；清单限制 2 MiB，ZIP 限制 100 MiB，按实际响应流字节执行限制。

浏览器核对包大小和 SHA-256 后，将 ZIP 连同 `archiveSha256`、`expectedId`、`expectedVersion`、
`catalogUrl`、`archivePath` 发送原 `POST /api/extension-admin/upload`。
五项元数据必须全部存在或全部省略；仅接受固定目录与同目录 `published/<id>/<version>/<id>-<version>.zip` 路径。
服务器不访问上传的 URL，而是在替换文件前检查实际字节 hash、清单 ID/版本、宿主兼容和原有安全解包规则。
失败保留已安装内容和启用状态，首次安装关闭，更新沿用原启用状态。SHA-256 只表示完整性，不是发布者签名。
兼容错误仍可能由宿主最终拒绝；公开目录存在不代表任意旧站点已有所需宿主接口。

固定源和双端交互/失败矩阵见 [在线安装工作记录](../workflows/2026-09/18_extension_repository_marketplace.md)。

## 原生 UI

`native v1` 页面由宿主前端动态加载模块声明的入口和样式资源。入口文件必须默认
导出 React 组件，宿主会注入 `globalThis.__NEW_API_EXTENSION_NATIVE_SDK__`。

Default 前端可使用宿主 SDK 暴露的 `@/lib/api` 客户端；Classic 前端可使用
`../../helpers` 中的 `API`。模块页面应通过这些宿主客户端访问后端接口，继承当前
后台登录态，不应在扩展资源中持久化 bearer token、个人访问令牌或 API Key。

`native v1` 的 SDK 模块表按目标模板分别定义，不能跨模板复用别名或组件：Default 的
`@/components/*`、`@/lib/api` 和 React Query 不属于 Classic 契约；Classic 页面必须只
使用 Classic 宿主公开的模块（例如 `react`、`react/jsx-runtime`、`react-i18next`、
`../../helpers` 及按需的 Semi UI）。每个 `targets.default` 与 `targets.classic` 入口都应
在对应宿主 SDK 下独立加载验证。

原生资源通过同源 `/api/extensions/{id}/native/{pageKey}/{target}/{asset}` 加载。
浏览器加载这类资源时不一定带 `Authorization` 头，因此宿主会在已认证的扩展列表
请求上发放 `new_api_extension` HttpOnly cookie。该 cookie 仅限
`/api/extensions` 路径，用于资源加载阶段识别当前后台会话。

多节点共享模块目录时，`state.json` 的修改不会自动更新其他进程的模块快照。
安装、启停或卸载后，应对每个节点调用 Root 接口 `POST /api/extension-admin/refresh`，
再通过各节点的 `GET /api/extension-admin/?all=true` 核对状态、版本、清单错误和资源修订号。
只对负载均衡入口执行一次刷新不能保证节点一致。模块在本节点仍关闭时，原生资源会返回 403，
可能表现为样式成功但动态入口加载失败。该路径要求后台会话，不能使用管理 PAT 代替资源 Cookie 验收。
现场证据及恢复边界见 [maolaoapi 部署记录](../workflows/2026-09/18_maolaoapi_model_guard_deployment.md#启用后的原生页面加载故障)。

## OKX 支付宝汇率模块

`okx-alipay-rate` 是内置 Root 模块，用于读取 OKX C2C 支付宝 USDT/CNY 档位，
再按模块配置做固定值或百分比调价。

该模块不会直接覆盖全站 `USDExchangeRate`，也不会改写 `/api/status.price`。
`USDExchangeRate` 仍是站点计价展示和美元转 CNY 的通用兜底汇率；`price` 继续表示
普通充值使用的 `operation_setting.Price`。

OKPay 充值要使用本模块时，支付设置中的 `OkpayRateSource` 必须保存为
`okx-alipay-rate-module`。此时 OKPay 先按 `OkpayExchangeRate` 计算 CNY 应付金额，
再用模块返回的最终 USDT/CNY 汇率换算为实际 USDT 币数创建支付订单。
模块启用状态是 OKPay 模块源缓存契约的一部分；模块被禁用后，新订单必须立即回退到
手动兜底汇率，不能继续使用禁用前缓存的 OKX 报价。

模型广场折扣展示按 `分组倍率 * (price / usd_exchange_rate)` 计算综合折扣；
`usd_exchange_rate` 失败时回退配置的 `USDExchangeRate`，不直接读取 OKX 模块汇率。

旧值 `okx-alipay-tier` 仍表示 OKPay 内置的 OKX 档位配置路径，与本模块配置分开。

## 上游模型校验扩展

`upstream-model-guard` 按分组与请求模型配置允许的上游响应模型，发现不匹配时禁用
整条出错渠道，保留原因并通过通知中心 Bot 通知，需人工启用恢复。接口、生命周期、
匹配边界和验证方法见 [上游模型校验扩展](upstream-model-guard.md)。

本模块是通过 ZIP 上传安装的外置扩展，源码位于 `extensions/upstream-model-guard`，
不自动安装或嵌入宿主二进制。七语资源随插件包携带，界面使用既有 `native v1` SDK；
Classic 通过 `../../helpers.getAPI()` 取得刷新登录态后的当前客户端。

清单声明 `channel.upstream-model-guard` 能力，仅允许同 ID、Root、静态模块。
该能力表示宿主提供实时响应采集、规则保存、整渠道禁用及通知入队支持。
旧宿主不认识该能力时会拒绝清单，避免页面安装成功但检测没有生效。
这不是可执行任意后台代码的通用插件运行时，ZIP 不包含 Go 二进制。
安装后默认关闭，Root 显式启用才绑定检测。停用或卸载后不再绑定，重启不会自动恢复插件。
`0.2.0` 额外要求 `channel.upstream-model-guard-tolerance` 能力，支持渠道白名单和默认两次的连续不匹配阈值。
旧宿主必须拒绝新能力，避免页面配置可保存而运行时仍单次关渠。配置和记录契约见模块专题文档。

## 对话归档扩展

`conversation-archive` 是仅 Root 可用的补丁模块，用于定位异常对话。配置 API 位于 `/api/extensions/conversation-archive`，请求体在认证后按用户 ID 和稳定分组代码筛选，命中后才进入清洗和持久化。

清洗载荷只保留 `messages[].role` 与纯文本 `messages[].text`，以及模型、协议、请求 ID、用户和分组等必要元数据；媒体、base64、工具 schema、请求头、Cookie、Authorization 和 URL 查询均丢弃。未知协议仅在能提取到有限文本时保存，协议字段保留调用链提供的标识；无可识别文本时跳过。单条消息、消息数和总字节数均有硬上限。OpenAI Realtime 会合并客户端和上游增量文本，忽略音频与完成事件重复正文。

列表接口仅返回元数据，详情接口才返回清洗后的消息。所有接口使用 `RootAuth`、禁缓存和限流，详情按纯文本渲染，防止扩展页面执行 HTML 或再次加载超大 JSON。配置在进程内使用 2 秒 TTL 快照，更新通过版本 CAS 后立即失效本地快照。

Default 使用 Default 原生 SDK 与 `@/lib/api`；Classic 使用 Classic 原生 SDK 与
`../../helpers.API`，并分别维护入口和样式。两套入口不能互相复制宿主组件依赖。

## 兼容性与运维

归档模型使用 GORM 标准字段，正文采用项目的大文本类型封装，兼容 SQLite、MySQL 5.7.8+ 与 PostgreSQL 9.6+。配置了稳定的 `CRYPTO_SECRET` 时正文使用 AES-GCM 加密后入库，详情接口在服务端解密；未配置时保留明确的明文兼容模式。过期记录不会出现在列表或详情接口；主节点每小时按 ID 小批量删除数据库记录。

除按天过期外，配置中的 `max_archive_count` 以全局最近会话数限制归档量，默认
1000，允许 1 至 100000。新归档写入和数量裁剪在同一数据库事务中完成，并锁定
配置单例；超出限制时按 `created_at ASC, id ASC` 删除最旧记录。将上限调低后，
配置 CAS 事务会立即裁剪已有记录。Root 可通过
`POST /api/extensions/conversation-archive/conversations/clear` 携带
`{"confirm":true}` 清空全部归档；采集未关闭时，后续命中的请求仍会再次归档。
当前实现没有外部对象存储，配置更新使用 `config_version` 乐观锁。升级前确认数据库
迁移已执行，回滚时先停用扩展再处理新增表。定时过期清理与写入、数量裁剪、手动
清空使用同一配置行锁，避免并发操作删除同一批归档。手动清空还要求显式确认并
使用关键操作限流。
