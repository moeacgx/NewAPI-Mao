# 扩展宿主版本门禁移除

## 问题与根因

线上宿主版本为 `dev-*` 时，扩展注册表、在线安装、启用和代理路径都会调用
`hostCompatibilityError`。版本解析只接受 SemVer，因此即使模块实际具备所需的
Native SDK、后端接口和能力声明，也会被标记为 `module is incompatible with current host version`。

## 变更范围

- 移除宿主扫描、在线安装、启用、列表和代理路径中的 `host.min/max` 比较。
- 保留 `manifest.json` 中版本字段的格式校验及 `min <= max` 元数据校验。
- 保留 ZIP SHA-256、模块身份、权限角色、能力声明、Native SDK 资源和代理安全校验。
- 内置 `channel-quality`、`conversation-archive`、`okx-alipay-rate` 快照同步到无版本下限的新版本；渠道观测前端只检查实际统计接口，不再比较宿主版本。

## 验证计划

- `go test ./extension ./controller -count=1 -timeout 60s`。
- 覆盖 `dev-*` 宿主扫描、安装、启用和在线安装路径。
- `node --check extension/builtin/channel-quality/public/app.js`。
- `git diff --check`。

## 兼容边界

移除版本门禁不会补齐旧宿主缺失的 API、Native SDK 或能力。旧宿主仍可能在实际
资源加载或业务请求阶段返回依赖未就绪；该行为由接口和能力检查报告。生产部署仍需
先构建包含新内置快照的宿主镜像，再按节点健康门禁滚动更新。
