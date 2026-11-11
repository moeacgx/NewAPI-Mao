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

import { GroupTagSettings } from '../group-tag-settings'

vi.mock('@lobehub/icons', () => ({ OpenAI: () => <svg aria-hidden='true' /> }))
const group = { id: 11, code: 'hidden_code', name: '当前分组名称' }
const tag = {
  id: 4,
  name: '现有标签',
  description: '标签说明',
  sort_order: 2,
  icons: ['OpenAI'],
  match_keywords: [],
  group_ids: [11],
}
let client: QueryClient
beforeEach(() => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
})
afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.getState().auth.reset()
})

describe('Default group tag administration', () => {
  test('creates a tag with stable group IDs and refreshes the list', async () => {
    let stored: (typeof tag)[] = []
    vi.spyOn(api, 'get').mockImplementation(async () => ({
      data: { success: true, data: stored },
    }))
    const post = vi
      .spyOn(api, 'post')
      .mockImplementation(async (_url, payload) => {
        stored = [{ ...(payload as typeof tag), id: 5 }]
        return { data: { success: true, data: stored[0] } }
      })
    render(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[group]} />
      </QueryClientProvider>
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'New group tag' })
      ).toBeEnabled()
    )
    fireEvent.click(screen.getByRole('button', { name: 'New group tag' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.change(within(dialog).getByLabelText('Tag name'), {
      target: { value: '平台标签' },
    })
    fireEvent.click(within(dialog).getByLabelText('当前分组名称'))
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith('/api/group/tags', {
        name: '平台标签',
        description: '',
        sort_order: 0,
        icons: [],
        match_keywords: [],
        group_ids: [11],
      })
    )
    expect(await screen.findByText('平台标签')).toBeVisible()
    expect(screen.queryByText('hidden_code')).not.toBeInTheDocument()
  })

  test('does not expose or request management operations to ordinary users', () => {
    useAuthStore.getState().auth.setUser({ id: 2, username: 'user', role: 1 })
    const get = vi.spyOn(api, 'get')
    render(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[group]} />
      </QueryClientProvider>
    )
    expect(
      screen.queryByRole('button', { name: 'New group tag' })
    ).not.toBeInTheDocument()
    expect(get).not.toHaveBeenCalled()
  })

  test('uses refreshed group names and removes deleted bindings from an edited tag', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: [tag] },
    })
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: { success: true, data: { ...tag, group_ids: [] } },
    })
    const view = render(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[group]} />
      </QueryClientProvider>
    )
    expect(await screen.findByText('当前分组名称')).toBeInTheDocument()
    view.rerender(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[{ ...group, name: '改名后的分组' }]} />
      </QueryClientProvider>
    )
    expect(screen.getByText('改名后的分组')).toBeInTheDocument()
    expect(screen.queryByText('当前分组名称')).not.toBeInTheDocument()
    expect(screen.queryByText('hidden_code')).not.toBeInTheDocument()
    view.rerender(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[]} />
      </QueryClientProvider>
    )
    expect(screen.getByText('Deleted (11)')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/group/tags/4',
        expect.objectContaining({ group_ids: [] })
      )
    )
  })

  test('updates existing tag fields by URL identity and keeps the draft on a business failure', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: [tag] },
    })
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValueOnce({
        data: { success: false, message: '保存被拒绝' },
      })
      .mockResolvedValueOnce({
        data: { success: true, data: { ...tag, name: '修改名称' } },
      })
    render(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[group]} />
      </QueryClientProvider>
    )
    fireEvent.click(await screen.findByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.change(within(dialog).getByLabelText('Tag name'), {
      target: { value: '修改名称' },
    })
    fireEvent.change(within(dialog).getByLabelText('Description'), {
      target: { value: '新说明' },
    })
    fireEvent.change(within(dialog).getByLabelText('Sort order'), {
      target: { value: '-5' },
    })
    fireEvent.change(
      within(dialog).getByLabelText('Logos (one per line, up to 6)'),
      { target: { value: 'OpenAI\nhttps://cdn.example.com/icon.png' } }
    )
    fireEvent.change(
      within(dialog).getByLabelText('Match keywords (one per line, up to 32)'),
      { target: { value: '当前\n专业' } }
    )
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      '保存被拒绝'
    )
    expect(within(dialog).getByLabelText('Tag name')).toHaveValue('修改名称')
    expect(put).toHaveBeenCalledWith('/api/group/tags/4', {
      name: '修改名称',
      description: '新说明',
      sort_order: -5,
      icons: ['OpenAI', 'https://cdn.example.com/icon.png'],
      match_keywords: ['当前', '专业'],
      group_ids: [11],
    })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
  })

  test('requires confirmation and keeps the tag when deletion fails, then refreshes on success', async () => {
    let stored = [tag]
    vi.spyOn(api, 'get').mockImplementation(async () => ({
      data: { success: true, data: stored },
    }))
    const remove = vi
      .spyOn(api, 'delete')
      .mockRejectedValueOnce(new Error('删除失败'))
      .mockImplementationOnce(async () => {
        stored = []
        return { data: { success: true } }
      })
    render(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[group]} />
      </QueryClientProvider>
    )
    fireEvent.click(await screen.findByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('alertdialog')
    expect(remove).not.toHaveBeenCalled()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      '删除失败'
    )
    expect(
      screen.getByRole('article', { name: '现有标签', hidden: true })
    ).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    expect(await screen.findByText('No group tags yet.')).toBeInTheDocument()
    expect(remove).toHaveBeenLastCalledWith('/api/group/tags/4')
  })

  test('shows load failures, blocks mutations, and retries the management list', async () => {
    vi.spyOn(api, 'get')
      .mockResolvedValueOnce({
        data: { success: false, message: '无管理权限' },
      })
      .mockResolvedValueOnce({ data: { success: true, data: [tag] } })
    render(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[group]} />
      </QueryClientProvider>
    )
    expect(await screen.findByRole('alert')).toHaveTextContent('无管理权限')
    expect(screen.getByRole('button', { name: 'New group tag' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(
      await screen.findByRole('article', { name: '现有标签' })
    ).toBeInTheDocument()
  })

  test('rejects invalid logo URLs before sending and allows cancelling without mutation', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: [tag] },
    })
    const post = vi.spyOn(api, 'post')
    await render(
      <QueryClientProvider client={client}>
        <GroupTagSettings groups={[group]} />
      </QueryClientProvider>
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'New group tag' })
      ).toBeEnabled()
    )
    fireEvent.click(screen.getByRole('button', { name: 'New group tag' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.change(within(dialog).getByLabelText('Tag name'), {
      target: { value: '非法图片' },
    })
    fireEvent.change(
      within(dialog).getByLabelText('Logos (one per line, up to 6)'),
      { target: { value: 'javascript:alert(1)' } }
    )
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'Check the tag name'
    )
    expect(post).not.toHaveBeenCalled()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
  })
})
