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
import { beforeEach, afterEach, expect, test } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TaskPlugins from '../index';
import PluginDetails from '../PluginDetails';
import { API } from '../../../helpers/api';

const adapter = API.defaults.adapter;
let calls, plugin, failDetail, denyList, denyWrite, runtime;
beforeEach(() => {
  calls = [];
  failDetail = denyList = denyWrite = false;
  runtime = {
    enabled: false,
    channel_type: 62,
    builtin_only: true,
    production_ready: false,
  };
  plugin = {
    key: 'sunoapi',
    version: '1.0',
    api_version: 1,
    source_kind: 'builtin',
    source_hash: 'hash',
    enabled: false,
    active: false,
    meta: { name: 'Suno', models: ['suno'] },
  };
  localStorage.setItem('user', JSON.stringify({ role: 100 }));
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ url: config.url, method: config.method, body });
    const list = config.url === '/api/plugin/task';
    const detail = config.url === '/api/plugin/task/sunoapi';
    if (
      (denyList && list) ||
      (failDetail && detail) ||
      (denyWrite && config.method !== 'get')
    ) {
      const error = new Error('Access denied');
      error.response = { status: 403 };
      error.config = config;
      throw error;
    }
    let data;
    if (list) data = [plugin];
    else if (config.url === '/api/plugin/task/runtime/status') {
      if (config.method === 'put') runtime = { ...runtime, ...body };
      data = runtime;
    } else if (detail)
      data = {
        ...plugin,
        ...(localStorage.getItem('user').includes('100')
          ? { source: 'export default {}' }
          : {}),
      };
    else if (config.url.endsWith('/versions')) data = [{ ...plugin }];
    else if (config.url.endsWith('/activate')) {
      plugin = { ...plugin, active: true, enabled: true };
      data = null;
    } else if (config.url.endsWith('/status')) {
      plugin = { ...plugin, enabled: body.enabled };
      data = null;
    } else throw new Error('Unexpected API ' + config.url);
    return {
      data: { success: true, data },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
});
afterEach(() => {
  API.defaults.adapter = adapter;
});

test('首装无 active 能读取详情、激活最新归档；缺失计数不伪造为零', async () => {
  render(<TaskPlugins />);
  await screen.findByText('Suno');
  expect(screen.getAllByText('Not provided').length).toBeGreaterThan(0);
  await userEvent.click(screen.getByRole('button', { name: 'Details' }));
  await screen.findByText('hash');
  await userEvent.click(screen.getByRole('button', { name: 'Activate' }));
  await userEvent.click(await screen.findByRole('button', { name: 'Confirm' }));
  await waitFor(() =>
    expect(calls).toContainEqual({
      url: '/api/plugin/task/sunoapi/activate',
      method: 'post',
      body: { version: '1.0' },
    }),
  );
  expect(
    calls.every((c) => c.method === 'get' || c.url.endsWith('/activate')),
  ).toBe(true);
});

test('Admin 只读列表和详情且无源码或写入控件', async () => {
  localStorage.setItem('user', JSON.stringify({ role: 10 }));
  render(<TaskPlugins />);
  await screen.findByText('Suno');
  expect(
    screen.queryByRole('button', { name: 'Enable new task submissions' }),
  ).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Details' }));
  await screen.findByText('hash');
  expect(screen.queryByRole('button', { name: 'Activate' })).toBeNull();
  expect(screen.queryByText('export default {}')).toBeNull();
  expect(calls.every((c) => c.method === 'get')).toBe(true);
});

test('详情失败退出加载并允许关闭，不显示上一插件内容', async () => {
  failDetail = true;
  render(<TaskPlugins />);
  await screen.findByText('Suno');
  await userEvent.click(screen.getByRole('button', { name: 'Details' }));
  await screen.findByRole('alert');
  expect(screen.queryByLabelText('Loading plugin details')).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Close details' }));
  await waitFor(() => expect(screen.queryByText('Plugin versions')).toBeNull());
});

