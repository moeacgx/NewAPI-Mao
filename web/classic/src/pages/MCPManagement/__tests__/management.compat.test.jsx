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
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MCPManagement from '../index';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const tools = [
  {
    name: 'users_list',
    description: 'List users',
    category: 'users',
    write: false,
    min_role: 10,
    inputSchema: { type: 'object', properties: { limit: { type: 'integer' } } },
  },
  {
    name: 'channels_update',
    description: 'Update channel',
    category: 'channels',
    write: true,
    min_role: 100,
    inputSchema: { type: 'object', required: ['request_id'] },
  },
];
let calls, keys, fail, denyMutation;
beforeEach(() => {
  calls = [];
  fail = false;
  denyMutation = false;
  keys = [
    {
      id: 1,
      name: 'Laptop',
      key_prefix: 'mcp_prefix',
      tools: ['users_list'],
      created_at: 1700000000,
      expires_at: 4102444800,
      last_used_at: 1700000010,
      revoked_at: 0,
    },
    {
      id: 2,
      name: 'Expired key',
      key_prefix: 'expired',
      tools: [],
      created_at: 1700000000,
      expires_at: 1700000001,
      last_used_at: 0,
      revoked_at: 0,
    },
    {
      id: 3,
      name: 'Revoked key',
      key_prefix: 'revoked',
      tools: [],
      created_at: 1700000000,
      expires_at: 4102444800,
      last_used_at: 0,
      revoked_at: 1700000010,
    },
  ];
  localStorage.setItem('user', JSON.stringify({ id: 1, role: 100 }));
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ method: config.method, url: config.url, body });
    let data;
    if (config.method === 'post')
      data = { key: keys[0], secret: 'one-time-secret' };
    else if (config.method === 'delete') {
      keys = keys.map((key) =>
        key.id === Number(config.url.split('/').at(-1))
          ? { ...key, revoked_at: 1700000010 }
          : key,
      );
      data = null;
    } else if (config.url === '/api/mcp/tools') data = tools;
    else if (config.url.startsWith('/api/mcp/keys?'))
      data = { items: keys, total: 3 };
    else if (config.url.startsWith('/api/mcp/logs?'))
      data = {
        items: [
          {
            id: 1,
            key_id: 1,
            tool_name: 'users_list',
            success: true,
            duration_ms: 42,
            request_id: config.url.includes('p=2&')
              ? 'request-page-2'
              : 'request-page-1',
            created_at: 1700000000,
          },
          {
            id: 2,
            key_id: 1,
            tool_name: 'channels_update',
            success: false,
            duration_ms: 9,
            request_id: 'failed-request',
            created_at: 1700000001,
          },
        ],
        total: 21,
      };
    else throw new Error(`Unexpected request: ${config.url}`);
    return {
      data: {
        success: !(fail || (denyMutation && config.method !== 'get')),
        data,
      },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
});
afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

async function openKeys() {
  const user = userEvent.setup();
  render(<MCPManagement />);
  await user.click(screen.getByRole('tab', { name: 'Call keys' }));
  await screen.findByRole('checkbox', { name: /users_list/ });
  return user;
}

async function fillKey(user) {
  await user.type(screen.getByLabelText('Key name'), ' Workstation ');
  fireEvent.change(screen.getByLabelText('Expires at'), {
    target: { value: '2099-01-01T12:00' },
  });
  await user.click(screen.getByRole('checkbox', { name: /users_list/ }));
}

test('non-admin users cannot see management tabs or issue requests', () => {
  localStorage.setItem('user', JSON.stringify({ role: 1 }));
  render(<MCPManagement />);
  expect(screen.getByRole('alert').textContent).toBe(
    'Administrator access required',
  );
  expect(screen.queryByRole('tab')).toBeNull();
  expect(calls).toEqual([]);
});

