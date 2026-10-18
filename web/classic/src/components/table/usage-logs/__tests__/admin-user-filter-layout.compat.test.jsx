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
import { cleanup, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, test, vi } from 'vitest';

import LogsFilters from '../UsageLogsFilters';

vi.mock('../../../../helpers', () => ({
  API: {
    get: vi.fn(() => Promise.resolve({ data: { success: true, data: [] } })),
  },
  showError: vi.fn(),
}));

afterEach(() => {
  cleanup();
});

function renderFilters(isAdminUser, formInitValues = {}) {
  let currentFormApi;
  function Harness() {
    const [formApi, setFormApi] = useState();
    currentFormApi = formApi;
    return (
      <LogsFilters
        formInitValues={formInitValues}
        formApi={formApi}
        setFormApi={setFormApi}
        refresh={vi.fn()}
        setShowColumnSelector={vi.fn()}
        setLogType={vi.fn()}
        loading={false}
        isAdminUser={isAdminUser}
        t={(key) => key}
      />
    );
  }
  render(<Harness />);
  return () => currentFormApi;
}

test('管理员用户名和用户 ID 模式各自显示对应 placeholder，筛选组保持末尾成组', () => {
  renderFilters(true, { userSearchType: 'username' });
  const group = screen.getByRole('group', { name: 'User filter' });
  const mode = within(group).getByRole('combobox', {
    name: 'User filter type',
  });
  expect(mode).toBeTruthy();
  expect(group.contains(screen.getByPlaceholderText('用户名'))).toBe(true);
  expect(group.querySelectorAll('.semi-select, input')).toHaveLength(2);
  expect(group.closest('.grid').lastElementChild).toBe(group);
  expect(mode.getAttribute('aria-labelledby')).toBe('userSearchType-label');

  cleanup();
  renderFilters(true, { userSearchType: 'id' });
  expect(screen.getByPlaceholderText('用户 ID')).toBeTruthy();
});

test('重置恢复用户名模式，普通用户不显示管理员用户筛选', async () => {
  const getFormApi = renderFilters(true, {
    userSearchType: 'username',
    username: '42',
  });
  const formApi = getFormApi();
  formApi.setValue('userSearchType', 'id');
  await userEvent.click(screen.getByRole('button', { name: '重置' }));
  expect(formApi.getValues().userSearchType).toBe('username');
  expect(screen.getByPlaceholderText('用户名')).toBeTruthy();

  cleanup();
  renderFilters(false);
  expect(screen.queryByPlaceholderText('用户名')).toBeNull();
  expect(screen.queryByPlaceholderText('用户 ID')).toBeNull();
  expect(screen.queryByRole('group', { name: 'User filter' })).toBeNull();
});
