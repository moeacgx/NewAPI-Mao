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
import { webcrypto, createHash } from 'node:crypto';
import { beforeEach, afterEach, expect, test, vi } from 'vitest';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Marketplace from '../Marketplace';
import MarketplaceSources from '../MarketplaceSources';
import TaskPlugins from '../index';
import { API } from '../../../helpers/api';

const adapter = API.defaults.adapter;
const source = '// 可审查源码\nexport default {};';
const hash = createHash('sha256').update(source).digest('hex');
const sourceUrl = 'https://plugins.example/stable/index.json';
let calls, sources, installed, payload, rejectInstall;

beforeEach(() => {
  calls = [];
  rejectInstall = false;
  installed = vi.fn();
  sources = [
    { name: 'Official', index_url: sourceUrl },
    { name: 'MaoLao', index_url: 'https://maintained.example/index.json' },
  ];
  payload = {
    indexVersion: 1,
    plugins: [
      {
        key: 'demo',
        name: 'Stable Demo',
        latest: '1.0',
        versions: [{ version: '1.0', path: 'demo/plugin.js', sha256: hash }],
      },
    ],
  };
  localStorage.setItem('user', JSON.stringify({ role: 100 }));
  vi.stubGlobal('crypto', webcrypto);
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async (url) =>
        new Response(
          url.endsWith('index.json') ? JSON.stringify(payload) : source,
        ),
    ),
  );
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ method: config.method, url: config.url, body });
    let data;
    if (config.url.endsWith('/marketplace/sources')) {
      if (config.method === 'put') sources = body;
      data = sources;
    } else if (config.url === '/api/plugin/task' && config.method === 'post') {
      if (rejectInstall)
        throw Object.assign(new Error('Conflict'), {
          response: { status: 409, data: { message: 'Version conflict' } },
          config,
        });
      data = { key: 'demo', version: '1.0', active: false, enabled: false };
    } else if (config.url === '/api/plugin/task') data = [];
    else if (config.url.endsWith('/runtime/status')) data = { enabled: false };
    else throw new Error('Unexpected API: ' + config.url);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = adapter;
  vi.unstubAllGlobals();
});

test('Root 确认已校验源码后提交精确版本与来源，安装不自动激活', async () => {
  render(<Marketplace canManage onInstalled={installed} />);
  await screen.findByText('Stable Demo');
  await userEvent.click(
    screen.getByRole('button', { name: 'Review and install' }),
  );
  await screen.findByLabelText('Plugin source');
  expect(screen.getByLabelText('Plugin source').textContent).toBe(source);
  expect(calls.filter((call) => call.method === 'post')).toEqual([]);
  await userEvent.click(
    screen.getByRole('button', { name: 'Install version' }),
  );
  await screen.findByText(
    'Plugin installed disabled and inactive. Activate it from Installed when ready.',
  );
  expect(calls).toContainEqual({
    method: 'post',
    url: '/api/plugin/task',
    body: {
      source,
      sourceSha256: hash,
      expectedKey: 'demo',
      expectedVersion: '1.0',
      marketplace: {
        name: 'Official',
        index_url: sourceUrl,
        path: 'demo/plugin.js',
      },
    },
  });
  expect(calls.some((call) => call.url.endsWith('/activate'))).toBe(false);
  expect(installed).toHaveBeenCalledOnce();
});

test('哈希不匹配时安装按钮禁用，源码不上传', async () => {
  payload.plugins[0].versions[0].sha256 = '0'.repeat(64);
  render(<Marketplace canManage onInstalled={installed} />);
  await screen.findByText('Stable Demo');
  await userEvent.click(
    screen.getByRole('button', { name: 'Review and install' }),
  );
  await screen.findByText('Plugin SHA-256 does not match the index.');
  expect(screen.getByRole('button', { name: 'Install version' }).disabled).toBe(
    true,
  );
  expect(calls.every((call) => call.method === 'get')).toBe(true);
});

test('Admin 可切换多个源但没有配置、安装和上传控件', async () => {
  localStorage.setItem('user', JSON.stringify({ role: 10 }));
  render(<TaskPlugins />);
  await screen.findByText('No task plugins found');
  await userEvent.click(screen.getByRole('tab', { name: 'Marketplace' }));
  await screen.findByText('Stable Demo');
  expect(
    screen.queryByRole('button', { name: 'Manage plugin sources' }),
  ).toBeNull();
  expect(
    screen.queryByRole('button', { name: 'Review and install' }),
  ).toBeNull();
  expect(
    screen.queryByRole('button', { name: 'Upload custom task plugin' }),
  ).toBeNull();
  await userEvent.selectOptions(
    screen.getByLabelText('Plugin source repository'),
    sources[1].index_url,
  );
  await waitFor(() =>
    expect(fetch).toHaveBeenCalledWith(
      sources[1].index_url,
      expect.objectContaining({ credentials: 'omit' }),
    ),
  );
  expect(calls.every((call) => call.method === 'get')).toBe(true);
});

