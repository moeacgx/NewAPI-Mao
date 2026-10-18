# zzapi 会话刷新修复 `.328` 发布

## 目标与范围

用户授权发布版本 `v1.0.0-rc.10.1.10.328` 并更新 zzapi。
包含已合并的 [PR #238](https://github.com/moeacgx/maolaonewapi/pull/238)：
Classic 刷新遇到 429、服务端故障或断网时不再误退登，保留真实错误和静默错误配置；
明确 401 或会话身份不匹配仍要求重新登录。共享后端为刷新使用独立按 IP 限流桶，
沿用 CT 参数并保留 GA、来源、Cookie 和会话校验。
行为和回归详见[修复记录](19_classic_refresh_rate_limit.md)。

本次版本提交仅修改 `VERSION` 与发布文档，基线为 `486d0491b`。
目标为 CloudSSH `API中转站 / RS2000 德国建站`，serverId=52、hostId=17，
目录 `/home/docker/zzapi`。只更新 `zzapi`、`zzapi-slave-1`、`zzapi-slave-2`，
端口分别为 18097、18098、18099；不操作 maolaoapi、zhishiapi。

## 发布与部署

版本 PR 经 CI 后合入 `custom-main`，标签固定到合并提交；由现有工作流构建
Linux amd64/arm64 二进制、SHA-256 校验文件和 GHCR 多架构镜像。

现场预检确认三个应用均为 `.327`、healthy、restart=0。
更新前备份 PostgreSQL 自定义格式逻辑备份、Compose、现有环境文件及模块目录，
记录应用、数据库、Redis 容器身份和配置摘要。备份不包含 Redis 快照；
`pg_restore --list` 仅校验备份目录，不表示完成恢复演练。

拉取固定版本镜像并验证摘要。通过结构化配置比较确保只改变三个应用镜像，
保留现有角色、端口、挂载和模块状态。依次重建应用，每台验证健康、镜像、
版本、无身份接口拒绝及启动日志后再进入下一台。数据库和 Redis 不重建。

## 兼容性与回滚

本版本没有数据库迁移，不改变模型 ID、API Key、Relay 或计费契约。
Default 前端已有临时故障保护，前端修复仅针对 Classic；后端独立刷新计数适用于两者。
管理员不获得全局免限流。独立刷新桶自身耗尽仍返回 429 与 `Retry-After`。

任一节点不健康或版本不符即停止后续更新，恢复该应用的旧镜像配置并验证。
保留 `.327` 镜像和备份，不回写业务数据库。回滚到 `.327` 会恢复旧的误退登行为。

## 验证

PR #238 的后端 CI、前端 CI、Classic 10 项真实 Axios 回归、既有 9 项认证兼容测试、
lint 和构建已通过。发布验收检查工作流、产物校验与镜像摘要；部署验收检查三节点和
公网 `/api/status`、实际运行镜像、健康和重启次数、静态资源、无身份认证边界、
模块文件摘要以及数据库/Redis 容器身份。没有后台测试会话时，不宣称线上浏览器
已复现完整登录续期流程，也不读取用户令牌制造限流。

## 执行记录

版本 PR [#240](https://github.com/moeacgx/maolaonewapi/pull/240) 已合并。
标签 `.328` 固定指向 `5ad71c2c4ce74fffedc6f20b847c025eccdee310`。
CI `35426719865` 首次在 Classic 依赖安装时出现 `mermaid` 包解压失败；
未改依赖或锁文件，重跑失败作业后全部通过，PR 质量检查也通过。

Linux 工作流 `35427227489`、GHCR 多架构工作流 `35427227460` 均成功。
[正式 Release](https://github.com/moeacgx/maolaonewapi/releases/tag/v1.0.0-rc.10.1.10.328)
已补齐中文说明，发布两架构二进制及 `checksums-linux.txt`。
现场作业 `915465d2-6974-4d56-8a15-7e0d423d880d` 下载两份二进制并核验 SHA-256，均为 OK。

- amd64 二进制：`88e9796f9d25f06c08317ba89324195f2d3ae00c6b849ca8c0687f94ee32378d`。
- arm64 二进制：`7d516f185e6ab1182beee76ed4c17f4ff7b6c2b2b6b2251fdc1e9f5aa41a72dc`。
- 多架构摘要：`sha256:b3a9f9ca1be205efb3d6d147ca7162a654bf1d66667f7cc2b79128d50bcdf72e`。
- 现场 amd64 image ID：`sha256:f5f66fd530e3b3c71ad803f6ee565670c367ba9968d77d54bb7d666df188d573`。
- 镜像 revision 标签与上述发布提交完全一致。

### 备份与配置保留

备份作业 `9f5c1b45-8bf9-49cd-a94a-28afb4ef094f` 成功，目录为：
`/home/docker/zzapi/backups/pre-328-20260919T063324Z`。
保存 PostgreSQL 自定义格式备份、Compose、环境文件、模块归档、29 个模块文件摘要
及脱敏容器身份快照。数据库备份 47,986,057 字节，`pg_restore --list` 通过。

- 数据库 SHA-256：`b4845a29a82fcfca22f74f7dd0b61f4ed226c2b4c698cc9fff4c0c48fbe25912`。
- Compose SHA-256：`eabd710a647dbf98c6c2a5b51b8ff3c526010979cd05424cc0c828e7229f936e`。
- 模块归档 SHA-256：`527d3359f1b5419eb7b1d51ea5f7e5c88a05a7ea94cce3f8327f647fcc1961bb`。

镜像与配置准备作业 `e09cf028-2c63-4278-b25a-68a1479131ba` 通过 YAML 节点定位
仅替换三个 image 值，再用 Compose 规范化配置比较，证明其他配置保持。
保留原角色：主节点未设置 `NODE_TYPE`，两个从节点为 `slave`。

### 滚动更新与检查修正

- `zzapi-slave-1` / 18098：更新作业 `447bf761-73b0-42e9-957f-71c4d649945d`。
  容器健康和版本通过，但旧校验器把环境变量数组顺序变化识别为配置差异，因此暂停后续更新。
  只读核对 `0d7673e0-59b2-4f4e-8638-b501402df6a1` 证明环境变量值、挂载和端口未变。
  改为对照备份 Compose 和旧镜像的结构化比较，复验作业
  `77175b84-9c02-450b-8bc6-3a12da516720` 通过；没有再次重建该节点。
- `zzapi-slave-2` / 18099：`497ac813-4731-42f4-ae7e-ba5845bf21f0` 成功。
- `zzapi` / 18097：`cd497ebd-66f4-44dd-bb4d-797c77665859` 成功。

### 最终验收

最终作业 `e9720ef0-dbe1-4486-8d8e-ecd57bd5fe36` 成功：

- 三节点实际 image ID 一致，均为 `.328`、healthy、restart=0；原环境变量值、挂载与端口保持。
- 公网 `/api/status` 返回 `success=true`、`.328`，本机独立请求也得到同一结果。
- 每节点无身份 `/api/user/self`、`/api/extensions/`、`/v1/models` 均返回 401，
  Classic HTML 返回 200；静态资源作业 `6152b7ea-d70b-4d66-8d76-dc541820fe28`
  验证公网入口脚本 `/assets/index-CvZhUcOZ.js` 返回 200，内容为非空 JavaScript。
- 29 个模块文件的集合与 SHA-256 全部不变。
- PostgreSQL 和 Redis 的容器 ID、启动时间、运行状态和重启次数与更新前相同。
- 各节点启动后日志未出现检查范围内的 panic、FATAL 或 `[ERROR]` 标记。

验收记录保存在 `/home/docker/zzapi/releases/328/verification.json`。
旧镜像和备份保留；未修改 maolaoapi、zhishiapi 或用户、令牌、渠道业务数据。
没有使用后台测试会话触发真实 429，登录态行为以已通过的真实 Axios 回归为依据。
