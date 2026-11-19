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
import { act, cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import type { User } from '../../types'
import { UsersMutateDrawer } from '../users-mutate-drawer'
import { UsersProvider } from '../users-provider'

const originalAdapter = api.defaults.adapter
const originalAuth = useAuthStore.getState().auth
let client: QueryClient
let requests: string[]
let submitted: { group?: string } | undefined
let target: User
let detailName: string
let groupsAvailable: boolean

beforeEach(() => {
  requests = []
  submitted = undefined
  detailName = '当前名称'
  groupsAvailable = true
  target = {
    id: 7,
    username: 'fixture-user',
    display_name: '测试用户',
    role: 1,
    status: 1,
    group: 'current-code',
    group_name: '当前名称',
    quota: 0,
    used_quota: 0,
    request_count: 0,
  }
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  useAuthStore.getState().auth.setUser({ id: 1, username: 'root', role: 100 })
  api.defaults.adapter = async (config) => {
    requests.push(config.url || '')
    let data: unknown = {}
    if (config.url === '/api/group/') data = ['current-code', 'other-code']
    if (config.url === '/api/group/details') {
      data = [
        { id: 31, code: 'current-code', name: detailName },
        { id: 32, code: 'other-code', name: '另一分组名称' },
      ]
    }
    if (config.url === '/api/user/7') data = target
    if (config.url === '/api/authz/catalog') data = { resources: [], roles: [] }
    if (config.method === 'put') {
      submitted = JSON.parse(config.data as string) as { group?: string }
    }
    return {
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: {
        success: config.url !== '/api/group/details' || groupsAvailable,
        data,
      },
    }
  }
})

afterEach(() => {
  cleanup()
  client.clear()
  api.defaults.adapter = originalAdapter
  useAuthStore.setState({ auth: originalAuth })
  localStorage.clear()
})

function renderEditor() {
  return render(
    <QueryClientProvider client={client}>
      <UsersProvider>
        <UsersMutateDrawer open onOpenChange={() => {}} currentRow={target} />
      </UsersProvider>
    </QueryClientProvider>
  )
}

test('管理员编辑用户显示当前分组名称，选项选中后提交原code', async () => {
  renderEditor()
  const group = await screen.findByRole('combobox', { name: 'Group' })
  await waitFor(() => expect(group).toHaveTextContent('当前名称'))
  expect(requests).toContain('/api/group/details')
  await userEvent.click(group)
  await userEvent.click(
    await screen.findByRole('option', { name: '另一分组名称' })
  )
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(submitted?.group).toBe('other-code'))
})

test.each([
  ['legacy-code', '改名后名称'],
  ['deleted-code', ''],
])(
  '历史或删除标识%s保持原值并显示当前名称或允许的回退',
  async (group, group_name) => {
    target = { ...target, group, group_name }
    renderEditor()
    const select = await screen.findByRole('combobox', { name: 'Group' })
    await waitFor(() => expect(select).toHaveTextContent(group_name || group))
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(submitted?.group).toBe(group))
  }
)

test('分组改名刷新选项后显示新名称且保留原code', async () => {
  renderEditor()
  const group = await screen.findByRole('combobox', { name: 'Group' })
  await waitFor(() => expect(group).toHaveTextContent('当前名称'))
  await waitFor(() => expect(group).not.toBeDisabled())
  detailName = '更新后的名称'
  await act(async () => {
    await client.invalidateQueries({ queryKey: ['groups'] })
  })
  await waitFor(() => expect(group).toHaveTextContent('更新后的名称'))
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(submitted?.group).toBe('current-code'))
})

test('选项加载失败禁用选择并提示重试，恢复后使用名称', async () => {
  groupsAvailable = false
  renderEditor()
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Failed to load groups'
  )
  const group = screen.getByRole('combobox', { name: 'Group' })
  expect(group).toBeDisabled()
  groupsAvailable = true
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
  await waitFor(() => expect(group).not.toBeDisabled())
  expect(group).toHaveTextContent('当前名称')
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
})