test('Root 可启停活跃内置插件并修改 runtime 总开关，明确保留历史轮询', async () => {
  plugin = {
    ...plugin,
    active: true,
    enabled: true,
    channel_count: 2,
    in_flight_count: 3,
  };
  render(<TaskPlugins />);
  await screen.findByText('Suno');
  await userEvent.click(screen.getByRole('button', { name: 'Disable plugin' }));
  await userEvent.click(await screen.findByRole('button', { name: 'Confirm' }));
  await screen.findByRole('button', { name: 'Enable plugin' });
  expect(calls).toContainEqual({
    url: '/api/plugin/task/sunoapi/status',
    method: 'post',
    body: { enabled: false },
  });
  await userEvent.click(
    screen.getByRole('button', { name: 'Enable new task submissions' }),
  );
  await userEvent.click(await screen.findByRole('button', { name: 'Confirm' }));
  await screen.findByRole('button', { name: 'Disable new task submissions' });
  expect(calls).toContainEqual({
    url: '/api/plugin/task/runtime/status',
    method: 'put',
    body: { enabled: true },
  });
  expect(
    screen.getAllByText(
      'Disabling blocks new submissions only. Existing tasks continue with their pinned versions.',
    ).length,
  ).toBeGreaterThan(0);
});

test('列表无权限显示错误而非空列表，不提供写入入口', async () => {
  denyList = true;
  render(<TaskPlugins />);
  await screen.findByRole('alert');
  expect(screen.queryByText('No task plugins found')).toBeNull();
  expect(
    screen.queryByRole('button', { name: 'Enable new task submissions' }),
  ).toBeNull();
  expect(calls.every((c) => c.method === 'get')).toBe(true);
});

test('启停被拒绝保留服务端状态并显示失败', async () => {
  plugin = { ...plugin, active: true, enabled: true };
  denyWrite = true;
  render(<TaskPlugins />);
  await screen.findByText('Suno');
  await userEvent.click(screen.getByRole('button', { name: 'Disable plugin' }));
  await userEvent.click(await screen.findByRole('button', { name: 'Confirm' }));
  await screen.findByRole('alert');
  expect(plugin.enabled).toBe(true);
});

test('普通用户不发管理 API 请求并显示无权限', async () => {
  localStorage.setItem('user', JSON.stringify({ role: 1 }));
  render(<TaskPlugins />);
  expect(screen.getByRole('alert')).toBeTruthy();
  expect(calls).toEqual([]);
});

test('runtime 请求失败禁用总开关，列表仍可只读查看', async () => {
  const current = API.defaults.adapter;
  API.defaults.adapter = (config) => {
    if (config.url.endsWith('/runtime/status'))
      throw Object.assign(new Error('runtime unavailable'), { config });
    return current(config);
  };
  render(<TaskPlugins />);
  await screen.findByText('Suno');
  await screen.findByRole('alert');
  expect(
    screen.queryByRole('button', { name: 'Enable new task submissions' }),
  ).toBeNull();
});

test('详情切换清空旧版本，失败后重试能恢复详情', async () => {
  let resolveDetail;
  const current = API.defaults.adapter;
  API.defaults.adapter = async (config) => {
    if (config.url.includes('/google')) {
      if (config.url.endsWith('/versions'))
        return {
          data: { success: true, data: [] },
          config,
          status: 200,
          headers: {},
        };
      return new Promise((resolve) => {
        resolveDetail = resolve;
      });
    }
    return current(config);
  };
  const view = render(
    <PluginDetails plugin={plugin} canManage={false} onClose={() => {}} />,
  );
  await screen.findByText('Plugin versions');
  expect(screen.getByRole('button', { name: 'View version' })).toBeTruthy();
  view.rerender(
    <PluginDetails
      plugin={{ key: 'google' }}
      canManage={false}
      onClose={() => {}}
    />,
  );
  await screen.findByLabelText('Loading plugin details');
  expect(screen.queryByRole('button', { name: 'View version' })).toBeNull();
  await waitFor(() => expect(resolveDetail).toBeTruthy());
  resolveDetail({
    data: { success: false, message: 'details unavailable' },
    config: {},
    status: 200,
    headers: {},
  });
  await screen.findByRole('alert');
  expect(screen.queryByLabelText('Loading plugin details')).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
  await screen.findByLabelText('Loading plugin details');
  resolveDetail({
    data: { success: true, data: { key: 'google', version: '2.0' } },
    config: {},
    status: 200,
    headers: {},
  });
  await screen.findByText('2.0');
  expect(screen.queryByRole('alert')).toBeNull();
});
