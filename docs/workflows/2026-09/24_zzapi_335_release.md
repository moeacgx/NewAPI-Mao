# zzapi 原生任务插件 .335 发布

## 范围

发布 `v1.0.0-rc.10.1.10.335`，以 `custom-main@46c6ddb4b` 为准备基线。
包含已合并的 PR #265（Classic 供应商 Logo URL）、#266（官方任务插件原生同步能力和 Jev 用量计费）、
#267（网站 SEO 设置）。插件继续从官方源安装，MaoLao 插件源已移除重复 TypeSafe。

用户授权发版并更新 zzapi；不操作 maolaoapi。CloudSSH 项目为「API中转站」，
serverId=52、hostId=17、服务器「RS2000 德国建站」，目录 `/home/docker/zzapi`。
现场预检三个应用均为 `.334`、healthy、restart=0，端口为 18097/18098/18099。
不改变环境、端口、挂载、角色、PostgreSQL 或 Redis 容器。

## 发布与部署顺序

1. 合并版本准备 PR，固定 tag 在通过检查的提交，等待 Linux Release 与 GHCR 多架构镜像工作流成功。
2. 备份 Compose、环境文件、容器快照、数据目录及 PostgreSQL 自定义格式备份；校验备份目录可读。
3. 拉取 `ghcr.io/moeacgx/newapi-mao:v1.0.0-rc.10.1.10.335`，核实摘要、AMD64 架构及源码 revision。
4. 用 YAML 节点定位仅替换三个应用的 image 值；结构化对比其他配置一致。
5. 依次更新 `zzapi-slave-1`、`zzapi-slave-2`、`zzapi`，每个节点健康、实际运行版本、
   无身份访问门禁和日志检查通过后继续。
6. 最终核对公网版本与静态资源、数据库/Redis 身份和模块文件不变。

失败时停止后续节点，以备份 Compose 和旧 `.334` 镜像恢复失败应用；保留数据库备份，
不自动覆盖在线数据库。标签发布不等于实例升级，镜像构建不等于功能验收。

## 验证边界

PR #266、#267 的前后端 CI 已通过；发布准备不改业务代码。
TypeSafe 原样插件已通过本地真实 TokenAuth、SQLite 和模拟 HTTP 上游六项集成测试。
本次不自动安装、激活插件或创建收费渠道，不调用真实 TypeSafe 推理 API。
部署成功后管理员可在官方插件源安装 `typesafe@1.0.0` 并配置渠道与定价。

实际标签提交、镜像摘要、作业、备份及逐节点验收结果将在执行后补录。
