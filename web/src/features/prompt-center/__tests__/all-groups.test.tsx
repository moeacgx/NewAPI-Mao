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

const allGroupsLabel = 'All groups (no restriction)'

// 分组显示名称与 code 不同，用于确认界面只展示名称。
const premiumGroup = { id: 12, code: 'premium-code', name: 'Premium models' }

const source = {
  url: 'https://github.com/acme/review-skills/tree/main/skills/review',
  ref: 'main',
  path: 'skills/review',
  commit: 'a'.repeat(40),
  imported_at: 1760000000,
  content_hash: 'b'.repeat(64),
  locally_modified: false,
}

const restricted: PromptTemplate = {
  id: 7,
  title: 'Review',
  description: 'Review code',
  content: 'Check correctness',
  enabled: true,
  sort_order: 0,
  version: 4,
  all_groups: false,
  group_ids: [12],
  group_details: [premiumGroup],
}

const unrestricted: PromptTemplate = {
  ...restricted,
  id: 8,
  title: 'Global writing',
  all_groups: true,
  group_ids: [],
  group_details: [],
}

beforeEach(() => {
  vi.mocked(api.get)
    .mockReset()
    .mockImplementation(async (url) => {
      if (url === '/api/group/tags') {
        return { data: { success: true, data: [] } }
      }
      if (url === '/api/group/details') {
        return { data: { success: true, data: [premiumGroup] } }
      }
      if (String(url).includes('categories')) {
        return { data: { success: true, data: [] } }
      }
      return { data: { success: true, data: [unrestricted, restricted] } }
    })
  vi.mocked(api.post)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
  vi.mocked(api.put)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
  useAuthStore.getState().auth.setUser({ id: 2, username: 'member', role: 1 })
})

afterEach(() => useAuthStore.getState().auth.reset())

function renderWithQuery(children: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
}

describe('技能编辑器不限分组选项', () => {
  it('勾选不限分组后隐藏分组列表，提交空白名单并保留仓库来源', async () => {
    renderWithQuery(
      <PromptTemplateEditor
        template={{ ...restricted, repository_source: source }}
        onClose={vi.fn()}
      />
    )
    const allGroups = await screen.findByRole('checkbox', {
      name: allGroupsLabel,
    })
    expect(allGroups).not.toBeChecked()
    expect(
      await screen.findByRole('checkbox', { name: 'Premium models' })
    ).toBeChecked()
    fireEvent.click(allGroups)
    expect(
      screen.queryByRole('checkbox', { name: 'Premium models' })
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7',
        expect.objectContaining({
          all_groups: true,
          group_ids: [],
          enabled: true,
          version: 4,
          repository_source: source,
        })
      )
    )
  })

  it('打开不限分组的技能时选项已勾选且不展示分组列表', async () => {
    renderWithQuery(
      <PromptTemplateEditor template={unrestricted} onClose={vi.fn()} />
    )
    expect(
      await screen.findByRole('checkbox', { name: allGroupsLabel })
    ).toBeChecked()
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
    )
    expect(
      screen.queryByRole('checkbox', { name: 'Premium models' })
    ).not.toBeInTheDocument()
  })

  it('取消不限分组后发布必须重新选择分组', async () => {
    renderWithQuery(
      <PromptTemplateEditor template={unrestricted} onClose={vi.fn()} />
    )
    fireEvent.click(
      await screen.findByRole('checkbox', { name: allGroupsLabel })
    )
    const premium = await screen.findByRole('checkbox', {
      name: 'Premium models',
    })
    expect(premium).not.toBeChecked()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(
      await screen.findByText('Select at least one allowed group')
    ).toBeInTheDocument()
    expect(api.put).not.toHaveBeenCalled()
    fireEvent.click(premium)
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/prompt-center/admin/8',
        expect.objectContaining({ all_groups: false, group_ids: [12] })
      )
    )
  })

  it('发布校验接受不限分组或至少一个分组，缺少时错误指向白名单字段', () => {
    const input = {
      title: 'Review',
      description: 'Review code',
      content: 'Check correctness',
      enabled: true,
      sort_order: 0,
      version: 0,
      group_ids: [],
    }
    expect(
      promptTemplateSchema.safeParse({ ...input, all_groups: true }).success
    ).toBe(true)
    const missing = promptTemplateSchema.safeParse({
      ...input,
      all_groups: false,
    })
    expect(missing.success).toBe(false)
    expect(missing.error?.issues.map((issue) => issue.path)).toEqual([
      ['group_ids'],
    ])
  })
})

