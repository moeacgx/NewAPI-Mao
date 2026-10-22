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
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { CommonLogsStats } from '../common-logs-stats'
import { UsageLogsTable } from '../usage-logs-table'

const view = vi.hoisted(() => ({
  search: {} as Record<string, string | number>,
  isAdmin: true,
}))
vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({
    useSearch: () => view.search,
    useNavigate: () => vi.fn(),
  }),
}))
vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))
vi.mock('../usage-logs-provider', () => ({
  useLogsViewScope: () => ({ isAdminView: view.isAdmin }),
  useUsageLogsContext: () => ({ sensitiveVisible: true }),
}))
vi.mock('@/hooks', () => ({ useMediaQuery: () => false }))
vi.mock('@/hooks/use-table-url-state', () => ({
  useTableUrlState: () => ({
    columnFilters: [],
    pagination: { pageIndex: Number(view.search.page || 1) - 1, pageSize: 20 },
  }),
}))
vi.mock('../../lib/columns', () => ({ useColumnsByCategory: () => [] }))
vi.mock('../common-logs-filter-bar', () => ({
  CommonLogsFilterBar: () => null,
}))
vi.mock('../task-logs-filter-bar', () => ({ TaskLogsFilterBar: () => null }))
vi.mock('../usage-logs-mobile-card', () => ({
  UsageLogsMobileList: () => null,
}))
// 表格展示边界仅呈现生产组件实际提供的数据；查询与占位缓存逻辑仍走真实 React Query。
vi.mock('@/components/data-table', () => ({
  useDataTable: (options: { data: { username: string }[] }) => ({
    table: options,
  }),
  DataTableRow: () => null,
  DataTablePage: (props: {
    table: { data: { username: string }[] }
    isLoading: boolean
  }) => (
    <div aria-busy={props.isLoading}>
      {props.table.data.map((row) => (
        <span key={row.username}>{row.username}</span>
      ))}
    </div>
  ),
}))

const clients: QueryClient[] = []
afterEach(() => {
  cleanup()
  for (const client of clients) client.clear()
  clients.length = 0
  vi.resetAllMocks()
})

describe.each(['stats', 'table'] as const)('%s 查询主体隔离', (component) => {
  function mount() {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    clients.push(client)
    const element = () => (
      <QueryClientProvider client={client}>
        {component === 'stats' ? (
          <CommonLogsStats />
        ) : (
          <UsageLogsTable logCategory='common' />
        )}
      </QueryClientProvider>
    )
    const rendered = render(element())
    return () => rendered.rerender(element())
  }

  function arrange(search: Record<string, string | number>, isAdmin = true) {
    view.isAdmin = isAdmin
    view.search = search
    vi.mocked(api.get)
      .mockResolvedValueOnce({
        data: {
          success: true,
          data:
            component === 'stats'
              ? { quota: 0, rpm: 1733, tpm: 0 }
              : {
                  items: [{ id: 1, username: 'alice-log' }],
                  page: 1,
                  page_size: 20,
                  total: 1,
                },
        },
      })
      .mockImplementation(() => new Promise(() => undefined))
    return mount()
  }

  const previousValue = component === 'stats' ? '1733' : 'alice-log'

  it.each([
    ['用户 ID', { userId: '1' }, { userId: '2' }],
    ['用户名', { username: 'alice' }, { username: 'bob' }],
    ['用户筛选重置', { userId: '1' }, {}],
  ] as const)(
    '切换%s后等待新请求期间不展示旧用户数据',
    async (_, before, after) => {
      const rerender = arrange(before)
      expect(await screen.findByText(previousValue)).toBeInTheDocument()
      view.search = after
      rerender()
      await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2))
      expect(screen.queryByText(previousValue)).not.toBeInTheDocument()
    }
  )

  it.each([
    ['全部切到仅本人', true],
    ['仅本人切到全部', false],
  ] as const)('从%s时不展示先前范围数据', async (_, isAdmin) => {
    const rerender = arrange({ username: 'alice' }, isAdmin)
    expect(await screen.findByText(previousValue)).toBeInTheDocument()
    view.isAdmin = !isAdmin
    rerender()
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2))
    expect(screen.queryByText(previousValue)).not.toBeInTheDocument()
  })

  it('同一用户翻页时等待新请求期间继续显示已有数据', async () => {
    const rerender = arrange({ userId: '1', page: 1 })
    expect(await screen.findByText(previousValue)).toBeInTheDocument()
    view.search = { userId: '1', page: 2 }
    rerender()
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2))
    expect(screen.getByText(previousValue)).toBeInTheDocument()
  })
})
