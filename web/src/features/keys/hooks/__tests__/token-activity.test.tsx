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
import { act, cleanup, renderHook, waitFor } from '@testing-library/react'
import { AxiosError } from 'axios'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { useApiKeysQuery } from '../use-api-keys-query'
import { useTokenActivity } from '../use-token-activity'

function queryWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return {
    client,
    wrapper: (props: { children: ReactNode }) => (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    ),
  }
}

function response(byToken: Record<string, number> = { 31: 2, 32: 1 }) {
  return {
    data: {
      success: true,
      data: {
        items: [
          {
            id: 'ended',
            token_id: 33,
            status: 'completed',
            completed_at: Date.now(),
          },
        ],
        total: 4,
        observed_at: Date.now(),
        scope: 'cluster',
        summary: {
          total: Object.values(byToken).reduce((sum, count) => sum + count, 0),
          active_tokens: Object.keys(byToken).length,
          by_token: byToken,
        },
      },
    },
  }
}

afterEach(() => {
  cleanup()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('本人令牌活动生命周期', () => {
  test('只读本人接口并使用全量摘要，排除预览页中结束保留记录', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue(response())
    const view = renderHook(
      () => useTokenActivity(41, 'session'),
      queryWrapper()
    )
    await waitFor(() =>
      expect(view.result.current.snapshot?.summary.active_tokens).toBe(2)
    )
    expect(view.result.current.snapshot?.summary.by_token).toEqual({
      31: 2,
      32: 1,
    })
    expect(get.mock.calls[0][0]).toBe('/api/log/self/active')
    expect(get.mock.calls[0][1]?.params).toEqual({ page: 1, page_size: 1 })
    expect(get.mock.calls[0][1]?.timeout).toBe(10000)
  })

  test('失败保留最后快照并标为陈旧，恢复为真实零活动', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue(response())
    const view = renderHook(
      () => useTokenActivity(41, 'session'),
      queryWrapper()
    )
    await waitFor(() =>
      expect(view.result.current.snapshot?.summary.total).toBe(3)
    )
    get.mockRejectedValue(new Error('timeout'))
    await act(async () => {
      await view.result.current.refresh()
    })
    await waitFor(() => expect(view.result.current.error).toBeTruthy())
    expect(view.result.current.snapshot?.summary.total).toBe(3)
    expect(view.result.current.stale).toBe(true)
    get.mockResolvedValue(response({}))
    await act(async () => {
      await view.result.current.refresh()
    })
    await waitFor(() => expect(view.result.current.error).toBeNull())
    expect(view.result.current.snapshot?.summary.total).toBe(0)
    expect(view.result.current.stale).toBe(false)
  })

  test('首次故障没有虚构零活动摘要', async () => {
    vi.spyOn(api, 'get').mockRejectedValue(new AxiosError('500'))
    const view = renderHook(
      () => useTokenActivity(41, 'session'),
      queryWrapper()
    )
    await waitFor(() => expect(view.result.current.error).toBeTruthy())
    expect(view.result.current.snapshot).toBeUndefined()
    expect(view.result.current.error).not.toBeInstanceOf(AxiosError)
  })

  test('隐藏与卸载取消进行中查询，切换身份不接收迟到快照', async () => {
    let finishOld: ((value: ReturnType<typeof response>) => void) | undefined
    const get = vi
      .spyOn(api, 'get')
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finishOld = resolve
          })
      )
      .mockResolvedValue(response({ 44: 1 }))
    const { client, wrapper } = queryWrapper()
    const view = renderHook(
      ({ userId }) => useTokenActivity(userId, 'session'),
      { wrapper, initialProps: { userId: 41 } }
    )
    await waitFor(() => expect(get).toHaveBeenCalledTimes(1))
    const oldSignal = get.mock.calls[0][1]?.signal
    view.rerender({ userId: 42 })
    expect(oldSignal?.aborted).toBe(true)
    expect(view.result.current.snapshot).toBeUndefined()
    await waitFor(() =>
      expect(view.result.current.snapshot?.summary.by_token).toEqual({ 44: 1 })
    )
    await act(async () => {
      finishOld?.(response())
    })
    expect(view.result.current.snapshot?.summary.by_token).toEqual({ 44: 1 })
    expect(
      client.getQueryData(['token-activity', 41, 'session'])
    ).toBeUndefined()
    get.mockImplementationOnce(() => new Promise(() => {}))
    act(() => {
      void view.result.current.refresh()
    })
    const hiddenSignal = get.mock.calls.at(-1)?.[1]?.signal
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden')
    act(() => document.dispatchEvent(new Event('visibilitychange')))
    expect(hiddenSignal?.aborted).toBe(true)
    const callsBefore = get.mock.calls.length
    vi.useFakeTimers()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(6000)
    })
    expect(get).toHaveBeenCalledTimes(callsBefore)
    vi.useRealTimers()
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    act(() => document.dispatchEvent(new Event('visibilitychange')))
    await waitFor(() =>
      expect(get.mock.calls.length).toBeGreaterThan(callsBefore)
    )
    get.mockImplementationOnce(() => new Promise(() => {}))
    act(() => {
      void view.result.current.refresh()
    })
    const unmountSignal = get.mock.calls.at(-1)?.[1]?.signal
    view.unmount()
    expect(unmountSignal?.aborted).toBe(true)
    expect(
      client.getQueryData(['token-activity', 42, 'session'])
    ).toBeUndefined()
  })

  test('未登录不发活动请求', () => {
    const get = vi.spyOn(api, 'get')
    renderHook(() => useTokenActivity(undefined, ''), queryWrapper())
    expect(get).not.toHaveBeenCalled()
  })
})

