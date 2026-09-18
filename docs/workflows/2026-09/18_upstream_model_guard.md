# 上游模型校验与整渠道禁用

## 分组显示名称修复

用户反馈模型不匹配通知展示内部 code。根因为事件负载和比较摘要直接读取记录的 `Group`，
插件分组选项还额外追加 code。项目根目录 `AGENTS.md` 增加全局分组显示名称规则。
通知改为在原有事务内按分组 ID 读取名称，记录 API 增加不落库的 `group_name`，
两模板展示名称且保留 code 提交契约；分组删除或名称缺失时才回退原标识，已解析分组的错误提示同样使用名称。
名称回归先复现通知出现 `Codex-Pro`、本地 Bot 出现 `default` 及重复规则错误显示 `default`，修复后均通过。
`go test ./model ./service ./controller ./router ./extension ./middleware -run 'TestUpstreamModelGuard' -count=1 -timeout=60s`
通过六包 Guard 回归；错误提示修复后重新运行 service 的 Guard 回归通过。
覆盖名称与 code 不同、通知入队及本地 Bot 请求、改名、删除重建。
Default 原生页面测试 9/9、Classic 2/2 通过，分别验证名称优先、无名称回退和保留 `group_codes` 提交。
两套原生入口已重建；TypeScript、相关文件 lint、格式及差异检查通过。
文档链接检查未发现新增断链，开发文档索引原有 81 个失效链接不属于本项修复范围。
本修复为本地源码变更，尚未重新部署宿主或上传新版插件；不重写已发通知。

## 目标与范围

提交前审查另发现两处边界并补回归：自动状态任务可能覆盖模型校验的人工禁用状态，
以及 Gemini 等适配器改写计费模型名后可能导致规则漏匹配。
前者需要自动状态入口与禁用事务使用相同的渠道行锁、提交后再更新缓存；
后者在绑定时固定客户端模型名，保持实际使用分组随选渠更新。
新测试已复现后者修改前漏关，固定模型名后通过。自动状态保护由显式的
`UpdateChannelStatusAutomatically` 与人工 `UpdateChannelStatus` 入口区分，原因文案不参与权限判断；
通知服务、TokensPro 概览和 Midjourney 自动调用均接入自动入口。
model/service 全包和状态定向回归通过，包含单/多 Key、迟到错误、旧节点缓存、能力更新失败回滚、
人工单条/标签恢复及恢复后的正常自动管理。新增及重跑 Go 测试均设置 60 秒超时。
最新源码独立打包成功，ZIP SHA-256 为 `0372d3bab6e34fa3442b5b9f47f8a602d9c9d67ae16b006894a72f5ade033173`；
该包保存在本地验证目录，历史部署产物保留，不将构建二进制或临时 ZIP 纳入源码提交。

用户确认按指定分组配置多条「请求模型 -> 允许的上游响应模型」规则；
上游明确返回不匹配的模型时立即关闭整条渠道，包含其他分组、模型和多密钥。
未返回模型名时跳过。Default 与 Classic 都提供原生扩展页面。
禁用后通过通知中心既有 Bot、Chat ID 和消息模板通知渠道名称及模型对比。
最终交付为可在模块管理上传安装的外置 ZIP；不随宿主自动安装，界面语言资源随包携带。

## 方案与边界

复用 `RelayInfo.SetUpstreamResponseModelName`，在解析到非空模型时同步执行状态变更。
使用请求原始模型和实际使用分组进行规则匹配；映射后的请求模型不能代替响应声明模型。
规则配置使用版本 CAS；渠道状态、触发记录及通知事件采用事务与条件更新，避免并发重复通知。
将整条渠道置为需要人工恢复的禁用状态，保留模型映射和密钥，普通测试不自动恢复。
客户端当前响应和已在途请求保持既有处理及计费流程。通知异步投递，不在流式回调中调用 Telegram。

API、权限、默认值、数据生命周期及回滚边界见[专题文档](../../developer/upstream-model-guard.md)。
实现阶段不操作远端；后续用户明确授权仅更新 zzapi 并验证插件安装，部署记录见下文。未发送真实 Telegram 消息。

## 验证计划

- 后端：分组和多模型匹配、空模型跳过、流式首个模型和后续变化、配置冲突与重复规则拒绝。
- 状态：整渠道多密钥禁用、其他渠道保持可用、并发幂等、禁止健康检查恢复、管理员显式启用。
- 通知：事件发现、默认模板保存、未知变量拒绝、消息转义、任务使用既有 Bot 和接收目标。
- 界面：双模板加载、新增和编辑规则、分组多选、保存失败与冲突、触发记录分页和窄屏布局。
- 检查：定向 Go 与前端测试、TypeScript、相关文件 lint、格式、链接及 `git diff --check`。

