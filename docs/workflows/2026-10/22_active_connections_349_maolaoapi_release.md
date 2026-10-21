# maolaoapi .349 当前连接发布

## 目标与基线

将 Classic 当前连接、令牌实时使用状态及最终固定卡位布局发布为
`v1.0.0-rc.10.1.10.349`，并保留 maolaoapi 已运行的 dev 能力。功能仅为 Classic
新增页面呈现；活动请求接口由共用后端提供，Default 本次不增加入口。

- 生产 dev 基线：`bbb3180d59a75cd12a7a2854a66c02e48551df3c`。
- 功能 PR：[#312](https://github.com/moeacgx/NewAPI-Mao/pull/312)。
- 功能合并提交：`7785c19e98b908d7edbad983a89831a5b2932e83`。
- 整合分支从上述 dev 基线创建，再合入 `origin/custom-main`；业务代码自动合并，
  `docs/developer/README.md` 与 `custom-development.md` 按段保留两边内容。
- 整合提交：`575855e9df8f279b3fb65ec0686aa19c57109003`，PR
  [#313](https://github.com/moeacgx/NewAPI-Mao/pull/313)；创建和合并前均确认相对目标
  `behind=0`。发布合并提交为 `94249a98aacac241512a7ffe68f22421dc58c588`，同时包含
  原 dev 与当前连接两条提交历史。

## 兼容与安全边界

- 当前连接不新增数据库表、迁移、环境变量或持久业务数据；多节点通过现有 Redis
  短租约汇总，旧键会自动过期。
- 请求注册、首响应观察和完成清理不改变路由、重试、用量与计费；生产更新期间只有已
  升级节点可发布活动快照，全节点完成后再验收集群汇总。
- 只更新 `maolaoapi`、`maolaoapi-slave-1`、`maolaoapi-slave-2` 三个应用镜像，
  不重建或修改 PostgreSQL、Redis，不更改端口、挂载、命令和环境变量内容。
- 本次只执行状态、静态资源、鉴权和日志等基础验收，不调用收费模型，也不把基础部署
  验收表述为真实供应商业务验收。

## 发布门禁

1. 整合分支同步最新 `origin/custom-main`，确认 `behind=0`，运行活动模块测试、相关
   Go 回归、独立 relaykit 构建、Classic 生产构建与差异检查。
2. 整合 PR 的后端、前端和审查门禁全部通过后才合并；标签必须指向准确的
   `custom-main` 合并提交。
3. 验证 Release 工作流、Docker 工作流、三个 Release 资产、amd64/arm64 manifest，
   并确认 GHCR OCI revision 等于发布提交。
4. 生产 Compose 固定到已验证的正式镜像摘要，不使用浮动 `latest`。

## 生产备份、顺序与回滚

目标为 CloudSSH 项目“API中转站”的 OVH 兜底主机；执行前重新核对 serverId、hostId、
根目录和三应用实际镜像。备份至少包括 Compose、三应用与 PostgreSQL/Redis inspect、
应用数据目录、PostgreSQL custom dump、`pg_restore --list` 结果和 SHA-256 清单。

节点顺序固定为：

1. `maolaoapi-slave-1`
2. `maolaoapi-slave-2`
3. `maolaoapi`

每个节点必须在 healthy、restart=0、版本/revision/镜像摘要、配置、401 鉴权、Classic
资源标记和启动日志检查通过后，才更新下一节点。任一节点失败即停止，使用备份 Compose
和部署前固定 dev 镜像回滚已更新应用；在线数据库不随应用回滚恢复。

## 验证与正式发布

- 组合源码本地通过
  `go test ./pkg/activerequests ./service ./controller ./relay/common ./model ./router -timeout 60s`；
  relaykit 在 `GOWORK=off` 下独立构建及 `go test ./... -timeout 60s` 通过。
- 组合 CI [39666054190](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39666054190)
  的根模块和独立 relaykit vet/build/test、Default typecheck/test、Classic 构建通过；
  PR 质量检查通过。CodeRabbit 状态显示通过，但其说明为跳过自动审查，不能算独立代码审查。
- 只读协作审查确认业务代码自动合并，活动观察不重置原有计费首响应指标；原 dev 的
  common、model/main.go、relaykit 和发布工作流相对 dev 基线没有被活动功能覆盖。
- 本轮未重复安装本机 Bun；前端组合验证使用上述 CI。Classic 活动模块的 28 项测试、
  手机与桌面浏览器布局验证见原 zzapi 试用记录，整合未改变其代码。
- 修改文档通过现有 Prettier 与差异检查，新增链接有效。README 存在基线失效链接，
  未在本任务扩大修复范围，不宣称全目录链接检查通过。
- 标签 `v1.0.0-rc.10.1.10.349` 的 peeled SHA 精确为上述发布提交，标签推送触发
  [Linux Release 39666902758](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39666902758)
  和 [多架构 Docker 39666902615](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39666902615)，
  两者均成功。
- [GitHub Release](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.349)
  已发布，非草稿；包含 amd64 二进制 159,834,377 字节、arm64 二进制 154,599,689 字节
  及 198 字节的 `checksums-linux.txt`。下载校验文件的 SHA-256 与 GitHub 资产元数据
  一致，其两条二进制哈希也与对应资产元数据一致。
- GHCR 多架构固定摘要：
  `ghcr.io/moeacgx/newapi-mao@sha256:7d864896c6263c92b835b38f614f0820a7fd10144aa6f669f4870d9f3d364ee7`。
- amd64 manifest：`sha256:ac592fd82cae9a26c8bf68653638874c1b92ad0c8c3c369011fbb19346690c53`；
  arm64 manifest：`sha256:da5d1366be843ac3fb895e8ee6e20150f2e08bc46067705b239574d969bfb9bf`。
  两个平台的 config 中 OCI revision 均为发布提交、version 均为 `.349`。

## 生产准备与备份证据

- 现场复核作业 `5d2ac896-8dcb-434a-ace0-27581902f16a` 确认 CloudSSH 项目
  `API中转站`、`serverId=63`、`hostId=29`，根目录 `/apps/maolaoapi`，Compose 为
  `docker-compose.json`，项目名 `maolaoapi-ovh`；三应用端口为 18100、18101、18095。
- 更新前三应用均为 `dev-bbb3180d59a7`、healthy、restart=0；旧固定镜像为
  `ghcr.io/moeacgx/newapi-mao@sha256:4ab0e8a70a09124b8343cd819e29678f739287943bbdcb78a50b6a46028a860c`，
  已确认本地主机仍可解析此引用。没有使用旧生产主机或旧目录。
- 备份作业 `cf92d1b7-8ebb-4b97-99a8-e1a391c7cfd9` 成功、exitCode=0，目录为
  `/apps/maolaoapi/backups/pre-349-20261021T201016Z`。目录名使用 UTC，对应北京时间
  2026-10-22；包含 Compose、五容器 inspect、data 归档、数据库 custom dump、TOC 和清单。
- PostgreSQL dump 为 1,575,513,167 字节，`pg_restore --list` 成功，SHA-256：
  `d018de769c2c7902be32dc4637a6f5cb77f2e215dd55d6bc6d27dcdc30974808`。
- data 归档为 88,640,019 字节，tar 目录校验通过，SHA-256：
  `ac72e7a85f6ae9f431eef89e7efbbad291ec224b109b94ed77a33c020c4ce2ec`。
  私有配置和 inspect 仅留主机，文件权限 0600、备份目录 0700；未执行数据库恢复演练。
- 首次 Docker 拉取收到 registry `denied`，此时未重建任何应用。匿名 manifest 读取
  正常；改用发布目录内独立空 Docker 配置进行匿名拉取后成功，原登录配置保持不变。
  镜像验证及拉取成功作业为 `9c9746cc-0e0a-487d-8277-1d0f8064c1c1`。
- 节点脚本检查完整有效环境变量、挂载、端口、命令、重启策略、资源限制和依赖身份。
  镜像默认环境与旧镜像完全相同；固定网格验收定位到活动卡片选择器及对应媒体查询。
  运维脚本与应用回滚入口保存在 `/apps/maolaoapi/releases/349/`。

## 逐节点与最终验收

- 第一从节点更新作业 `8f41caa2-f7f9-458a-86ce-94a9581b7212` 成功、exitCode=0。
- 第二从节点更新作业 `24dfd67e-cf75-4c39-9d78-d338a3ad7f33` 已换镜像，但在挂载
  数组比较时停止。只读检查 `1de3d173-a907-4c7b-8c39-569b8900e3d9` 确认仅两条
  Binds/Mounts 的顺序改变，路径、权限和传播模式均未变化，节点 healthy、restart=0。
  修正验收为忽略顺序的完整内容比较后，只读复验作业
  `eb0f2e4b-5e86-4ce0-9330-4a197c4aa4ff` 成功；未重复重建第二从节点。
- 主节点更新作业 `b1657ad7-77c7-489e-bff5-f8e89ba4b011` 成功、exitCode=0。
  更新期间逐节点等待停止/启动与健康检查，没有同时重建三应用。
- 最终验收作业 `e6e980e4-640e-4253-ad82-4e0c48f6b649` 成功、exitCode=0：三应用
  均为正式镜像、`.349`、Classic、healthy、restart=0；有效环境变量无变化，挂载、端口、
  命令及资源配置保持，PostgreSQL/Redis 容器 ID、启动时间和重启次数保持。
- 每节点匿名 `/api/log/self/active`、`/api/token/search?active_only=true`、
  `/api/user/self` 和 `/v1/models` 均返回 401；未发现 panic、fatal、迁移失败或
  数据库连接失败标记。备份文件再次逐一通过 SHA-256 校验。
- 目标主机与本地独立访问公网，均确认 `/api/status` 返回 200、`.349` 和 Classic，
  使用日志入口与 JS/CSS 返回 200，四个匿名接口保持 401。首次默认 Python User-Agent
  收到 403，改用普通浏览器 User-Agent 后复验成功；未修改站点访问策略。
- 公网和三节点 JS 为 `/assets/index-CIFDYZFz.js`（11,907,634 字节），CSS 为
  `/assets/index-DPWCaPH3.css`（788,759 字节）；活动接口、令牌筛选、手机紧凑行和
  桌面固定列标记存在，旧的单卡铺满变体不存在。
- 最终脱敏报告为备份目录内 `deployment.verified.json`；旧镜像、备份和
  `/apps/maolaoapi/releases/349/ops.py rollback` 保留，未实际执行回滚。
- 本次未调用收费供应商，不声明完成登录用户真实流式请求、多物理主机故障或压力验收。
  原 dev 功能以提交历史包含、业务差异核对和组合回归证明保留，不能替代每条功能的
  生产业务验收。

## 共享契约与分享建议

本次新增本人活动状态只读接口与可选令牌筛选条件，原转发、鉴权、用量和计费请求契约
保持兼容；未发现必须让兄弟系统同步改动的共享契约。

- NewAPIForDouDi：值得评估分享活动连接概览与手机紧凑展示，需要适配其模板及请求
  生命周期；本次未审查目标源码，不认定可以直接移植。
- NewAPIModifyByGang：值得评估分享令牌活动筛选和多节点租约，需要核对其 Redis
  与令牌搜索契约；本次未审查目标源码。
- 本次未向兄弟项目创建 Issue、PR 或发送通知。
