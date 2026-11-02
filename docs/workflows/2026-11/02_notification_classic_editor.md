# Classic Telegram 通知模板编辑器

## 目标与范围

仅修改 Classic 通知中心的任务模板编辑区。模板源码保留为保存契约，提供可在光标位置插入的格式、变量、表情工具，工单预设仅在管理员主动点击时应用。Default 模板与后端由同一工作项的其他部分负责。

## 方案与接口

- 编辑区和实时消息预览在宽屏并排、窄屏上下排列，各自保留边框、背景和标题层次。
- 预览按 300 毫秒防抖调用 `POST /api/notification/preview`，请求包括 `event_type`、`template` 和首个目标的 `mention_user_id`、`mention_name`。
- 服务端以事件样例返回 `html`、`text_length`，不发送 Telegram 消息；编辑切换和失败时不保留旧成功结果，忽略陈旧响应。
- 工具栏支持加粗、斜体、下划线、删除线、引用、折叠引用、行内代码、代码块、剧透、链接和常用表情。

## 安全与兼容边界

预览用 DOMParser 解析后以严格标签及链接协议白名单转换为 React 元素，禁止直接注入 HTML。保存继续提交原模板字符串，无额外固定前缀。现有自定义模板不自动替换；变量插值和 Telegram HTML 合法性由服务端校验。

历史任务事件已下线或对应扩展未加载时，预览仍使用任务保存的真实 `event_type`，不能使用事件元数据回退定义中的发票事件替代。变量工具继续使用当前可用定义，服务端对未知事件返回错误；原自定义模板保持原样。

## 测试计划

使用 React Testing Library 验证选区包裹、光标插入、焦点恢复、显式预设、样例预览、竞态与失败状态，以及不安全标签、属性和链接协议无法进入 DOM。运行通知中心定向测试、修改文件 lint、Classic 生产构建与差异检查。Classic 无独立 typecheck 脚本，新增实现保持 JSX。

## 验证结果

- Classic 路由使用 `pages/NotificationCenter/index.jsx`；保留现有任务 Modal、目标配置和保存请求，新的编辑器仅替换该模板中的 TextArea 区域。
- 新增 25 个英文文案键，脚本覆盖 en、zh、zh-CN、zh-TW、fr、ja、ru、vi 八份 Classic 语言包，并以已有 `i18next-cli sync` 同步通过。历史语言包未整体排序，本次仅以字母顺序插入新增键，避免产生无关的大范围重排，已有键和值保留。
- 环境没有 Bun，使用现有 node_modules 和 Node 直接执行对应的 Vitest、ESLint、Prettier、i18next CLI 和 Vite 命令，未安装或更新依赖。独立工作树通过仅本工作树的 node_modules Junction 复用已安装依赖。
- 新编辑器 17 项 RTL 测试覆盖格式工具、变量和表情光标插入、焦点及选区、链接协议与属性转义、显式预设、样例预览、陈旧响应及未知历史事件，另有 2 项原工单事件测试，共 19/19 通过。15 项原筛选、Modal 响应式回归和 2 项新增编辑器布局规则测试通过，总计 36 项通过。
- 修改文件 ESLint、Prettier 和 `git diff --check` 通过。Classic Vite 生产构建通过；既有 Browserslist 数据陈旧、lottie-web eval 和大分块告警仍存在。
- Classic 为 JSX 项目，package.json 没有 typecheck 脚本，本次未新增 TS/TSX；没有运行不存在的类型检查。
- 本次没有发送真实 Telegram 消息；服务端契约及真实投递链路由后端工作项验证。
- Classic API 成功响应拦截器直接返回响应，不对 `success:false` 弹全局提示；预览已有 `skipErrorHandler:true`，取消请求同样不弹全局错误。新增断言沿用真实 API adapter 和拦截器路径，验证业务失败、陈旧响应取消及未知历史事件均仅在预览区域反馈。

## 限制与共享契约

前端预览展示服务端事件样例，不能替代真实 Telegram 客户端的像素级排版。折叠引用及剧透以可键盘展开的 details 展示，安全渲染不复制语言 class、style 或事件属性。模板为空、变量或 HTML 非法时展示服务端错误，不发送测试通知。

共享契约为新增 Root 预览接口；模板保存字段和变量契约保留原状。此记录仅覆盖 Classic 编辑器，管理员回复后编辑原消息与状态清理由后端工作项验收。Default 编辑器由并行工作项独立实现。
