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
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { PromptTemplateEditor } from '../components/prompt-template-editor'
import { promptTemplateSchema } from '../lib/validation'
import type { PromptTemplate } from '../types'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))

const template = {
  id: 7,
  title: 'Code review',
  description: 'Review correctness and maintainability',
  content: `Read references/rules.md when needed.\n${'中'.repeat(12000)}`,
  load_mode: 'progressive',
  kind: 'prompt',
  skills: [
    {
      id: 'review',
      title: 'Code review',
      description: 'Review correctness and maintainability',
      path: 'SKILL.md',
      content: '',
      resources: [{ path: 'references/rules.md', content: 'Check ownership.' }],
    },
  ],
  enabled: true,
  sort_order: 0,
  version: 2,
  group_ids: [12],
  group_details: [{ id: 12, name: 'Premium models' }],
} as const

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

function renderEditor(value?: PromptTemplate) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return render(
    <QueryClientProvider client={client}>
      <PromptTemplateEditor template={value} onClose={vi.fn()} />
    </QueryClientProvider>
  )
}

describe('渐进式技能的管理契约', () => {
  it('超过 32 KiB 的渐进式技能可校验，旧始终装载模式仍拒绝', () => {
    expect(promptTemplateSchema.safeParse(template).success).toBe(true)
    expect(
      promptTemplateSchema.safeParse({ ...template, load_mode: 'eager' })
        .success
    ).toBe(false)
    expect(
      promptTemplateSchema.safeParse({ ...template, load_mode: undefined })
        .success
    ).toBe(false)
  })

  it('合集独立保留正文而不应用 32 KiB 拼接上限', () => {
    const skills = [
      {
        title: 'Review',
        description: 'Review code',
        content: 'a'.repeat(40000),
      },
      { title: 'Write', description: 'Write code', content: 'b'.repeat(40000) },
    ]
    expect(
      promptTemplateSchema.safeParse({ ...template, kind: 'bundle', skills })
        .success
    ).toBe(true)
    expect(
      promptTemplateSchema.safeParse({
        ...template,
        kind: 'bundle',
        skills: skills.map((skill) => ({ ...skill, description: '' })),
      }).success
    ).toBe(false)
  })

  it('发布缺少描述时拒绝，草稿允许补充用途', () => {
    expect(
      promptTemplateSchema.safeParse({ ...template, description: '' }).success
    ).toBe(false)
    expect(
      promptTemplateSchema.safeParse({
        ...template,
        description: '',
        enabled: false,
      }).success
    ).toBe(true)
  })

  it('渐进式允许同名但独立路径的技能及超过旧 16 项的合集', () => {
    const skills = Array.from({ length: 17 }, (_, index) => ({
      title: 'Review',
      description: 'Review code',
      path: `reviews/${index}/SKILL.md`,
      content: 'Check code',
    }))
    expect(
      promptTemplateSchema.safeParse({ ...template, kind: 'bundle', skills })
        .success
    ).toBe(true)
    expect(
      promptTemplateSchema.safeParse({
        ...template,
        kind: 'bundle',
        load_mode: 'eager',
        skills,
      }).success
    ).toBe(false)
  })

  it.each([
    '../outside.md',
    '/absolute.md',
    'references\\rules.md',
    'references//rules.md',
    'skill.md',
    'a'.repeat(1025),
  ])('拒绝越界或不规范资源路径 %s', (path) => {
    expect(
      promptTemplateSchema.safeParse({
        ...template,
        skills: [
          { ...template.skills[0], resources: [{ path, content: 'text' }] },
        ],
      }).success
    ).toBe(false)
  })

  it('拒绝重复资源路径与超过单文件预算的正文', () => {
    const resource = { path: 'references/rules.md', content: 'text' }
    expect(
      promptTemplateSchema.safeParse({
        ...template,
        skills: [{ ...template.skills[0], resources: [resource, resource] }],
      }).success
    ).toBe(false)
    expect(
      promptTemplateSchema.safeParse({
        ...template,
        content: 'a'.repeat(8 * 1024 * 1024 + 1),
      }).success
    ).toBe(false)
  })

  it('新建默认按需读取且旧模板显示始终装载', async () => {
    const view = renderEditor()
    await screen.findByRole('checkbox', { name: 'Premium models' })
    expect(screen.getByRole('combobox', { name: 'Loading mode' })).toHaveValue(
      'progressive'
    )
    view.unmount()
    renderEditor({
      ...template,
      load_mode: undefined,
      content: 'Always follow this instruction.',
      skills: [],
    } as unknown as PromptTemplate)
    expect(screen.getByRole('combobox', { name: 'Loading mode' })).toHaveValue(
      'eager'
    )
  })

  it('旧单技能切换按需模式后用可见字段保存正文和新增引用', async () => {
    renderEditor({
      ...template,
      load_mode: 'eager',
      content: 'Original prompt',
      skills: [],
    } as unknown as PromptTemplate)
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
    )
    fireEvent.change(screen.getByRole('combobox', { name: 'Loading mode' }), {
      target: { value: 'progressive' },
    })
    fireEvent.change(screen.getByRole('textbox', { name: 'Title' }), {
      target: { value: 'Review updated code' },
    })
    fireEvent.change(screen.getByRole('textbox', { name: 'Description' }), {
      target: { value: 'Review new changes before release' },
    })
    fireEvent.change(screen.getByRole('textbox', { name: 'Prompt content' }), {
      target: { value: 'Read references/review.md before reviewing.' },
    })
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Skill entry path' }),
      { target: { value: 'review/SKILL.md' } }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Add reference file' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Reference path' }), {
      target: { value: 'references/review.md' },
    })
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Reference content' }),
      {
        target: { value: 'Check correctness and permissions.' },
      }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7',
        expect.objectContaining({
          load_mode: 'progressive',
          title: 'Review updated code',
          description: 'Review new changes before release',
          content: 'Read references/review.md before reviewing.',
          skills: [
            {
              title: 'Review updated code',
              description: 'Review new changes before release',
              content: 'Read references/review.md before reviewing.',
              path: 'review/SKILL.md',
              resources: [
                {
                  path: 'references/review.md',
                  content: 'Check correctness and permissions.',
                },
              ],
            },
          ],
        })
      )
    )
  })

  it('修改大技能后保存保留稳定身份及引用资源，失败保留草稿', async () => {
    vi.mocked(api.put).mockResolvedValueOnce({
      data: { success: false, message: 'Version conflict' },
    })
    renderEditor(structuredClone(template) as unknown as PromptTemplate)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Reference content' }),
      { target: { value: 'New rule draft' } }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7',
        expect.objectContaining({
          load_mode: 'progressive',
          content: template.content,
          skills: [
            expect.objectContaining({
              id: 'review',
              path: 'SKILL.md',
              resources: [
                { path: 'references/rules.md', content: 'New rule draft' },
              ],
            }),
          ],
        })
      )
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Version conflict'
    )
    expect(
      screen.getByRole('textbox', { name: 'Reference content' })
    ).toHaveValue('New rule draft')
  })

  it('切到始终装载时提示不保留资源，未保存切回时恢复资源草稿', async () => {
    renderEditor(structuredClone(template) as unknown as PromptTemplate)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Reference content' }),
      { target: { value: 'Unsaved reference' } }
    )
    const mode = screen.getByRole('combobox', { name: 'Loading mode' })
    fireEvent.change(mode, { target: { value: 'eager' } })
    expect(
      screen.getByText(
        'Saving in always-loaded mode removes reference files. Choose progressive disclosure to keep them.'
      )
    ).toBeInTheDocument()
    fireEvent.change(mode, { target: { value: 'progressive' } })
    expect(
      screen.getByRole('textbox', { name: 'Reference content' })
    ).toHaveValue('Unsaved reference')
    expect(api.put).not.toHaveBeenCalled()
  })

  it('重排合集后技能身份、用途和引用资源保持归属', async () => {
    const skills = [
      {
        id: 'review',
        title: 'Review',
        description: 'Review code',
        path: 'review/SKILL.md',
        content: 'Review instructions',
        resources: [{ path: 'references/review.md', content: 'Review rules' }],
      },
      {
        id: 'write',
        title: 'Write',
        description: 'Write prose',
        path: 'write/SKILL.md',
        content: 'Writing instructions',
        resources: [
          { path: 'references/writing.md', content: 'Writing style' },
        ],
      },
    ]
    renderEditor({
      ...template,
      kind: 'bundle',
      content: '',
      skills,
    } as unknown as PromptTemplate)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    fireEvent.click(screen.getByRole('button', { name: 'Move skill 2 up' }))
    expect(
      screen
        .getAllByRole('textbox', { name: 'Skill title' })
        .map((input) => (input as HTMLInputElement).value)
    ).toEqual(['Write', 'Review'])
    expect(
      screen
        .getAllByRole('textbox', { name: 'Reference content' })
        .map((input) => (input as HTMLTextAreaElement).value)
    ).toEqual(['Writing style', 'Review rules'])
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7',
        expect.objectContaining({ skills: [skills[1], skills[0]] })
      )
    )
  })
})

