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
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { apiKeySchema } from '../../types'
import { ApiKeysMutateDrawer } from '../api-keys-mutate-drawer'
import { ApiKeysProvider } from '../api-keys-provider'

const tags = [
  { id: 1, name: '标准平台', icons: [], group_ids: [11, 999] },
  { id: 2, name: '专业平台', icons: [], group_ids: [12] },
  { id: 3, name: '无权平台', icons: [], group_ids: [999] },
]
const key = apiKeySchema.parse({
  id: 71,
  name: 'existing',
  key: '',
  status: 1,
  remain_quota: 1000,
  used_quota: 0,
  unlimited_quota: true,
  expired_time: -1,
  created_time: 0,
  accessed_time: 0,
  group: 'historical_alias',
  group_mode: 'explicit',
  group_details: [{ id: 11, code: 'stable_a', name: '标准当前名称' }],
  model_limits_enabled: false,
})
let client: QueryClient
beforeEach(() => {
  localStorage.clear()
  useAuthStore.getState().auth.setUser({ id: 2, username: 'normal', role: 1 })
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
})
afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.getState().auth.reset()
  localStorage.clear()
})

describe('user token group tag request contracts', () => {
  test.each([false, true])(
    'tag selection preserves stable group IDs/codes in create/edit=%s',
    async (editing) => {
      const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
        if (url === '/api/status') {
          return { data: { data: { default_use_auto_group: false } } }
        }
        if (url === '/api/user/models') {
          return { data: { success: true, data: [] } }
        }
        if (url === '/api/user/self/groups') {
          return {
            data: {
              success: true,
              data: {
                stable_a: {
                  id: 11,
                  name: '标准当前名称',
                  code: 'stable_a',
                  ratio: 1,
                  desc: '',
                },
                stable_b: {
                  id: 12,
                  name: '专业当前名称',
                  code: 'stable_b',
                  ratio: 2,
                  desc: '',
                },
              },
              group_tags: tags,
            },
          }
        }
        if (url === '/api/token/auto-groups') {
          return { data: { success: true, data: { groups: [], max_count: 0 } } }
        }
        if (url === '/api/token/71') {
          return { data: { success: true, data: key } }
        }
        throw new Error(`Unexpected GET ${url}`)
      })
      const post = vi
        .spyOn(api, 'post')
        .mockResolvedValue({ data: { success: true } })
      const put = vi
        .spyOn(api, 'put')
        .mockResolvedValue({ data: { success: true } })
      render(
        <QueryClientProvider client={client}>
          <ApiKeysProvider>
            <ApiKeysMutateDrawer
              open
              onOpenChange={() => undefined}
              currentRow={editing ? key : undefined}
            />
          </ApiKeysProvider>
        </QueryClientProvider>
      )
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: 'Save changes' })
        ).toBeEnabled()
      )
      expect(
        screen.queryByRole('combobox', { name: 'Group' })
      ).not.toBeInTheDocument()
      if (editing) {
        expect(
          screen.getByRole('button', { name: 'Remove 标准当前名称' })
        ).toBeInTheDocument()
      }
      if (!editing) {
        const standardCard = await screen.findByRole('button', {
          name: /标准平台/,
        })
        fireEvent.click(standardCard)
        expect(standardCard).toHaveAttribute('aria-pressed', 'true')
        fireEvent.click(
          within(await screen.findByRole('listbox')).getByRole('option', {
            name: /标准当前名称/,
          })
        )
        await waitFor(() =>
          expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
        )
      }
      const professionalCard = await screen.findByRole('button', {
        name: /专业平台/,
      })
      fireEvent.click(professionalCard)
      expect(professionalCard).toHaveAttribute('aria-pressed', 'true')
      expect(
        screen.queryByRole('button', { name: /无权平台/ })
      ).not.toBeInTheDocument()
      const list = await screen.findByRole('listbox')
      expect(within(list).queryByText('标准当前名称')).not.toBeInTheDocument()
      fireEvent.click(
        within(list).getByRole('option', { name: /专业当前名称/ })
      )
      await waitFor(() =>
        expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
      )
      expect(
        screen.queryByRole('combobox', { name: 'Group' })
      ).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: 'Add group' })
      ).not.toBeInTheDocument()
      const order = screen.getByRole('list', { name: 'Group priority order' })
      expect(
        within(order)
          .getAllByRole('listitem')
          .map((item) => item.textContent)
          .join('|')
      ).toMatch(/标准当前名称.*\|.*专业当前名称/)
      if (!editing) {
        fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
          target: { value: 'new-tag-token' },
        })
      }
      fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
      const request = editing ? put : post
      await waitFor(() =>
        expect(request).toHaveBeenCalledWith(
          '/api/token/',
          expect.objectContaining({
            group: 'stable_a,stable_b',
            group_ids: [11, 12],
            group_mode: 'explicit',
          })
        )
      )
      expect(request.mock.calls[0]?.[1]).not.toHaveProperty('group_tags')
      expect(request.mock.calls[0]?.[1]).not.toHaveProperty('group_channel_ids')
      expect(get.mock.calls.every(([url]) => url !== '/api/group/tags')).toBe(
        true
      )
      if (editing) expect(request.mock.calls[0]?.[1]).toHaveProperty('id', 71)
    }
  )

  test('single selected group is a removable chip without a second picker trigger', async () => {
    vi.spyOn(api, 'get').mockImplementation(async (url) => {
      if (url === '/api/status') {
        return { data: { data: { default_use_auto_group: false } } }
      }
      if (url === '/api/user/models') {
        return { data: { success: true, data: [] } }
      }
      if (url === '/api/user/self/groups') {
        return {
          data: {
            success: true,
            data: {
              stable_a: {
                id: 11,
                name: '标准当前名称',
                code: 'stable_a',
                ratio: 1,
                desc: '',
              },
            },
            group_tags: tags,
          },
        }
      }
      if (url === '/api/token/auto-groups') {
        return { data: { success: true, data: { groups: [], max_count: 0 } } }
      }
      throw new Error(`Unexpected GET ${url}`)
    })
    render(
      <QueryClientProvider client={client}>
        <ApiKeysProvider>
          <ApiKeysMutateDrawer open onOpenChange={() => undefined} />
        </ApiKeysProvider>
      </QueryClientProvider>
    )
    const card = await screen.findByRole('button', { name: /标准平台/ })
    expect(screen.queryByRole('combobox', { name: 'Group' })).toBeNull()
    fireEvent.click(card)
    fireEvent.click(
      within(await screen.findByRole('listbox')).getByRole('option', {
        name: /标准当前名称/,
      })
    )
    const remove = await screen.findByRole('button', {
      name: 'Remove 标准当前名称',
    })
    fireEvent.click(remove)
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: 'Remove 标准当前名称' })
      ).not.toBeInTheDocument()
    )
    expect(screen.queryByRole('combobox', { name: 'Group' })).toBeNull()
  })
})