test('access configuration uses current origin and environment references and copies the URL', async () => {
  const user = userEvent.setup();
  const write = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue();
  const { container } = render(<MCPManagement />);
  expect(container.querySelectorAll('pre')).toHaveLength(1);
  expect(container.querySelector('pre').textContent).toContain(
    `url = "${window.location.origin}/mcp"`,
  );
  expect(container.querySelector('pre').textContent).toContain(
    'bearer_token_env_var = "NEWAPI_MCP_KEY"',
  );
  await user.click(screen.getByRole('button', { name: 'Cursor', exact: true }));
  expect(
    JSON.parse(container.querySelector('pre').textContent).mcpServers.newapi_mao
      .headers.Authorization,
  ).toBe('Bearer ${env:NEWAPI_MCP_KEY}');
  await user.click(screen.getByRole('button', { name: 'Copy MCP URL' }));
  expect(write).toHaveBeenCalledWith(window.location.origin + '/mcp');
  expect(screen.getByText('Copied')).toBeTruthy();
  expect(calls).toEqual([]);
});

test('large key permissions stay compact and historical tools remain searchable in the dialog', async () => {
  keys[0].tools = [
    'users_list',
    ...Array.from({ length: 367 }, (_, index) => `historical_tool_${index}`),
  ];
  const user = await openKeys();
  const row = screen.getByText('Laptop').closest('tr');
  expect(row.textContent).toContain('368');
  expect(row.textContent).not.toContain('historical_tool_');
  await user.click(
    within(row).getByRole('button', { name: 'View permissions' }),
  );
  const dialog = screen.getByRole('dialog');
  expect(within(dialog).getByText('historical_tool_366')).toBeTruthy();
  await user.type(
    within(dialog).getByLabelText('Search tools'),
    'historical_tool_366',
  );
  expect(within(dialog).queryByText('users_list')).toBeNull();
  expect(
    within(dialog).getByText('Permission details unavailable'),
  ).toBeTruthy();
  expect(calls.filter((call) => call.method !== 'get')).toEqual([]);
});

test('clipboard failure offers a manual-copy fallback', async () => {
  const user = userEvent.setup();
  vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(
    new Error('Denied'),
  );
  render(<MCPManagement />);
  await user.click(
    screen.getByRole('button', { name: 'Copy Codex configuration' }),
  );
  expect(
    await screen.findByText('Copy failed. Select and copy the text manually.'),
  ).toBeTruthy();
});

test('keyboard navigation selects tabs and tool search filters category, mode and schema', async () => {
  const user = userEvent.setup();
  render(<MCPManagement />);
  screen.getByRole('tab', { name: 'MCP access' }).focus();
  await user.keyboard('{ArrowRight}');
  expect(
    screen
      .getByRole('tab', { name: 'Tool reference' })
      .getAttribute('aria-selected'),
  ).toBe('true');
  await screen.findByText('users_list');
  await user.selectOptions(screen.getByLabelText('Tool category'), 'channels');
  expect(screen.queryByText('users_list')).toBeNull();
  expect(
    within(screen.getByRole('region', { name: 'channels' })).getByText(
      'Write tool',
    ),
  ).toBeTruthy();
  await user.click(screen.getByText('Input schema'));
  expect(screen.getByText(/"request_id"/)).toBeTruthy();
  await user.type(screen.getByLabelText('Search tools'), 'missing');
  expect(screen.getByText('No matching tools')).toBeTruthy();
});

test('只读与明确写入批选提交显式白名单，隐藏选择保持有效', async () => {
  const user = await openKeys();
  await user.type(screen.getByLabelText('Key name'), 'Scoped client');
  fireEvent.change(screen.getByLabelText('Expires at'), {
    target: { value: '2099-01-01T12:00' },
  });
  await user.click(
    screen.getByRole('button', { name: 'Select visible read-only tools' }),
  );
  await user.selectOptions(screen.getByLabelText('Tool category'), 'channels');
  await user.click(
    screen.getByRole('button', {
      name: 'Explicitly select visible write tools',
    }),
  );
  await user.click(screen.getByRole('button', { name: 'Create call key' }));
  await screen.findByText('one-time-secret');
  expect(calls.filter((call) => call.method === 'post')).toEqual([
    {
      method: 'post',
      url: '/api/mcp/keys',
      body: {
        name: 'Scoped client',
        tools: ['users_list', 'channels_update'],
        expires_at: new Date('2099-01-01T12:00').getTime() / 1000,
      },
    },
  ]);
});

