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
import { act, renderHook, waitFor } from '@testing-library/react'
import { AxiosError, AxiosHeaders } from 'axios'
import { toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/http-client'

import { useBillingHistory } from '../hooks/use-billing-history'
import { usePayment } from '../hooks/use-payment'

vi.mock('@/hooks/use-admin', () => ({ useIsAdmin: () => false }))

const originalAdapter = api.defaults.adapter

beforeEach(() => {
  api.defaults.adapter = async (config) => {
    if (config.method === 'get') {
      return {
        status: 200,
        statusText: 'OK',
        config,
        headers: new AxiosHeaders(),
        data: { success: true, data: { items: [], total: 0 } },
      }
    }
    throw new AxiosError(
      'Request failed',
      'ERR_BAD_REQUEST',
      config,
      undefined,
      {
        status: 429,
        statusText: 'Too Many Requests',
        config,
        headers: new AxiosHeaders({ 'Retry-After': '138' }),
        data: '',
      }
    )
  }
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
})

describe('payment HTTP 429 feedback', () => {
  test('first payment displays one retry delay through the real Axios interceptor', async () => {
    const errorToast = vi.spyOn(toast, 'error').mockReturnValue('toast')
    const hook = renderHook(() => usePayment())

    await act(async () => {
      expect(await hook.result.current.processPayment(10, 'alipay')).toBe(false)
    })

    expect(errorToast).toHaveBeenCalledExactlyOnceWith(
      'Payment requests are too frequent. Please try again in 138 seconds.'
    )
    expect(hook.result.current.processing).toBe(false)
  })

  test('retry payment displays one retry delay and releases its loading state', async () => {
    const errorToast = vi.spyOn(toast, 'error').mockReturnValue('toast')
    const hook = renderHook(() => useBillingHistory())
    await waitFor(() => expect(hook.result.current.loading).toBe(false))

    await act(async () => {
      expect(await hook.result.current.handleRetryPayment('test-order')).toBe(
        false
      )
    })

    expect(errorToast).toHaveBeenCalledExactlyOnceWith(
      'Payment requests are too frequent. Please try again in 138 seconds.'
    )
    expect(hook.result.current.retryingTradeNo).toBeNull()
  })
})
