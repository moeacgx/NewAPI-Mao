# Classic 连接结束后保留 5 秒

> `.350` 首次上线后用户报告白屏，已回退 `.349` 并由用户确认恢复。以下部署通过项仅
> 表示当时的基础检查，不代表页面实际可用；后续处置见
> [静态资源错误缓存排查](22_static_asset_negative_cache.md)。

## 目标与契约

用户要求响应结束后连接卡片停留约 5 秒，避免立即消失。仅改 Classic 展示及其活动观察后端；
Default 不接入页面，不修改转发、鉴权、计费、数据库或依赖。

- `status=completed` 表示生命周期已结束，包含正常返回、失败、取消与 panic，不代表业务成功。
- 新增可选 `completed_at`（Unix 毫秒），首次结束时固定，重复结束与迟到响应不得续期或复活。
- 结束记录在结束时刻起保留 5 秒；列表 `items`、`total` 包含尚未到期的结束记录，活动记录优先。
- `summary` 和令牌 `active_only` 仅统计尚在运行的请求，结束记录不冒充活动连接或正在使用的凭据。
- Redis 完成记录租约截止于结束后 5 秒，正常活动仍续租 15 秒；固定节拍清理内存与发布。
- Classic 显示“已结束”，停止运行计时；无首响应时不再显示“等待响应”。保持桌面固定卡位、手机一行。
- 保留窗口按服务端结束时间计算，Redis 发布与页面各 2 秒轮询，页面移除会有轮询延迟。

## 验证计划

先观察失败回归，再实现：确定性时间边界（4999/5000 毫秒）、重复清理幂等、迟到回调、
活动统计、活动优先、多节点 Redis 到期与发布失败重试；取消、panic、真实本地 HTTP 结算；
Classic 卡片耗时冻结、仅剩结束记录时面板保留、桌面与手机展示、翻译、lint 与构建。
Go 单次测试超时 60 秒；不得以本地测试代替生产部署或真实供应商收费请求验证。

## 验证结果

- 已先观察注册表立即删除、Redis 立即删除及 Classic 缺少结束状态的失败断言，再实现保留行为。
- Go 确定性回归覆盖 4999/5000 毫秒边界、重复收尾、迟到回调、活动优先、统计排除、
  结束前尚未发布的短请求、多节点租约固定到期、删除重试和发布期间结束。
- `go test ./pkg/activerequests ./service ./controller ./router -count=1 -timeout 60s` 通过；
  后续新增短请求 Redis 与结束状态查询测试单独复跑通过。含本地真实 HTTP Relay 结算验证，钱包
  扣费仍为原预期值；取消和 panic 保留结束记录，活动统计归零。
- 后端只读协作审查未发现阻断问题；未进行 CGO race、实机收费请求或压力验收。
- Classic 活动模块 31 项回归通过，受影响 JSX 的 ESLint、locale 同步、Prettier 及生产构建通过。
  当前主机无 Bun，使用已安装包的 Node CLI；Windows 链接依赖需保留 symlink 并预打包真实图标、
  Mermaid 依赖。冷启动曾在测试收集阶段触发 60 秒超时，未将超时当成通过。
- Chromium 151 使用真实组件及 API 边界演示数据验收：1440px 桌面三卡约 447×230，
  390px 手机每行约 328×51，无横向溢出或页面错误；中性“已结束”、8.0 秒冻结耗时、
  无首响应提示、零首响应与手机展开均符合预期。这是本地渲染证据，不代表生产登录业务验收。

## 发布与更新计划

继续此前已授权的发版与逐节点更新流程，版本为 `v1.0.0-rc.10.1.10.350`。
在 PR、CI 和正式镜像验证通过后，从 `.349` 固定摘要滚动更新 maolaoapi：
`maolaoapi-slave-1` → `maolaoapi-slave-2` → `maolaoapi`。
执行前重新核对主机、镜像、版本及 Compose，备份配置、应用数据和数据库，
每节点通过健康、配置、鉴权与静态资源验收后才更新下一节点。失败停止并使用旧镜像回滚应用，
不恢复在线数据库。实际发布及更新结果见下文。

## 正式发布证据

