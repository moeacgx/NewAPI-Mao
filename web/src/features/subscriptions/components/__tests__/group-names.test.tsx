import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { flexRender, type CellContext } from '@tanstack/react-table'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import {
  createPlan,
  getAdminPlans,
  getPublicPlans,
  updatePlan,
} from '../../api'
import { formValuesToPlanPayload, planToFormValues } from '../../lib/plan-form'
import { subscriptionPlanSchema, type PlanRecord } from '../../types'
import { SubscriptionPurchaseDialog } from '../dialogs/subscription-purchase-dialog'
import { useSubscriptionsColumns } from '../subscriptions-columns'
import { SubscriptionsMutateDrawer } from '../subscriptions-mutate-drawer'
import { SubscriptionsProvider } from '../subscriptions-provider'

const plan = subscriptionPlanSchema.parse({
  id: 7,
  title: '测试套餐',
  price_amount: 10,
  duration_unit: 'month',
  duration_value: 1,
  quota_reset_period: 'never',
  enabled: true,
  sort_order: 0,
  max_purchase_per_user: 0,
  total_amount: 500000,
  upgrade_group: 'legacy-code',
  upgrade_group_name: '当前分组名称',
})

function GroupCell(props: { record: PlanRecord }) {
  const column = useSubscriptionsColumns().find(
    (item) => item.id === 'upgrade_group'
  )
  if (!column) throw new Error('upgrade_group column missing')
  return flexRender(column.cell, {
    row: { original: props.record },
  } as CellContext<PlanRecord, unknown>)
}

beforeEach(() => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: [{ plan }] },
  })
  vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
})

test('真实管理列展示当前名称，旧响应回退原标识', () => {
  const { rerender } = render(<GroupCell record={{ plan }} />)
  expect(screen.getByText('当前分组名称')).toBeInTheDocument()
  expect(screen.queryByText('legacy-code')).not.toBeInTheDocument()
  rerender(
    <GroupCell record={{ plan: { ...plan, upgrade_group_name: undefined } }} />
  )
  expect(screen.getByText('legacy-code')).toBeInTheDocument()
})

test('真实购买确认展示名称，改名后的响应立即更新展示', () => {
  const { rerender } = render(
    <SubscriptionPurchaseDialog open onOpenChange={() => {}} plan={{ plan }} />
  )
  expect(screen.getByText('当前分组名称')).toBeInTheDocument()
  rerender(
    <SubscriptionPurchaseDialog
      open
      onOpenChange={() => {}}
      plan={{ plan: { ...plan, upgrade_group_name: '改名后的名称' } }}
    />
  )
  expect(screen.getByText('改名后的名称')).toBeInTheDocument()
  expect(screen.queryByText('legacy-code')).not.toBeInTheDocument()
})

test('用户及管理列表保留名称，新增编辑的实际API仍提交历史code', async () => {
  const user = await getPublicPlans()
  const admin = await getAdminPlans()
  expect(user.data?.[0].plan.upgrade_group_name).toBe('当前分组名称')
  expect(admin.data?.[0].plan.upgrade_group_name).toBe('当前分组名称')
  const payload = formValuesToPlanPayload(planToFormValues(plan))
  await createPlan(payload)
  await updatePlan(7, payload)
  expect(api.post).toHaveBeenCalledWith(
    '/api/subscription/admin/plans',
    expect.objectContaining({
      plan: expect.objectContaining({ upgrade_group: 'legacy-code' }),
    })
  )
  expect(api.put).toHaveBeenCalledWith(
    '/api/subscription/admin/plans/7',
    expect.objectContaining({
      plan: expect.objectContaining({ upgrade_group: 'legacy-code' }),
    })
  )
  expect(payload.plan).not.toHaveProperty('upgrade_group_name')
})

test('真实编辑抽屉的升级和降级选择器显示名称，保存仍提交原code', async () => {
  vi.mocked(api.get).mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === '/api/group/details'
          ? [{ code: 'stable-code', name: '当前分组名称' }]
          : [],
      group_names: {
        'legacy-code': '当前分组名称',
        'stable-code': '当前分组名称',
      },
    },
  }))
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  render(
    <QueryClientProvider client={client}>
      <SubscriptionsProvider>
        <SubscriptionsMutateDrawer
          open
          onOpenChange={() => {}}
          currentRow={{ plan: { ...plan, downgrade_group: 'stable-code' } }}
        />
      </SubscriptionsProvider>
    </QueryClientProvider>
  )
  const upgrade = screen.getByRole('combobox', { name: 'Upgrade Group' })
  const downgrade = screen.getByRole('combobox', { name: 'Downgrade Group' })
  await waitFor(() => expect(upgrade).toHaveTextContent('当前分组名称'))
  expect(downgrade).toHaveTextContent('当前分组名称')
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() =>
    expect(api.put).toHaveBeenCalledWith(
      '/api/subscription/admin/plans/7',
      expect.objectContaining({
        plan: expect.objectContaining({
          upgrade_group: 'legacy-code',
          downgrade_group: 'stable-code',
        }),
      })
    )
  )
  client.clear()
})

test('真实创建抽屉选项展示名称，保存提交code且不提交名称', async () => {
  vi.mocked(api.get).mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === '/api/group/details'
          ? [{ code: 'stable-code', name: '当前分组名称' }]
          : [],
      group_names: { 'stable-code': '当前分组名称' },
    },
  }))
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  render(
    <QueryClientProvider client={client}>
      <SubscriptionsProvider>
        <SubscriptionsMutateDrawer open onOpenChange={() => {}} />
      </SubscriptionsProvider>
    </QueryClientProvider>
  )
  await userEvent.type(screen.getByPlaceholderText('e.g. Basic Plan'), '新套餐')
  await userEvent.click(screen.getByRole('combobox', { name: 'Upgrade Group' }))
  await userEvent.click(
    await screen.findByRole('option', { name: '当前分组名称' })
  )
  await waitFor(() =>
    expect(
      screen.getByRole('combobox', { name: 'Upgrade Group' })
    ).toHaveTextContent('当前分组名称')
  )
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() =>
    expect(api.post).toHaveBeenCalledWith(
      '/api/subscription/admin/plans',
      expect.objectContaining({
        plan: expect.objectContaining({
          title: '新套餐',
          upgrade_group: 'stable-code',
        }),
      })
    )
  )
  const body = vi.mocked(api.post).mock.calls[0][1] as { plan: object }
  expect(body.plan).not.toHaveProperty('upgrade_group_name')
  client.clear()
})
