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
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { Tickets } from '..'
import { FloatingTicketButton } from '../floating-ticket-button'
import type { Ticket } from '../types'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
}))
vi.mock('@hugeicons/core-free-icons', () => ({
  Cancel01Icon: [],
  UnfoldMoreIcon: [],
  Tick02Icon: [],
}))

const ticket: Ticket = {
  id: 17,
  user_id: 7,
  title: 'Account A request',
  category: 'general',
  priority: 'normal',
  status: 'open',
  version: 1,
  created_at: 1792634400,
  updated_at: 1792634400,
}
const accountA = {
  id: 7,
  username: 'account-a',
  role: 10,
  permissions: { admin_permissions: { tickets: { read: true, write: true } } },
}
const accountB = { ...accountA, id: 8, username: 'account-b' }

function response<T>(data: T) {
  return { data: { success: true, message: '', data } }
}

function deferredResponse() {
  let resolve!: (value: ReturnType<typeof response<Ticket>>) => void
  let reject!: (reason: Error) => void
  const promise = new Promise<ReturnType<typeof response<Ticket>>>(
    (resolvePromise, rejectPromise) => {
      resolve = resolvePromise
      reject = rejectPromise
    }
  )
  return { promise, resolve, reject }
}

beforeEach(() => {
  useAuthStore.getState().auth.setUser(accountA)
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
    if (String(url).endsWith('/17')) {
      return response({
        ticket,
        messages: [
          {
            id: 1,
            ticket_id: 17,
            author_id: 7,
            author_role: 'user',
            content: 'Original request',
            created_at: 1792634400,
          },
        ],
        has_more: false,
      })
    }
    return response({ page: 1, page_size: 20, total: 1, items: [ticket] })
  })
})
afterEach(() => {
  toast.dismiss()
  useAuthStore.getState().auth.reset()
  vi.resetAllMocks()
})

describe('账号切换后旧工单请求的回调隔离', () => {
  it.each([
    { surface: '浮窗', result: '成功' },
    { surface: '浮窗', result: '失败' },
    { surface: '个人工单页', result: '成功' },
    { surface: '个人工单页', result: '失败' },
  ])(
    '$surface 的 A 创建$result不能关闭 B 弹窗或弹出旧账号通知',
    async ({ surface, result }) => {
      const user = userEvent.setup()
      const pending = deferredResponse()
      vi.mocked(api.post).mockReturnValueOnce(pending.promise)
      const client = new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      })
      render(
        <QueryClientProvider client={client}>
          {surface === '浮窗' ? <FloatingTicketButton /> : <Tickets />}
          <Toaster />
        </QueryClientProvider>
      )
      await user.click(screen.getByRole('button', { name: 'Create ticket' }))
      await user.type(screen.getByLabelText('Ticket title'), 'A pending draft')
      await user.type(screen.getByLabelText('Message'), 'A message')
      await user.click(screen.getByRole('button', { name: 'Submit ticket' }))
      expect(
        await screen.findByRole('button', { name: 'Submitting...' })
      ).toBeDisabled()

      act(() => useAuthStore.getState().auth.setUser(accountB))
      await waitFor(() =>
        expect(screen.getByLabelText('Ticket title')).toHaveValue('')
      )
      await user.type(screen.getByLabelText('Ticket title'), 'B new draft')
      await user.type(screen.getByLabelText('Message'), 'B message')
      await act(async () => {
        if (result === '成功') pending.resolve(response(ticket))
        else pending.reject(new Error('old account request failed'))
        await pending.promise.catch(() => undefined)
      })
      await waitFor(() => expect(client.isMutating()).toBe(0))

      expect(
        screen.getByRole('dialog', { name: 'Create ticket' })
      ).toBeVisible()
      expect(screen.getByLabelText('Ticket title')).toHaveValue('B new draft')
      expect(screen.getByLabelText('Message')).toHaveValue('B message')
      expect(
        screen.getByRole('button', { name: 'Submit ticket' })
      ).toBeEnabled()
      expect(screen.queryByText('Ticket created.')).not.toBeInTheDocument()
      expect(
        screen.queryByText('Something went wrong!')
      ).not.toBeInTheDocument()
    }
  )

  it.each([
    { action: '回复', result: '成功' },
    { action: '回复', result: '失败' },
    { action: '状态变更', result: '成功' },
    { action: '状态变更', result: '失败' },
  ])(
    'A 的 $action $result 不能清除 B 回复草稿或弹出旧账号通知',
    async ({ action, result }) => {
      const user = userEvent.setup()
      const pending = deferredResponse()
      if (action === '回复') {
        vi.mocked(api.post).mockReturnValueOnce(pending.promise)
      } else {
        vi.mocked(api.put).mockReturnValueOnce(pending.promise)
      }
      const client = new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      })
      render(
        <QueryClientProvider client={client}>
          <Tickets admin />
          <Toaster />
        </QueryClientProvider>
      )
      await user.click(
        await screen.findByRole('button', { name: '#17 Account A request' })
      )
      await screen.findByText('Original request')
      if (action === '回复') {
        await user.type(screen.getByLabelText('Reply'), 'A pending reply')
        await user.click(screen.getByRole('button', { name: 'Send reply' }))
      } else {
        await user.selectOptions(
          screen.getByLabelText('Ticket status'),
          'resolved'
        )
      }
      await waitFor(() => expect(client.isMutating()).toBe(1))

      act(() => useAuthStore.getState().auth.setUser(accountB))
      await waitFor(() => expect(screen.getByLabelText('Reply')).toBeEnabled())
      await user.type(screen.getByLabelText('Reply'), 'B reply draft')
      await act(async () => {
        if (result === '成功') pending.resolve(response(ticket))
        else pending.reject(new Error('old account request failed'))
        await pending.promise.catch(() => undefined)
      })
      await waitFor(() => expect(client.isMutating()).toBe(0))

      expect(screen.getByLabelText('Reply')).toHaveValue('B reply draft')
      expect(screen.getByRole('button', { name: 'Send reply' })).toBeEnabled()
      expect(screen.queryByText('Reply sent.')).not.toBeInTheDocument()
      expect(
        screen.queryByText('Ticket status updated.')
      ).not.toBeInTheDocument()
      expect(
        screen.queryByText('Something went wrong!')
      ).not.toBeInTheDocument()
    }
  )
})
