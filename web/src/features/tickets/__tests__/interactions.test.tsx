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
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState, type ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { Tickets } from '..'
import { CreateTicketDialog } from '../create-ticket-dialog'
import { FloatingTicketButton } from '../floating-ticket-button'
import { TicketDetailDialog } from '../ticket-detail'
import type { Ticket, TicketMessage, TicketStatus } from '../types'

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
      return response({ enabled: false, content: 'Received' })
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

describe('工单创建与浮窗', () => {
  it('空表单展示字段错误且不发送创建请求', async () => {
    const user = userEvent.setup()
    render(<CreateTicketDialog open onOpenChange={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await user.click(screen.getByRole('button', { name: 'Submit ticket' }))
    expect(await screen.findByText('Enter a ticket title.')).toBeVisible()
    expect(screen.getByLabelText('Ticket title')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    expect(screen.getByText('Enter a message.')).toBeVisible()
    expect(api.post).not.toHaveBeenCalled()
  })
  it('创建失败保留草稿，再次提交成功后关闭并刷新列表', async () => {
    const user = userEvent.setup()
    vi.mocked(api.post)
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce(response(ticket))
    render(<Tickets />, { wrapper: Wrapper })
    await screen.findByRole('button', { name: '#17 Order payment question' })
    await user.click(screen.getByRole('button', { name: 'Create ticket' }))
    await user.type(
      screen.getByLabelText('Ticket title'),
      'Order payment question'
    )
    await user.type(screen.getByLabelText('Message'), 'Please check this order')
    await user.selectOptions(screen.getByLabelText('Category'), 'billing')
    await user.click(screen.getByRole('button', { name: 'Submit ticket' }))
    expect(
      await screen.findByText(
        'Could not create the ticket. Your draft is preserved. Try again.'
      )
    ).toBeVisible()
    expect(screen.getByLabelText('Ticket title')).toHaveValue(
      'Order payment question'
    )
    expect(screen.getByLabelText('Message')).toHaveValue(
      'Please check this order'
    )
    await user.click(screen.getByRole('button', { name: 'Submit ticket' }))
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
    expect(api.post).toHaveBeenLastCalledWith('/api/tickets', {
      title: 'Order payment question',
      category: 'billing',
      priority: 'normal',
      content: 'Please check this order',
    })
    expect(
      vi.mocked(api.get).mock.calls.filter(([url]) => url === '/api/tickets')
        .length
    ).toBeGreaterThan(1)
  })
  it('未登录隐藏浮窗，登录后可键盘打开并在关闭时恢复焦点', async () => {
    const user = userEvent.setup()
    useAuthStore.getState().auth.setUser(null)
    render(<FloatingTicketButton />, { wrapper: Wrapper })
    expect(
      screen.queryByRole('button', { name: 'Create ticket' })
    ).not.toBeInTheDocument()
    act(() =>
      useAuthStore
        .getState()
        .auth.setUser({ id: 7, username: 'requester', role: 1 })
    )
    const trigger = screen.getByRole('button', { name: 'Create ticket' })
    trigger.focus()
    await user.keyboard('{Enter}')
    const dialog = await screen.findByRole('dialog', { name: 'Create ticket' })
    expect(dialog).toHaveClass('overflow-y-auto')
    expect(dialog).toHaveClass('max-h-[calc(100dvh-2rem)]')
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    await user.type(screen.getByLabelText('Ticket title'), 'Keep this draft')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(trigger).toHaveFocus())
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    await user.keyboard('{Enter}')
    expect(await screen.findByLabelText('Ticket title')).toHaveValue(
      'Keep this draft'
    )
  })
  it('创建请求处理中禁止重复提交和关闭', async () => {
    const user = userEvent.setup()
    let finish:
      | ((value: ReturnType<typeof response<Ticket>>) => void)
      | undefined
    vi.mocked(api.post).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        })
    )
    const onOpenChange = vi.fn()
    render(<CreateTicketDialog open onOpenChange={onOpenChange} />, {
      wrapper: Wrapper,
    })
    await user.type(screen.getByLabelText('Ticket title'), 'Request')
    await user.type(screen.getByLabelText('Message'), 'Details')
    await user.click(screen.getByRole('button', { name: 'Submit ticket' }))
    expect(
      await screen.findByRole('button', { name: 'Submitting...' })
    ).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    await user.keyboard('{Escape}')
    expect(onOpenChange).not.toHaveBeenCalled()
    await act(async () => finish?.(response(ticket)))
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
  })
})

