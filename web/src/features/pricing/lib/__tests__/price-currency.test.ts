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
import { afterEach, beforeEach, describe, expect, test } from 'vitest'

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import type { PricingModel } from '../../types'
import {
  formatPrice,
  formatGroupPrice,
  formatFixedPriceDisplay,
  formatRequestPriceDisplay,
  formatVariantRulePrice,
} from '../price'

const groupRatio = { free: 0, paid: 2 }
const model: PricingModel = {
  id: 1,
  model_name: 'test-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 2,
  model_price: 0.4,
  enable_groups: ['free', 'paid'],
  group_ratio: groupRatio,
}
const originalConfig = useSystemConfigStore.getState().config

beforeEach(() => {
  useSystemConfigStore.getState().setConfig({
    currency: { ...DEFAULT_CURRENCY_CONFIG, quotaDisplayType: 'TOKENS' },
  })
})
afterEach(() => {
  useSystemConfigStore.getState().setConfig(originalConfig)
  localStorage.clear()
})

describe('价格货币与本地倍率契约', () => {
  test('Tokens 额度模式下按量价格和指定分组价格仍显示美元', () => {
    expect(formatPrice(model, 'input', 'M', false, 1, 1, 'paid')).toBe('$4')
    expect(
      formatGroupPrice(model, 'paid', 'output', 'M', false, 1, 1, groupRatio)
    ).toBe('$8')
    expect(formatPrice(model, 'input', 'M', false, 1, 1, 'free')).toBe('$0')
  })
  test('Tokens 额度模式下保留规格路线价格区间和分组倍率', () => {
    const fixed: PricingModel = {
      ...model,
      quota_type: 1,
      model_price_variants: {
        resolution_enabled: true,
        quality_enabled: false,
        rules: [{ resolution: '1080p', price: 0.7 }],
      },
      model_route_price_variants: {
        'image.edit': {
          resolution_enabled: false,
          quality_enabled: true,
          rules: [{ quality: 'high', price: 0.2 }],
        },
      },
    }
    expect(
      formatFixedPriceDisplay(fixed, 'paid', false, 1, 1, groupRatio)
    ).toEqual({
      formatted: '$0.4',
      formattedMaximum: '$1.4',
      hasVariants: true,
    })
    expect(formatRequestPriceDisplay(fixed, false, 1, 1, 'free')).toEqual({
      formatted: '$0',
      formattedMaximum: '$0',
      hasVariants: true,
    })
    expect(formatVariantRulePrice(0.2, false, 1, 1, 2)).toBe('$0.4')
  })
  test('人民币价格保留充值汇率与分组折扣', () => {
    useSystemConfigStore.getState().setConfig({
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        quotaDisplayType: 'CNY',
        usdExchangeRate: 7,
      },
    })
    expect(formatPrice(model, 'input', 'M', true, 4, 7, 'paid')).toBe('¥16')
    expect(formatVariantRulePrice(0.2, true, 4, 7, 2)).toBe('¥1.6')
  })
})
