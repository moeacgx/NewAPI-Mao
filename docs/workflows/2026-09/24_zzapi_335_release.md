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

## 实际发布

版本准备 PR [#268](https://github.com/moeacgx/NewAPI-Mao/pull/268) 经前后端 CI 通过后合并。
标签固定在 `e94fb60a4edccb729a7a1dc6e254526a5972cbc7`。
[Release](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.335) 已发布。
Linux 工作流 `35888633586`、多架构镜像工作流 `35888633576` 均成功。

本机下载两架构二进制并与 `checksums-linux.txt` 比较，两者均通过：

- AMD64：`8f97d1b377dac04f6f393cc6517cd74bd188fa3aaae3a091d5662c45baac162a`。
- ARM64：`53c1193a35472ab3926bf89df14b94e19a33743c044f5d8072bf137c4080060c`。
- 镜像摘要：`sha256:3ebcd65ca90e33ede0da7a5892c3edda90c1261d0468572a4c6568ef43df6753`。
- 现场 AMD64 image ID：`sha256:cc92e6a81da7947ea6003b1f50a01bcac33b4e48c79dde3935df2cca85ed5eac`。
- 镜像 revision 与标签提交完全一致。

## 备份与配置

备份作业 `3da80647-52b8-492c-a10e-c9b80054e23c` 成功，目录为
`/home/docker/zzapi/backups/pre-335-20260924`。保存 Compose、环境文件、容器快照、
数据目录归档及 PostgreSQL 自定义格式备份；数据库备份 20,241,502 字节，
`pg_restore --list` 通过，没有执行恢复演练。

- 数据库 SHA-256：`258c7c89605b47e02fc38ecef84903ea221231c25dba94f923221e4348640bfa`。
- 原 Compose SHA-256：`7ec6310a0ec873294e0b00a12dd2c93b65dc44ac4bbca0c1e1c3d6f825e0d669`。
- 数据归档 SHA-256：`2e69b2545e1dd8e3889b3e60ea933c25449389ce93d672363a4e146b8bc6871f`。

准备作业 `05e5a9f3-9bdb-4c9b-8e20-16dba123d3a8` 拉取固定镜像，通过 YAML 节点定位
仅替换三个应用 image。结构化对比确认其他 Compose 字段没有变化。

## 逐节点更新

以下作业均成功、exitCode=0；每节点 `/api/status` 与 `/proc/1/exe --version` 均返回 `.335`，
镜像 ID 一致，healthy、restart=0；无身份访问 user/log/models/plugin 管理入口均为 401。

1. `zzapi-slave-1` / 18098：`61137286-8f3a-4cde-a274-0a421548da86`。
2. `zzapi-slave-2` / 18099：`bc506fc1-8a72-4cef-a3a6-df82edfb1af2`。
3. `zzapi` / 18097：`106d24af-3157-48ee-8cb5-03451a15204a`。

启动日志未发现 `panic:`、`fatal error:` 或 `[FATAL]`。本机独立请求公网状态返回
`success=true` 和 `.335`。未改动 maolaoapi，未安装或激活 TypeSafe，未执行回滚。

首次最终校验把 Docker 挂载列表按顺序比较，因主节点仅列表顺序改变而报错；
逐项核对路径、目标和读写模式一致后，改为排序比较，未修改实际挂载。

最终作业 `a8aef3ad-6f9d-41d7-be62-1f51daaed034` 成功、exitCode=0：

- 三节点环境变量、端口、挂载和启动命令与备份一致；PostgreSQL、Redis 的容器 ID、启动时间和重启次数不变。
- 三节点任务插件页面及 `/assets/index-BpjaYuOg.js` 均可读取，脚本大小为 11,823,659 字节。
- 备份清单中的数据文件 SHA-256 全部一致；没有读取或输出文件正文。
- 远端与本机独立请求公网 `/api/status` 均返回 `.335`。

部署只验证新宿主和资源已上线，不替代登录后的插件安装、渠道绑定或真实供应商调用验收。
