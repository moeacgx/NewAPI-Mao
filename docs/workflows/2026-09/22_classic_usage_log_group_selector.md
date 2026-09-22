# Classic 使用日志分组下拉筛选（MAO-4）

## 目标与范围

Classic `/console/log` 的分组筛选原来使用文本框，用户必须记住分组标识。
本次将其改为可搜索、可清空的单选下拉框，只修改 Classic 使用日志。
Default 模板不在本次范围内；当前检出版本的 Default 分组筛选也仍为输入框。

## 方案与契约

- 管理员从 `GET /api/group/details` 获取分组详情，保留停用分组供历史日志查询。
- 普通用户从 `GET /api/user/self/groups` 获取当前有权使用的分组，不请求管理员接口。
- 复用现有分组选项转换函数，以名称展示、以内部 `code` 为值；搜索只过滤已有选项，不允许创建分组。
- 选择后点击“查询”沿用现有日志与统计请求的 `group` 参数；清空或重置表示不限制分组。
- 分组请求独立于日志请求；加载时显示状态，失败显示现有本地化错误提示，其他筛选及日志查询仍可使用。
- 不改变后端接口、认证权限、日志数据或计费规则，无数据库迁移；回滚前端修改即可。

## 验证计划

- 运行现有分组选项契约测试，确认显示名与提交标识保持分离。
- 检查 Classic 修改文件的格式与 lint，执行 Classic 构建。
- 核查管理员和普通用户的数据源、清空与重置行为，并运行 `git diff --check`。

## 验证结果

- `node --test --test-timeout=60000 web/classic/src/helpers/groupDetails.test.mjs`：21 项通过，覆盖名称展示、稳定 code 值、响应转换与空选择等现有契约。
- Classic 修改组件的 ESLint、Prettier 检查通过；文档使用仓库安装的 Prettier 格式化，新增索引链接目标存在，`git diff --check` 通过。
- 当前环境 PATH 没有 Bun，通过 `npx --yes bun install --frozen-lockfile` 安装锁定依赖，再执行 `npx --yes bun run build`；Classic 生产构建通过，锁文件无变化。构建提示 Browserslist 数据过旧、第三方 lottie 使用 eval 和大分块警告。
- 只读审查确认权限数据源、名称搜索、选项 value、清空/重置、异步旧响应保护与现有统计查询兼容；没有进行真实浏览器交互或线上验收。
- 无 TypeScript/TSX 修改，Classic 未配置 typecheck 脚本；不运行 Default 的类型检查或构建。
- 未变更共享 API 契约，未部署。