## 验证结果

- `go test ./model ./service ./controller ./router -count=1 -timeout=60s`：全部通过。
- `go test ./extension ./middleware ./relay/common ./relay/channel/openai -count=1 -timeout=60s`：全部通过。
- 新增集成用例使用真实 OpenAI SSE 解析器，确认错误模型帧转发前完成禁用、当前流正常结束、重复帧只入队一次。
- 本地 `httptest` Telegram 端点验证已有 Bot 的发送路径、渠道名称和 ID、模型摘要、HTML 转义、提及、成功状态及不重复投递；没有发送真实消息。
- Default：`node node_modules/vitest/vitest.mjs run src/features/extensions/__tests__/upstream-model-guard.test.tsx`，7 个用例通过，包含插件独立翻译与宿主 `zhCN` 语言代码；`tsgo -b` 与所改文件 lint 通过。
- Classic：`node --test web/classic/src/pages/Extensions/__tests__/upstream-model-guard.test.mjs`，实际 React/Semi UI、插件自带表单控件及 `zh-CN` 切换通过；宿主 SDK live API 测试通过。
- Classic 标准 Vite 构建通过，保留已有的大包体积与 lottie eval 警告。
- Default 标准 Rsbuild 构建失败于本机 C 盘工作树与 D 盘依赖 Junction 产生的字体 URL，报 `Unhandled scheme D:`。仅在忽略目录 `.local-tests` 的临时构建脚本中规范化字体 URL 后构建通过，没有修改仓库构建配置或依赖。
- 独立 SQLite 数据库、本地测试账号、1440px 桌面与 390px 手机 Playwright 验证通过：两规则、两分组、多个允许模型、保存回读、跨模板编辑、记录分页、无页面横向溢出、无页面运行错误。
- 浏览器验收修复 Classic 通知中心错误路径、中文运行时语言代码别名，以及固定顶栏与模块标题的重叠。
- 两套运行时各 7 种语言的 35 个页面文案键完整，翻译在 ZIP 中注册独立命名空间，覆盖 Default 的 `zhCN/zhTW` 与 Classic 的 `zh-CN/zh-TW`。宿主 SDK 和语言包没有剩余修改。当前环境没有 Bun，使用 Node 运行脚本。
- 截图保存在忽略目录 `.local-tests/default-desktop.png`、`default-mobile.png`、`classic-desktop.png`、`classic-mobile.png`。记录分页使用本地合成数据，禁用与通知行为由真实 Go 集成用例验证。
- Markdown 格式与 `git diff --check` 通过。检查 258 个相对文档链接，本次新增链接全部有效；开发索引原有 81 个缺失目标已与 `HEAD` 核对，不在本项修改范围。

## 外置安装验收

- `extensions/upstream-model-guard/build-package.ps1` 生成独立上传 ZIP 与 `.sha256`，不包含宿主二进制、源码测试或私有配置。
- 清单声明 `channel.upstream-model-guard` 能力；旧宿主拒绝未知能力，避免仅依赖版本字符串误安装。
- Go 生命周期测试通过：启动不预装、真实 ZIP 安装默认关闭、启停切换检测状态、通知声明生效、卸载后重启不复活、重新安装仍关闭。
- 隔离本地服务器通过真实 multipart 上传成品 ZIP，分别完成 Default/Classic 保存、多选、分页和窄屏检查，未捕获页面异常。
- 实际停用和卸载 API 验证成功，配置记录保留。测试环境首次启动遇到无关内置模块的 Windows 重命名占用，预先复制相同内置资源后启动成功；待测外置插件仍通过真实上传安装。

## 交付边界

经用户确认后已部署 zzapi 测试补丁，未正式发布、未修改业务渠道。PostgreSQL 迁移和配置读取已验证；MySQL 尚未执行真实数据库验收。
外置插件包可独立安装、更新与卸载，但实时检测和关渠仍由配套宿主支持执行，不能宣称旧版宿主零修改即用。
多节点各自维护扩展状态快照，即使共享模块目录也必须逐节点开启或刷新；共享数据库中的规则修改和渠道禁用由所有已启用节点读取。
共享契约影响仅为命中规则时渠道停止参与调度，客户端 API 请求、响应内容和既有计费语义不变。

## zzapi 上传失败定位

2026-09-18 用户在 zzapi 上传 ZIP 后收到 `module archive could not be installed`。
只读 CloudSSH 作业 `5071ecba-237e-442a-9b78-1ee32d215613` 核实三个应用均仍为正式 `.326`，
主节点在 09:06:49 记录一次 `extension archive installation failed`；未修改任何远端状态。

