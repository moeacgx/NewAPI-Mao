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
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TicketAutomationSettings from '../TicketAutomationSettings';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;

function response(config, data, success = true, message = '') {
  return {
    data: { success, data, message },
    status: success ? 200 : 400,
    statusText: success ? 'OK' : 'Bad Request',
    headers: {},
    config,
  };
}

let calls;
beforeEach(() => {
  calls = [];
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ url: config.url, method: config.method, body });
    if (config.method === 'get') {
      return response(config, {
        enabled: true,
        content: '已收到工单，我们会尽快处理。',
      });
    }
    return response(config, {
      enabled: body.enabled,
      content: body.content,
    });
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

test('管理员能加载自动回复配置并保存开关和内容', async () => {
  render(<TicketAutomationSettings canWrite />);

  const content = await screen.findByLabelText('Automatic reply content');
  expect(content.value).toBe('已收到工单，我们会尽快处理。');

  await userEvent.clear(content);
  await userEvent.type(content, '我们已收到您的工单。');
  await userEvent.click(
    screen.getByRole('button', { name: 'Save automation settings' }),
  );

  await waitFor(() =>
    expect(
      calls.find(
        (call) =>
          call.method === 'put' &&
          call.url === '/api/ticket-admin/automation/auto-reply',
      ),
    ).toMatchObject({
      body: { enabled: true, content: '我们已收到您的工单。' },
    }),
  );
});

test('启用自动回复时拒绝空内容且不发送保存请求', async () => {
  render(<TicketAutomationSettings canWrite />);
  const content = await screen.findByLabelText('Automatic reply content');
  await userEvent.clear(content);
  await userEvent.click(
    screen.getByRole('button', { name: 'Save automation settings' }),
  );

  await screen.findByText(
    'Enter an automatic reply before enabling this rule.',
  );
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('只有读权限时展示配置但禁用编辑和保存', async () => {
  render(<TicketAutomationSettings canWrite={false} />);
  const content = await screen.findByLabelText('Automatic reply content');

  expect(content.disabled).toBe(true);
  expect(
    screen.getByRole('button', { name: 'Save automation settings' }),
  ).toHaveProperty('disabled', true);
  expect(
    screen.getByText('You have read-only access to ticket automation.'),
  ).toBeTruthy();
});

test('自动回复保存失败保留草稿且允许修正后直接重试', async () => {
  let fail = true;
  API.defaults.adapter = async (config) => {
    if (config.method === 'get')
      return response(config, { enabled: true, content: 'Received' });
    if (fail) throw new Error('Save failed');
    return response(config, JSON.parse(config.data));
  };
  render(<TicketAutomationSettings canWrite />);
  const content = await screen.findByLabelText('Automatic reply content');
  await userEvent.clear(content);
  await userEvent.type(content, 'Keep my draft');
  await userEvent.click(
    screen.getByRole('button', { name: 'Save automation settings' }),
  );
  await screen.findByRole('alert', { name: '' });
  expect(screen.getByLabelText('Automatic reply content').value).toBe(
    'Keep my draft',
  );
  fail = false;
  await userEvent.click(
    screen.getByRole('button', { name: 'Save automation settings' }),
  );
  await waitFor(() =>
    expect(screen.queryByRole('alert', { name: '' })).toBeNull(),
  );
});

test('卸载后的自动回复保存失败不污染新挂载账号', async () => {
  let finish;
  API.defaults.adapter = async (config) => {
    if (config.method === 'get')
      return response(config, { enabled: true, content: 'Received' });
    return new Promise((resolve) => {
      finish = () =>
        resolve(response(config, null, false, 'Old account failed'));
    });
  };
  const view = render(<TicketAutomationSettings key='first' canWrite />);
  await screen.findByLabelText('Automatic reply content');
  await userEvent.click(
    screen.getByRole('button', { name: 'Save automation settings' }),
  );
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  view.rerender(<TicketAutomationSettings key='second' canWrite />);
  await screen.findByLabelText('Automatic reply content');
  await act(() => finish());
  expect(screen.queryByText('Old account failed')).toBeNull();
});
