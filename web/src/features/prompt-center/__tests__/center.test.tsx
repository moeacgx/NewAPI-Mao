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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { PromptCenter } from '../index'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))

const template = {
  id: 7,
  title: 'Review',
  description: 'Review code',
  content: 'Check correctness',
  enabled: true,
  sort_order: 0,
  version: 1,
  group_ids: [12],
  group_details: [{ id: 12, name: 'Premium models' }],
}

beforeEach(() => {
  vi.mocked(api.get)
    .mockReset()
    .mockResolvedValue({ data: { success: true, data: [template] } })
  vi.mocked(api.put)
    .mockReset()
    .mockResolvedValue({ data: { success: true } })
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'test-user', role: 1 })
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

describe('提示词中心权限和浏览', () => {
  it('普通用户只能浏览目录和安装入口，不显示提示词正文', async () => {
    renderCenter()
    expect(await screen.findByText('Review')).toBeInTheDocument()
    expect(
      screen.getByText('Reusable prompts for your API keys')
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Create prompt' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Manage templates' })
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Install to token' })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Details' })
    ).not.toBeInTheDocument()
    expect(screen.queryByText('Check correctness')).not.toBeInTheDocument()
    expect(api.get).toHaveBeenCalledWith('/api/prompt-center/')
  })

  it('管理员切换管理模式后读取管理接口并展示编辑按钮', async () => {
    useAuthStore
      .getState()
      .auth.setUser({ id: 1, username: 'test-admin', role: 10 })
    renderCenter()
    await screen.findByText('Review')
    fireEvent.click(screen.getByRole('button', { name: 'Manage templates' }))
    expect(
      await screen.findByRole('button', { name: 'Edit' })
    ).toBeInTheDocument()
    expect(api.get).toHaveBeenCalledWith('/api/prompt-center/admin/')
  })

  it('管理员仍可在模板编辑器查看提示词正文', async () => {
    useAuthStore
      .getState()
      .auth.setUser({ id: 1, username: 'test-admin', role: 10 })
    renderCenter()
    await screen.findByText('Review')
    fireEvent.click(screen.getByRole('button', { name: 'Manage templates' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Edit' }))
    expect(
      await screen.findByRole('textbox', { name: 'Prompt content' })
    ).toHaveValue('Check correctness')
  })

  it('空结果仍保留明确的模块边界与搜索入口', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: true, data: [] },
    })
    renderCenter()
    const empty = await screen.findByText('No prompts found')
    expect(empty.closest('[data-slot="card"]')).toBeInTheDocument()
    expect(
      screen.getByRole('textbox', { name: 'Search prompts' })
    ).toBeInTheDocument()
  })

  it('完整安装流程只提交模板和令牌 ID，不读取完整密钥', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (String(url).startsWith('/api/token/search')) {
        return {
          data: {
            success: true,
            data: { items: [{ id: 3, name: 'Development' }], total: 1 },
          },
        }
      }
      if (url === '/api/token/3/prompts') {
        return { data: { success: true, data: [] } }
      }
      return { data: { success: true, data: [template] } }
    })
    renderCenter()
    fireEvent.click(
      await screen.findByRole('button', { name: 'Install to token' })
    )
    fireEvent.click(await screen.findByRole('button', { name: 'Development' }))
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }))
    fireEvent.click(
      await screen.findByRole('button', { name: 'Save installations' })
    )
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/token/3/prompts', {
        bindings: [{ template_id: 7, enabled: true, sort_order: 0 }],
      })
    )
    expect(
      vi
        .mocked(api.get)
        .mock.calls.some(([url]) => String(url).endsWith('/key'))
    ).toBe(false)
  })
})
