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
import { useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { PromptToken } from '@/features/prompt-center/types'
import { api } from '@/lib/api'

import { PlaygroundTokenSwitch } from '../components/playground-token-switch'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))

function renderSwitcher(disabled = false) {
  const onChange = vi.fn()
  function Harness() {
    const [token, setToken] = useState<PromptToken>()
    return (
      <PlaygroundTokenSwitch
        disabled={disabled}
        token={token}
        onChange={(next) => {
          setToken(next)
          onChange(next)
        }}
      />
    )
  }
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <Harness />
    </QueryClientProvider>
  )
  return { onChange }
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
              id: 7,
              name: 'Prompt testing',
              prompt_count: 2,
              group_details: [
                { id: 5, code: 'private-code', name: 'Advanced models' },
              ],
              key: 'secret-key',
            },
          ],
          total: 21,
        },
      },
    })
})

describe('操练场身份选择', () => {
  it('搜索分页后选择令牌显示名称和技能数量，并可返回账号模式', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSwitcher()
    expect(screen.getByText('Account mode')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Switch API key' }))
    const card = await screen.findByRole('button', { name: 'Prompt testing' })
    expect(within(card).getByText('Advanced models')).toBeInTheDocument()
    expect(screen.queryByText('private-code')).not.toBeInTheDocument()
    expect(screen.queryByText('secret-key')).not.toBeInTheDocument()
    await user.click(card)
    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() =>
      expect(api.get).toHaveBeenCalledWith('/api/token/search?p=2&size=20')
    )
    fireEvent.change(
      screen.getByRole('searchbox', { name: 'Search by name' }),
      { target: { value: 'Prompt' } }
    )
    await waitFor(() =>
      expect(api.get).toHaveBeenCalledWith(
        '/api/token/search?keyword=Prompt&p=1&size=20'
      )
    )
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Continue' })).toBeEnabled()
    )
    await user.click(screen.getByRole('button', { name: 'Continue' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.getByText('Prompt testing')).toBeInTheDocument()
    expect(screen.getByText('Skills · 2')).toBeInTheDocument()
    expect(onChange).toHaveBeenLastCalledWith({
      id: 7,
      name: 'Prompt testing',
      prompt_count: 2,
      enabled_prompt_count: undefined,
    })
    await user.click(screen.getByRole('button', { name: 'Use account mode' }))
    expect(screen.getByText('Account mode')).toBeInTheDocument()
    expect(onChange).toHaveBeenLastCalledWith(undefined)
  })

  it('生成期间禁止打开选择器且不发起令牌请求', () => {
    renderSwitcher(true)
    expect(
      screen.getByRole('button', { name: 'Switch API key' })
    ).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Switch API key' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalled()
  })
})
