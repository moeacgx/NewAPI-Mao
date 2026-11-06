# .366 令牌分组默认不限制发布

## 目标与范围

用户要求默认不限制令牌分组数量后再次发布。本次版本为 `v1.0.0-rc.10.1.10.366`，接续 `.365`，仅调整 `MaxTokenAutoGroups` 默认值及零值语义，覆盖后端、Default、Classic 和验证文档。实现契约见[默认不限制](06_unlimited_token_groups.md)。

发布后延续本轮已授权的 zzapi 更新范围，逐节点更新三个应用容器；不变更 maolaoapi、数据库服务、Redis、认证、挂载和端口。功能写入验证使用本地隔离 SQLite，不创建生产测试账号或消耗真实模型额度。

## 验证与发布计划

1. 验证后端配置、令牌创建/编辑、鉴权和 Auto 快照；验证 Default、Classic 的设置及选组交互、翻译、静态检查和生产构建。
2. 实际 HTTP 冒烟验证新站点默认不限制、七组令牌完整保存和鉴权、正数上限拦截、恢复 0 后同一密钥可用，以及无效配置不污染已保存值。
3. 通过 PR 和 CI 合并准确源码，为 `.366` 创建新附注标签，不覆盖旧标签。
4. 核验 Linux amd64/arm64 二进制、SHA256、构建版本及 revision，核验 GHCR 多架构清单和镜像版本。
5. 用 CloudSSH 核对 zzapi 当前版本与拓扑，备份 Compose、环境、容器快照、数据库和数据目录；核验备份后逐节点更新，检查健康、版本、未登录权限边界及静态资源。

## 升级与回滚

不新增表、不迁移历史令牌或已保存配置。未保存 `MaxTokenAutoGroups` 的站点升级后使用默认 0；显式保存过的正整数继续生效。当前 zzapi 的实际配置应在部署前再次只读核对。

若新版本曾保存 0，回滚旧版前应先在新版本中设置受旧版支持的正整数；已有超限令牌可能需要调整。滚动更新期间不要保存旧实例不支持的 0；所有节点升级完成后再启用新配置。只回滚应用，不自动恢复数据库或覆盖新业务记录。备份校验不等于恢复演练。

## 共享契约与分享评估

`GET /api/token/auto-groups` 的 `max_count=0` 新增“不限制”语义，外部调用方若自行限制选组数需同步理解；请求格式、稳定分组 ID/code、模型标识、鉴权凭据及计费不变。

NewAPIForDouDi：如果存在固定多分组数量限制，适合参考零值无限制的配置契约，需核对其选组界面及鉴权实现，待目标项目评估。

NewAPIModifyByGang：如果需要多组顺序回退，适合参考可配置正数上限和无限制并存的方式，需核对 Auto 快照语义，待目标项目评估。未向两个仓库发送通知。

## 验证与发布结果

本地后端六包相关回归 428 项、Default 六文件 53 项、Classic 三文件 40 项与语言编码 1 项、实际 HTTP 24 项检查通过。后端 `go vet`、Default 类型检查和定向 oxlint、Classic 定向 ESLint/Prettier 通过。两套真实生产构建通过（Default 10.6 秒、Classic 58.11 秒）。具体边界和日志见[默认不限制验证记录](06_unlimited_token_groups.md)。

清理过时测试后，主代理重新执行后端相关回归、Default 类型和定向静态检查、设置组件 8 项，全部通过。未做真实浏览器验收、真实上游推理或 MySQL/PostgreSQL 功能测试。

PR #389 已合并，附注标签 `v1.0.0-rc.10.1.10.366` 指向 `b362be9dcb425c08afa5b87810bc81e2d9da0565`。合并源码树与验证过的 PR 提交 `5884ec2b229a1b74bdf58b62b752ad02c7abd403` 一致。PR CI `42160046527`、合并后 CI `42160502882` 均成功。

文档经 oxfmt 格式化，Git 差异检查通过。本次五份文档的 490 个本地链接中，新增链接全部有效；完整检查发现索引已有 81 个失效引用，与本次变更前一致，未扩大范围修改历史索引。

CloudSSH 在个人项目入口 `serverId=46` 核验德国 RS2000（`hostId=17`）的三个 zzapi 节点均为 `.365`，共享 PostgreSQL 与 Redis。部署前只读查询确认数据库未保存 `MaxTokenAutoGroups`，因此升级后的缺省行为为不限制，无需改写生产选项。

备份 `/home/docker/zzapi/backups/release-366-20261106` 已完成，私有权限 0700，包含 Compose、环境、容器快照、约 23 MB PostgreSQL dump、约 16 MB 数据归档。TOC、归档可读性和全部校验和已核验；备份作业 `b4eb0f97-d447-4360-b51d-a84dac44beff`，复核作业 `0fed8f3e-95ee-4572-9bc1-e08d2481c9d1`。备份不代表数据库恢复演练。

Linux Release 工作流 `42160625022` 已成功，正式 Release 已发布中文说明。实际下载 amd64、arm64 两份二进制，本地 SHA256 与 `checksums-linux.txt`、GitHub 资产摘要一致：

- amd64：`5d692f1e4814a5f9d45e5365d891498b8596893abec00a22d81ea2e3b35e8645`。
- arm64：`61c1edf062fc3c87b5ae3bac4434d14dd04a219827fd15a957c96c02baefc443`。

两份二进制的内嵌版本和 revision 均匹配 `.366` 标签，GOOS 为 Linux，架构正确；未在 Windows 执行 Linux 二进制。amd64 的 `vcs.modified=false`，arm64 为 true；工作流先在源码目录生成 amd64 产物，再编译 arm64，与此前版本一致，未把该标记写成 false。

Docker 多架构工作流 `42160625154` 已成功，GHCR 清单包含 linux/amd64 和 linux/arm64。zzapi 主机实际拉取后的 RepoDigest 为 `sha256:0d80892c22b7e1edf191b3e884118a774f787759418d9dc5840879c1916f716f`，amd64 镜像 ID 为 `sha256:5a6b50ce5759ee8d96d3b419950db888d82957c66a3276a73715bc2d91b72ef6`，OCI 版本与 revision 匹配发布标签。

zzapi 已按主节点、slave-1、slave-2 顺序完成更新。三节点均运行上述同一 `.366` 镜像，healthy、重启计数 0；逐一比较环境、挂载、端口一致。PostgreSQL 和 Redis 的容器 ID、启动时间不变。

- 主节点更新与验证：`96d45079-8098-4b5f-aff7-6e2494a1df69`。
- slave-1 更新与验证：`ac30cc18-fca5-4b88-b9d7-890f58986f2d`。
- slave-2 更新与验证：`10839023-91ec-42df-a8f8-fcfa6a9e60ec`。
- 最终公网与配置复核：`b0399632-a125-47d8-9224-0222d96573af`。

三个节点的配置、分组、令牌管理接口在未登录时均为 401，页面及八个入口 JS/CSS 正常，启动日志未见 panic 或迁移失败。公网状态返回 `.366` 和 Classic；八个资源 SHA256 与主节点一致，入口脚本包含 `MaxTokenAutoGroups` 和 `Default is 0 (unlimited).`。

最终数据库只读查询仍没有 `MaxTokenAutoGroups` 覆盖值，三个新实例使用默认 0；没有改写生产配置或历史令牌，没有创建生产测试用户/令牌，没有消耗真实上游额度。部署健康与公网资源检查不替代业务 UI 的真实浏览器验收。
