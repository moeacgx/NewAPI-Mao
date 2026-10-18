# Classic 启动用户缓存损坏容错

## 问题与证据

Classic 启动时创建 API 客户端会调用 `getUserIdFromLocalStorage()`，该路径对
`localStorage.user` 直接执行 `JSON.parse`。`PageLayout.loadUser()` 和
`authHeader()` 也有直接解析路径。临时确定性复现覆盖了 `"undefined"`、损坏
JSON、`"null"`、数组和 Storage 读取异常：前四类会在至少一个启动调用点抛出，
数组虽可解析但不是合法用户缓存形状。

这证明了 Classic 启动容错缺口，但没有证明 maolaoapi 用户现场一定存在损坏的
`localStorage.user`，也没有替代浏览器 DOM/console 证据作为白屏最终归因。

## 修改范围

- Classic `helpers/auth-data.js` 新增安全用户缓存读取：缺失、解析失败、`null`、
  数组、其他非对象值，以及缺少有效正整数 `id` 或有限数字 `role` 的对象按未登录
  处理；可删除时清理 `user` 缓存，Storage 读取或删除异常不向启动路径抛出。
- Classic `utils.jsx`、`auth.jsx` 和 `PageLayout.jsx` 复用该 helper，包括登录页
  跳转、普通用户、管理员和 Root 路由守卫；保持合法用户对象、用户 ID 和 Bearer
  header 的既有形状。
- 不修改 Default，不重构认证拦截器；明确 401 或会话身份不匹配的清理契约保持
  不变，429、503、网络错误仍保留登录态。

## 回归与边界

- 新增实际 helper、API 模块初始化、既有启动调用点和四个路由守卫回归：损坏缓存
  不抛出、按未登录处理且不生成伪造用户 ID/header；合法用户行为不变。
- 已有 Classic 会话刷新兼容测试继续覆盖 429、503、网络错误保留登录态，以及
  401 和 `AUTH_SESSION_MISMATCH` 清理。
- 本工作项只覆盖 Classic startup cache recovery；生产现场是否命中该边界仍需
  浏览器运行时证据确认。

## 2026-09-25 集成核验

- 从最新 `custom-main` 摘取原缓存修复的两个提交，保留现有 `PageLayout` 的
  导入清理；不引入 PR #262 中的密码限流和静态资源嵌入变更。该 PR 的缓存恢复
  实现与本次原始提交相同，后续处理该 PR 时必须去重。
- 将四个路由守卫的 Storage getter 异常回归拆成独立用例，防止前一个渲染结果
  掩盖后续守卫错误。真实缓存 helper、API 初始化和会话刷新定向测试共 27 项通过，
  测试进程设置 60 秒超时。
- 涉及文件 ESLint、Prettier 和 `git diff --check` 通过；未改 TypeScript，
  Classic 没有独立 typecheck 脚本。本次只涉及 Classic，Default 不在范围内。
- Classic Vite 生产构建通过；沿用旧依赖的 Browserslist 数据、`lottie-web` 的
  `eval` 和大分块警告仍存在。本机 Bun 不在 PATH，本次通过 Node 调用已安装的
  Vitest、ESLint、Prettier 和 Vite，没有改动依赖清单或锁文件。
