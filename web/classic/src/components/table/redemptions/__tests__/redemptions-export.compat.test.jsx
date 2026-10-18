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
import { expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import RedemptionExportModal from '../modals/RedemptionExportModal';
test('导出默认不下载，取消不产生下载；显式Markdown下载生成文件', async () => {
  const clicks = vi
    .spyOn(HTMLAnchorElement.prototype, 'click')
    .mockImplementation(() => {});
  const close = vi.fn();
  const { unmount } = render(
    <RedemptionExportModal
      data={{ codes: ['code-a'], name: '活动', quota: 500000 }}
      onClose={close}
    />,
  );
  expect(clicks).not.toHaveBeenCalled();
  await userEvent.setup().click(screen.getByRole('button', { name: '取消' }));
  expect(clicks).not.toHaveBeenCalled();
  expect(close).toHaveBeenCalled();
  unmount();
  render(
    <RedemptionExportModal
      data={{ codes: ['code-a'], name: '活动', quota: 500000 }}
      onClose={close}
    />,
  );
  const user = userEvent.setup();
  await user.click(screen.getByRole('radio', { name: 'Markdown' }));
  await user.click(screen.getByRole('button', { name: '下载' }));
  expect(clicks).toHaveBeenCalledOnce();
  expect(clicks.mock.instances[0].download).toBe('活动.md');
});
