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
import { expect, test } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UserQuotaCell from '../UserQuotaCell';
test('负余额抵消已用额度时仍能用键盘打开完整额度详情', async () => {
  localStorage.setItem('quota_display_type', 'TOKENS');
  localStorage.setItem('quota_per_unit', '500000');
  render(
    <UserQuotaCell record={{ quota: -123456789, used_quota: 123456789 }} />,
  );
  const user = userEvent.setup();
  await user.tab();
  expect(document.activeElement).toBe(
    screen.getByRole('button', { name: '额度详情' }),
  );
  await user.keyboard('{Enter}');
  expect(await screen.findByText('剩余额度 (Tokens): -123456789')).toBeTruthy();
  expect(screen.getByText('已用额度 (Tokens): 123456789')).toBeTruthy();
  expect(screen.getByText('总额度 (Tokens): 0')).toBeTruthy();
});
