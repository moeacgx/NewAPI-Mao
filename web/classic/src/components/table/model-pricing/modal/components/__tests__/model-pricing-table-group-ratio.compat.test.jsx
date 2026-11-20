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
import React from 'react';
import { render } from '@testing-library/react';
import { expect, test } from 'vitest';
import ModelPricingTable from '../ModelPricingTable';

const groups = ['vip_code', 'free_code', 'std_code', 'unset_code'];

const groupNames = {
  vip_code: 'VIP 显示名',
  free_code: '免费显示名',
  std_code: '标准显示名',
  unset_code: '未配置显示名',
};

// unset_code 没有配置倍率，应按实际定价回退为 1。
const groupRatio = { vip_code: 2.4, free_code: 0, std_code: 1 };

const usableGroup = Object.fromEntries(
  groups.map((group) => [group, { desc: group, ratio: 1 }]),
);

const expectedRatios = [
  ['VIP 显示名', '2.4x'],
  ['免费显示名', '0x'],
  ['标准显示名', '1x'],
  ['未配置显示名', '1x'],
];

const renderTable = (modelData, priceRate = 1) =>
  render(
    <ModelPricingTable
      modelData={{ enable_groups: groups, ...modelData }}
      groupRatio={groupRatio}
      groupNames={groupNames}
      currency='USD'
      siteDisplayType='USD'
      tokenUnit='M'
      displayPrice={(usd) => `$${usd}`}
      priceRate={priceRate}
      usdExchangeRate={1}
      usableGroup={usableGroup}
      t={(key, options) =>
        options?.discount ? key.replace('{{discount}}', options.discount) : key
      }
    />,
  );

const getGroupRows = (container) =>
  new Map(
    Array.from(
      container.querySelectorAll(
        '.classic-pricing-detail-group-table-wrap tbody tr',
      ),
    ).map((row) => {
      const cells = Array.from(row.querySelectorAll('td'));
      return [
        cells[0].querySelector('.classic-pricing-detail-group-link')
          .textContent,
        cells.map((cell) => cell.textContent),
      ];
    }),
  );

test.each([
  [
    '按 Token',
    { model_name: 'token', quota_type: 0, model_ratio: 1, completion_ratio: 1 },
  ],
  ['按次', { model_name: 'fixed', quota_type: 1, model_price: 0.1 }],
])('Classic %s 分组定价在独立倍率列显示 0、1、2.4 与缺失回退 1', (_, model) => {
  const { container } = renderTable(model);
  const headers = Array.from(
    container.querySelectorAll('.classic-pricing-detail-group-table-wrap th'),
  ).map((node) => node.textContent);
  expect(headers[1]).toBe('倍率');

  const rows = getGroupRows(container);
  for (const [name, ratio] of expectedRatios) {
    expect(rows.get(name)?.[1]).toBe(ratio);
  }
  expect(container.textContent).not.toContain('vip_code');
});

test('Classic 充值折扣徽标与倍率列相互独立', () => {
  const { container } = renderTable(
    { model_name: 'token', quota_type: 0, model_ratio: 1, completion_ratio: 1 },
    0.5,
  );
  const rows = getGroupRows(container);
  expect(rows.get('标准显示名')[0]).toContain('5折');
  expect(rows.get('标准显示名')[1]).toBe('1x');
  expect(rows.get('VIP 显示名')[0]).not.toContain('折');
  expect(rows.get('VIP 显示名')[1]).toBe('2.4x');
});

test('Classic 可解析表达式卡片标题右端显示倍率，不可解析表达式不生成卡片', () => {
  const { container } = renderTable({
    model_name: 'dynamic',
    quota_type: 0,
    billing_mode: 'tiered_expr',
    billing_expr:
      'len <= 272000 ? tier("standard", p * 0.5 + c * 3) : tier("long_context", p * 1 + c * 6)',
  });
  const cards = new Map(
    Array.from(
      container.querySelectorAll('.classic-pricing-detail-group-card-header'),
    ).map((header) => [
      header.querySelector('.classic-pricing-detail-group-link').textContent,
      header.querySelector('.classic-pricing-detail-group-card-ratio')
        .textContent,
    ]),
  );
  expect([...cards.entries()]).toEqual(expectedRatios);

  const invalid = renderTable({
    model_name: 'invalid',
    quota_type: 0,
    billing_mode: 'tiered_expr',
    billing_expr: 'p * max(1, 2)',
  });
  expect(
    invalid.container.querySelector('.classic-pricing-detail-group-card'),
  ).toBeNull();
  expect(
    invalid.container.querySelector('.classic-pricing-detail-special-expr'),
  ).not.toBeNull();
});
