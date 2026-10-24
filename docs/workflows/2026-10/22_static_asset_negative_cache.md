# 静态资源 404 缓存与 .350 白屏排查

## 目标与现场

用户在 `.350` 三应用更新后报告整页白屏，截图显示当前 Classic CSS 返回 404。
此前发布验收只覆盖 HTTP、资源标记与匿名鉴权，未覆盖生产浏览器实际渲染，不能说明
页面可用。当前优先恢复 `.349`，再修复资源错误缓存并补齐实际页面验收。

## 已核实与待证实

- CloudSSH 三节点与本地独立公网访问当前 `.350` CSS 均返回 200；本地公网为 LAX
  Cloudflare 节点。用户浏览器实际访问同一 CSS 返回 `Invalid URL` JSON。
- 源码 `middleware.Cache` 给非首页响应设置一周缓存；缺失 `/assets` 最终 404 未覆盖
  缓存头，确实允许客户端和 CDN 长期保存失败响应。
- 滚动更新时新 HTML 与旧节点可能交错，旧节点不包含新哈希资源。这是可触发缺失资源的
  部署条件；尚未取得用户失败响应的 Age、CF-Cache-Status 等完整响应头，不能将
  所有白屏直接归因于已证实的 CDN 负缓存。
- 5 秒保留改动未修改前端启动入口；生产包静态依赖和卡片错误边界审查未发现启动崩溃
  原因。CSS 404 通常不足以单独解释根节点空白，仍需实际浏览器验证。

## 修改方案与兼容边界

- Web 缺失静态资产返回 404 前明确设置 `Cache-Control: no-store`，保留原 JSON 错误
  契约。现有资源仍使用原缓存策略，当前主题资源兜底继续 no-cache。
- 两套模板使用共用 Web 路由，必须同时验证；不修改任一模板业务组件、API、鉴权、
  数据库、转发或计费。
- 已缓存的旧响应不会因新响应头自动消失；正式重新上线后针对准确资源 URL 清理 CDN，
  并在真实浏览器验证使用日志页面，不能用状态接口代替渲染。
- 当前 Cloudflare 凭据能读取目标 zone，但精确 URL 清理返回 401 / 10000，未清理成功，
  未尝试扩大权限或清理整站。内置 HTML 的本地样式表 URL 改为携带当前发布版本参数，
  保留原查询参数与片段、不改外部样式表，避免复用上一版本缓存的失败响应。
- 重新上线前以独立空 SQLite 预览实例提取正式镜像静态资源，不连接生产数据库或 Redis，
  将新旧版本公开资源预置到独立静态目录。Nginx 仅 `/assets/` 优先读取预置资源，缺失时
  回到既有应用 upstream；原业务和 API location 保持。变更先备份、nginx -t 后重载，
  避免滚动更新时新 HTML 命中仍未升级的旧节点而缺少资源。

## 测试与发布计划

先写 Gin 路由回归复现：缺失 JS/CSS 为 404 且 no-store，覆盖模板、方法与查询参数；
正常资源和主题兜底维持既有行为。Go 每次超时 60 秒，再执行 router 包回归、格式检查。
使用独立 PR 与新标签发布，保留已发布 `.350` 标签；逐节点更新、失败停止，继续保留
应用回滚能力。记录 CDN 操作范围与浏览器实际证据，禁止清理整站缓存或修改 DNS。

## 恢复证据

回滚作业 `5cbf27d4-d03d-4ef9-b5d0-1854b3161dbf` 成功、exitCode=0，按主节点、第二
从节点、第一从节点顺序恢复 `.349` 固定镜像；三节点 healthy、restart=0，配置与依赖
身份保持，未恢复或改写数据库。用户浏览器实际访问 `/console/log` 已显示顶栏、筛选与
表格，恢复渲染；用户已明确确认“已恢复正常”。5 秒保留随应用回退暂时撤下，待修复后
重新发布上线。

## 本地修复验证

