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
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { usePlaygroundConversation } from '../use-playground-conversation'
import { usePlaygroundOptions } from '../use-playground-options'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))

beforeEach(() => {
  vi.mocked(api.get).mockReset()
})

describe('令牌模型切换', () => {
  it('模型未就绪时快捷提示和重试不能新增等待消息或发送请求', () => {
    const updateMessages = vi.fn()
    const sendChat = vi.fn()
    const message = {
      key: 'user-1',
      from: 'user' as const,
      versions: [{ id: 'v1', content: 'Hello' }],
    }
    const { result } = renderHook(() =>
      usePlaygroundConversation({
        messages: [message],
        updateMessages,
        sendChat,
        canSend: false,
      })
    )
    act(() => {
      result.current.handleSendMessage('Hello')
      result.current.handleRegenerateMessage(message)
    })
    expect(sendChat).not.toHaveBeenCalled()
    expect(updateMessages).not.toHaveBeenCalled()
  })

  it('切换后仅提供令牌模型并清理失效模型，失败时清空可选项且不回退账号', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === '/api/user/self/groups') {
        return {
          data: {
            success: true,
            data: { account: { name: 'Account group', ratio: 1 } },
          },
        }
      }
      if (url === '/pg/token/models') {
        return { data: { data: [{ id: 'token-only' }] } }
      }
      return { data: { success: true, data: ['account-only'] } }
    })
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const wrapper = (props: { children: ReactNode }) => (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    )
    const updateConfig = vi.fn()
    const { result, rerender } = renderHook(
      ({ tokenId }: { tokenId?: number }) =>
        usePlaygroundOptions({
          tokenId,
          currentModel: 'account-only',
          currentGroup: 'account',
          setGroups: vi.fn(),
          setModels: vi.fn(),
          updateConfig,
        }),
      { initialProps: { tokenId: undefined as number | undefined }, wrapper }
    )
    await waitFor(() =>
      expect(result.current.models).toEqual([
        { value: 'account-only', label: 'account-only' },
      ])
    )
    rerender({ tokenId: 7 })
    await waitFor(() =>
      expect(result.current.models).toEqual([
        { value: 'token-only', label: 'token-only' },
      ])
    )
    expect(updateConfig).toHaveBeenCalledWith('model', 'token-only')
    const previousAccountRequests = vi
      .mocked(api.get)
      .mock.calls.filter(([url]) => url === '/api/user/models').length
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === '/pg/token/models') throw new Error('Token disabled')
      throw new Error(`Unexpected account fallback: ${String(url)}`)
    })
    await act(async () => {
      await result.current.refetchModels()
    })
    await waitFor(() => expect(result.current.isModelsError).toBe(true))
    expect(result.current.models).toEqual([])
    expect(
      vi
        .mocked(api.get)
        .mock.calls.filter(([url]) => url === '/api/user/models')
    ).toHaveLength(previousAccountRequests)
  })
})
