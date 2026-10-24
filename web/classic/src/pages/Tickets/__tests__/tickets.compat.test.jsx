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
import { beforeEach, afterEach, expect, test, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import Tickets from '../index';
import TicketFloatingButton from '../TicketFloatingButton';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
let calls,
  ticket,
  messages,
  failCreate,
  failList,
  failReply,
  denyMutation,
  emptyList,
  listTotal,
  allowed,
  older;
beforeEach(() => {
  calls = [];
  failCreate = failList = false;
  failReply = denyMutation = emptyList = false;
  listTotal = 1;
  allowed = { read: true, write: true };
  older = true;
  ticket = {
    id: 8,
    user_id: 7,
    title: 'Payment issue',
    category: 'billing',
    priority: 'normal',
    status: 'open',
    created_at: 1792600000,
    updated_at: 1792600000,
  };
  messages = [
    {
      id: 51,
      ticket_id: 8,
      author_id: 7,
      author_role: 'user',
      content: '<script>private details</script>',
      created_at: 1792600000,
    },
  ];
  localStorage.setItem('user', JSON.stringify({ id: 7, role: 10 }));
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({
      url: config.url,
      method: config.method,
      body,
      params: config.params,
    });
    if (denyMutation && config.method !== 'get') {
      const error = new Error('Permission denied');
      error.response = { status: 403, data: { message: 'Permission denied' } };
      error.config = config;
      throw error;
    }
    let data;
    if (config.url === '/api/user/self')
      data = { permissions: { admin_permissions: { tickets: allowed } } };
    else if (config.url === '/api/ticket-admin/automation/auto-archive')
      data = {
        enabled: false,
        inactive_days: 30,
        statuses: ['resolved', 'closed'],
      };
    else if (config.url === '/api/ticket-admin/automation/auto-reply')
      data = { enabled: false, content: 'Received' };
    else if (config.method === 'post' && config.url.endsWith('/messages')) {
      if (failReply) throw new Error('Ticket changed, please refresh');
      const message = {
        ...messages[0],
        id: 52,
        content: body.content,
        author_role: config.url.includes('ticket-admin') ? 'admin' : 'user',
      };
      messages = [...messages, message];
      ticket.status = message.author_role === 'admin' ? 'processing' : 'open';
      data = message;
    } else if (config.method === 'put') {
      ticket = { ...ticket, status: body.status };
      data = ticket;
    } else if (config.method === 'post') {
      if (failCreate)
        return {
          data: { success: false, message: 'Please try again' },
          status: 400,
          statusText: 'Bad Request',
          headers: {},
          config,
        };
      data = { ...ticket, ...body, id: 9 };
    } else if (['/api/tickets', '/api/ticket-admin'].includes(config.url)) {
      if (failList) throw new Error('Service unavailable');
      data = {
        items: emptyList ? [] : [ticket],
        total: emptyList ? 0 : listTotal,
        page: 1,
        page_size: 20,
      };
    } else if (config.params?.before_message_id) {
      data = {
        ticket,
        messages: [{ ...messages[0], id: 1, content: 'Earlier conversation' }],
        has_more: false,
      };
    } else data = { ticket, messages, has_more: older };
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

test('管理员个人中心仍使用本人接口，详情正文按纯文本显示并能加载更早消息', async () => {
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Payment issue' }),
  );
  await screen.findByText('<script>private details</script>');
  expect(document.querySelector('script')).toBeNull();
  await userEvent.click(
    screen.getByRole('button', { name: 'Load earlier messages' }),
  );
  await screen.findByText('Earlier conversation');
  expect(calls.some((call) => call.url.includes('ticket-admin'))).toBe(false);
  expect(calls.some((call) => call.params?.before_message_id === 51)).toBe(
    true,
  );
});

test('创建失败保留草稿，修正后按个人接口提交且成功关闭弹窗', async () => {
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await screen.findByRole('button', { name: 'Payment issue' });
  await userEvent.click(screen.getByRole('button', { name: 'Create ticket' }));
  const title = screen.getByLabelText('Ticket title');
  const content = screen.getByLabelText('Describe your issue');
  await userEvent.type(title, 'API question');
  await userEvent.type(content, 'Please investigate my request.');
  failCreate = true;
  await userEvent.click(screen.getByRole('button', { name: 'Submit ticket' }));
  await screen.findByText('Please try again');
  expect(title.value).toBe('API question');
  expect(content.value).toBe('Please investigate my request.');
  failCreate = false;
  await userEvent.click(screen.getByRole('button', { name: 'Submit ticket' }));
  await waitFor(() =>
    expect(screen.queryByRole('button', { name: 'Submit ticket' })).toBeNull(),
  );
  expect(
    calls.some(
      (call) =>
        call.url === '/api/tickets' &&
        call.method === 'post' &&
        call.body.category === 'general' &&
        call.body.priority === 'normal',
    ),
  ).toBe(true);
});

test('已关闭工单不能回复，重新打开后提交回复', async () => {
  ticket.status = 'closed';
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Payment issue' }),
  );
  await screen.findByText('Reopen this ticket before replying.');
  expect(screen.queryByRole('button', { name: 'Send reply' })).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Reopen ticket' }));
  const input = await screen.findByLabelText('Your reply');
  await userEvent.type(input, 'More details');
  await userEvent.click(screen.getByRole('button', { name: 'Send reply' }));
  await screen.findByText('More details');
  expect(
    calls.some((call) => call.method === 'put' && call.body.status === 'open'),
  ).toBe(true);
});

test('管理只读权限允许查看但没有回复和状态写入口', async () => {
  allowed.write = false;
  render(
    <MemoryRouter>
      <Tickets admin />
    </MemoryRouter>,
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Payment issue' }),
  );
  await screen.findByText('<script>private details</script>');
  expect(screen.queryByLabelText('Your reply')).toBeNull();
  expect(screen.queryByRole('button', { name: 'Save status' })).toBeNull();
  expect(screen.queryByRole('button', { name: 'Create ticket' })).toBeNull();
  expect(calls.some((call) => call.url === '/api/ticket-admin')).toBe(true);
});

test('管理读取被拒绝时不请求工单数据', async () => {
  allowed.read = false;
  render(
    <MemoryRouter>
      <Tickets admin />
    </MemoryRouter>,
  );
  await screen.findByText(
    'You do not have permission to view ticket management.',
  );
  expect(calls.some((call) => call.url.startsWith('/api/ticket-admin'))).toBe(
    false,
  );
});

test('列表请求失败提供重试，重试成功恢复工单', async () => {
  failList = true;
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await screen.findByRole('alert');
  failList = false;
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
  await screen.findByRole('button', { name: 'Payment issue' });
});

test('控制台浮动按钮打开真实表单，公共页面不展示入口', async () => {
  const view = render(
    <MemoryRouter initialEntries={['/console/token']}>
      <TicketFloatingButton user={{ id: 7 }} />
    </MemoryRouter>,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Create support ticket' }),
  );
  expect(await screen.findByLabelText('Ticket title')).toBeTruthy();
  view.unmount();
  render(
    <MemoryRouter initialEntries={['/pricing']}>
      <TicketFloatingButton user={{ id: 7 }} />
    </MemoryRouter>,
  );
  expect(
    screen.queryByRole('button', { name: 'Create support ticket' }),
  ).toBeNull();
});

test('用户搜索和分页保持筛选值，切换状态回到第一页', async () => {
  listTotal = 21;
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await screen.findByRole('button', { name: 'Payment issue' });
  await userEvent.type(
    screen.getByLabelText('Search tickets'),
    'Payment{Enter}',
  );
  await waitFor(() =>
    expect(calls.some((call) => call.params?.keyword === 'Payment')).toBe(true),
  );
  await userEvent.click(screen.getByRole('button', { name: 'Next' }));
  await waitFor(() =>
    expect(
      calls.some(
        (call) => call.params?.p === 2 && call.params.keyword === 'Payment',
      ),
    ).toBe(true),
  );
  await userEvent.selectOptions(
    screen.getByLabelText('Ticket status'),
    'resolved',
  );
  await waitFor(() =>
    expect(calls.at(-1).params).toMatchObject({
      p: 1,
      status: 'resolved',
      keyword: 'Payment',
    }),
  );
});

test('没有工单时展示空状态并禁用翻页', async () => {
  emptyList = true;
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await screen.findByText('No tickets found');
  expect(
    screen.getByRole('button', { name: 'Next' }).getAttribute('aria-disabled'),
  ).toBe('true');
  expect(
    screen
      .getByRole('button', { name: 'Previous' })
      .getAttribute('aria-disabled'),
  ).toBe('true');
});

test('标题 Unicode 字符超限时保留输入并阻止创建请求', async () => {
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await userEvent.click(screen.getByRole('button', { name: 'Create ticket' }));
  const title = screen.getByLabelText('Ticket title');
  await userEvent.click(title);
  await userEvent.paste('问'.repeat(121));
  await userEvent.type(
    screen.getByLabelText('Describe your issue'),
    'A question',
  );
  await userEvent.click(screen.getByRole('button', { name: 'Submit ticket' }));
  await screen.findByText(
    'Enter a title of 1–120 characters and a message of 1–10000 characters.',
  );
  expect(title.value.length).toBe(121);
  expect(calls.some((call) => call.method === 'post')).toBe(false);
});

test('回复失败保留正文且不会显示伪造的新消息', async () => {
  failReply = true;
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Payment issue' }),
  );
  const reply = await screen.findByLabelText('Your reply');
  await userEvent.type(reply, 'Keep this draft');
  await userEvent.click(screen.getByRole('button', { name: 'Send reply' }));
  await screen.findByText('Ticket changed, please refresh');
  expect(reply.value).toBe('Keep this draft');
  expect(
    within(screen.getByLabelText('Ticket conversation')).queryByText(
      'Keep this draft',
    ),
  ).toBeNull();
});

