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
  fireEvent,
  render,
  screen,
  waitFor,
  cleanup,
} from '@testing-library/react'
import { useState } from 'react'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { apiKeySchema } from '../../types'
import { ApiKeysMutateDrawer } from '../api-keys-mutate-drawer'
import { ApiKeysProvider } from '../api-keys-provider'

vi.mock('@lobehub/icons', () => ({}))
afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  useAuthStore.getState().auth.reset()
  localStorage.clear()
})

function setup(
  role = 1,
  raw = '{"stable_a":1.25,"stable_b":2}',
  create = false,
  mode: 'explicit' | 'auto' | 'inherit' = 'explicit'
) {
  useAuthStore.getState().auth.setUser({ id: 41, username: 'owner', role })
  let stored = {
    ...apiKeySchema.parse({
      id: 7,
      name: 'protected',
      key: 'masked',
      status: 1,
      remain_quota: 0,
      used_quota: 0,
      unlimited_quota: true,
      expired_time: -1,
      created_time: 1,
      accessed_time: 1,
      model_limits_enabled: false,
      group: { explicit: 'stable_a,stable_b', auto: 'auto', inherit: '' }[mode],
      group_ids: mode === 'explicit' ? [11, 22] : [],
      group_mode: mode,
      group_details: [
        { id: 11, code: 'stable_a', name: 'Standard Display' },
        { id: 22, code: 'stable_b', name: 'Premium Display' },
      ],
    }),
    group_ratio_limits: raw,
  }
  const payloads: Record<string, unknown>[] = []
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/status') {
      return { data: { data: { default_use_auto_group: false } } }
    }
    if (url === '/api/user/models') return { data: { success: true, data: [] } }
    if (url === '/api/user/self/groups') {
      return {
        data: {
          success: true,
          data: {
            default: {
              id: 1,
              code: 'default',
              name: 'Default Display',
              ratio: 1,
            },
            stable_a: {
              id: 11,
              code: 'stable_a',
              name: 'Standard Display',
              ratio: 1,
            },
            stable_b: {
              id: 22,
              code: 'stable_b',
              name: 'Premium Display',
              ratio: 2,
            },
            auto: { name: 'Auto', ratio: 'auto' },
          },
        },
      }
    }
    if (url === '/api/token/auto-groups') {
      return {
        data: {
          success: true,
          data: { groups: ['stable_a', 'stable_b'], max_count: 2 },
        },
      }
    }
    if (url === '/api/token/7') return { data: { success: true, data: stored } }
    throw new Error(`Unexpected GET ${url}`)
  })
  const save = async (_url: string, data: unknown) => {
    payloads.push(data as Record<string, unknown>)
    stored = { ...stored, ...(data as Partial<typeof stored>) }
    return { data: { success: true } }
  }
  vi.spyOn(api, 'put').mockImplementation(save)
  vi.spyOn(api, 'post').mockImplementation(save)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  function Harness() {
    const [open, setOpen] = useState(true)
    return (
      <>
        <button type='button' onClick={() => setOpen(true)}>
          Reopen
        </button>
        <ApiKeysProvider>
          <ApiKeysMutateDrawer
            open={open}
            onOpenChange={setOpen}
            currentRow={create ? undefined : stored}
          />
        </ApiKeysProvider>
      </>
    )
  }
  render(
    <QueryClientProvider client={client}>
      <Harness />
    </QueryClientProvider>
  )
  return { payloads }
}

