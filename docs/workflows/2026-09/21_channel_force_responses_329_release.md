# zzapi 渠道强制 Responses `.329` 发布

## 目标与范围

用户授权提交 PR、合入 `custom-main`、发布版本并更新 zzapi。
版本为 `v1.0.0-rc.10.1.10.329`，功能与边界见
[渠道强制使用 Responses 上游](../../developer/channel-force-responses.md)。
Default、Classic 均提供默认关闭的 `setting.force_responses` 开关，
保留客户端协议，要求上游支持 Responses；不会替用户批量开启已有渠道。

集成基线为 `028970ff4`，包含已合并的 PR #242 Classic 性能配色修复。
发布保留该修复，并从通过 CI 的合并提交创建固定标签。

## 现场预检

CloudSSH 入口为 `API中转站 / RS2000 德国建站`，serverId=52、hostId=17，
Compose 为 `/home/docker/zzapi/docker-compose.yml`。
预检作业 `37ecdb7f-c5aa-4abb-9b44-212f91d28920` 确认：

- `zzapi`、`zzapi-slave-1`、`zzapi-slave-2` 分别监听 18097、18098、18099。
- 三节点版本均为 `.328`、healthy、restart=0，使用相同正式镜像。
- 主节点未设置 `NODE_TYPE`，两个从节点为 `slave`；保留现场角色与挂载。
- PostgreSQL 与 Redis 独立运行，升级只重建三个应用服务。

## 发布与滚动更新

1. PR 通过当前基线检查与 CI 后合并，标签固定到该合并提交。
2. 由现有工作流构建 Linux amd64/arm64 二进制、SHA-256 校验文件及 GHCR 多架构镜像。
3. 备份 PostgreSQL、Compose、环境文件、模块目录，记录容器身份与配置摘要。
4. 拉取固定版本镜像并核对 revision；只替换三个应用的 image，结构化比较其余配置。
5. 按 `zzapi-slave-1`、`zzapi-slave-2`、`zzapi` 顺序逐个重建；每个节点健康、版本、镜像与日志检查通过后再继续。
6. 检查公网状态和静态资源、无身份认证边界、模块文件摘要及数据库/Redis 容器身份。

备份不包含 Redis 快照；`pg_restore --list` 只验证备份目录，不代表完成恢复演练。
不修改 maolaoapi、zhishiapi，不读取业务令牌或向真实上游发送付费请求。

## 兼容与回滚

此功能不新增数据库迁移，不改变客户端鉴权、模型 ID、返回协议或计费契约。
只有显式开启的支持渠道改变上游对话协议；未开启保持现有规则。
默认关闭使滚动更新期间已有渠道保持一致行为，待节点全部更新后再配置使用。

任一节点检查失败即停止后续更新，按备份 Compose 恢复该应用的旧镜像并验证。
保留 `.328` 镜像和备份，不回写业务数据库；旧版本忽略新增开关字段。
真实供应商协议兼容性没有通过线上付费请求验证，依据模拟上游回归判断。

## 验证依据

功能实现已通过后端定向回归、`relay/common` 和 `relay/channel/openai` 全包测试、
`relaykit` 独立构建、31 项前端相关回归、Default 类型检查、受影响文件 lint、
两套前端生产构建与根模块构建。浏览器工具受认证限制，未完成真实浏览器检查。

## 发布记录

