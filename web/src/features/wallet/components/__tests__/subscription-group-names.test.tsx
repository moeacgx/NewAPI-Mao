import { render, screen } from '@testing-library/react'
import { expect, test, vi } from 'vitest'
import { api } from '@/lib/api'
import { SubscriptionPlansCard } from '../subscription-plans-card'

test('钱包真实套餐列表使用公开接口名称，不显示内部code', async () => {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({ data: {
    success: true,
    data: url === '/api/subscription/plans' ? [{ plan: {
      id: 7, title: '测试套餐', price_amount: 10, duration_unit: 'month',
      duration_value: 1, quota_reset_period: 'never', enabled: true,
      sort_order: 0, max_purchase_per_user: 0, total_amount: 100,
      upgrade_group: 'legacy-code', upgrade_group_name: '当前分组名称',
    } }] : { subscriptions: [], all_subscriptions: [], billing_preference: 'subscription_first' },
  } }))
  render(<SubscriptionPlansCard topupInfo={null} />)
  expect(await screen.findByText('Upgrade Group: 当前分组名称')).toBeInTheDocument()
  expect(screen.queryByText(/legacy-code/)).not.toBeInTheDocument()
})
