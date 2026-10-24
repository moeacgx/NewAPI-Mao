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
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { CategoryManager } from '../components/category-manager'
import { SkillImportDialog } from '../components/skill-import-dialog'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))

function renderQuery(children: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
}

beforeEach(() => {
  vi.mocked(api.get)
    .mockReset()
    .mockResolvedValue({
      data: {
        success: true,
        data: [{ id: 3, name: 'Communication', version: 4 }],
      },
    })
  vi.mocked(api.post)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
  vi.mocked(api.put)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
})

describe('技能分类管理', () => {
  it('新增分类提交修剪后名称并刷新列表', async () => {
    renderQuery(<CategoryManager onClose={vi.fn()} />)
    await screen.findByText('Communication')
    fireEvent.change(
      screen.getByRole('textbox', { name: 'New skill category' }),
      { target: { value: ' Development ' } }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Add category' }))
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        '/api/prompt-center/admin/categories/',
        { name: 'Development', version: 0 }
      )
    )
    await waitFor(() =>
      expect(
        screen.getByRole('textbox', { name: 'New skill category' })
      ).toHaveValue('')
    )
  })

  it('重命名冲突保留名称草稿和原版本，不改变列表名称', async () => {
    vi.mocked(api.put).mockResolvedValueOnce({
      data: { success: false, message: 'Version conflict' },
    })
    renderQuery(<CategoryManager onClose={vi.fn()} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Rename' }))
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Rename skill category' }),
      { target: { value: 'My category draft' } }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Version conflict'
    )
    expect(
      screen.getByRole('textbox', { name: 'Rename skill category' })
    ).toHaveValue('My category draft')
    expect(screen.getByText('Communication')).toBeInTheDocument()
    expect(api.put).toHaveBeenCalledWith(
      '/api/prompt-center/admin/categories/3',
      { name: 'My category draft', version: 4 }
    )
  })
})

describe('技能商店与 ZIP 导入', () => {
  const entry = {
    id: 'bundled:writing-kit',
    title: 'Writing kit',
    description: 'Write and review',
    kind: 'bundle',
    category: 'Writing',
    skill_count: 2,
    source: 'bundled',
    version: '1',
  }
  const draft = {
    title: 'Imported writing kit',
    kind: 'bundle',
    skills: [
      { title: 'Draft', content: 'Draft instructions' },
      { title: 'Review', content: 'Review instructions' },
    ],
    category_id: 0,
    enabled: false,
    group_ids: [],
    version: 0,
  }

  it('在线刷新失败保留内置目录，仍可导入副本而不直接保存', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => ({
      data:
        url === '/api/prompt-center/admin/store/?source=online'
          ? { success: false, message: 'Offline' }
          : { success: true, data: [entry] },
    }))
    vi.mocked(api.post).mockResolvedValue({
      data: { success: true, data: draft },
    })
    const onImported = vi.fn()
    renderQuery(<SkillImportDialog onClose={vi.fn()} onImported={onImported} />)
    await screen.findByText('Writing kit')
    fireEvent.click(
      screen.getByRole('button', { name: 'Refresh online store' })
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Online refresh failed. Your current store list is still available.'
    )
    expect(screen.getByText('Writing kit')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Import and edit' }))
    await waitFor(() =>
      expect(onImported).toHaveBeenCalledWith({
        ...draft,
        load_mode: 'progressive',
        id: 0,
      })
    )
    expect(api.post).toHaveBeenCalledExactlyOnceWith(
      '/api/prompt-center/admin/store/bundled%3Awriting-kit/import'
    )
  })

  it('商店搜索空结果保留搜索和上传入口', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: [entry] },
    })
    renderQuery(<SkillImportDialog onClose={vi.fn()} onImported={vi.fn()} />)
    await screen.findByText('Writing kit')
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Search skill store' }),
      { target: { value: 'Unknown skill' } }
    )
    expect(screen.getByText('No skills found in store')).toBeInTheDocument()
    expect(
      screen.getByLabelText('Import skill ZIP archive')
    ).toBeInTheDocument()
  })

  it('ZIP 上传失败保持弹窗和当前列表，不调用草稿替换', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: [entry] },
    })
    vi.mocked(api.post).mockResolvedValue({
      data: { success: false, message: 'Invalid archive' },
    })
    const onImported = vi.fn()
    renderQuery(<SkillImportDialog onClose={vi.fn()} onImported={onImported} />)
    await screen.findByText('Writing kit')
    const file = new File(['archive'], 'skills.zip', {
      type: 'application/zip',
    })
    fireEvent.change(screen.getByLabelText('Import skill ZIP archive'), {
      target: { files: [file] },
    })
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Invalid archive'
    )
    expect(onImported).not.toHaveBeenCalled()
    expect(screen.getByText('Writing kit')).toBeInTheDocument()
    const body = vi.mocked(api.post).mock.calls[0][1] as FormData
    expect(body.get('file')).toBe(file)
  })

  it('仓库链接失败保留 URL 和列表，重试成功后只打开未发布草稿', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: [entry] },
    })
    vi.mocked(api.post)
      .mockResolvedValueOnce({
        data: { success: false, message: 'Repository unavailable' },
      })
      .mockResolvedValueOnce({ data: { success: true, data: draft } })
    const onImported = vi.fn()
    renderQuery(<SkillImportDialog onClose={vi.fn()} onImported={onImported} />)
    await screen.findByText('Writing kit')
    const input = screen.getByRole('textbox', {
      name: 'Import from GitHub repository',
    })
    fireEvent.change(input, {
      target: { value: 'https://github.com/example/skills/tree/main/writing' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Import repository' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Repository unavailable'
    )
    expect(input).toHaveValue(
      'https://github.com/example/skills/tree/main/writing'
    )
    expect(screen.getByText('Writing kit')).toBeInTheDocument()
    expect(onImported).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Import repository' }))
    await waitFor(() =>
      expect(onImported).toHaveBeenCalledWith({
        ...draft,
        load_mode: 'progressive',
        id: 0,
      })
    )
    expect(api.post).toHaveBeenCalledTimes(2)
    expect(api.post).toHaveBeenLastCalledWith(
      '/api/prompt-center/admin/import/repository',
      { url: 'https://github.com/example/skills/tree/main/writing' }
    )
  })

  it('仓库导入期间禁止重复提交和关闭，完成后恢复操作', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: true, data: [entry] },
    })
    let finish: ((value: unknown) => void) | undefined
    vi.mocked(api.post).mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve
        })
    )
    const onClose = vi.fn()
    const onImported = vi.fn()
    renderQuery(<SkillImportDialog onClose={onClose} onImported={onImported} />)
    await screen.findByText('Writing kit')
    const input = screen.getByRole('textbox', {
      name: 'Import from GitHub repository',
    })
    fireEvent.change(input, {
      target: { value: 'https://github.com/example/skills' },
    })
    const button = screen.getByRole('button', { name: 'Import repository' })
    fireEvent.click(button)
    await waitFor(() => expect(button).toBeDisabled())
    expect(input).toBeDisabled()
    expect(
      screen.queryByRole('button', { name: 'Close' })
    ).not.toBeInTheDocument()
    fireEvent.click(button)
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' })
    expect(api.post).toHaveBeenCalledTimes(1)
    expect(onClose).not.toHaveBeenCalled()
    await act(async () => {
      finish?.({ data: { success: true, data: draft } })
    })
    await waitFor(() =>
      expect(onImported).toHaveBeenCalledWith({
        ...draft,
        load_mode: 'progressive',
        id: 0,
      })
    )
    expect(button).not.toBeDisabled()
  })
})