[PR #243](https://github.com/moeacgx/maolaonewapi/pull/243) 已通过 CI `35525524665`
及 PR 质量检查 `35525524660` 后合入 `custom-main`。
标签固定到合并提交 `299a4baeb003442b738a3336d39f87949f899199`。
合并后 CI `35525977569` 首次因 Classic 的 `mermaid` 依赖压缩包解包失败，
未修改依赖与锁文件，重跑失败作业后全部通过。

Linux 发布工作流 `35526010663`、GHCR 多架构工作流 `35526010636` 均成功。
[正式 Release](https://github.com/moeacgx/maolaonewapi/releases/tag/v1.0.0-rc.10.1.10.329)
已补齐中文说明。现场作业 `e196df09-df99-4038-a0a2-68380937aa7e` 下载两份二进制，
根据发布校验文件核验 SHA-256 均为 OK，结果与 GitHub 资产摘要一致：

- amd64：`f14335f93a4ee3057c749220d48f62d6797102c09d392a1c1bcbf80c13c085be`。
- arm64：`1c78ba600b45c60ac8be071e0924e2f64d6b5a4e9ab34d27f2e331b5153b1c52`。
- 多架构镜像：`sha256:3c42b83f1b0ec7e31056cff4b034123e6c875379557a54c759885deac4aecc5d`。
- 实际 amd64 image ID：`sha256:2385226a9cd73f2b7311199e8bbb628188be17788a378cd719a5ce99e8cc861e`。
- 镜像 revision 标签与上述合并提交一致。

## 备份与配置准备

备份作业 `f839ad97-e31b-48a1-afe6-9cdd15ce68c8` 成功，目录为
`/home/docker/zzapi/backups/pre-329-20260920T172800Z`。
CloudSSH 首次返回网络错误，按完全相同参数重试获取原作业结果，没有重复运行备份。
保留 PostgreSQL 自定义格式备份、Compose、环境文件、模块归档、29 个模块文件摘要，
以及脱敏容器身份与运行配置摘要。数据库备份目录校验通过：

- 数据库大小：21,049,543 字节。
- 数据库 SHA-256：`002371c67aafcb509c9a5ff0987b6b3c52bd9863d995f97e167f0b121bfc1b4c`。
- Compose SHA-256：`cc66033fbc10e2ec8bb1de2d0a851aa2091e4dd4aacf50589223e2187f7e8115`。
- 模块归档 SHA-256：`d60050369d32d1ce4e1eafbde6e49b2c36a0caa9b67b314ecd73c0e26829af83`。

准备作业 `c88e2a52-dd12-471f-8c93-d495204c3fa3` 已拉取正确镜像，
但现场 Compose 的 JSON 输出不兼容，解析失败发生在改写配置和重建应用之前。
改用上一版已验证的 YAML 解析后，作业 `26722bc2-7818-4267-8d9e-863dc31420f7`
完成结构化配置比较，确认仅三个应用的 image 改变。
逐节点操作显式指定同一 Compose、项目目录和备份环境文件，并在重建前检查环境文件未变，
避免默认文件发现加载额外 override。环境变量数组按值排序后比较摘要。

## 滚动更新与最终验收

以下节点按顺序更新，每台通过验收后才继续下一台：

- `zzapi-slave-1` / 18098：`b2981215-b2e0-4265-905b-58f069a4dedd`。
- `zzapi-slave-2` / 18099：`5aa68bcf-5d78-4712-a8f2-46cc21b6a880`。
- `zzapi` / 18097：`18521767-adc5-4948-8f4d-face467ab7d3`。

最终验收作业 `2843315a-53a7-45df-b0f9-226581e0d0b5` 成功：

- 三节点版本与实际 image ID 一致，均为 `.329`、healthy、restart=0。
- 环境变量值、挂载、端口、启动命令、重启策略和网络配置摘要均保持。
- 公网 `/api/status` 返回 `success=true`、`.329`，本机独立请求也得到相同版本。
- 每节点无身份 `/api/user/self`、`/api/extensions/`、`/v1/models` 均返回 401，Classic HTML 返回 200。
- 29 个模块文件的集合与 SHA-256 完全一致。
- PostgreSQL、Redis 的容器 ID、启动时间、运行状态和重启次数与更新前相同。
- 启动后最近 300 行日志没有发现 panic、FATAL 或 `[ERROR]` 标记。

静态资源作业 `fbdfb6e5-1084-404f-b7ba-3ccce2842030` 验证公网当前 Classic 入口脚本
`/assets/index-DeCaGX23.js` 返回 200、非空 JavaScript，包含 `force_responses` 与对应开关文案。
脚本 11,791,967 字节，SHA-256 为
`4137b4142f8fff3ef9ef74c2cb024c7d52f3e1925681b10250199c9b16b18c8e`。
这证明当前静态产物已发布，不等同于线上登录后的双模板浏览器交互验收。

验收结果保存在 `/home/docker/zzapi/releases/329/verification.json` 和 `static-assets.json`。
保留旧镜像与备份；没有批量修改渠道开关，没有发送真实上游付费请求，
也没有更新 maolaoapi 或 zhishiapi。
