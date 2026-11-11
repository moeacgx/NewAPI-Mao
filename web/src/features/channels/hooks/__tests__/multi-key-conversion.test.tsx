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
import { act, cleanup, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { CHANNEL_FORM_DEFAULT_VALUES } from '../../lib/channel-form'
import { channelSchema, type Channel } from '../../types'
import { useChannelMutateForm } from '../use-channel-mutate-form'

const originalAuth = useAuthStore.getState().auth
let client: QueryClient
let channel: Channel

function QueryProvider(props: { children: ReactNode }) {
  return (
    <QueryClientProvider client={client}>{props.children}</QueryClientProvider>
  )
}

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  channel = channelSchema.parse({
    id: 42,
    name: 'Existing channel',
    key: '',
    type: 1,
    status: 1,
    created_time: 1,
    test_time: 0,
    response_time: 0,
    balance_updated_time: 0,
    models: 'gpt-4',
    group: 'default',
  })
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: {
        id: 2,
        username: 'operator',
        role: ROLE.ADMIN,
        permissions: {
          admin_permissions: {
            channel: {
              read: true,
              write: true,
              sensitive_write: true,
            },
          },
        },
      },
    },
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.setState({ auth: originalAuth })
})

test('非 Root 管理员有写入和敏感权限时可转换，空新密钥保留已有密钥', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const { result } = renderHook(
    () =>
      useChannelMutateForm({
        currentRow: channel,
        isEditing: true,
        isMultiKeyChannel: false,
        onSuccess: () => undefined,
      }),
    { wrapper: QueryProvider }
  )
  await act(async () => {
    await result.current.mutateAsync({
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'Existing channel',
      models: 'gpt-4',
      convert_to_multi_key: true,
      key: '',
      multi_key_type: 'polling',
    })
  })
  const payload = put.mock.calls[0]?.[1]
  expect(payload).toMatchObject({
    id: 42,
    convert_to_multi_key: true,
    multi_key_mode: 'polling',
  })
  expect(payload).not.toHaveProperty('key')
  expect(payload).not.toHaveProperty('key_mode')
})

test.each([
  [41, 1, ''],
  [57, 1, ''],
  [1, 41, ''],
  [1, 57, ''],
  [0, 1, ''],
  [1, 1, '{"source":"ionet"}'],
])(
  '来源 %s 目标 %s 元数据 %s 的非法转换在请求前被拒绝',
  async (source, target, metadata) => {
    channel.type = source
    channel.other_info = metadata
    const put = vi.spyOn(api, 'put')
    const { result } = renderHook(
      () =>
        useChannelMutateForm({
          currentRow: channel,
          isEditing: true,
          isMultiKeyChannel: false,
          onSuccess: () => undefined,
        }),
      { wrapper: QueryProvider }
    )
    await act(async () => {
      await expect(
        result.current.mutateAsync({
          ...CHANNEL_FORM_DEFAULT_VALUES,
          type: target,
          convert_to_multi_key: true,
        })
      ).rejects.toThrow("You don't have necessary permission")
    })
    expect(put).not.toHaveBeenCalled()
  }
)

test.each([
  [false, true],
  [true, false],
])(
  'write=%s sensitive_write=%s 时拒绝显式转换',
  async (write, sensitiveWrite) => {
    useAuthStore.setState({
      auth: {
        ...originalAuth,
        user: {
          id: 2,
          username: 'operator',
          role: ROLE.ADMIN,
          permissions: {
            admin_permissions: {
              channel: { write, sensitive_write: sensitiveWrite },
            },
          },
        },
      },
    })
    const put = vi.spyOn(api, 'put')
    const { result } = renderHook(
      () =>
        useChannelMutateForm({
          currentRow: channel,
          isEditing: true,
          isMultiKeyChannel: false,
          onSuccess: () => undefined,
        }),
      { wrapper: QueryProvider }
    )
    await act(async () => {
      await expect(
        result.current.mutateAsync({
          ...CHANNEL_FORM_DEFAULT_VALUES,
          convert_to_multi_key: true,
        })
      ).rejects.toThrow("You don't have necessary permission")
    })
    expect(put).not.toHaveBeenCalled()
  }
)

test('新建多 Key 渠道保留创建协议且不发送编辑转换标记', async () => {
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  const { result } = renderHook(
    () =>
      useChannelMutateForm({
        isEditing: false,
        isMultiKeyChannel: false,
        onSuccess: () => undefined,
      }),
    { wrapper: QueryProvider }
  )
  await act(async () => {
    await result.current.mutateAsync({
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'New pool',
      models: 'gpt-4',
      key: 'first-key\nsecond-key',
      multi_key_mode: 'multi_to_single',
      multi_key_type: 'polling',
    })
  })
  const payload = post.mock.calls[0]?.[1]
  expect(post.mock.calls[0]?.[0]).toBe('/api/channel/')
  expect(payload).toMatchObject({
    mode: 'multi_to_single',
    multi_key_mode: 'polling',
    channel: { key: 'first-key\nsecond-key', name: 'New pool' },
  })
  expect(payload).not.toHaveProperty('convert_to_multi_key')
  expect(payload).not.toHaveProperty('channel.convert_to_multi_key')
})
