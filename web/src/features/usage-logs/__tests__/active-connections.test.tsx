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
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { useState, type ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { ActiveConnectionsPanel } from '../components/active-connections/active-connections-panel'
import { useActiveConnections } from '../components/active-connections/use-active-connections'
import { buildSearchParams } from '../lib/filter'
import { buildApiParams } from '../lib/utils'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))

// 图标包包含浏览器专用 JSON 资源，连接行为测试仅隔离第三方图标渲染。
vi.mock('@lobehub/icons', () => ({}))

function QueryWrapper(props: { children: ReactNode }) {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } })
  )
  return (
    <QueryClientProvider client={client}>{props.children}</QueryClientProvider>
  )
}

const snapshot = {
  items: [
    {
      id: 'one',
      model_name: 'long-model-name',
      token_name: 'token',
      request_type: 'openai_responses',
      is_stream: true,
      started_at: 1000,
      completed_at: 3000,
      status: 'completed',
      first_response_ms: null,
      retry_count: 0,
    },
  ],
  total: 1,
  page: 1,
  page_size: 3,
  observed_at: 4000,
  scope: 'cluster',
  summary: {
    total: 0,
    streaming: 0,
    non_streaming: 0,
    waiting: 0,
    active_tokens: 0,
  },
  subject: { id: 5215, username: 'someone' },
}

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.resetAllMocks()
})

