# NewAPI-Mao 项目身份与发布兼容

## 目标与边界

本项目名称为 **NewAPI-Mao**，GitHub 仓库为 `moeacgx/NewAPI-Mao`。
上游为 `QuantumNous/new-api`，保留上游名称、版权、许可证、作者归属和 Go 模块路径。
`maolaoapi.com` 是站点品牌；`maolaoapi`、`zzapi`、数据库、卷、节点、认证及缓存命名空间是运行标识，不随项目更名迁移。
历史工作记录中的仓库地址、部署镜像和文件名保留原始证据，当前入口以本文为准。

## 源码与页面

- Default 和 Classic 的 HTML 标题、默认站点名、关于页项目仓库、反馈入口使用新身份；已有管理员设置的 `SystemName` 继续生效，不自动改数据库。
- Electron 的产品名和窗口标题使用新身份；既有应用 ID、用户数据目录和 `new-api.db` 保留，防止升级后出现空数据库。
- 配套仓库 `moeacgx/maolaonewapi-extensions` 和 `moeacgx/maolaonewapi-plugins` 独立维护；子模块路径、目录和市场地址不改名。
- `New API` 渠道类型、扩展协议、请求头、加密用途字符串、客户端接入 ID 和通用模块路径属于兼容契约，保留原值。

## 更新与构建契约

- `GET /api/status/github-latest-release` 与 `POST /api/status/self-update` 保持原路径和 Root 权限。
- 默认 Release 仓库改为 `moeacgx/NewAPI-Mao`；`SELF_UPDATE_REPO` 优先于 `SELF_UPDATE_GITHUB_REPO`，显式覆盖仍有效。仅对本仓库旧名 `moeacgx/maolaonewapi` 和新名执行大小写不敏感匹配，统一映射至规范的 `moeacgx/NewAPI-Mao`，保证 GitHub 返回新资产 URL 时通过校验；其他仓库不映射，HTTPS、域名和仓库路径限制不变。
- 继续生成 `new-api-<tag>`、`new-api-arm64-<tag>` 和 `checksums-linux.txt`，容器入口仍为 `/new-api`，确保旧版自更新器和现有启动命令可用。
- 标签镜像流程从 `GITHUB_REPOSITORY` 转小写，发布至 `ghcr.io/moeacgx/newapi-mao`；源码元数据仍使用大小写正确的 GitHub 仓库地址。
- 手动分支流程保留上游 `calciumion/new-api` Docker Hub 目标与凭据约定，并仅允许 `QuantumNous/new-api` 执行；本仓库中该流程跳过，不声称自有分支镜像可用。

## 镜像迁移策略

当前 README 与 Compose 可执行部署默认采用已发布的 `ghcr.io/moeacgx/maolaonewapi:latest`，这是 NewAPI-Mao 更名前的兼容镜像路径。未来发布目标为 `ghcr.io/moeacgx/newapi-mao`，完成首次发布与公开拉取验证后才能修改安装默认值。
GitHub 仓库重定向不等于 GHCR 包迁移；旧镜像 `ghcr.io/moeacgx/maolaonewapi` 不会自动获得新标签。
本次仅准备新路径发布配置，不推送镜像、不删除旧包、不更新任何运行中的部署。
暂不双发旧镜像；若需要让旧路径继续接收更新，须由发布协调者决定兼容周期、包权限、签名和同一 digest 校验后再实施。
现有部署继续固定旧镜像和 digest；切换新路径前应先验证新包可拉取、两种架构 manifest、健康检查和回滚镜像。

### 已部署旧程序的首次升级

本次别名兼容只在包含该修复的新二进制中生效，已经部署的旧二进制不会自动得到修复。旧程序若仍将旧 slug 用于严格资产 URL 校验，更名后可能出现“检查更新成功、下载被拒绝”。首次跨更名升级需另行授权维护：将进程环境中的有效 `SELF_UPDATE_REPO` 改为 `moeacgx/NewAPI-Mao` 并重新创建/启动应用，或直接安装经过校验的新版二进制/镜像。仅改低优先级的 `SELF_UPDATE_GITHUB_REPO` 不能覆盖已设置的旧 `SELF_UPDATE_REPO`。本任务没有执行这些线上动作，不能据此认定生产自更新已修复。

## GitHub 更名前后的 CI 风险

1. 大写 `github.repository` 直接作为镜像名会导致 Docker reference 无效；标签流程每个独立发布 job 均须完成小写归一。
2. 新 GHCR 包需要正确的 `packages: write`、仓库关联和可见性；旧包不会重命名。首次发布须单独验收拉取权限。
3. 历史标签含旧 workflow；重跑旧标签可能继续使用旧逻辑。应从包含本次修复的新提交发布新标签，不能用旧标签重跑代替验收。
4. OIDC/cosign 的仓库身份会变化，外部验证策略如固定旧仓库路径须由维护者调整。
5. GitHub Release/CI 使用当前仓库上下文；外部 GitCode 镜像、仓库变量、保护规则、Webhook、Paseo、Vault 和用户级工具由协调者核查。
6. 用户提供的公开 registry 验证：旧路径 `latest` 匿名 HEAD 返回 200，digest 为 `sha256:dc3f5308ea5394a117ae64ed8abbc9242371e5851353912862789b2522f5eddb`；新路径匿名 token/拉取返回 403，当前不能作为公开拉取入口。403 不证明包不存在。本任务未重复探测或发布镜像。

## 验证计划与交付规则

先盘点仓库与可见身份，再修改；检查全仓残留、自更新测试、双模板类型/lint/构建、workflow 语法及本地链接。
Go 测试使用 `-timeout 60s`。不为纯名称常量增加机械测试；复用既有更新和页面行为测试。
实际验证结果记录在[更名工作记录](../workflows/2026-09/21_newapi_mao_identity.md)。

每次功能交付分别评估 `NewAPIForDouDi`、`NewAPIModifyByGang` 是否值得采纳，向用户提醒理由；没有具体提交授权不发送 Issue。本次纯身份更名无需兄弟仓库采纳。
