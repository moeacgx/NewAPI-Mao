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
import type { ReactNode } from 'react'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { useApiKeysQuery } from '../use-api-keys-query'
import { useTokenActivity } from '../use-token-activity'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

function listResponse(id: number) {
  return {
    data: {
      success: true,
      data: {
        items: [{ id, name: `token-${id}` }],
        total: 1,
        page: 1,
        page_size: 20,
      },
    },
  }
}

function mountPendingList(activeOnly = true) {
  let activeId = 31
  let unavailable = false
  const requests: {
    signal?: AbortSignal
    finish: (value: ReturnType<typeof listResponse>) => void
  }[] = []
  vi.spyOn(api, 'get').mockImplementation(async (url, config) => {
    if (url === '/api/log/self/active') {
      if (unavailable) throw new Error('offline')
      return {
        data: {
          success: true,
          data: {
            summary: {
              total: 1,
              active_tokens: 1,
              by_token: { [activeId]: 1 },
            },
            scope: 'instance',
            observed_at: Date.now(),
          },
        },
      }
    }
    return new Promise<ReturnType<typeof listResponse>>((finish) => {
      requests.push({
        signal: config?.signal as AbortSignal | undefined,
        finish,
      })
    })
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const view = renderHook(
    ({ userId, sessionId }) => {
      const activity = useTokenActivity(userId, sessionId)
      const keys = useApiKeysQuery({
        userId,
        sessionId,
        params: { active_only: activeOnly, p: 1, size: 20 },
        refreshTrigger: 0,
        activity,
      })
      return { activity, keys }
    },
    {
      initialProps: { userId: 41, sessionId: 'session-a' },
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    }
  )
  return {
    ...view,
    requests,
    client,
    refresh: async (id: number, failed = false) => {
      activeId = id
      unavailable = failed
      await act(async () => {
        await view.result.current.activity.refresh()
      })
      await waitFor(() => {
        expect(Boolean(view.result.current.activity.error)).toBe(failed)
        if (!failed) {
          expect(
            view.result.current.activity.snapshot?.summary.by_token
          ).toEqual({ [id]: 1 })
        }
      })
    },
  }
}

test('首次列表在途连续活动变化合并为完成后一次补查，相同集合不再重查', async () => {
  const view = mountPendingList()
  await waitFor(() =>
    expect(view.result.current.activity.snapshot).toBeDefined()
  )
  expect(view.requests).toHaveLength(1)
  expect(view.result.current.keys.data).toBeUndefined()
  await view.refresh(32)
  await view.refresh(33)
  expect(view.requests).toHaveLength(1)
  await act(async () => view.requests[0].finish(listResponse(31)))
  await waitFor(() => expect(view.requests).toHaveLength(2))
  await act(async () => view.requests[1].finish(listResponse(33)))
  await waitFor(() =>
    expect(view.result.current.keys.data?.items[0].id).toBe(33)
  )
  await view.refresh(33)
  await view.refresh(33)
  expect(view.requests).toHaveLength(2)
})

test('首次请求期间故障后同集合恢复也在完成后补查一次', async () => {
  const view = mountPendingList()
  await waitFor(() =>
    expect(view.result.current.activity.snapshot).toBeDefined()
  )
  await view.refresh(31, true)
  await view.refresh(31)
  await act(async () => view.requests[0].finish(listResponse(30)))
  await waitFor(() => expect(view.requests).toHaveLength(2))
  await act(async () => view.requests[1].finish(listResponse(31)))
  await waitFor(() =>
    expect(view.result.current.keys.data?.items[0].id).toBe(31)
  )
  await view.refresh(31)
  expect(view.requests).toHaveLength(2)
})

test.each(['subject', 'session', 'unmount'] as const)(
  '待补查期间 %s 变化取消旧请求，迟到结果不触发旧主体补查',
  async (change) => {
    const view = mountPendingList()
    await waitFor(() =>
      expect(view.result.current.activity.snapshot).toBeDefined()
    )
    await view.refresh(32)
    const oldRequest = view.requests[0]
    if (change === 'unmount') {
      view.unmount()
    } else {
      view.rerender({
        userId: change === 'subject' ? 42 : 41,
        sessionId: change === 'session' ? 'session-b' : 'session-a',
      })
    }
    expect(oldRequest.signal?.aborted).toBe(true)
    const count = view.requests.length
    await act(async () => oldRequest.finish(listResponse(31)))
    expect(view.requests).toHaveLength(count)
    if (change !== 'unmount') {
      expect(view.result.current.keys.data).toBeUndefined()
      expect(view.requests).toHaveLength(2)
      view.unmount()
      expect(view.requests[1].signal?.aborted).toBe(true)
    }
  }
)

test('普通列表在途活动变化和故障恢复不增加请求', async () => {
  const view = mountPendingList(false)
  await waitFor(() =>
    expect(view.result.current.activity.snapshot).toBeDefined()
  )
  await view.refresh(32)
  await view.refresh(32, true)
  await view.refresh(32)
  await act(async () => view.requests[0].finish(listResponse(31)))
  await waitFor(() => expect(view.result.current.keys.isFetching).toBe(false))
  await view.refresh(33)
  expect(view.requests).toHaveLength(1)
  expect(view.result.current.keys.data?.items[0].id).toBe(31)
})

test('旧请求及补查均失败时保留错误，同集合轮询不形成重试循环', async () => {
  const view = mountPendingList()
  await waitFor(() =>
    expect(view.result.current.activity.snapshot).toBeDefined()
  )
  await view.refresh(32)
  const failed = listResponse(31)
  failed.data.success = false
  await act(async () => view.requests[0].finish(failed))
  await waitFor(() => expect(view.requests).toHaveLength(2))
  await act(async () => view.requests[1].finish(failed))
  await waitFor(() => expect(view.result.current.keys.isError).toBe(true))
  await view.refresh(32)
  await view.refresh(32)
  expect(view.requests).toHaveLength(2)
  expect(view.result.current.keys.isFetching).toBe(false)
  expect(view.result.current.keys.data).toBeUndefined()
})
