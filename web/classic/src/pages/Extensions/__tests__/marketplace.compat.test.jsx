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
import { createHash, webcrypto } from 'node:crypto';
import { beforeEach, afterEach, expect, test, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import ExtensionMarketplace from '../ExtensionMarketplace';
import Extensions from '../index';
import { API } from '../../../helpers/api';
import {
  EXTENSION_CATALOG_URL,
  EXTENSION_REPOSITORY_URL,
} from '../marketplace-utils';

const originalAdapter = API.defaults.adapter;
const zip = new Uint8Array([80, 75, 3, 4, 5, 6]);
let catalog, calls, failMetadata, failUpload, installed;

beforeEach(() => {
  calls = [];
  failMetadata = false;
  failUpload = false;
  installed = vi.fn();
  localStorage.setItem('user', JSON.stringify({ role: 100 }));
  catalog = {
    catalogVersion: 1,
    purpose: 'extension-catalog',
    name: 'Official modules',
    modules: [
      {
        id: 'demo',
        name: 'Catalog Demo',
        version: '0.2.0',
        path: 'published/demo/0.2.0/demo-0.2.0.zip',
        sha256: createHash('sha256').update(zip).digest('hex'),
        size: zip.length,
        host: { min: 'v1.0.0' },
        status: 'requires-host-upgrade',
        compatibility: 'Requires installed host support',
      },
    ],
  };
  vi.stubGlobal('crypto', webcrypto);
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async (url) =>
        new Response(
          url === EXTENSION_CATALOG_URL ? JSON.stringify(catalog) : zip,
        ),
    ),
  );
  API.defaults.adapter = async (config) => {
    calls.push(config);
    let data;
    if (config.url.endsWith('/marketplace')) {
      if (failMetadata)
        return {
          data: { success: false, message: 'Metadata unavailable' },
          status: 200,
          headers: {},
          config,
        };
      data = {
        catalog_url: EXTENSION_CATALOG_URL,
        repository_url: EXTENSION_REPOSITORY_URL,
        host_version: 'v1.1.0',
        max_archive_bytes: 104857600,
      };
    } else if (config.url.endsWith('/upload')) {
      if (failUpload)
        return {
          data: { success: false, message: 'Host capability unavailable' },
          status: 200,
          headers: {},
          config,
        };
      data = {
        module: {
          id: 'demo',
          name: 'Installed Demo',
          version: '0.2.0',
          enabled: false,
        },
      };
    } else if (config.url === '/api/extension-admin/') {
      data = {
        root: 'data/modules',
        modules: calls.some((item) => item.url.endsWith('/upload'))
          ? [
              {
                id: 'demo',
                name: 'Installed Demo',
                version: '0.2.0',
                enabled: false,
              },
            ]
          : [],
      };
    } else throw new Error('Unexpected API: ' + config.url);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
  vi.unstubAllGlobals();
});

test('Root 先确认模块和版本再下载上传，传递完整来源字段且成功回调刷新', async () => {
  render(<ExtensionMarketplace canManage onInstalled={installed} />);
  await screen.findByText('Catalog Demo');
  expect(screen.getByText('Requires installed host support')).toBeTruthy();
  await userEvent.click(
    screen.getByRole('button', { name: 'Install from repository' }),
  );
  const dialog = await screen.findByRole('dialog');
  expect(within(dialog).getByText('Catalog Demo')).toBeTruthy();
  expect(within(dialog).getByText('0.2.0')).toBeTruthy();
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(calls.some((item) => item.method === 'post')).toBe(false);
  await userEvent.click(
    within(dialog).getByRole('button', { name: 'Confirm installation' }),
  );
  await waitFor(() => expect(installed).toHaveBeenCalledOnce());
  const upload = calls.find((item) => item.url.endsWith('/upload'));
  expect(upload.data.get('file').name).toBe('demo-0.2.0.zip');
  expect(upload.data.get('file').size).toBe(zip.length);
  expect(
    Object.fromEntries(
      [...upload.data.entries()].filter(([key]) => key !== 'file'),
    ),
  ).toEqual({
    archiveSha256: catalog.modules[0].sha256,
    expectedId: 'demo',
    expectedVersion: '0.2.0',
    catalogUrl: EXTENSION_CATALOG_URL,
    archivePath: catalog.modules[0].path,
  });
  expect(calls.some((item) => item.url.endsWith('/enabled'))).toBe(false);
});

