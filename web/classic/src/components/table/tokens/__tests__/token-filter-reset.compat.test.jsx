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
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';
import TokensFilters from '../TokensFilters';

test('状态与名称筛选后重置会清空全部条件并重新查询第一页', async () => {
  let formApi;
  const search = vi.fn();
  render(
    <TokensFilters
      formInitValues={{ searchKeyword: '', searchToken: '', searchStatus: 0 }}
      setFormApi={(api) => {
        formApi = api;
      }}
      searchTokens={search}
      t={(key) => key}
    />,
  );
  await userEvent.type(
    screen.getByRole('textbox', { name: 'Filter by name...' }),
    'Codex',
  );
  await userEvent.click(screen.getByRole('button', { name: '状态' }));
  await userEvent.click(screen.getByRole('radio', { name: '已禁用' }));
  expect(formApi.getValues().searchStatus).toBe(2);
  await userEvent.click(screen.getByRole('button', { name: '重置' }));
  await waitFor(() =>
    expect(formApi.getValues()).toMatchObject({
      searchKeyword: '',
      searchToken: '',
      searchStatus: 0,
    }),
  );
  expect(screen.queryByRole('button', { name: '重置' })).toBeNull();
  await waitFor(() => expect(search).toHaveBeenLastCalledWith(1));
});
