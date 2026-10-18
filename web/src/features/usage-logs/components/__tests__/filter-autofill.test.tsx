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
import { afterEach, describe, expect, test, vi } from 'vitest'

import { CommonLogsFilterBar } from '../common-logs-filter-bar'
import { UsageLogsProvider } from '../usage-logs-provider'

const navigate = vi.hoisted(() => vi.fn())
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

afterEach(() => localStorage.clear())

describe('日志筛选凭据自动填充边界', () => {
  test('隐藏分组用户名令牌时保留文本输入且提交原始筛选值', async () => {
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
    const group = screen.getByPlaceholderText('Group')
    const token = screen.getByPlaceholderText('Token Name')
    const username = screen.getByPlaceholderText('Username')
    await user.type(group, 'vip-internal')
    await user.type(token, 'work-token')
    await user.type(username, 'alice')
    await user.click(screen.getByRole('button', { name: 'Hide' }))
    for (const input of [group, token, username]) {
      expect(input).toHaveAttribute('type', 'text')
      expect(input).toHaveClass('[-webkit-text-security:disc]')
    }
    expect(group).toHaveValue('vip-internal')
    await user.click(group)
    await user.keyboard('{Enter}')
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
    for (const input of [group, token, username]) {
      expect(input).not.toHaveClass('[-webkit-text-security:disc]')
    }
    client.clear()
  })
})
