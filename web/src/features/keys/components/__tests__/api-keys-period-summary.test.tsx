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
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { PeriodQuotaSummary, UnlimitedQuotaBadge } =
  await import('../api-keys-cells')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

// Default display is USD with 500,000 quota units per USD.
const USD = 500_000

const apiKey = {
  id: 1,
  name: 'k',
  key: 'sk-test',
  status: 1,
  remain_quota: 0,
  used_quota: 0,
  unlimited_quota: true,
  quota_period: 'none',
  quota_period_limit: 0,
  quota_period_used: 0,
  quota_period_reset_at: 0,
  expired_time: -1,
  created_time: 1,
  accessed_time: 0,
  group: 'default',
  group_ids: [],
  group_details: [],
  cross_group_retry: false,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
}

function renderSummary(overrides: Record<string, unknown>) {
  const key = { ...apiKey, ...overrides } as never
  return render(
    <I18nextProvider i18n={i18n}>
      <UnlimitedQuotaBadge used={0} />
      <PeriodQuotaSummary apiKey={key} />
    </I18nextProvider>
  )
}

afterEach(() => cleanup())

describe('token list period quota summary', () => {
  test('总额度不限时同时显示不限徽标与每日周期额度', () => {
    renderSummary({
      quota_period: 'daily',
      quota_period_limit: 500 * USD,
      quota_period_used: 120 * USD,
    })
    expect(screen.getByText('Unlimited')).toBeTruthy()
    const summary = screen.getByTestId('period-quota-summary')
    expect(summary.textContent).toContain('Daily')
    expect(summary.textContent).toContain('120')
    expect(summary.textContent).toContain('500')
  })

  test('每周周期额度显示每周标签而不是内部值', () => {
    renderSummary({
      quota_period: 'weekly',
      quota_period_limit: 500 * USD,
    })
    const text = screen.getByTestId('period-quota-summary').textContent
    expect(text).toContain('Weekly')
    expect(text).not.toContain('weekly')
  })

  test('不设周期或周期额度为 0 时不显示周期行', () => {
    renderSummary({ quota_period: 'none', quota_period_limit: 0 })
    expect(screen.queryByTestId('period-quota-summary')).toBeNull()
    cleanup()
    renderSummary({ quota_period: 'daily', quota_period_limit: 0 })
    expect(screen.queryByTestId('period-quota-summary')).toBeNull()
  })
})
