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
import UserInfoModal from '../UserInfoModal';

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
    <UserInfoModal
      t={t}
      showUserInfo
      setShowUserInfoModal={() => {}}
      userInfoData={{ ...user, group_name: '当前分组' }}
    />,
  );
  expect(screen.getByText('当前分组')).toBeTruthy();
  expect(screen.queryByText('legacy-code')).toBeNull();
  view.rerender(
    <UserInfoModal
      t={t}
      showUserInfo
      setShowUserInfoModal={() => {}}
      userInfoData={{ ...user, id: 8, group_name: '另一用户分组' }}
    />,
  );
  expect(screen.getByText('另一用户分组')).toBeTruthy();
  expect(screen.queryByText('当前分组')).toBeNull();
});

test.each([undefined, '', '   '])(
  '旧响应或空名称%j回退原标识',
  (group_name) => {
    render(
      <UserInfoModal
        t={t}
        showUserInfo
        setShowUserInfoModal={() => {}}
        userInfoData={{ ...user, group_name }}
      />,
    );
    expect(screen.getByText('legacy-code')).toBeTruthy();
  },
);
