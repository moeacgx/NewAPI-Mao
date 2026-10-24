import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState, type ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { TicketAutoReplySettings } from '../ticket-auto-reply-settings'

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), put: vi.fn() },
}))

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
  useAuthStore.getState().auth.setUser({ id: 7, username: 'admin', role: 100 })
  vi.mocked(api.get).mockResolvedValue(
    response({ enabled: true, content: '收到，我们会尽快处理。' })
  )
  vi.mocked(api.put).mockResolvedValue(
    response({ enabled: true, content: '收到，我们会尽快处理。' })
  )
})

afterEach(() => {
  useAuthStore.getState().auth.reset()
  vi.resetAllMocks()
})

describe('Default 工单自动回复配置', () => {
  it('后台刷新不覆盖尚未保存的回复草稿', async () => {
    const user = userEvent.setup()
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <TicketAutoReplySettings canWrite />
      </QueryClientProvider>
    )
    const input = await screen.findByLabelText('Automatic reply message')
    await user.clear(input)
    await user.type(input, 'My unsaved draft')
    vi.mocked(api.get).mockResolvedValue(
      response({ enabled: false, content: 'Changed remotely' })
    )
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['ticket-auto-reply-config'] })
    })
    expect(input).toHaveValue('My unsaved draft')
    expect(screen.getByRole('switch')).toBeChecked()
  })

  it('换账号后读取独立缓存，旧账号保存响应不覆盖新草稿', async () => {
    const user = userEvent.setup()
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    let finish!: (value: ReturnType<typeof response>) => void
    const pending = new Promise<ReturnType<typeof response>>((resolve) => {
      finish = resolve
    })
    vi.mocked(api.put).mockReturnValueOnce(pending)
    render(
      <QueryClientProvider client={client}>
        <TicketAutoReplySettings canWrite />
      </QueryClientProvider>
    )
    await user.clear(await screen.findByLabelText('Automatic reply message'))
    await user.type(
      screen.getByLabelText('Automatic reply message'),
      'Account A draft'
    )
    await user.click(
      screen.getByRole('button', { name: 'Save automatic reply' })
    )
    vi.mocked(api.get).mockResolvedValue(
      response({ enabled: false, content: 'Account B config' })
    )
    act(() =>
      useAuthStore
        .getState()
        .auth.setUser({ id: 8, username: 'second-admin', role: 100 })
    )
    await waitFor(() =>
      expect(screen.getByLabelText('Automatic reply message')).toHaveValue(
        'Account B config'
      )
    )
    await act(async () => {
      finish(response({ enabled: true, content: 'Account A saved' }))
      await pending
    })
    expect(screen.getByLabelText('Automatic reply message')).toHaveValue(
      'Account B config'
    )
    expect(client.getQueryData(['ticket-auto-reply-config', 8])).toEqual({
      enabled: false,
      content: 'Account B config',
    })
  })
  it('读取配置并按只读权限禁用编辑控件', async () => {
    render(<TicketAutoReplySettings canWrite={false} />, { wrapper: Wrapper })

    expect(await screen.findByLabelText('Automatic reply message')).toHaveValue(
      '收到，我们会尽快处理。'
    )
    expect(screen.getByRole('switch')).toHaveAttribute('aria-disabled', 'true')
    expect(screen.getByLabelText('Automatic reply message')).toBeDisabled()
    expect(
      screen.getByRole('button', { name: 'Save automatic reply' })
    ).toBeDisabled()
    expect(api.get).toHaveBeenCalledWith(
      '/api/ticket-admin/automation/auto-reply',
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
  })

  it('启用时拒绝空内容且不提交请求', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get).mockResolvedValue(
      response({ enabled: false, content: '' })
    )
    render(<TicketAutoReplySettings canWrite />, { wrapper: Wrapper })

    const message = await screen.findByLabelText('Automatic reply message')
    await user.clear(message)
    await user.click(screen.getByRole('switch'))
    await user.click(
      screen.getByRole('button', { name: 'Save automatic reply' })
    )

    expect(
      await screen.findByText('Enter an automatic reply message when enabled.')
    ).toBeVisible()
    expect(api.put).not.toHaveBeenCalled()
  })

  it('保存启用状态和回复内容并刷新已保存值', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get).mockResolvedValue(
      response({ enabled: false, content: '' })
    )
    render(<TicketAutoReplySettings canWrite />, { wrapper: Wrapper })

    const message = await screen.findByLabelText('Automatic reply message')
    await user.clear(message)
    await user.type(message, '您好，工单已收到。')
    await user.click(screen.getByRole('switch'))
    await user.click(
      screen.getByRole('button', { name: 'Save automatic reply' })
    )

    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        '/api/ticket-admin/automation/auto-reply',
        { enabled: true, content: '您好，工单已收到。' }
      )
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Save automatic reply' })
      ).toBeDisabled()
    )
  })
})
