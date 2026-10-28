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
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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
  vi.mocked(api.get).mockImplementation(async (url) => {
    if (String(url).endsWith('auto-reply')) {
      return response({ enabled: true, content: 'Received' })
    }
    if (String(url).endsWith('auto-archive')) {
      return response({
        enabled: false,
        inactive_days: 30,
        statuses: ['resolved', 'closed'],
      })
    }
    return response({ page: 1, page_size: 20, total: 1, items: [ticket] })
  })
})
afterEach(() => {
  useAuthStore.getState().auth.reset()
  vi.resetAllMocks()
})

describe('工单管理设置折叠', () => {
  it('进入管理页先展示工单列表，自动回复与归档默认不可见', async () => {
    useAuthStore
      .getState()
      .auth.setUser({ id: 7, username: 'admin', role: 100 })
    render(<Tickets admin />, { wrapper: Wrapper })

    const table = await screen.findByRole('table', { name: 'All tickets' })
    const settings = screen.getByText('Ticket settings')
    expect(
      table.compareDocumentPosition(settings) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
    expect(settings.closest('details')).not.toHaveAttribute('open')
    expect(
      await screen.findByLabelText('Automatic reply message')
    ).not.toBeVisible()
    expect(await screen.findByLabelText('Inactive days')).not.toBeVisible()
  })

  it('展开统一设置后可编辑两项规则，收起再展开保留未保存草稿', async () => {
    const user = userEvent.setup()
    useAuthStore
      .getState()
      .auth.setUser({ id: 7, username: 'admin', role: 100 })
    render(<Tickets admin />, { wrapper: Wrapper })
    await screen.findByLabelText('Automatic reply message')
    const settings = screen.getByText('Ticket settings')

    await user.click(settings)
    const reply = screen.getByLabelText('Automatic reply message')
    const days = screen.getByLabelText('Inactive days')
    expect(reply).toBeVisible()
    expect(days).toBeVisible()
    await user.clear(reply)
    await user.type(reply, 'Unsaved reply')
    await user.clear(days)
    await user.type(days, '7')
    await user.click(settings)
    expect(reply).not.toBeVisible()
    expect(days).not.toBeVisible()
    await user.click(settings)
    expect(reply).toHaveValue('Unsaved reply')
    expect(days).toHaveValue(7)
    expect(
      screen.getByRole('button', { name: 'Save automatic reply' })
    ).toBeEnabled()
    expect(
      screen.getByRole('button', { name: 'Save automatic archive' })
    ).toBeEnabled()
    expect(api.put).not.toHaveBeenCalled()
  })

  it('切换管理员账号后设置恢复折叠，不保留前一账号的草稿', async () => {
    const user = userEvent.setup()
    useAuthStore
      .getState()
      .auth.setUser({ id: 7, username: 'admin', role: 100 })
    render(<Tickets admin />, { wrapper: Wrapper })
    await screen.findByLabelText('Automatic reply message')
    await user.click(screen.getByText('Ticket settings'))
    await user.clear(screen.getByLabelText('Automatic reply message'))
    await user.type(
      screen.getByLabelText('Automatic reply message'),
      'Account A draft'
    )

    act(() => {
      useAuthStore
        .getState()
        .auth.setUser({ id: 8, username: 'admin-b', role: 100 })
    })
    const reply = await screen.findByLabelText('Automatic reply message')
    expect(
      screen.getByText('Ticket settings').closest('details')
    ).not.toHaveAttribute('open')
    expect(reply).toHaveValue('Received')
    expect(reply).not.toBeVisible()
  })
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