describe('工单列表筛选与权限', () => {
  it('加载失败提供重试，恢复后展示空列表状态', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get)
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce(
        response({ page: 1, page_size: 20, total: 0, items: [] })
      )
    render(<Tickets />, { wrapper: Wrapper })
    expect(await screen.findByText('Could not load tickets.')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('No tickets found.')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  })
  it('切换分页并提交关键字时回到第一页，管理员个人中心仍调用本人接口', async () => {
    const user = userEvent.setup()
    useAuthStore.getState().auth.setUser({ id: 7, username: 'admin', role: 10 })
    vi.mocked(api.get).mockResolvedValue(
      response({ page: 1, page_size: 20, total: 21, items: [ticket] })
    )
    render(<Tickets />, { wrapper: Wrapper })
    await screen.findByRole('button', { name: '#17 Order payment question' })
    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() =>
      expect(api.get).toHaveBeenLastCalledWith(
        '/api/tickets',
        expect.objectContaining({ params: expect.objectContaining({ p: 2 }) })
      )
    )
    await user.type(screen.getByLabelText('Search tickets'), 'payment')
    await user.click(screen.getByRole('button', { name: 'Search' }))
    await waitFor(() =>
      expect(api.get).toHaveBeenLastCalledWith(
        '/api/tickets',
        expect.objectContaining({
          params: expect.objectContaining({ p: 1, keyword: 'payment' }),
        })
      )
    )
    expect(screen.queryByLabelText('User ID')).not.toBeInTheDocument()
  })
  it('缺少管理读取权限时不发送管理查询', async () => {
    useAuthStore.getState().auth.setUser({
      id: 7,
      username: 'admin',
      role: 10,
      permissions: {
        admin_permissions: { tickets: { read: false, write: true } },
      },
    })
    render(<Tickets admin />, { wrapper: Wrapper })
    expect(
      screen.getByText('You do not have permission to view tickets.')
    ).toBeVisible()
    expect(api.get).not.toHaveBeenCalled()
    expect(
      screen.queryByRole('button', { name: 'Create ticket' })
    ).not.toBeInTheDocument()
  })
  it('管理员按用户与状态筛选所有工单且不显示代用户创建', async () => {
    const user = userEvent.setup()
    useAuthStore.getState().auth.setUser({
      id: 7,
      username: 'admin',
      role: 10,
      permissions: {
        admin_permissions: { tickets: { read: true, write: true } },
      },
    })
    render(<Tickets admin />, { wrapper: Wrapper })
    await screen.findByRole('button', { name: '#17 Order payment question' })
    await user.type(screen.getByLabelText('User ID'), '9')
    await user.selectOptions(screen.getByLabelText('Status'), 'closed')
    await user.click(screen.getByRole('button', { name: 'Search' }))
    await waitFor(() =>
      expect(api.get).toHaveBeenLastCalledWith(
        '/api/ticket-admin',
        expect.objectContaining({
          params: expect.objectContaining({
            user_id: 9,
            status: 'closed',
            p: 1,
          }),
        })
      )
    )
    expect(
      screen.queryByRole('button', { name: 'Create ticket' })
    ).not.toBeInTheDocument()
  })
})

