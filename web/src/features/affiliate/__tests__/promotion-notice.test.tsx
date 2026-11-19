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
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { Affiliate } from '../index'

const http = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }))
vi.mock('@/lib/api', () => ({ api: http }))

const notice = vi.hoisted(() => ({
  value: { enabled: true, title: '', text: '' } as {
    enabled: boolean
    title: string
    text: string
  },
}))

const balance = {
  user_id: 1,
  pending_quota: 0,
  available_quota: 0,
  frozen_quota: 0,
  risk_frozen_quota: 0,
  confiscated_quota: 0,
  withdrawn_quota: 0,
  transferred_quota: 0,
  total_quota: 0,
}

function responseFor(url: string) {
  if (url === '/api/affiliate/summary') {
    return {
      balance,
      aff_code: 'abcd',
      aff_count: 0,
      invite_link: 'https://example.test/register?aff=abcd',
      promotion_text: 'promo text',
      can_invite: true,
      promotion_notice: notice.value,
      setting: {
        first_level_enabled: true,
        first_level_ratio: 5,
        second_level_enabled: false,
        second_level_ratio: 0,
        settlement_delay_seconds: 0,
        min_withdrawal_amount: 10,
        trigger_topup_enabled: true,
        trigger_subscription_enabled: false,
        filter_redemption_topup_enabled: false,
        payout_methods: ['usdt'],
        usdt_chain: 'TRC20',
      },
    }
  }
  if (url === '/api/affiliate/application-status') {
    return {
      review_enabled: false,
      agreement_enabled: false,
      status: 'approved',
      can_invite: true,
    }
  }
  if (url === '/api/affiliate/agreement') {
    return { agreement_enabled: false, agreement_text: '' }
  }
  if (url === '/api/affiliate/leaderboard') return []
  if (url === '/api/affiliate/payout-account') return {}
  return { items: [], total: 0, page: 1, page_size: 20 }
}

