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
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { UserContext } from '../../../context/User';
import { API } from '../../../helpers/api';
import Tickets from '../index';
import TicketDetails from '../TicketDetails';
import TicketFloatingButton from '../TicketFloatingButton';

const originalAdapter = API.defaults.adapter;
const ticket = {
  id: 8,
  user_id: 1,
  title: 'First account ticket',
  category: 'general',
  priority: 'normal',
  status: 'open',
  created_at: 1792600000,
  updated_at: 1792600000,
};
function response(config, data, success = true, message = '') {
  return {
    data: { success, data, message },
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  };
}
function CurrentRoute() {
  const location = useLocation();
  return (
    <output aria-label='Current route'>
      {location.pathname}
      {location.search}
    </output>
  );
}
beforeEach(() => {
  localStorage.setItem('user', JSON.stringify({ id: 1, role: 10 }));
});
afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

test('回复保存期间不能用关闭按钮或 Escape 退出，失败后保留草稿并恢复关闭', async () => {
  let finish;
  const onClose = vi.fn();
  const onChanged = vi.fn();
  API.defaults.adapter = async (config) => {
    if (config.method === 'post')
      return new Promise((resolve) => {
        finish = () =>
          resolve(
            response(config, null, false, 'Ticket changed, please refresh'),
          );
      });
    return response(config, { ticket, messages: [], has_more: false });
  };
  render(
    <TicketDetails
      ticketId={8}
      canWrite
      onClose={onClose}
      onChanged={onChanged}
    />,
  );
  await userEvent.type(
    await screen.findByLabelText('Your reply'),
    'Draft must survive',
  );
  await userEvent.click(screen.getByRole('button', { name: 'Send reply' }));
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  expect(screen.queryByRole('button', { name: 'close' })).toBeNull();
  await userEvent.keyboard('{Escape}');
  expect(onClose).not.toHaveBeenCalled();
  await act(() => finish());
  await screen.findByText('Ticket changed, please refresh');
  expect(screen.getByLabelText('Your reply').value).toBe('Draft must survive');
  expect(onChanged).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole('button', { name: 'close' }));
  expect(onClose).toHaveBeenCalledOnce();
});

test('回复保存成功后仍刷新列表并显示服务器消息，不因保存中关闭尝试丢失结果', async () => {
  let finish;
  const onClose = vi.fn();
  const onChanged = vi.fn();
  API.defaults.adapter = async (config) => {
    if (config.method === 'post')
      return new Promise((resolve) => {
        finish = () =>
          resolve(
            response(config, {
              id: 2,
              ticket_id: 8,
              author_id: 1,
              author_role: 'user',
              content: 'Saved reply',
            }),
          );
      });
    return response(config, { ticket, messages: [], has_more: false });
  };
  render(
    <TicketDetails
      ticketId={8}
      canWrite
      onClose={onClose}
      onChanged={onChanged}
    />,
  );
  await userEvent.type(
    await screen.findByLabelText('Your reply'),
    'Saved reply',
  );
  await userEvent.click(screen.getByRole('button', { name: 'Send reply' }));
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  await userEvent.keyboard('{Escape}');
  expect(onClose).not.toHaveBeenCalled();
  await act(() => finish());
  await screen.findByText('Saved reply');
  expect(onChanged).toHaveBeenCalledOnce();
  expect(screen.getByLabelText('Your reply').value).toBe('');
});

test('浮窗切换账号清空草稿，旧账号延迟创建结果不会导航新账号', async () => {
  let finish;
  API.defaults.adapter = async (config) =>
    new Promise((resolve) => {
      finish = () => resolve(response(config, { ...ticket, id: 99 }));
    });
  const view = render(
    <MemoryRouter initialEntries={['/console']}>
      <TicketFloatingButton user={{ id: 1 }} />
      <CurrentRoute />
    </MemoryRouter>,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Create support ticket' }),
  );
  await userEvent.type(
    screen.getByLabelText('Ticket title'),
    'Private account A',
  );
  await userEvent.type(
    screen.getByLabelText('Describe your issue'),
    'Private draft A',
  );
  await userEvent.click(screen.getByRole('button', { name: 'Submit ticket' }));
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  view.rerender(
    <MemoryRouter initialEntries={['/console']}>
      <TicketFloatingButton user={{ id: 2 }} />
      <CurrentRoute />
    </MemoryRouter>,
  );
  expect(screen.queryByLabelText('Ticket title')).toBeNull();
  await userEvent.click(
    screen.getByRole('button', { name: 'Create support ticket' }),
  );
  expect(screen.getByLabelText('Ticket title').value).toBe('');
  expect(screen.getByLabelText('Describe your issue').value).toBe('');
  await act(() => finish());
  expect(screen.getByLabelText('Current route').textContent).toBe('/console');
  expect(screen.getByLabelText('Ticket title').value).toBe('');
});

