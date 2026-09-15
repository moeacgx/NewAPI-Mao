# Classic 内置任务插件管理与渠道绑定

## 范围与基线

- 独立工作树：official-plugin-classic；分支：agent/official-plugin-classic。
- 本地功能基线：5d90556e217576ce17fa5f27ddc2c7260ad16b50。
- 仅修改 web/classic 与配套文档。Default、Go 后端、支付、福利、发票、
  通知与上游响应模型不在本次变更范围。
- 对接唯一后端实现树 official-plugin-backend 的扁平记录与 meta 合同，
  不采用历史上游 source=factory/override 的页面推断。
- 保留 /console/extensions、动态扩展菜单、proxy/native 页面及 NativeExtensionHost。
- 共享 README/custom-development 由主协调合并并集，本工作记录由 Classic 维护。

## 实际接入

路由 /console/task-plugins 使用 AdminRoute。Admin 可查看列表、详情、版本和运行时；
Root 才显示启停、激活和总开关操作，操作需要二次确认。侧栏及管理员/个人侧栏设置
新增 admin.task_plugins，并继续遵循现有主题、权限和 i18n。

| 接口                                    | 权限与用途                                    |
| --------------------------------------- | --------------------------------------------- |
| GET /api/plugin/task                    | Admin；每 key 一项，active 优先，否则最新归档 |
| GET /api/plugin/task/:key，可带 version | Admin；Root 返回 source，Admin 不展示源码     |
| GET /api/plugin/task/:key/versions      | Admin；历史版本，无源码                       |
| GET /api/plugin/task/runtime/status     | Admin；总开关和真实能力边界                   |
| PUT /api/plugin/task/runtime/status     | Root；请求 {enabled: boolean}                 |
| POST /api/plugin/task/:key/status       | Root；请求 {enabled: boolean}                 |
| POST /api/plugin/task/:key/activate     | Root；请求 {version: string}，激活并启用      |
| GET /api/task_plugin_options            | Admin；读取渠道绑定选项                       |

响应包装为 {success,message,data}。列表/详情使用扁平
key/version/api_version/source_hash/enabled/active/source_kind 和完整 meta。
channel_count 明确标为“启用渠道”，in_flight_count 为在途任务；缺失计数和可选
meta 显示“未提供”，不伪造零值或能力。当前管理页所需必填字段无缺失。

新安装 active=false、enabled=false 时正常展示归档并提供版本激活入口。
详情切换清空旧版本和旧内容；请求失败退出加载，提供重试和关闭；过期响应被取消。
HTTP 403 显示权限错误，写入失败不会乐观改写列表状态。

总开关默认 false，通过 runtime/status 读写，不使用 /api/option。
关闭仅禁止新提交，历史任务继续按 pin 处理、轮询和退款，不声称立即停止轮询。
上传、市场、删除、dryrun 不显示操作控件且不发对应请求；不加载市场来源，
不把 S3、匿名签名或尚未验收的用量计费宣称为可用。

## 渠道合同

- TaskPlugin=62，AtlasCloud=61，保留已有渠道类型。
- 选择框读取 key/name/version/models/channel_type，仅接纳 channel_type=62 的选项。
- 渠道创建和编辑保存 JSON 字符串 setting.task_plugin_key，不混入 settings。
- 选择插件只在模型列表为空时填入 models，保留手动配置的模型。
- 已有绑定在选项缺失、接口失败或权限不足时仍回显，不自动清空。
- 已有渠道无 sensitive_write 时禁改绑定；后端继续负责最终授权。
- 不修改原生扩展入口和既有渠道密钥、代理、多 Key、分组合同。

## 本轮验证

- Bun：使用本机缓存 1.2.17，frozen lockfile 安装；依赖声明与锁文件未变。
- Classic 生产构建实际通过，55.35 秒；存在既有大 chunk 与 Browserslist 警告。
- 插件/渠道定向组件测试：13 项通过，15.41 秒；使用真实 Semi 组件与 Axios adapter。
- 全部 Classic compat 组件测试：8 文件、22 项通过，16.66 秒。
- bun run test:native：11 项通过；八语言命名空间与插值合同测试通过。
- 定向 ESLint 通过；定向 Prettier 除已有 EditChannelModal.jsx 格式问题外通过。
  已用相同 Prettier 配置核对 HEAD 基线与工作文件，两者均存在该格式问题，未进行无关整文件重排。
- Node 静态合同：320 项通过，1 个既有测试文件因无扩展名 JSX import 无法加载。
  失败文件为 pricing-template-header.test.mjs，其引用 customNav.jsx 的方式未由本次修改。
  Bun test 不运行这些 node:test 注册用例，不能用其退出成功代替 Node 合同验证。
- 各测试和构建进程限制 60 秒；未执行生产 API、部署或真实供应商任务。
- 两份修改文档的 Prettier、相对链接与 git diff --check 已通过。
- 翻译由限定 Classic 的补键脚本写入并验证八语言命名空间及插值；未执行可能改写全量
  历史键的 i18n:sync，未触碰 Default 翻译。

## 后端依赖与验收边界

此分支只交付 Classic 接线。当前 base 尚不含上述后端管理接口，需与
official-plugin-backend 的管理及任务执行实现组合验收。
真实供应商提交、按版本轮询、持久化、退款、资源访问和生产就绪仍由后端闭环验证；
组件 fixture 与前端构建不证明这些链路已完成。

完整“官方插件可体验”目标仍需组合验收，不能因 builtin_only 的管理页面可用就
标为整体完成。回滚此 UI 工作不迁移数据库、不删除任务或扩展资源。
共享契约影响限于新管理请求与渠道 type=62/setting.task_plugin_key，
没有向兄弟项目发送通知。
