# zzapi 使用日志筛选 `.334` 发布

## 目标与范围

发布 `v1.0.0-rc.10.1.10.334`，包含已合并的 MAO-4 / PR #261：
Classic 使用日志分组下拉、用户名与用户 ID 切换筛选，以及查询参数编码和失败恢复。
行为与测试详见[使用日志用户筛选](22_classic_usage_log_user_filter.md)和
[分组下拉](22_classic_usage_log_group_selector.md)。

版本基线为 `custom-main` 的 `36ec12f4a`。标签固定在本版本提交合入后的提交上，
通过现有 GitHub 工作流构建 Linux 二进制和 GHCR 多架构镜像。
固定版本镜像为 `ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.334`。

用户仅授权更新 zzapi。目标为 CloudSSH 项目“API中转站”的
“RS2000 德国建站”（serverId=52、hostId=17），目录 `/home/docker/zzapi`。
只串行重建三个应用容器，保持 PostgreSQL、Redis、现有环境变量、角色、端口和挂载。
原工作区保留，发布使用独立集成工作区。

## 部署计划与回滚

现场预检已确认应用镜像为 `.329`，三节点健康；正式更新前重新检查版本、运行状态和配置。
保存 Compose、环境文件、容器身份与配置摘要、模块状态归档，并完成 PostgreSQL
自定义格式逻辑备份及 `pg_restore --list` 校验；这不等同于恢复演练。

拉取并核对固定版本镜像摘要、架构与 revision。配置比较必须证明仅应用镜像改变。
依次更新 `zzapi-slave-1`、`zzapi-slave-2`、`zzapi`；每节点健康、实际版本、镜像、
无身份接口及静态资源检查通过后继续。最终核对公网状态与数据库/Redis 容器身份不变。

任一节点失败立即停止后续操作，以备份配置中的旧镜像恢复该节点并再次检查。
保留旧镜像及数据库备份，不自动回写数据库。MAO-4 本身没有新增数据库迁移；
跨 `.329` 至 `.334` 升级仍需依赖备份和现场启动验证。

## 验证与状态

- PR #261 合并前 CI 通过；最新主分支集成后的 Go 定向测试、25 项 Classic 测试、lint、格式和 Classic 构建通过。
- 版本准备 PR [#263](https://github.com/moeacgx/NewAPI-Mao/pull/263) 已合并，CI 通过；标签 `.334` 固定在 `164224d89e9b9061b4b9d14842d2874030ce194f`。
- 无真实后台测试会话时，不宣称完成登录后的使用日志浏览器验收。

## 实际发布与备份

2026-09-23 完成发布与 zzapi 更新。[正式 Release](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.334)
包含两架构二进制及校验文件；Linux 工作流 `35862522203`、多架构镜像工作流
`35862522266` 均成功。本机下载两份二进制并与 `checksums-linux.txt` 比较，均通过。

- AMD64 二进制 SHA-256：`2503e6175f07ba2d158796c739880ede9429a410eaed48f8753147a5bf65d50b`。
- ARM64 二进制 SHA-256：`c1cd573fc604407524867e82a82d145c410030634eaaa119cc42297592579eba`。
- 镜像摘要：`sha256:8ae99ab73c36cef680e0c8516f60398eb6719054afdebc6e09c0203535f598a9`。
- 现场 AMD64 image ID：`sha256:65069fd6516f6d4d42755e63081e5a4df1b3e6fb1dae2f5c8d1bcf11d6fb39cc`。
- 镜像 revision 与发布提交完全一致。

备份作业 `54ea35c0-3597-4808-be5b-2c2ee4c2463d` 成功，目录为
`/home/docker/zzapi/backups/pre-334-mao4-20260923`。数据库备份 20,277,738 字节，
`pg_restore --list` 通过；同时保存 Compose、环境文件、容器配置快照和模块归档。
未备份 Redis 内存，也未执行数据库恢复演练。

- 数据库 SHA-256：`a23f59f7512da2e9869a04d5d7dd318e3f0c18dbdb1089e5bdbe125bfdb0f01a`。
- 原 Compose SHA-256：`24fa673aabb14de3f2c430d5bc972139c66e4f149b27c8703d966e6ae9b19363`。
- 模块归档 SHA-256：`aef8966e2530b9d3d374d46605146684b0302e6ef188700b614656354d2e9ec9`。

配置准备作业 `dd2586d5-08a2-4320-8744-36a2aafaa289` 拉取固定镜像并检查架构、摘要与 revision。
通过 YAML 节点定位只修改三个应用镜像值，结构化对比证明其他配置不变。
旧版 Compose 的 `config --format json` 实际返回 YAML，首次解析预检失败，
改用 YAML 解析后通过；没有因预检错误重建容器。

## 逐节点与最终验收

按以下顺序完成，每个作业均 `SUCCEEDED`、`exitCode=0`：

1. `zzapi-slave-1` / 18098：`84e77cfb-47da-4f4d-a8ae-dd5d473ba7cd`。
2. `zzapi-slave-2` / 18099：`d7023ff6-a1d3-43c9-abd2-56cfae76630f`。
3. `zzapi` / 18097：`37e5152e-d4f9-4a19-8f52-cb17ae24732f`。

最终作业 `671f28a1-d8f7-42cb-9be2-fe4f5c6f1f11` 成功：

- 三节点版本均为 `.334`，实际 image ID 一致，healthy、restart=0，环境变量、端口和挂载未变。
- 每节点 `/api/user/self`、`/api/log/`、`/api/log/self/`、`/v1/models` 未携带身份时均返回 401。
- 节点启动日志检查未发现 `panic`、`fatal error` 或 `[FATAL]`。
- 公网状态返回 `success=true`、`.334`；本机独立请求得到相同版本。
- `/console/log` 及 `/assets/index-zCiIpZh4.js` 返回 200；脚本 11,813,363 字节，包含 `userSearchType` 和分组接口引用。这证明新 Classic 资源已到位，不替代登录后的真实筛选验收。
- 29 个模块文件的集合和 SHA-256 不变；PostgreSQL、Redis 的容器 ID、启动时间和重启次数不变。
- maolaoapi 未操作；未执行回滚，旧镜像和备份保留。

PR #261 合并后的一次后台 CI 因临时目录清理失败而报错，重跑作业 `35771110908` 已通过；
版本准备 PR 与合并后的 CI `35862019193`、`35862511688` 均通过。
