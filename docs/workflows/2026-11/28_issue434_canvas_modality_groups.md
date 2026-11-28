# Issue #434 无限画布四类分组

## 问题

经典版无限画布可以分别选择文本、生图、音频、视频分组，打开时通过 `textGroup`、`imageGroup`、`audioGroup`、`videoGroup` 传给画布。新版页面和全局模型配置原先只有 `global.canvas_default_group`。

## 行为

- 新增四个配置，默认空字符串，表示不预设：
  - `global.canvas_text_default_group`
  - `global.canvas_image_default_group`
  - `global.canvas_audio_default_group`
  - `global.canvas_video_default_group`
- `global.canvas_default_group` 的保存 code、停用分组只回显名称且不可新选的行为保持不变。
- `GET /api/user/self/groups` 用同一套可用分组判断追加四个字段。用户不能使用的预设返回空字符串，不另造分组。
- 新版与经典版画布页在用户有权使用时初始选中对应分组，用户仍可修改。分类项可清空；清空后不传该查询参数，跟随当前默认分组。
- 新版画布下拉直接渲染用户分组接口返回的显示名称。已有分组时不再使用会显示“未找到分组”的命令面板空态。
- 模型设置中的空项文案是“不预设”，不复用默认分组的“跟随系统默认”。
- 不改余额、渠道或计费。

## 验证

定向测试、类型检查和静态检查见本次交付说明。未做真实浏览器、生产数据库和部署验收。
