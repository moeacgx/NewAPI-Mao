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
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  afterAll,
  afterEach,
  beforeEach,
  describe,
  expect,
  test,
  vi,
} from 'vitest'

import { PageFooterProvider } from '@/components/layout/components/page-footer'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { apiKeySearchSchema } from '../../lib/api-key-search'
import { apiKeySchema } from '../../types'
import { ApiKeysProvider } from '../api-keys-provider'
import { ApiKeysTable } from '../api-keys-table'

// 图标供应商的浏览器 JSON 资源不参与令牌筛选契约。
vi.mock('@lobehub/icons', () => ({}))
const storage = vi.hoisted(() => {
  const values = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
    clear: () => values.clear(),
  })
  return values
})

const token31 = apiKeySchema.parse({
  id: 31,
  name: 'Same name',
  key: 'masked',
  status: 2,
  remain_quota: 100,
  used_quota: 0,
  unlimited_quota: false,
  expired_time: -1,
  created_time: 1,
  accessed_time: 1,
  model_limits_enabled: false,
})
let footer: HTMLDivElement

beforeEach(() => {
  storage.clear()
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  useAuthStore.getState().auth.setUser({ id: 41, username: 'owner', role: 1 })
  footer = document.createElement('div')
  document.body.append(footer)
})
afterEach(() => {
  cleanup()
  footer?.remove()
  useAuthStore.getState().auth.reset()
  vi.restoreAllMocks()
})
afterAll(() => vi.unstubAllGlobals())

function mountTable(initialEntry: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const root = createRootRoute({ component: Outlet })
  const auth = createRoute({
    getParentRoute: () => root,
    id: '_authenticated',
    component: Outlet,
  })
  const keys = createRoute({
    getParentRoute: () => auth,
    path: '/keys/',
    validateSearch: apiKeySearchSchema,
    component: () => (
      <ApiKeysProvider>
        <ApiKeysTable />
      </ApiKeysProvider>
    ),
  })
  const history = createMemoryHistory({ initialEntries: [initialEntry] })
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([keys])]),
    history,
  })
  const view = render(
    <QueryClientProvider client={client}>
      <PageFooterProvider container={footer}>
        <RouterProvider router={router} />
      </PageFooterProvider>
    </QueryClientProvider>
  )
  return { ...view, router, client, history }
}

function mockResponses(total = 21) {
  return vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/log/self/active') {
      return {
        data: {
          success: true,
          data: {
            summary: { total: 3, active_tokens: 2, by_token: { 31: 2, 32: 1 } },
            scope: 'cluster',
            observed_at: Date.now(),
          },
        },
      }
    }
    if (url.startsWith('/api/token/')) {
      return {
        data: {
          success: true,
          data: {
            items: [token31, { ...token31, id: 32 }],
            total,
            page: 2,
            page_size: 20,
          },
        },
      }
    }
    return { data: { success: true, data: {} } }
  })
}

