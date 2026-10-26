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
import { Link, MemoryRouter, Route, Routes } from 'react-router-dom';
import { API } from '../../../helpers/api';
import { UserContext } from '../../../context/User';
import TicketPendingProvider from '../TicketPendingProvider';
import PendingTicketsCard from '../../../components/dashboard/PendingTicketsCard';
import Tickets from '../index';

const originalAdapter = API.defaults.adapter;
let calls, total, permission, failure, user;
function response(config, data) {
  return {
    data: { success: true, data },
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  };
}
function Harness({ identity = user, active = true }) {
  return (
    <UserContext.Provider value={[{ user: identity }, () => {}]}>
      <TicketPendingProvider user={identity} active={active}>
        <input aria-label='Background input' />
        <Link to='/console/log'>Usage log</Link>
        <Link to='/console'>Dashboard</Link>
        <Routes>
          <Route
            path='/console/ticket-management'
            element={<Tickets admin />}
          />
          <Route path='*' element={<PendingTicketsCard />} />
        </Routes>
      </TicketPendingProvider>
    </UserContext.Provider>
  );
}
function mount(props, path = '/console') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Harness {...props} />
    </MemoryRouter>,
  );
}
beforeEach(() => {
  user = { id: 1, role: 10 };
  localStorage.setItem('user', JSON.stringify(user));
  calls = [];
  total = 23;
  permission = true;
  failure = false;
  API.defaults.adapter = async (config) => {
    calls.push(config);
    if (config.url === '/api/user/self') {
      return response(config, {
        permissions: {
          admin_permissions: { tickets: { read: permission, write: false } },
        },
      });
    }
    if (config.url === '/api/ticket-admin') {
      if (failure) {
        const error = new Error('Service unavailable');
        error.config = config;
        error.response = { status: failure === 'forbidden' ? 403 : 500 };
        throw error;
      }
      return response(config, {
        items: [],
        total,
        page: 1,
        page_size: config.params.page_size,
      });
    }
    if (config.url.endsWith('/auto-archive'))
      return response(config, {
        enabled: false,
        inactive_days: 30,
        statuses: ['resolved'],
      });
    if (config.url.endsWith('/auto-reply'))
      return response(config, { enabled: false, content: '' });
    throw new Error('Unexpected request: ' + config.url);
  };
});
afterEach(() => {
  API.defaults.adapter = originalAdapter;
  vi.useRealTimers();
});

test('读取分页总数，浮动提醒不抢焦点且允许继续使用后台', async () => {
  mount();
  const input = screen.getByLabelText('Background input');
  input.focus();
  const reminder = await screen.findByRole('complementary', {
    name: 'Pending ticket reminder',
  });
  expect(within(reminder).getByRole('status').textContent).toBe(
    'Tickets awaiting your response: 23',
  );
  expect(screen.getByLabelText('Pending ticket count').textContent).toBe('23');
  expect(document.activeElement).toBe(input);
  expect(screen.queryByRole('dialog')).toBeNull();
  await userEvent.type(input, 'working');
  expect(input.value).toBe('working');
  expect(calls.find((call) => call.url === '/api/ticket-admin').params).toEqual(
    { p: 1, page_size: 1, status: 'open' },
  );
});

test('关闭浮动提醒后导航及数量更新均不会重复打开', async () => {
  mount();
  await userEvent.click(
    await screen.findByRole('button', { name: 'Dismiss ticket reminder' }),
  );
  await userEvent.click(screen.getByRole('link', { name: 'Usage log' }));
  total = 24;
  await act(async () =>
    window.dispatchEvent(new Event('classic-tickets-changed')),
  );
  await waitFor(() =>
    expect(screen.getByLabelText('Pending ticket count').textContent).toBe(
      '24',
    ),
  );
  expect(screen.queryByRole('complementary')).toBeNull();
});

test('无待处理时卡片显示零且不弹提醒，新工单出现后更新提醒', async () => {
  total = 0;
  mount();
  await screen.findByText('No tickets awaiting your response');
  expect(screen.getByLabelText('Pending ticket count').textContent).toBe('0');
  expect(screen.queryByRole('complementary')).toBeNull();
  total = 1;
  await act(async () =>
    window.dispatchEvent(new Event('classic-tickets-changed')),
  );
  expect((await screen.findByRole('complementary')).textContent).toContain(
    'Tickets awaiting your response: 1',
  );
});

test.each([
  ['普通用户', { id: 2, role: 1 }, true],
  ['未登录', null, true],
  ['离开后台', { id: 1, role: 10 }, false],
])('%s 不查询管理员统计且无入口', async (_, identity, active) => {
  mount({ identity, active });
  await act(async () => {});
  expect(calls).toHaveLength(0);
  expect(screen.queryByRole('region', { name: 'Pending tickets' })).toBeNull();
  expect(screen.queryByRole('complementary')).toBeNull();
});

test('无工单读取权限不查询工单，权限撤销后清除原有统计', async () => {
  const view = mount();
  await screen.findByRole('complementary');
  permission = false;
  const before = calls.filter(
    (call) => call.url === '/api/ticket-admin',
  ).length;
  await act(async () => window.dispatchEvent(new Event('focus')));
  await waitFor(() => expect(screen.queryByRole('complementary')).toBeNull());
  expect(screen.queryByLabelText('Pending ticket count')).toBeNull();
  expect(calls.filter((call) => call.url === '/api/ticket-admin')).toHaveLength(
    before,
  );
  view.unmount();
  calls = [];
  mount();
  await waitFor(() =>
    expect(calls.some((call) => call.url === '/api/user/self')).toBe(true),
  );
  await act(async () => {});
  expect(calls.some((call) => call.url === '/api/ticket-admin')).toBe(false);
});

