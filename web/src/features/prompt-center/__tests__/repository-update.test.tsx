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
import { AxiosError } from 'axios'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { SkillRepositoryDetails } from '../components/skill-repository-details'
import { PromptCenter } from '../index'

vi.mock('@lobehub/icons', () => ({}))

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))

const source = {
  url: 'https://github.com/acme/review-skills/tree/main/skills/review',
  ref: 'main',
  path: 'skills/review',
  commit: 'a'.repeat(40),
  imported_at: 1792857600,
  content_hash: 'b'.repeat(64),
  locally_modified: false,
}

const template = {
  id: 7,
  title: 'Review',
  description: 'Review code',
  content: 'Existing private skill body',
  enabled: true,
  sort_order: 0,
  version: 4,
  group_ids: [12],
  group_details: [{ id: 12, name: 'Premium models' }],
  repository_source: source,
}

beforeEach(() => {
  vi.mocked(api.get)
    .mockReset()
    .mockImplementation(async (url) => {
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
      if (
        String(url).includes('categories') ||
        String(url).includes('/store/')
      ) {
        return { data: { success: true, data: [] } }
      }
      return { data: { success: true, data: [template] } }
    })
  vi.mocked(api.post)
    .mockReset()
    .mockResolvedValue({
      data: { success: true, data: { template, updated: true } },
    })
  vi.mocked(api.put)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
})

afterEach(() => useAuthStore.getState().auth.reset())

function renderCenter() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <PromptCenter />
    </QueryClientProvider>
  )
}

async function renderAdmin() {
  renderCenter()
  fireEvent.click(
    await screen.findByRole('button', { name: 'Manage templates' })
  )
  await screen.findByRole('button', { name: 'Edit' })
}

describe('GitHub 技能来源与更新', () => {
  it('不安全来源仅展示文本，不生成可执行链接', () => {
    render(
      <SkillRepositoryDetails
        source={{ ...source, url: 'javascript:alert(1)' }}
      />
    )
    expect(screen.getByText('javascript:alert(1)')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })

  it('管理员展示仓库、引用、目录和短提交，普通用户不显示来源', async () => {
    await renderAdmin()
    expect(
      screen.getByRole('link', { name: 'acme/review-skills' })
    ).toHaveAttribute('href', source.url)
    expect(screen.getByText('main')).toBeInTheDocument()
    expect(screen.getByText('skills/review')).toBeInTheDocument()
    expect(screen.getByText('aaaaaaa')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Browse prompts' }))
    expect(
      screen.queryByRole('button', { name: 'Update from GitHub' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('link', { name: 'acme/review-skills' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByText('Existing private skill body')
    ).not.toBeInTheDocument()
  })

  it('没有来源的历史模板不显示更新入口', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: [{ ...template, repository_source: undefined }],
      },
    })
    await renderAdmin()
    expect(
      screen.queryByRole('button', { name: 'Update from GitHub' })
    ).not.toBeInTheDocument()
  })

  it('GitHub 导入后保存完整来源和正文，不发布未审核草稿', async () => {
    vi.mocked(api.post)
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: {
            ...template,
            id: 0,
            enabled: false,
            group_ids: [],
            load_mode: 'progressive',
          },
        },
      })
      .mockResolvedValue({ data: { success: true } })
    renderCenter()
    fireEvent.click(await screen.findByRole('button', { name: 'Skill store' }))
    fireEvent.change(
      screen.getByRole('textbox', { name: 'Import from GitHub repository' }),
      { target: { value: source.url } }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Import repository' }))
    await screen.findByRole('textbox', { name: 'Prompt content' })
    await screen.findByRole('checkbox', { name: 'Premium models' })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        '/api/prompt-center/admin/',
        expect.objectContaining({
          repository_source: source,
          content: 'Existing private skill body',
          enabled: false,
          group_ids: [],
        })
      )
    )
  })

  it('没有本地修改时一次点击提交版本并显示更新成功', async () => {
    await renderAdmin()
    fireEvent.click(screen.getByRole('button', { name: 'Update from GitHub' }))
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7/update-repository',
        { version: 4, overwrite_local: false },
        expect.anything()
      )
    )
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Skill updated from GitHub'
    )
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('仓库没有变化时显示已是最新版本', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { success: true, data: { template, updated: false } },
    })
    await renderAdmin()
    fireEvent.click(screen.getByRole('button', { name: 'Update from GitHub' }))
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Skill is already up to date'
    )
  })

  it.each(['GitHub unavailable', 'Version conflict'])(
    '更新失败 %s 时保留原副本并允许重试',
    async (message) => {
      vi.mocked(api.post).mockRejectedValue(
        new AxiosError(message, undefined, undefined, undefined, {
          status: 409,
          data: { success: false, message },
        } as never)
      )
      await renderAdmin()
      fireEvent.click(
        screen.getByRole('button', { name: 'Update from GitHub' })
      )
      expect(await screen.findByRole('alert')).toHaveTextContent(message)
      expect(screen.getByText('Review')).toBeInTheDocument()
      expect(screen.getByText('aaaaaaa')).toBeInTheDocument()
      expect(
        screen.getByRole('button', { name: 'Update from GitHub' })
      ).toBeEnabled()
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    }
  )

  it('已知本地修改先确认，取消不调用，确认后提交覆盖标志', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: [
          {
            ...template,
            repository_source: { ...source, locally_modified: true },
          },
        ],
      },
    })
    await renderAdmin()
    fireEvent.click(screen.getByRole('button', { name: 'Update from GitHub' }))
    expect(await screen.findByRole('alertdialog')).toHaveTextContent(
      'Replace locally edited skill content?'
    )
    expect(api.post).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    fireEvent.click(screen.getByRole('button', { name: 'Update from GitHub' }))
    fireEvent.click(screen.getByRole('button', { name: 'Replace and update' }))
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        '/api/prompt-center/admin/7/update-repository',
        { version: 4, overwrite_local: true },
        expect.anything()
      )
    )
  })

  it('服务端报告新本地修改时再确认，普通版本冲突不会被强制覆盖', async () => {
    vi.mocked(api.post).mockRejectedValueOnce(
      new AxiosError('Conflict', undefined, undefined, undefined, {
        status: 409,
        data: {
          success: false,
          message: 'Local changes',
          code: 'repository_local_changes',
        },
      } as never)
    )
    await renderAdmin()
    fireEvent.click(screen.getByRole('button', { name: 'Update from GitHub' }))
    expect(await screen.findByRole('alertdialog')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Replace and update' }))
    await waitFor(() =>
      expect(api.post).toHaveBeenLastCalledWith(
        '/api/prompt-center/admin/7/update-repository',
        { version: 4, overwrite_local: true },
        expect.anything()
      )
    )
  })

  it('请求未完成时禁用更新并防止重复提交', async () => {
    let resolveUpdate!: (value: {
      data: {
        success: boolean
        data: { template: typeof template; updated: boolean }
      }
    }) => void
    vi.mocked(api.post).mockReturnValue(
      new Promise((resolve) => {
        resolveUpdate = resolve
      })
    )
    await renderAdmin()
    const update = screen.getByRole('button', { name: 'Update from GitHub' })
    fireEvent.click(update)
    await waitFor(() => expect(update).toBeDisabled())
    fireEvent.click(update)
    expect(api.post).toHaveBeenCalledTimes(1)
    await act(async () =>
      resolveUpdate({
        data: { success: true, data: { template, updated: true } },
      })
    )
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Skill updated from GitHub'
    )
  })
})
