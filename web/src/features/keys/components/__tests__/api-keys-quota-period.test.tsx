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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { api } = await import('@/lib/api')
const { ApiKeysProvider } = await import('../api-keys-provider')
const { ApiKeysMutateDrawer } = await import('../api-keys-mutate-drawer')
const { useAuthStore } = await import('@/stores/auth-store')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

// Default display is USD with 500,000 quota units per USD.
const USD = 500_000

type ApiMethod = (url: string, data?: unknown) => Promise<{ data: unknown }>
type MockableApi = { get: ApiMethod; put: ApiMethod; post: ApiMethod }
const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPut = apiClient.put
const originalPost = apiClient.post
let queryClient: InstanceType<typeof QueryClient> | null = null

const storedToken = {
  id: 7,
  name: 'period-token',
  key: 'sk-test',
  status: 1,
  remain_quota: 10000 * USD,
  used_quota: 0,
  unlimited_quota: false,
  quota_period: 'weekly',
  quota_period_limit: 500 * USD,
  quota_period_used: 0,
  quota_period_reset_at: 0,
  expired_time: -1,
  created_time: 1,
  accessed_time: 0,
  group: 'default',
  group_mode: 'explicit',
  group_ids: [11],
  group_details: [
    { id: 11, code: 'default', name: 'Standard models', exclusive: false },
  ],
  auto_groups: null,
  cross_group_retry: false,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
}

function installFixtures(token: Record<string, unknown>, puts: unknown[]) {
  apiClient.get = async (url) => {
    switch (url) {
      case '/api/status':
        return { data: { data: { default_use_auto_group: false } } }
      case '/api/user/models':
        return { data: { success: true, data: [] } }
      case '/api/user/self/groups':
        return {
          data: {
            success: true,
            data: {
              default: {
                id: 11,
                name: 'Standard models',
                desc: 'Standard access',
                ratio: 1,
              },
            },
          },
        }
      case '/api/token/auto-groups':
        return {
          data: { success: true, data: { groups: ['default'], max_count: 0 } },
        }
      case '/api/token/7':
        return { data: { success: true, data: token } }
      default:
        throw new Error(`Unexpected GET ${url}`)
    }
  }
  apiClient.put = async (url, data) => {
    expect(url).toBe('/api/token/')
    puts.push(data)
    return { data: { success: true, data: {} } }
  }
}

async function renderEditDrawer(token: Record<string, unknown>) {
  useAuthStore.getState().auth.setUser({ id: 2, username: 'user', role: 1 })
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ApiKeysProvider>
          <ApiKeysMutateDrawer
            open
            onOpenChange={() => undefined}
            currentRow={token as never}
          />
        </ApiKeysProvider>
      </I18nextProvider>
    </QueryClientProvider>
  )
  await waitFor(
    () => {
      expect(getInputByLabel('Name').value).toBe('period-token')
    },
    { timeout: 3000 }
  )
}

function findLabel(startsWith: string): HTMLLabelElement {
  const label = [...document.querySelectorAll<HTMLLabelElement>('label')].find(
    (candidate) => candidate.textContent?.trim().startsWith(startsWith)
  )
  if (!label) throw new Error(`Expected label starting with "${startsWith}"`)
  return label
}

function getInputByLabel(startsWith: string): HTMLInputElement {
  const item = findLabel(startsWith).closest('[data-slot="form-item"]')
  const input = item?.querySelector<HTMLInputElement>('input')
  if (!input) throw new Error(`Expected input for "${startsWith}"`)
  return input
}

function getPeriodTrigger(): HTMLElement {
  const item = findLabel('Period quota cycle').closest(
    '[data-slot="form-item"]'
  )
  const trigger = item?.querySelector<HTMLElement>('[role="combobox"]')
  if (!trigger) throw new Error('Expected the period select trigger')
  return trigger
}

function findButton(text: string): HTMLButtonElement {
  const button = screen
    .queryAllByRole<HTMLButtonElement>('button')
    .find((candidate) => candidate.textContent?.includes(text))
  if (!button) throw new Error(`Expected button containing "${text}"`)
  return button
}

afterEach(() => {
  useAuthStore.getState().auth.setUser(null)
  apiClient.get = originalGet
  apiClient.put = originalPut
  apiClient.post = originalPost
  localStorage.clear()
  queryClient?.clear()
  queryClient = null
})

describe('API key drawer period quota and total quota', () => {
  test('重新打开编辑显示每周标签并回填 500 与 10000，保存提交内部额度单位', async () => {
    const puts: Array<Record<string, unknown>> = []
    installFixtures(storedToken, puts)
    await renderEditDrawer(storedToken)

    const trigger = getPeriodTrigger()
    expect(trigger.textContent).toContain('Weekly')
    expect(trigger.textContent).not.toContain('weekly')
    expect(getInputByLabel('Period quota (').value).toBe('500')
    expect(getInputByLabel('Total quota (').value).toBe('10000')

    fireEvent.click(findButton('Save changes'))
    await waitFor(() => expect(puts).toHaveLength(1))
    expect(puts[0]).toMatchObject({
      id: 7,
      quota_period: 'weekly',
      quota_period_limit: 500 * USD,
      remain_quota: 10000 * USD,
      unlimited_quota: false,
    })
  })

  test('总额度不限时仍显示并提交每日周期额度，不显示总额度输入', async () => {
    const puts: Array<Record<string, unknown>> = []
    const token = {
      ...storedToken,
      unlimited_quota: true,
      remain_quota: 0,
      quota_period: 'daily',
    }
    installFixtures(token, puts)
    await renderEditDrawer(token)

    expect(getPeriodTrigger().textContent).toContain('Daily')
    expect(getInputByLabel('Period quota (').value).toBe('500')
    expect(() => findLabel('Total quota (')).toThrow()

    fireEvent.click(findButton('Save changes'))
    await waitFor(() => expect(puts).toHaveLength(1))
    expect(puts[0]).toMatchObject({
      quota_period: 'daily',
      quota_period_limit: 500 * USD,
      unlimited_quota: true,
      remain_quota: 0,
    })
  })

  test('选择周期但周期额度为 0 时不发送请求并显示提示', async () => {
    const puts: Array<Record<string, unknown>> = []
    installFixtures(storedToken, puts)
    await renderEditDrawer(storedToken)

    fireEvent.input(getInputByLabel('Period quota ('), {
      target: { value: '0' },
    })
    fireEvent.click(findButton('Save changes'))
    await waitFor(() =>
      expect(
        document.body.textContent?.includes(
          'Period quota must be greater than 0'
        )
      ).toBe(true)
    )
    expect(puts).toHaveLength(0)
  })

  test('总额度超过上限时显示以当前货币单位表示的上限', async () => {
    const puts: Array<Record<string, unknown>> = []
    installFixtures(storedToken, puts)
    await renderEditDrawer(storedToken)

    fireEvent.input(getInputByLabel('Total quota ('), {
      target: { value: '1000000001' },
    })
    fireEvent.click(findButton('Save changes'))
    await waitFor(() =>
      expect(
        document.body.textContent?.includes('Quota cannot exceed 1,000,000,000')
      ).toBe(true)
    )
    expect(puts).toHaveLength(0)
  })
})
