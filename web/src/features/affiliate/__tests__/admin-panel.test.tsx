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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { AffiliateAdminPanel } from '../admin-panel'

const http = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock('@/lib/api', () => ({ api: http }))

beforeEach(() => {
  http.get.mockImplementation(async (url: string) => ({
    data: {
      success: true,
      data: {
        items: url.endsWith('/withdrawals')
          ? [
              {
                id: 17,
                user_id: 7,
                username: 'reviewer',
                method: 'usdt',
                quota: 500000,
                display_amount: 1,
                display_currency: 'USDT',
                status: 'pending',
                created_at: 1700000000,
                payout_details: {
                  usdt_address: 'test-address',
                  usdt_chain: 'TRC20',
                },
              },
            ]
          : [],
        total: url.endsWith('/withdrawals') ? 1 : 0,
      },
    },
  }))
  http.post.mockResolvedValue({ data: { success: true, data: null } })
})

describe('返佣业务和 Root 配置隔离', () => {
  it('业务空列表和各个标签可访问且没有 Root 配置请求或表单', async () => {
    render(<AffiliateAdminPanel />)
    expect(await screen.findByText('No invitation records')).toBeInTheDocument()
    expect(
      screen.queryByRole('tab', { name: 'Commission rules' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('tab', { name: 'Anti-Fraud' })
    ).not.toBeInTheDocument()
    for (const label of [
      'Manual referral',
      'Commission Records',
      'Affiliate Risk Control',
      'Withdrawals',
      'Applications',
      'Fraud Detection',
    ]) {
      fireEvent.click(screen.getByRole('tab', { name: label }))
      expect(screen.getByRole('tab', { name: label })).toHaveAttribute(
        'aria-selected',
        'true'
      )
    }
    expect(http.get.mock.calls.map(([url]) => url).sort()).toEqual([
      '/api/affiliate/admin/applications',
      '/api/affiliate/admin/fraud-alerts',
      '/api/affiliate/admin/invitations',
      '/api/affiliate/admin/records',
      '/api/affiliate/admin/risk-users',
      '/api/affiliate/admin/withdrawals',
    ])
    expect(
      screen.queryByRole('button', { name: 'Save settings' })
    ).not.toBeInTheDocument()
  })

  it('确认提现审核后调用原业务端点，拒绝确认时不提交', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<AffiliateAdminPanel />)
    fireEvent.click(screen.getByRole('tab', { name: 'Withdrawals' }))
    const approve = await screen.findByRole('button', { name: 'Approve' })
    fireEvent.click(approve)
    expect(http.post).not.toHaveBeenCalled()
    confirm.mockReturnValue(true)
    fireEvent.click(approve)
    await waitFor(() =>
      expect(http.post).toHaveBeenCalledWith(
        '/api/affiliate/admin/withdrawals/17/approve',
        { remark: '' }
      )
    )
    expect(
      http.get.mock.calls.some(([url]) => url.startsWith('/api/option'))
    ).toBe(false)
  })
})
