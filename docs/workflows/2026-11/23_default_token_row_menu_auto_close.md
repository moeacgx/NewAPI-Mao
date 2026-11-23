# Default 令牌行「更多」菜单自动关闭

## 问题

Default 模板令牌列表每行末尾的「更多」菜单（复制密钥、复制连接信息、CC Switch、聊天子菜单、删除）打开约一秒后，即使不操作也会自行关闭。

## 根因

`useApiKeysColumns` 每次渲染都返回新的 `actions` 列对象，其 `cell` 是新的内联函数。TanStack Table 以新的组件类型渲染该单元格，`DataTableRowActions` 被卸载后重新挂载，菜单的打开状态随之丢失。

以下来源会在菜单打开后触发列重建：

- 打开菜单时读取密钥并更新密钥上下文；
- 令牌活动轮询；
- 用于计算相对时间的 `now` 时钟。

分组列此前已使用 `useMemo` 规避同类问题，操作列没有采用相同处理。

## 修改

- `web/src/features/keys/components/api-keys-columns.tsx`：用 `useMemo` 固定操作列，只依赖 `t`，让行菜单在后台刷新期间保持挂载。
- 新增回归测试 `web/src/features/keys/components/__tests__/token-row-actions-menu.test.tsx`：渲染完整令牌表并模拟活动轮询，验证打开菜单后空等超过两个刷新周期仍保持展开，Esc 和点击外部仍能正常关闭；另验证复制密钥、聊天子菜单和删除确认入口保持原有行为（只用模拟数据，不删除真实数据）。

Classic 模板（`web/classic`）有独立实现，后端也没有改动，本次都不涉及。

## 验证

```sh
cd web
bunx vitest run src/features/keys/components/__tests__/token-row-actions-menu.test.tsx
bunx vitest run src/features/keys
bunx tsc --noEmit -p tsconfig.app.json
bunx oxfmt --check src/features/keys/components/api-keys-columns.tsx src/features/keys/components/__tests__/token-row-actions-menu.test.tsx
bunx oxlint -c .oxlintrc.json src/features/keys/components/api-keys-columns.tsx src/features/keys/components/__tests__/token-row-actions-menu.test.tsx
```

- 新回归测试 2 项通过；撤回修复后该测试失败。
- `src/features/keys` 下其余测试中有 21 项失败。修复前后的失败列表逐条一致，属于基线已有问题，与本次改动无关。
- 未做真实浏览器验收：需要已登录会话和模拟后端，本次没有搭建。
