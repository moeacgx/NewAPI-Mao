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
  数组和其他非对象值按未登录处理；可删除时清理 `user` 缓存，Storage 读取或
  删除异常不向启动路径抛出。
- Classic `utils.jsx`、`auth.jsx` 和 `PageLayout.jsx` 复用该 helper，保持合法
  用户对象、用户 ID 和 Bearer header 的既有形状。
- 不修改 Default，不重构认证拦截器；明确 401 或会话身份不匹配的清理契约保持
  不变，429、503、网络错误仍保留登录态。

## 回归与边界

- 新增实际 helper、API 模块初始化和既有启动调用点回归：损坏缓存不抛出、按未
  登录处理且不生成伪造用户 ID/header；合法用户行为不变。
- 已有 Classic 会话刷新兼容测试继续覆盖 429、503、网络错误保留登录态，以及
  401 和 `AUTH_SESSION_MISMATCH` 清理。
- 本工作项只覆盖 Classic startup cache recovery；生产现场是否命中该边界仍需
  浏览器运行时证据确认。
