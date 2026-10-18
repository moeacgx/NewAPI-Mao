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

// 图标资产不参与渠道设置交互，避免加载完整供应商图标库。
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

test.each([false, true])(
  '强制 Responses 开关从 %s 切换后写入渠道设置',
  async (enabled) => {
    channel.setting = JSON.stringify({
      force_responses: enabled,
      proxy: 'http://proxy.example:8080',
    })
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    renderChannel()
    const toggle = await screen.findByRole('switch', {
      name: 'Force Responses upstream',
    })
    expect(toggle).toHaveAttribute('aria-checked', String(enabled))
    await user.click(toggle)
    await user.click(screen.getByRole('button', { name: 'Update Channel' }))
    await waitFor(() => expect(put).toHaveBeenCalled())
    const payload = put.mock.calls[0]?.[1] as { setting: string }
    expect(JSON.parse(payload.setting)).toMatchObject({
      force_responses: !enabled,
      proxy: 'http://proxy.example:8080',
    })
  }
)

test('仅强制 Responses 已开启时自动展开并受敏感写权限限制', async () => {
  channel.setting = JSON.stringify({ force_responses: true })
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
  const toggle = await screen.findByRole('switch', {
    name: 'Force Responses upstream',
  })
  expect(toggle).toHaveAttribute('aria-disabled', 'true')
  expect(toggle).toHaveAttribute('aria-checked', 'true')
  await userEvent.click(screen.getByRole('button', { name: 'Update Channel' }))
  await waitFor(() => expect(put).toHaveBeenCalled())
  expect(put.mock.calls[0]?.[1]).not.toHaveProperty('setting')
})

test('切换到不支持的渠道类型时关闭并隐藏强制 Responses', async () => {
  channel.setting = JSON.stringify({ force_responses: true })
  const user = userEvent.setup()
  renderChannel()
  await screen.findByRole('switch', { name: 'Force Responses upstream' })
  await user.click(screen.getByPlaceholderText('Search channel type...'))
  await user.click(await screen.findByRole('option', { name: 'Anthropic' }))
  expect(
    screen.queryByRole('switch', { name: 'Force Responses upstream' })
  ).not.toBeInTheDocument()
  await user.click(screen.getByPlaceholderText('Search channel type...'))
  await user.click(await screen.findByRole('option', { name: 'OpenAI' }))
  expect(
    screen.getByRole('switch', { name: 'Force Responses upstream' })
  ).toHaveAttribute('aria-checked', 'false')
})
