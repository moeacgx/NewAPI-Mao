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
import { expect, test } from 'vitest';
import TokenTimestamp from '../TokenTimestamp';

test.each([0, -1, Number.MAX_SAFE_INTEGER, 9223372036854775807, Infinity, NaN])(
  '时间戳 %s 超出日期范围或为空时降级显示且不崩溃',
  (timestamp) => {
    const { container } = render(<TokenTimestamp timestamp={timestamp} />);
    expect(screen.getByText('-')).toBeTruthy();
    expect(container.querySelector('time')).toBeNull();
  },
);
test('时钟前进时从刚刚更新为相对时间且保留绝对时间', () => {
  const now = Date.parse('2026-10-02T08:00:00Z');
  const timestamp = (now - 30000) / 1000;
  const { rerender, container } = render(
    <TokenTimestamp timestamp={timestamp} now={now} />,
  );
  expect(screen.getByText('Just now')).toBeTruthy();
  rerender(<TokenTimestamp timestamp={timestamp} now={now + 60000} />);
  expect(screen.queryByText('Just now')).toBeNull();
  expect(container.querySelector('time').getAttribute('datetime')).toBe(
    '2026-10-02T07:59:30.000Z',
  );
  expect(container.querySelector('time').textContent).toBe(
    new Intl.RelativeTimeFormat('zh', { numeric: 'always' }).format(
      -1,
      'minute',
    ),
  );
});
