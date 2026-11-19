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
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { channelSchema, type Channel } from '../../types'
import { ChannelsProvider } from '../channels-provider'
import { ChannelMutateDrawer } from '../drawers/channel-mutate-drawer'

vi.mock('@/lib/lobe-icon', () => ({ getLobeIcon: () => null }))

const originalAuth = useAuthStore.getState().auth
let client: QueryClient
let channel: Channel

// Display names differ from codes so the test proves names are only shown.
const groupDetails = [
  { id: 11, code: 'code_std', name: '当前标准组' },
  { id: 12, code: 'code_pro', name: '当前专业组' },
  { id: 13, code: 'code_cn', name: '国产直连组' },
]
const groupTags = [
  {
    id: 1,
    name: '海外平台',
    description: '',
    sort_order: 0,
    icons: [],
    match_keywords: [],
    group_ids: [11, 12],
  },
  {
    id: 2,
    name: '国产模型',
    description: '',
    sort_order: 1,
    icons: [],
    match_keywords: [],
    group_ids: [13],
  },
]

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
    group: 'code_pro',
    group_ids: [12],
    group_details: [{ id: 12, code: 'code_pro', name: '当前专业组' }],
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
    const responses: Record<string, unknown> = {
      '/api/channel/42': channel,
      '/api/channel/models': [{ id: 'gpt-4' }],
      '/api/group/': groupDetails.map((group) => group.code),
      '/api/group/details': groupDetails,
      '/api/group/tags': groupTags,
      '/api/prefill_group': [],
      '/api/vendors/': { items: [], total: 0 },
      '/api/user/2fa/status': { enabled: false },
      '/api/user/passkey': { enabled: false },
    }
    if (!(url in responses)) throw new Error(`Unexpected GET ${url}`)
    return { data: { success: true, data: responses[url] } }
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.setState({ auth: originalAuth })
})

test('编辑渠道经两个标签卡片累加选组后提交稳定 code 与 ID', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const user = userEvent.setup()
  render(
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

  const chips = await screen.findByRole('group', { name: 'Selected groups' })
  expect(within(chips).getByText('当前专业组')).toBeInTheDocument()

  await user.click(screen.getByRole('button', { name: /海外平台/ }))
  await user.click(await screen.findByRole('option', { name: '当前标准组' }))
  await user.keyboard('{Escape}')
  await user.click(screen.getByRole('button', { name: /国产模型/ }))
  await user.click(await screen.findByRole('option', { name: '国产直连组' }))
  await user.keyboard('{Escape}')

  await user.click(screen.getByRole('button', { name: 'Update Channel' }))
  await waitFor(() => expect(put).toHaveBeenCalled())
  expect(put.mock.calls[0]?.[1]).toMatchObject({
    group: 'code_pro,code_std,code_cn',
    group_ids: [12, 11, 13],
  })
})
