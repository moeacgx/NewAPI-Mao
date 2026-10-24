import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { TicketAutoArchiveSettings } from '../ticket-auto-archive-settings'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn(), put: vi.fn() } }))
vi.mock('@hugeicons/core-free-icons', () => ({ Tick02Icon: [] }))
const config = {
  enabled: false,
  inactive_days: 30,
  statuses: ['resolved', 'closed'],
}
function response<T>(data: T) {
  return { data: { success: true, message: '', data } }
}
function mount(canWrite = true) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <TicketAutoArchiveSettings canWrite={canWrite} />
    </QueryClientProvider>
  )
  return client
}
beforeEach(() => {
  useAuthStore.getState().auth.setUser({ id: 7, username: 'admin', role: 100 })
  vi.mocked(api.get).mockResolvedValue(response(config))
  vi.mocked(api.put).mockImplementation(async (_url, value) => response(value))
})
afterEach(() => {
  useAuthStore.getState().auth.reset()
  vi.resetAllMocks()
})
describe('Default 超时自动归档配置', () => {
  it('读取默认关闭30天及已解决和已关闭，保存显式启用配置', async () => {
    const user = userEvent.setup()
    mount()
    expect(await screen.findByLabelText('Inactive days')).toHaveValue(30)
    expect(screen.getByRole('switch')).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Resolved' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Closed' })).toBeChecked()
    await user.click(screen.getByRole('switch'))
    await user.clear(screen.getByLabelText('Inactive days'))
    await user.type(screen.getByLabelText('Inactive days'), '7')
    await user.click(
      screen.getByRole('button', { name: 'Save automatic archive' })
    )
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/ticket-admin/automation/auto-archive',
        { ...config, enabled: true, inactive_days: 7 }
      )
    )
  })
  it.each(['0', '3651', '1.5'])('归档天数 %s 不提交', async (days) => {
    const user = userEvent.setup()
    mount()
    await user.clear(await screen.findByLabelText('Inactive days'))
    await user.type(screen.getByLabelText('Inactive days'), days)
    await user.click(
      screen.getByRole('button', { name: 'Save automatic archive' })
    )
    expect(
      await screen.findByText('Enter a whole number of days from 1 to 3650.')
    ).toBeVisible()
    expect(api.put).not.toHaveBeenCalled()
  })
  it('无状态不提交，保存失败与后台刷新保留编辑草稿', async () => {
    const user = userEvent.setup()
    const client = mount()
    await user.click(await screen.findByRole('checkbox', { name: 'Resolved' }))
    await user.click(screen.getByRole('checkbox', { name: 'Closed' }))
    await user.click(
      screen.getByRole('button', { name: 'Save automatic archive' })
    )
    expect(
      await screen.findByText('Select at least one ticket status.')
    ).toBeVisible()
    expect(api.put).not.toHaveBeenCalled()
    await user.click(screen.getByRole('checkbox', { name: 'Open' }))
    vi.mocked(api.put).mockRejectedValue(new Error('offline'))
    await user.click(
      screen.getByRole('button', { name: 'Save automatic archive' })
    )
    expect(
      await screen.findByText('Could not save automatic archive settings.')
    ).toBeVisible()
    await act(async () => {
      await client.invalidateQueries({
        queryKey: ['ticket-auto-archive-config'],
      })
    })
    expect(screen.getByRole('checkbox', { name: 'Open' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Closed' })).not.toBeChecked()
  })
  it('只读管理员可看配置但不能编辑保存', async () => {
    mount(false)
    expect(await screen.findByLabelText('Inactive days')).toBeDisabled()
    expect(screen.getByRole('switch')).toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.getByRole('button', { name: 'Save automatic archive' })
    ).toBeDisabled()
  })
})
