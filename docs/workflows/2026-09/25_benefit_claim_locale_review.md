# 福利领取卡片 locale 审查修复

## 问题

对 `25_benefit_voucher_lifecycle_random_fix.md` 的复核发现五项前端问题：

1. Default 繁体中文 `zh-TW.json` 为领取条件新增的六个 key（`Not eligible` / `Eligible` /
   `Activity is not active` / `Activity has not started` / `Activity has ended` /
   `Fully claimed`）与文件中已存在的同名通用 key 重复；JSON 后一份定义覆盖前一份，实际渲染
   出的是位于文件更后面的旧值（例如 `Not eligible` 实际显示 `Not 符合条件`），而不是随本次改动
   写入的正确译文。简体中文希望明确领取语境的"不符合领取条件"，而不是可能与其他业务共用的
   泛化文案。
2. 上述新增/新门槛 key 只补了英文与两种中文，法语、俄语、日语、越南语（Default）以及 Classic
   实际加载的 fr/ru/ja/vi 完全缺失。
3. 卡片上的"领取条件"只展示了历史充值门槛，但后端 `getBenefitClaimEligibilityTx` 还有一条隐藏
   条件：账号需要注册满 30 分钟，否则同样返回 `ineligible`（与充值门槛不足共用同一个 reason
   code，无法从返回值区分）。前端完全没有提示这一条件。
4. `formatBenefitDisplayAmount`（Default）与 `ClaimableActivityCard.jsx` 调用的
   `getCurrencyConfig()`（Classic，读 `localStorage.quota_display_type`）都没有使用 API
   响应自带的 `activity.amount_display_type`，而是另外读取前端自己缓存的全局展示配置
   （Zustand store / `localStorage`）。复核 `controller/benefit.go` 发现：
   `/api/benefit/activities` 返回的 `claim_paid_threshold` 与 `amount_display_type`
   都由 `benefitCurrentDisplayValues` 通过 `model.CurrentBenefitAmountDisplayContext()`
   （系统当前实时展示设置）现算现转，**不是**活动发布时固化的快照——`amount_display_type_snapshot`
   等快照列只在一次性迁移回填里写入，实际读路径不会用到。也就是说金额数字本身也是按当前
   展示类型换算的；两条前端路径通常会取到相同的展示类型，但金额和类型分别来自两次独立读取
   （一次是活动列表接口响应，一次是前端更早缓存的系统配置），无法保证原子一致——只要两次
   读取之间管理员改过展示设置，就可能出现数字和符号对不上。
5. 现有测试要么只在英文 mock（`t = key`）下断言、要么是纯源码正则契约测试
   （`user-benefits-contract.test.mjs` 用 `assert.match(source, ...)` 只检查代码里出现了某个
   调用，并不验证真实渲染结果），两者都无法暴露以上 1、4 两类问题。

## 根因

- 分组显示名称类的 key 复用问题类似：这几个 key 是通用英文短语，被在文件另一处以不同插入点
  重复添加，JSON 规范允许重复 key，但解析器只保留最后一份，形成"改了却没生效"的假象。
- `formatBenefitDisplayAmount` / `getCurrencyConfig()` 都从前端自己另外缓存的全局配置
  （Zustand store 或 `localStorage`）取展示类型，而不是与 `claim_paid_threshold`
  同一份响应里的 `activity.amount_display_type`；两者通常一致，但分别来自两次独立读取，
  无法保证原子一致。活动模型确实有 `amount_display_type_snapshot` /
  `amount_display_rate_snapshot` / `quota_per_unit_snapshot` 三个快照列，但当前读路径
  （`controller.benefitCurrentDisplayValues` → `model.CurrentBenefitAmountDisplayContext()`）
  并不读取它们，只在迁移回填时写入——这看起来更像是尚未接入读路径的预留字段，不属于本次
  前端范围能确认或修改的部分，仅在此记录供后端后续评估（未改动任何后端代码）。
- 测试基础设施本身：`web/src/test-setup.ts` 用空 `en` 资源初始化共享 `i18next` 单例（同一个
  `i18next` 包实例），所以仅调用 `@/i18n/config` 导出的 `i18n.changeLanguage()`
  不会让组件读到真实译文；Classic 的 contract 测试是源码文本正则匹配，不执行组件。

