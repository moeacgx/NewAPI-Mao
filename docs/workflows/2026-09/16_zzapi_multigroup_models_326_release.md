# zzapi 多分组模型列表修复 `.326`

## 目标与范围

用户授权审查、创建 PR、合并 `custom-main`，发布并更新 zzapi。
修复令牌绑定多个显式分组时模型列表错误返回空数组；使用鉴权已授权的分组列表合并模型。
OpenAI、Anthropic、Gemini 及 Gemini OpenAI 兼容入口共同生效；保留模型白名单、定价和分组权限过滤。
根因及测试见[多分组令牌模型列表回归](16_token_multigroup_model_list.md)。

代码修复 `fc197ceee` 已由独立 Agent 复审通过，10 项真实 TokenAuth 路由回归独立复跑通过。
协调者重新执行相关 router/controller 测试和 vet，通过；此前完整 router 测试与根应用构建通过。
发布前仍须固定最终 PR head、核对最新主分支和 CI，并等待 Linux 与 GHCR 发布工作流成功。

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

发布和部署完成后追加工作流、镜像摘要、备份与节点验证结果。
