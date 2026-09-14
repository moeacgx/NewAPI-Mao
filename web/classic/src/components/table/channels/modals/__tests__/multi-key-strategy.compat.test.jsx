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
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/react';
import EditChannelModal from '../EditChannelModal';
import { API } from '../../../../../helpers/api';

for (const allowed of [false, true]) {
  test(`敏感写权限${allowed}控制已有聚合渠道策略编辑`, async () => {
    const adapter = API.defaults.adapter;
    let payload;
    API.defaults.adapter = async (config) => {
      let data = [];
      if (config.method === 'put') payload = JSON.parse(config.data);
      if (config.url === '/api/user/self')
        data = {
          permissions: {
            admin_permissions: { channel: { sensitive_write: allowed } },
          },
        };
      if (config.url === '/api/channel/42')
        data = {
          id: 42,
          name: '聚合',
          type: 1,
          models: 'gpt-4o',
          group: 'default',
          model_mapping: '',
          status_code_mapping: '',
          key: '',
          setting: '{}',
          other: '{}',
          channel_info: { is_multi_key: true, multi_key_mode: 'polling' },
        };
      return {
        data: { success: true, data },
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      };
    };
    try {
      render(
        <EditChannelModal
          visible
          editingChannel={{ id: 42 }}
          handleClose={() => {}}
          refresh={() => {}}
        />,
      );
      await screen.findAllByText('密钥聚合模式');
      await waitFor(() => {
        const input = document.querySelector('[id="multi_key_mode"]');
        expect(input).toBeTruthy();
        expect(
          input.closest('.semi-select').getAttribute('aria-disabled'),
        ).toBe(String(!allowed));
      });
      await userEvent
        .setup()
        .click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() => expect(payload).toBeTruthy());
      expect(payload.key).toBeUndefined();
      expect(payload.key_mode).toBeUndefined();
      expect(payload.multi_key_mode).toBe(allowed ? 'polling' : undefined);
    } finally {
      API.defaults.adapter = adapter;
    }
  });
}