test('creating a key validates name, expiry and explicit tool selection before sending', async () => {
  const user = await openKeys();
  const create = screen.getByRole('button', { name: 'Create call key' });
  await user.click(create);
  expect(screen.getByRole('alert').textContent).toContain('1-100');
  fireEvent.change(screen.getByLabelText('Key name'), {
    target: { value: '🔑'.repeat(101) },
  });
  await user.click(create);
  expect(screen.getByRole('alert').textContent).toContain('1-100');
  await user.clear(screen.getByLabelText('Key name'));
  await user.type(screen.getByLabelText('Key name'), 'Laptop');
  fireEvent.change(screen.getByLabelText('Expires at'), {
    target: { value: '2000-01-01T12:00' },
  });
  await user.click(create);
  expect(screen.getByRole('alert').textContent).toBe(
    'Choose a future expiry time.',
  );
  fireEvent.change(screen.getByLabelText('Expires at'), {
    target: { value: '2099-01-01T12:00' },
  });
  await user.click(create);
  expect(screen.getByRole('alert').textContent).toBe(
    'Select at least one available tool.',
  );
  expect(calls.filter((call) => call.method === 'post')).toEqual([]);
});

test('year 10000 expiry is rejected before sending a key creation request', async () => {
  const user = await openKeys();
  await fillKey(user);
  fireEvent.change(screen.getByLabelText('Expires at'), {
    target: { value: '10000-01-02T12:00' },
  });
  expect(screen.getByLabelText('Expires at').value).toBe('10000-01-02T12:00');
  await user.click(screen.getByRole('button', { name: 'Create call key' }));
  expect(screen.getByRole('alert').textContent).toBe(
    'Choose a future expiry time.',
  );
  expect(calls.filter((call) => call.method === 'post')).toEqual([]);
});

test('100 Unicode characters are accepted without counting surrogate pairs twice', async () => {
  const user = await openKeys();
  await fillKey(user);
  const name = '🔑'.repeat(100);
  fireEvent.change(screen.getByLabelText('Key name'), {
    target: { value: ` ${name} ` },
  });
  await user.click(screen.getByRole('button', { name: 'Create call key' }));
  await screen.findByText('one-time-secret');
  expect(calls.find((call) => call.method === 'post').body.name).toBe(name);
});

test('write tools start unchecked and the secret is cleared by closing or leaving the tab', async () => {
  const user = await openKeys();
  expect(
    screen.getByRole('checkbox', { name: /channels_update/ }).checked,
  ).toBe(false);
  await fillKey(user);
  await user.click(screen.getByRole('button', { name: 'Create call key' }));
  await screen.findByText('one-time-secret');
  expect(calls.find((call) => call.method === 'post').body).toEqual({
    name: 'Workstation',
    tools: ['users_list'],
    expires_at: Math.floor(new Date('2099-01-01T12:00').getTime() / 1000),
  });
  expect(JSON.stringify(localStorage)).not.toContain('one-time-secret');
  expect(JSON.stringify(sessionStorage)).not.toContain('one-time-secret');
  await user.click(screen.getByRole('button', { name: 'Close secret' }));
  expect(screen.queryByText('one-time-secret')).toBeNull();
  await fillKey(user);
  await user.click(screen.getByRole('checkbox', { name: /channels_update/ }));
  await user.click(screen.getByRole('button', { name: 'Create call key' }));
  await screen.findByText('one-time-secret');
  expect(calls.filter((call) => call.method === 'post')[1].body.tools).toEqual([
    'users_list',
    'channels_update',
  ]);
  await user.click(screen.getByRole('tab', { name: 'MCP access' }));
  await user.click(screen.getByRole('tab', { name: 'Call keys' }));
  await screen.findByRole('checkbox', { name: /users_list/ });
  expect(screen.queryByText('one-time-secret')).toBeNull();
});