test('失败清除旧数量并提供重试，403 则隐藏统计', async () => {
  mount();
  await screen.findByRole('complementary');
  failure = true;
  await act(async () =>
    window.dispatchEvent(new Event('classic-tickets-changed')),
  );
  await screen.findByText('Failed to load pending tickets');
  expect(screen.queryByLabelText('Pending ticket count')).toBeNull();
  expect(screen.queryByRole('complementary')).toBeNull();
  failure = false;
  total = 5;
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
  await waitFor(() =>
    expect(screen.getByLabelText('Pending ticket count').textContent).toBe('5'),
  );
  failure = 'forbidden';
  await act(async () =>
    window.dispatchEvent(new Event('classic-tickets-changed')),
  );
  await waitFor(() =>
    expect(screen.queryByLabelText('Pending ticket count')).toBeNull(),
  );
  expect(screen.queryByRole('region', { name: 'Pending tickets' })).toBeNull();
});

test('浮窗和卡片都直达待处理筛选，管理页面隐藏浮窗', async () => {
  mount();
  const reminder = await screen.findByRole('complementary');
  const card = screen.getByRole('region', { name: 'Pending tickets' });
  expect(
    within(card)
      .getByRole('link', { name: 'View pending tickets' })
      .getAttribute('href'),
  ).toBe('/console/ticket-management?status=open');
  await userEvent.click(
    within(reminder).getByRole('link', { name: 'View pending tickets' }),
  );
  const filter = await screen.findByLabelText('Ticket status');
  expect(filter.value).toBe('open');
  await waitFor(() =>
    expect(
      calls.some(
        (call) =>
          call.url === '/api/ticket-admin' &&
          call.params.page_size === 20 &&
          call.params.status === 'open',
      ),
    ).toBe(true),
  );
  expect(screen.queryByRole('complementary')).toBeNull();
  await userEvent.selectOptions(filter, 'processing');
  await waitFor(() =>
    expect(
      calls.some(
        (call) =>
          call.url === '/api/ticket-admin' &&
          call.params.status === 'processing',
      ),
    ).toBe(true),
  );
});

test('切换账号后取消旧请求且迟到响应不泄漏旧统计', async () => {
  let finish;
  const adapter = API.defaults.adapter;
  API.defaults.adapter = async (config) => {
    if (config.url === '/api/ticket-admin') {
      return new Promise((resolve) => {
        finish = () => resolve(response(config, { total: 999, items: [] }));
      });
    }
    return adapter(config);
  };
  const view = mount();
  await waitFor(() => expect(finish).toBeTypeOf('function'));
  API.defaults.adapter = adapter;
  total = 2;
  view.rerender(
    <MemoryRouter>
      <Harness identity={{ id: 3, role: 10 }} />
    </MemoryRouter>,
  );
  await waitFor(() =>
    expect(screen.getByLabelText('Pending ticket count').textContent).toBe('2'),
  );
  await act(async () => finish());
  expect(screen.getByLabelText('Pending ticket count').textContent).toBe('2');
  view.rerender(
    <MemoryRouter>
      <Harness identity={{ id: 4, role: 1 }} />
    </MemoryRouter>,
  );
  expect(screen.queryByLabelText('Pending ticket count')).toBeNull();
});

test.each(['/api/user/self', '/api/ticket-admin'])(
  '%s 超时后清除旧数量并允许后续刷新恢复',
  async (endpoint) => {
    mount();
    await screen.findByRole('complementary');
    const adapter = API.defaults.adapter;
    let timedOut = false;
    API.defaults.adapter = async (config) => {
      if (config.url === endpoint) {
        expect(config.timeout).toBe(10000);
        timedOut = true;
        const error = new Error('timeout');
        error.code = 'ECONNABORTED';
        error.config = config;
        throw error;
      }
      return adapter(config);
    };
    await act(async () =>
      window.dispatchEvent(new Event('classic-tickets-changed')),
    );
    await screen.findByText('Failed to load pending tickets');
    expect(timedOut).toBe(true);
    expect(screen.queryByLabelText('Pending ticket count')).toBeNull();
    API.defaults.adapter = adapter;
    total = 4;
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await waitFor(() =>
      expect(screen.getByLabelText('Pending ticket count').textContent).toBe(
        '4',
      ),
    );
  },
);

test('可见页面每分钟刷新，隐藏页面暂停，卸载后停止', async () => {
  vi.useFakeTimers();
  const view = mount();
  await act(async () => {});
  total = 7;
  await act(async () => vi.advanceTimersByTimeAsync(60000));
  expect(screen.getByLabelText('Pending ticket count').textContent).toBe('7');
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
  const before = calls.length;
  await act(async () => vi.advanceTimersByTimeAsync(60000));
  expect(calls).toHaveLength(before);
  view.unmount();
  await act(async () => vi.advanceTimersByTimeAsync(60000));
  expect(calls).toHaveLength(before);
});