test('管理员能按用户筛选并通过管理接口改状态', async () => {
  render(
    <MemoryRouter>
      <Tickets admin />
    </MemoryRouter>,
  );
  await screen.findByRole('button', { name: 'Payment issue' });
  await userEvent.type(screen.getByLabelText('User ID'), '7');
  await userEvent.click(screen.getByRole('button', { name: 'Search' }));
  await waitFor(() =>
    expect(calls.some((call) => call.params?.user_id === '7')).toBe(true),
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Payment issue' }),
  );
  await screen.findByRole('button', { name: 'Save status' });
  await userEvent.selectOptions(
    screen.getAllByLabelText('Ticket status').at(-1),
    'resolved',
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save status' }));
  await waitFor(() =>
    expect(
      calls.some(
        (call) =>
          call.url === '/api/ticket-admin/8/status' &&
          call.body.status === 'resolved',
      ),
    ).toBe(true),
  );
});

test('管理员写权限被回收后保留只读内容并移除写入口', async () => {
  render(
    <MemoryRouter>
      <Tickets admin />
    </MemoryRouter>,
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Payment issue' }),
  );
  await userEvent.type(
    await screen.findByLabelText('Your reply'),
    'Support reply',
  );
  denyMutation = true;
  await userEvent.click(screen.getByRole('button', { name: 'Send reply' }));
  await screen.findByText('You have read-only access to this ticket.');
  expect(screen.queryByRole('button', { name: 'Save status' })).toBeNull();
  expect(screen.queryByRole('button', { name: 'Send reply' })).toBeNull();
  expect(screen.getByText('<script>private details</script>')).toBeTruthy();
});

test('移动端使用可换行卡片并保留详情操作', async () => {
  vi.spyOn(window, 'matchMedia').mockImplementation(() => ({
    matches: true,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
  }));
  ticket.title = 'Long support ticket title '.repeat(4);
  render(
    <MemoryRouter>
      <Tickets />
    </MemoryRouter>,
  );
  await screen.findByRole('button', { name: ticket.title.trim() });
  expect(screen.queryByRole('table')).toBeNull();
  await userEvent.click(
    screen.getByRole('button', { name: ticket.title.trim() }),
  );
  expect(await screen.findByLabelText('Your reply')).toBeTruthy();
});

test('未登录的控制台不展示浮动创建入口', () => {
  render(
    <MemoryRouter initialEntries={['/console']}>
      <TicketFloatingButton />
    </MemoryRouter>,
  );
  expect(
    screen.queryByRole('button', { name: 'Create support ticket' }),
  ).toBeNull();
});
