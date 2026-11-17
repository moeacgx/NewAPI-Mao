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
import UserInfoHeader from '../UserInfoHeader';

const user = {
  id: 7,
  username: 'fixture-user',
  group: 'legacy-code',
  quota: 0,
  used_quota: 0,
  request_count: 0,
};
const t = (key) => key;

test('展示当前名称并在刷新后更新', () => {
  const view = render(
    <UserInfoHeader
      t={t}
      userState={{ user: { ...user, group_name: '当前分组' } }}
    />,
  );
  expect(screen.getAllByText('当前分组')).toHaveLength(2);
  expect(screen.queryByText('legacy-code')).toBeNull();
  view.rerender(
    <UserInfoHeader
      t={t}
      userState={{ user: { ...user, group_name: '改名后分组' } }}
    />,
  );
  expect(screen.getAllByText('改名后分组')).toHaveLength(2);
  expect(screen.queryByText('当前分组')).toBeNull();
  expect(user.group).toBe('legacy-code');
});

test.each([undefined, '', '   '])(
  '旧响应或空名称%j回退原标识',
  (group_name) => {
    render(
      <UserInfoHeader t={t} userState={{ user: { ...user, group_name } }} />,
    );
    expect(screen.getAllByText('legacy-code')).toHaveLength(2);
  },
);