- 新增真实 Gin 路由回归先复现 16 个失败：Default/Classic 的缺失 CSS、JS，GET/HEAD
  与带查询参数请求均错误继承 `max-age=604800`。
- 最小修复只在缺失 `/assets` 返回前覆盖为 `no-store`；原 JSON 404、正常资源一周
  缓存、跨主题兜底 no-cache 均通过契约检查。
- 受影响路由回归及 `go test ./router -count=1 -timeout 60s` 完整包通过；gofmt、
  Prettier 与差异空白检查通过。修改两套模板共用的路由，未改模板业务实现。
- `.350` 已改为预发布并注明暂停采用，`.349` 恢复为 Latest；原标签和资产保留审计。
- 本地样式链接版本隔离回归先观察 12 个断言失败，再实现 `v=common.Version`；
  双模板的首页、index.html 和 SPA 入口、已有参数与片段、外部资源不变、重复处理
  幂等及实际 CSS 响应均通过。完整 router 回归再次通过。

## 共享边界与分享建议

仅缺失 Web 资源的缓存响应头变化，转发、鉴权、计费、模型与令牌契约保持。
NewAPIForDouDi 和 NewAPIModifyByGang 均值得检查同源路由是否缓存缺失静态资源；
只有存在相同链路才适合采用该小修复，未审查目标源码，也未发送 Issue 或 PR。

## .351 发布证据

