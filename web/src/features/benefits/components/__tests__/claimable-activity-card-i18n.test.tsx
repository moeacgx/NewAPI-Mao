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
import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'

import zhTWResource from '@/i18n/locales/zh-TW.json'
import zhCNResource from '@/i18n/locales/zh.json'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyConfig,
} from '@/stores/system-config-store'

import type { BenefitActivityUserView } from '../../types'
import { ClaimableActivityCard } from '../claimable-activity-card'

// src/test-setup.ts boots the shared test i18next singleton with an EMPTY
// `en` resource bundle (so `t(key)` simply echoes `key` back, which is what
// every English-language test in this suite asserts against). That singleton
// is the exact same `i18next` module instance `@/i18n/config` initializes in
// the real app, so switching `i18n.language` alone does not load real
// translations here — we explicitly register the real zh/zh-TW resource
// bundles below. Rendering through them (not an English-only `t = (k) => k`
// mock) means a regression like the zh-TW duplicate-JSON-key bug that once
// silently swapped in the wrong translation would fail these assertions.
beforeAll(() => {
  i18next.addResourceBundle(
    'zhCN',
    'translation',
    zhCNResource.translation,
    true,
    true
  )
  i18next.addResourceBundle(
    'zhTW',
    'translation',
    zhTWResource.translation,
    true,
    true
  )
})

function activity(
  overrides: Partial<BenefitActivityUserView> = {}
): BenefitActivityUserView {
  return {
    id: 1,
    name: 'Weekend Boost',
    description: '',
    group_id: 7,
    group_code_snapshot: 'default',
    group_name_snapshot: 'Default',
    status: 'published',
    amount_mode: 'fixed',
    amount_display_type: 'USD',
    total_amount: 10,
    total_quota: 5000000,
    fixed_quota: 500000,
    min_quota: 0,
    max_quota: 0,
    total_count: 10,
    fixed_amount: 1,
    min_amount: 0,
    max_amount: 0,
    claim_paid_threshold: 0,
    personal_valid_hours: 24,
    starts_at: 1,
    ends_at: 9999999999,
    published_at: 1,
    eligible: true,
    has_claimed: false,
    single_user_concurrency_limit: 1,
    remaining_count: 5,
    ...overrides,
  }
}

function setCurrencyConfig(overrides: Partial<CurrencyConfig>) {
  useSystemConfigStore.getState().setConfig({
    currency: { ...DEFAULT_CURRENCY_CONFIG, ...overrides },
  })
}

function renderCard(props: Partial<BenefitActivityUserView> = {}) {
  return render(
    <ClaimableActivityCard
      activity={activity(props)}
      onClaim={() => {}}
      claiming={false}
    />
  )
}

afterEach(async () => {
  setCurrencyConfig(DEFAULT_CURRENCY_CONFIG)
  await i18next.changeLanguage('en')
})

describe('claim eligibility labels render real zh/zh-TW translations', () => {
  const cases: Array<{
    locale: 'zhCN' | 'zhTW'
    reason: string
    text: string
  }> = [
    { locale: 'zhCN', reason: 'ineligible', text: '不符合领取条件' },
    { locale: 'zhCN', reason: 'sold_out', text: '活动券已领完' },
    { locale: 'zhCN', reason: 'inactive', text: '活动当前不可领取' },
    { locale: 'zhCN', reason: 'not_started', text: '活动尚未开始' },
    { locale: 'zhCN', reason: 'ended', text: '活动已结束' },
    { locale: 'zhTW', reason: 'ineligible', text: '不符合領取條件' },
    { locale: 'zhTW', reason: 'sold_out', text: '活動券已領完' },
    { locale: 'zhTW', reason: 'inactive', text: '活動目前不可領取' },
    { locale: 'zhTW', reason: 'not_started', text: '活動尚未開始' },
    { locale: 'zhTW', reason: 'ended', text: '活動已結束' },
  ]

  it.each(cases)(
    '$locale renders "$text" for reason=$reason',
    async ({ locale, reason, text }) => {
      await i18next.changeLanguage(locale)
      renderCard({ eligible: false, eligibility_reason: reason })

      expect(screen.getByText(text)).toBeTruthy()
    }
  )

  it('zh-CN renders the positive eligible badge distinctly from "not eligible"', async () => {
    await i18next.changeLanguage('zhCN')
    renderCard({ eligible: true })

    expect(screen.getByText('符合领取条件')).toBeTruthy()
    expect(screen.queryByText('不符合领取条件')).toBeNull()
  })

  it('zh-TW renders the exact 30-minute registration-age claim condition', async () => {
    await i18next.changeLanguage('zhTW')
    renderCard()

    expect(screen.getByText('新註冊帳號需滿 30 分鐘後才能領取')).toBeTruthy()
  })
})

describe('claim threshold amount follows the type it arrived with, not a separately-cached global setting', () => {
  it.each([
    { displayType: 'USD' as const, expected: '$10.00' },
    { displayType: 'CNY' as const, expected: '¥10.00' },
    { displayType: 'TOKENS' as const, expected: '10 Tokens' },
  ])(
    'formats $displayType from the activity payload',
    async ({ displayType, expected }) => {
      // This client's own system-config store is deliberately set to a
      // DIFFERENT type than what the activity payload carries, so a
      // symbol/unit match here proves the card reads
      // `activity.amount_display_type` (the type the backend actually
      // computed `claim_paid_threshold` with for THIS response), not this
      // separately-fetched global store — which can only ever be *usually*
      // in sync with it, never guaranteed.
      setCurrencyConfig({
        quotaDisplayType: 'CUSTOM',
        customCurrencySymbol: '€',
      })
      renderCard({
        claim_paid_threshold: 10,
        amount_display_type: displayType,
        eligible: false,
        eligibility_reason: 'ineligible',
      })

      expect(
        screen.getByText(new RegExp(expected.replace('$', '\\$')))
      ).toBeTruthy()
    }
  )

  it('formats CUSTOM using the current global custom symbol (no backend response ever carries a symbol string)', () => {
    setCurrencyConfig({ quotaDisplayType: 'CUSTOM', customCurrencySymbol: '€' })
    renderCard({
      claim_paid_threshold: 10,
      amount_display_type: 'CUSTOM',
      eligible: false,
      eligibility_reason: 'ineligible',
    })

    expect(screen.getByText(/€10\.00/)).toBeTruthy()
  })

  it("never double-converts: a CNY-typed amount is shown as-is, not multiplied again by this client store's exchange rate", () => {
    setCurrencyConfig({
      quotaDisplayType: 'CNY',
      usdExchangeRate: 7,
    })
    renderCard({
      claim_paid_threshold: 100,
      amount_display_type: 'CNY',
      eligible: false,
      eligibility_reason: 'ineligible',
    })

    expect(screen.getByText(/¥100\.00/)).toBeTruthy()
    expect(screen.queryByText(/¥700/)).toBeNull()
  })
})