describe('管理员连接查询与日志筛选', () => {
  it('提交用户 ID 时日志接口保留 ID 且不同时提交用户名', () => {
    const params = buildSearchParams(
      { userId: '5215', username: 'ignored' },
      'common'
    )
    expect(params).toMatchObject({ userId: '5215' })
    expect(params).not.toHaveProperty('username')
    expect(
      buildApiParams({
        page: 1,
        pageSize: 20,
        searchParams: params,
        columnFilters: [],
        isAdmin: true,
      })
    ).toMatchObject({ user_id: '5215' })
    expect(
      buildApiParams({
        page: 1,
        pageSize: 20,
        searchParams: params,
        columnFilters: [],
        isAdmin: false,
      })
    ).not.toHaveProperty('user_id')
  })

  it('切换目标用户时取消前一次请求，迟到响应不会覆盖新用户', async () => {
    let resolveOld: (value: unknown) => void = () => undefined
    let oldSignal: AbortSignal | undefined
    vi.mocked(api.get)
      .mockImplementationOnce((_url, config) => {
        oldSignal = config?.signal as AbortSignal
        return new Promise((resolve) => {
          resolveOld = resolve
        })
      })
      .mockResolvedValueOnce({ data: { success: true, data: snapshot } })
    const hook = renderHook(
      ({ userId }) =>
        useActiveConnections({ subject: { userId }, pageSize: 3 }),
      { wrapper: QueryWrapper, initialProps: { userId: '1' } }
    )
    await waitFor(() => expect(api.get).toHaveBeenCalled())
    hook.rerender({ userId: '5215' })
    await waitFor(() =>
      expect(hook.result.current.snapshot?.subject?.id).toBe(5215)
    )
    expect(oldSignal?.aborted).toBe(true)
    await act(async () => {
      resolveOld({
        data: {
          success: true,
          data: { ...snapshot, subject: { id: 1, username: 'old' } },
        },
      })
    })
    expect(hook.result.current.snapshot?.subject?.id).toBe(5215)
    expect(api.get).toHaveBeenLastCalledWith(
      '/api/log/active',
      expect.objectContaining({
        params: expect.objectContaining({ user_id: '5215', page_size: 3 }),
      })
    )
  })

  it('结束记录保留显示且耗时冻结，没有首响应时不伪装成等待', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: snapshot },
    })
    render(<ActiveConnectionsPanel subject={{ userId: '5215' }} />, {
      wrapper: QueryWrapper,
    })
    expect(
      await screen.findByText('Connections for someone (ID: 5215)')
    ).toBeInTheDocument()
    expect(screen.getByText('Request ended')).toBeInTheDocument()
    expect(screen.getAllByText('2.0s').length).toBeGreaterThan(0)
    expect(screen.getByText('No response received')).toBeInTheDocument()
    expect(
      screen.getByText('Ended requests remain visible for 5 seconds.')
    ).toBeInTheDocument()
    vi.useFakeTimers()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000)
    })
    expect(screen.getByText('2.0s')).toBeInTheDocument()
  })

  it('查询失败显示不可用并允许重试，不展示零连接', async () => {
    vi.mocked(api.get).mockRejectedValue(new Error('unavailable'))
    render(<ActiveConnectionsPanel subject={{ username: 'someone' }} />, {
      wrapper: QueryWrapper,
    })
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Connection status is unavailable.'
    )
    expect(screen.getByRole('button', { name: 'Retry' })).toBeEnabled()
    expect(
      screen.queryByText('This user has no current connections.')
    ).not.toBeInTheDocument()
  })
  it('默认只查询本人，不走管理员接口', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: snapshot },
    })
    const hook = renderHook(() => useActiveConnections({}), {
      wrapper: QueryWrapper,
    })
    await waitFor(() => expect(hook.result.current.snapshot).toBeDefined())
    expect(api.get).toHaveBeenCalledWith(
      '/api/log/self/active',
      expect.objectContaining({ params: { page: 1, page_size: 3 } })
    )
  })

  it('观察请求超时后显示错误，下一轮轮询自动恢复', async () => {
    vi.useFakeTimers()
    vi.mocked(api.get)
      .mockImplementationOnce(
        (_url, config) =>
          new Promise((_resolve, reject) => {
            if (config?.timeout) {
              window.setTimeout(
                () => reject(new Error('timeout')),
                config.timeout
              )
            }
          })
      )
      .mockResolvedValue({ data: { success: true, data: snapshot } })
    const hook = renderHook(() => useActiveConnections({}), {
      wrapper: QueryWrapper,
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10001)
    })
    expect(hook.result.current.error).toBeInstanceOf(Error)
    expect(hook.result.current.snapshot).toBeUndefined()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2001)
    })
    expect(api.get).toHaveBeenCalledTimes(2)
    expect(hook.result.current.error).toBeNull()
    expect(hook.result.current.snapshot?.subject?.id).toBe(5215)
    expect(api.get).toHaveBeenLastCalledWith(
      '/api/log/self/active',
      expect.objectContaining({ timeout: 10000 })
    )
  })

  it('隐藏页面触发真实中止异常后不残留错误，恢复时可继续查询', async () => {
    let visible = 'visible'
    vi.spyOn(document, 'visibilityState', 'get').mockImplementation(
      () => visible as DocumentVisibilityState
    )
    vi.mocked(api.get)
      .mockImplementationOnce(
        (_url, config) =>
          new Promise((_resolve, reject) => {
            config?.signal?.addEventListener?.('abort', () =>
              reject(new DOMException('aborted', 'AbortError'))
            )
          })
      )
      .mockResolvedValue({ data: { success: true, data: snapshot } })
    const hook = renderHook(() => useActiveConnections({}), {
      wrapper: QueryWrapper,
    })
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(1))
    await act(async () => {
      visible = 'hidden'
      document.dispatchEvent(new Event('visibilitychange'))
    })
    expect(hook.result.current.error).toBeNull()
    act(() => {
      visible = 'visible'
      document.dispatchEvent(new Event('visibilitychange'))
    })
    await waitFor(() =>
      expect(hook.result.current.snapshot?.subject?.id).toBe(5215)
    )
    expect(hook.result.current.error).toBeNull()
  })

  it('隐藏页面时取消在途查询，恢复可见后重新查询', async () => {
    let visible = 'visible'
    vi.spyOn(document, 'visibilityState', 'get').mockImplementation(
      () => visible as DocumentVisibilityState
    )
    let signal: AbortSignal | undefined
    vi.mocked(api.get)
      .mockImplementationOnce((_url, config) => {
        signal = config?.signal as AbortSignal
        return new Promise(() => undefined)
      })
      .mockResolvedValue({ data: { success: true, data: snapshot } })
    const hook = renderHook(() => useActiveConnections({}), {
      wrapper: QueryWrapper,
    })
    await waitFor(() => expect(signal).toBeDefined())
    act(() => {
      visible = 'hidden'
      document.dispatchEvent(new Event('visibilitychange'))
    })
    expect(signal?.aborted).toBe(true)
    act(() => {
      visible = 'visible'
      document.dispatchEvent(new Event('visibilitychange'))
    })
    await waitFor(() => expect(hook.result.current.snapshot).toBeDefined())
    expect(api.get).toHaveBeenCalledTimes(2)
  })

  it('预览最多三条，抽屉分页显示请求 ID，切用户清空抽屉', async () => {
    const items = Array.from({ length: 3 }, (_, index) => ({
      ...snapshot.items[0],
      id: `request-${index}`,
      request_id: `trace-${index}`,
    }))
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: { ...snapshot, items, total: 23 } },
    })
    const panel = render(
      <ActiveConnectionsPanel subject={{ userId: '5215' }} />,
      { wrapper: QueryWrapper }
    )
    expect(
      await screen.findAllByRole('button', {
        name: 'Connection details for long-model-name',
      })
    ).toHaveLength(3)
    fireEvent.click(
      screen.getByRole('button', { name: 'View all connections' })
    )
    const drawer = await screen.findByRole('dialog')
    expect(
      await within(drawer).findByText('Request ID: trace-0')
    ).toBeInTheDocument()
    await waitFor(() =>
      expect(api.get).toHaveBeenLastCalledWith(
        '/api/log/active',
        expect.objectContaining({
          params: expect.objectContaining({ page_size: 10, page: 1 }),
        })
      )
    )
    fireEvent.click(
      within(drawer).getByRole('button', { name: 'Next connections page' })
    )
    await waitFor(() =>
      expect(api.get).toHaveBeenLastCalledWith(
        '/api/log/active',
        expect.objectContaining({
          params: expect.objectContaining({ page_size: 10, page: 2 }),
        })
      )
    )
    fireEvent.change(
      within(drawer).getByRole('textbox', { name: 'Search connections' }),
      { target: { value: 'trace' } }
    )
    await waitFor(() =>
      expect(api.get).toHaveBeenLastCalledWith(
        '/api/log/active',
        expect.objectContaining({
          params: expect.objectContaining({
            page_size: 10,
            page: 1,
            search: 'trace',
          }),
        })
      )
    )
    panel.rerender(<ActiveConnectionsPanel subject={{ userId: '42' }} />)
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
    expect(api.get).toHaveBeenLastCalledWith(
      '/api/log/active',
      expect.objectContaining({
        params: expect.objectContaining({
          user_id: '42',
          page_size: 3,
          page: 1,
        }),
      })
    )
  })

  it('指定用户没有连接时保留身份，收起和展开不改变目标', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: { ...snapshot, items: [], total: 0 } },
    })
    render(<ActiveConnectionsPanel subject={{ userId: '5215' }} />, {
      wrapper: QueryWrapper,
    })
    expect(
      await screen.findByText('This user has no current connections.')
    ).toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', { name: 'Collapse connections' })
    )
    expect(
      screen.queryByText('This user has no current connections.')
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Expand connections' }))
    expect(
      screen.getByText('Connections for someone (ID: 5215)')
    ).toBeInTheDocument()
  })
})
