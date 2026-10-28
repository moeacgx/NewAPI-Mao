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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { PromptTemplateEditor } from '../components/prompt-template-editor'
import { TokenPromptDialog } from '../components/token-prompt-dialog'
import { PromptCenter } from '../index'
import { promptTemplateSchema } from '../lib/validation'
import type { PromptTemplate } from '../types'

vi.mock('@lobehub/icons', () => ({}))

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))

const bundle: PromptTemplate = {
  id: 7,
  title: 'Writing kit',
  description: 'Write and review',
  kind: 'bundle',
  category_id: 3,
  category_name: 'Communication',
  skills: [
    { title: 'Draft', content: 'Private draft instructions' },
    { title: 'Review', content: 'Private review instructions' },
  ],
  skill_count: 2,
  content: '',
  enabled: true,
  sort_order: 0,
  version: 4,
  group_ids: [12],
  group_details: [{ id: 12, name: 'Premium models' }],
}

function renderQuery(children: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
}

beforeEach(() => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'test-user', role: 1 })
  vi.mocked(api.get).mockImplementation(async (url) => {
    if (url === '/api/prompt-center/categories/') {
      return {
        data: {
          success: true,
          data: [{ id: 3, name: 'Communication', version: 1 }],
        },
      }
    }
    if (url === '/api/group/tags') {
      return { data: { success: true, data: [] } }
    }
    if (url === '/api/group/details') {
      return {
        data: {
          success: true,
          data: [{ id: 12, code: 'premium-code', name: 'Premium models' }],
        },
      }
    }
    if (url === '/api/token/3/prompts') {
      return {
        data: {
          success: true,
          data: [{ template: bundle, enabled: true, sort_order: 0 }],
        },
      }
    }
    return {
      data: {
        success: true,
        data: [
          bundle,
          {
            ...bundle,
            id: 8,
            title: 'Single review',
            kind: 'prompt',
            category_id: 0,
            category_name: '',
            skills: [],
          },
        ],
      },
    }
  })
  vi.mocked(api.put)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
})
afterEach(() => useAuthStore.getState().auth.reset())

describe('技能合集浏览和安装摘要', () => {
  it('按分类及类型组合筛选，并可展开合集标题而不泄露正文', async () => {
    renderQuery(<PromptCenter />)
    await screen.findByText('Writing kit')
    expect(screen.getByText('2 skills in this bundle')).toBeInTheDocument()
    fireEvent.click(screen.getByText('2 skills in this bundle'))
    expect(screen.getByText('Draft')).toBeVisible()
    expect(
      screen.queryByText('Private draft instructions')
    ).not.toBeInTheDocument()
    fireEvent.change(screen.getByRole('combobox', { name: 'Skill category' }), {
      target: { value: '3' },
    })
    expect(screen.getByText('Writing kit')).toBeInTheDocument()
    expect(screen.queryByText('Single review')).not.toBeInTheDocument()
    fireEvent.change(screen.getByRole('combobox', { name: 'Skill type' }), {
      target: { value: 'prompt' },
    })
    expect(screen.getByText('No prompts found')).toBeInTheDocument()
    fireEvent.change(screen.getByRole('combobox', { name: 'Skill category' }), {
      target: { value: '0' },
    })
    expect(screen.getByText('Single review')).toBeInTheDocument()
  })

  it('安装摘要展示合集技能列表，保存仍将整个合集作为一项', async () => {
    renderQuery(
      <TokenPromptDialog
        open
        token={{ id: 3, name: 'Development' }}
        onOpenChange={vi.fn()}
      />
    )
    await screen.findByText('Writing kit')
    fireEvent.click(screen.getByText('2 skills in this bundle'))
    expect(screen.getByText('Draft')).toBeVisible()
    expect(
      screen.queryByText('Private draft instructions')
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save installations' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/token/3/prompts', {
        bindings: [{ template_id: 7, enabled: true, sort_order: 0 }],
      })
    )
  })
})

describe('技能合集编辑与校验', () => {
  it('可新增、移除和重排技能，版本冲突保留重排后草稿及分类', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === '/api/group/tags') {
        return { data: { success: true, data: [] } }
      }
      const data =
        url === '/api/prompt-center/categories/'
          ? [{ id: 3, name: 'Communication', version: 1 }]
          : [{ id: 12, code: 'premium-code', name: 'Premium models' }]
      return { data: { success: true, data } }
    })
    vi.mocked(api.put).mockResolvedValueOnce({
      data: { success: false, message: 'Version conflict' },
    })
    renderQuery(<PromptTemplateEditor template={bundle} onClose={vi.fn()} />)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    fireEvent.click(screen.getByRole('button', { name: 'Add skill' }))
    expect(
      screen.getAllByRole('textbox', { name: 'Skill title' })
    ).toHaveLength(3)
    fireEvent.click(screen.getByRole('button', { name: 'Remove skill 3' }))
    fireEvent.click(screen.getByRole('button', { name: 'Move skill 2 up' }))
    expect(
      screen.getAllByRole('textbox', { name: 'Skill title' })[0]
    ).toHaveValue('Review')
    expect(
      screen.getByRole('button', { name: 'Remove skill 1' })
    ).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Version conflict'
    )
    expect(
      screen.getAllByRole('textbox', { name: 'Skill title' })[0]
    ).toHaveValue('Review')
    expect(
      screen.getByRole('combobox', { name: 'Skill category' })
    ).toHaveValue('3')
    expect(api.put).toHaveBeenCalledWith(
      '/api/prompt-center/admin/7',
      expect.objectContaining({
        version: 4,
        category_id: 3,
        kind: 'bundle',
        content: '',
        skills: [bundle.skills?.[1], bundle.skills?.[0]],
      })
    )
  })

  it('合集不能保存重名技能，并显示对应字段错误', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => ({
      data: {
        success: true,
        data:
          url === '/api/group/tags'
            ? []
            : [{ id: 12, name: 'Premium models', code: 'premium-code' }],
      },
    }))
    renderQuery(<PromptTemplateEditor template={bundle} onClose={vi.fn()} />)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    const editor = screen.getByRole('region', { name: 'Bundle skills' })
    fireEvent.change(
      within(editor).getAllByRole('textbox', { name: 'Skill title' })[1],
      { target: { value: 'Draft' } }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(
      await screen.findByText('Use a unique skill title of 1 to 80 characters')
    ).toBeInTheDocument()
    expect(api.put).not.toHaveBeenCalled()
  })

  it('合集必须有 2 至 16 项，拼接正文包含标题后仍受字节上限保护', () => {
    const valid = {
      ...bundle,
      skills:
        bundle.skills?.map((skill) => ({
          ...skill,
          content: skill.content ?? '',
        })) ?? [],
    }
    expect(promptTemplateSchema.safeParse(valid).success).toBe(true)
    expect(
      promptTemplateSchema.safeParse({ ...valid, skills: [valid.skills[0]] })
        .success
    ).toBe(false)
    expect(
      promptTemplateSchema.safeParse({
        ...valid,
        skills: [
          { title: 'A', content: 'a'.repeat(32750) },
          { title: 'B', content: 'b' },
        ],
      }).success
    ).toBe(false)
  })
})
