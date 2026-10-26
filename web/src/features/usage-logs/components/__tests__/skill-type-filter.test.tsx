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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { buildApiParams } from '../../lib/utils'
import { CommonLogsFilterBar } from '../common-logs-filter-bar'
import { UsageLogsProvider } from '../usage-logs-provider'

const navigate = vi.hoisted(() => vi.fn())
const routeSearch = vi.hoisted(() => ({
  current: {} as Record<string, unknown>,
}))
vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({ useSearch: () => routeSearch.current }),
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

function renderBar() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <UsageLogsProvider>
        <FilterFixture />
      </UsageLogsProvider>
    </QueryClientProvider>
  )
  return client
}

async function pickType(
  user: ReturnType<typeof userEvent.setup>,
  option: string
) {
  await user.click(screen.getByRole('combobox', { name: 'Log Type' }))
  await user.click(screen.getByRole('option', { name: option }))
  await user.click(screen.getByRole('button', { name: 'Search' }))
}

beforeEach(() => {
  navigate.mockClear()
  routeSearch.current = {}
})
afterEach(() => localStorage.clear())

describe('日志类型菜单中的技能选项', () => {
  test('技能只出现在类型菜单，不存在独立技能筛选', async () => {
    const client = renderBar()
    const user = userEvent.setup()
    await user.click(screen.getByRole('combobox', { name: 'Log Type' }))
    expect(screen.getAllByRole('option', { name: 'Skills' })).toHaveLength(1)
    expect(screen.queryByRole('combobox', { name: 'Skills' })).toBeNull()
    expect(screen.queryByText('Used skills')).toBeNull()
    client.clear()
  })

  test('选择技能提交 type=0 与 skill=used 并回到第一页', async () => {
    const client = renderBar()
    const user = userEvent.setup()
    await pickType(user, 'Skills')
    const search = navigate.mock.lastCall?.[0].search
    expect(search).toMatchObject({ type: ['0'], skill: 'used', page: 1 })
    client.clear()
  })

  test('从技能切换到消费或全部会清除 skill', async () => {
    routeSearch.current = { type: ['0'], skill: 'used', page: 3 }
    const client = renderBar()
    const user = userEvent.setup()
    await pickType(user, 'Consume')
    let search = navigate.mock.lastCall?.[0].search
    expect(search).toMatchObject({ type: ['2'], page: 1 })
    expect(search.skill).toBeUndefined()

    await pickType(user, 'All Types')
    search = navigate.mock.lastCall?.[0].search
    expect(search).toMatchObject({ type: ['0'], page: 1 })
    expect(search.skill).toBeUndefined()
    client.clear()
  })

  test('重置清除 skill', async () => {
    routeSearch.current = { type: ['0'], skill: 'used' }
    const client = renderBar()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    const search = navigate.mock.lastCall?.[0].search
    expect(search.skill).toBeUndefined()
    expect(search.type).toEqual(['0'])
    client.clear()
  })
})

describe('技能筛选的列表与统计参数', () => {
  test('列表和统计共用 buildApiParams，skill=used 固定 type=0，虚拟值不外泄', () => {
    const searchParams = { type: ['0'], skill: 'used', page: 2 }
    const list = buildApiParams({
      page: 2,
      pageSize: 100,
      searchParams,
      isAdmin: true,
    })
    const stats = buildApiParams({
      page: 1,
      pageSize: 1,
      searchParams,
      columnFilters: [],
      isAdmin: true,
    })
    expect(list).toMatchObject({ p: 2, type: 0, skill: 'used' })
    expect(stats).toMatchObject({ type: 0, skill: 'used' })
  })

  test('老链接仅有 skill=used 也映射为 type=0，原类型保持数值', () => {
    expect(
      buildApiParams({
        page: 1,
        pageSize: 10,
        searchParams: { skill: 'used' },
        isAdmin: false,
      })
    ).toMatchObject({ type: 0, skill: 'used' })
    const consume = buildApiParams({
      page: 1,
      pageSize: 10,
      searchParams: { type: ['2'] },
      isAdmin: false,
    })
    expect(consume.type).toBe(2)
    expect(consume).not.toHaveProperty('skill')
  })
})
