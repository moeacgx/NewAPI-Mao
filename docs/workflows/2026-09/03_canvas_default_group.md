# 无限画布默认分组

## 目标

管理员可在“模型相关设置 → 全局设置”预设无限画布的默认分组，减少用户首次使用时的配置成本。

## 行为契约

- 配置键为 `global.canvas_default_group`，默认值为空字符串。
- 另有 `global.canvas_text_default_group`、`global.canvas_image_default_group`、`global.canvas_audio_default_group`、`global.canvas_video_default_group`。空字符串表示不预设，不改变 `canvas_default_group` 的保存 code 和停用分组回显。
- 该配置只影响无限画布页面首次加载时下拉框的初始选中项，不锁定分组，用户仍可手动切换。
- `GET /api/user/self/groups` 仅在配置分组属于当前用户可用分组时返回 `canvas_default_group` 及四个分类字段；否则对应字段返回空值。
- 当自动分组允许用户选择时，模型设置中的默认分组下拉会提供“自动选择（auto）”。
- 未使用预设时，前端继续按 `default`、首个可用分组的顺序回退。
- 四个分类分组留空，或预设分组当前用户不可用时，不选中具体分组，打开链接不带对应参数，从而跟随默认分组。
- 打开链接的分类参数与经典版相同：`textGroup`、`imageGroup`、`audioGroup`、`videoGroup`。
- Default 和 Classic 无限画布入口会将“自动选择（auto）”置于分组列表首位；该排序不改变预设分组的初始选中规则。
- Canvas 请求仍必须携带显式 `group`，原有会话、来源和分组权限校验不变。

## 模板范围

- Default：模型设置与无限画布入口均支持该配置。
- Classic：模型设置与无限画布入口同步支持该配置。
- 两套模板共享后端配置和用户分组接口，但页面实现与验证链路独立。
- 画布页分组来自 `GET /api/user/self/groups`，模型设置分组来自 `GET /api/group/details`。名称与 code 不同时展示名称，保存和链接仍使用 code。

## 验证

- Go：`go test ./setting/model_setting -run CanvasDefaultGroup -count=1 -timeout 60s`。
- Go：`go test ./controller -run CanvasDefaultGroup -count=1 -timeout 60s`。
- 前端：运行受影响的 Canvas 选择逻辑测试、默认分组选项测试、Classic 分组工具测试，以及 Default typecheck/lint；环境缺少 Bun 或依赖时记录为未执行。
