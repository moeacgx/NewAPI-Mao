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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { PromptTemplateEditor } from '../components/prompt-template-editor'
import { promptTemplateSchema } from '../lib/validation'

vi.mock('@lobehub/icons', () => ({}))

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))

const input = {
  title: 'Review',
  description: 'Review code',
  content: 'Check correctness',
  enabled: true,
  sort_order: 0,
  version: 4,
  group_ids: [12],
}

beforeEach(() => {
  vi.mocked(api.get)
    .mockReset()
    .mockImplementation(async (url) => ({
      data: {
        success: true,
        data:
          url === '/api/group/details'
            ? [{ id: 12, code: 'premium-code', name: 'Premium models' }]
            : [],
      },
    }))
  vi.mocked(api.post)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
  vi.mocked(api.put)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
})

function renderEditor(existing = false) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <PromptTemplateEditor
        onClose={vi.fn()}
        template={
          existing
            ? {
                ...input,
                id: 7,
                group_details: [{ id: 12, name: 'Premium models' }],
              }
            : undefined
        }
      />
    </QueryClientProvider>
  )
}

describe('提示词管理员编辑', () => {
  it('删除旧白名单分组后可取消失效项并下架', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { success: true, data: [] } })
    renderEditor(true)
    fireEvent.click(
      await screen.findByRole('button', { name: 'Remove group Deleted (12)' })
    )
    fireEvent.click(screen.getByRole('checkbox', { name: 'Published' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/prompt-center/admin/7', {
        ...input,
        kind: 'prompt',
        load_mode: 'eager',
        category_id: 0,
        skills: [],
        enabled: false,
        all_groups: false,
        group_ids: [],
      })
    )
  })
  it('展示当前分组名称，提交稳定分组 ID 和已有版本', async () => {
    renderEditor(true)
    expect(
      await screen.findByRole('checkbox', { name: 'Premium models' })
    ).toBeChecked()
    expect(screen.queryByText('premium-code')).not.toBeInTheDocument()
    fireEvent.change(screen.getByRole('textbox', { name: 'Title' }), {
      target: { value: 'Updated review' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/prompt-center/admin/7', {
        ...input,
        kind: 'prompt',
        load_mode: 'eager',
        category_id: 0,
        skills: [],
        all_groups: false,
        title: 'Updated review',
      })
    )
  })

  it('一次编辑可保存多个模型分组，提交稳定 ID', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => ({
      data: {
        success: true,
        data:
          url === '/api/group/details'
            ? [
                { id: 12, code: 'premium-code', name: 'Premium models' },
                { id: 21, code: 'review-code', name: 'Review models' },
              ]
            : [],
      },
    }))
    renderEditor(true)
    expect(
      await screen.findByRole('checkbox', { name: 'Premium models' })
    ).toBeChecked()
    fireEvent.click(screen.getByRole('checkbox', { name: 'Review models' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7',
        expect.objectContaining({ group_ids: [12, 21], version: 4 })
      )
    )
  })

  it('标签加载失败时允许全量选组，并可重试恢复标签选择', async () => {
    let tagsAvailable = false
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === '/api/group/tags') {
        if (!tagsAvailable) throw new Error('Unavailable')
        return {
          data: {
            success: true,
            data: [{ id: 1, name: 'GPT', icons: [], group_ids: [12] }],
          },
        }
      }
      return {
        data: {
          success: true,
          data:
            url === '/api/group/details'
              ? [{ id: 12, code: 'premium-code', name: 'Premium models' }]
              : [],
        },
      }
    })
    renderEditor(true)
    expect(
      await screen.findByText('Failed to load group tags')
    ).toBeInTheDocument()
    expect(
      screen.getByRole('checkbox', { name: 'Premium models' })
    ).toBeChecked()
    tagsAvailable = true
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(
      await screen.findByRole('button', { name: /GPT/ })
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Failed to load group tags')
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Remove group Premium models' })
    ).toBeInTheDocument()
  })

  it('未选择白名单分组时阻止创建', async () => {
    renderEditor()
    await screen.findByRole('checkbox', { name: 'Premium models' })
    fireEvent.change(screen.getByRole('textbox', { name: 'Title' }), {
      target: { value: 'Review' },
    })
    fireEvent.change(screen.getByRole('textbox', { name: 'Prompt content' }), {
      target: { value: 'Check code' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(
      await screen.findByText('Select at least one allowed group')
    ).toBeInTheDocument()
    expect(api.post).not.toHaveBeenCalled()
  })

  it('保存失败后保留编辑内容及原始版本供处理冲突', async () => {
    vi.mocked(api.put).mockResolvedValueOnce({
      data: { success: false, message: 'Version conflict' },
    })
    renderEditor(true)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    fireEvent.change(screen.getByRole('textbox', { name: 'Title' }), {
      target: { value: 'My draft' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Version conflict'
    )
    expect(screen.getByRole('textbox', { name: 'Title' })).toHaveValue(
      'My draft'
    )
  })
})

describe('提示词大小和分组校验', () => {
  it('允许负排序并允许无白名单的下架草稿', () => {
    expect(
      promptTemplateSchema.safeParse({ ...input, sort_order: -1 }).success
    ).toBe(true)
    expect(
      promptTemplateSchema.safeParse({
        ...input,
        enabled: false,
        group_ids: [],
      }).success
    ).toBe(true)
  })
  it('按 UTF-8 字节限制中文正文', () => {
    expect(
      promptTemplateSchema.safeParse({ ...input, content: '中'.repeat(10922) })
        .success
    ).toBe(true)
    expect(
      promptTemplateSchema.safeParse({ ...input, content: '中'.repeat(10923) })
        .success
    ).toBe(false)
  })

  it('拒绝空白正文和无实体分组', () => {
    expect(
      promptTemplateSchema.safeParse({ ...input, content: '   ' }).success
    ).toBe(false)
    expect(
      promptTemplateSchema.safeParse({ ...input, group_ids: [] }).success
    ).toBe(false)
    expect(
      promptTemplateSchema.safeParse({ ...input, group_ids: [0] }).success
    ).toBe(false)
  })
})
