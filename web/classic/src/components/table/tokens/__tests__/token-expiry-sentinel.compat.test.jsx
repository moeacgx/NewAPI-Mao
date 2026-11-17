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
import { afterEach, expect, test, vi } from 'vitest';
import EditTokenModal from '../modals/EditTokenModal';
import { API } from '../../../../helpers/api';
import { StatusContext } from '../../../../context/Status';

afterEach(() => {
  vi.restoreAllMocks();
});

function mockTokenApi(expiredTime) {
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
            name: '过期哨兵令牌',
            expired_time: expiredTime,
            unlimited_quota: true,
            remain_quota: 0,
            model_limits: '',
            group: 'base',
            group_ids: [11],
            group_mode: 'explicit',
            group_ratio_limits: '{}',
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

function expiryInput() {
  return document.querySelector('.semi-datepicker input');
}

test('编辑永不过期令牌时过期字段显示永不过期而不是 1970 日期，并按 -1 提交', async () => {
  const save = mockTokenApi(-1);
  renderEditModal();
  await screen.findByDisplayValue('过期哨兵令牌');
  await waitFor(() => expect(expiryInput()).toBeTruthy());

  expect(expiryInput().value).toBe('');
  expect(expiryInput().value).not.toMatch(/1970/);
  expect(expiryInput().getAttribute('placeholder')).toBe('永不过期');

  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() =>
    expect(save).toHaveBeenCalledWith(
      '/api/token/',
      expect.objectContaining({ id: 7, expired_time: -1 }),
    ),
  );
});

test('编辑有限过期令牌时保留日期显示并按原秒级时间戳提交', async () => {
  const expiredTime = Math.floor(Date.now() / 1000) + 30 * 86400;
  const save = mockTokenApi(expiredTime);
  renderEditModal();
  await screen.findByDisplayValue('过期哨兵令牌');
  await waitFor(() =>
    expect(expiryInput()?.value).toMatch(/^\d{4}-\d{2}-\d{2}/),
  );

  expect(expiryInput().value).not.toMatch(/1970/);

  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() =>
    expect(save).toHaveBeenCalledWith(
      '/api/token/',
      expect.objectContaining({ id: 7, expired_time: expiredTime }),
    ),
  );
});

test('点击永不过期快捷按钮会清空日期显示并按 -1 提交', async () => {
  const expiredTime = Math.floor(Date.now() / 1000) + 30 * 86400;
  const save = mockTokenApi(expiredTime);
  renderEditModal();
  await screen.findByDisplayValue('过期哨兵令牌');
  await waitFor(() => expect(expiryInput()?.value).toMatch(/^\d{4}-/));

  fireEvent.click(screen.getByRole('button', { name: '永不过期' }));
  await waitFor(() => expect(expiryInput().value).toBe(''));

  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() =>
    expect(save).toHaveBeenCalledWith(
      '/api/token/',
      expect.objectContaining({ id: 7, expired_time: -1 }),
    ),
  );
});
