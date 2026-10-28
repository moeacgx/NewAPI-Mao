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
import { afterEach, beforeEach, expect, test } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import Tickets from '../index';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const replyEndpoint = '/api/ticket-admin/automation/auto-reply';
const archiveEndpoint = '/api/ticket-admin/automation/auto-archive';
let writes, canWrite;

beforeEach(() => {
  writes = [];
  canWrite = true;
  localStorage.setItem('user', JSON.stringify({ id: 7, role: 10 }));
  API.defaults.adapter = async (config) => {
    let data;
    if (config.method === 'put') {
      data = JSON.parse(config.data);
      writes.push({ url: config.url, body: data });
    } else if (config.url === '/api/user/self') {
      data = {
        permissions: {
          admin_permissions: { tickets: { read: true, write: canWrite } },
        },
      };
    } else if (config.url === replyEndpoint) {
      data = { enabled: true, content: 'Received' };
    } else if (config.url === archiveEndpoint) {
      data = {
        enabled: true,
        inactive_days: 7,
        statuses: ['resolved', 'closed'],
      };
    } else if (config.url === '/api/ticket-admin') {
      data = {
        items: [
          {
            id: 8,
            user_id: 7,
            title: 'Payment issue',
            category: 'billing',
            priority: 'normal',
            status: 'open',
            updated_at: 1792600000,
          },
        ],
        total: 1,
      };
    } else {
      throw new Error(`Unexpected request: ${config.method} ${config.url}`);
    }
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
  API.defaults.adapter = originalAdapter;
});

test('进入工单管理时先显示列表，自动回复与归档共同默认折叠', async () => {
  render(
    <MemoryRouter>
      <Tickets admin />
    </MemoryRouter>,
  );
  const ticket = await screen.findByRole('button', { name: 'Payment issue' });
  const summary = screen.getByText('Ticket settings');
  const settings = summary.closest('details');

  expect(settings.open).toBe(false);
  expect(
    ticket.compareDocumentPosition(summary) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();

  await userEvent.click(summary);

  expect(settings.open).toBe(true);
  expect(
    await screen.findByLabelText('Automatic reply content'),
  ).toHaveProperty('value', 'Received');
  expect(await screen.findByLabelText('Inactive days')).toHaveProperty(
    'value',
    '7',
  );
  expect(
    settings.contains(
      screen.getByRole('button', { name: 'Save automation settings' }),
    ),
  ).toBe(true);
  expect(
    settings.contains(
      screen.getByRole('button', { name: 'Save archive settings' }),
    ),
  ).toBe(true);
});

test('设置收起再展开保留两个草稿，两项保存仍使用各自配置接口', async () => {
  render(
    <MemoryRouter>
      <Tickets admin />
    </MemoryRouter>,
  );
  await screen.findByRole('button', { name: 'Payment issue' });
  const summary = screen.getByText('Ticket settings');
  await userEvent.click(summary);
  const content = await screen.findByLabelText('Automatic reply content');
  const days = await screen.findByLabelText('Inactive days');
  await userEvent.clear(content);
  await userEvent.type(content, 'Please keep this reply draft.');
  await userEvent.clear(days);
  await userEvent.type(days, '14');
  await userEvent.click(screen.getByLabelText('Closed'));

  await userEvent.click(summary);
  expect(summary.closest('details').open).toBe(false);
  await userEvent.click(summary);

  expect(screen.getByLabelText('Automatic reply content')).toHaveProperty(
    'value',
    'Please keep this reply draft.',
  );
  expect(screen.getByLabelText('Inactive days')).toHaveProperty('value', '14');
  expect(screen.getByLabelText('Closed').checked).toBe(false);
  expect(writes).toHaveLength(0);

  await userEvent.click(
    screen.getByRole('button', { name: 'Save automation settings' }),
  );
  await waitFor(() =>
    expect(writes).toEqual([
      {
        url: replyEndpoint,
        body: { enabled: true, content: 'Please keep this reply draft.' },
      },
    ]),
  );
  expect(screen.getByLabelText('Inactive days')).toHaveProperty('value', '14');
  expect(screen.getByLabelText('Closed').checked).toBe(false);

  await userEvent.click(
    screen.getByRole('button', { name: 'Save archive settings' }),
  );
  await screen.findByText('Archive settings saved');
  expect(writes).toHaveLength(2);
  expect(writes[1]).toEqual({
    url: archiveEndpoint,
    body: { enabled: true, inactive_days: 14, statuses: ['resolved'] },
  });
  expect(screen.getByLabelText('Automatic reply content')).toHaveProperty(
    'value',
    'Please keep this reply draft.',
  );
});

test('只读管理员展开工单设置后能读取两项配置但不能编辑或保存', async () => {
  canWrite = false;
  render(
    <MemoryRouter>
      <Tickets admin />
    </MemoryRouter>,
  );
  await screen.findByRole('button', { name: 'Payment issue' });
  await userEvent.click(screen.getByText('Ticket settings'));
  const content = await screen.findByLabelText('Automatic reply content');
  const days = await screen.findByLabelText('Inactive days');

  expect(content).toHaveProperty('disabled', true);
  expect(days).toHaveProperty('disabled', true);
  expect(screen.getByLabelText('Enable automatic first reply')).toHaveProperty(
    'disabled',
    true,
  );
  expect(screen.getByLabelText('Enable automatic archive')).toHaveProperty(
    'disabled',
    true,
  );
  expect(screen.getByLabelText('Resolved').matches(':disabled')).toBe(true);
  for (const name of ['Save automation settings', 'Save archive settings']) {
    const save = screen.getByRole('button', { name });
    expect(save).toHaveProperty('disabled', true);
    await userEvent.click(save);
  }
  expect(writes).toHaveLength(0);
});
