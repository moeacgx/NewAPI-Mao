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
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test } from 'vitest';
import TokensTable from '../TokensTable';

// 显示名称与 code 不同；token.group 首项是改名前的历史 alias，靠 group_ids 关联当前分组。
const record = {
  id: 41,
  name: 'multi-group-token',
  status: 1,
  key: 'demo***789',
  group: 'legacy_a,stable_b,deleted_group',
  group_ids: [11, 22, 33],
  group_details: [
    { id: 22, code: 'stable_b', name: 'Premium Display' },
    { id: 11, code: 'stable_a', name: 'Standard Display' },
  ],
  unlimited_quota: true,
  used_quota: 0,
  created_time: 1,
  expired_time: -1,
};

test('多分组折叠为首个显示名+倍率+N，悬停/点击/键盘展开完整顺序并可关闭', async () => {
  localStorage.setItem('quota_per_unit', '500000');
  const user = userEvent.setup();
  render(
    <TokensTable
      tokens={[record]}
      selectedKeys={[]}
      setSelectedKeys={() => {}}
      visibleColumns={[]}
      groupRatios={{ stable_a: 1.5, stable_b: 2 }}
      showKeys={{}}
      resolvedTokenKeys={{}}
      loadingTokenKeys={{}}
      copyTokenKey={() => {}}
      t={(key) => key}
    />,
  );
  const cell = screen.getByRole('button', {
    name: '分组: Standard Display, Premium Display, deleted_group',
  });
  expect(within(cell).getByText('Standard Display')).toBeTruthy();
  expect(within(cell).getByText('1.5x')).toBeTruthy();
  expect(within(cell).getByText('+2')).toBeTruthy();
  expect(within(cell).queryByText('Premium Display')).toBeNull();
  expect(screen.queryByText(/legacy_a|stable_a|stable_b/)).toBeNull();
  const list = () => document.querySelector('.tokens-group-list');

  await user.hover(cell);
  await waitFor(() => expect(list()).not.toBeNull());
  expect([...list().children].map((item) => item.textContent)).toEqual([
    '1. Standard Display',
    '2. Premium Display',
    '3. deleted_group',
  ]);
  await user.unhover(cell);
  await waitFor(() => expect(cell.getAttribute('aria-expanded')).toBe('false'));

  // 点击（触屏同路径）固定展开，移开不收起，再次点击收起
  await user.click(cell);
  await user.unhover(cell);
  expect(cell.getAttribute('aria-expanded')).toBe('true');
  await user.click(cell);
  await waitFor(() => expect(cell.getAttribute('aria-expanded')).toBe('false'));

  // 键盘：Enter 打开，Escape 关闭
  cell.focus();
  await user.keyboard('{Enter}');
  await waitFor(() => expect(cell.getAttribute('aria-expanded')).toBe('true'));
  await user.keyboard('{Escape}');
  await waitFor(() => expect(cell.getAttribute('aria-expanded')).toBe('false'));
});
