# .362 WorkBuddy Responses 缓存结算修复发布

## 目标与范围

本次准备发布 `v1.0.0-rc.10.1.10.362`，基于 `.361` 后的 `custom-main`。发布 GitHub Release 与 GHCR amd64/arm64 镜像，不更新生产容器。

- 纳入 PR #372：强制 Chat→Responses 时把原生缓存用量纳入站内结算，并让 Chat/Responses 双向转换保留 `prompt_cache_key`、`prompt_cache_options` 和 `prompt_cache_retention`。
- 修复只在 Prompt 缓存详情缺失时从 Responses 输入详情补齐；显式零值优先，保持原始 BillingUsage 快照不被结算归一化修改。
- 本次不改变路由优先级、API 响应协议、计费倍率、表达式语法、数据库结构、鉴权或两套前端。

## 兼容、回滚与发布边界

没有数据库迁移。回滚至 `.361` 会恢复 Responses 缓存详情未映射时的历史结算行为，并使 Chat→Responses 再次忽略新增缓存选项和保留时长。本地验证不能代表真实供应商会执行这些缓存策略，也不能证明真实账户命中率。

本次仅发布代码与镜像，不部署生产环境，不重算历史账务。生产使用固定 GHCR digest 和业务验收需另行授权与记录。

## 发布门禁

1. 版本 PR 仅修改根目录 `VERSION`、本记录和开发文档索引；提 PR/合并前刷新 `origin/custom-main`，确认分支无落后，并核验准确提交的 CI。
2. 对版本 PR 的合并提交创建新的附注标签 `v1.0.0-rc.10.1.10.362`，不得移动现有标签。
3. 标签触发 Linux Release 与 Docker 多架构发布。核验两项工作流的精确 SHA 和成功状态、Linux 二进制及 `checksums-linux.txt`、GHCR amd64/arm64 清单与 OCI version/revision。
4. 发布说明与镜像验证完成后记录真实产物；不以标签存在或 CI 成功替代线上业务验收。

## 验证结果

PR #372 合并提交 `49906fa7f5d5b60483a4f8a49a86570d47d55645`，目标为 `custom-main`。修复 CI 对准确提交 `8d434686abfa5f275a65db760ce27200386325f3` 的后端 vet/build/test、relaykit 独立检查、Default 类型检查及前端测试通过；PR 质量检查通过。功能审查未发现阻断项。

版本 PR #373 合并提交 `ff76e6cdceff4f06491554e5cc8eb1729e4e57d5`。附注标签 `v1.0.0-rc.10.1.10.362` 指向此提交。Linux Release 工作流 `41044000708` 与 Docker 多架构工作流 `41044000804` 均成功，准确 SHA 为上述版本提交。

- GitHub Release：[`v1.0.0-rc.10.1.10.362`](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.362)，已上传 amd64、arm64 Linux 二进制和 `checksums-linux.txt`。下载后两二进制 SHA-256 与校验文件完全一致：amd64 `09ea5e19825450f96dd1ca82ca59430943e5deb10508e3d9f8b86b3adf3cb307`，arm64 `4f516b9284e769295b67e15913fd7607be55ab2c11a6881347ea156de158b99b`。
- GHCR 版本与 `latest` manifest 均已发布，SHA-256 `bfb4ed9e3da167e0b1f86499b28418858d46c3ce9a05654c191b08e66ca7c264`，包含 linux/amd64 与 linux/arm64。
- 本轮没有用本机运行镜像内二进制，也没有生产部署、真实 WorkBuddy/GPT 账务路径或供应商缓存命中验收。发布产物验证不等于生产业务验收。

功能实现、本地复现、回归范围与共享契约见 [WorkBuddy 缓存结算与参数修复](30_workbuddy_responses_cache_billing.md)。

## 共享契约与分享评估

本版仅修复已兼容字段的保留与原生缓存用量结算，不新增跨项目接口字段或鉴权/模型契约。

- `NewAPIForDouDi`：若使用相同 Responses 快照结算方式，建议参考 #372 的缺失字段回填与显式零测试；尚未核对其当前实现。
- `NewAPIModifyByGang`：若提供 Chat↔Responses 转换，建议对照其缓存字段透传和计费快照优先级；尚未核对其当前实现。

本次不向兄弟仓库创建 Issue、PR 或发送消息。
