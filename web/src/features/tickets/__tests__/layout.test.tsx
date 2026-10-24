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
import { render, screen } from '@testing-library/react'
import { useState, type ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { Tickets } from '..'
import { TicketDetailDialog } from '../ticket-detail'
import type { Ticket, TicketMessage } from '../types'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))
// 仅隔离第三方图标的浏览器 JSON 资源，不替换被测页面或业务逻辑。
vi.mock('@lobehub/icons', () => ({}))
vi.mock('@hugeicons/core-free-icons', () => ({
  Cancel01Icon: [],
  UnfoldMoreIcon: [],
  Tick02Icon: [],
}))

const ticket: Ticket = {
  id: 17,
  user_id: 7,
  title: 'Order payment question',
  category: 'billing',
  priority: 'normal',
  status: 'open',
  version: 1,
  created_at: 1792634400,
  updated_at: 1792634400,
}
const message: TicketMessage = {
  id: 51,
  ticket_id: 17,
  author_id: 7,
  author_role: 'user',
  content: 'My original question',
  created_at: 1792634400,
}

function response<T>(data: T) {
  return { data: { success: true, message: '', data } }
}

function Wrapper(props: { children: ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      })
  )
  return (
    <QueryClientProvider client={client}>{props.children}</QueryClientProvider>
  )
}

beforeEach(() => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 7, username: 'requester', role: 1 })
  vi.mocked(api.get).mockResolvedValue(
    response({ page: 1, page_size: 20, total: 1, items: [ticket] })
  )
})
afterEach(() => {
  useAuthStore.getState().auth.reset()
  vi.resetAllMocks()
})

describe('工单详情长文本布局', () => {
  it('列表连续长标题在列内换行，不覆盖状态和其他字段', async () => {
    const longTitle = 'W'.repeat(120)
    vi.mocked(api.get).mockResolvedValue(
      response({
        page: 1,
        page_size: 20,
        total: 1,
        items: [{ ...ticket, title: longTitle }],
      })
    )
    render(<Tickets />, { wrapper: Wrapper })
    const titleButton = await screen.findByRole('button', {
      name: `#17 ${longTitle}`,
    })
    expect(titleButton).toHaveClass('block', 'max-w-72', 'whitespace-normal')
  })
  it('连续长标题和正文限制在弹窗列宽内并允许纵向滚动', async () => {
    const longTitle = 'W'.repeat(120)
    const longContent = 'X'.repeat(5000)
    vi.mocked(api.get).mockResolvedValue(
      response({
        ticket: { ...ticket, title: longTitle },
        messages: [{ ...message, content: longContent }],
        has_more: false,
      })
    )
    render(<TicketDetailDialog id={17} admin={false} onClose={vi.fn()} />, {
      wrapper: Wrapper,
    })
    const heading = await screen.findByRole('heading', {
      name: `#17 ${longTitle}`,
    })
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveClass('grid-cols-1', 'overflow-y-auto')
    expect(dialog).toHaveClass('w-[calc(100vw-2rem)]')
    expect(heading).toHaveClass('[overflow-wrap:anywhere]')
    expect(screen.getByText(longContent)).toHaveClass(
      '[overflow-wrap:anywhere]'
    )
    expect(screen.getByRole('button', { name: 'Send reply' })).toBeVisible()
  })
})
