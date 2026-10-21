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
- 功能提交 `ae3d0ad85a24ae9bf1248cd393ac3729b99d84c2` 保持原 PR #312；CI
  `39644928381` 的前后端检查通过。基于上一轮 zzapi 组合分支追加 UI，组合源码为
  `341edd97157ca726d3c91ef216c8523b2379dc4b`；与旧部署相比仅有 Classic 活动组件、
  测试与文档差异，后端、Default、计费及 relaykit 未变化。
- 本轮采用差量归档传输，并对完整 `git archive` 的 3774 个文件逐个校验 SHA-256。
  首次准备作业因 `VERSION` 的归档 CRLF 与单独提取的 LF 不一致被校验拦截，未开始
  构建或更新应用。改为保留归档原始字节后重新完整校验，再仅在构建目录设置测试版本。
- 新备份目录为 `/home/docker/zzapi/backups/active-cards-v2-20261022`，备份作业
  `cc4e3253-924b-4a7f-bd7e-6141988bb451` 成功。数据库备份 18,212,917 字节，
  `pg_restore --list` 成功，SHA-256 为
  `2301faeba5f00708af8ca8987a538490ba36117e130ea695953b7b6b4169498c`；数据归档
  16,184,228 字节且目录可读。配置与容器快照仅留目标主机，保留上一轮备份及镜像。
- 构建作业 `b65b3ed4-7d98-4164-9815-b9a0b9799259` 成功、exitCode=0，完整源码校验通过。
  测试版本 `zzapi-pr312-ui-341edd97157c`，镜像标签
  `newapi-mao-zzapi:pr312-ui-341edd97157c`，镜像 ID
  `sha256:5577c258890712c8c298c493e01d742c1f35899c6b7ac714652eac7543ff1b3c`。
  amd64、OCI revision 与实际 `--version` 均匹配。
- 前两节点更新作业分别为 `c5b236f1-a797-418d-aeb5-a8b9e101bfed`、
  `02081ec1-9ea6-4063-b1d6-820f8c277b53`，均成功、exitCode=0。节点 healthy、
  restart=0，鉴权、配置、依赖身份及启动错误检查通过；JS/CSS 同时包含新版手机行标记。
- 主节点更新作业 `e023aab6-71fb-4923-a3d2-738499077029` 成功、exitCode=0，并通过
  相同门禁。最终验收作业 `1656c3db-2464-4f44-930c-69823c0a626e` 确认三个节点均为
  新版本、healthy、restart=0；配置、挂载、端口与依赖身份/启动时间保持一致，数据库
  备份校验仍有效。无需回滚，保留上一版镜像与回滚脚本供试用期间恢复。
- 目标主机与本机独立访问公网确认版本 `zzapi-pr312-ui-341edd97157c`、`theme=classic`。
  Classic JS `/assets/index-DLMtWYAZ.js`（11,907,718 字节）和 CSS
  `/assets/index-7yk_Jc_5.css`（789,555 字节）均返回 200，包含紧凑手机行与桌面横卡标记。
  脱敏验收报告保存在本轮备份目录 `deployment.verified.json`。
- 本轮手机与深浅色视觉验收使用实际组件配合演示数据，部署后完成版本、资源、健康和
  未登录鉴权验证；未发起真实供应商收费请求。PR 保持 OPEN、未合并，等待用户试用。

## 第三轮：桌面单卡不再占满整行

用户反馈桌面单个连接卡片过宽。本轮删除单卡横向铺满变体与按数量自动撑满网格的规则；
宽屏固定三列，较窄桌面固定两列，单条只占一个卡位。手机紧凑行、抽屉、数据与后端保持不变。

- 功能提交 `75b832a3a178cdfad378f11b70c25d96626dc3ec`，CI `39654897910` 前后端通过。
  本地活动模块 28 项测试、ESLint、Prettier、差异检查与 Classic 生产构建通过。
- 浏览器使用实际组件和演示数据验证 1920px 下 1/2/3 张均各占 602px，1000px 下两列，
  390px 手机仍为三条 49px 紧凑行，各宽度均无横向溢出。只读代码复核未发现阻断问题。
- 组合源码 `e08463eef82033570e5cd431cee2051163329753` 相对第二轮只变更三个 Classic
  展示文件与两份文档，未覆盖其他 dev 功能。测试版本 `zzapi-pr312-grid-e08463eef820`，
  镜像标签 `newapi-mao-zzapi:pr312-grid-e08463eef820`。
- 新备份目录 `/home/docker/zzapi/backups/active-cards-v3-20261022`；作业
  `24666939-289c-4773-9f6f-af7df3f2973d` 成功，包含六份配置/容器/数据/数据库文件，
  PostgreSQL dump 18,219,704 字节、数据归档 16,184,228 字节。TOC 和归档目录校验通过，
  SHA-256 记录在 `backup.manifest.json`，私有文件仅存于目标主机且权限为 0600。
- 构建作业 `3c319391-b622-4097-a5b5-1cd6a07522bb` 成功；基于已核验的第二轮源码
  覆盖五个变更文件，完整 3774 个文件逐一通过 SHA-256 校验。镜像 ID 为
  `sha256:d4feaa8cfd2a9b7f5e3254cb43d74b7d83d1433dc02778834dc5fc8b98047987`，
  amd64、OCI revision 和实际版本均匹配。首次上传因目标目录尚未创建失败，建目录后
  顺序重传并校验成功，未影响当时运行的应用。
- 按从一、从二、主节点顺序更新，作业分别为 `ba2e55bc-ff95-465b-83be-505306482bde`、
  `dc348061-3103-4b5a-9d9b-a927f6544920`、`13b37692-8c39-44db-8bf7-b3754172795e`，
  均成功且 exitCode=0。各节点 healthy、restart=0，未登录接口保持 401。
- 最终验收作业 `7900eefd-1a3e-4eeb-89f0-c284998b18a2` 成功；三个应用为新版本，
  配置保持一致，PostgreSQL/Redis 身份及启动时间未变，备份文件哈希校验通过。
  目标主机与本机独立验证公网版本及资源，JS `/assets/index-CIFDYZFz.js` 和 CSS
  `/assets/index-DPWCaPH3.css` 已移除宽卡片变体，保留桌面固定列和手机紧凑行。
- 本轮未发起收费推理；布局验证为实际组件演示数据，线上完成基础部署与资源验收。
  PR 保持未合并，未更新 maolaoapi，保留第二轮镜像与回滚脚本。
