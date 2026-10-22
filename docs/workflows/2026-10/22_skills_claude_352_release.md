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

以上为本地及 PR #318 验证，不代替最终整合 PR CI。发布结果以标签工作流及 GitHub Release 为准，未执行正式版主机部署。
