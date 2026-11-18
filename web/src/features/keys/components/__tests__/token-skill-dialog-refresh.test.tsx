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
  within,
} from '@testing-library/react'
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

const template = {
  id: 7,
  title: 'Review',
  description: 'Review code',
  content: 'Check correctness',
  enabled: true,
  sort_order: 0,
  version: 1,
  group_ids: [12],
  group_details: [{ id: 12, name: 'Premium models' }],
}
const baseToken = apiKeySchema.parse({
  id: 410,
  name: 'Skill token',
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
const badgeName = /Manage installed skills/
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

function mountTable() {
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
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([keys])]),
    history: createMemoryHistory({ initialEntries: ['/keys'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <PageFooterProvider container={footer}>
        <RouterProvider router={router} />
      </PageFooterProvider>
    </QueryClientProvider>
  )
  return client
}

function mockListLifecycle() {
  const state = { promptCount: 1, hold: false, release: () => {} }
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/log/self/active') {
      return {
        data: {
          success: true,
          data: {
            summary: { total: 0, active_tokens: 0, by_token: {} },
            scope: 'instance',
            observed_at: Date.now(),
          },
        },
      }
    }
    if (url === `/api/token/${baseToken.id}/prompts`) {
      return {
        data: {
          success: true,
          data: [{ template, enabled: true, sort_order: 0 }],
        },
      }
    }
    const respond = () => ({
      data: {
        success: true,
        data: {
          items: [
            {
              ...baseToken,
              prompt_count: state.promptCount,
              enabled_prompt_count: state.promptCount,
            },
          ],
          total: 1,
          page: 1,
          page_size: 20,
        },
      },
    })
    if (!state.hold) return respond()
    return new Promise<ReturnType<typeof respond>>((resolve) => {
      state.release = () => resolve(respond())
    })
  })
  return state
}

describe('令牌技能弹窗跟随真实列表刷新生命周期', () => {
  test.each([false, true])(
    '刷新在途、计数 1→0→1 期间弹窗节点保持不变，Escape 后可在 0→1 后重新打开；手机=%s',
    async (mobile) => {
      if (mobile) {
        const original = window.matchMedia
        vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
          ...original(query),
          matches: query === '(max-width: 640px)',
        }))
      }
      const state = mockListLifecycle()
      const client = mountTable()
      fireEvent.click(
        await screen.findByRole('button', {
          name: 'Manage installed skills (1)',
        })
      )
      const dialog = await screen.findByRole('dialog')
      expect(await within(dialog).findByText('Review')).toBeVisible()

      // 暂扣刷新响应：请求在途期间也不能卸载弹窗。
      state.hold = true
      await act(async () => {
        void client.invalidateQueries({ queryKey: ['keys'] })
      })
      await waitFor(() =>
        expect(
          vi
            .mocked(api.get)
            .mock.calls.filter(([url]) =>
              String(url).startsWith('/api/token/?')
            )
        ).toHaveLength(2)
      )
      expect(screen.getByRole('dialog')).toBe(dialog)

      state.promptCount = 0
      await act(async () => state.release())
      await waitFor(() =>
        expect(
          screen.queryByRole('button', { name: badgeName, hidden: true })
        ).not.toBeInTheDocument()
      )
      expect(screen.getByRole('dialog')).toBe(dialog)
      expect(dialog).toBeVisible()

      fireEvent.keyDown(dialog, { key: 'Escape' })
      await waitFor(() =>
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      )
      // 计数仍为 0：焦点留在同一令牌行的稳定控件上，而不是页面根节点。
      await waitFor(() => expect(document.body).not.toHaveFocus())
      expect(
        document.activeElement?.closest('tr, [data-token-row="410"]')
      ).not.toBeNull()

      state.hold = false
      state.promptCount = 1
      await act(async () => {
        await client.invalidateQueries({ queryKey: ['keys'] })
      })
      fireEvent.click(
        await screen.findByRole('button', {
          name: 'Manage installed skills (1)',
        })
      )
      expect(await screen.findByRole('dialog')).toBeVisible()
    }
  )

  test.each([false, true])(
    '弹窗打开期间徽标经 1→0→1 重建后 Escape，焦点落到新徽标；手机=%s',
    async (mobile) => {
      if (mobile) {
        const original = window.matchMedia
        vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
          ...original(query),
          matches: query === '(max-width: 640px)',
        }))
      }
      const state = mockListLifecycle()
      const client = mountTable()
      fireEvent.click(
        await screen.findByRole('button', {
          name: 'Manage installed skills (1)',
        })
      )
      const dialog = await screen.findByRole('dialog')
      state.promptCount = 0
      await act(async () => {
        await client.invalidateQueries({ queryKey: ['keys'] })
      })
      await waitFor(() =>
        expect(
          screen.queryByRole('button', { name: badgeName, hidden: true })
        ).not.toBeInTheDocument()
      )
      state.promptCount = 1
      await act(async () => {
        await client.invalidateQueries({ queryKey: ['keys'] })
      })
      const recreated = await screen.findByRole('button', {
        name: 'Manage installed skills (1)',
        hidden: true,
      })
      fireEvent.keyDown(dialog, { key: 'Escape' })
      await waitFor(() =>
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      )
      await waitFor(() => expect(recreated).toHaveFocus())
    }
  )
})
