# Classic 当前连接：PR 与 zzapi 试用部署

## 授权与范围

2026-10-22 用户要求提交 PR 并先部署 zzapi 供试用。功能 PR 为
[#312](https://github.com/moeacgx/NewAPI-Mao/pull/312)，目标 `custom-main`，保持未合并。
不发布稳定版本，不更新 maolaoapi，不修改数据库/Redis 服务或业务配置。

## 源码与构建

- 功能提交：`70b1ba1c7b7e95093567213f6ed87018703efd8c`，基于稳定基础 `149fe868e`。
- 部署前 zzapi 实际运行：`dev-320e44fcfd0b`，三个节点均 healthy、restart=0。
- 为保留 zzapi 其他 dev 测试功能，单独建立部署分支
  `deploy/zzapi-active-connections-20261022`，以现场源码
  `320e44fcfd0b0b66839ebfa5da2ec14db5ef68d7` 合入功能提交。
- 组合源码：`290045ce6b544c26a1abde30322bd88c0a42bcec`。仅文档目录合并冲突，
  保留两边内容；相对现场源码的 41 个文件均为本功能。原 dev 流统计、超时、计费等
  实现没有被旧版覆盖。未合入当前 dev 上其他尚未部署的修改。
- 使用组合源码 `git archive`，SHA-256 为
  `dea6c82351a7db365b407dfa60b9b354d993cdc393b4c8cb13aad8a7e5c6bb31`；
  在目标主机独立构建目录使用仓库 Dockerfile 构建，不覆盖运行目录。
- 测试版本：`zzapi-pr312-290045ce6b54`；本机镜像标签
  `newapi-mao-zzapi:pr312-290045ce6b54`，OCI revision 指向组合源码。
  只在构建目录写入版本标记，不改变 PR 的 VERSION 或 stable/latest 标签。
- 首次构建在 Classic 阶段因 6 GiB 容器内存限制退出 137，Docker 明确标记
  `OOMKilled=true`。当时三个业务应用保持旧镜像、healthy、restart=0。
  将单次构建内存上限调整为 8 GiB 后复用缓存重试，未修改程序和依赖。

## 部署前验证与备份

- PR CI `39634004604` 的后端 vet/build/test、独立 relaykit、前端 typecheck/test 与
  Classic 构建全部通过；PR 格式检查通过。
- 组合源码的注册表、relay/common、service、controller、router、relay/helper、
  OpenAI/Claude/AWS 适配器全量测试通过，单包 timeout 60 秒；relaykit 独立
  `GOWORK=off go build ./...` 通过。
- 本功能 Classic 25 条测试、原令牌 7 条回归、ESLint/Prettier、生产构建及
  桌面/手机浏览器验收通过。真实供应商调用不由演示页面或本地模拟上游代替。
- CloudSSH 使用项目「API中转站」，serverId=52、hostId=17。现场三个应用共享同一
  SQL_DSN、Redis 和会话/加密配置；实际 NODE_TYPE 为默认、slave、slave，以现场为准。
- 备份目录：`/home/docker/zzapi/backups/active-connections-20261022-0026`。
  包含 Compose、私有环境文件、五个容器快照、数据目录归档、PostgreSQL 自洽备份。
  备份仅留目标主机，限制文件权限，不上传凭据到仓库。
- PostgreSQL 备份 18,202,852 字节，`pg_restore --list` 成功；SHA-256：
  `730c1eeba2149843a1d3522197f2d4f1ceb47b633f123257b2eaff1fa4be11cd`。
  数据归档 16,184,022 字节，tar 目录可读。未执行数据库恢复演练。
- 旧镜像保留：
  `ghcr.io/moeacgx/newapi-mao@sha256:13755d78a8958291cbc2e80235c028f791f0431e09590744201f89f57b3ef35f`。

## 逐节点门禁与回滚

1. 核验构建成功、镜像架构、revision 和版本；部署前再次确认应用未被其他任务更新。
2. 结构化比较 Compose，只替换三个应用的 image。
3. 按 `zzapi-slave-1`、`zzapi-slave-2`、`zzapi` 顺序执行 `--no-deps` 应用更新。
4. 每节点确认 healthy、实际 API 版本、静态资源、未登录鉴权和启动错误后再继续。
5. 确认环境、端口、挂载、PostgreSQL/Redis 容器 ID 和启动时间保持一致。

失败时停止后续节点，使用备份 Compose 与旧镜像恢复已更新的应用，不回写数据库。
活动注册表无需迁移，旧短期 Redis 键会随租约自动过期。
回滚脚本已准备于构建目录 `rollback-apps.py`，先核对容器版本与依赖身份，再恢复应用；
实际部署脚本为同目录 `deploy-node.py`，拒绝覆盖被其他任务改变过的目标容器。

## 实际部署结果

- 构建作业 `d5600e0a-088f-4051-8574-8bac92e2b450` 成功、exitCode=0。
  镜像 ID：`sha256:6edf78fd1c8107b982d17296fd25321aa771e82fd3b084591baf53f67cb5e911`，
  amd64 架构，revision 和 `--version` 均精确匹配组合源码/测试版本。
- 首个节点已升级，初次验收因 Docker 环境变量数组排序变化而中断。后续按键和值核对
  确认无新增、删除或变更环境变量，调整验收脚本为忽略顺序的精确比较，通过只读复验，
  没有重复重建该节点，也没有放松配置值校验。
- zzapi-slave-1 只读复验作业 `34bae7bd-0e6e-4942-9595-880739855a98` 成功。
  healthy、restart=0，新活动状态与令牌筛选接口在无登录身份时均返回 401。
  Classic 资源 `/assets/index-BtlLBmig.js` 为 11,904,921 字节，包含活动接口和筛选功能。
- zzapi-slave-2 更新作业 `71b92831-6ab3-47e8-b6a2-1af7ceae31dd`、zzapi 更新作业
  `37d446a6-c0c8-4c83-a7c9-98ab903aac71` 均成功、exitCode=0，并通过同样的节点门禁。
- 最终验收作业 `0cc8aa50-d11c-4f88-86e0-2263e563b0a0` 成功：三个节点均使用上述
  镜像、healthy、restart=0，环境变量内容、挂载和端口保持一致；PostgreSQL/Redis
  容器 ID 与启动时间未变，数据库备份校验仍有效。
- 目标主机及本机独立访问公网 `/api/status`，均确认为 `zzapi-pr312-290045ce6b54`、
  `theme=classic`；公网使用日志页面及相应 Classic JS 返回 200，新活动接口和
  `active_only=true` 标记存在。脱敏验收报告保存在备份目录 `deployment.verified.json`。
- 本次不执行收费推理，不声明已完成真实供应商活动连接或登录态多节点业务验收。
  用户可在 zzapi 发起持续流式请求，查看使用日志和令牌页；观察可能延迟约 4 秒，
  短请求可能不出现在快照。PR 保持 OPEN、未合并，等待用户试用。

共享契约与两个兄弟项目的分享建议见[功能专题](../../developer/active-requests.md)。
本次不向兄弟项目发送 Issue、PR 或通知。

## 第二轮：连接卡片与手机紧凑行

用户确认第二版视觉方案，并要求手机一行一条。本轮仅修改 Classic 活动组件、CSS、
对应测试与文档，不修改后端。桌面单条横向、多条最多三列，抽屉一列；手机首页最多
三行，抽屉每页十行，点击展开完整信息。

- 旧版先出现三项预期回归失败，当前活动模块 28 项通过；ESLint、Prettier、
  `git diff --check` 和 Classic 完整构建通过。
- 390px 手机实际每行约 49px；320px 下验证长模型名、86400 秒级耗时，展开指标
  使用纵向紧凑排列，未产生横向溢出。模型图标由模型名推断，与入站协议独立。
- 保持原 PR #312，基于上一轮 zzapi 组合分支追加本轮 UI。构建和部署结果待完成后补充。