test('revoking another key preserves the new secret, while revoking its own key clears it', async () => {
  const user = await openKeys();
  await fillKey(user);
  await user.click(screen.getByRole('button', { name: 'Create call key' }));
  await screen.findByText('one-time-secret');
  await user.click(
    within(screen.getByText('Expired key').closest('tr')).getByRole('button', {
      name: 'Revoke',
    }),
  );
  await user.click(
    await screen.findByRole('button', { name: 'Confirm revoke' }),
  );
  await waitFor(() =>
    expect(
      calls.some((call) => call.method === 'delete' && call.url.endsWith('/2')),
    ).toBe(true),
  );
  expect(screen.getByText('one-time-secret')).toBeTruthy();
  const revoke = within(screen.getByText('Laptop').closest('tr')).getByRole(
    'button',
    { name: 'Revoke' },
  );
  await waitFor(() => expect(revoke.disabled).toBe(false));
  await user.click(revoke);
  await user.click(
    await screen.findByRole('button', { name: 'Confirm revoke' }),
  );
  await waitFor(() => expect(screen.queryByText('one-time-secret')).toBeNull());
});

test('revocation requires confirmation and shows revoked, expired and last-used metadata', async () => {
  const user = await openKeys();
  const table = await screen.findByRole('table', { name: 'Call keys' });
  expect(within(table).getByText('Active')).toBeTruthy();
  expect(within(table).getByText('Expired')).toBeTruthy();
  expect(within(table).getByText('Revoked')).toBeTruthy();
  expect(
    within(table).getByText(
      new Date(1700000010 * 1000).toLocaleString('zh-CN'),
    ),
  ).toBeTruthy();
  expect(
    within(table).getAllByRole('button', { name: 'Revoke' })[2].disabled,
  ).toBe(true);
  await user.click(within(table).getAllByRole('button', { name: 'Revoke' })[0]);
  expect(calls.filter((call) => call.method === 'delete')).toEqual([]);
  await user.click(await screen.findByRole('button', { name: 'Cancel' }));
  expect(calls.filter((call) => call.method === 'delete')).toEqual([]);
  await user.click(within(table).getAllByRole('button', { name: 'Revoke' })[0]);
  await user.click(
    await screen.findByRole('button', { name: 'Confirm revoke' }),
  );
  await waitFor(() =>
    expect(
      calls.some(
        (call) => call.url === '/api/mcp/keys/1' && call.method === 'delete',
      ),
    ).toBe(true),
  );
  await waitFor(() => expect(screen.queryByText('Active')).toBeNull());
});

test('failed creation stays visible as an error without showing a secret', async () => {
  const user = await openKeys();
  denyMutation = true;
  await fillKey(user);
  await user.click(screen.getByRole('button', { name: 'Create call key' }));
  expect(await screen.findByRole('alert')).toBeTruthy();
  expect(screen.queryByText('one-time-secret')).toBeNull();
  expect(screen.getByLabelText('Key name').value).toBe(' Workstation ');
});

test('logs paginate and refresh using p and page_size while retaining result metadata', async () => {
  const user = userEvent.setup();
  render(<MCPManagement />);
  await user.click(screen.getByRole('tab', { name: 'Call logs' }));
  await screen.findByText('request-page-1');
  expect(screen.getByText('Succeeded')).toBeTruthy();
  expect(screen.getByText('Failed')).toBeTruthy();
  expect(screen.getByText('42')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Previous page' }).disabled).toBe(
    true,
  );
  await user.click(screen.getByRole('button', { name: 'Next page' }));
  await screen.findByText('request-page-2');
  expect(screen.getByRole('button', { name: 'Next page' }).disabled).toBe(true);
  await user.click(screen.getByRole('button', { name: 'Refresh' }));
  await screen.findByText('request-page-2');
  expect(
    calls.filter((call) => call.url === '/api/mcp/logs?p=2&page_size=20'),
  ).toHaveLength(2);
});

test('a failed list request clears stale rows and refresh recovers', async () => {
  const user = userEvent.setup();
  render(<MCPManagement />);
  await user.click(screen.getByRole('tab', { name: 'Call logs' }));
  await screen.findByText('request-page-1');
  fail = true;
  await user.click(screen.getByRole('button', { name: 'Refresh' }));
  await screen.findByRole('alert');
  expect(screen.queryByText('request-page-1')).toBeNull();
  fail = false;
  await user.click(screen.getByRole('button', { name: 'Refresh' }));
  await screen.findByText('request-page-1');
});

test('empty key lists show a bounded empty state and disable pagination', async () => {
  keys = [];
  await openKeys();
  expect(await screen.findByText('No MCP records')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Next page' }).disabled).toBe(true);
});
