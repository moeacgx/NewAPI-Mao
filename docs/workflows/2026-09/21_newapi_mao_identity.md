# NewAPI-Mao 源码身份更名

## 范围与方案

基线 `2d7f427bf5a888fee49ff4baa0f6d0f5c83f57fd`，专属分支 `chore/rename-newapi-mao`。
按用户授权仅修改隔离源码仓库；GitHub 仓库已由协调者改名并更新共享 origin。
盘点覆盖 README、开发文档、项目规则、双模板、后端默认名称、自更新、Electron、CI 与部署示例。
完整契约、保留项与发布风险见[项目身份](../../developer/project-identity.md)。

## 发现

- 标签镜像原先直接拼接 `github.repository`，更名后大小写不符合 GHCR 要求。
- 手动分支镜像原先仍推送上游 `calciumion/new-api`。
- 自更新默认仓库和 Classic 详情回退指向旧仓库；多数 README 安装命令指向上游。
- 前端、后端和 Electron 默认标题仍使用上游通用项目名。

## 验证

- `go test ./common -count=1 -timeout 60s` 通过。
- `go test ./common ./service -run 'Test(SelfUpdate|ValidateSelfUpdateRepo|ReleaseAssetDownloadURL|ValidateGitHubDownloadURL|ChecksumForAsset)' -count=1 -timeout 60s` 通过；匹配的既有自更新测试在 service 包，common 在此命令无匹配用例。
- `go build ./...` 通过。未修改独立 relaykit 模块及其 API。
- 两套前端均使用 `bun install --frozen-lockfile` 安装并完成 `bun run build`；Default `bun run typecheck` 通过。本机 Bun 通过 `npx --yes bun` 调用，未全局安装。
- Default 既有 public-header、canvas/lib、canvas/selection 三个测试文件共 8 项通过；Classic `node --test web/classic/src/self-update-release-link.test.mjs` 通过。
- Classic 修改文件 ESLint 通过。Default 修改文件除 footer 外 oxlint 通过；footer 仍有原有两处 `react/no-array-index-key`（第 276、282 行）和一处 `no-danger` warning，仅改默认名称，相关列表实现未动。不将其报告为全量 lint 通过。
- actionlint `v1.7.11` 检查三份修改 workflow 通过（未安装 shellcheck/pyflakes，显式关闭对应外部检查）。
- `node --check electron/main.js` 通过；Electron 未打包和运行安装版，保留目录的跨系统升级仍需发布前验收。
- 前端新身份文案经过翻译脚本处理，Default 七语和 Classic 八份 locale JSON 均已更新；执行 `bun run i18n:sync`，旧通用翻译键保留，避免影响上游名称的其他用途。
- GitHub 只读查询确认当前仓库为 `moeacgx/NewAPI-Mao`、默认分支 `custom-main`，最新 Release URL 已使用新仓库路径，标签仍为 `.331`。
- 新开发专题与工作记录使用仓库 Prettier 检查；既有 Markdown 的全文件格式化会产生大量无关差异，因此保留原有布局，仅修改任务相关内容。

## 最终范围与保留项

覆盖六份 README、项目 AGENTS/CLAUDE/i18n Skill、Issue/PR/安全反馈入口、开发文档、Default/Classic 标题与文案、后端默认名称和反馈链接、自更新、Electron 元数据及数据目录兼容、Compose 镜像示例、三份构建 workflow。

保留 `QuantumNous/new-api`、版权与作者、Go 模块路径、通用 New API 渠道/协议名称；保留生产域名、容器/DB/服务/缓存/认证标识、两个独立配套仓库、历史记录、Release 资产文件名和旧 GHCR 包。标签发布只使用小写新路径，不双发；上游 Docker Hub 手动分支流程在本仓库跳过。

本次没有新增业务功能，`NewAPIForDouDi` 与 `NewAPIModifyByGang` 均无需采纳身份改名；仅登记后续功能交付的分享提醒规则，不发 Issue。

本任务不发布 tag、不发布镜像、不部署、不改数据库；GHCR 新包的权限、拉取、多架构 manifest 和新仓库 OIDC 签名留待首次正式发布验收。旧标签 workflow 不会随本 PR 自动更新。

## 定向复审修复

- 所有 README 安装命令和 Compose 默认使用 `ghcr.io/moeacgx/maolaonewapi:latest`，并就地注明为 NewAPI-Mao 更名前的已发布兼容镜像；新路径只作为后续发布目标。
- 用户提供公开 registry 证据：旧路径匿名 HEAD 200，digest 为 `sha256:dc3f5308ea5394a117ae64ed8abbc9242371e5851353912862789b2522f5eddb`；新路径匿名 token/拉取 403。403 不能证明包不存在。本 Agent 未重复探测或发布。
- 自更新在读取有效环境变量后，仅对本仓库的新旧名称执行大小写不敏感匹配并映射为 `moeacgx/NewAPI-Mao`；严格的资产域名/仓库路径校验未放宽。新增 9 个行为场景覆盖旧主配置和旧备用配置、小写新名称的新规范资产 URL、环境优先级、其他仓库、相似仓库、其他作者和 HTTP 拒绝。
- 新回归先复现旧配置拒绝 canonical URL，修复后 `go test ./service -run 'Test(SelfUpdate|ValidateSelfUpdateRepo|ReleaseAssetDownloadURL|ValidateGitHubDownloadURL|ChecksumForAsset)' -count=1 -timeout 60s` 通过。
- 日语邮件发送者示例修正为完整品牌及邮箱；重新执行 i18n 脚本和同步，扫描本次新增/变更的 148 条 locale 值，内部 `__ PH_0 __` 类占位符残留为 0。
- 已部署旧二进制不能获得本次修复，首次跨更名升级的环境配置和直接安装限制见[项目身份说明](../../developer/project-identity.md#已部署旧程序的首次升级)，本任务无线上修复声明。

## 提交验收

最终本地检查：9 个自更新配置/资产场景及既有自更新测试通过；Classic 更新详情回归通过；新增/变更 Markdown 本地链接缺失数 0；148 条新增/变更翻译值的内部占位符残留 0；两份新文档 Prettier 检查和 `git diff --check` 通过。未重复运行未受后续修订影响的全量检查。

主脑于 2026-09-21 回报 QA `2eca7730-7829-4518-a0fc-83fcca1011cf` 在 18:16:52 最终复查无 P1/P2 阻断，确认 9 个自更新行为场景、Compose 兼容镜像和日语修复。提交推送后交主脑核验 CI 并决定合并，本 Agent 不合并、不发布、不部署。
