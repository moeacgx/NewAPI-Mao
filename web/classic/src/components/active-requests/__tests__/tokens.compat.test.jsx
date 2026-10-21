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
import {
  act,
  render,
  renderHook,
  screen,
  waitFor,
} from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { API } from '../../../helpers/api';
import { useTokensData } from '../../../hooks/tokens/useTokensData';
import TokensTable from '../../table/tokens/TokensTable';
import TokenActivityBar from '../TokenActivityBar';
import ActiveRequestsDrawer from '../ActiveRequestsDrawer';

test('相同令牌名称按稳定 ID 显示各自数量，活动令牌数与请求数分开', async () => {
  const activity = {
    snapshot: {
      summary: { total: 3, active_tokens: 2, by_token: { 31: 2, 32: 1 } },
      scope: 'cluster',
    },
    error: false,
  };
  render(
    <>
      <TokenActivityBar activity={activity} />
      <TokensTable
        tokens={[
          {
            id: 31,
            name: '同名令牌',
            status: 1,
            key: 'masked',
            group: '',
            unlimited_quota: true,
          },
          {
            id: 32,
            name: '同名令牌',
            status: 1,
            key: 'masked',
            group: '',
            unlimited_quota: true,
          },
        ]}
        selectedKeys={[]}
        showKeys={{}}
        loadingTokenKeys={{}}
        resolvedTokenKeys={{}}
        setSelectedKeys={vi.fn()}
        activity={activity}
        t={(key) => key}
      />
    </>,
  );
  expect(screen.getByText('2 tokens in use')).toBeTruthy();
  expect(screen.getByText('3 active requests')).toBeTruthy();
  expect(screen.getByText('2 active requests')).toBeTruthy();
  expect(screen.getByText('1 active requests')).toBeTruthy();
});

test('正在使用筛选交给后端与原有筛选分页组合，活动消失时纠正越界页', async () => {
  const requests = [];
  vi.spyOn(API, 'get').mockImplementation((url) => {
    requests.push(url);
    if (url === '/api/log/self/active')
      return Promise.resolve({
        data: {
          success: true,
          data: {
            summary: { by_token: { 41: 1 } },
            items: [],
            observed_at: Date.now(),
          },
        },
      });
    if (url.startsWith('/api/user/'))
      return Promise.resolve({ data: { success: true, data: {} } });
    const filtered = url.includes('active_only=true');
    return Promise.resolve({
      data: {
        success: true,
        data: {
          items: [],
          total: filtered ? 0 : 100,
          page: url.includes('p=4') ? 4 : 1,
          page_size: 10,
        },
      },
    });
  });
  const hook = renderHook(() => useTokensData());
  await waitFor(() => expect(hook.result.current.loading).toBe(false));
  act(() =>
    hook.result.current.setFormApi({
      getValues: () => ({ searchKeyword: 'Claude', searchStatus: 1 }),
    }),
  );
  await act(async () => hook.result.current.setActiveFilter(true));
  expect(
    requests.some(
      (url) =>
        url.includes('keyword=Claude') &&
        url.includes('status=1') &&
        url.includes('active_only=true') &&
        url.includes('p=1'),
    ),
  ).toBe(true);
  await act(async () => hook.result.current.handlePageChange(4));
  await waitFor(() => expect(hook.result.current.activePage).toBe(1));
});

test('点击行内活动数量按稳定令牌 ID 打开连接列表', async () => {
  vi.spyOn(API, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        items: [],
        total: 0,
        observed_at: Date.now(),
        summary: { total: 3, active_tokens: 2 },
      },
    },
  });
  const { TokenActivityBadge } = await import('../TokenActivityBar');
  render(<TokenActivityBadge tokenId={31} tokenName='相同名称' count={2} />);
  await userEvent.click(
    screen.getByRole('button', { name: 'View connection details: 相同名称' }),
  );
  await screen.findByRole('dialog');
  await waitFor(() =>
    expect(
      API.get.mock.calls.some(
        ([, config]) =>
          config.params.token_id === 31 && config.params.page_size === 10,
      ),
    ).toBe(true),
  );
});

test('抽屉切换令牌后清空搜索并重置分页，全用户摘要保持独立', async () => {
  vi.spyOn(API, 'get').mockImplementation((_url, config) =>
    Promise.resolve({
      data: {
        success: true,
        data: {
          items: [],
          total: 24,
          page: config.params.page,
          observed_at: Date.now(),
          summary: { total: 99, active_tokens: 12 },
        },
      },
    }),
  );
  const view = render(
    <ActiveRequestsDrawer visible tokenId={31} onClose={vi.fn()} />,
  );
  await screen.findByText('99 active requests · 12 tokens');
  await userEvent.click(
    screen.getByRole('button', { name: 'Next connections page' }),
  );
  await userEvent.type(
    screen.getByRole('textbox', { name: 'Search connections' }),
    'Claude',
  );
  await waitFor(() =>
    expect(
      API.get.mock.calls.some(
        ([, config]) => config.params.search === 'Claude',
      ),
    ).toBe(true),
  );
  view.rerender(
    <ActiveRequestsDrawer visible tokenId={32} onClose={vi.fn()} />,
  );
  expect(
    screen.getByRole('textbox', { name: 'Search connections' }).value,
  ).toBe('');
  await waitFor(() =>
    expect(
      API.get.mock.calls.some(
        ([, config]) =>
          config.params.token_id === 32 &&
          config.params.page === 1 &&
          config.params.search === '',
      ),
    ).toBe(true),
  );
});

test('活动接口错误恢复后即使令牌集合相同也刷新一次列表，正常轮询不重复搜索', async () => {
  let unavailable = false;
  let tokenSearches = 0;
  vi.spyOn(API, 'get').mockImplementation((url) => {
    if (url === '/api/log/self/active') {
      if (unavailable) return Promise.reject(new Error('offline'));
      return Promise.resolve({
        data: {
          success: true,
          data: {
            items: [],
            observed_at: Date.now(),
            summary: { total: 1, by_token: { 41: 1 } },
          },
        },
      });
    }
    if (url.includes('active_only=true')) tokenSearches++;
    if (url.startsWith('/api/user/'))
      return Promise.resolve({ data: { success: true, data: {} } });
    return Promise.resolve({
      data: {
        success: true,
        data: { items: [], total: 1, page: 1, page_size: 10 },
      },
    });
  });
  const hook = renderHook(() => useTokensData());
  await waitFor(() =>
    expect(hook.result.current.activity.snapshot?.summary.total).toBe(1),
  );
  await act(async () => hook.result.current.setActiveFilter(true));
  const baseline = tokenSearches;
  await act(async () => hook.result.current.activity.refresh());
  expect(tokenSearches).toBe(baseline);
  unavailable = true;
  await act(async () => hook.result.current.activity.refresh());
  await waitFor(() => expect(hook.result.current.activity.error).toBe(true));
  unavailable = false;
  await act(async () => hook.result.current.activity.refresh());
  await waitFor(() => expect(tokenSearches).toBe(baseline + 1));
});