describe('活动筛选刷新和缓存隔离', () => {
  test('状态变化和切账号期间不展示前一查询的数据', async () => {
    const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
      if (url === '/api/log/self/active') return response()
      return {
        data: {
          success: true,
          data: {
            items: [{ id: 31, name: 'owner only' }],
            total: 1,
            page: 1,
            page_size: 20,
          },
        },
      }
    })
    const view = renderHook(
      ({ userId, status }) => {
        const activity = useTokenActivity(userId, 'session')
        return useApiKeysQuery({
          userId,
          sessionId: 'session',
          params: { status, p: 1, size: 20 },
          refreshTrigger: 0,
          activity,
        })
      },
      { ...queryWrapper(), initialProps: { userId: 41, status: 1 } }
    )
    await waitFor(() =>
      expect(view.result.current.data?.items[0].name).toBe('owner only')
    )
    get.mockImplementation(async (url) =>
      url === '/api/log/self/active' ? response() : new Promise(() => {})
    )
    view.rerender({ userId: 41, status: 2 })
    expect(view.result.current.data).toBeUndefined()
    const signal = get.mock.calls.at(-1)?.[1]?.signal
    view.rerender({ userId: 42, status: 2 })
    expect(view.result.current.data).toBeUndefined()
    expect(signal?.aborted).toBe(true)
  })

  test('活动 ID 不变不重查列表，ID 改变与故障恢复分别刷新一次', async () => {
    let activityResponse = response({ 31: 1 })
    let unavailable = false
    const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
      if (url === '/api/log/self/active') {
        if (unavailable) throw new Error('offline')
        return activityResponse
      }
      return {
        data: {
          success: true,
          data: { items: [], total: 0, page: 1, page_size: 20 },
        },
      }
    })
    const view = renderHook(() => {
      const activity = useTokenActivity(41, 'session')
      const list = useApiKeysQuery({
        userId: 41,
        sessionId: 'session',
        params: {
          active_only: true,
          status: 2,
          keyword: 'same',
          token: 'needle',
          p: 1,
          size: 20,
        },
        refreshTrigger: 0,
        activity,
      })
      return { activity, list }
    }, queryWrapper())
    await waitFor(() =>
      expect(view.result.current.activity.snapshot).toBeDefined()
    )
    await waitFor(() => expect(view.result.current.list.isFetching).toBe(false))
    const searches = () =>
      get.mock.calls.filter(([url]) => url.startsWith('/api/token/search'))
    const baseline = searches().length
    activityResponse = response({ 31: 2 })
    await act(async () => {
      await view.result.current.activity.refresh()
    })
    expect(searches()).toHaveLength(baseline)
    activityResponse = response({ 32: 1 })
    await act(async () => {
      await view.result.current.activity.refresh()
    })
    await waitFor(() => expect(searches()).toHaveLength(baseline + 1))
    unavailable = true
    await act(async () => {
      await view.result.current.activity.refresh()
    })
    await waitFor(() => expect(view.result.current.activity.error).toBeTruthy())
    expect(searches()).toHaveLength(baseline + 1)
    unavailable = false
    await act(async () => {
      await view.result.current.activity.refresh()
    })
    await waitFor(() => expect(searches()).toHaveLength(baseline + 2))
    const params = new URL(searches().at(-1)?.[0] || '', 'http://localhost')
      .searchParams
    expect(Object.fromEntries(params)).toEqual({
      keyword: 'same',
      token: 'needle',
      status: '2',
      active_only: 'true',
      p: '1',
      size: '20',
    })
  })

  test('普通列表不因活动 ID 改变重新搜索', async () => {
    let activityResponse = response({ 31: 1 })
    const get = vi
      .spyOn(api, 'get')
      .mockImplementation(async (url) =>
        url === '/api/log/self/active'
          ? activityResponse
          : { data: { success: true, data: { items: [], total: 0 } } }
      )
    const view = renderHook(() => {
      const activity = useTokenActivity(41, '')
      const list = useApiKeysQuery({
        userId: 41,
        sessionId: '',
        params: {},
        refreshTrigger: 0,
        activity,
      })
      return { activity, list }
    }, queryWrapper())
    await waitFor(() =>
      expect(view.result.current.activity.snapshot).toBeDefined()
    )
    const before = get.mock.calls.filter(([url]) =>
      url.startsWith('/api/token/')
    ).length
    activityResponse = response({ 32: 1 })
    await act(async () => {
      await view.result.current.activity.refresh()
    })
    expect(
      get.mock.calls.filter(([url]) => url.startsWith('/api/token/'))
    ).toHaveLength(before)
  })
})
