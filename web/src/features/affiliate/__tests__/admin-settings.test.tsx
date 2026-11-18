/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { expect, it, vi } from 'vitest'

import { AffiliateSettingsSection } from '@/features/system-settings/billing/affiliate-settings-section'

const http = vi.hoisted(() => ({ put: vi.fn(), get: vi.fn() }))
vi.mock('@/lib/api', () => ({ api: http }))

const defaultValues = {
  affiliate_setting: {
    first_level_enabled: true,
    first_level_ratio: 5,
    second_level_enabled: false,
    second_level_ratio: 0,
    settlement_delay_seconds: 120,
    min_withdrawal_amount: 10,
    trigger_topup_enabled: true,
    trigger_subscription_enabled: false,
    filter_redemption_topup_enabled: false,
    payout_methods: 'usdt',
    usdt_chain: 'TRC20',
    promotion_template: '{invite_link}',
    promotion_notice_enabled: false,
    promotion_notice_title: '',
    promotion_notice_text: '',
    review_enabled: false,
    auto_approve_after_days: 0,
    agreement_enabled: false,
    agreement_text: '',
    inviter_min_account_age_days: 0,
    inviter_min_recharge_amount: 0,
    invitee_min_account_age_days: 0,
    invitee_min_recharge_amount: 0,
  },
}

it('Root 配置保留分钟转秒保存且不加载管理员业务列表', async () => {
  http.put.mockResolvedValue({ data: { success: true } })
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const route = createRootRoute({
    component: () => <AffiliateSettingsSection defaultValues={defaultValues} />,
  })
  const router = createRouter({
    routeTree: route,
    history: createMemoryHistory(),
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  const input = await screen.findByRole('spinbutton', {
    name: 'Settlement delay minutes',
  })
  expect(input).toHaveValue(2)
  fireEvent.change(input, { target: { value: '3' } })
  fireEvent.click(
    screen.getByRole('button', { name: 'Save affiliate settings' })
  )
  await waitFor(() =>
    expect(http.put).toHaveBeenCalledWith('/api/option/affiliate-batch', {
      options: [
        { key: 'affiliate_setting.settlement_delay_seconds', value: 180 },
      ],
    })
  )
  expect(http.get).not.toHaveBeenCalled()
  expect(
    screen.queryByRole('tab', { name: 'Withdrawals' })
  ).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('tab', { name: 'Anti-Fraud' }))
  expect(
    screen.getByRole('button', { name: 'Save settings' })
  ).toBeInTheDocument()
  queryClient.clear()
})
