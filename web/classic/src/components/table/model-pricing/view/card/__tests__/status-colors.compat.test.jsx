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
import { render, screen } from '@testing-library/react';
import { beforeEach, expect, test, vi } from 'vitest';
import ModelPerformanceBadge from '../ModelPerformanceBadge';

const endTs = Date.UTC(2026, 8, 21) / 1000;
const translate = (key) => key;
const colorCases = [
  [100, '#10b981', 'rgb(16, 185, 129)'],
  [99.99, '#34d399', 'rgb(52, 211, 153)'],
  [95, '#34d399', 'rgb(52, 211, 153)'],
  [90, '#34d399', 'rgb(52, 211, 153)'],
  [89.99, '#f59e0b', 'rgb(245, 158, 11)'],
  [70, '#f59e0b', 'rgb(245, 158, 11)'],
  [69.99, '#ef4444', 'rgb(239, 68, 68)'],
  [0, '#ef4444', 'rgb(239, 68, 68)'],
];

beforeEach(() => {
  vi.spyOn(Date, 'now').mockReturnValue(endTs * 1000);
});

test.each(colorCases)(
  '状态率为 %s 时，三个历史段使用官方色阶',
  (rate, hex, rgb) => {
    render(
      <ModelPerformanceBadge
        t={translate}
        performance={{
          status_rate: rate,
          success_rate: 12,
          avg_latency_ms: 1900,
          avg_tps: 16.9,
          series: [20, 12, 4].map((hoursAgo) => ({
            ts: endTs - hoursAgo * 3600,
            status_rate: rate,
            success_rate: 12,
          })),
        }}
      />,
    );

    const signal = screen.getByRole('img', { name: `${rate.toFixed(2)}%` });
    const bars = [...signal.querySelectorAll('[style]')];
    expect(bars.map((bar) => bar.style.backgroundColor)).toEqual([
      rgb,
      rgb,
      rgb,
    ]);
    expect(bars.map((bar) => bar.style.height)).toEqual([
      '8px',
      '10px',
      '12px',
    ]);
    expect(screen.getByText('1.90s')).toBeTruthy();
    expect(screen.getByText('16.9t')).toBeTruthy();
  },
);

test.each(colorCases)(
  '无历史序列且状态率为 %s 时，摘要柱沿用官方色阶',
  (rate, hex) => {
    render(
      <ModelPerformanceBadge
        t={translate}
        performance={{ status_rate: rate, success_rate: 12, series: [] }}
      />,
    );

    const signal = screen.getByRole('img', { name: `${rate.toFixed(1)}%` });
    expect(signal.children).toHaveLength(3);
    expect(
      signal.parentElement.style.getPropertyValue(
        '--classic-pricing-performance-status-color',
      ),
    ).toBe(hex);
  },
);

test('缺少状态率时回退成功率，24 小时内没有流量的时间段保持灰色', () => {
  render(
    <ModelPerformanceBadge
      t={translate}
      performance={{
        success_rate: 98,
        series: [{ ts: endTs - 3600, success_rate: 98 }],
      }}
    />,
  );

  const signal = screen.getByRole('img', { name: '98.00%' });
  expect(
    [...signal.querySelectorAll('[style]')].map(
      (bar) => bar.style.backgroundColor,
    ),
  ).toEqual([
    'var(--semi-color-fill-1)',
    'var(--semi-color-fill-1)',
    'rgb(52, 211, 153)',
  ]);
});
