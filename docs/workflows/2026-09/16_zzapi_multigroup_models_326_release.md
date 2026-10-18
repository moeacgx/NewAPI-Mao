# zzapi 多分组模型列表修复 `.326`

## 目标与范围

用户授权审查、创建 PR、合并 `custom-main`，发布并更新 zzapi。
修复令牌绑定多个显式分组时模型列表错误返回空数组；使用鉴权已授权的分组列表合并模型。
OpenAI、Anthropic、Gemini 及 Gemini OpenAI 兼容入口共同生效；保留模型白名单、定价和分组权限过滤。
根因及测试见[多分组令牌模型列表回归](16_token_multigroup_model_list.md)。

代码修复 `fc197ceee` 已由独立 Agent 复审通过，10 项真实 TokenAuth 路由回归独立复跑通过。
协调者重新执行相关 router/controller 测试和 vet，通过；此前完整 router 测试与根应用构建通过。
最终 PR head、最新主分支、CI 和 Linux/GHCR 发布检查已完成，结果见执行记录。

## 部署与回滚

目标：CloudSSH `API中转站 / RS2000 德国建站`，serverId=52、hostId=17，目录 `/home/docker/zzapi`。
现场预检作业 `968e4b09-bcca-40be-8035-35da157fe767` 确认三节点 `.325`、healthy、restart=0，
端口为 18097、18098、18099。镜像与 Compose 服务名均与目标一致。

先备份 Compose 和 PostgreSQL，保留 `.325` 镜像。拉取固定 `.326` 并核对摘要，
仅逐节点重建 `zzapi`、`zzapi-slave-1`、`zzapi-slave-2`，每节点健康与版本通过后才继续。
PostgreSQL、Redis 不重建，不改变插件、用户、令牌、渠道或计费配置；maolaoapi、zhishiapi 不在范围内。
任一节点失败时停止后续更新并恢复该应用旧镜像，不回写业务数据库。

## 验收边界

检查实际运行 image ID、健康状态、重启次数、三端口与公网 `/api/status`，
验证数据库与 Redis 容器 ID、启动时间保持不变，并检查模型列表无身份请求仍被拒绝。
本次无数据库迁移、不调用付费供应商。真实多分组行为由隔离数据库上的完整鉴权路由回归验证；
没有用户提供的专用测试令牌时，不读取已有用户密钥或宣称完成线上认证模型列表验收。

## 执行记录

修复及版本 PR [#230](https://github.com/moeacgx/maolaonewapi/pull/230) 已合并，
合并提交 `85959f3b39fbb9737413dbe75015380230984c7f`，标签 `.326` 固定指向此提交。
远端 PR head `5142fc38e22108138a4a68c6e99e7dda794db402` 与本地验证文件树完全一致，
tree 为 `5423a55706db77c840d6c156101b7e7e0609e501`，合入前 behind=0。

CI `35102149419` 首次前端作业在安装 Classic 依赖时出现 `mermaid` 包解压失败，
当次类型检查及 654 项前端测试通过，后端检查通过。未更改依赖或锁文件，
重跑失败作业后 Classic 构建及所有检查通过；PR 质量检查 `35102149465` 通过。

CloudSSH 备份作业 `e63e6ea7-3088-4610-952d-4da5d311b178` 成功，备份目录为
`/home/docker/zzapi/backups/pre-326-20260916T132913Z`。数据库自定义格式逻辑备份约 122 MiB，
`pg_restore --list` 校验通过；另保存 Compose、已有环境文件和脱敏容器身份快照。
目录校验不代表完整恢复演练，不包含 Redis 数据快照。

- 数据库 SHA-256：`a792ee22e306fa3e82ab86139406d66495e5af2feae9d3bc18a300b903a15481`。
- Compose SHA-256：`7108b0f21391af2c10b0613ab763d02906af78ef90dd16ebf9b93262c588d1a1`。

部署前插件状态作业 `16843e4f-71da-4202-9874-0fef4188d243` 确认现有
`TaskPluginEnabled=true`、10 个插件版本。本次不改变这些状态。

Linux 发布工作流 `35103232425`、GHCR 多架构工作流 `35103232422` 均成功。
Release 包含 amd64/arm64 二进制和 `checksums-linux.txt`，两套前端真实构建产物已嵌入。
镜像准备作业 `5211a40c-7e79-4ae6-b3bc-75322c8bd5f7` 成功，精确替换三个应用镜像，保留旧镜像。

- 多架构 digest：`sha256:a5097e68b262063a5d8bbdf2081c2baa30c78c3942055f5b7476f377e398a6a0`。
- 现场 amd64 image ID：`sha256:ebd127a550d6d7bd62317bc5e5b586193dcdafb50968df836d459b8cd632c42a`。

2026-09-16 完成以下滚动更新，每个作业确认当前节点健康后才开始下一节点：

- `zzapi` / 18097：`4bd3809b-cbbc-4752-8c5e-26735ede0eb2`。
- `zzapi-slave-1` / 18098：`72b6bde6-c46c-4dc8-b83c-e9498cd01f31`。
- `zzapi-slave-2` / 18099：`fcc7386e-7456-4e4a-a400-c95840ee60a7`。

最终复核作业 `a7f9dc86-a4b2-48a0-9a8a-d92077ba28cb` 成功：

- 三应用实际 image ID 与上述镜像一致，均 `healthy`、restart=0，三个端口状态返回 `.326`。
- 公网 `https://zzapi.maolaoapi.com/api/status` 独立返回 HTTP 200、`success=true`、`.326`。
- 三节点 `/v1/models`、`/v1beta/models`、`/v1beta/openai/models` 和插件源管理接口
  的无身份请求均返回 401；未使用真实用户令牌做认证模型列表调用。
- 现有插件管理 HTML 与本地入口脚本均返回 200。`TaskPluginEnabled=true` 与
  10 个插件版本的启用、激活状态逐项保持；既有来源字段存在。
- PostgreSQL/Redis 容器 ID 和启动时间与备份前一致，均运行中，restart=0。
- 应用启动后最后 300 行日志未出现 panic/fatal 或标准 ERROR 标记。

Compose 的既有 `version` 字段过时提示不影响本次更新，未夹带配置清理。
备份和 `.325` 镜像保留；未修改 maolaoapi、zhishiapi、用户令牌或渠道。
外部兼容性变化仅为已授权多分组模型发现恢复，不改变请求格式、模型 ID、计费和实际路由。