- 功能 PR [#315](https://github.com/moeacgx/NewAPI-Mao/pull/315) 已合并，发布提交为
  `7d052443eddf9c52276314c751d3e2137ac36900`；标签 `.350` 的 peeled SHA 与之完全相同。
- 功能 CI [39680736446](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39680736446)、
  合并后 CI [39681381052](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39681381052)
  均通过。CodeRabbit 跳过自动审查，不作为独立代码审查证据。
- [Linux Release](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39681428966) 与
  [双架构镜像](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39681428808) 工作流均成功。
  [Release](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.350)
  已发布且非草稿，包含 amd64、arm64 和 `checksums-linux.txt` 三个资产。
  下载校验文件的摘要与 GitHub 元数据一致，两条二进制摘要也与各自资产元数据一致；
  未下载二进制执行安装验收。
- GHCR 固定镜像：
  `ghcr.io/moeacgx/newapi-mao@sha256:875c38f3572ef92d9450f57304b24cd1f76d4c9f0bb40975c787f44b0693aff5`。
  amd64 manifest 为 `sha256:34105f05a51019be004e6a70d186ebeb5f75ff39ed25f854db8e9c507c84821e`，
  arm64 manifest 为 `sha256:6747096620644e7f6b84b79b89313a19a7649c2695bee73d7a9cd6bfc6bbe531`；
  两个平台 config 的 OCI revision 与 version 均符合发布提交及 `.350`。

## 生产准备与备份

- CloudSSH 重新核对项目“API中转站”的完整主机名“OVH兜底”，`serverId=63`、
  `hostId=29`。目录 `/apps/maolaoapi`，Compose 项目 `maolaoapi-ovh`。
  三应用更新前均为 `.349`、Classic、healthy、restart=0。
- 旧镜像为
  `ghcr.io/moeacgx/newapi-mao@sha256:7d864896c6263c92b835b38f614f0820a7fd10144aa6f669f4870d9f3d364ee7`，
  revision 为 `94249a98aacac241512a7ffe68f22421dc58c588`。
- 备份作业 `fc4d4b32-bb49-4cf1-827b-5266e53298d1` 成功、exitCode=0。
  目录 `/apps/maolaoapi/backups/pre-350-20261021T214250Z` 使用 UTC 命名，对应北京时间
  2026-10-22；包含配置、五容器 inspect、data、数据库 custom dump、TOC 和 SHA 清单。
- 数据库备份为 1,573,069,612 字节，`pg_restore --list` 成功，SHA-256 为
  `42ebbe5513d6fca6fc295565cc848641bceb13251cf631fb11815436e42d6a89`；
  data 为 88,640,019 字节，SHA-256 为
  `ac72e7a85f6ae9f431eef89e7efbbad291ec224b109b94ed77a33c020c4ce2ec`。
  私有配置与 inspect 仅留目标主机，未进行在线数据库恢复演练。
- 更新脚本与回滚入口为 `/apps/maolaoapi/releases/350/ops.py`；固定目标 revision，
  脚本 SHA-256 为 `a1b1f2dd30e546c158d759b3fb04ee73ed008e12934834c18b9e53603b519830`。
  使用独立空 Docker 配置匿名拉取固定摘要，保留原登录配置。
- 候选准备作业 `ac9f3b50-9819-44ac-95c8-9d3d4c5c37a0` 成功、exitCode=0；
  目标主机已拉取的 amd64 镜像标签、revision、version 符合预期，新旧镜像默认环境完全一致。
  拉取前后三应用和依赖身份均未改变，之后才允许进入节点重建。

## 逐节点更新与最终验收

- 第一从节点作业 `e738ab10-c29a-4596-a7a6-2ad411a682fa`、第二从节点作业
  `be100cdf-4d0c-433d-9e18-9d505cde65f8`、主节点作业
  `edd37675-bc35-468f-820e-691616f54f87` 按顺序执行，均为 SUCCEEDED、exitCode=0。
  每次更新前重新验收前序节点，未同时重建应用。
- 最终作业 `90caff82-2e0d-46b9-9f11-ebf582e11d85` 成功、exitCode=0：三应用均为
  `.350`、Classic、上述固定镜像及准确 revision，healthy、restart=0。
  有效环境变量、端口、挂载、命令、重启策略和资源限制保持，PostgreSQL/Redis
  容器 ID、启动时间和重启次数均保持。
- 三节点匿名 `/api/user/self`、`/v1/models`、`/api/log/self/active` 与
  `/api/token/search?active_only=true` 均返回 401，启动日志未发现 panic、fatal、迁移
  失败或数据库连接失败标记；备份文件大小、权限与 SHA-256 再次逐项校验通过。
- 目标主机访问公网 `/api/status` 返回 200、`.350` 与 Classic；使用日志入口及其 JS/CSS
  返回 200。公网和三节点 JS 为 `/assets/index-BhzpjldY.js`（11,909,622 字节），
  CSS 为 `/assets/index-C0aJS442.css`（788,955 字节）。
  资源包含 `completed_at`、“已结束”对应文案、5 秒提示、手机紧凑行与桌面固定网格标记。
- 本地独立公网检查也确认版本、Classic、JS/CSS 及四个匿名接口的 401，与目标主机
  报告一致；没有把静态标记检查当成登录用户真实请求验收。
- 最终脱敏报告保存在备份目录的 `deployment.verified.json`；旧镜像、备份及
  `/apps/maolaoapi/releases/350/ops.py rollback` 保留；后续用户报告白屏后已执行回滚。
- 本次生产验收覆盖部署、资源与匿名鉴权，未使用登录用户执行真实收费供应商请求；
  5 秒时间边界与冻结耗时由上述本地回归及真实组件渲染验证。

## 共享契约与分享建议

新增可选 `completed_at`、`completed` 状态和结束保留，活动统计及令牌筛选继续只计进行中请求；
转发、鉴权、用量和计费契约不变。Default 未增加页面入口。

- NewAPIForDouDi：若采用当前连接观察，建议同时评估 5 秒保留与冻结耗时，能改善短请求可见性；
  未检查目标实现，适配需确认轮询和请求结束时机。
- NewAPIModifyByGang：若采用当前连接观察，建议同步评估 Redis 固定结束租约及活动统计排除，
  避免结束记录误算为凭据仍在使用；未检查目标实现。
- 本次未向兄弟仓库发送消息、Issue 或 PR。
