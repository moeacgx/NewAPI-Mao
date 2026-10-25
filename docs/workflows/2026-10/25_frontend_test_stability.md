# 前端测试稳定性修复

## 问题

发版前检查发现两处与业务改动无关的前端测试问题：

1. Default `src/features/extensions/__tests__/upstream-model-guard.test.tsx` 的
   “keeps selected channel identities and edited rules across allowlist search and pagination”
   在 CI 中间歇失败（PR #335、#337 各一次，重跑通过）：等待关键字 `Provider` 的渠道查询超时，
   实际只有关键字为空的首次加载请求。
1. Classic `src/components/table/usage-logs/__tests__/filter-autofill.compat.test.jsx` 在主分支上稳定失败：
   找不到占位符为“分组”的文本框。CI 只构建 Classic、不运行其测试，因此未被发现。

## 根因

1. 上游模型校验页面的“Search”按钮在渠道加载期间禁用（`disabled: props.disabled || state.loading`）。
   测试修改搜索框后立即点击；CI 较慢时首次渠道加载尚未结束，点击落在禁用按钮上被忽略。
   本地把首次加载延迟 300ms 即稳定复现同样失败。
1. `bffe91161` 把分组筛选从文本框改为按接口加载选项的 `Form.Select`，测试仍按文本框查找且没有模拟分组接口。

## 修改（仅测试）

1. 点击 Search 前等待按钮可用，与同一用例中翻页按钮的等待方式一致。
1. 去掉过时的文本框断言，保留原用例意图（不创建密码输入、表单关闭自动填充、查询提交稳定分组 code），
   并与同目录测试使用相同的接口模拟。Classic 测试配置为单线程且关闭隔离，模块模拟在文件间共享，
   使用不同模拟数据会让结果依赖执行顺序，因此显示名称展示不在此用例验证。

## 验证

- 上游模型校验：首次加载延迟 300ms 时，原用例失败、修复后通过；正常速度下整个文件 21 项连续 3 次通过；
  oxfmt、oxlint、`tsgo -b` 通过。
- Classic 使用日志目录 2 个文件 3 项连续 2 次通过，单独运行通过；Prettier、ESLint 通过。
