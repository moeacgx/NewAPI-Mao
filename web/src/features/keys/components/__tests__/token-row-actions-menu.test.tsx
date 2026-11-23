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
import { act, cleanup, render, screen, waitFor } from '@testing-library/react'
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
import { ApiKeysProvider, useApiKeys } from '../api-keys-provider'
import { ApiKeysTable } from '../api-keys-table'

vi.mock('@lobehub/icons', () => ({}))
const clipboard = vi.hoisted(() => ({ copy: vi.fn(async () => true) }))
vi.mock('@/lib/copy-to-clipboard', () => ({
  copyToClipboard: clipboard.copy,
}))
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

// 仅用本地虚构令牌；密钥为测试夹具字符串，不涉及真实数据。
const token = apiKeySchema.parse({
  id: 31,
  name: 'Fixture token',
  key: 'masked',
  status: 1,
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
  clipboard.copy.mockClear()
  vi.restoreAllMocks()
})
afterAll(() => vi.unstubAllGlobals())

function DialogProbe() {
  const { open, currentRow } = useApiKeys()
  return (
    <output data-testid='dialog-probe'>{`${open}:${currentRow?.id}`}</output>
  )
}

function mountTable() {
  let activityCalls = 0
  let activeCount = 1
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/log/self/active') {
      activityCalls += 1
      // 每次轮询返回不同计数，模拟真实后台刷新带来的数据变化。
      activeCount += 1
      return {
        data: {
          success: true,
          data: {
            summary: {
              total: activeCount,
              active_tokens: 1,
              by_token: { 31: activeCount },
            },
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
          data: { items: [token], total: 1, page: 1, page_size: 20 },
        },
      }
    }
    if (url === '/api/status') {
      return {
        data: {
          success: true,
          data: { chats: [{ 'Fixture Chat': 'https://chat.example/' }] },
        },
      }
    }
    return { data: { success: true, data: {} } }
  })
  const post = vi.spyOn(api, 'post').mockImplementation(async (url) => {
    if (url === '/api/token/31/key') {
      return { data: { success: true, data: { key: 'fixture-not-a-key' } } }
    }
    throw new Error(`unexpected POST ${url}`)
  })
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
        <DialogProbe />
      </ApiKeysProvider>
    ),
  })
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([keys])]),
    history: createMemoryHistory({ initialEntries: ['/keys'] }),
  })
  const view = render(
    <QueryClientProvider client={client}>
      <PageFooterProvider container={footer}>
        <RouterProvider router={router} />
      </PageFooterProvider>
    </QueryClientProvider>
  )
  return { ...view, client, get, post, activityCalls: () => activityCalls }
}

const menu = () => document.querySelector('[role="menu"]')

async function openRowMenu(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('button', { name: 'Open menu' }))
  await waitFor(() => expect(menu()).not.toBeNull())
  return menu() as HTMLElement
}

describe('令牌行「更多」菜单', () => {
  test(
    '空等两个以上轮询周期及列表后台刷新时保持展开，Esc 与点击外部仍可关闭',
    { timeout: 20_000 },
    async () => {
      const user = userEvent.setup()
      const view = mountTable()
      await screen.findByText('Fixture token')
      await waitFor(() => expect(view.activityCalls()).toBeGreaterThan(0))
      const opened = await openRowMenu(user)
      const callsAtOpen = view.activityCalls()

      // 活动轮询 2s、时钟 1s：真实等待 4.5s 覆盖两个以上刷新周期。
      await act(() => new Promise((resolve) => setTimeout(resolve, 4500)))
      expect(view.activityCalls()).toBeGreaterThanOrEqual(callsAtOpen + 2)
      expect(menu()).toBe(opened)
      expect(opened).toBeVisible()

      await act(async () => {
        await view.client.invalidateQueries({ queryKey: ['keys'] })
      })
      expect(menu()).toBe(opened)

      await user.keyboard('{Escape}')
      await waitFor(() => expect(menu()).toBeNull())

      await openRowMenu(user)
      await user.click(document.body)
      await waitFor(() => expect(menu()).toBeNull())
    }
  )

  test('复制密钥、聊天子菜单与删除入口保持原有行为', async () => {
    const user = userEvent.setup()
    const view = mountTable()
    await screen.findByText('Fixture token')

    await openRowMenu(user)
    await waitFor(() =>
      expect(view.post).toHaveBeenCalledWith('/api/token/31/key')
    )
    await user.click(screen.getByRole('menuitem', { name: /Copy Key/ }))
    await waitFor(() =>
      expect(clipboard.copy).toHaveBeenCalledWith('sk-fixture-not-a-key')
    )

    await openRowMenu(user)
    await user.click(screen.getByRole('menuitem', { name: /Chat/ }))
    expect(
      await screen.findByRole('menuitem', { name: /Fixture Chat/ })
    ).toBeVisible()
    await user.keyboard('{Escape}')
    await user.keyboard('{Escape}')
    await waitFor(() => expect(menu()).toBeNull())

    await openRowMenu(user)
    await user.click(screen.getByRole('menuitem', { name: /Delete/ }))
    expect(screen.getByTestId('dialog-probe')).toHaveTextContent('delete:31')
    expect(view.post).not.toHaveBeenCalledWith(
      expect.stringContaining('delete'),
      expect.anything()
    )
  })
})
