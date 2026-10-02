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

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, test, vi } from 'vitest';

import { API } from '../../../../helpers';
import TokenApiInfoPopover from '../TokenApiInfoPopover';

afterEach(() => vi.restoreAllMocks());

const renderPopover = () => render(<TokenApiInfoPopover t={(key) => key} />);

test('打开 API 信息入口后显示已配置地址', async () => {
  vi.spyOn(API, 'get').mockResolvedValue({
    data: {
      data: {
        api_info_enabled: true,
        api_info: [
          {
            route: 'OpenAI 兼容接口',
            url: 'https://api.example.test/v1',
            description: '用于兼容客户端接入',
          },
        ],
      },
    },
  });

  renderPopover();
  await userEvent.click(screen.getByRole('button', { name: 'API地址' }));

  expect(await screen.findByText('OpenAI 兼容接口')).toBeTruthy();
  expect(screen.getByText('https://api.example.test/v1')).toBeTruthy();
  expect(screen.getByText('用于兼容客户端接入')).toBeTruthy();
  expect(API.get).toHaveBeenCalledWith('/api/status');
});

test('API 信息未配置时显示空状态', async () => {
  vi.spyOn(API, 'get').mockResolvedValue({
    data: { data: { api_info_enabled: false, api_info: [] } },
  });

  renderPopover();
  await userEvent.click(screen.getByRole('button', { name: 'API地址' }));

  expect(await screen.findByText('暂无API信息')).toBeTruthy();
  expect(screen.getByText('请联系管理员在系统设置中配置API信息')).toBeTruthy();
});
