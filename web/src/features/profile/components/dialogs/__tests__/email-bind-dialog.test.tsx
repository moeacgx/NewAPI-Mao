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
import { act, cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { InternalAxiosRequestConfig } from 'axios'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { useProfile } from '../../../hooks/use-profile'
import { EmailBindDialog } from '../email-bind-dialog'

interface VerificationWidget {
  callback: (token: string) => void
  'expired-callback': () => void
  'error-callback': () => void
}

const originalAdapter = api.defaults.adapter
const originalAuth = useAuthStore.getState().auth
let client: QueryClient
let widgets: VerificationWidget[]
let requests: InternalAxiosRequestConfig[]
let verificationResult: 'success' | 'rejected' | 'network-error'
let boundEmail: string

function EmailBindingFlow() {
  const [open, setOpen] = useState(true)
  const { profile, refreshProfile } = useProfile()

  return (
    <>
      <button type='button' onClick={() => setOpen(true)}>
        Open email binding
      </button>
      <output aria-label='Account identity'>{profile?.id}</output>
      <output aria-label='Bound email'>{profile?.email}</output>
      <EmailBindDialog
        open={open}
        onOpenChange={setOpen}
        currentEmail={profile?.email}
        onSuccess={refreshProfile}
      />
    </>
  )
}

function renderEmailBinding(enabled = true) {
  client.setQueryData(['status'], {
    turnstile_check: enabled,
    turnstile_site_key: enabled ? 'test-site-key' : '',
  })
  return render(
    <QueryClientProvider client={client}>
      <EmailBindingFlow />
    </QueryClientProvider>
  )
}

async function completeHumanCheck(token = 'verified-token') {
  await waitFor(() => expect(widgets.length).toBeGreaterThan(0))
  act(() => widgets.at(-1)?.callback(token))
}

beforeEach(() => {
  localStorage.clear()
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  widgets = []
  requests = []
  verificationResult = 'success'
  boundEmail = ''
  window.turnstile = {
    render: (_element, options) => {
      widgets.push(options as unknown as VerificationWidget)
    },
  }
  useAuthStore.setState({
    auth: {
      ...originalAuth,
      user: { id: 42, username: 'existing-user', role: 1 },
      accessToken: 'existing-session-token',
      accessExpiresAt: 4_102_444_800,
      session: {
        sid: 'existing-session',
        current: true,
        login_method: 'password',
        ip: '',
        user_agent: '',
        created_at: 1,
        last_active_at: 1,
        expires_at: 4_102_444_800,
      },
    },
  })
  api.defaults.adapter = async (config) => {
    requests.push(config)
    const path = new URL(config.url ?? '', 'https://example.test').pathname
    let data: unknown = { success: true }
    if (path === '/api/user/self') {
      data = {
        success: true,
        data: { id: 42, username: 'existing-user', email: boundEmail },
      }
    } else if (path === '/api/verification') {
      if (verificationResult === 'network-error') throw new Error('offline')
      data = { success: verificationResult === 'success' }
    } else if (path === '/api/oauth/email/bind') {
      boundEmail = JSON.parse(config.data).email
    } else {
      throw new Error(`Unexpected request: ${config.method} ${path}`)
    }
    return { data, status: 200, statusText: 'OK', headers: {}, config }
  }
})

afterEach(() => {
  cleanup()
  client.clear()
  api.defaults.adapter = originalAdapter
  useAuthStore.setState({ auth: originalAuth })
  delete window.turnstile
  localStorage.clear()
  vi.useRealTimers()
})

describe('邮箱绑定验证码的人机验证', () => {
  test('开启人机验证且未取得 token 时不能发送验证码', async () => {
    renderEmailBinding()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Email Address'), 'user@example.test')

    expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
    expect(
      requests.every((request) => !request.url?.startsWith('/api/verification'))
    ).toBe(true)
  })

  test('验证成功后发送编码后的邮箱及 token，冷却结束不能复用已消费 token', async () => {
    renderEmailBinding()
    const user = userEvent.setup()
    await user.type(
      screen.getByLabelText('Email Address'),
      'user+alias@example.test'
    )
    await completeHumanCheck('token-with-+&')
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    await user.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() =>
      expect(screen.getByRole('button', { name: '60s' })).toBeDisabled()
    )
    const verification = requests.filter((request) =>
      request.url?.startsWith('/api/verification')
    )
    expect(verification).toHaveLength(1)
    const url = new URL(verification[0].url ?? '', 'https://example.test')
    expect(url.searchParams.get('email')).toBe('user+alias@example.test')
    expect(url.searchParams.get('turnstile')).toBe('token-with-+&')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(60_000)
    })
    vi.useRealTimers()
    expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
    await completeHumanCheck('next-token')
    expect(screen.getByRole('button', { name: 'Send' })).toBeEnabled()
  })

  test.each(['expired-callback', 'error-callback'] as const)(
    '%s 清除旧 token，重新验证后才能发送',
    async (callback) => {
      renderEmailBinding()
      const user = userEvent.setup()
      await user.type(
        screen.getByLabelText('Email Address'),
        'user@example.test'
      )
      await completeHumanCheck()
      expect(screen.getByRole('button', { name: 'Send' })).toBeEnabled()

      act(() => widgets.at(-1)?.[callback]())
      expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
      await completeHumanCheck('renewed-token')
      await user.click(screen.getByRole('button', { name: 'Send' }))
      await waitFor(() =>
        expect(screen.getByRole('button', { name: '60s' })).toBeDisabled()
      )
      expect(
        new URL(
          requests.at(-1)?.url ?? '',
          'https://example.test'
        ).searchParams.get('turnstile')
      ).toBe('renewed-token')
    }
  )

  test.each(['rejected', 'network-error'] as const)(
    '发送发生 %s 后重新验证可以重试',
    async (result) => {
      verificationResult = result
      renderEmailBinding()
      const user = userEvent.setup()
      await user.type(
        screen.getByLabelText('Email Address'),
        'user@example.test'
      )
      await completeHumanCheck('first-token')
      await user.click(screen.getByRole('button', { name: 'Send' }))
      await waitFor(() =>
        expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
      )
      expect(
        requests.filter((request) =>
          request.url?.startsWith('/api/verification')
        )
      ).toHaveLength(1)

      verificationResult = 'success'
      await completeHumanCheck('retry-token')
      await user.click(screen.getByRole('button', { name: 'Send' }))
      await waitFor(() =>
        expect(screen.getByRole('button', { name: '60s' })).toBeDisabled()
      )
      const verification = requests.filter((request) =>
        request.url?.startsWith('/api/verification')
      )
      expect(verification).toHaveLength(2)
      expect(
        new URL(
          verification[1].url ?? '',
          'https://example.test'
        ).searchParams.get('turnstile')
      ).toBe('retry-token')
    }
  )

  test('关闭再打开清空表单和旧 token，不能沿用前一次验证', async () => {
    renderEmailBinding()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Email Address'), 'user@example.test')
    await completeHumanCheck('old-token')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await user.click(screen.getByRole('button', { name: 'Open email binding' }))

    expect(screen.getByLabelText('Email Address')).toHaveValue('')
    await user.type(screen.getByLabelText('Email Address'), 'next@example.test')
    expect(screen.getByRole('button', { name: 'Send' })).toBeDisabled()
    await completeHumanCheck('new-token')
    expect(screen.getByRole('button', { name: 'Send' })).toBeEnabled()
  })

  test('关闭人机验证时可以直接发送且不携带 token', async () => {
    renderEmailBinding(false)
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Email Address'), 'user@example.test')
    await user.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() =>
      expect(screen.getByRole('button', { name: '60s' })).toBeDisabled()
    )
    expect(widgets).toHaveLength(0)
    const verification = requests.find((request) =>
      request.url?.startsWith('/api/verification')
    )
    expect(
      new URL(verification?.url ?? '', 'https://example.test').searchParams.has(
        'turnstile'
      )
    ).toBe(false)
  })

  test('只提交邮箱验证码绑定并回读原账号资料，保留原登录身份', async () => {
    renderEmailBinding()
    const user = userEvent.setup()
    await user.type(
      screen.getByLabelText('Email Address'),
      'user+alias@example.test'
    )
    await user.type(screen.getByLabelText('Verification Code'), '123456')
    await user.click(screen.getByRole('button', { name: 'Bind Email' }))

    await waitFor(() =>
      expect(screen.getByLabelText('Bound email')).toHaveTextContent(
        'user+alias@example.test'
      )
    )
    expect(screen.getByLabelText('Account identity')).toHaveTextContent('42')
    const binding = requests.find(
      (request) => request.url === '/api/oauth/email/bind'
    )
    expect(binding?.method).toBe('post')
    expect(JSON.parse(binding?.data ?? '{}')).toEqual({
      email: 'user+alias@example.test',
      code: '123456',
    })
    expect(binding?.headers.Authorization).toBe('Bearer existing-session-token')
    expect(requests.map((request) => request.url)).toEqual([
      '/api/user/self',
      '/api/oauth/email/bind',
      '/api/user/self',
    ])
    const auth = useAuthStore.getState().auth
    expect(auth.user?.id).toBe(42)
    expect(auth.session?.sid).toBe('existing-session')
    expect(auth.accessToken).toBe('existing-session-token')
  })
})
