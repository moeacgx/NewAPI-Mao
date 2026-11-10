# 图片任务跨域核验（ALHE-102 / #26）

## 目标与范围

核验 [#26](https://github.com/moeacgx/NewAPI-Mao/issues/26) 中图片任务创建、编辑、
查询和内容获取的浏览器跨域行为。反馈只有路径及期望响应头，没有受影响版本、
浏览器请求或反向代理信息。本轮按用户 2026-11-10 最新要求恢复执行。

核验基线为 `origin/custom-main`（`6422f334cdbdc48ac36750bafcd811de83cb2fda`）。不得因历史报告
直接修改安全策略；先验证路由、预检、错误和成功响应，有可复现缺陷才改实现。

## 接口与安全契约

| 方法 | 路径                                         | 认证               |
| ---- | -------------------------------------------- | ------------------ |
| POST | `/v1/images/tasks`                           | API 令牌           |
| POST | `/v1/images/tasks?action=edits`              | API 令牌           |
| GET  | `/v1/images/tasks/{task_id}`                 | API 令牌、任务归属 |
| GET  | `/v1/images/tasks/{task_id}/content/{index}` | API 令牌、任务归属 |

- 这些是内置异步图片任务接口，注册在 `router/canvas-router.go`，不是 JS Task Plugin。
- `router/main.go` 的全局 OPTIONS 边界按路径选择 `RelayCORS`，在业务鉴权和
  Web fallback 前返回预检；实际请求的路由组也在鉴权前应用同一策略。
- Bearer 接口允许第三方来源，响应为 `Access-Control-Allow-Origin: *`，允许
  GET、POST、OPTIONS 以及 Authorization、Content-Type；不批准浏览器凭据。
- 浏览器通过 `Authorization: Bearer <API_KEY>` 发送令牌，使用
  `credentials: 'omit'`。`credentials: 'include'` 与通配来源不兼容；不得为它
  把公共 Bearer 策略改成任意来源加凭据。直接 `<img src>` 无法附加 Bearer 头，
  应认证 fetch 内容后显示 Blob URL。
- 预检无须 API 令牌；真实请求仍鉴权。查询和内容读取按用户与任务平台隔离，
  不属于当前用户或平台的任务返回 404。CORS 通过不代表任务创建、上游推理成功。
- `/canvas/v1/**` 是独立会话入口，使用精确来源和凭据策略，不适用上述通配策略。
- Issue 中 `Access-Control-Max-Age: 86400` 是预检缓存建议，不是跨域通过的必要条件。

## 验证计划

1. 在真实路由注册上为四种请求发送无认证 OPTIONS，断言 204、空正文、通配来源、
   必要方法/请求头且无允许凭据头，避免只断言 HTTP 成功。
2. 同四种真实请求无令牌时应得到可跨域读取的 401 JSON，避免让鉴权错误表现为
   浏览器无法读取的响应。
3. 使用隔离 SQLite、固定任务数据和 API 令牌验证成功查询、图片字节读取与跨用户
   拒绝，保持 CORS 头和归属边界。不调用收费上游，不触及生产环境。
4. 运行相关路由、中间件和控制器回归；检查格式、文档链接及 `git diff --check`。

## 前端、兼容与回滚

Default (`web/src`) 和 Classic (`web/classic/src`) 共用上述 Go 路由与中间件，
本项不改变页面、配置、国际化或提交值。分别搜索两个入口目录后，`images/tasks`
仅出现在图片结果保留时间的设置说明与语言包，没有这四个接口的页面调用。
Default 对应 `web/src/features/system-settings/maintenance/performance-section.tsx`，
Classic 对应 `web/classic/src/pages/Setting/Performance/SettingsPerformance.jsx`。
因此不制造页面修改；两套模板构建与浏览器页面运行未执行。鉴权、计费、用量、数据库模型和 relaykit API
均不在修改范围。若只增加测试与文档，回滚对应提交即可，不涉及数据迁移。

## 验证结果

`router/image_task_cors_test.go` 的两个测试、10 个子场景已通过：

- 四种请求分别验证无认证 OPTIONS 为 204、正文为空、来源为 `*`、允许必要方法和头，
  真实请求无令牌为带 CORS 头的 401 JSON；已注册 Web fallback，确保没有返回首页。
- 使用有效 API 令牌，任务所有者查询为 200，返回本站内容地址；读取为 200，返回
  精确 PNG 字节和 `image/png`，两者均带跨域头。
- 另一个用户查询/读取，以及将 Canvas 平台任务交给普通图片任务接口查询/读取，
  均为带跨域头的 404，且不泄露任务标识或图片数据。

命令（Windows 本地 Go 1.25.5；进程级 `GOARCH=amd64`）：

```powershell
$env:GOARCH = 'amd64'
go test ./router -run 'TestImageTaskBrowser' -count=1 -timeout=120s -v
go test ./router ./middleware ./controller -count=1 -timeout=180s
```

上述完整包测试全部通过：router 21.609 秒、middleware 15.568 秒、controller 11.972 秒。
没有确认源码层 CORS 缺陷，因此仅增加行为回归与文档，
不修改生产 CORS、鉴权或任务实现，不将预检缓存时长差异当作故障修复。

Not run：真实浏览器跨域、反向代理/负载均衡环境、各部署节点版本核验、收费上游
创建/编辑图片、MySQL/PostgreSQL 实例测试、双模板构建。没有模型或 SQL 变更，
数据库用例使用内存 SQLite，真实创建/编辑仅核验预检与无认证拒绝路径。

## 部署侧复核方法

将 `api.example.test` 替换为实际部署地址，并从发生问题的浏览器 Network 面板
复制真实 Origin 与 Access-Control-Request-Headers；不要在反馈中携带真实令牌或 Cookie。

```sh
curl -i -X OPTIONS 'https://api.example.test/v1/images/tasks?action=edits' \
  -H 'Origin: https://browser.example.test' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: authorization,content-type'

curl -i 'https://api.example.test/v1/images/tasks/nonexistent-task' \
  -H 'Origin: https://browser.example.test'
```

预期第一条为 204，来源 `*`，允许 POST 及请求头；第二条为带来源 `*` 的 401 JSON。
这两个请求不创建图片或消费额度。OPTIONS 若返回 HTML、301/302 或缺少跨域头，应
定位接入代理、路由与实际处理节点；多个节点分别取证，不能只用一个成功响应判断全站。
真实请求若是可读的 401/404，则继续检查 API 令牌与任务归属，不应扩大 CORS 权限。

浏览器凭据模式核验示例（`apiKey`、`taskId` 从调用方现有安全上下文取得）：

```js
const response = await fetch(
  `https://api.example.test/v1/images/tasks/${encodeURIComponent(taskId)}/content/0`,
  {
    credentials: "omit",
    headers: { Authorization: `Bearer ${apiKey}` },
  },
);
if (!response.ok) throw new Error(`Image content HTTP ${response.status}`);
const objectURL = URL.createObjectURL(await response.blob());
// 将 objectURL 交给图片元素；元素不再使用时调用 URL.revokeObjectURL(objectURL)。
```

后续定位需要：受影响版本、浏览器完整错误文字、失败方法/路径、脱敏 Origin/请求头名、
OPTIONS 与实际请求的状态/响应头、是否启用 `credentials: 'include'`、代理/负载均衡结构。
本地测试不等于部署环境复现，也不代表 #26 已修复或应关闭。

## 共享契约与分享建议

本次没有 API、模型、认证、计费、配额或跨系统契约变更。NewAPIForDouDi：若保留同源
异步图片路由，值得参考四路径回归与 Bearer/凭据区分；没有已确认的实现补丁供移植。
NewAPIModifyByGang：同样可参考核验方法，但应先确认自己的路由和鉴权策略，不能照搬
通配策略到会话接口。本轮不向兄弟仓库创建 Issue、PR 或发送通知。
