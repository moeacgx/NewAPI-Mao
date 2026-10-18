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
import { act, renderHook } from '@testing-library/react'
import type { PropsWithChildren } from 'react'
import { toast } from 'sonner'
import { beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { channelsQueryKeys } from '../../lib'
import type { Channel } from '../../types'
import { useChannelPriorityOrder } from '../use-channel-priority-order'

const drop = { id: 3, target_id: 2, position: 'before' } as const
const visible = [
  { id: 1, priority: 100 },
  { id: 2, priority: 80 },
  { id: 3, priority: 0 },
] as Channel[]

beforeEach(() => {
  vi.spyOn(console, 'log').mockImplementation(() => undefined)
  vi.spyOn(toast, 'error').mockImplementation(() => '')
  vi.spyOn(toast, 'success').mockImplementation(() => '')
  vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      success: true,
      data: {
        revision: 'saved',
        items: [
          { id: 1, priority: 100 },
          { id: 2, priority: 80 },
          { id: 3, priority: 90 },
        ],
        changes: [{ id: 3, previous: 0, priority: 90 }],
      },
    },
  })
})

function mount(channels = visible) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const queryKey = channelsQueryKeys.lists()
  client.setQueryData(queryKey, channels)
  const hook = renderHook(
    (props: { channels: Channel[] }) =>
      useChannelPriorityOrder({
        channels: props.channels,
        desc: true,
        scope: 'visible-page',
        enabled: true,
      }),
    {
      initialProps: { channels },
      wrapper: (props: PropsWithChildren) => (
        <QueryClientProvider client={client}>
          {props.children}
        </QueryClientProvider>
      ),
    }
  )
  return { ...hook, client, queryKey }
}

test.each([2, 3])(
  'a concurrent edit to visible channel %s rejects the stale target despite a fresh revision',
  async (changedID) => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          revision: 'other-admin-edit',
          items: visible.map((item) => ({
            id: item.id,
            priority: item.id === changedID ? 120 : item.priority,
          })),
          changes: [],
        },
      },
    })
    const hook = mount()

    act(() => {
      hook.result.current.begin()
      hook.result.current.select(drop)
    })
    await act(() => hook.result.current.commit())

    expect(api.post).not.toHaveBeenCalled()
    expect(hook.client.getQueryState(hook.queryKey)?.isInvalidated).toBe(true)
    expect(toast.error).toHaveBeenCalledExactlyOnceWith(
      'Channel priorities changed. Refresh and try again.'
    )
    expect(hook.result.current.items).toEqual(visible)
    expect(hook.result.current.active).toBe(false)
    expect(hook.result.current.undo).toBeNull()
  }
)

test.each([2, 3])(
  'a missing visible channel %s conflicts even when its priority was null',
  async (missingID) => {
    const channels = visible.map((item) => ({
      ...item,
      priority: item.id === missingID ? null : item.priority,
    }))
    vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          revision: 'other-admin-delete',
          items: channels.filter((item) => item.id !== missingID),
          changes: [],
        },
      },
    })
    const hook = mount(channels)

    act(() => {
      hook.result.current.begin()
      hook.result.current.select(drop)
    })
    await act(() => hook.result.current.commit())

    expect(api.post).not.toHaveBeenCalled()
    expect(hook.client.getQueryState(hook.queryKey)?.isInvalidated).toBe(true)
    expect(toast.error).toHaveBeenCalledExactlyOnceWith(
      'Channel priorities changed. Refresh and try again.'
    )
    expect(hook.result.current.items).toEqual(channels)
  }
)

test.each([
  { visiblePriority: undefined, snapshotPriority: null },
  { visiblePriority: null, snapshotPriority: undefined },
])(
  'visible priority $visiblePriority matches snapshot priority $snapshotPriority for an existing channel',
  async ({ visiblePriority, snapshotPriority }) => {
    const channels = visible.map((item) => ({
      ...item,
      priority: item.id === 3 ? visiblePriority : item.priority,
    }))
    vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          revision: 'matching-snapshot',
          // 全局快照可包含当前筛选外的渠道，且不要求与页面顺序一致。
          items: [
            { id: 4, priority: 200 },
            ...channels
              .map((item) => ({
                id: item.id,
                priority: item.id === 3 ? snapshotPriority : item.priority,
              }))
              .reverse(),
          ],
          changes: [],
        },
      },
    })
    const hook = mount(channels)

    act(() => {
      hook.result.current.begin()
      hook.result.current.select(drop)
    })
    await act(() => hook.result.current.commit())

    expect(api.post).toHaveBeenCalledExactlyOnceWith(
      '/api/channel/priority/reorder',
      { ...drop, revision: 'matching-snapshot' }
    )
    expect(toast.error).not.toHaveBeenCalled()
    expect(hook.result.current.undo?.revision).toBe('saved')
  }
)

test('a list refresh during a gesture cannot replace the expected visible snapshot', async () => {
  const updated = visible.map((item) => ({
    ...item,
    priority: item.id === 2 ? 120 : item.priority,
  }))
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: { revision: 'other-admin-edit', items: updated, changes: [] },
    },
  })
  const hook = mount()

  act(() => hook.result.current.begin())
  hook.rerender({ channels: updated })
  act(() => hook.result.current.select(drop))
  await act(() => hook.result.current.commit())

  expect(api.post).not.toHaveBeenCalled()
  expect(hook.client.getQueryState(hook.queryKey)?.isInvalidated).toBe(true)
  expect(toast.error).toHaveBeenCalledExactlyOnceWith(
    'Channel priorities changed. Refresh and try again.'
  )
})