在隔离目录提取 `HEAD` 的原始扩展安装器，同一成品 ZIP 可稳定复现：
清单拒绝 `unsupported permission capability: channel.upstream-model-guard`，安装器转成
`module manifest is invalid`，控制器再用通用文案返回。包的文件完整性与结构正常，根因是配套宿主尚未部署。
仅移除能力声明会造成可安装但无实时检测，不能作为修复。

已本地编译 Linux amd64 `.326.guard.1` 配套宿主，并准备固定 `.326` 基础镜像的构建包；
它不是已发布版本。宿主支持涉及两个新增数据库表与三个应用更新，实际部署须取得明确授权。

## zzapi 授权更新与安装验证

用户明确确认「仅更新 zzapi 并验证安装」。使用 CloudSSH `API中转站 / RS2000 德国建站`，
serverId=52、hostId=17；未操作其他应用实例。预检作业 `8fd3e421-73f2-4bbf-8b2f-ad4ec1ba4dfe`
确认三节点 `.326`、healthy、restart=0，端口 18097/18098/18099。主节点负责迁移，两个从节点保留 `NODE_TYPE=slave`。

备份作业 `fccbde25-c448-483f-a9ff-b6a05685e76e` 成功，目录：
`/home/docker/zzapi/backups/pre-model-guard-20260918T011911Z`。
数据库自定义格式备份 88,561,577 字节，SHA-256 为
`bf7393711c1dfc8a011a48b8d8c8c0d34ac42ad8bbd23e152f9e0c45ceedc901`，`pg_restore --list` 通过；
同时备份 Compose、环境文件、32 项模块目录条目及容器身份快照。未执行数据库恢复演练。

构建与验证产物：

- 宿主包 SHA-256：`d0f280883ebe75432e17a132945c1427cbaff04d0ce460287d4463e23ea191aa`。
- 二进制 SHA-256：`3ad7e52c10bbc034c07babc5b0a0533b8556c2e28c814b9a6524e3f90b84c5ec`。
- ZIP SHA-256：`2c32245c721587f3ae5513e0d8bae7313e1192c72594969e13ca28a274af1080`。
- 固定基础镜像：`.326` / `sha256:a5097e68b262063a5d8bbdf2081c2baa30c78c3942055f5b7476f377e398a6a0`。
- 本地测试镜像：`maolaonewapi:zzapi-326.guard.1-20260918`。
- 实际 image ID：`sha256:c1c1e32822afe081840d76b72c6b0d5e1b2f8f71dde3d2de40d53039296aa0c2`。

CloudSSH 整包传输遇到并发限额，改为六段串行上传后合并，并验证完整包摘要。
现场 Docker 使用旧构建器，不支持 `COPY --chmod`；使用等价 `COPY` 并保留解包后二进制的 0755 权限。
Compose 包装器的 `--format json` 仍返回 YAML，改用结构化 YAML 比较；确认只改变三个应用的镜像。
这些调整均在应用重启前完成，不涉及全局依赖安装或其他服务修改。

隔离镜像验收作业 `58046d02-46cf-40ab-8ce9-239678921d03` 使用独立 SQLite 与临时测试账号，
通过真实 multipart ZIP 上传、两套原生入口、配置和空记录验证。测试容器已停止并移除，插件文件通过摘要校验后安装到
共享目录 `/home/docker/zzapi/data/data/modules/upstream-model-guard`，保持关闭。

串行滚动更新每台通过健康、版本、实际管理 API 模块列表及关闭状态校验后才继续：

- `zzapi`：`94210332-ade2-47aa-a4e4-8011965c9630`。
- `zzapi-slave-1`：`86bb0e23-baa3-42c4-9058-4a929a8bb37a`。
- `zzapi-slave-2`：`638f50c7-2670-42ab-aa21-958bbcccf2d1`。

最终作业 `315ba71e-7277-4748-8dc8-751fa6d121c6` 在实际 zzapi 上传接口重新上传同一 ZIP 并成功安装，
三节点都返回模块 `0.1.0`、相同资源摘要、`enabled=false`、无模块加载错误。
三个端口和公网 `/api/status` 返回 `.326.guard.1`；均 healthy、restart=0。
数据库中配置行数为 1、启用配置为 0、禁用记录为 0。PostgreSQL 与 Redis 容器 ID 和启动时间保持不变。
现场验证复用已有管理凭据，仅在目标主机进程内存中使用，没有输出、保存或轮换凭据。

保留原镜像和备份。回滚时恢复备份 Compose，仅逐台重建三个应用；新增表可保留，不自动删表或回写业务数据库。
完整现场验证结果位于 `/home/docker/zzapi/releases/326.guard.1/verification.json`。
