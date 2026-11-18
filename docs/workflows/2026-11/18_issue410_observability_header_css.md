# Issue #410 渠道可观测性原生样式泄漏到全局顶栏

## 目标

渠道可观测性（`channel-quality`）原生页面打开后，全局顶栏导航被挤掉、搜索框宽度异常。扩展样式只能作用于自身页面根节点，不得覆盖宿主顶栏等作用域外元素。不改顶栏共享组件，不改其他扩展。

## 根因

- 页面走全局 `AppHeader`，原生样式由宿主以文档级 `<link>` 注入（Default：`web/src/features/extensions/native-page.tsx` 的 `loadStyles`；Classic：`web/classic/src/pages/Extensions/NativeExtensionHost.jsx` 的 `loadStylesheet`），页面挂载期间存在。
- `extension/builtin/channel-quality/public/native/default.css` 是未作用域的 Tailwind v4 生成产物：`@layer utilities` 中输出裸 `.w-full`、`.flex` 等，另有全局 `:root`/`:host` 主题变量、`@property --tw-*` 和 `@keyframes spin`。顶栏 Search 的 `xl:w-40` 与扩展 `.w-full` 同层、同权重，后加载的扩展规则获胜，搜索框被撑满并挤掉导航。
- `classic.css` 是无层、无作用域的 Tailwind v3 产物，裸 `.flex`、`.block`、`.w-full` 等未分层规则会压过 Classic 宿主的 `hidden md:block` / `hidden md:flex`，属于同类风险。
- 仓库内没有这两个 CSS 的构建源，只能靠后置 CSS 层顺序无法修复：后声明的层优先级更高，无层规则更高，必须按选择器作用域限定。

## 方案

- 新增构建入口 `extension/native-source/channel-quality/build.mjs`，输入为保存下来的原始 Tailwind 产物 `default.tailwind.css`、`classic.tailwind.css`，输出 `public/native/{default,classic}.css`。源文件放在 `extension/builtin/` 之外，避免被 `go:embed builtin/*` 打进内置包。
- 构建规则：所有选择器改写为 `:where(.channel-quality-native) <选择器>`，不增加优先级；输出放入 `@layer utilities`，`--tw-*` 默认值放入 `@layer properties` 并同样限定作用域；`:root`/`:host` 主题变量改挂在作用域根；`@keyframes spin` 改名为 `channel-quality-spin` 并同步 `--animate-spin`；去掉全局 `@property`。
- 两个入口 `default.mjs`、`classic.mjs` 的根节点各包一层 `div.channel-quality-native`。Default 在 `Tabs` 内的页面容器外层包裹，Classic 在默认导出外层包裹。
- Classic 宿主声明的层序为 `tailwind-base, semi, tailwind-components, tailwind-utils`，扩展层 `utilities` 排在其后，作用域内优先级与修复前的无层行为一致，不会被宿主 preflight 反向压制。

## 测试

新增 `extension/channel_quality_native_css_scope.test.mjs`（`node --test`，放在 builtin 之外），对 Default 与 Classic 各 3 项：

1. 样式只输出作用域内选择器，不含裸 `.w-full`、`.flex`、`.block` 等。
2. 样式不注册全局 `@property` 与 `@keyframes`。
3. 入口根节点携带 `channel-quality-native` 作用域类。

## 结果

- 红：未修复代码上 6 例中 5 例失败（Classic 的 `@property/@keyframes` 一例本就通过）。
- 绿：修复后新测试 6/6 通过；既有 `extension/channel_quality_group_names.test.mjs` 6/6 通过。
- Go：`GOARCH=amd64 go test ./extension -run 'TestManagerOpenNativeAsset|TestBuiltin|TestNative'` 通过。默认 386 目标的同类命令在编译阶段超过 60 秒被终止（exit 124），不计为通过。
- `git diff --check` 退出码 0。
- Default 与 Classic：两套模板都有同类未作用域样式，均已修复并由同一测试覆盖；宿主顶栏组件、其他扩展未改动。

## 未覆盖与已知限制

- 未在真实浏览器验证 Default 顶栏导航在桌面可见、搜索框在 `xl` 下为 `w-40`，也未目视检查渠道可观测页面；以上以 CSS 源码断言代替，不等同于浏览器验收。
- 重新生成 CSS 以 `*.tailwind.css` 为输入，原始 Tailwind 源入口仍不在仓库内；若页面新增工具类，需要同时更新这两个输入文件。
- 外置扩展仓库的同名产物、完整构建与部署未处理。