describe('不限分组的白名单展示', () => {
  it('top group summary shows unrestricted label when any skill has no restriction', async () => {
    renderWithQuery(<PromptCenter />)
    await screen.findByRole('heading', { name: 'Global writing' })
    const stat = screen.getAllByText('Allowed groups')[0].parentElement
    expect(stat).toHaveTextContent(allGroupsLabel)
  })

  it('top group summary counts unique groups for restricted skills', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url === '/api/group/tags') {
        return { data: { success: true, data: [] } }
      }
      if (url === '/api/group/details') {
        return { data: { success: true, data: [premiumGroup] } }
      }
      if (String(url).includes('categories')) {
        return { data: { success: true, data: [] } }
      }
      return {
        data: {
          success: true,
          data: [restricted, { ...restricted, id: 9 }],
        },
      }
    })
    renderWithQuery(<PromptCenter />)
    await screen.findAllByRole('heading', { name: 'Review' })
    const stat = screen.getAllByText('Allowed groups')[0].parentElement
    expect(stat).toHaveTextContent('1')
    expect(stat).not.toHaveTextContent(allGroupsLabel)
  })
  it('技能目录默认收起白名单，展开显示分组名称而非 code', async () => {
    renderWithQuery(<PromptCenter />)
    const global = (
      await screen.findByRole('heading', { name: 'Global writing' })
    ).closest('article') as HTMLElement
    const review = screen
      .getByRole('heading', { name: 'Review' })
      .closest('article') as HTMLElement
    expect(
      within(global).getByRole('button', { name: /Allowed groups.*All groups/ })
    ).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(
      within(global).getByRole('button', { name: /Allowed groups/ })
    )
    expect(within(global).getByText('All groups')).toBeInTheDocument()
    expect(
      within(global).queryByText('No available groups')
    ).not.toBeInTheDocument()
    expect(within(review).queryByText('Premium models')).not.toBeInTheDocument()
    fireEvent.click(
      within(review).getByRole('button', { name: /Allowed groups/ })
    )
    expect(within(review).getByText('Premium models')).toBeInTheDocument()
    expect(within(review).queryByText('All groups')).not.toBeInTheDocument()
    expect(screen.queryByText('premium-code')).not.toBeInTheDocument()
  })

  it('令牌安装列表对不限分组显示单个标签，白名单技能显示分组名称而非 code', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: [
          { template: unrestricted, enabled: true, sort_order: 0 },
          { template: restricted, enabled: true, sort_order: 1 },
        ],
      },
    })
    renderWithQuery(
      <TokenPromptDialog
        open
        onOpenChange={vi.fn()}
        token={{ id: 3, name: 'Development' }}
      />
    )
    const global = await screen.findByRole('region', {
      name: 'Global writing',
    })
    const review = screen.getByRole('region', { name: 'Review' })
    expect(within(global).getByText('All groups')).toBeInTheDocument()
    expect(
      within(global).queryByText('No available groups')
    ).not.toBeInTheDocument()
    expect(within(review).getByText('Premium models')).toBeInTheDocument()
    expect(within(review).queryByText('All groups')).not.toBeInTheDocument()
    expect(screen.queryByText('premium-code')).not.toBeInTheDocument()
  })
})
