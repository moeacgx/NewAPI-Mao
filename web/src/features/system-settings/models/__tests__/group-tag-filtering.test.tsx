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
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { GroupRatioVisualEditor } from '../group-ratio-visual-editor'

const groups = [
  {
    id: 11,
    code: 'internal_a',
    name: '标准显示名',
    description: '',
    ratio: '1',
    user_selectable: true,
    exclusive: false,
    status: 1,
    auto_enabled: false,
    auto_order: 0,
    single_user_concurrency_limit: 0,
    _key: 'a',
  },
  {
    id: 12,
    code: 'internal_b',
    name: '专业显示名',
    description: '',
    ratio: '2',
    user_selectable: true,
    exclusive: false,
    status: 1,
    auto_enabled: false,
    auto_order: 0,
    single_user_concurrency_limit: 0,
    _key: 'b',
  },
]
const tags = [
  {
    id: 1,
    name: '标准标签',
    icons: [],
    group_ids: [11],
    match_keywords: [],
    description: '',
    sort_order: 0,
  },
  {
    id: 2,
    name: '空标签',
    icons: [],
    group_ids: [],
    match_keywords: [],
    description: '',
    sort_order: 1,
  },
]
let client: QueryClient
afterEach(() => {
  cleanup()
  client?.clear()
  useAuthStore.getState().auth.reset()
})

function renderTable() {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: tags },
  })
  const onGroupsChange = vi.fn()
  const view = render(
    <QueryClientProvider client={client}>
      <GroupRatioVisualEditor
        groups={groups}
        autoGroup={{ user_selectable: true, description: '' }}
        autoSelectableLocked={false}
        reservedGroupCodes={new Set()}
        isLoadingGroups={false}
        groupLoadError={null}
        topupGroupRatio='{}'
        groupGroupRatio='{}'
        onChange={() => undefined}
        onGroupsChange={onGroupsChange}
        onAutoGroupChange={() => undefined}
        onRetryGroups={() => undefined}
      />
    </QueryClientProvider>
  )
  return { ...view, onGroupsChange }
}

describe('pricing group tag filtering', () => {
  test('filters rows without dropping hidden groups from updates, then clears and shows empty results', async () => {
    const { onGroupsChange } = renderTable()
    fireEvent.click(await screen.findByRole('button', { name: '标准标签' }))
    expect(screen.queryByDisplayValue('专业显示名')).not.toBeInTheDocument()
    fireEvent.change(screen.getByDisplayValue('标准显示名'), {
      target: { value: '改名标准组' },
    })
    expect(onGroupsChange).toHaveBeenLastCalledWith([
      { ...groups[0], name: '改名标准组' },
      groups[1],
    ])
    fireEvent.click(screen.getByRole('button', { name: '空标签' }))
    expect(screen.getByText('No available groups')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Untagged' }))
    expect(screen.getByDisplayValue('专业显示名')).toBeInTheDocument()
    expect(screen.queryByDisplayValue('标准显示名')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'All groups' }))
    expect(screen.getByDisplayValue('标准显示名')).toBeInTheDocument()
    expect(screen.getByDisplayValue('专业显示名')).toBeInTheDocument()
  })

  test('resets to all after the active tag is deleted and does not restore the old selection', async () => {
    renderTable()
    fireEvent.click(await screen.findByRole('button', { name: '标准标签' }))
    act(() => client.setQueryData(['group-tags'], []))
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'All groups' })
      ).toHaveAttribute('aria-pressed', 'true')
    )
    expect(screen.getByDisplayValue('专业显示名')).toBeInTheDocument()
    act(() => client.setQueryData(['group-tags'], tags))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '标准标签' })).toHaveAttribute(
        'aria-pressed',
        'false'
      )
    )
  })
})