test('Root 保存新源只提交名称和地址，允许清空且不删除插件', async () => {
  const onSaved = vi.fn();
  render(
    <MarketplaceSources
      sources={sources}
      onSaved={onSaved}
      onClose={() => {}}
      onDenied={() => {}}
    />,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Add plugin source' }),
  );
  await userEvent.type(screen.getByLabelText('Source name 3'), 'Testing');
  await userEvent.type(
    screen.getByLabelText('Index URL 3'),
    'https://testing.example/index.json',
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Save plugin sources' }),
  );
  await waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
  expect(calls[0].body).toEqual([
    { name: 'Official', index_url: sourceUrl },
    { name: 'MaoLao', index_url: 'https://maintained.example/index.json' },
    { name: 'Testing', index_url: 'https://testing.example/index.json' },
  ]);
  for (let i = 0; i < 3; i++)
    await userEvent.click(
      screen.getByRole('button', { name: 'Remove source 1' }),
    );
  await userEvent.click(
    screen.getByRole('button', { name: 'Save plugin sources' }),
  );
  await waitFor(() => expect(calls.at(-1).body).toEqual([]));
  expect(calls.every((call) => call.url.endsWith('/marketplace/sources'))).toBe(
    true,
  );
});

test('重复源阻止保存且保留表单', async () => {
  render(
    <MarketplaceSources
      sources={[sources[0], sources[0]]}
      onSaved={vi.fn()}
      onClose={() => {}}
      onDenied={() => {}}
    />,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Save plugin sources' }),
  );
  await screen.findByRole('alert');
  expect(calls).toEqual([]);
  expect(screen.getByLabelText('Source name 2')).toBeTruthy();
});

test('远端失败显示错误并可重试，不伪造成功或安装', async () => {
  fetch.mockRejectedValueOnce(new TypeError('Failed to fetch'));
  render(<Marketplace canManage onInstalled={installed} />);
  await screen.findByRole('alert');
  expect(
    screen.queryByRole('button', { name: 'Review and install' }),
  ).toBeNull();
  await userEvent.click(
    screen.getByRole('button', { name: 'Refresh marketplace' }),
  );
  await screen.findByText('Stable Demo');
  expect(screen.queryByRole('alert')).toBeNull();
});

test('后端安装冲突显示错误，不调用成功刷新', async () => {
  rejectInstall = true;
  render(<Marketplace canManage onInstalled={installed} />);
  await screen.findByText('Stable Demo');
  await userEvent.click(
    screen.getByRole('button', { name: 'Review and install' }),
  );
  await screen.findByLabelText('Plugin source');
  await userEvent.click(
    screen.getByRole('button', { name: 'Install version' }),
  );
  await screen.findByText('Version conflict');
  expect(installed).not.toHaveBeenCalled();
});

test('切换源后迟到的旧索引不覆盖当前源，也不能安装旧版本', async () => {
  let completeOld;
  fetch.mockImplementation(async (url) => {
    if (url === sourceUrl)
      return new Promise((resolve) => {
        completeOld = resolve;
      });
    return new Response(JSON.stringify(payload));
  });
  render(<Marketplace canManage onInstalled={installed} />);
  const selector = await screen.findByLabelText('Plugin source repository');
  await waitFor(() => expect(completeOld).toBeTypeOf('function'));
  await userEvent.selectOptions(selector, sources[1].index_url);
  await screen.findByText('Stable Demo');
  await act(async () =>
    completeOld(
      new Response(
        JSON.stringify({
          ...payload,
          plugins: [{ ...payload.plugins[0], name: 'Old Source Plugin' }],
        }),
      ),
    ),
  );
  expect(screen.queryByText('Old Source Plugin')).toBeNull();
  expect(selector.value).toBe(sources[1].index_url);
});

test('源配置接口拒绝时显示权限错误而非空市场，阻止配置和安装', async () => {
  API.defaults.adapter = async (config) => {
    throw Object.assign(new Error('Denied'), {
      response: { status: 403 },
      config,
    });
  };
  render(<Marketplace canManage onInstalled={installed} />);
  await screen.findByText('No permission to access task plugins');
  expect(
    screen.queryByText('No compatible task plugins in this source'),
  ).toBeNull();
  expect(
    screen.queryByRole('button', { name: 'Manage plugin sources' }),
  ).toBeNull();
  expect(fetch).not.toHaveBeenCalled();
});
