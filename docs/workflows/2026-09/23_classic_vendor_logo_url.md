# Classic 供应商自定义 Logo URL

## 目标与范围

为没有内置品牌图标的供应商提供图片直链配置。用户截图来自 Classic 的
`EditVendorModal.jsx`；仅修改 Classic，Default 不在本次范围内。

## 方案与契约

- 复用供应商 `icon` 字符串，保留 LobeHub 图标名及点号链式参数。
- HTTP/HTTPS 绝对地址使用浏览器图片加载，固定尺寸并等比完整展示；失败显示默认图标，
  切换地址后重新加载。其他协议不作为图片加载，不将 SVG 内容插入 HTML。
- 表单限制 128 字符并在校验时再次检查，匹配现有 `varchar(128)`，无需数据库迁移。
- 管理接口仍为 `/api/vendors/` 的 POST/PUT，详情为 `/api/vendors/:id`；沿用管理员权限。
- 后端不下载或代理图片。浏览器不发送 Referer；图片托管方仍会接收浏览器图片请求。
- 推荐公开可访问的 HTTPS 图片直链；HTTPS 页面可能阻止 HTTP 图片。
- 供应商、模型和渠道选择、模型广场沿用统一 `getLobeHubIcon` 渲染。
- Default 仍只识别内置图标名，共享记录中的 URL 在 Default 会显示首字母兜底。
- 定价缓存可能在保存后最多约一分钟才刷新。回滚前可将 URL 改回内置图标名或清空，
  回滚本身不删除数据。

## 验证计划

使用真实 React 渲染验证 HTTP/HTTPS、空白、非法协议、加载失败、切换地址恢复及旧图标。
检查输入提示、长度限制、多语言键、受影响文件 lint、Classic 构建与 Git 差异。
测试不访问真实外部图片；线上环境与 Default 不作为已验证范围。

## 本地验证结果

- 基线：`origin/custom-main` 的 `b2370db11`；专属分支 `feat/classic-vendor-logo-url`。
- 两组 Vitest / React Testing Library 测试共 14 项：HTTP/HTTPS、非法地址及协议、
  带凭据地址、固定尺寸、失败回退与换址恢复、内置图标及链式参数、保存请求和超长回填校验。
- 运行命令：在 `web/classic` 执行
  `node scripts/run-compat-tests.mjs src/helpers/__tests__/vendor-logo.compat.test.jsx src/components/table/models/__tests__/vendor-logo.compat.test.jsx`。
  测试进程设 60 秒上限；开发及审查期间出现过收集阶段超时，根因尚未确认。
  2026-09-23 提交前再次执行，两组 14 项全部通过，用时 14.14 秒。
  当前 CI 只构建 Classic，不自动运行这组测试，后续修改需按上述命令复测。
- 受影响 JS/JSX 文件 ESLint 通过；新增测试、表单、语言文件与文档完成 Prettier 检查，
  `render.jsx` 的新增代码保持该文件已有的无分号风格，未批量重排既有代码。
- 使用 `node node_modules/vite/bin/vite.js build` 验证 Classic 生产构建。
  构建有既有依赖 eval、Browserslist 数据过期和大分块警告。
- 本机 Bun 不在 PATH，使用已安装的 Node 和已有 Classic 依赖运行相同工具；
  i18n 使用 `node node_modules/i18next-cli/dist/esm/cli.js sync`，七个运行时语言及历史 `zh` 文件均补齐三项新键。
- 修改只涉及 JSX/JSON/Markdown，无 TypeScript 变更。检查新增文档链接、翻译键与
  `git diff --check`；未执行真实外部 Logo 请求、浏览器视觉验收或部署。

## 共享契约影响

供应商 `icon` 字符串、管理权限和定价返回结构保持不变；Classic 新增 HTTP/HTTPS 值的
图片展示语义。无模型 ID、认证、计费、调度或 relay API 改动；未通知其他项目。