test.each([1, 10])(
  '角色 %s 修改名称保留逐组保护，保存重开回显稳定 code 对应当前名称',
  async (role) => {
    const { payloads } = setup(role)
    const first = await screen.findByLabelText(
      'Ratio limit for Standard Display'
    )
    expect(first).toHaveValue('1.25')
    expect(
      screen.getByLabelText('Ratio limit for Premium Display')
    ).toHaveValue('2')
    fireEvent.change(screen.getByLabelText('Name'), {
      target: { value: 'renamed' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(payloads).toHaveLength(1))
    expect(JSON.parse(String(payloads[0].group_ratio_limits))).toEqual({
      stable_a: 1.25,
      stable_b: 2,
    })
    expect(payloads[0].group_ids).toEqual([11, 22])
    expect(payloads[0].group).toBe('stable_a,stable_b')
    if (role === 1) expect(payloads[0]).not.toHaveProperty('group_channel_ids')
    fireEvent.click(screen.getByText('Reopen'))
    expect(
      await screen.findByLabelText('Ratio limit for Standard Display')
    ).toHaveValue('1.25')
  }
)

test('负数阻止保存，0及空白清除；取消重开不遗留草稿', async () => {
  const { payloads } = setup()
  const input = await screen.findByLabelText('Ratio limit for Standard Display')
  fireEvent.change(input, { target: { value: '-1' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  expect(
    await screen.findByText('Enter a finite ratio of zero or greater.')
  ).toBeVisible()
  expect(payloads).toHaveLength(0)
  fireEvent.click(screen.getAllByRole('button', { name: 'Close' })[0])
  fireEvent.click(screen.getByText('Reopen'))
  const restored = await screen.findByLabelText(
    'Ratio limit for Standard Display'
  )
  await waitFor(() => expect(restored).toHaveValue('1.25'))
  fireEvent.change(restored, { target: { value: '0' } })
  fireEvent.change(screen.getByLabelText('Ratio limit for Premium Display'), {
    target: { value: '' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(payloads).toHaveLength(1))
  expect(payloads[0].group_ratio_limits).toBe('')
})

test('非法存储 JSON 不被默默清空，明确清除后才可保存', async () => {
  const { payloads } = setup(1, '{broken')
  expect(
    await screen.findByText(
      'Stored ratio protection is invalid. Clear it explicitly before saving.'
    )
  ).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  expect(payloads).toHaveLength(0)
  fireEvent.click(
    screen.getByRole('button', { name: 'Clear ratio protection' })
  )
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(payloads).toHaveLength(1))
  expect(payloads[0].group_ratio_limits).toBe('')
})

test.each(['auto', 'inherit'] as const)(
  '已有 %s 模式修改名称保留保护和模式',
  async (mode) => {
    const { payloads } = setup(1, '{"stable_a":2}', false, mode)
    expect(
      await screen.findByLabelText('Ratio limit for Standard Display')
    ).toHaveValue('2')
    fireEvent.change(screen.getByLabelText('Name'), {
      target: { value: 'renamed' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(payloads).toHaveLength(1))
    expect(payloads[0].group_mode).toBe(mode)
    expect(payloads[0].group_ids).toEqual([])
    expect(JSON.parse(String(payloads[0]['group' + '_ratio_limits']))).toEqual({
      stable_a: 2,
    })
  }
)

test('取消已选组只清除该组保护，重新选择不复活旧保护', async () => {
  const { payloads } = setup()
  await screen.findByLabelText('Ratio limit for Standard Display')
  fireEvent.click(screen.getByRole('combobox', { name: 'Group' }))
  fireEvent.click(
    await screen.findByRole('option', { name: /Standard Display/ })
  )
  expect(screen.queryByLabelText('Ratio limit for Standard Display')).toBeNull()
  expect(screen.getByLabelText('Ratio limit for Premium Display')).toHaveValue(
    '2'
  )
  fireEvent.click(screen.getByRole('combobox', { name: 'Group' }))
  fireEvent.click(
    await screen.findByRole('option', { name: /Standard Display/ })
  )
  expect(screen.getByLabelText('Ratio limit for Standard Display')).toHaveValue(
    ''
  )
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(payloads).toHaveLength(1))
  expect(JSON.parse(String(payloads[0]['group' + '_ratio_limits']))).toEqual({
    stable_b: 2,
  })
})

test('创建令牌提交逐组保护并保留普通用户权限边界', async () => {
  const { payloads } = setup(1, '', true)
  fireEvent.click(await screen.findByRole('combobox', { name: 'Group' }))
  fireEvent.click(
    await screen.findByRole('option', { name: /Default Display/ })
  )
  const input = await screen.findByLabelText('Ratio limit for Default Display')
  fireEvent.change(input, { target: { value: '1.5' } })
  fireEvent.change(screen.getByLabelText('Name'), {
    target: { value: 'created' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(payloads).toHaveLength(1))
  expect(JSON.parse(String(payloads[0]['group' + '_ratio_limits']))).toEqual({
    default: 1.5,
  })
  expect(payloads[0].group_ids).toEqual([1])
  expect(payloads[0]).not.toHaveProperty('group_channel_ids')
})