describe('真实令牌表格的服务端筛选与 URL 状态', () => {
  test.each([false, true])(
    'URL 恢复完整组合条件，稳定 ID 徽标和服务端 total；手机=%s',
    async (mobile) => {
      if (mobile) {
        const original = window.matchMedia
        vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
          ...original(query),
          matches: query === '(max-width: 640px)',
        }))
      }
      const get = mockResponses()
      const view = mountTable(
        '/keys?page=2&pageSize=20&status=%5B%222%22%5D&filter=Same&token=needle&active_only=true'
      )
      await screen.findByText('2 tokens in use')
      await waitFor(() =>
        expect(screen.getAllByText('Same name')).toHaveLength(2)
      )
      expect(
        screen.getByText('2 active requests').getAttribute('data-token-id')
      ).toBe('31')
      expect(
        screen.getByText('1 active requests').getAttribute('data-token-id')
      ).toBe('32')
      expect(footer).toHaveTextContent('21')
      expect(
        screen.getByRole('checkbox', { name: 'Only active tokens' })
      ).toBeChecked()
      expect(
        get.mock.calls.some(
          ([url]) =>
            url.includes('status=2') &&
            url.includes('active_only=true') &&
            url.includes('keyword=Same') &&
            url.includes('token=needle') &&
            url.includes('p=2')
        )
      ).toBe(true)
      await userEvent.click(
        screen.getByRole('checkbox', { name: 'Only active tokens' })
      )
      await waitFor(() =>
        expect(view.router.state.location.search.active_only).toBeUndefined()
      )
      expect(view.router.state.location.search.page).toBeUndefined()
      await waitFor(() =>
        expect(
          get.mock.calls.some(
            ([url]) =>
              url.includes('status=2') &&
              url.includes('keyword=Same') &&
              url.includes('token=needle') &&
              url.includes('p=1') &&
              !url.includes('active_only')
          )
        ).toBe(true)
      )
    }
  )

  test('状态单选触发后端搜索并回第一页，清除后恢复普通列表', async () => {
    const get = mockResponses(41)
    const view = mountTable('/keys?page=2&pageSize=20')
    await screen.findAllByText('Same name')
    await userEvent.click(screen.getByRole('button', { name: 'Status' }))
    await userEvent.click(screen.getByRole('option', { name: 'Disabled' }))
    await waitFor(() =>
      expect(
        get.mock.calls.some(
          ([url]) =>
            url.includes('/search?') &&
            url.includes('status=2') &&
            url.includes('p=1')
        )
      ).toBe(true)
    )
    expect(view.router.state.location.search.page).toBeUndefined()
    await userEvent.click(screen.getByRole('option', { name: 'Disabled' }))
    await waitFor(() =>
      expect(get.mock.calls.at(-1)?.[0]).toBe('/api/token/?p=1&size=20')
    )
  })

  test('活动集合归零后校正空的越界页，不能保留第四页', async () => {
    const get = mockResponses(0)
    const view = mountTable('/keys?page=4&pageSize=20&active_only=true')
    await waitFor(() =>
      expect(get.mock.calls.some(([url]) => url.includes('p=4'))).toBe(true)
    )
    await waitFor(() => expect(view.router.state.location.search.page).toBe(1))
    expect(
      get.mock.calls.some(
        ([url]) => url.includes('p=1') && url.includes('active_only=true')
      )
    ).toBe(true)
  })

  test('浏览器后退恢复名称和密钥，不被旧的防抖值覆盖', async () => {
    const get = mockResponses()
    const view = mountTable('/keys?filter=Same&token=first')
    await screen.findAllByText('Same name')
    fireEvent.change(screen.getByPlaceholderText('Filter by API key...'), {
      target: { value: 'second' },
    })
    await waitFor(() =>
      expect(view.router.state.location.search.token).toBe('second')
    )
    await act(async () => {
      view.history.back()
    })
    await waitFor(() =>
      expect(screen.getByPlaceholderText('Filter by API key...')).toHaveValue(
        'first'
      )
    )
    fireEvent.change(screen.getByPlaceholderText('Filter by name...'), {
      target: { value: 'Changed' },
    })
    await waitFor(() =>
      expect(view.router.state.location.search.filter).toBe('Changed')
    )
    await act(async () => {
      view.history.back()
    })
    await waitFor(() =>
      expect(screen.getByPlaceholderText('Filter by name...')).toHaveValue(
        'Same'
      )
    )
    await waitFor(() =>
      expect(
        get.mock.calls.some(
          ([url]) => url.includes('keyword=Same') && url.includes('token=first')
        )
      ).toBe(true)
    )
  })

  test('重置同时清除活动、状态、名称和密钥，恢复第一页普通列表', async () => {
    const get = mockResponses()
    const view = mountTable(
      '/keys?page=2&status=%5B%222%22%5D&filter=Same&token=needle&active_only=true'
    )
    await screen.findAllByText('Same name')
    await userEvent.click(screen.getByRole('button', { name: 'Reset' }))
    await waitFor(() => expect(view.router.state.location.search).toEqual({}))
    expect(
      screen.getByRole('checkbox', { name: 'Only active tokens' })
    ).not.toBeChecked()
    expect(screen.getByPlaceholderText('Filter by name...')).toHaveValue('')
    expect(screen.getByPlaceholderText('Filter by API key...')).toHaveValue('')
    await waitFor(() =>
      expect(
        get.mock.calls.some(([url]) => url === '/api/token/?p=1&size=20')
      ).toBe(true)
    )
  })

  test.each(['name', 'API key'])(
    'Reset 立即取消未提交的 %s 草稿，不允许防抖写回',
    async (field) => {
      const get = mockResponses()
      const view = mountTable('/keys?status=%5B%222%22%5D')
      await screen.findAllByText('Same name')
      const input = screen.getByPlaceholderText(`Filter by ${field}...`)
      // 同一个事件周期内重置，不能先等待草稿提交。
      fireEvent.change(input, { target: { value: 'pending' } })
      fireEvent.click(screen.getByRole('button', { name: 'Reset' }))
      await waitFor(() => expect(view.router.state.location.search).toEqual({}))
      expect(screen.getByPlaceholderText(`Filter by ${field}...`)).toHaveValue(
        ''
      )
      await act(async () => {
        await new Promise((resolve) => window.setTimeout(resolve, 550))
      })
      expect(view.router.state.location.search).toEqual({})
      expect(get.mock.calls.some(([url]) => url.includes('pending'))).toBe(
        false
      )
    }
  )

  test.each([
    ['name', 'back'],
    ['API key', 'back'],
    ['name', 'forward'],
    ['API key', 'forward'],
  ] as const)(
    '历史 %s/%s 仅改变活动条件时仍取消未提交草稿',
    async (field, direction) => {
      const get = mockResponses()
      const view = mountTable('/keys?status=%5B%222%22%5D')
      await screen.findAllByText('Same name')
      await act(async () => {
        await view.router.navigate({
          to: '/keys',
          search: { status: ['2'], active_only: true },
        })
      })
      if (direction === 'forward') {
        await act(async () => view.history.back())
        await waitFor(() =>
          expect(view.router.state.location.search.active_only).toBeUndefined()
        )
      }
      fireEvent.change(screen.getByPlaceholderText(`Filter by ${field}...`), {
        target: { value: 'pending' },
      })
      await act(async () => view.history[direction]())
      await waitFor(() =>
        expect(view.router.state.location.search.active_only).toBe(
          direction === 'forward' ? true : undefined
        )
      )
      expect(screen.getByPlaceholderText(`Filter by ${field}...`)).toHaveValue(
        ''
      )
      await act(async () => {
        await new Promise((resolve) => window.setTimeout(resolve, 550))
      })
      expect(view.router.state.location.search.filter).toBeUndefined()
      expect(view.router.state.location.search.token).toBeUndefined()
      expect(get.mock.calls.some(([url]) => url.includes('pending'))).toBe(
        false
      )
    }
  )

  test('一个字段提交及普通活动切换不会清除另一个字段的输入法草稿', async () => {
    mockResponses()
    const view = mountTable('/keys')
    await screen.findAllByText('Same name')
    const token = screen.getByPlaceholderText('Filter by API key...')
    fireEvent.compositionStart(token)
    fireEvent.change(token, { target: { value: '输入中' } })
    fireEvent.change(screen.getByPlaceholderText('Filter by name...'), {
      target: { value: 'submitted-name' },
    })
    fireEvent.click(
      screen.getByRole('checkbox', { name: 'Only active tokens' })
    )
    await waitFor(() =>
      expect(view.router.state.location.search.filter).toBe('submitted-name')
    )
    expect(token).toHaveValue('输入中')
    expect(view.router.state.location.search.token).toBeUndefined()
    fireEvent.compositionEnd(token, { target: { value: '完整密钥' } })
    await waitFor(() =>
      expect(view.router.state.location.search.token).toBe('完整密钥')
    )
    expect(view.router.state.location.search.filter).toBe('submitted-name')
    expect(view.router.state.location.search.active_only).toBe(true)
  })

  test('交错编辑两个普通输入，各自提交时保留另一个未到期草稿', async () => {
    mockResponses()
    const view = mountTable('/keys')
    await screen.findAllByText('Same name')
    fireEvent.change(screen.getByPlaceholderText('Filter by name...'), {
      target: { value: 'first-name' },
    })
    await act(async () => {
      await new Promise((resolve) => window.setTimeout(resolve, 250))
    })
    fireEvent.change(screen.getByPlaceholderText('Filter by API key...'), {
      target: { value: 'later-key' },
    })
    await waitFor(() =>
      expect(view.router.state.location.search.filter).toBe('first-name')
    )
    expect(screen.getByPlaceholderText('Filter by API key...')).toHaveValue(
      'later-key'
    )
    await waitFor(() =>
      expect(view.router.state.location.search.token).toBe('later-key')
    )
    expect(view.router.state.location.search.filter).toBe('first-name')
  })

  test('活动查询失败保留可见旧计数与陈旧提示，点击重试恢复', async () => {
    const get = mockResponses()
    const view = mountTable('/keys')
    await screen.findByText('2 tokens in use')
    const original = get.getMockImplementation()
    if (!original) throw new Error('API fixture missing')
    get.mockImplementation(async (url, config) => {
      if (url === '/api/log/self/active') throw new Error('offline')
      return original(url, config)
    })
    await act(async () => {
      await view.client.invalidateQueries({ queryKey: ['token-activity'] })
    })
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Showing the last known state.'
    )
    expect(screen.getByText('2 tokens in use')).toBeVisible()
    expect(screen.getAllByText(/Stale/).length).toBeGreaterThan(0)
    get.mockImplementation(original)
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() =>
      expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    )
    expect(screen.queryByText('Stale')).not.toBeInTheDocument()
  })

  test('列表刷新使用最新分组名称，保留原始稳定 ID 和 code', async () => {
    const get = mockResponses()
    const original = get.getMockImplementation()
    if (!original) throw new Error('API fixture missing')
    let groupName = 'Standard Display'
    const token = {
      ...token31,
      group: 'old_alias,stable_b',
      group_ids: [11, 22],
      group_details: [
        { id: 11, code: 'stable_a', name: groupName, exclusive: false },
        { id: 22, code: 'stable_b', name: 'Premium Display', exclusive: false },
      ],
    }
    get.mockImplementation(async (url, config) => {
      if (url.startsWith('/api/token/')) {
        return {
          data: {
            success: true,
            data: {
              items: [
                {
                  ...token,
                  group_details: [
                    { ...token.group_details[0], name: groupName },
                    token.group_details[1],
                  ],
                },
              ],
              total: 1,
            },
          },
        }
      }
      return original(url, config)
    })
    const view = mountTable('/keys')
    expect(
      await screen.findByText('Standard Display, Premium Display')
    ).toBeVisible()
    groupName = 'Renamed Display'
    await act(async () => {
      await view.client.invalidateQueries({ queryKey: ['keys'] })
    })
    expect(
      await screen.findByText('Renamed Display, Premium Display')
    ).toBeVisible()
    expect(screen.queryByText(/old_alias|stable_b/)).toBeNull()
    expect(token.group).toBe('old_alias,stable_b')
    expect(token.group_ids).toEqual([11, 22])
  })
})
