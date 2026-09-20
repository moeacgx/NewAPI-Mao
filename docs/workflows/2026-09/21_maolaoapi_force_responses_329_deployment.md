# maolaoapi 强制 Responses `.329` 逐节点更新

## 授权与范围

用户明确要求逐个更新 maolaoapi。使用已经发布并在 zzapi 完成三节点验证的
`v1.0.0-rc.10.1.10.329`，发布提交为 `299a4baeb003442b738a3336d39f87949f899199`。
功能与限制见[渠道强制使用 Responses 上游](../../developer/channel-force-responses.md)。

CloudSSH 入口为 `API中转站 / 美国netcup rs4000 猫佬API`，serverId=38、hostId=11，
仅更新 `/home/docker/maolaoapi` 的三个应用，不重建数据库或 Redis，不修改业务渠道开关。
顺序为 `maolaoapi`、`maolaoapi-slave-1`、`maolaoapi-slave-2`。

## 现场预检

作业 `17d51f8c-1b14-4421-a0ac-37e2a501772e` 核实三个应用为 `.328`、healthy、restart=0，
端口分别为 18095、18100、18101，Compose 为 `/home/docker/maolaoapi/docker-compose.yml`。
主节点未设置 `NODE_TYPE`，两个从节点为 `slave`。保留现场角色、端口和挂载。

- 旧镜像摘要：`sha256:b3a9f9ca1be205efb3d6d147ca7162a654bf1d66667f7cc2b79128d50bcdf72e`。
- 旧 image ID：`sha256:f5f66fd530e3b3c71ad803f6ee565670c367ba9968d77d54bb7d666df188d573`。
- 目标镜像摘要：`sha256:3c42b83f1b0ec7e31056cff4b034123e6c875379557a54c759885deac4aecc5d`。
- 目标 image ID：`sha256:2385226a9cd73f2b7311199e8bbb628188be17788a378cd719a5ce99e8cc861e`。

## 执行与回滚方案

先备份 PostgreSQL 自定义格式逻辑备份、Compose、环境文件及模块目录，记录摘要及依赖容器身份。
`pg_restore --list` 校验备份目录，不表示完成全量恢复演练，备份不包含 Redis 快照。
拉取固定摘要镜像并核对 revision，结构化比较 Compose，确保仅三个应用的 image 变化。

逐节点使用同一份明确指定的 Compose 和环境文件执行 `up -d --no-deps --force-recreate`。
每节点检查健康、版本、镜像、零重启、运行配置摘要、未认证接口和启动错误，再更新下一台。
最终检查公网状态及静态资源、模块文件是否保持、数据库/Redis 容器身份和启动时间。
模块数量以此次备份实际结果为准，不把测试环境数量当作生产事实。

任一检查失败停止后续更新；需要回滚时恢复该应用的旧镜像配置并逐个验证，不自动回写业务数据库。
此次 `.328` 到 `.329` 无新增数据库迁移，强制开关默认关闭，旧版本忽略该字段。
保留旧镜像与备份，不访问真实上游模型，不把静态资源验证称为登录态浏览器验收。

## 备份记录

备份作业 `be941713-3ad0-469d-b21a-f94548adfb13` 成功。
目录为 `/home/docker/maolaoapi/backups/pre-329-20260920T215252Z`，
保存数据库、Compose、环境文件、模块归档、29 个模块文件摘要和脱敏容器配置摘要。
预检数据库约 43 GB，磁盘可用空间充足；备份生成期间应用保持运行。

- PostgreSQL 备份：2,065,547,985 字节，目录校验通过。
- 数据库 SHA-256：`8a32f9b8e4a537de5c5142b35f055006c75e25d43fa76975a8592c927e2c6971`。
- Compose SHA-256：`1468b7ab5032252f07398600ba6692d984754d4da15eea9168ef92dc3449c88d`。
- 模块归档 SHA-256：`f7764643ad4791f5e1373dae9c910452fd5d4082ffc4f6445da86e5237c82d3f`。

固定摘要镜像拉取作业 `a0059a05-4e82-4825-95da-151a3dcfe96c` 成功，
摘要与发布工作流及 zzapi 实际镜像一致。

配置准备作业 `54b57538-a213-4ca4-bd39-fc6574ef1a29` 成功：
revision 与发布提交一致，Compose 结构化比较仅三个应用镜像不同。
部署时继续使用 `tag@sha256` 固定引用，环境文件与备份逐字节比较；
显式指定项目目录、Compose 和备份环境文件，不加载其他默认覆盖配置。

## 逐节点更新记录

- `maolaoapi` / 18095：作业 `4684f934-de57-4957-b456-7eacc7e6b3e9` 成功。
  版本 `.329`、healthy、restart=0，运行配置摘要保持；无身份管理接口与 `/v1/models` 返回 401，Classic HTML 返回 200。
- `maolaoapi-slave-1` / 18100：作业 `11ada830-ea18-4e1a-b27d-05695c3aa340` 成功。
  版本、健康、零重启、配置保留与接口检查均通过。
- `maolaoapi-slave-2` / 18101：作业 `561c42e7-9561-47b2-abf5-b31aa5075dd4` 成功。
  版本、健康、零重启、配置保留与接口检查均通过。

## 最终验收

作业 `81c1882f-9fb9-4671-b930-27b005fb2259` 完成三节点复核：

- 三节点版本为 `.329`，实际 image ID 与预期一致，healthy、restart=0。
- 环境变量值、挂载、端口、启动命令、重启策略和网络配置摘要保持。
- 29 个模块文件集合与 SHA-256 完全一致。
- PostgreSQL、Redis 的容器 ID、启动时间、运行状态和重启次数保持。
- 各节点未认证 `/api/user/self`、`/api/extensions/`、`/v1/models` 返回 401，Classic HTML 返回 200。
- 启动后最近 300 行日志未发现 panic、FATAL 或 `[ERROR]` 标记。

服务器访问公网 `/api/status` 被 Cloudflare 返回 403，沿用原访问控制，没有绕过或修改。
操作端独立访问 `https://maolaoapi.com/api/status` 返回 200、`success=true`、`.329`。
读取公网 HTML 后下载其当前入口 `/assets/index-DeCaGX23.js`，返回 200、11,791,967 字节，
包含 `force_responses` 与对应开关文案。SHA-256 为
`4137b4142f8fff3ef9ef74c2cb024c7d52f3e1925681b10250199c9b16b18c8e`，与已发布静态资源一致。

现场记录位于 `/home/docker/maolaoapi/releases/329/verification.json` 和 `public-verification.json`。
保留旧镜像与备份；没有修改用户、令牌、渠道开关或规则，没有执行真实上游付费请求。
本次只更新 maolaoapi 三个应用，没有更新 zzapi、zhishiapi 或其他应用。
