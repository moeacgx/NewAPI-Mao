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
import { AxiosError, AxiosHeaders } from 'axios';
import { Toast } from '@douyinfe/semi-ui';
import { afterEach, expect, it, vi } from 'vitest';
import { API } from '../../../helpers/api';
import * as utils from '../../../helpers/utils';
import TopupHistoryModal from '../modals/TopupHistoryModal';

const originalAdapter = API.defaults.adapter;

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

it('shows one waiting message when retry payment receives HTTP 429', async () => {
  const globalToast = vi.spyOn(utils, 'showError').mockImplementation(() => {});
  const paymentToast = vi.spyOn(Toast, 'error').mockImplementation(() => {});
  API.defaults.adapter = async (config) => {
    if (config.method === 'get') {
      return {
        status: 200,
        statusText: 'OK',
        config,
        headers: new AxiosHeaders(),
        data: {
          success: true,
          data: {
            items: [
              {
                id: 1,
                trade_no: 'test-order',
                status: 'pending',
                payment_method: 'alipay',
                money: 10,
                amount: 10,
              },
            ],
            total: 1,
          },
        },
      };
    }
    throw new AxiosError(
      'Request failed',
      'ERR_BAD_REQUEST',
      config,
      undefined,
      {
        status: 429,
        statusText: 'Too Many Requests',
        config,
        headers: new AxiosHeaders({ 'Retry-After': '138' }),
        data: '',
      },
    );
  };

  render(
    <TopupHistoryModal
      visible
      onCancel={() => {}}
      t={(key, values) => key.replace('{{seconds}}', values?.seconds ?? '')}
    />,
  );
  fireEvent.click(await screen.findByRole('button', { name: '重新支付' }));

  await waitFor(() => expect(paymentToast).toHaveBeenCalledOnce());
  expect(paymentToast).toHaveBeenCalledWith({
    content: '请求过于频繁，请在 138 秒后重试',
  });
  expect(globalToast).not.toHaveBeenCalled();
});
