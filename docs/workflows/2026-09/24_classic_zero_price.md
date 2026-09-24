# Classic 零价格编辑与回填

## 目标与范围

根据 Classic 定价页面零价回退反馈，复现价格输入、表达式生成、保存及同一模型配置回填。
零值必须作为明确价格保留，不能替换成缺省值或旧价格；不会修改线上售价。
只处理 `web/classic`；Default 不在截图与本次需求范围。任务用量公式 `u()` 的预览
与后端单位区分属于另一工作项，不混入本次修复。

## 调查与方案

用户确认输出价格改为 0 后在保存或刷新时变化。真实组件复现三个失败：已保存的
`c * 0` 回填成空白、编辑模式往返后零值消失、同名模型外部回填后仍显示旧输出价。

`PriceInput` 原来把数值零变成空串，以灰色占位符显示；修改为明确的字符串 `0`。
编辑器原来只监听模型名称，保存后同名模型的公式变化不能同步。现在同时监听公式和
请求规则，记录本地发出的值，区分父组件回传和真正的外部更新；不在打开或回填时
自动回写价格，保留 `7.` 等尚未完成的小数输入及原始编辑模式。

保存链路没有复现输出零价变为其他价格：普通按量保存 `CompletionRatio: 0`，阶梯
保存 `tier("base", p * 0.5 + c * 0)`。输入价格 0.5 与输出零价相互独立。
后端 Jev 倍率查询按配置键存在性读取零值，本项不修改计费计算。

## 验证计划

- 零价输入、模式切换与重新打开保留零值。
- 保存请求含明确的零价或零价公式，刷新后展示同一份配置。
- 同名模型配置变化可见，不拿旧草稿覆盖新配置。
- 原始复杂公式、历史请求条件和小数草稿不被重置。
- 定向组件测试限制 60 秒，执行改动文件 lint、格式检查和 Classic 构建。

## 验证记录

- 修复前真实组件 4 项中 3 项按上述症状失败、1 项小数草稿测试通过；修复后新增
  7 项加已有 2 项复杂表达式/规则兼容测试共 9 项通过。保存接口仅模拟边界，实际
  组件、序列化和重新加载 hook 均真实运行，不声称线上保存已验收。
- `node scripts/run-compat-tests.mjs src/pages/Setting/Ratio/components/__tests__/zero-price.compat.test.jsx src/pages/Setting/Ratio/components/__tests__/legacy-request-rules.compat.test.jsx --pool=forks --poolOptions.forks.singleFork`
  通过，运行器保持 60 秒总超时。首次依赖冷加载触发超时，依赖预热后正式结果为
  16.50 秒；Semi UI 的旧 React API 弃用告警不影响结果。
- `node --test --test-timeout=60000 src/pages/Setting/Ratio/components/__tests__/time-rule-expr.test.mjs`
  6 项通过，保持原时间条件与跨午夜规则语义。
- 改动 JSX 的 ESLint 通过；新增测试和两份 Markdown 的 Prettier 检查通过。
  原 `TieredPricingEditor.jsx` 在基线已不符合全文件 Prettier（格式化会把 1699 行扩成
  2089 行），本次保持周边风格，未夹带整文件格式重写；全文件格式检查仍有该既有告警。
- Classic `node node_modules/vite/bin/vite.js build` 通过，保留既有 Browserslist 数据过期、
  lottie eval 和较大构建块告警。Classic 无 typecheck 脚本，本项仅改 JSX，无新增 TS。
- 当前环境 Bun 不在 PATH，复用与本工作区 `bun.lock` SHA256 一致的现有 Classic
  依赖目录，以 Node 运行项目现有工具；没有安装依赖、改锁文件或运行 Default 构建。
- `git diff --check`、新增 Markdown 相对链接检查及独立只读补丁审查通过。
- 尚未合并、发版或部署；没有修改生产价格、渠道、插件或发起收费模型调用。

## 共享契约影响

只修复 Classic 编辑器零值显示与配置回填，不改变 API 字段、模型 ID、倍率存储、
用户鉴权或后端账务语义。Default 不在本次范围。

## 2026-09-25 主分支整合复验

- 在 `origin/custom-main` 的 `ca5d5ebbd` 上重新应用原修复，开发文档索引冲突按条目
  合并，保留已合入的任务插件性能钩子与发布记录。
- 重新核对按量保存：输入 0.5 对应 `ModelRatio=0.25`，输出 0 对应
  `CompletionRatio=0`；阶梯保存保留 `tier("base", p * 0.5 + c * 0)`。
  已存在的 `isBasePricingUnset` 判定明确排除阶梯模式，后端倍率查询按键存在性
  读取零值。本次不新增后端计费或 Default 变更。
- 两份真实组件兼容测试重新通过，共 9 项；首次冷加载触发运行器的 60 秒上限，
  同一命令重跑耗时 40.57 秒通过。时间规则回归 6 项、改动 JSX 的 ESLint、
  测试及两份 Markdown 的 Prettier 检查通过。
- 本轮继续复用锁文件 SHA256 一致的 Classic 依赖目录。Bun 不在 PATH，使用 Node
  调用现有工具；没有修改依赖、锁文件或生产配置。
- Classic 生产构建通过，耗时 69 秒；保留 Browserslist 数据过期、lottie eval、
  大构建块告警。编辑器整文件 Prettier 仍为前述基线告警，未扩大全文件格式变更。
- 充值日志 PR #277 合入后再次同步至 `512910188`，代码无冲突，前端源码与
  9 项组件测试通过及构建通过时完全一致。同步后组件复验触发 60 秒上限，未出现
  断言失败；该后端变更不影响 Classic 资源，沿用前述测试及生产构建结果。
