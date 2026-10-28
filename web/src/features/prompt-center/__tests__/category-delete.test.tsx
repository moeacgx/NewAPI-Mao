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
  within,
} from '@testing-library/react'
import { AxiosError } from 'axios'
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { CategoryManager } from '../components/category-manager'
import { PromptCenter } from '../index'
import type { PromptCategory, PromptTemplate } from '../types'

vi.mock('@lobehub/icons', () => ({}))

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

const communication: PromptCategory = {
  id: 3,
  name: 'Communication',
  version: 4,
}
const writing: PromptCategory = { id: 5, name: 'Writing', version: 1 }

const categorized: PromptTemplate = {
  id: 7,
  title: 'Reply drafts',
  description: 'Draft replies',
  enabled: true,
  sort_order: 0,
  version: 1,
  group_ids: [],
  group_details: [],
  category_id: 3,
  category_name: 'Communication',
}
const uncategorized: PromptTemplate = {
  ...categorized,
  id: 8,
  title: 'Release review',
  category_id: 0,
  category_name: '',
}

const deleteLabel = 'Delete category Communication'
const conflictMessage = '技能分类已被其他管理员更新，请刷新后重试'

let serverCategories: PromptCategory[] = []
let serverTemplates: PromptTemplate[] = []

beforeEach(() => {
  serverCategories = [communication, writing]
  serverTemplates = [categorized, uncategorized]
  vi.mocked(api.get)
    .mockReset()
    .mockImplementation(async (url) => {
      if (String(url).includes('categories')) {
        return { data: { success: true, data: serverCategories } }
      }
      return { data: { success: true, data: serverTemplates } }
    })
  vi.mocked(api.post).mockReset()
  vi.mocked(api.put).mockReset()
  vi.mocked(api.delete)
    .mockReset()
    .mockImplementation(async () => {
      serverCategories = [writing]
      serverTemplates = serverTemplates.map((template) =>
        template.category_id === 3
          ? { ...template, category_id: 0, category_name: '' }
          : template
      )
      return { data: { success: true, data: { id: 3, moved_templates: 2 } } }
    })
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
})

afterEach(() => {
  useAuthStore.getState().auth.reset()
  toast.dismiss()
})

function renderWithQuery(children: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      {children}
      <Toaster />
    </QueryClientProvider>
  )
  return client
}

async function openDeleteConfirmation() {
  fireEvent.click(await screen.findByRole('button', { name: deleteLabel }))
  return screen.findByRole('alertdialog')
}

describe('删除技能分类', () => {
  it('取消确认时不发送删除请求并保留分类', async () => {
    renderWithQuery(<CategoryManager onClose={vi.fn()} />)
    const dialog = await openDeleteConfirmation()
    expect(dialog).toHaveTextContent(
      'Deleting "Communication" moves its skills to Uncategorized. This action cannot be undone.'
    )
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    expect(api.delete).not.toHaveBeenCalled()
    expect(screen.getByText('Communication')).toBeInTheDocument()
  })

  it('确认后按分类版本删除，提示移动数量并刷新分类与技能缓存', async () => {
    const client = renderWithQuery(<CategoryManager onClose={vi.fn()} />)
    const relatedKeys = [
      ['prompt-center', true],
      ['token-prompts', 9],
    ]
    for (const key of relatedKeys) client.setQueryData(key, [])
    const dialog = await openDeleteConfirmation()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() =>
      expect(api.delete).toHaveBeenCalledWith(
        '/api/prompt-center/admin/categories/3',
        expect.objectContaining({ params: { version: 4 } })
      )
    )
    expect(
      await screen.findByText(
        'Category deleted. 2 skill(s) moved to Uncategorized.'
      )
    ).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByText('Communication')).not.toBeInTheDocument()
    )
    expect(screen.getByText('Writing')).toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    for (const key of relatedKeys) {
      expect(client.getQueryState(key)?.isInvalidated).toBe(true)
    }
  })

  it('删除正在重命名的分类后重置为新建表单', async () => {
    serverCategories = [communication]
    renderWithQuery(<CategoryManager onClose={vi.fn()} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Rename' }))
    expect(
      screen.getByRole('textbox', { name: 'Rename skill category' })
    ).toHaveValue('Communication')
    const dialog = await openDeleteConfirmation()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    expect(
      await screen.findByRole('textbox', { name: 'New skill category' })
    ).toHaveValue('')
  })

  it.each([
    [
      'HTTP 409 版本冲突',
      () =>
        vi.mocked(api.delete).mockRejectedValue(
          new AxiosError('Conflict', undefined, undefined, undefined, {
            status: 409,
            data: { success: false, message: conflictMessage },
          } as never)
        ),
      conflictMessage,
    ],
    [
      '分类已不存在',
      () =>
        vi.mocked(api.delete).mockResolvedValue({
          data: { success: false, message: 'Skill category not found' },
        }),
      'Skill category not found',
    ],
  ])('%s 时展示服务端错误并保留分类', async (_, mockFailure, message) => {
    mockFailure()
    renderWithQuery(<CategoryManager onClose={vi.fn()} />)
    const dialog = await openDeleteConfirmation()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(message)
    expect(within(dialog).getByRole('button', { name: 'Delete' })).toBeEnabled()
    expect(screen.getByText('Communication')).toBeInTheDocument()
  })
})

describe('技能目录分类筛选', () => {
  it('当前筛选分类被删除后回到全部分类并展示移出的技能', async () => {
    renderWithQuery(<PromptCenter />)
    await screen.findByText('Reply drafts')
    const filter = screen.getByRole('combobox', { name: 'Skill category' })
    await within(filter).findByRole('option', { name: 'Communication' })
    fireEvent.change(filter, { target: { value: '3' } })
    await waitFor(() =>
      expect(screen.queryByText('Release review')).not.toBeInTheDocument()
    )
    fireEvent.click(
      screen.getByRole('button', { name: 'Manage skill categories' })
    )
    const dialog = await openDeleteConfirmation()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await screen.findByText(
      'Category deleted. 2 skill(s) moved to Uncategorized.'
    )
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() =>
      expect(
        screen.getByRole('combobox', { name: 'Skill category' })
      ).toHaveValue('all')
    )
    expect(screen.getByText('Reply drafts')).toBeInTheDocument()
    expect(screen.getByText('Release review')).toBeInTheDocument()
  })
})
