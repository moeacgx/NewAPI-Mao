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
import userEvent from '@testing-library/user-event';
import i18n from 'i18next';
import { expect, test, vi } from 'vitest';
import TokensPagination from '../TokensPagination';

test('分页首前后末和页容量调用原有回调并传递数字', async () => {
  const changePage = vi.fn();
  const changePageSize = vi.fn();
  const user = userEvent.setup();
  render(
    <TokensPagination
      tokenCount={100}
      pageSize={10}
      activePage={5}
      handlePageChange={changePage}
      handlePageSizeChange={changePageSize}
      t={i18n.t.bind(i18n)}
    />,
  );

  for (const [label, page] of [
    ['Go to first page', 1],
    ['Go to previous page', 4],
    ['Go to next page', 6],
    ['Go to last page', 10],
    ['Go to page 5', 5],
  ]) {
    await user.click(screen.getByRole('button', { name: label, exact: true }));
    expect(changePage).toHaveBeenLastCalledWith(page);
  }

  await user.selectOptions(
    screen.getByRole('combobox', { name: 'Rows per page' }),
    '50',
  );
  expect(changePageSize).toHaveBeenCalledTimes(1);
  expect(changePageSize).toHaveBeenCalledWith(50);
});

test.each([
  [40, 2, ['1', '2', '3', '4'], 0],
  [100, 2, ['1', '2', '10'], 1],
  [100, 3, ['1', '3', '10'], 2],
  [100, 8, ['1', '8', '10'], 2],
  [100, 9, ['1', '9', '10'], 1],
  [0, 1, ['1'], 0],
])(
  '共 %i 条第 %i 页时显示正确页码与省略边界',
  (tokenCount, activePage, expectedPages, expectedGaps) => {
    render(
      <TokensPagination
        tokenCount={tokenCount}
        pageSize={10}
        activePage={activePage}
        handlePageChange={vi.fn()}
        handlePageSizeChange={vi.fn()}
        t={i18n.t.bind(i18n)}
      />,
    );

    expect(
      screen
        .getAllByRole('button', { name: /^Go to page / })
        .map((button) => button.textContent),
    ).toEqual(expectedPages);
    expect(screen.queryAllByText('...', { exact: true })).toHaveLength(
      expectedGaps,
    );
    expect(screen.getByRole('button', { current: 'page' }).textContent).toBe(
      String(activePage),
    );
  },
);

test.each([
  [1, ['Go to first page', 'Go to previous page']],
  [10, ['Go to next page', 'Go to last page']],
])('第 %i 页禁止越界导航', async (activePage, disabledLabels) => {
  const changePage = vi.fn();
  const user = userEvent.setup();
  render(
    <TokensPagination
      tokenCount={100}
      pageSize={10}
      activePage={activePage}
      handlePageChange={changePage}
      handlePageSizeChange={vi.fn()}
      t={i18n.t.bind(i18n)}
    />,
  );

  for (const label of disabledLabels) {
    const button = screen.getByRole('button', { name: label });
    expect(button.disabled).toBe(true);
    await user.click(button);
  }
  expect(changePage).not.toHaveBeenCalled();
});
