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
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, expect, test, vi } from 'vitest';
import { API } from '../../../helpers';
import { AdminRoute } from '../../../helpers/auth';
import PaymentSetting from '../PaymentSetting';

beforeEach(() => {
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (url === '/api/option/') {
      return { data: { success: true, data: [] } };
    }
    return {
      data: {
        success: true,
        data: {
          items: url.endsWith('/withdrawals')
            ? [
                {
                  id: 17,
                  user_id: 7,
                  username: 'reviewer',
                  method: 'usdt',
                  quota: 500000,
                  display_amount: 1,
                  display_currency: 'USDT',
                  status: 'pending',
                  created_at: 1700000000,
                  payout_details: {
                    usdt_address: 'test-address',
                    usdt_chain: 'TRC20',
                  },
                },
              ]
            : [],
          total: url.endsWith('/withdrawals') ? 1 : 0,
          page: 1,
          page_size: 50,
        },
      },
    };
  });
  vi.spyOn(API, 'post').mockResolvedValue({
    data: { success: true, data: null },
  });
});

function renderPage(role) {
  localStorage.setItem(
    'user',
    JSON.stringify({ id: 7, role, username: 'reviewer' }),
  );
  return render(
    <MemoryRouter initialEntries={['/console/affiliate-admin']}>
      <Routes>
        <Route
          path='/console/affiliate-admin'
          element={
            <AdminRoute>
              <PaymentSetting defaultActiveKey='affiliate' onlyAffiliate />
            </AdminRoute>
          }
        />
        <Route path='/forbidden' element={<p>Forbidden</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

test('role 10 可查询返佣业务且不请求或展示 Root 配置', async () => {
  renderPage(10);
  await waitFor(() =>
    expect(API.get).toHaveBeenCalledWith(
      '/api/affiliate/admin/invitations',
      expect.any(Object),
    ),
  );
  expect(
    API.get.mock.calls.some(([url]) => url.startsWith('/api/option')),
  ).toBe(false);
  expect(screen.queryByText('返佣规则')).toBeNull();
  expect(screen.queryByText('反欺诈设置')).toBeNull();
  expect(screen.queryByText('需要确认合规声明')).toBeNull();
  fireEvent.click(screen.getByRole('tab', { name: '提现审核' }));
  fireEvent.click(await screen.findByRole('button', { name: '通过' }));
  expect(API.post).not.toHaveBeenCalled();
  fireEvent.click(await screen.findByRole('button', { name: 'confirm' }));
  await waitFor(() =>
    expect(API.post).toHaveBeenCalledWith(
      '/api/affiliate/admin/withdrawals/17/approve',
      { remark: '' },
    ),
  );
});

test('role 1 被管理员路由拒绝且不发起业务或配置请求', () => {
  renderPage(1);
  expect(screen.getByText('Forbidden')).toBeTruthy();
  expect(API.get).not.toHaveBeenCalled();
});

test('role 100 保留配置加载和未确认时的合规门禁', async () => {
  renderPage(100);
  await waitFor(() => expect(API.get).toHaveBeenCalledWith('/api/option/'));
  expect(await screen.findByText('需要确认合规声明')).toBeTruthy();
  expect(screen.getByRole('tab', { name: '返佣规则' })).toBeTruthy();
});
