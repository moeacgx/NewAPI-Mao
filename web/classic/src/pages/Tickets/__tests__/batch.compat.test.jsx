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
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import Tickets from '../index';
import { API } from '../../../helpers/api';
import { UserContext } from '../../../context/User';

const originalAdapter = API.defaults.adapter;
let calls, canWrite, failBatch, total, tickets, finishBatch;
const first = {
  id: 8,
  version: 3,
  user_id: 7,
  title: 'Payment issue',
  category: 'billing',
  priority: 'normal',
  status: 'open',
};
function response(config, data) {
  return {
    data: { success: true, data },
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  };
}
beforeEach(() => {
  calls = [];
  canWrite = true;
  failBatch = false;
  finishBatch = undefined;
  total = 2;
  tickets = [first, { ...first, id: 9, version: 5, title: 'Account issue' }];
  localStorage.setItem('user', JSON.stringify({ id: 7, role: 10 }));
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({
      url: config.url,
      method: config.method,
      body,
      params: config.params,
    });
    if (config.url === '/api/user/self')
      return response(config, {
        permissions: {
          admin_permissions: { tickets: { read: true, write: canWrite } },
        },
      });
    if (config.url.includes('/automation/auto-archive'))
      return response(config, {
        enabled: false,
        inactive_days: 30,
        statuses: ['resolved', 'closed'],
      });
    if (config.url.includes('/automation/auto-reply'))
      return response(config, { enabled: false, content: 'Received' });
    if (config.url.includes('/batch/')) {
      if (failBatch) throw new Error('Ticket changed, please refresh');
      if (finishBatch === null)
        return new Promise((resolve) => {
          finishBatch = () => resolve(response(config, { updated_count: 2 }));
        });
      tickets = [];
      total = 0;
      return response(config, { updated_count: 2, deleted_count: 2 });
    }
    if (config.url.endsWith('/status'))
      return response(config, { ...first, status: body.status });
    if (/\/8$/.test(config.url))
      return response(config, {
        ticket: tickets[0],
        messages: [],
        has_more: false,
      });
    return response(config, { items: tickets, total });
  };
});
afterEach(() => {
  API.defaults.adapter = originalAdapter;
});
function Page(props) {
  return (
    <MemoryRouter>
      <Tickets admin {...props} />
    </MemoryRouter>
  );
}

test('勾选当页后批量改状态需确认并提交每条工单的版本', async () => {
  render(<Page />);
  await userEvent.click(
    await screen.findByLabelText('Select all tickets on this page'),
  );
  expect(screen.getByText('2 selected')).toBeTruthy();
  await userEvent.selectOptions(
    screen.getByLabelText('Batch status'),
    'archived',
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Change selected status' }),
  );
  expect(calls.some((call) => call.url.includes('/batch/'))).toBe(false);
  await userEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', {
      name: 'Confirm status change',
    }),
  );
  await waitFor(() =>
    expect(
      calls.find((call) => call.url.endsWith('/batch/status')),
    ).toMatchObject({
      method: 'put',
      body: {
        tickets: [
          { id: 8, version: 3 },
          { id: 9, version: 5 },
        ],
        status: 'archived',
      },
    }),
  );
  await screen.findByText('No tickets found');
});

test('移动端可勾选工单且批量删除取消不会发送请求', async () => {
  vi.spyOn(window, 'matchMedia').mockImplementation(() => ({
    matches: true,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
  }));
  render(<Page />);
  await userEvent.click(await screen.findByLabelText('Select ticket #8'));
  expect(screen.queryByRole('table')).toBeNull();
  await userEvent.click(
    screen.getByRole('button', { name: 'Delete selected tickets' }),
  );
  await userEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }),
  );
  expect(calls.some((call) => call.url.endsWith('/batch/delete'))).toBe(false);
  await userEvent.click(
    screen.getByRole('button', { name: 'Delete selected tickets' }),
  );
  await userEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', {
      name: 'Confirm deletion',
    }),
  );
  await waitFor(() =>
    expect(
      calls.find((call) => call.url.endsWith('/batch/delete')),
    ).toMatchObject({
      method: 'post',
      body: { tickets: [{ id: 8, version: 3 }] },
    }),
  );
});

test('批量版本冲突保留选择并提示刷新，刷新后清空选择', async () => {
  failBatch = true;
  render(<Page />);
  await userEvent.click(await screen.findByLabelText('Select ticket #8'));
  await userEvent.click(
    screen.getByRole('button', { name: 'Delete selected tickets' }),
  );
  await userEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', {
      name: 'Confirm deletion',
    }),
  );
  await screen.findByText('Ticket changed, please refresh');
  expect(screen.getByText('1 selected')).toBeTruthy();
  expect(screen.getByText('Refresh the list before retrying.')).toBeTruthy();
  await userEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }),
  );
  await userEvent.click(screen.getByRole('button', { name: 'Refresh' }));
  await waitFor(() =>
    expect(screen.getByLabelText('Select ticket #8').checked).toBe(false),
  );
});

test('翻页和筛选后清空选择且只读管理页没有勾选及批量入口', async () => {
  total = 30;
  const view = render(<Page />);
  await userEvent.click(await screen.findByLabelText('Select ticket #8'));
  await userEvent.click(screen.getByRole('button', { name: 'Next' }));
  await waitFor(() =>
    expect(screen.getByLabelText('Select ticket #8').checked).toBe(false),
  );
  await userEvent.click(screen.getByLabelText('Select ticket #8'));
  await userEvent.selectOptions(
    screen.getByLabelText('Ticket status'),
    'closed',
  );
  await waitFor(() =>
    expect(screen.getByLabelText('Select ticket #8').checked).toBe(false),
  );
  view.unmount();
  canWrite = false;
  render(<Page />);
  await screen.findByRole('button', { name: 'Payment issue' });
  expect(screen.queryByLabelText('Select ticket #8')).toBeNull();
  expect(
    screen.queryByRole('button', { name: 'Delete selected tickets' }),
  ).toBeNull();
});

test('归档工单不可回复且用户重新打开后恢复回复框', async () => {
  tickets = [{ ...first, status: 'archived' }];
  render(<Page admin={false} />);
  await userEvent.click(
    await screen.findByRole('button', { name: 'Payment issue' }),
  );
  await screen.findByText('Reopen this ticket before replying.');
  expect(screen.queryByLabelText('Your reply')).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Reopen ticket' }));
  expect(await screen.findByLabelText('Your reply')).toBeTruthy();
});

test('账号切换清空批量选择并忽略旧账号完成回调', async () => {
  finishBatch = null;
  const view = render(
    <UserContext.Provider value={[{ user: { id: 7, role: 10 } }]}>
      <Page />
    </UserContext.Provider>,
  );
  await userEvent.click(await screen.findByLabelText('Select ticket #8'));
  await userEvent.click(
    screen.getByRole('button', { name: 'Change selected status' }),
  );
  await userEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', {
      name: 'Confirm status change',
    }),
  );
  await waitFor(() => expect(finishBatch).toBeTypeOf('function'));
  view.rerender(
    <UserContext.Provider value={[{ user: { id: 9, role: 10 } }]}>
      <Page />
    </UserContext.Provider>,
  );
  await screen.findByLabelText('Select ticket #8');
  const requestCount = calls.length;
  await act(() => finishBatch());
  expect(screen.queryByRole('dialog')).toBeNull();
  expect(screen.getByLabelText('Select ticket #8').checked).toBe(false);
  expect(calls).toHaveLength(requestCount);
});
