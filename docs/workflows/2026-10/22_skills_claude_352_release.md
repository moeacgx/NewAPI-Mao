# .352 技能中心与 Claude 结算整合发布

## 目标与范围

将 zzapi 已验证的双模板技能中心、合集分类、多来源导入、仓库技能商店和操练场令牌切换整合到最新 custom-main；同时合入 PR #318 的 Claude 空流与客户端取消结算修复。保留管理员用户连接查询、连接结束保留与静态资源缓存修复。

## 合并与兼容边界

采用三方合并保留双方新增业务与翻译键，不用旧树覆盖主分支。分组继续以稳定 ID 关联、当前名称展示。技能只装载文本，不执行脚本和工具；普通用户不能读取正文。ZIP/公开 GitHub 导入受大小、路径及网络限制，导入为管理员可编辑的未发布站点副本。

Claude Messages 转 Responses 的成功流在客户端取消后最多继续读取 180 秒实际用量，以实际返回用量结算；没有用量则退款。此期间上游可能继续产生费用。其他协议和错误路径保持及时取消。无费率及历史账务改写。

技能迁移新增表、字段与索引，按 SQLite/MySQL/PostgreSQL 兼容设计；zzapi PostgreSQL 已验证，未实跑 MySQL。PR #318 不增加数据库迁移。

## 验证与发布计划

- 在最新主分支组合运行后端测试、独立 relaykit 构建、双模板相关测试与构建，最终以 PR CI 验证 Linux 环境。
- 合并前重新 fetch 并确认未落后，代码通过 PR 合入 custom-main。
- VERSION 与标签统一为 v1.0.0-rc.10.1.10.352；标签自动触发 Linux amd64/arm64 二进制及 GHCR 多架构镜像发布。
- 发布后核对 Release 资产校验和、镜像架构与 latest 摘要，并检查在线商店索引和正文哈希。
- 本项不执行 maolaoapi 或 zzapi 正式版部署；保留上轮 zzapi 测试镜像。

## 已完成验证

- PR #318 已于 2026-10-22 合入 custom-main，合并提交 `a92dd2ff6bf48e2f154b773cd5f35a1b665ad73e`；[同步后 CI](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39768934075) 全部成功。
- 技能组合的 model/controller/service/router/middleware 测试通过；加入 Claude 修复后，service/controller/channel/claude/helper/relay 再次完整测试通过，均设置 `-timeout=60s`。
- `relaykit` 在 `GOWORK=off` 下独立构建通过。
- Default 类型检查、58 项相关测试、生产构建通过；Classic 42 项相关交互测试、3 项原生鉴权测试、生产构建通过。
- 三方合并保留主分支全部翻译键及连接管理、静态资源缓存、扩展兼容行为；未包含凭据、截图或测试数据库。

以上本地验证已由下述最终整合 CI 补充确认。发布结果以标签工作流及 GitHub Release 为准，未执行正式版主机部署。

## 主分支与在线商店验收

- PR [#320](https://github.com/moeacgx/NewAPI-Mao/pull/320) 已合并，最终发布源码为 `3cf99ca4e3f334ba3ceda3663d5651f79a42ebce`，合并前 [CI](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39769907953) 与 PR 检查全部成功。
- 合并后对主分支再次执行的 [CI](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39770381221) 也全部成功，覆盖根模块和独立 relaykit 全量 vet/build/test、Default 类型与全量测试及 Classic 构建。
- 标签 `v1.0.0-rc.10.1.10.352` 指向上述提交。Linux 发布流程为 [39770492026](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39770492026)，Docker 多架构流程为 [39770491976](https://github.com/moeacgx/NewAPI-Mao/actions/runs/39770491976)。
- 固定公开主分支在线商店已真实读取：3 个条目、5 份 Markdown 原始字节及 SHA256 与 Git 一致。实际调用 `ListSkillStore(ctx, true)` 和全部 `ImportStoreSkill` 成功，返回未发布、无分组的规范草稿；未初始化或写入数据库。
- 发布与部署分开：本轮不改运行中的 maolaoapi/zzapi，也没有调用真实收费供应商。

## 发布产物验收

- Docker 发布流程全部成功，固定标签为 `ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.352`。
- 多架构索引摘要为 `sha256:68c51c9d134868b68568e9ec39bab1a62bd61d6fae006b40d2a0352bc6bd7009`，实际包含 `linux/amd64` 与 `linux/arm64`。
- 两个架构镜像均已拉取，在断网、只读容器中执行 `--version`，均返回 `v1.0.0-rc.10.1.10.352`；未连接业务数据库。
- 匿名读取确认 `latest` 与固定标签摘要一致；两架构均附 SPDX SBOM 与 SLSA v1 来源记录，源码 revision 与发布提交一致。cosign v3 签名对象存在且 subject 匹配，工作流签名成功；本次未独立执行密码学验签。
- [GitHub Release](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.352) 已正式发布并设为 Latest，Linux 与 Docker 两条发布流程全部成功。
- 两个 Linux 二进制实际读取后 SHA256 均与 `checksums-linux.txt` 及 GitHub 资产摘要一致；均在对应架构的断网、只读容器中执行 `--version` 返回 `.352`。
  - amd64：`7b9cb8aa890c28111fbeb12f0c89cfb7aa19f298b197d82c3b4972e73f1a45a5`。
  - arm64：`fc781c5050ccdf41d403b6cdc73a30c16162660f1bae7bed407d9272f328eb7b`。
- ARM64 发布步骤耗时偏长的实际原因是 Ubuntu apt 交叉编译依赖下载重试，45.2 MB 下载耗时 14 分 39 秒；随后编译与发布成功，未取消、重跑或改写标签。

## 跨项目边界与分享建议

- 对接方可继续使用原密钥、模型 ID、路由与 usage 契约；仅新增管理 API、令牌摘要计数和操练场本人令牌身份。Claude 取消结算语义影响实际扣费，窗口与退款边界见上文。
- NewAPIForDouDi：建议评估提示词装载与令牌徽标，需先核对其分组、预扣及前端结构，不能直接照搬迁移。
- NewAPIModifyByGang：技能合集和管理员商店导入有分享价值，需核对其模板权限、分组与网络导入边界。以上均为建议，未创建 Issue、PR 或发送通知。
