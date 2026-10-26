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
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { PendingTicketCard } from '../pending-ticket-card'
import { PendingTicketProvider } from '../pending-ticket-provider'

const router = vi.hoisted(() => ({ pathname: '/dashboard' }))
vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))
vi.mock('@tanstack/react-router', () => ({
  useLocation: ({ select }: { select: (l: { pathname: string }) => string }) =>
    select({ pathname: router.pathname }),
  Link: (props: {
    to: string
    search?: unknown
    children: React.ReactNode
  }) => (
    <a href={`${props.to}?${JSON.stringify(props.search)}`}>{props.children}</a>
  ),
}))

const admin = {
  id: 7,
  username: 'admin',
  role: 10,
  permissions: { admin_permissions: { tickets: { read: true, write: true } } },
}
const otherAdmin = { ...admin, id: 8, username: 'other-admin' }

function response<T>(data: T) {
  return { data: { success: true, message: '', data } }
}

function statsOf(open: number) {
  return { open, processing: 11, resolved: 22, closed: 33, archived: 44 }
}

function mockServer(self: unknown, open: number | Error) {
  vi.mocked(api.get).mockImplementation(async (url) => {
    if (url === '/api/user/self') return response(self)
    if (open instanceof Error) throw open
    return response(statsOf(open))
  })
}

function setVisibility(state: 'visible' | 'hidden') {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    get: () => state,
  })
}

function renderPending() {
  const client = new QueryClient()
  const view = render(
    <QueryClientProvider client={client}>
      <PendingTicketProvider>
        <PendingTicketCard />
      </PendingTicketProvider>
    </QueryClientProvider>
  )
  return { client, ...view }
}

async function expectCount(count: string) {
  await waitFor(() =>
    expect(screen.getByLabelText('Pending ticket count')).toHaveTextContent(
      count
    )
  )
}

beforeEach(() => {
  router.pathname = '/dashboard'
  setVisibility('visible')
  useAuthStore.getState().auth.setUser(admin)
})
afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
})

describe('待处理工单提醒', () => {
  it('服务端复核管理员后按 open 口径查询，并显示可关闭的浮动提醒', async () => {
    mockServer(admin, 3)
    renderPending()
    expect(await screen.findByRole('complementary')).toHaveTextContent('3')
    expect(screen.getByLabelText('Pending ticket count')).toHaveTextContent('3')
    expect(
      screen
        .getByRole('button', { name: 'Refresh' })
        .querySelector('svg[aria-hidden="true"]')
    ).not.toBeNull()
    expect(
      within(screen.getByLabelText('Ticket overview'))
        .getByRole('link', { name: 'View tickets' })
        .querySelectorAll('svg[aria-hidden="true"]')
    ).toHaveLength(1)
    expect(api.get).toHaveBeenCalledWith(
      '/api/ticket-admin/stats',
      expect.objectContaining({
        skipErrorHandler: true,
        skipBusinessError: true,
      })
    )
    expect(api.get).not.toHaveBeenCalledWith(
      '/api/ticket-admin',
      expect.anything()
    )
    await userEvent.click(screen.getByLabelText('Dismiss ticket reminder'))
    expect(screen.queryByRole('complementary')).toBeNull()
  })

  it('数量为 0 时不弹出浮动提醒', async () => {
    mockServer(admin, 0)
    renderPending()
    await expectCount('0')
    expect(screen.queryByRole('complementary')).toBeNull()
  })

  it('普通用户不会请求任何管理接口', async () => {
    useAuthStore.getState().auth.setUser({ ...admin, role: 1 })
    mockServer(admin, 3)
    renderPending()
    await act(async () => {})
    expect(api.get).not.toHaveBeenCalled()
    expect(screen.queryByRole('complementary')).toBeNull()
  })

  it('服务端确认权限已被收回时隐藏全部内容', async () => {
    mockServer({ ...admin, permissions: { admin_permissions: {} } }, 3)
    renderPending()
    await waitFor(() =>
      expect(screen.queryByLabelText('Pending ticket count')).toBeNull()
    )
    expect(screen.queryByRole('complementary')).toBeNull()
  })

  it('数量请求失败时只在卡片内显示错误，不弹出浮动提醒', async () => {
    mockServer(admin, new Error('boom'))
    renderPending()
    expect(
      await screen.findByText('Failed to load ticket overview')
    ).toBeInTheDocument()
    expect(screen.queryByRole('complementary')).toBeNull()
  })

  it('在工单管理页保持浮动提醒隐藏，数量仍可用', async () => {
    router.pathname = '/ticket-management'
    mockServer(admin, 4)
    renderPending()
    await expectCount('4')
    expect(screen.queryByRole('complementary')).toBeNull()
  })
})

