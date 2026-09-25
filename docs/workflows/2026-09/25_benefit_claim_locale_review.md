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
  保留使用前端当前全局自定义符号的兜底。原先按类型取符号的单一调用点 `benefitAmountSymbol`
  已按项目规范就地内联进 `formatBenefitDisplayAmount`（只有一处调用，不构成独立业务概念，
  不再保留为单独 helper）；函数上方的契约说明收敛为 1-3 行中文注释，细节指向本文档。
- Classic 新增 `benefitAmountCurrency(displayType, getFallbackCurrencyConfig)`
  辅助函数，同样由 `activity.amount_display_type` 驱动，替换直接调用
  `getCurrencyConfig()`（`localStorage`）的用法。**CUSTOM 兜底修正**：`getCurrencyConfig()`
  返回的 `symbol` 字段是按其自身当前 `type` 解析的（`type=USD` 时就是 `"$"`、`type=CNY`
  时是 `"¥"`），只有当 `fallback.type === 'CUSTOM'` 时那个 `symbol` 才是真正的自定义符号；
  若活动是 `CUSTOM` 但本地缓存当前是别的类型，不能借用缓存返回的 USD/CNY 符号冒充自定义
  符号，已改为回退中性符号 `¤`（与 `getCurrencyConfig()`/`DEFAULT_CURRENCY_CONFIG` 自身
  对"无自定义符号"场景使用的占位符一致）。新增测试覆盖"活动是 CUSTOM、本地缓存是 USD"
  这一具体场景。
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
- Classic（`web/classic/`）：
  - `node --test` 运行 `user-benefits-contract.test.mjs` /
    `benefit-contract.test.mjs` / `activity-management-contract.test.mjs`：48 个用例全部
    通过（用 `New-Item -ItemType Junction` 重建 `web/classic/node_modules` 指向原始
    `D:\脚本程序\开源程序二开\newapi\web\classic\node_modules`，创建前核对目标存在、
    链接路径不存在；只读复用，未写入共享目录）。
  - `eslint` / `prettier --check`：`benefitLabels.js`、
    `claimable-activity-card-i18n.compat.test.jsx` 均通过。
  - 新增 `claimable-activity-card-i18n.compat.test.jsx`（`i18next.addResourceBundle`
    注入真实 zh-CN/zh-TW 资源，覆盖领取原因文案、"忽略过期 localStorage 缓存、按活动自身
    `amount_display_type` 展示"、"CUSTOM 兜底不冒用缓存的 USD/CNY 符号"三类回归，共 5 个
    `test()`）：本机隔离 runner 实际运行 **5/5 通过**。先前的排障过程：
    1. 复用的 `node_modules`（含原始 D 盘 checkout）都没有 `vitest`/`@testing-library/*`/
       `jsdom` 二进制，`package.json` 声明了依赖但从未 `bun install` 过。
    2. 在会话 scratchpad 建了一个隔离目录，按 `bun.lock` 里的精确版本
       （`vitest@2.1.9`、`@testing-library/react@16.3.0`、
       `@testing-library/dom@10.4.2`、`@testing-library/user-event@14.6.1`、
       `jsdom@25.0.1`）单独 `npm install`，不带 Classic 的 `package.json`、不触发对
       768 个既有包的整体依赖解析，装完 168 个包、无冲突。
    3. 曾经尝试过用 `npm install --legacy-peer-deps` 直接装进
       `web/classic/node_modules`：npm 把这当成对整棵依赖树的重新解析，删掉了 341 个、
       改动了 24 个既有包（包括实际被引用的 `@emoji-mart`），确认为破坏性操作后已回退
       （删除损坏的本地副本，用 Junction 重新指回原始只读数据，未改动 D 盘）。
    4. 用隔离目录里的 `vitest` + 临时 `vitest.config.mjs`（`resolve.alias` 把
       `@testing-library/react` 等指向隔离安装的绝对路径，`root`/相对路径指向 Classic
       worktree，保留 `scripts/setup-compat-tests.mjs`）运行：配置文件放在 scratchpad
       （物理路径含空格 `D:\Program Files\Git\...`）时 esbuild 无法解析配置文件路径；
       把配置文件挪到 Classic 目录（无空格路径）后配置能加载，但 `resolve.alias`
       （试过对象写法、`{ find, replacement }` 数组写法，并加了 `deps.inline` /
       `server.deps.inline` 强制不外部化）始终没有应用到 `scripts/setup-compat-tests.mjs`
       里对 `@testing-library/react` 的导入，报
       `Failed to resolve import "@testing-library/react" from "scripts/setup-compat-tests.mjs"`——
       怀疑 vitest 对 `setupFiles` 的导入走了和普通测试文件导入不同的解析路径，未进一步排查。
    5. 上述失败的临时配置文件仅用于诊断，已删除，未改动 Classic 真实 `vitest.config.mjs`。
    6. 后续在本机 `C:\Users\Administrator\AppData\Local\Temp\classic-benefit-vitest-20260925`
       独立安装 `vitest@2.1.9`、`@testing-library/react@16.3.0`、
       `@testing-library/dom@10.4.2`、`jsdom@25.0.1`，仅以 Junction 指向 D 盘已有的
       React/ReactDOM/i18next/react-i18next/Semi UI；临时 config 的 `root` 指向 Classic，
       `cacheDir` 留在 TEMP，并将临时 setup 和目标测试解析到同一套 Vitest、RTL、React、i18n。
       无需在仓库或共享 D 盘 `node_modules` 安装依赖。执行：

       ```powershell
       node C:\Users\Administrator\AppData\Local\Temp\classic-benefit-vitest-20260925\node_modules\vitest\vitest.mjs run src/components/benefits/__tests__/claimable-activity-card-i18n.compat.test.jsx --config C:\Users\Administrator\AppData\Local\Temp\classic-benefit-vitest-20260925\vitest.config.mjs --reporter verbose
       ```

       无调试日志复跑退出码 0，1 个文件、5 个用例全部通过（14.37 秒）。初次加载大依赖图
       超过 60 秒后人为中断，退出码 1；开启 `vite:resolve` 定位到 `helpers/index.js`
       展开的依赖后，第二次运行亦输出 5/5 通过、退出码 0。临时 runner 不纳入仓库提交。
  - `vite build`：本轮未重新构建全项目（改动只在 `benefitLabels.js` 内部逻辑和注释，
    未涉及构建配置）；上一轮已验证构建成功，Junction 指向的是同一份原始文件，结论不变。