test('账号切换重新获取列表且忽略旧账号尚未完成的列表响应', async () => {
  let finish;
  let userId = 1;
  API.defaults.adapter = async (config) => {
    if (userId === 1)
      return new Promise((resolve) => {
        finish = () => resolve(response(config, { items: [ticket], total: 1 }));
      });
    return response(config, {
      items: [{ ...ticket, id: 9, user_id: 2, title: 'Second account ticket' }],
      total: 1,
    });
  };
  const view = render(
    <MemoryRouter>
      <UserContext.Provider value={[{ user: { id: 1, role: 10 } }]}>
        <Tickets />
      </UserContext.Provider>
    </MemoryRouter>,
  );
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  expect(screen.getByRole('status').textContent).toContain('Loading tickets');
  userId = 2;
  view.rerender(
    <MemoryRouter>
      <UserContext.Provider value={[{ user: { id: 2, role: 10 } }]}>
        <Tickets />
      </UserContext.Provider>
    </MemoryRouter>,
  );
  await screen.findByRole('button', { name: 'Second account ticket' });
  await act(() => finish());
  expect(
    screen.queryByRole('button', { name: 'First account ticket' }),
  ).toBeNull();
});

test('账号切换后旧详情的延迟消息不会出现在无权访问的新账号中', async () => {
  let finish;
  let userId = 1;
  API.defaults.adapter = async (config) => {
    if (config.url === '/api/tickets')
      return response(config, { items: [], total: 0 });
    if (userId === 1)
      return new Promise((resolve) => {
        finish = () =>
          resolve(
            response(config, {
              ticket,
              messages: [
                {
                  id: 1,
                  content: 'Private account A conversation',
                  author_role: 'user',
                  author_id: 1,
                },
              ],
              has_more: false,
            }),
          );
      });
    return response(config, null, false, 'Ticket not found');
  };
  const view = render(
    <MemoryRouter initialEntries={['/console/tickets?ticket=8']}>
      <UserContext.Provider value={[{ user: { id: 1, role: 10 } }]}>
        <Tickets />
      </UserContext.Provider>
    </MemoryRouter>,
  );
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  userId = 2;
  view.rerender(
    <MemoryRouter initialEntries={['/console/tickets?ticket=8']}>
      <UserContext.Provider value={[{ user: { id: 2, role: 10 } }]}>
        <Tickets />
      </UserContext.Provider>
    </MemoryRouter>,
  );
  await screen.findByText('Ticket not found');
  await act(() => finish());
  expect(screen.queryByText('Private account A conversation')).toBeNull();
});

test('账号切换后旧权限响应不能恢复新账号被拒绝的工单管理访问', async () => {
  let finish;
  let userId = 1;
  const requests = [];
  API.defaults.adapter = async (config) => {
    requests.push(config.url);
    if (userId === 1)
      return new Promise((resolve) => {
        finish = () =>
          resolve(
            response(config, {
              permissions: {
                admin_permissions: { tickets: { read: true, write: true } },
              },
            }),
          );
      });
    return response(config, {
      permissions: {
        admin_permissions: { tickets: { read: false, write: false } },
      },
    });
  };
  const view = render(
    <MemoryRouter>
      <UserContext.Provider value={[{ user: { id: 1, role: 10 } }]}>
        <Tickets admin />
      </UserContext.Provider>
    </MemoryRouter>,
  );
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  userId = 2;
  view.rerender(
    <MemoryRouter>
      <UserContext.Provider value={[{ user: { id: 2, role: 10 } }]}>
        <Tickets admin />
      </UserContext.Provider>
    </MemoryRouter>,
  );
  await screen.findByText(
    'You do not have permission to view ticket management.',
  );
  await act(() => finish());
  expect(requests).not.toContain('/api/ticket-admin');
  expect(
    screen.getByText('You do not have permission to view ticket management.'),
  ).toBeTruthy();
});