- 修复 PR [#316](https://github.com/moeacgx/NewAPI-Mao/pull/316) 已合并，标签
  `v1.0.0-rc.10.1.10.351` 指向 `2088c02169b3ed1dbd8b7c58dc1811ef777c90b6`，包含
  原 dev 能力、5 秒保留和本次缓存修复。
- 最终源码 [CI 39687816881](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39687816881)
  后端与前端均通过；独立只读审查未发现阻断，未把 CodeRabbit 的跳过作为审查证据。
- [Linux Release 39688978607](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39688978607)
  与 [Docker 39688978581](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39688978581)
  均成功。[Release](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.351)
  三资产齐全，下载校验文件与两条二进制元数据摘要匹配；未执行二进制安装验收。
- 固定镜像为
  `ghcr.io/moeacgx/newapi-mao@sha256:55406df0ecd564fa62f7fd491d0e402fc7143de1d788f126a81c111be7ab8754`。
  amd64 manifest 为 `sha256:57b206b46c836b98bb2dc2c1f46444561fbd8d7bf484d4865a5250507866a83d`，
  arm64 为 `sha256:c4053222e9a95ebf057e9036edc347fc93f76ea1093315e50d5be685c2686f99`，
  两个平台 config 的 revision/version 均匹配上述发布提交及标签。
- 修订后的运维脚本作业 `efac9512-4e1f-42fb-820f-3bbe72b59489` 成功，目录为
  `/apps/maolaoapi/releases/351/`。`ops.py` SHA-256 为
  `495f18f465cc602417f793b2faa1824118d5c1add66b5434269a6a2bf2d1a025`；
  预览与 Nginx 脚本经过只读审查，安装后验收失败也恢复原配置并测试重载。
- 新备份作业 `300b3a45-3697-4e7c-8fbd-789d75ce193f` 成功、exitCode=0，目录
  `/apps/maolaoapi/backups/pre-351-20261021T230031Z`。数据库 custom dump 为
  1,577,339,874 字节，SHA-256
  `821b905ff1a8a0413265fc0233591f4e038cc6216335af9007137ec1a7505d01`；
  data 为 88,640,019 字节，摘要与 `.350` 前备份相同，TOC 与 tar 目录检查通过。
- 候选拉取作业 `227bcf4d-7988-40f9-a6ce-d1c8c2f49c45` 成功，镜像版本、revision、
  amd64 与旧镜像默认环境均通过；三应用尚未重建。
- 首次空 SQLite 预览按 slave 启动，因从节点跳过建表而缺失 `casbin_rule`，未就绪即停止；
  生产应用、数据库和代理配置均未修改。随后使用另一独立预览目录按 master 初始化空库，
  仍无生产 DSN、Redis 或生产数据挂载，容器仅监听宿主回环地址，提取结束后停止。
- 递归清单在旧版与候选均发现两个下划线前缀 JS 分块返回 404；旧版为
  `_basePickBy-CGq9fpEe.js`、`_baseUniq-Dzy54Pbl.js`，候选为
  `_basePickBy-GSv9Hmqw.js`、`_baseUniq-Bo7eqJcR.js`。这是两版本已有的嵌入资源边界，
  本轮将缺失逐项记录且不宣称所有懒加载路径已验收；未扩大修改打包规则。
- 提取作业 `717dc334-6c73-407e-b26c-e4257a1c75d5` 成功，预置新旧 174 个公开资源、
  共 32,771,286 字节，目录 `/var/www/maolaoapi-static/assets/`。同名文件必须 SHA-256
  一致才复用，空库预览的 Classic、版本样式参数、结束保留标记和 404 no-store 均通过。
- Nginx 安装验收作业 `4bc7c582-04dd-483b-bf16-8c5329c47014` 成功，`nginx -t` 与重载
  通过，本源 TLS 入口对 174 个资源逐个核对内容 SHA-256，未知资源为 404/no-store。
  配置从 `a32b20c4a3819e4a588f61a7dfc4819b6231fe71d4e9a9ad142dc068f5eee63a`
  变为 `7a421c03f1ecec1efd6d327b6c18a89343023db7f419e703759ac68fea748e8b`，
  备份为发布目录内 `nginx.before-static.conf`；仅前端 `/assets/` 优先静态读取，
  缺失回到原 upstream，API 站点和 location 保持，应用与依赖身份未改变。
- 第一从节点作业 `1323e046-92b6-4a60-aeef-6b2dbcfdaa79`、第二从节点作业
  `4416583d-0e63-4dc3-960f-f9a8d4c9eb4e` 依次成功、exitCode=0。均为 `.351`、准确
  revision、固定镜像、healthy、restart=0，配置与依赖保持，匿名接口 401、直接节点
  缺失资产 404/no-store、样式 `?v=v1.0.0-rc.10.1.10.351` 与结束保留资源标记均通过。
- 两从节点完成、主节点尚为 `.349` 时，Chrome 实际 `/console/log` 已显示顶栏、筛选与
  表格，未出现白屏；该检查只证明页面渲染，不代表真实收费供应商请求验收。
- 主节点作业 `2c5b3c66-84b8-40d6-b519-3849235fe203` 成功、exitCode=0；最终验收
  `989f8c73-b93c-4696-8b85-125b5c676e66` 成功，三节点 `.351`、healthy、restart=0，
  配置与依赖身份保持，备份逐项哈希再次通过。
- 独立公网检查：首页 200/no-cache、版本 `.351`；带版本参数的 CSS 为 200，788,955
  字节、Cloudflare MISS；未知 CSS 为 404/no-store、Cloudflare BYPASS，无 Age。
  原清理 API 未成功，不将版本参数隔离表述为已清空 CDN。
- 全量更新后在用户 Chrome 刷新 `/console/log`，实际看到带样式的顶栏、侧栏、实时
  当前连接卡片、“已结束的请求保留显示 5 秒”提示、筛选和有数据的日志表格；查询按钮
  可见，当前捕获的控制台 error 列表为空。未主动发送收费请求，未声称完成所有懒加载
  页面或严格生产五秒计时验收。浏览器页保留供用户复查。
- `.351` 回滚入口仍是发布目录的 `ops.py rollback`，仅恢复应用与 Compose；静态目录
  同时保留 `.349` 和 `.351` 公开资源。若需撤销 Nginx 静态入口，核对现有配置摘要后
  恢复 `nginx.before-static.conf`，先 `nginx -t` 再重载；不恢复在线数据库。
