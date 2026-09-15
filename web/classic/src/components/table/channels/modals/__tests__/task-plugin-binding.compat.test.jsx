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
import { afterEach, expect, test } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import EditChannelModal from '../EditChannelModal';
import TaskPluginSelect from '../TaskPluginSelect';
import { API } from '../../../../../helpers/api';
import { CHANNEL_OPTIONS } from '../../../../../constants/channel.constants';

const original = API.defaults.adapter;
afterEach(() => {
  API.defaults.adapter = original;
});
for (const type of [61, 62]) {
  test(
    '编辑渠道 ' + type + ' 保存原类型和正确的 setting 绑定，并保留其他设置',
    async () => {
      let payload;
      API.defaults.adapter = async (config) => {
        let data = [];
        if (config.method === 'put') payload = JSON.parse(config.data);
        if (config.url === '/api/user/self')
          data = {
            permissions: {
              admin_permissions: { channel: { sensitive_write: true } },
            },
          };
        if (config.url === '/api/channel/42')
          data = {
            id: 42,
            name: '绑定测试',
            type,
            models: 'existing-model',
            group: 'default',
            key: '',
            setting: JSON.stringify({
              task_plugin_key: 'sunoapi',
              proxy: 'http://proxy.test',
              http_protocol: 'http1',
            }),
            settings: JSON.stringify({ allow_service_tier: false }),
            model_mapping: '',
            status_code_mapping: '',
            other: '{}',
          };
        if (config.url === '/api/task_plugin_options')
          data = [
            {
              key: 'sunoapi',
              name: 'Suno',
              channel_type: 62,
              models: ['suno'],
            },
          ];
        return {
          data: { success: true, data },
          status: 200,
          statusText: 'OK',
          headers: {},
          config,
        };
      };
      render(
        <EditChannelModal
          visible
          editingChannel={{ id: 42 }}
          handleClose={() => {}}
          refresh={() => {}}
        />,
      );
      await screen.findByDisplayValue('绑定测试');
      if (type === 62) await screen.findByText('Suno (sunoapi)');
      await userEvent.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() => expect(payload).toBeTruthy());
      expect(payload.type).toBe(type);
      expect(JSON.parse(payload.setting).task_plugin_key).toBe(
        type === 62 ? 'sunoapi' : undefined,
      );
      expect(JSON.parse(payload.setting).proxy).toBe('http://proxy.test');
      expect(JSON.parse(payload.settings).task_plugin_key).toBeUndefined();
      expect(payload.task_plugin_key).toBeUndefined();
      expect(payload.models).toBe('existing-model');
      expect(CHANNEL_OPTIONS.find((item) => item.value === 61).label).toBe(
        'AtlasCloud',
      );
    },
  );
}

test('无敏感写权限时保留已有插件绑定，不请求选项或触发修改', async () => {
  let calls = 0;
  API.defaults.adapter = async () => {
    calls++;
    throw new Error('Unexpected API');
  };
  render(
    <TaskPluginSelect
      disabled
      value='sunoapi'
      onChange={() => {
        throw new Error('Unexpected change');
      }}
    />,
  );
  expect(screen.getByText('sunoapi')).toBeTruthy();
  expect(screen.getByRole('combobox').getAttribute('aria-disabled')).toBe(
    'true',
  );
  expect(calls).toBe(0);
});

test('选项失败保留绑定并显示重试，成功后可选择真实插件', async () => {
  let fail = true;
  let selected;
  API.defaults.adapter = async (config) => {
    if (fail) throw Object.assign(new Error('unavailable'), { config });
    return {
      data: {
        success: true,
        data: [
          { key: 'google', name: 'Google', channel_type: 62, models: ['veo'] },
        ],
      },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
  render(
    <TaskPluginSelect
      value='sunoapi'
      onChange={(value, models) => {
        selected = { value, models };
      }}
    />,
  );
  await screen.findByRole('alert');
  expect(screen.getByText('sunoapi')).toBeTruthy();
  fail = false;
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
  await waitFor(() => expect(screen.queryByRole('alert')).toBeNull());
  await waitFor(() =>
    expect(screen.getByRole('combobox').getAttribute('aria-disabled')).toBe(
      'false',
    ),
  );
  await userEvent.click(screen.getByRole('combobox'));
  await userEvent.click(await screen.findByRole('option', { name: /Google/ }));
  // jsdom 不运行动画；模拟浏览器的退场动画结束，使 Semi 提交受控选择值。
  document
    .querySelectorAll('[class*="animation-hide"]')
    .forEach((node) => fireEvent.animationEnd(node));
  await waitFor(() =>
    expect(selected).toEqual({ value: 'google', models: ['veo'] }),
  );
});