describe('工单会话', () => {
  it('管理员通过管理接口更新状态和回复，刷新后展示支持团队身份', async () => {
    const user = userEvent.setup()
    useAuthStore.getState().auth.setUser({
      id: 8,
      username: 'support',
      role: 10,
      permissions: {
        admin_permissions: { tickets: { read: true, write: true } },
      },
    })
    let currentTicket = { ...ticket }
    const currentMessages = [message]
    vi.mocked(api.get).mockImplementation(async () =>
      response({
        ticket: currentTicket,
        messages: [...currentMessages],
        has_more: false,
      })
    )
    vi.mocked(api.put).mockImplementation(async () => {
      currentTicket = { ...currentTicket, status: 'resolved' }
      return response(currentTicket)
    })
    vi.mocked(api.post).mockImplementation(async () => {
      const reply: TicketMessage = {
        ...message,
        id: 52,
        author_id: 8,
        author_role: 'admin',
        content: 'Payment confirmed',
      }
      currentMessages.push(reply)
      return response(reply)
    })
    render(<TicketDetailDialog id={17} admin onClose={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await screen.findByText('My original question')
    await user.selectOptions(screen.getByLabelText('Ticket status'), 'resolved')
    await waitFor(() =>
      expect(screen.getByLabelText('Ticket status')).toHaveValue('resolved')
    )
    expect(api.put).toHaveBeenCalledWith('/api/ticket-admin/17/status', {
      status: 'resolved',
    })
    await user.type(screen.getByLabelText('Reply'), 'Payment confirmed')
    await user.click(screen.getByRole('button', { name: 'Send reply' }))
    expect(await screen.findByText('Payment confirmed')).toBeVisible()
    expect(screen.getByText('Support team #8')).toBeVisible()
    expect(api.post).toHaveBeenCalledWith('/api/ticket-admin/17/messages', {
      content: 'Payment confirmed',
    })
  })

  it('手动刷新会话取得新回复并按文本展示上游内容', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get)
      .mockResolvedValueOnce(
        response({ ticket, messages: [message], has_more: false })
      )
      .mockResolvedValueOnce(
        response({
          ticket,
          messages: [
            message,
            { ...message, id: 52, content: '<img src=x onerror=alert(1)>' },
          ],
          has_more: false,
        })
      )
    render(<TicketDetailDialog id={17} admin={false} onClose={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await screen.findByText('My original question')
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(
      await screen.findByText('<img src=x onerror=alert(1)>')
    ).toBeVisible()
    expect(screen.getByRole('list').querySelector('img')).toBeNull()
  })

  it('加载较早消息时传递游标并按时间顺序展示，长文本允许换行', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get)
      .mockResolvedValueOnce(
        response({ ticket, messages: [message], has_more: true })
      )
      .mockResolvedValueOnce(
        response({
          ticket,
          messages: [{ ...message, id: 1, content: 'Earlier question' }],
          has_more: false,
        })
      )
    render(<TicketDetailDialog id={17} admin={false} onClose={vi.fn()} />, {
      wrapper: Wrapper,
    })
    await screen.findByText('My original question')
    await user.click(
      screen.getByRole('button', { name: 'Load earlier messages' })
    )
    expect(await screen.findByText('Earlier question')).toBeVisible()
    expect(api.get).toHaveBeenLastCalledWith(
      '/api/tickets/17',
      expect.objectContaining({ params: { before_message_id: 51 } })
    )
    const items = within(screen.getByRole('list')).getAllByRole('listitem')
    expect(items[0]).toHaveTextContent('Earlier question')
    expect(items[1]).toHaveTextContent('My original question')
    expect(screen.getByText('My original question')).toHaveClass(
      '[overflow-wrap:anywhere]'
    )
    expect(
      screen.queryByRole('button', { name: 'Load earlier messages' })
    ).not.toBeInTheDocument()
  })
  it('只读管理员能够看消息但不能回复或更改状态', async () => {
    useAuthStore.getState().auth.setUser({
      id: 8,
      username: 'reader',
      role: 10,
      permissions: {
        admin_permissions: { tickets: { read: true, write: false } },
      },
    })
    vi.mocked(api.get).mockResolvedValue(
      response({ ticket, messages: [message], has_more: false })
    )
    render(<TicketDetailDialog id={17} admin onClose={vi.fn()} />, {
      wrapper: Wrapper,
    })
    expect(await screen.findByText('My original question')).toBeVisible()
    expect(screen.getByLabelText('Ticket status')).toBeDisabled()
    expect(screen.getByLabelText('Reply')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Send reply' })).toBeDisabled()
    expect(
      screen.getByText('You have read-only access to tickets.')
    ).toBeVisible()
  })
  it.each(['closed', 'archived'] as const)(
    '%s 工单不能回复，显式重开后回复失败仍保留内容',
    async (initialStatus) => {
      const user = userEvent.setup()
      let status: TicketStatus = initialStatus
      vi.mocked(api.get).mockImplementation(async () =>
        response({
          ticket: { ...ticket, status },
          messages: [message],
          has_more: false,
        })
      )
      vi.mocked(api.put).mockImplementation(async () => {
        status = 'open'
        return response({ ...ticket, status })
      })
      vi.mocked(api.post)
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValueOnce(
          response({ ...message, id: 52, content: 'Please follow up' })
        )
      render(<TicketDetailDialog id={17} admin={false} onClose={vi.fn()} />, {
        wrapper: Wrapper,
      })
      expect(
        await screen.findByText(
          initialStatus === 'archived'
            ? 'This ticket is archived. Reopen it before replying.'
            : 'This ticket is closed. Reopen it before replying.'
        )
      ).toBeVisible()
      expect(screen.queryByLabelText('Reply')).not.toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: 'Reopen ticket' }))
      const reply = await screen.findByLabelText('Reply')
      expect(api.put).toHaveBeenCalledWith('/api/tickets/17/status', {
        status: 'open',
      })
      await user.type(reply, 'Please follow up')
      await user.click(screen.getByRole('button', { name: 'Send reply' }))
      expect(
        await screen.findByText(
          'Could not send the reply. Your draft is preserved. Try again.'
        )
      ).toBeVisible()
      expect(reply).toHaveValue('Please follow up')
      await user.click(screen.getByRole('button', { name: 'Send reply' }))
      await waitFor(() => expect(reply).toHaveValue(''))
      expect(api.post).toHaveBeenLastCalledWith('/api/tickets/17/messages', {
        content: 'Please follow up',
      })
    }
  )
})