function summaryCalls() {
  return http.get.mock.calls.filter(([url]) => url === '/api/affiliate/summary')
    .length
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const route = createRootRoute({ component: () => <Affiliate /> })
  const router = createRouter({
    routeTree: route,
    history: createMemoryHistory(),
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  notice.value = {
    enabled: true,
    title: 'Custom <b>title</b>',
    text: 'Plain <b>body</b> text',
  }
  http.get.mockImplementation(async (url: string) => ({
    data: { success: true, data: responseFor(url) },
  }))
})

describe('推广页提示弹窗', () => {
  it('开启时显示一次纯文本提示，关闭后刷新数据不会再次弹出', async () => {
    renderPage()
    expect(await screen.findByText('Custom <b>title</b>')).toBeInTheDocument()
    expect(screen.getByText('Plain <b>body</b> text')).toBeInTheDocument()
    expect(document.querySelector('b')).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'I understand' }))
    await waitFor(() =>
      expect(screen.queryByText('Custom <b>title</b>')).not.toBeInTheDocument()
    )

    const before = summaryCalls()
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(summaryCalls()).toBe(before + 1))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Refresh' })).toBeEnabled()
    )
    expect(screen.queryByText('Custom <b>title</b>')).not.toBeInTheDocument()
  })

  it('重新进入页面视为新的访问并再次弹出', async () => {
    const first = renderPage()
    expect(await screen.findByText('Custom <b>title</b>')).toBeInTheDocument()
    first.unmount()
    renderPage()
    expect(await screen.findByText('Custom <b>title</b>')).toBeInTheDocument()
  })

  it('开关关闭时不弹出', async () => {
    notice.value = { enabled: false, title: 'Custom <b>title</b>', text: 'x' }
    renderPage()
    expect(await screen.findByText('Promotion Copy')).toBeInTheDocument()
    expect(screen.queryByText('Custom <b>title</b>')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'I understand' })).toBeNull()
  })

  it('标题和正文为空时使用默认文案', async () => {
    notice.value = { enabled: true, title: '', text: '' }
    renderPage()
    expect(await screen.findByText('Promotion notice')).toBeInTheDocument()
    expect(
      screen.getByText(
        'Please follow the affiliate rules when promoting. Abusive invitations may be penalized.'
      )
    ).toBeInTheDocument()
  })

  it('乱序返回的旧汇总不会覆盖新设置，也不会在关闭后重新弹出', async () => {
    const pending: Array<(value: unknown) => void> = []
    http.get.mockImplementation((url: string) => {
      if (url !== '/api/affiliate/summary') {
        return Promise.resolve({
          data: { success: true, data: responseFor(url) },
        })
      }
      return new Promise((resolve) => {
        pending.push((data) => resolve({ data: { success: true, data } }))
      })
    })
    renderPage()
    await waitFor(() => expect(pending).toHaveLength(1))
    // Changing the leaderboard period starts a newer refresh while the first
    // summary request is still outstanding.
    fireEvent.change((await screen.findAllByRole('combobox'))[0], {
      target: { value: 'week' },
    })
    await waitFor(() => expect(pending).toHaveLength(2))
    const fresh = { ...(responseFor('/api/affiliate/summary') as object) }
    const stale = { ...fresh }
    Object.assign(fresh, {
      promotion_notice: { enabled: true, title: 'Fresh title', text: 'fresh' },
    })
    Object.assign(stale, {
      promotion_notice: { enabled: false, title: 'Stale title', text: 'stale' },
    })

    pending[1](fresh)
    expect(await screen.findByText('Fresh title')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'I understand' }))
    await waitFor(() =>
      expect(screen.queryByText('Fresh title')).not.toBeInTheDocument()
    )

    pending[0](stale)
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(screen.queryByText('Stale title')).not.toBeInTheDocument()
    expect(screen.queryByText('Fresh title')).not.toBeInTheDocument()
  })

  it('尚未满足邀请资格时同样弹出一次提示，且不解除申请门槛', async () => {
    http.get.mockImplementation(async (url: string) => {
      const data = responseFor(url) as Record<string, unknown>
      if (url === '/api/affiliate/application-status') {
        return {
          data: {
            success: true,
            data: { ...data, can_invite: false, status: 'none' },
          },
        }
      }
      if (url === '/api/affiliate/summary') {
        return { data: { success: true, data: { ...data, can_invite: false } } }
      }
      return { data: { success: true, data } }
    })
    renderPage()
    expect(await screen.findByText('Custom <b>title</b>')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'I understand' }))
    await waitFor(() =>
      expect(screen.queryByText('Custom <b>title</b>')).not.toBeInTheDocument()
    )
    expect(screen.queryByText('Promotion Copy')).not.toBeInTheDocument()
  })

  it('较早发出的旧汇总先返回时不会锁定过期的开关状态', async () => {
    const pending: Array<(value: unknown) => void> = []
    http.get.mockImplementation((url: string) => {
      if (url !== '/api/affiliate/summary') {
        return Promise.resolve({
          data: { success: true, data: responseFor(url) },
        })
      }
      return new Promise((resolve) => {
        pending.push((data) => resolve({ data: { success: true, data } }))
      })
    })
    renderPage()
    await waitFor(() => expect(pending).toHaveLength(1))
    fireEvent.change((await screen.findAllByRole('combobox'))[0], {
      target: { value: 'week' },
    })
    await waitFor(() => expect(pending).toHaveLength(2))
    const base = responseFor('/api/affiliate/summary') as object
    pending[0]({
      ...base,
      promotion_notice: { enabled: false, title: 'Stale title', text: 'stale' },
    })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(screen.queryByText('Stale title')).not.toBeInTheDocument()
    pending[1]({
      ...base,
      promotion_notice: { enabled: true, title: 'Fresh title', text: 'fresh' },
    })
    expect(await screen.findByText('Fresh title')).toBeInTheDocument()
  })

  it('首次汇总请求失败（429）不产生未处理的 Promise，重试后只弹出一次且之后不再重弹', async () => {
    const unhandled: unknown[] = []
    const onUnhandled = (reason: unknown) => unhandled.push(reason)
    process.on('unhandledRejection', onUnhandled)
    try {
      let failed = false
      http.get.mockImplementation(async (url: string) => {
        if (url === '/api/affiliate/summary' && !failed) {
          failed = true
          throw Object.assign(new Error('Too Many Requests'), {
            response: { status: 429 },
          })
        }
        return { data: { success: true, data: responseFor(url) } }
      })
      renderPage()
      await waitFor(() => expect(summaryCalls()).toBe(1))
      await waitFor(() =>
        expect(screen.getByRole('button', { name: 'Refresh' })).toBeEnabled()
      )
      expect(screen.queryByText('Custom <b>title</b>')).not.toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))
      expect(await screen.findByText('Custom <b>title</b>')).toBeInTheDocument()
      expect(screen.getAllByText('Custom <b>title</b>')).toHaveLength(1)

      fireEvent.click(screen.getByRole('button', { name: 'I understand' }))
      await waitFor(() =>
        expect(
          screen.queryByText('Custom <b>title</b>')
        ).not.toBeInTheDocument()
      )
      const before = summaryCalls()
      fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))
      await waitFor(() => expect(summaryCalls()).toBe(before + 1))
      await waitFor(() =>
        expect(screen.getByRole('button', { name: 'Refresh' })).toBeEnabled()
      )
      expect(screen.queryByText('Custom <b>title</b>')).not.toBeInTheDocument()

      await new Promise((resolve) => setTimeout(resolve, 50))
      expect(unhandled).toEqual([])
    } finally {
      process.off('unhandledRejection', onUnhandled)
    }
  })
})
