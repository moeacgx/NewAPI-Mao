# Classic 令牌编辑永不过期显示修复

## 问题

令牌 `expired_time=-1` 是永不过期哨兵值，后端保存与列表展示均正确。Issue #398–#405
独立复审（`D:\Temp\issue398-405-ratio-header-independent-review-20261118\report.md`）
发现 Classic 编辑令牌弹窗的过期时间日期控件显示 `1970-01-01 07:29:59`，与“永不过期”语义不符（P3 展示问题）。

## 根因

`web/classic/src/components/table/tokens/modals/EditTokenModal.jsx` 载入令牌时只在
`expired_time !== -1` 时转换为日期字符串，`-1` 原样写入 Semi `Form.DatePicker`，控件将其当作毫秒时间戳渲染为 1970 年。
“永不过期”快捷按钮和新建初值同样把 `-1` 写入控件。

## 修改

- 表单内部用空值（`null`）表示永不过期：新建初值、“永不过期”快捷按钮、载入时 `expired_time <= 0` 均置空；
  载入规则与 Default（`expired_time > 0` 才转为日期）一致。
- 日期控件占位文案改为 `t('永不过期')`（Classic 既有 i18n 键），去掉必填规则；清空日期即表示永不过期。
- 提交（编辑 PUT 与新建 POST）时空值转换为 `-1`，非空值仍按原逻辑 `Math.ceil(Date.parse(v) / 1000)` 并校验格式与“不能早于当前时间”。
- 接口契约不变：永不过期仍提交 `expired_time: -1`，有限时间仍提交秒级时间戳。

行为变化：此前清空日期会触发“请选择过期时间”必填错误，现在清空即为永不过期，与 Default 一致。

## Default 评估

Default（`web/src/features/keys/lib/api-key-form.ts`）载入时 `expired_time > 0` 才转为 `Date`，否则为 `undefined`，
`DateTimePicker` 显示 `Never expires` 占位，提交时空值转为 `-1`。复审报告亦确认 Default 渲染永不过期占位，无需修改。

## 测试

- 新增 `web/classic/src/components/table/tokens/__tests__/token-expiry-sentinel.compat.test.jsx`：
  编辑 `-1` 令牌时控件为空、占位为“永不过期”且不含 1970，提交 `-1`；有限时间显示日期并按原时间戳提交；
  “永不过期”快捷按钮清空控件并提交 `-1`。修复前前两项断言以 `expected '1970-01-01 07:29:59' to be ''` 失败。
- `token-group-limit.compat.test.jsx` 新建/编辑两种模式补充断言提交 `expired_time: -1`。
- 运行：`web/classic` 下 `NODE_OPTIONS=--no-webstorage node scripts/run-compat-tests.mjs src/components/table/tokens/__tests__/`，9 文件 44 用例通过。
  本机 Node 26 内置 `localStorage` 全局会遮蔽 jsdom 实现导致 `localStorage` 为 `undefined`，需加 `--no-webstorage`。

## 未覆盖

未做真实浏览器/二进制运行验收、完整构建与部署。
