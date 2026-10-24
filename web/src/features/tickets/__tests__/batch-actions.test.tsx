import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { Tickets } from '..'
import type { Ticket } from '../types'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn() },
}))
vi.mock('@hugeicons/core-free-icons', () => ({
  Cancel01Icon: [],
  UnfoldMoreIcon: [],
  Tick02Icon: [],
}))

const account = {
  id: 7,
  username: 'admin',
  role: 10,
  permissions: { admin_permissions: { tickets: { read: true, write: true } } },
}
const tickets: Ticket[] = [17, 18].map((id) => ({
  id,
  user_id: 3,
  title: `Request ${id}`,
  category: 'general',
  priority: 'normal',
  status: 'open',
  version: id - 15,
  created_at: 1792634400,
  updated_at: 1792634400,
}))
function response<T>(data: T) {
  return { data: { success: true, message: '', data } }
}
function mount() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <Tickets admin />
    </QueryClientProvider>
  )
  return client
}
beforeEach(() => {
  useAuthStore.getState().auth.setUser(account)
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
    return response({ page: 1, page_size: 20, total: 40, items: tickets })
  })
  vi.mocked(api.put).mockResolvedValue(response({ updated_count: 2 }))
  vi.mocked(api.post).mockResolvedValue(response({ deleted_count: 2 }))
})
afterEach(() => {
  useAuthStore.getState().auth.reset()
  vi.resetAllMocks()
})

describe('Default 管理员工单批量操作', () => {
  it('勾选当页后先确认再提交带版本的批量状态更新', async () => {
    const user = userEvent.setup()
    mount()
    await user.click(
      await screen.findByRole('checkbox', {
        name: 'Select all tickets on this page',
      })
    )
    expect(screen.getByText('Selected tickets: 2')).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Change selected status' })
    ).toHaveClass('max-w-full', 'whitespace-normal')
    expect(
      screen.getByRole('button', { name: 'Delete selected tickets' })
    ).toHaveClass('max-w-full', 'whitespace-normal')
    await user.selectOptions(
      screen.getByLabelText('Batch ticket status'),
      'archived'
    )
    await user.click(
      screen.getByRole('button', { name: 'Change selected status' })
    )
    expect(api.put).not.toHaveBeenCalled()
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Confirm',
      })
    )
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/ticket-admin/batch/status', {
        tickets: [
          { id: 17, version: 2 },
          { id: 18, version: 3 },
        ],
        status: 'archived',
      })
    )
    await waitFor(() =>
      expect(screen.getByText('Selected tickets: 0')).toBeVisible()
    )
  })
  it('批量删除显示软删除边界，取消不会发送请求', async () => {
    const user = userEvent.setup()
    mount()
    await user.click(
      await screen.findByRole('checkbox', { name: 'Select ticket #17' })
    )
    await user.click(
      screen.getByRole('button', { name: 'Delete selected tickets' })
    )
    expect(
      screen.getByText(
        'Deleted tickets disappear from both users and administrators. Conversation and audit records are retained; there is no restore action.'
      )
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(api.post).not.toHaveBeenCalled()
    await user.click(
      screen.getByRole('button', { name: 'Delete selected tickets' })
    )
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Confirm delete',
      })
    )
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith('/api/ticket-admin/batch/delete', {
        tickets: [{ id: 17, version: 2 }],
      })
    )
  })
  it('批量失败保留选择并提示刷新，刷新后清空旧版本选择', async () => {
    const user = userEvent.setup()
    vi.mocked(api.put).mockRejectedValue(
      new Error('Ticket changed, please refresh')
    )
    mount()
    await user.click(
      await screen.findByRole('checkbox', { name: 'Select ticket #17' })
    )
    await user.click(
      screen.getByRole('button', { name: 'Change selected status' })
    )
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Confirm',
      })
    )
    expect(
      await screen.findByText('Ticket changed, please refresh')
    ).toBeVisible()
    expect(
      screen.getByText(
        'Refresh the list and select tickets again before retrying.'
      )
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(
      screen.getByRole('checkbox', { name: 'Select ticket #17' })
    ).toBeChecked()
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(screen.getByText('Selected tickets: 0')).toBeVisible()
  })
  it('翻页、筛选与账号切换清空勾选', async () => {
    const user = userEvent.setup()
    mount()
    await user.click(
      await screen.findByRole('checkbox', { name: 'Select ticket #17' })
    )
    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() =>
      expect(screen.getByText('Selected tickets: 0')).toBeVisible()
    )
    await user.click(
      screen.getByRole('checkbox', { name: 'Select ticket #17' })
    )
    await user.selectOptions(screen.getByLabelText('Status'), 'resolved')
    await waitFor(() =>
      expect(screen.getByText('Selected tickets: 0')).toBeVisible()
    )
    await user.click(
      screen.getByRole('checkbox', { name: 'Select ticket #17' })
    )
    act(() => useAuthStore.getState().auth.setUser({ ...account, id: 8 }))
    await waitFor(() =>
      expect(screen.getByText('Selected tickets: 0')).toBeVisible()
    )
  })
  it('只读管理员不显示批量选择或写入操作', async () => {
    useAuthStore.getState().auth.setUser({
      ...account,
      permissions: {
        admin_permissions: { tickets: { read: true, write: false } },
      },
    })
    mount()
    await screen.findByRole('button', { name: '#17 Request 17' })
    expect(
      screen.queryByRole('checkbox', { name: 'Select ticket #17' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Delete selected tickets' })
    ).not.toBeInTheDocument()
  })
  it('旧账号延迟批量响应不清除新账号选择', async () => {
    let resolve!: (value: ReturnType<typeof response>) => void
    const pending = new Promise<ReturnType<typeof response>>((done) => {
      resolve = done
    })
    vi.mocked(api.put).mockReturnValueOnce(pending)
    const user = userEvent.setup()
    mount()
    await user.click(
      await screen.findByRole('checkbox', { name: 'Select ticket #17' })
    )
    await user.click(
      screen.getByRole('button', { name: 'Change selected status' })
    )
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Confirm',
      })
    )
    act(() => useAuthStore.getState().auth.setUser({ ...account, id: 8 }))
    await user.click(
      await screen.findByRole('checkbox', { name: 'Select ticket #18' })
    )
    await act(async () => {
      resolve(response({ updated_count: 1 }))
      await pending
    })
    expect(
      screen.getByRole('checkbox', { name: 'Select ticket #18' })
    ).toBeChecked()
  })
})
