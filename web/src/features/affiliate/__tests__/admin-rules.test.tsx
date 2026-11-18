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

import { AffiliateAdminPanel } from '../admin-panel'

const http = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }))
vi.mock('@/lib/api', () => ({ api: http }))

const session = vi.hoisted(() => ({ auth: { user: { role: 100 } } }))
vi.mock('@/stores/auth-store', () => ({
  useAuthStore: Object.assign(
    (selector: (state: typeof session) => unknown) => selector(session),
    { getState: () => session }
  ),
}))

const options = [
  { key: 'affiliate_setting.settlement_delay_seconds', value: '120' },
  { key: 'affiliate_setting.promotion_notice_enabled', value: 'true' },
  { key: 'affiliate_setting.promotion_notice_title', value: 'Saved title' },
  { key: 'affiliate_setting.promotion_notice_text', value: 'Saved body' },
]

beforeEach(() => {
  vi.clearAllMocks()
  http.get.mockImplementation(async (url: string) => ({
    data: {
      success: true,
      data: url.startsWith('/api/option') ? options : { items: [], total: 0 },
    },
  }))
  http.put.mockResolvedValue({ data: { success: true } })
})

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const route = createRootRoute({ component: () => <AffiliateAdminPanel /> })
  const router = createRouter({
    routeTree: route,
    history: createMemoryHistory(),
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return queryClient
}

describe('管理员返佣页内的返佣规则', () => {
  it('Root 在管理页内直接编辑规则与推广提示，并按原选项键保存', async () => {
    session.auth.user.role = 100
    const queryClient = renderPanel()
    fireEvent.click(
      await screen.findByRole('tab', { name: 'Commission rules' })
    )
    const delay = await screen.findByRole('spinbutton', {
      name: 'Settlement delay minutes',
    })
    expect(delay).toHaveValue(2)
    expect(
      screen.getByRole('switch', { name: 'Promotion page notice' })
    ).toBeChecked()
    const title = screen.getByRole('textbox', {
      name: 'Promotion notice title',
    })
    expect(title).toHaveValue('Saved title')
    expect(
      screen.getByRole('textbox', { name: 'Promotion notice body' })
    ).toHaveValue('Saved body')
    fireEvent.change(title, { target: { value: 'New title' } })
    fireEvent.click(
      screen.getByRole('button', { name: 'Save affiliate settings' })
    )
    await waitFor(() =>
      expect(http.put).toHaveBeenCalledWith('/api/option/affiliate-batch', {
        options: [
          {
            key: 'affiliate_setting.promotion_notice_title',
            value: 'New title',
          },
        ],
      })
    )
    expect(http.put).toHaveBeenCalledTimes(1)
    queryClient.clear()
  })

  it('多项修改合并为一次原子保存请求，不并行逐项写入', async () => {
    session.auth.user.role = 100
    const queryClient = renderPanel()
    fireEvent.click(
      await screen.findByRole('tab', { name: 'Commission rules' })
    )
    const delay = await screen.findByRole('spinbutton', {
      name: 'Settlement delay minutes',
    })
    fireEvent.change(delay, { target: { value: '5' } })
    fireEvent.click(
      screen.getByRole('switch', { name: 'Promotion page notice' })
    )
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Promotion notice title' }),
      { target: { value: 'Mixed title' } }
    )
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Promotion notice body' }),
      { target: { value: 'Mixed body' } }
    )
    fireEvent.click(
      screen.getByRole('button', { name: 'Save affiliate settings' })
    )
    await waitFor(() => expect(http.put).toHaveBeenCalledTimes(1))
    const [url, body] = http.put.mock.calls[0] as [
      string,
      { options: { key: string; value: unknown }[] },
    ]
    expect(url).toBe('/api/option/affiliate-batch')
    expect(
      Object.fromEntries(body.options.map((o) => [o.key, o.value]))
    ).toEqual({
      'affiliate_setting.settlement_delay_seconds': 300,
      'affiliate_setting.promotion_notice_enabled': false,
      'affiliate_setting.promotion_notice_title': 'Mixed title',
      'affiliate_setting.promotion_notice_text': 'Mixed body',
    })
    queryClient.clear()
  })

  it('服务端拒绝批量保存时保留草稿并可再次提交', async () => {
    session.auth.user.role = 100
    http.put.mockResolvedValueOnce({
      data: { success: false, message: 'title too long' },
    })
    const queryClient = renderPanel()
    fireEvent.click(
      await screen.findByRole('tab', { name: 'Commission rules' })
    )
    const title = await screen.findByRole('textbox', {
      name: 'Promotion notice title',
    })
    fireEvent.change(title, { target: { value: 'Draft kept' } })
    fireEvent.click(
      screen.getByRole('button', { name: 'Save affiliate settings' })
    )
    await waitFor(() => expect(http.put).toHaveBeenCalledTimes(1))
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Save affiliate settings' })
      ).toBeEnabled()
    )
    expect(title).toHaveValue('Draft kept')
    fireEvent.click(
      screen.getByRole('button', { name: 'Save affiliate settings' })
    )
    await waitFor(() => expect(http.put).toHaveBeenCalledTimes(2))
    queryClient.clear()
  })

  it('role 10 保留业务标签，不显示规则标签也不请求 Root 选项', async () => {
    session.auth.user.role = 10
    const queryClient = renderPanel()
    expect(await screen.findByText('No invitation records')).toBeInTheDocument()
    expect(
      screen.queryByRole('tab', { name: 'Commission rules' })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Withdrawals' })).toBeInTheDocument()
    expect(
      http.get.mock.calls.some(([url]) => String(url).startsWith('/api/option'))
    ).toBe(false)
    queryClient.clear()
  })
})