## 修改范围（仅前端，Default + Classic）

- 领取原因/状态文案改用不与其他业务共享的领域 key（`Not eligible to claim` /
  `Eligible to claim` / `Benefit activity is not active` / `Benefit activity has not
  started` / `Benefit activity has ended` / `Benefit fully claimed`）：
  - `web/src/features/benefits/lib/labels.ts`（`claimEligibilityLabel`）
  - `web/src/features/benefits/components/claimable-activity-card.tsx`
  - `web/src/i18n/static-keys.ts`
  - `web/classic/src/components/benefits/benefitLabels.js`
    （`BENEFIT_CLAIM_REASON_LABEL_KEYS`）
  - `web/classic/src/components/benefits/ClaimableActivityCard.jsx`
  - 旧的通用 key（`Not eligible` / `Eligible` / `Activity is not active` / `Activity has
    not started` / `Activity has ended` / `Fully claimed`）在各 locale 文件中原样保留
    （单份、不再被本功能引用），不做无关清理；只删除了 Default `zh-TW.json` 中被误插入的
    那一份重复副本。
- 新增 `Claim requirement` 家族 key（含新的 `New accounts must wait before they can
  claim` 提示）与上面的领域 key 一起，补齐 Default 全部 7 种语言
  （en/zh/zh-TW/fr/ru/ja/vi）与 Classic 实际加载的 7 种语言（en/zh-CN/zh-TW/fr/ru/ja/vi，
  未加载的 `zh.json` 不在范围内）。
- 卡片在展示"领取条件"金额下方新增一行说明账号需要注册满一定时长才能领取，不写死具体分钟数
  （后端该阈值是硬编码常量，未通过 API 暴露，写死数字会有与后端脱节的风险）；不改变后端
  `ineligible` 判定语义。
- `formatBenefitDisplayAmount`（Default）新增 `displayType` 参数，直接由
  `activity.amount_display_type` 驱动 USD/CNY/TOKENS 的符号与格式，不再读全局
  `getCurrencyDisplay()`；`CUSTOM` 类型下没有任何接口会返回符号文本（无论当前还是快照），
  保留使用前端当前全局自定义符号的兜底并在代码注释说明原因。
- Classic 新增 `benefitAmountCurrency(displayType, getFallbackCurrencyConfig)`
  辅助函数，同样由 `activity.amount_display_type` 驱动，替换直接调用
  `getCurrencyConfig()`（`localStorage`）的用法；`CUSTOM` 符号兜底同上。
- 两处改动的实际收益：让"金额数字"与"展示类型"始终来自同一份 `/api/benefit/activities`
  响应，消除两次独立读取之间的不一致窗口；不是"历史快照优先于当前配置"。

## 兼容性与边界

- 不改动后端：`model/`、`service/`、`relay/` 均未触碰；领取权限判定逻辑保持原样。
- 不改动 API 响应字段；只是让前端正确使用已有的 `amount_display_type` 字段。
- 未新增/删除任何数据库字段或迁移。
- 未涉及生产、凭据、推送、PR、合并；仅本地 worktree 提交。

## 验证

- Default（`web/`，复用 `D:\脚本程序\开源程序二开\newapi\web\node_modules`，bun.lock 哈希核对一致）：
  - `tsgo -b`：通过，无类型错误。
  - `vitest run src/features/benefits`：8 个测试文件、79 个用例全部通过，含新增的真实
    zh/zh-TW 渲染测试（通过 `i18next.addResourceBundle` 注入真实 `zh.json`/`zh-TW.json`
    资源，覆盖 USD/CNY/CUSTOM/TOKENS 四种展示单位）与限定触及 key 的原始文本重复 key 检查。
  - `oxlint -c .oxlintrc.json src/features/benefits src/i18n/static-keys.ts`：无问题。
  - `rsbuild build`：构建成功。
