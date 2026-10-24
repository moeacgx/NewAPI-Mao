# 技能编辑弹窗：引用正文折叠、横向溢出与固定保存栏

## 范围

- 模板：仅 Default（`web/src`）。用户截图中的“合集内技能 · 89/128”计数格式只在 Default
  `bundle-skill-editor.tsx` 中出现；Classic（`web/classic`）未修改，是否存在同类问题未核实。
- 组件：`web/src/features/prompt-center/components/skill-resource-editor.tsx`、
  `prompt-template-editor.tsx`。无接口、数据或后端改动。

## 问题

导入 `zhaoxuya520/reverse-skill`（89 个技能、大量引用文件）后编辑合集：

1. 每个引用文件正文都以文本框展开，保存按钮在表单末尾，需要长距离下滑。
1. 弹窗显示不完整，并出现左右滚动。

## 根因（横向溢出）

`DialogContent` 是 CSS Grid，未声明列宽时为 `auto` 列，最小宽度取子元素的 min-content。
项目 `Textarea` 使用 `field-sizing: content`，宽度随内容计算；引用正文中的超长行（十六进制、
长路径、代码）把文本框撑到数千像素，网格列随之超出弹窗宽度。

headless Chrome 复现：672px 弹窗内放一行 1600 字符的文本框，修复前 `scrollWidth=4278`、
文本框 4220px；改为 `grid-template-columns: minmax(0, 1fr)` 后 `scrollWidth=672`、文本框 614px。

## 修改

- 引用正文默认折叠：已有引用只显示路径、删除按钮和“引用正文”展开按钮（`aria-expanded`、
  `aria-controls`），展开后文本框以该按钮为可访问名称。折叠时文本框不挂载，表单值由
  react-hook-form 保留，保存仍提交完整正文。
- 新增的引用文件默认展开，便于直接填写。
- 引用正文校验失败（NUL 或超过 8 MiB）时强制展开，错误与内容同时可见。
- 折叠状态保存在每个技能的引用编辑器内；合集重排时随技能移动。切换装载模式或技能类型会
  重新挂载编辑器，已有引用恢复折叠。
- `DialogContent` 增加 `grid-cols-1`（`repeat(1, minmax(0, 1fr))`），长行在文本框内换行，
  不再撑开弹窗。
- 取消/保存按钮和保存失败信息放入弹窗底部固定栏（`sticky -bottom-4`，抵消弹窗 `p-4`
  内边距贴住底边），长表单任意位置都可直接保存并看到保存错误。表单校验错误仍由
  react-hook-form 聚焦到首个错误字段。

## 验证

- `progressive.test.tsx` 新增：已有引用默认折叠、展开/收起后可编辑、折叠状态保存提交完整正文；
  新增引用默认展开且已有引用保持折叠；折叠正文校验失败时自动展开并显示错误。
  原有编辑引用正文的用例改为先展开；合集重排用例改为按引用路径断言归属，并由保存请求断言
  完整正文。
- `vitest run src/features/prompt-center`：9 文件 81 项通过；`tsgo -b` 通过；改动文件
  `oxfmt --check`、`oxlint` 通过；`rsbuild build` 成功，产物 CSS 含
  `.grid-cols-1{grid-template-columns:repeat(1,minmax(0,1fr))}`、
  `.-bottom-4{bottom:calc(var(--spacing) * -4)}`。
- headless Chrome 以相同 CSS 规则构造 89 个长行技能的弹窗，桌面与窄屏宽度下
  `scrollWidth` 等于可视宽度；滚动到顶部、中部、底部时保存栏均可见且与弹窗底边间隙为 0
  （`bottom: 0` 时留有 16px 间隙，因此使用 `-bottom-4`）。
- 未在真实页面中登录管理员截图验收。
