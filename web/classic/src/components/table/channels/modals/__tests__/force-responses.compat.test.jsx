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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import EditChannelModal from '../EditChannelModal';
import { API } from '../../../../../helpers/api';

for (const [enabled, allowed, switchType] of [
  [false, true],
  [true, true],
  [true, false],
  [true, true, true],
]) {
  test(`强制 Responses 初值 ${enabled} 权限 ${allowed} 切换类型 ${Boolean(switchType)} 时正确保存`, async () => {
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
          name: 'Responses channel',
          type: 1,
          models: 'gpt-4o',
          group: 'default',
          key: '',
          model_mapping: '',
          status_code_mapping: '',
          setting: JSON.stringify({
            force_responses: enabled,
            proxy: enabled ? '' : 'http://proxy.example:8080',
          }),
          other: '{}',
          channel_info: { is_multi_key: false },
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
      const toggle = await screen.findByRole('switch', {
        name: 'Force Responses upstream',
      });
      await waitFor(() => expect(toggle.disabled).toBe(!allowed));
      expect(toggle.getAttribute('aria-checked')).toBe(String(enabled));
      const user = userEvent.setup();
      if (switchType) {
        await user.click(
          document.querySelector('[id="type"]').closest('.semi-select'),
        );
        await user.click(await screen.findByText('Anthropic Claude'));
        // jsdom 不执行 CSS 退场动画，补发结束事件使 Semi 提交选择值。
        document
          .querySelectorAll('[class*="animation-hide"]')
          .forEach((node) => fireEvent.animationEnd(node));
        await waitFor(() =>
          expect(
            document.querySelector('[id="type"]').closest('.semi-select')
              .textContent,
          ).toContain('Anthropic Claude'),
        );
        await waitFor(() =>
          expect(
            screen.queryByRole('switch', { name: 'Force Responses upstream' }),
          ).toBeNull(),
        );
        await user.click(
          document.querySelector('[id="type"]').closest('.semi-select'),
        );
        await user.click(
          await screen.findByText('OpenAI', { ignore: 'script, style, title' }),
        );
        document
          .querySelectorAll('[class*="animation-hide"]')
          .forEach((node) => fireEvent.animationEnd(node));
        await waitFor(() =>
          expect(
            screen
              .getByRole('switch', { name: 'Force Responses upstream' })
              .getAttribute('aria-checked'),
          ).toBe('false'),
        );
        return;
      }
      if (allowed) await user.click(toggle);
      await user.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() => expect(payload).toBeTruthy());
      if (allowed)
        expect(JSON.parse(payload.setting).force_responses).toBe(!enabled);
      else expect(payload.setting).toBeUndefined();
      expect(payload.force_responses).toBeUndefined();
    } finally {
      API.defaults.adapter = adapter;
    }
  });
}
