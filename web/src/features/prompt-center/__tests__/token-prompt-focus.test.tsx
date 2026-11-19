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

import { TokenPromptBadge } from '../components/token-prompt-badge'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn(), put: vi.fn() } }))

const token = {
  id: 3,
  name: 'Development',
  prompt_count: 1,
  enabled_prompt_count: 1,
}
const zeroed = { ...token, prompt_count: 0, enabled_prompt_count: 0 }
const badge = { name: 'Manage installed skills (1)' }

beforeEach(() => {
  vi.mocked(api.get).mockReset()
  vi.mocked(api.get).mockResolvedValue({
    data: { success: true, data: [] },
  })
})

function mountRow(current: typeof token) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const row = (value: typeof token) => (
    <QueryClientProvider client={client}>
      <table>
        <tbody>
          <tr>
            <td>
              <input type='checkbox' aria-label='Select row' />
            </td>
            <td>
              <TokenPromptBadge token={value} />
              <button type='button'>Row action</button>
            </td>
          </tr>
        </tbody>
      </table>
    </QueryClientProvider>
  )
  const view = render(row(current))
  return { rerender: (value: typeof token) => view.rerender(row(value)) }
}

async function openDialog() {
  fireEvent.click(screen.getByRole('button', badge))
  return screen.findByRole('dialog')
}

async function pressEscape(dialog: HTMLElement) {
  fireEvent.keyDown(dialog, { key: 'Escape' })
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  )
}

describe('令牌技能弹窗关闭后的焦点', () => {
  it('无刷新直接打开后按 Escape，焦点回到同一个徽标', async () => {
    mountRow(token)
    const trigger = screen.getByRole('button', badge)
    await pressEscape(await openDialog())
    await waitFor(() => expect(trigger).toHaveFocus())
  })

  it('弹窗打开期间徽标经 1→0→1 重建，Escape 后焦点落到新连接的徽标', async () => {
    const view = mountRow(token)
    const dialog = await openDialog()
    const original = screen.getByRole('button', { ...badge, hidden: true })
    view.rerender(zeroed)
    view.rerender(token)
    const recreated = screen.getByRole('button', { ...badge, hidden: true })
    expect(recreated).not.toBe(original)
    await pressEscape(dialog)
    await waitFor(() => expect(recreated).toHaveFocus())
  })

  it('计数仍为 0 时 Escape 后焦点回到同一行的稳定控件，而不是页面根节点', async () => {
    const view = mountRow(token)
    const dialog = await openDialog()
    view.rerender(zeroed)
    await pressEscape(dialog)
    await waitFor(() =>
      expect(screen.getByRole('checkbox', { name: 'Select row' })).toHaveFocus()
    )
    expect(document.body).not.toHaveFocus()
  })

  it('关闭时焦点已在其他控件上，不抢走焦点', async () => {
    mountRow(token)
    const dialog = await openDialog()
    const other = document.createElement('button')
    other.textContent = 'Another dialog control'
    document.body.append(other)
    other.focus()
    fireEvent.keyDown(dialog, { key: 'Escape' })
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
    expect(other).toHaveFocus()
    other.remove()
  })
})
