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

function channelEditor(currentRow: Channel | null, open = true) {
  return (
    <QueryClientProvider client={client}>
      <ChannelsProvider>
        <ChannelMutateDrawer
          open={open}
          onOpenChange={() => undefined}
          currentRow={currentRow}
        />
      </ChannelsProvider>
    </QueryClientProvider>
  )
}

function renderChannel() {
  return render(channelEditor(channel))
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
    if (url === `/api/channel/${channel.id}`) {
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
  localStorage.clear()
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

test.each([
  ['append', 'Append to existing keys'],
  ['replace', 'Replace all existing keys'],
] as const)(
  '编辑单 Key 渠道显式转换以 %s 模式提交聚合请求',
  async (mode, label) => {
    channel.channel_info.is_multi_key = false
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    renderChannel()
    await screen.findByDisplayValue('Existing channel')
    await user.click(
      screen.getByRole('checkbox', { name: 'Convert to multi-key channel' })
    )
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
      convert_to_multi_key: true,
      key_mode: mode,
      multi_key_mode: 'polling',
    })
  }
)

test('取消转换后普通更新省略转换和模式字段', async () => {
  channel.channel_info.is_multi_key = false
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const user = userEvent.setup()
  renderChannel()
  await screen.findByDisplayValue('Existing channel')
  const conversion = screen.getByRole('checkbox', {
    name: 'Convert to multi-key channel',
  })
  await user.click(conversion)
  await user.click(screen.getByRole('combobox', { name: 'Key Update Mode' }))
  await user.click(
    screen.getByRole('option', { name: 'Replace all existing keys' })
  )
  await user.click(conversion)
  expect(
    screen.queryByRole('combobox', { name: 'Multi-Key Strategy' })
  ).not.toBeInTheDocument()
  await user.click(screen.getByRole('textbox', { name: 'API Key *' }))
  await user.paste('updated-single-key')
  await user.click(screen.getByRole('button', { name: 'Update Channel' }))
  await waitFor(() => expect(put).toHaveBeenCalled())
  const payload = put.mock.calls[0]?.[1]
  expect(payload).toMatchObject({ key: 'updated-single-key' })
  expect(payload).not.toHaveProperty('convert_to_multi_key')
  expect(payload).not.toHaveProperty('key_mode')
  expect(payload).not.toHaveProperty('multi_key_mode')
})

test.each(['reopen', 'switch'] as const)(
  '%s 渠道编辑时清空转换和策略选择',
  async (action) => {
    channel.channel_info.is_multi_key = false
    const user = userEvent.setup()
    const view = renderChannel()
    await screen.findByDisplayValue('Existing channel')
    await user.click(
      screen.getByRole('checkbox', { name: 'Convert to multi-key channel' })
    )
    await user.click(
      screen.getByRole('combobox', { name: 'Multi-Key Strategy' })
    )
    await user.click(screen.getByRole('option', { name: 'Polling' }))
    if (action === 'reopen') {
      view.rerender(channelEditor(channel, false))
      view.rerender(channelEditor(channel))
    } else {
      channel = { ...channel, id: 43, name: 'Another channel' }
      view.rerender(channelEditor(channel))
    }
    await screen.findByDisplayValue(channel.name)
    expect(
      screen.getByRole('checkbox', { name: 'Convert to multi-key channel' })
    ).not.toBeChecked()
    expect(
      screen.queryByRole('combobox', { name: 'Multi-Key Strategy' })
    ).not.toBeInTheDocument()
    await user.click(
      screen.getByRole('checkbox', { name: 'Convert to multi-key channel' })
    )
    expect(
      screen.getByRole('combobox', { name: 'Multi-Key Strategy' })
    ).toHaveTextContent('Random')
  }
)

test.each([
  ['Vertex AI', { type: 41, other: 'global' }],
  ['Codex', { type: 57 }],
  ['io.net', { other_info: '{"source":"ionet"}' }],
])('编辑 %s 单 Key 渠道时不提供转换入口', async (_label, overrides) => {
  channel = { ...channel, ...overrides }
  channel.channel_info.is_multi_key = false
  renderChannel()
  await screen.findByDisplayValue('Existing channel')
  expect(
    screen.queryByRole('checkbox', { name: 'Convert to multi-key channel' })
  ).not.toBeInTheDocument()
})

test.each([
  [true, false],
  [false, true],
])(
  'write=%s sensitive_write=%s 时不提供转换入口',
  async (write, sensitiveWrite) => {
    channel.channel_info.is_multi_key = false
    useAuthStore.setState({
      auth: {
        ...originalAuth,
        user: {
          id: 2,
          username: 'operator',
          role: ROLE.ADMIN,
          permissions: {
            admin_permissions: {
              channel: { read: true, write, sensitive_write: sensitiveWrite },
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
      screen.queryByRole('checkbox', { name: 'Convert to multi-key channel' })
    ).not.toBeInTheDocument()
    await userEvent.click(
      screen.getByRole('button', { name: 'Update Channel' })
    )
    if (write) {
      await waitFor(() => expect(put).toHaveBeenCalled())
      expect(put.mock.calls[0]?.[1]).not.toHaveProperty('convert_to_multi_key')
    } else {
      expect(put).not.toHaveBeenCalled()
    }
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

test('转换中目标类型切换到 Codex 后清除转换，切回普通类型也不恢复', async () => {
  channel.channel_info.is_multi_key = false
  const user = userEvent.setup()
  renderChannel()
  await screen.findByDisplayValue('Existing channel')
  await user.click(
    screen.getByRole('checkbox', { name: 'Convert to multi-key channel' })
  )
  const channelType = screen.getByPlaceholderText('Search channel type...')
  await user.click(channelType)
  await user.clear(channelType)
  await user.type(channelType, 'Codex')
  await user.click(
    screen.getByRole('option', { name: 'ChatGPT Subscription (Codex)' })
  )
  expect(
    screen.queryByRole('checkbox', { name: 'Convert to multi-key channel' })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('combobox', { name: 'Multi-Key Strategy' })
  ).not.toBeInTheDocument()
  await user.click(channelType)
  await user.clear(channelType)
  await user.type(channelType, 'OpenAI')
  await user.click(screen.getByRole('option', { name: 'OpenAI' }))
  expect(
    screen.getByRole('checkbox', { name: 'Convert to multi-key channel' })
  ).not.toBeChecked()
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
