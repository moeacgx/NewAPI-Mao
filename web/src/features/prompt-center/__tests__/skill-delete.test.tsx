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
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { PromptCenter } from '../index'
import type { PromptTemplate } from '../types'

vi.mock('@lobehub/icons', () => ({}))

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

const template: PromptTemplate = {
  id: 12,
  title: 'Release review',
  description: 'Review release notes before publishing',
  enabled: true,
  sort_order: 0,
  version: 5,
  group_ids: [3],
  group_details: [{ id: 3, name: 'Premium models' }],
}

const deleteLabel = 'Delete skill Release review'
const conflictMessage = '提示词已被其他管理员更新，请刷新后重试'

let serverTemplates: PromptTemplate[] = []

beforeEach(() => {
  serverTemplates = [template]
  vi.mocked(api.get)
    .mockReset()
    .mockImplementation(async (url) => {
      if (String(url).includes('categories')) {
        return { data: { success: true, data: [] } }
      }
      return { data: { success: true, data: serverTemplates } }
    })
  vi.mocked(api.delete)
    .mockReset()
    .mockImplementation(async () => {
      serverTemplates = []
      return {
        data: { success: true, data: { id: 12, uninstalled_tokens: 3 } },
      }
    })
  useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
})

afterEach(() => {
  useAuthStore.getState().auth.reset()
  toast.dismiss()
})

function renderCenter() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <PromptCenter />
      <Toaster />
    </QueryClientProvider>
  )
  return client
}

async function openDeleteConfirmation() {
  fireEvent.click(
    await screen.findByRole('button', { name: 'Manage templates' })
  )
  fireEvent.click(await screen.findByRole('button', { name: deleteLabel }))
  return screen.findByRole('alertdialog')
}

describe('管理员删除技能', () => {
  it('普通用户浏览技能时不显示删除按钮', async () => {
    useAuthStore.getState().auth.setUser({ id: 2, username: 'member', role: 1 })
    renderCenter()
    expect(await screen.findByText('Release review')).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: deleteLabel })
    ).not.toBeInTheDocument()
  })

  it('管理员只在管理模式下看到带技能标题的删除按钮', async () => {
    renderCenter()
    await screen.findByText('Release review')
    expect(
      screen.queryByRole('button', { name: deleteLabel })
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Manage templates' }))
    expect(
      await screen.findByRole('button', { name: deleteLabel })
    ).toBeEnabled()
  })

  it('取消确认时不发送删除请求并保留技能卡片', async () => {
    renderCenter()
    const dialog = await openDeleteConfirmation()
    expect(dialog).toHaveTextContent(
      'Deleting "Release review" automatically uninstalls it from every token that has it installed. This action cannot be undone.'
    )
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    expect(api.delete).not.toHaveBeenCalled()
    expect(screen.getByText('Release review')).toBeInTheDocument()
  })

  it('确认删除后按当前版本请求，移除卡片并失效令牌相关缓存', async () => {
    const client = renderCenter()
    const relatedKeys = [
      ['token-prompts', 3],
      ['keys'],
      ['prompt-center-tokens', '', 1],
    ]
    for (const key of relatedKeys) client.setQueryData(key, [])
    await openDeleteConfirmation()
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() =>
      expect(api.delete).toHaveBeenCalledWith(
        '/api/prompt-center/admin/12?version=5',
        expect.anything()
      )
    )
    expect(await screen.findByText('No prompts found')).toBeInTheDocument()
    expect(screen.queryByText('Release review')).not.toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(
      await screen.findByText('Skill deleted. Uninstalled from 3 token(s).')
    ).toBeInTheDocument()
    for (const key of relatedKeys) {
      expect(client.getQueryState(key)?.isInvalidated).toBe(true)
    }
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
      '业务失败',
      () =>
        vi.mocked(api.delete).mockResolvedValue({
          data: { success: false, message: 'Skill is used by a pending job' },
        }),
      'Skill is used by a pending job',
    ],
  ])('%s 时保留技能卡片并展示服务端错误', async (_, mockFailure, message) => {
    mockFailure()
    renderCenter()
    await openDeleteConfirmation()
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(message)
    expect(screen.getByText('Release review')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeEnabled()
  })

  it('删除请求进行中禁用确认并阻止重复提交', async () => {
    let resolveDelete!: (value: {
      data: {
        success: boolean
        data: { id: number; uninstalled_tokens: number }
      }
    }) => void
    vi.mocked(api.delete).mockReturnValue(
      new Promise((resolve) => {
        resolveDelete = resolve
      })
    )
    renderCenter()
    await openDeleteConfirmation()
    const confirm = screen.getByRole('button', { name: 'Delete' })
    fireEvent.click(confirm)
    await waitFor(() => expect(confirm).toBeDisabled())
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    fireEvent.click(confirm)
    expect(api.delete).toHaveBeenCalledTimes(1)
    serverTemplates = []
    await act(async () =>
      resolveDelete({
        data: { success: true, data: { id: 12, uninstalled_tokens: 0 } },
      })
    )
    expect(await screen.findByText('No prompts found')).toBeInTheDocument()
  })
})