- 所有触及的 Default/Classic locale JSON 均以 Node `JSON.parse` 逐一校验有效，并用原始文本
  正则确认本次触及的 10 个 key 在每个文件中只出现一次（不复用 `JSON.parse`，因为它会静默吞掉
  重复 key，正是本次要修的那类 bug）。
- Default 侧运行只读的 `node scripts/sync-i18n.mjs` 做结构校验：`missingCount`/
  `extrasCount` 全部为 0，产物与手工编辑内容一致，未引入非预期的结构性改动；`_reports`/
  `_extras` 输出目录已被仓库 `.git/info/exclude` 忽略，不会被提交。

## 已知限制 / 未做的事

- Classic 真实渲染测试已由本机 TEMP 隔离 runner 执行通过；该 runner/config 未纳入仓库，
  常规 CI 仍需自行安装 Classic 声明的测试依赖。此验证不代表重新执行了 Classic 全量构建。
- `CUSTOM` 展示类型下没有任何接口返回符号文本（`/api/benefit/activities` 只返回
  `amount_display_type` 这个类型名，不返回符号；活动模型上的
  `amount_display_type_snapshot`/`amount_display_rate_snapshot`/`quota_per_unit_snapshot`
  三个快照列也不含符号文本，且实际读路径不读它们），沿用前端当前全局自定义符号作为兜底；
  如需精确符号，需要后端新增返回字段，本次未新增该 API 字段，仅在代码注释与本文档中记录
  这一限制。Classic 侧已修正一个子场景：本地缓存的当前类型若不是 `CUSTOM`（例如缓存着
  USD/CNY），`getCurrencyConfig()` 返回的 `symbol` 就只是 USD/CNY 符号，不是自定义符号，
  这种情况下不再借用该符号，回退中性符号 `¤`；Default 侧因 `customCurrencySymbol` 是与
  `quotaDisplayType` 无关的独立字段，本来就不受这个子场景影响。
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
