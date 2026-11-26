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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import EditTokenModal from '../modals/EditTokenModal';
import TokensTable from '../TokensTable';
import { API } from '../../../../helpers/api';
import { StatusContext } from '../../../../context/Status';

// quota_per_unit = 500000：每周 500 美元 = 250,000,000，总额 10000 美元 = 5,000,000,000。
const WEEKLY_500 = 250000000;
const TOTAL_10000 = 5000000000;

beforeEach(() => {
  localStorage.setItem('quota_per_unit', '500000');
});

afterEach(() => {
  vi.restoreAllMocks();
});

function mockTokenApi(token) {
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (url === '/api/token/auto-groups')
      return { data: { success: true, data: { groups: [], max_count: 0 } } };
    if (url === '/api/user/self/groups')
      return {
        data: {
          success: true,
          data: {
            base: { id: 11, code: 'base', name: '标准显示名称', ratio: 1 },
          },
        },
      };
    if (url === '/api/token/7')
      return {
        data: {
          success: true,
          data: {
            id: 7,
            name: '周期额度令牌',
            expired_time: -1,
            model_limits: '',
            group: 'base',
            group_ids: [11],
            group_mode: 'explicit',
            group_ratio_limits: '{}',
            ...token,
          },
        },
      };
    return { data: { success: true, data: [] } };
  });
  return vi.spyOn(API, 'put').mockResolvedValue({ data: { success: true } });
}

function renderEditModal() {
  return render(
    <StatusContext.Provider value={[{ status: {} }, () => {}]}>
      <EditTokenModal
        visiable
        editingToken={{ id: 7 }}
        refresh={() => {}}
        handleClose={() => {}}
      />
    </StatusContext.Provider>,
  );
}

function periodAmountInput() {
  return document.querySelector('input[id*="quota_period_amount"]');
}

test('每周 500 美元加总额 10000 美元：编辑回显并按原值提交，不回传账本字段', async () => {
  const save = mockTokenApi({
    unlimited_quota: false,
    remain_quota: TOTAL_10000,
    quota_period: 'weekly',
    quota_period_limit: WEEKLY_500,
    quota_period_used: 1234,
    quota_period_reset_at: 1893456000,
  });
  renderEditModal();
  await screen.findByDisplayValue('周期额度令牌');

  await waitFor(() => expect(periodAmountInput()?.value).toBe('500.000000'));
  expect(screen.getAllByText('每周').length).toBeGreaterThan(0);
  expect(screen.getByDisplayValue('10000.000000')).toBeTruthy();

  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() => expect(save).toHaveBeenCalled());
  const [url, payload] = save.mock.calls[0];
  expect(url).toBe('/api/token/');
  expect(payload).toMatchObject({
    id: 7,
    unlimited_quota: false,
    remain_quota: TOTAL_10000,
    quota_period: 'weekly',
    quota_period_limit: WEEKLY_500,
  });
  expect(payload).not.toHaveProperty('quota_period_amount');
  expect(payload).not.toHaveProperty('quota_period_used');
  expect(payload).not.toHaveProperty('quota_period_reset_at');
});

test('总额度不限时仍保留并提交周期额度', async () => {
  const save = mockTokenApi({
    unlimited_quota: true,
    remain_quota: 0,
    quota_period: 'daily',
    quota_period_limit: WEEKLY_500,
  });
  renderEditModal();
  await screen.findByDisplayValue('周期额度令牌');
  await waitFor(() => expect(periodAmountInput()?.value).toBe('500.000000'));

  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save.mock.calls[0][1]).toMatchObject({
    unlimited_quota: true,
    remain_quota: 0,
    quota_period: 'daily',
    quota_period_limit: WEEKLY_500,
  });
});

test('未设置周期的令牌提交 none 与 0，不阻塞保存', async () => {
  const save = mockTokenApi({ unlimited_quota: true, remain_quota: 0 });
  renderEditModal();
  await screen.findByDisplayValue('周期额度令牌');

  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save.mock.calls[0][1]).toMatchObject({
    quota_period: 'none',
    quota_period_limit: 0,
  });
});

test('启用周期但周期额度为 0 时拒绝提交', async () => {
  const save = mockTokenApi({
    unlimited_quota: true,
    remain_quota: 0,
    quota_period: 'weekly',
    quota_period_limit: 0,
  });
  renderEditModal();
  await screen.findByDisplayValue('周期额度令牌');

  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() =>
    expect(screen.getByText('错误：周期额度必须大于 0')).toBeTruthy(),
  );
  expect(save).not.toHaveBeenCalled();
});

test('列表同时展示总额度状态与周期额度已用/上限', () => {
  const records = [
    {
      id: 41,
      name: '周期令牌',
      status: 1,
      key: 'demo***123',
      group: 'base',
      unlimited_quota: true,
      used_quota: 100,
      quota_period: 'weekly',
      quota_period_limit: WEEKLY_500,
      quota_period_used: 500000,
      created_time: 1,
      expired_time: -1,
    },
    {
      id: 42,
      name: '无周期令牌',
      status: 1,
      key: 'demo***456',
      group: 'base',
      unlimited_quota: true,
      used_quota: 0,
      quota_period: 'none',
      quota_period_limit: 0,
      created_time: 1,
      expired_time: -1,
    },
  ];
  render(
    <TokensTable
      tokens={records}
      selectedKeys={[]}
      setSelectedKeys={() => {}}
      visibleColumns={['model_limits', 'allow_ips']}
      groupRatios={{ base: 1 }}
      showKeys={{}}
      resolvedTokenKeys={{}}
      loadingTokenKeys={{}}
      copyTokenKey={() => {}}
      t={(key) => key}
    />,
  );

  const summaries = screen.getAllByTestId('period-quota-summary');
  expect(summaries).toHaveLength(1);
  expect(summaries[0].textContent).toContain('周期额度');
  expect(summaries[0].textContent).toContain('每周');
  expect(summaries[0].textContent).toContain('$1.00');
  expect(summaries[0].textContent).toContain('$500.00');
  // 总额度不限时，额度列显示「无限制」而不是周期额度数字。
  const periodRow = screen.getByText('周期令牌').closest('tr');
  expect(periodRow.textContent).toContain('无限制');
  expect(periodRow.textContent).not.toContain('$10000');
});
