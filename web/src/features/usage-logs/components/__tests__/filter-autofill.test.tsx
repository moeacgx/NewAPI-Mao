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
import { getCoreRowModel, useReactTable } from '@tanstack/react-table'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import { CommonLogsFilterBar } from '../common-logs-filter-bar'
import { UsageLogsProvider } from '../usage-logs-provider'

const navigate = vi.hoisted(() => vi.fn())
vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(async () => ({
      data: {
        success: true,
        data: [{ id: 42, code: 'vip-internal', name: '专线当前名称' }],
        group_names: { 'vip-internal': '专线当前名称' },
      },
    })),
  },
}))
vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({ useSearch: () => ({}) }),
  useNavigate: () => navigate,
}))
vi.mock('@/hooks/use-admin', () => ({ useIsAdmin: () => true }))
vi.mock('../../api', () => ({
  getLogStats: async () => ({
    success: true,
    data: { quota: 0, rpm: 0, tpm: 0 },
  }),
  getUserLogStats: async () => ({
    success: true,
    data: { quota: 0, rpm: 0, tpm: 0 },
  }),
}))

function FilterFixture() {
  const table = useReactTable({
    data: [],
    columns: [],
    getCoreRowModel: getCoreRowModel(),
  })
  return <CommonLogsFilterBar table={table} />
}

beforeEach(() => {
  navigate.mockClear()
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
})
afterEach(() => {
  localStorage.clear()
  useAuthStore.getState().auth.reset()
})

describe('日志筛选凭据自动填充边界', () => {
  test('用户查询草稿不提交，切换 ID 清除用户名，查询与重置只提交当前身份', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={client}>
        <UsageLogsProvider>
          <FilterFixture />
        </UsageLogsProvider>
      </QueryClientProvider>
    )
    await user.click(screen.getByRole('button', { name: 'Expand' }))
    await user.type(screen.getByPlaceholderText('Username'), 'alice')
    expect(navigate).not.toHaveBeenCalled()
    await user.click(screen.getByRole('combobox', { name: 'User search type' }))
    await user.click(screen.getByRole('option', { name: 'User ID' }))
    const input = screen.getByPlaceholderText('User ID')
    expect(input).toHaveValue('')
    await user.type(input, '5215')
    expect(navigate).not.toHaveBeenCalled()
    await user.keyboard('{Enter}')
    expect(navigate).toHaveBeenLastCalledWith(
      expect.objectContaining({
        search: expect.objectContaining({ userId: '5215', page: 1 }),
      })
    )
    expect(navigate.mock.lastCall?.[0].search).not.toHaveProperty('username')
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    const resetSearch = navigate.mock.lastCall?.[0].search
    expect(resetSearch).not.toHaveProperty('userId')
    expect(resetSearch).not.toHaveProperty('username')
    client.clear()
  })

  test('隐藏分组当前名称和用户名令牌时提交原始稳定筛选值，恢复后展示当前名称', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={client}>
        <UsageLogsProvider>
          <FilterFixture />
        </UsageLogsProvider>
      </QueryClientProvider>
    )
    await user.click(screen.getByRole('button', { name: 'Expand' }))
    const group = screen.getByRole('combobox', { name: 'Group' })
    const token = screen.getByPlaceholderText('Token Name')
    const username = screen.getByPlaceholderText('Username')
    await waitFor(() => expect(group).not.toHaveAttribute('aria-busy', 'true'))
    await user.click(group)
    await user.click(screen.getByRole('option', { name: '专线当前名称' }))
    expect(group).toHaveTextContent('专线当前名称')
    expect(group).not.toHaveTextContent('vip-internal')
    await user.type(token, 'work-token')
    await user.type(username, 'alice')
    await user.click(screen.getByRole('button', { name: 'Hide' }))
    for (const input of [token, username]) {
      expect(input).toHaveAttribute('type', 'text')
      expect(input).toHaveClass('[-webkit-text-security:disc]')
    }
    expect(group).toHaveTextContent('••••••')
    expect(group).not.toHaveTextContent('专线当前名称')
    expect(group).toHaveAccessibleName('Group')
    expect(group.querySelector('[title]')).toBeNull()
    await user.click(group)
    expect(screen.queryByRole('option', { name: '专线当前名称' })).toBeNull()
    expect(
      screen.getByRole('option', { name: '••••••' }).querySelector('[title]')
    ).toBeNull()
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('button', { name: 'Search' }))
    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith(
        expect.objectContaining({
          search: expect.objectContaining({
            group: 'vip-internal',
            token: 'work-token',
            username: 'alice',
          }),
        })
      )
    )
    await user.click(screen.getByRole('button', { name: 'Show' }))
    for (const input of [token, username]) {
      expect(input).not.toHaveClass('[-webkit-text-security:disc]')
    }
    expect(group).toHaveTextContent('专线当前名称')
    expect(group.querySelector('[title]')).toHaveAttribute(
      'title',
      '专线当前名称'
    )
    client.clear()
  })
})
