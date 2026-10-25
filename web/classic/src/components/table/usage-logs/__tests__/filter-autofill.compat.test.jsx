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
import LogsFilters from '../UsageLogsFilters';

// 分组筛选已改为按接口加载的下拉框（bffe91161）；与同目录测试使用相同的模拟，
// 避免 Classic 关闭隔离时模块模拟在文件间共享导致结果依赖执行顺序。
vi.mock('../../../../helpers', () => ({
  API: {
    get: vi.fn(() => Promise.resolve({ data: { success: true, data: [] } })),
  },
  showError: vi.fn(),
}));

test('日志筛选不创建密码输入，查询仍提交稳定分组标识', async () => {
  let api;
  const refresh = vi.fn();
  const { container } = render(
    <LogsFilters
      formInitValues={{ group: 'group-42' }}
      setFormApi={(value) => {
        api = value;
      }}
      refresh={refresh}
      setShowColumnSelector={() => {}}
      setLogType={() => {}}
      loading={false}
      isAdminUser
      t={(key) => key}
    />,
  );
  expect(container.querySelector('input[type=password]')).toBeNull();
  expect(container.querySelector('form').getAttribute('autocomplete')).toBe(
    'off',
  );
  await userEvent.setup().click(screen.getByRole('button', { name: '查询' }));
  expect(api.getValues().group).toBe('group-42');
});
