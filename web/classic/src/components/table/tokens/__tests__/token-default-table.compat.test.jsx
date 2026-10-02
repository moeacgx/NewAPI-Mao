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

import React, { useState } from 'react';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';
import TokensTable from '../TokensTable';

const records = [
  {
    id: 31,
    name: 'Codex-Enterprise | 企业级 稳定不限客户端',
    status: 1,
    key: 'demo***123',
    group: 'internal-codex',
    group_details: [{ id: 9, code: 'internal-codex', name: '企业正式分组' }],
    unlimited_quota: true,
    used_quota: 100,
    created_time: 1,
    expired_time: -1,
  },
  {
    id: 32,
    name: 'Grok-Official | Heavy 顶级订阅号',
    status: 2,
    key: 'demo***456',
    group: 'internal-codex',
    group_details: [{ id: 9, code: 'internal-codex', name: '企业正式分组' }],
    unlimited_quota: true,
    used_quota: 200,
    created_time: 1,
    expired_time: -1,
  },
];
function Fixture({ onSelection, onCopy, initialSelection = [] }) {
  const [selectedKeys, setSelectedKeys] = useState(initialSelection);
  return (
    <TokensTable
      tokens={records}
      selectedKeys={selectedKeys}
      setSelectedKeys={(next) => {
        setSelectedKeys(next);
        onSelection(next.map((token) => token.id));
      }}
      visibleColumns={['model_limits', 'allow_ips']}
      groupRatios={{ 'internal-codex': 1 }}
      showKeys={{}}
      resolvedTokenKeys={{}}
      loadingTokenKeys={{}}
      copyTokenKey={onCopy}
      t={(key) => key}
    />
  );
}
test('Default 原生列表保留显示名称、稳定选择与按需复制', async () => {
  localStorage.setItem('quota_per_unit', '500000');
  const select = vi.fn(),
    copy = vi.fn();
  render(<Fixture onSelection={select} onCopy={copy} />);
  expect(screen.getByText(records[0].name)).toBeTruthy();
  expect(screen.getAllByText('企业正式分组')).toHaveLength(2);
  expect(screen.queryByText('internal-codex')).toBeNull();
  expect(copy).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole('checkbox', { name: '全选' }));
  expect(select).toHaveBeenLastCalledWith([31, 32]);
  await userEvent.click(
    screen.getByRole('checkbox', { name: '选择 ' + records[0].name }),
  );
  expect(select).toHaveBeenLastCalledWith([32]);
  const row = screen.getByText(records[0].name).closest('tr');
  await userEvent.click(within(row).getByRole('button', { name: '复制密钥' }));
  expect(copy).toHaveBeenCalledWith(records[0]);
  expect(screen.queryByRole('columnheader', { name: '时间' })).toBeNull();
});

test('上一页选满100条后在新页选择只提交当前页ID', async () => {
  localStorage.setItem('quota_per_unit', '500000');
  const select = vi.fn();
  render(
    <Fixture
      onSelection={select}
      onCopy={vi.fn()}
      initialSelection={Array.from({ length: 100 }, (_, index) => ({
        id: 1000 + index,
      }))}
    />,
  );
  await userEvent.click(
    screen.getByRole('checkbox', { name: '选择 ' + records[0].name }),
  );
  expect(select).toHaveBeenLastCalledWith([31]);
});