- Classic（`web/classic/`，复用对应 `node_modules`，bun.lock 哈希核对一致）：
  - `node --test` 运行 `user-benefits-contract.test.mjs` /
    `benefit-contract.test.mjs` / `activity-management-contract.test.mjs`：48 个用例全部
    通过。
  - 新增 `claimable-activity-card-i18n.compat.test.jsx`（同样用
    `i18next.addResourceBundle` 注入真实 zh-CN/zh-TW 资源，覆盖领取原因文案与
    "忽略过期 localStorage 缓存、按活动自身 `amount_display_type` 展示"两类回归）：
    **未能在本环境执行** —— 复用的 `node_modules`（包括本机指向的原始
    `D:\脚本程序\开源程序二开\newapi\web\classic\node_modules`）均未安装 `vitest`
    二进制，`package.json` 虽声明依赖但未曾 `bun install`；未在本任务范围内改动共享
    `node_modules`（安装依赖属于环境变更，按合同应先报告而非扩大范围）。已用
    `eslint`（`node_modules/.bin/eslint`）与 `prettier --check`（均已修复格式问题后转为通过）
    验证该文件语法/风格正确，且其断言与已通过的 Default 同名测试逻辑一致。
  - `i18next-cli status` / `i18next-cli lint`：只读运行成功，未发现与本次改动相关的新增
    问题（`lint` 输出的大量 `Found hardcoded string` 均为改动前已存在的历史项，与本次
    触及文件无关）。
  - `prettier --check` / `eslint`：`benefitLabels.js`、`ClaimableActivityCard.jsx`
    经 `prettier --write` 修正格式后，二者均通过。
  - `vite build`：构建成功（仅原有的 chunk 体积提示，非本次改动引入）。
- 所有触及的 Default/Classic locale JSON 均以 Node `JSON.parse` 逐一校验有效，并用原始文本
  正则确认本次触及的 10 个 key 在每个文件中只出现一次（不复用 `JSON.parse`，因为它会静默吞掉
  重复 key，正是本次要修的那类 bug）。
- Default 侧运行只读的 `node scripts/sync-i18n.mjs` 做结构校验：`missingCount`/
  `extrasCount` 全部为 0，产物与手工编辑内容一致，未引入非预期的结构性改动；`_reports`/
  `_extras` 输出目录已被仓库 `.git/info/exclude` 忽略，不会被提交。

## 已知限制 / 未做的事

- Classic 新增的真实渲染测试无法在当前环境验证执行结果，只做了静态检查；后续如需在 CI 中
  真正跑起来，需要先在 `web/classic` 补齐 `vitest` 依赖安装（超出本次前端 locale 修复范围）。
- `CUSTOM` 展示类型下没有任何接口返回符号文本（`/api/benefit/activities` 只返回
  `amount_display_type` 这个类型名，不返回符号；活动模型上的
  `amount_display_type_snapshot`/`amount_display_rate_snapshot`/`quota_per_unit_snapshot`
  三个快照列也不含符号文本，且实际读路径不读它们），沿用前端当前全局自定义符号作为兜底；
  如需精确符号，需要后端新增返回字段，本次未新增该 API 字段，仅在代码注释与本文档中记录
  这一限制。
- 后端观察（仅记录，未改动任何后端代码）：`benefit_activities` 表的
  `amount_display_type_snapshot`/`amount_display_rate_snapshot`/`quota_per_unit_snapshot`
  三个快照列目前只在 `migrateBenefitActivityQuotaConfig` 一次性迁移回填时写入，
  `controller.benefitCurrentDisplayValues`（`/api/benefit/activities`、
  `/api/benefit/admin/activities` 等实际读路径）改用
  `model.CurrentBenefitAmountDisplayContext()`（系统当前实时展示设置）现算，不读取这三列；
  与 `docs/developer/benefit-vouchers.md` 里"页面按系统当前 `quota_display_type` 展示……
  活动快照字段只解释创建时的单位/汇率，不参与当前页面展示"的说明一致，看起来是有意为之而
  非本次要修的 bug，仅在此提示这批快照列当前处于"写入但未被读路径使用"的状态，供后端后续
  评估是否需要真正启用或移除。
- 未对全仓库做通用重复 key 清理；只处理了本次改动实际触及的 10 个 key。
