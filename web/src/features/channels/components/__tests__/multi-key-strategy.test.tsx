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
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { channelSchema, type Channel } from '../../types'
import { ChannelsProvider } from '../channels-provider'
import { ChannelMutateDrawer } from '../drawers/channel-mutate-drawer'

// 图标资产不参与密钥策略交互，避免加载完整供应商图标库。
vi.mock('@/lib/lobe-icon', () => ({ getLobeIcon: () => null }))

const originalAuth = useAuthStore.getState().auth
let client: QueryClient
let channel: Channel

function renderChannel() {
  return render(
    <QueryClientProvider client={client}>
      <ChannelsProvider>
        <ChannelMutateDrawer
          open
          onOpenChange={() => undefined}
          currentRow={channel}
        />
      </ChannelsProvider>
    </QueryClientProvider>
  )
}

beforeEach(() => {
  channel = channelSchema.parse({
    id: 42,
    name: 'Existing channel',
    type: 1,
    key: '',
    status: 1,
    created_time: 1,
    test_time: 0,
    response_time: 0,
    balance_updated_time: 0,
    models: 'gpt-4',
    group: 'default',
    group_ids: [1],
    group_details: [{ id: 1, code: 'default', name: 'Default' }],
    channel_info: {
      is_multi_key: true,
      multi_key_size: 2,
      multi_key_mode: 'random',
    },
  })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: { id: 1, username: 'root', role: ROLE.SUPER_ADMIN },
    },
  })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/channel/42') {
      return { data: { success: true, data: channel } }
    }
    if (url === '/api/channel/models') {
      return { data: { success: true, data: [{ id: 'gpt-4' }] } }
    }
    if (url === '/api/group/') {
      return { data: { success: true, data: ['default'] } }
    }
    if (url === '/api/group/details') {
      return {
        data: {
          success: true,
          data: [{ id: 1, code: 'default', name: 'Default' }],
        },
      }
    }
    if (url === '/api/prefill_group') {
      return { data: { success: true, data: [] } }
    }
    if (url === '/api/vendors/') {
      return { data: { success: true, data: { items: [], total: 0 } } }
    }
    if (url === '/api/user/2fa/status' || url === '/api/user/passkey') {
      return { data: { success: true, data: { enabled: false } } }
    }
    throw new Error(`Unexpected GET ${url}`)
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.setState({ auth: originalAuth })
})

test.each([
  ['random', 'Random', 'polling', 'Polling'],
  ['polling', 'Polling', 'random', 'Random'],
] as const)(
  '编辑多 Key 渠道从 %s 切换策略时保留已有密钥',
  async (initial, initialLabel, next, nextLabel) => {
    channel.channel_info.multi_key_mode = initial
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    renderChannel()
    await screen.findByDisplayValue('Existing channel')
    const strategy = screen.getByRole('combobox', {
      name: 'Multi-Key Strategy',
    })
    expect(strategy).toHaveTextContent(initialLabel)
    await user.click(strategy)
    await user.click(screen.getByRole('option', { name: nextLabel }))
    expect(strategy).toHaveTextContent(nextLabel)
    await user.click(screen.getByRole('button', { name: 'Update Channel' }))
    await waitFor(() => expect(put).toHaveBeenCalled())
    expect(put.mock.calls[0]?.[1]).toMatchObject({
      id: 42,
      multi_key_mode: next,
      group_ids: [1],
    })
    expect(put.mock.calls[0]?.[1]).not.toHaveProperty('key')
    expect(put.mock.calls[0]?.[1]).not.toHaveProperty('key_mode')
  }
)

test('编辑单 Key 渠道时隐藏策略且不发送策略字段', async () => {
  channel.channel_info.is_multi_key = false
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  renderChannel()
  await screen.findByDisplayValue('Existing channel')
  expect(
    screen.queryByRole('combobox', { name: 'Multi-Key Strategy' })
  ).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Update Channel' }))
  await waitFor(() => expect(put).toHaveBeenCalled())
  expect(put.mock.calls[0]?.[1]).not.toHaveProperty('multi_key_mode')
})

test('缺少敏感写权限时禁用多 Key 策略并从更新请求省略策略', async () => {
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: {
        id: 2,
        username: 'operator',
        role: ROLE.ADMIN,
        permissions: {
          admin_permissions: {
            channel: { read: true, write: true, sensitive_write: false },
          },
        },
      },
    },
  })
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  renderChannel()
  await screen.findByDisplayValue('Existing channel')
  expect(
    screen.getByRole('combobox', { name: 'Multi-Key Strategy' })
  ).toBeDisabled()
  await userEvent.click(screen.getByRole('button', { name: 'Update Channel' }))
  await waitFor(() => expect(put).toHaveBeenCalled())
  expect(put.mock.calls[0]?.[1]).not.toHaveProperty('multi_key_mode')
  expect(put.mock.calls[0]?.[1]).not.toHaveProperty('key')
})

test.each([
  ['append', 'Append to existing keys'],
  ['replace', 'Replace all existing keys'],
] as const)(
  '修改策略并以 %s 模式提交新密钥时保留密钥更新契约',
  async (mode, label) => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    renderChannel()
    await screen.findByDisplayValue('Existing channel')
    await user.click(screen.getByRole('textbox', { name: 'API Key *' }))
    await user.paste('new-test-key')
    await user.click(screen.getByRole('combobox', { name: 'Key Update Mode' }))
    await user.click(screen.getByRole('option', { name: label }))
    await user.click(
      screen.getByRole('combobox', { name: 'Multi-Key Strategy' })
    )
    await user.click(screen.getByRole('option', { name: 'Polling' }))
    await user.click(screen.getByRole('button', { name: 'Update Channel' }))
    await waitFor(() => expect(put).toHaveBeenCalled())
    expect(put.mock.calls[0]?.[1]).toMatchObject({
      id: 42,
      key: 'new-test-key',
      key_mode: mode,
      multi_key_mode: 'polling',
    })
  }
)
