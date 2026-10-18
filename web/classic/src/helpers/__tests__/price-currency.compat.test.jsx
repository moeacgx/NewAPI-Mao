/*
Copyright (C) 2025 QuantumNous

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

import { expect, test } from 'vitest';
import { calculateModelPrice, getModelPriceItems } from '../utils';
test('Tokens额度模式仍按货币展示，保留免费分组和充值折扣', () => {
  const record = {
    quota_type: 0,
    model_ratio: 2,
    completion_ratio: 3,
    enable_groups: ['g1'],
  };
  for (const ratio of [0, 0.5]) {
    const result = calculateModelPrice({
      record,
      selectedGroup: 'g1',
      groupRatio: { g1: ratio },
      tokenUnit: 'M',
      displayPrice: (price) => String(price * 0.5),
      currency: 'USD',
      quotaDisplayType: 'TOKENS',
    });
    expect(result.inputPrice).toBe(ratio === 0 ? '$0.0000' : '$1.0000');
    expect(getModelPriceItems(result, (key) => key, 'TOKENS')[0].key).toBe(
      'input',
    );
  }
});

test('Tokens额度模式的路线规格价格仍应用分组零倍率且保留按秒单位', () => {
  const result = calculateModelPrice({
    record: {
      quota_type: 1,
      model_price: 3,
      model_price_unit: 'second',
      enable_groups: ['g1'],
      model_price_variants: {
        resolution_enabled: true,
        quality_enabled: false,
        rules: [{ resolution: '1080p', price: 6 }],
      },
    },
    selectedGroup: 'g1',
    groupRatio: { g1: 0 },
    tokenUnit: 'M',
    displayPrice: (price) => `$${price.toFixed(4)}`,
    currency: 'USD',
    quotaDisplayType: 'TOKENS',
  });
  expect(result.price).toBe('$0.0000');
  expect(result.modelPriceUnit).toBe('second');
  expect(result.variantPrices[0].displayPrice).toBe('$0.0000');
  expect(result.variantPrices[0].resolution).toBe('1080p');
});
