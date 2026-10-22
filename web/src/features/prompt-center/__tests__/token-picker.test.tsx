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
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { TokenPicker } from '../components/token-picker'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))

const token = {
  id: 3,
  name: 'Development',
  prompt_count: 2,
  enabled_prompt_count: 1,
}

function renderPicker() {
  const onSelect = vi.fn()
  const onClose = vi.fn()
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <TokenPicker onSelect={onSelect} onClose={onClose} />
    </QueryClientProvider>
  )
  return { onSelect, onClose }
}

beforeEach(() => {
  vi.mocked(api.get)
    .mockReset()
    .mockResolvedValue({
      data: {
        success: true,
        data: {
          items: [
            {
              ...token,
              key: 'private-key-must-never-be-rendered',
              content: 'private-prompt-body',
            },
          ],
          total: 1,
        },
      },
    })
})

describe('令牌搜索卡片选择器', () => {
  it('卡片显示安装数量，支持键盘选中与确认且不展示密钥或正文', async () => {
    const user = userEvent.setup()
    const { onSelect } = renderPicker()
    const card = await screen.findByRole('button', { name: 'Development' })
    expect(within(card).getByText('Skills · 2')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continue' })).toBeDisabled()
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
    card.focus()
    await user.keyboard('{Enter}')
    expect(card).toHaveAttribute('aria-pressed', 'true')
    await user.click(screen.getByRole('button', { name: 'Continue' }))
    expect(onSelect).toHaveBeenCalledWith(token)
    expect(
      screen.queryByText('private-key-must-never-be-rendered')
    ).not.toBeInTheDocument()
    expect(screen.queryByText('private-prompt-body')).not.toBeInTheDocument()
    expect(
      vi
        .mocked(api.get)
        .mock.calls.every(([url]) =>
          String(url).startsWith('/api/token/search?')
        )
    ).toBe(true)
  })

  it('跨页和搜索保留选中的对象，确认不会切换到当前页其他令牌', async () => {
    vi.mocked(api.get).mockImplementation(async (url) => {
      const params = new URL(String(url), 'http://localhost').searchParams
      return {
        data: {
          success: true,
          data: {
            items:
              params.get('p') === '2' || params.get('keyword')
                ? [{ id: 8, name: 'Production' }]
                : [token],
            total: 21,
          },
        },
      }
    })
    const { onSelect } = renderPicker()
    fireEvent.click(await screen.findByRole('button', { name: 'Development' }))
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    await screen.findByRole('button', { name: 'Production' })
    expect(screen.getByText('Development')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Production' })).toHaveAttribute(
      'aria-pressed',
      'false'
    )
    fireEvent.change(
      screen.getByRole('searchbox', { name: 'Search by name' }),
      { target: { value: 'Production' } }
    )
    await waitFor(() =>
      expect(api.get).toHaveBeenCalledWith(
        '/api/token/search?keyword=Production&p=1&size=20'
      )
    )
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Continue' })).toBeEnabled()
    )
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }))
    expect(onSelect).toHaveBeenCalledWith(token)
  })

  it('搜索失败保留摘要但禁止继续，重试成功后恢复', async () => {
    const { onSelect } = renderPicker()
    fireEvent.click(await screen.findByRole('button', { name: 'Development' }))
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: false, message: 'failed' },
    })
    fireEvent.change(
      screen.getByRole('searchbox', { name: 'Search by name' }),
      { target: { value: 'broken' } }
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Failed to load API keys'
    )
    expect(screen.getByText('Development')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continue' })).toBeDisabled()
    expect(onSelect).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await screen.findByRole('button', { name: 'Development' })
    expect(screen.getByRole('button', { name: 'Continue' })).toBeEnabled()
  })

  it('空结果保留搜索和分页边界，不能继续', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: true, data: { items: [], total: 0 } },
    })
    renderPicker()
    expect(await screen.findByText('No API keys found')).toBeInTheDocument()
    expect(
      screen.getByRole('searchbox', { name: 'Search by name' })
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Continue' })).toBeDisabled()
  })
})