describe('技能用途说明长度', () => {
  const longDescription = '在审查发布说明前使用。'.repeat(90)

  it('单技能用途说明超过 512 字符时完整输入并原样提交', async () => {
    const user = userEvent.setup()
    renderEditor(structuredClone(template) as unknown as PromptTemplate)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    const description = screen.getByRole('textbox', { name: 'Description' })
    await user.clear(description)
    await user.click(description)
    await user.paste(longDescription)
    expect(description).toHaveValue(longDescription)
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7',
        expect.objectContaining({
          description: longDescription,
          skills: [
            expect.objectContaining({
              id: 'review',
              description: longDescription,
            }),
          ],
        })
      )
    )
  })

  it('合集技能用途说明超过 512 字符时完整输入并原样提交', async () => {
    const user = userEvent.setup()
    const skills = [
      {
        id: 'review',
        title: 'Review',
        description: 'Review code',
        path: 'review/SKILL.md',
        content: 'Review instructions',
        resources: [],
      },
      {
        id: 'write',
        title: 'Write',
        description: 'Write prose',
        path: 'write/SKILL.md',
        content: 'Writing instructions',
        resources: [],
      },
    ]
    renderEditor({
      ...template,
      kind: 'bundle',
      content: '',
      skills,
    } as unknown as PromptTemplate)
    await screen.findByRole('checkbox', { name: 'Premium models' })
    const description = screen.getAllByRole('textbox', {
      name: 'Skill description',
    })[0]
    await user.clear(description)
    await user.click(description)
    await user.paste(longDescription)
    expect(description).toHaveValue(longDescription)
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7',
        expect.objectContaining({
          skills: [{ ...skills[0], description: longDescription }, skills[1]],
        })
      )
    )
  })
})