describe('待处理工单异步状态', () => {
  it('账号 A 请求未返回时切换到 B：A 的请求被取消且结果不泄漏', async () => {
    let resolveA!: (value: ReturnType<typeof response>) => void
    let signalA: AbortSignal | undefined
    vi.mocked(api.get).mockImplementation(async (url, config) => {
      const user = useAuthStore.getState().auth.user
      if (user?.id === admin.id && url === '/api/user/self') {
        signalA = config?.signal as AbortSignal | undefined
        return new Promise((resolve) => {
          resolveA = resolve
        })
      }
      if (url === '/api/user/self') return response(otherAdmin)
      return response(statsOf(5))
    })
    renderPending()
    await waitFor(() => expect(signalA).toBeDefined())

    act(() => useAuthStore.getState().auth.setUser(otherAdmin))
    await expectCount('5')
    expect(signalA?.aborted).toBe(true)

    await act(async () => {
      resolveA(response(admin))
    })
    expect(screen.getByLabelText('Pending ticket count')).toHaveTextContent('5')
    expect(screen.getByRole('complementary')).toHaveTextContent('5')
  })

  it('已有数量后请求超时会清空旧数量，点击重试后恢复；两个请求都限时 10 秒', async () => {
    const timeout = Object.assign(new Error('timeout'), {
      code: 'ECONNABORTED',
    })
    let failing = false
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (failing) throw timeout
      if (url === '/api/user/self') return response(admin)
      return response(statsOf(3))
    })
    renderPending()
    await expectCount('3')

    failing = true
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(
      await screen.findByText('Failed to load ticket overview')
    ).toBeInTheDocument()
    expect(screen.queryByLabelText('Pending ticket count')).toBeNull()

    failing = false
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await expectCount('3')

    const timeouts = vi.mocked(api.get).mock.calls.map(([, c]) => c?.timeout)
    expect(timeouts.length).toBeGreaterThanOrEqual(4)
    expect(new Set(timeouts)).toEqual(new Set([10000]))
  })

  it('invalidateQueries([tickets, userId]) 会实际刷新数量', async () => {
    mockServer(admin, 3)
    const { client } = renderPending()
    await expectCount('3')

    mockServer(admin, 6)
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['tickets', admin.id] })
    })
    await expectCount('6')
  })

  it('关闭提醒后切换页面和新增数量都不会重新弹出', async () => {
    mockServer(admin, 3)
    const { client } = renderPending()
    await userEvent.click(
      await screen.findByLabelText('Dismiss ticket reminder')
    )

    const refetch = async (total: number) => {
      mockServer(admin, total)
      await act(async () => {
        await client.invalidateQueries({ queryKey: ['tickets', admin.id] })
      })
      await expectCount(String(total))
    }
    router.pathname = '/ticket-management'
    await refetch(8)
    router.pathname = '/dashboard'
    await refetch(9)
    expect(screen.queryByRole('complementary')).toBeNull()
  })

  it('页面可见时每 60 秒轮询，隐藏时暂停，回到可见并聚焦后立即刷新', async () => {
    vi.useFakeTimers()
    mockServer(admin, 3)
    renderPending()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(api.get).toHaveBeenCalledTimes(2)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(60000)
    })
    expect(api.get).toHaveBeenCalledTimes(4)

    setVisibility('hidden')
    await act(async () => {
      await vi.advanceTimersByTimeAsync(120000)
    })
    expect(api.get).toHaveBeenCalledTimes(4)

    setVisibility('visible')
    await act(async () => {
      window.dispatchEvent(new Event('focus'))
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(api.get).toHaveBeenCalledTimes(6)
  })
})
