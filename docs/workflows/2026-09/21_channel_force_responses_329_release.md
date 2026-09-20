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
合并、发布和线上部署结果在执行完成后补充，本段不作为尚未执行操作的成功证明。