test('取消确认不下载 ZIP、不上传', async () => {
  render(<ExtensionMarketplace canManage onInstalled={installed} />);
  await screen.findByText('Catalog Demo');
  await userEvent.click(
    screen.getByRole('button', { name: 'Install from repository' }),
  );
  await userEvent.click(
    within(await screen.findByRole('dialog')).getByRole('button', {
      name: 'Cancel',
    }),
  );
  expect(screen.getAllByText('Catalog Demo')).toHaveLength(1);
  expect(fetch).toHaveBeenCalledTimes(1);
  expect(calls.some((item) => item.method === 'post')).toBe(false);
});

test('哈希错误拒绝上传并保持确认错误可见', async () => {
  catalog.modules[0].sha256 = '0'.repeat(64);
  render(<ExtensionMarketplace canManage onInstalled={installed} />);
  await screen.findByText('Catalog Demo');
  await userEvent.click(
    screen.getByRole('button', { name: 'Install from repository' }),
  );
  await userEvent.click(
    within(await screen.findByRole('dialog')).getByRole('button', {
      name: 'Confirm installation',
    }),
  );
  await screen.findByText('Extension SHA-256 does not match the catalog.');
  expect(calls.some((item) => item.method === 'post')).toBe(false);
  expect(installed).not.toHaveBeenCalled();
});

test('宿主拒绝安装时保留失败原因，不刷新为成功', async () => {
  failUpload = true;
  render(<ExtensionMarketplace canManage onInstalled={installed} />);
  await screen.findByText('Catalog Demo');
  await userEvent.click(
    screen.getByRole('button', { name: 'Install from repository' }),
  );
  await userEvent.click(
    within(await screen.findByRole('dialog')).getByRole('button', {
      name: 'Confirm installation',
    }),
  );
  await screen.findByText('Host capability unavailable');
  expect(installed).not.toHaveBeenCalled();
});

test('空清单显示空态，来源失败可重试后加载', async () => {
  failMetadata = true;
  render(<ExtensionMarketplace canManage onInstalled={installed} />);
  await screen.findByText('Could not load the extension repository.');
  failMetadata = false;
  catalog.modules = [];
  await userEvent.click(
    screen.getByRole('button', { name: 'Reload repository' }),
  );
  await screen.findByText('No online modules are available.');
  expect(
    screen.queryByRole('button', { name: 'Install from repository' }),
  ).toBeNull();
});

test('非 Root 不展示在线入口也不发送来源请求', async () => {
  localStorage.setItem('user', JSON.stringify({ role: 10 }));
  render(
    <MemoryRouter>
      <Extensions />
    </MemoryRouter>,
  );
  await screen.findByText('只有 root 用户可以管理扩展模块');
  expect(screen.queryByText('Online modules')).toBeNull();
  expect(calls).toEqual([]);
  expect(fetch).not.toHaveBeenCalled();
});

test('模块管理安装后重载已安装模块并通知侧栏，手动上传入口保留', async () => {
  const sidebar = vi.fn();
  window.addEventListener('classic-extension-refresh', sidebar);
  try {
    render(
      <MemoryRouter>
        <Extensions />
      </MemoryRouter>,
    );
    await screen.findByText('Catalog Demo');
    expect(screen.getByRole('button', { name: '上传模块' })).toBeTruthy();
    await userEvent.click(
      screen.getByRole('button', { name: 'Install from repository' }),
    );
    await userEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', {
        name: 'Confirm installation',
      }),
    );
    await screen.findByText('Installed Demo');
    expect(sidebar).toHaveBeenCalledOnce();
    expect(
      calls.filter((item) => item.url === '/api/extension-admin/'),
    ).toHaveLength(2);
  } finally {
    window.removeEventListener('classic-extension-refresh', sidebar);
  }
});
