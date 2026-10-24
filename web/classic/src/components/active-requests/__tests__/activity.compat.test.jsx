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
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, test, expect, vi } from 'vitest';
import { API } from '../../../helpers/api';
import ActiveRequestsPanel from '../ActiveRequestsPanel';
import ActiveRequestCard from '../ActiveRequestCard';
import { useActiveRequests } from '../useActiveRequests';
import { renderHook } from '@testing-library/react';

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

const entries = Array.from({ length: 24 }, (_, i) => ({
  id: 'a' + i,
  request_id: 'req' + i,
  token_id: (i % 2) + 1,
  token_name: 'Key ' + (i % 2),
  model_name: 'model-' + i,
  request_type: 'claude',
  is_stream: true,
  started_at: 1000,
  first_response_ms: i ? 1200 : null,
  status: i ? 'receiving' : 'waiting_response',
}));
function response(items = entries.slice(0, 3), total = 24) {
  return {
    data: {
      success: true,
      data: {
        items,
        total,
        page: 1,
        page_size: 3,
        observed_at: 10000,
        scope: 'cluster',
        summary: {
          total: 24,
          streaming: 24,
          non_streaming: 0,
          waiting: 1,
          active_tokens: 2,
          by_token: { 1: 12, 2: 12 },
        },
      },
    },
  };
}

test('24 条连接桌面只展示 3 张并能折叠，零连接时隐藏', async () => {
  vi.spyOn(API, 'get').mockResolvedValue(response());
  const view = render(<ActiveRequestsPanel />);
  await screen.findByText('model-0');
  expect(screen.getAllByRole('article')).toHaveLength(3);
  expect(screen.getByText('21', { exact: false })).toBeTruthy();
  const collapse = screen.getByRole('button', { name: 'Collapse connections' });
  await userEvent.click(collapse);
  expect(screen.queryAllByRole('article')).toHaveLength(0);
  expect(collapse.getAttribute('aria-expanded')).toBe('false');
  view.unmount();
  API.get.mockResolvedValue({
    data: {
      success: true,
      data: {
        ...response().data.data,
        total: 0,
        items: [],
        summary: { total: 0 },
      },
    },
  });
  await act(async () => render(<ActiveRequestsPanel />));
  expect(
    screen.queryByRole('region', { name: 'Current connections' }),
  ).toBeNull();
});

test('手机展示最多 3 行，抽屉按 10 条分页且搜索后回第一页', async () => {
  vi.spyOn(window, 'matchMedia').mockReturnValue({
    matches: true,
    addEventListener() {},
    removeEventListener() {},
  });
  vi.spyOn(API, 'get').mockImplementation((url, config) => {
    const p = config.params;
    const result = response(
      entries.slice((p.page - 1) * p.page_size, p.page * p.page_size),
    );
    result.data.data.page = p.page;
    result.data.data.page_size = p.page_size;
    return Promise.resolve(result);
  });
  render(<ActiveRequestsPanel />);
  await screen.findByText('model-0');
  expect(screen.getAllByRole('article')).toHaveLength(3);
  expect(screen.getAllByRole('button', { expanded: false })).toHaveLength(3);
  await userEvent.click(
    screen.getByRole('button', { name: 'View all connections' }),
  );
  const drawer = await screen.findByRole('dialog');
  await waitFor(() =>
    expect(within(drawer).getAllByRole('article')).toHaveLength(10),
  );
  await userEvent.click(
    within(drawer).getByRole('button', { name: 'Next connections page' }),
  );
  await within(drawer).findByText('model-10');
  await userEvent.type(
    within(drawer).getByRole('textbox', { name: 'Search connections' }),
    'Claude',
  );
  await waitFor(() =>
    expect(
      API.get.mock.calls.some(
        ([, c]) => c.params.search === 'Claude' && c.params.page === 1,
      ),
    ).toBe(true),
  );
});

test('请求失败显示错误而非零连接', async () => {
  vi.spyOn(API, 'get').mockRejectedValue(new Error('offline'));
  render(<ActiveRequestsPanel />);
  expect(await screen.findByRole('alert')).toBeTruthy();
  expect(screen.queryByText('0 active requests')).toBeNull();
});

test('仅剩已结束连接时仍展示保留卡片并按列表总数计算更多连接', async () => {
  const result = response(
    entries.slice(0, 3).map((entry) => ({
      ...entry,
      status: 'completed',
      completed_at: 9000,
    })),
    5,
  );
  result.data.data.summary = {
    total: 0,
    streaming: 0,
    non_streaming: 0,
    waiting: 0,
    active_tokens: 0,
    by_token: {},
  };
  vi.spyOn(API, 'get').mockResolvedValue(result);
  render(<ActiveRequestsPanel />);
  await screen.findByRole('article', { name: 'model-0' });
  expect(screen.getAllByRole('article')).toHaveLength(3);
  expect(screen.getByText('2 more connections')).toBeTruthy();
  expect(
    screen.getByText('Ended requests remain visible for 5 seconds.'),
  ).toBeTruthy();
  expect(screen.queryByText('Showing your in-progress requests')).toBeNull();
  await userEvent.click(
    screen.getByRole('button', { name: 'View all connections' }),
  );
  const drawer = await screen.findByRole('dialog');
  await within(drawer).findByRole('article', { name: 'model-0' });
  expect(within(drawer).getByText('0 active requests · 0 tokens')).toBeTruthy();
  expect(
    within(drawer).getByText('Ended requests remain visible for 5 seconds.'),
  ).toBeTruthy();
});

test('服务端时间校准运行时长，未响应保持等待而非 0 秒', async () => {
  vi.spyOn(API, 'get').mockResolvedValue(response());
  render(<ActiveRequestsPanel />);
  await screen.findByText('model-0');
  expect(screen.getAllByRole('definition', { name: '9.0 s' })).toHaveLength(3);
  expect(screen.getAllByText('Waiting for response')).toHaveLength(3);
});

test('页面隐藏暂停轮询，恢复刷新，卸载取消在途请求', async () => {
  vi.useFakeTimers();
  let visibility = 'visible';
  vi.spyOn(document, 'visibilityState', 'get').mockImplementation(
    () => visibility,
  );
  const pending = [];
  vi.spyOn(API, 'get').mockImplementation(
    (url, config) =>
      new Promise((resolve) =>
        pending.push({ resolve, signal: config.signal }),
      ),
  );
  const hook = renderHook(() => useActiveRequests({ pageSize: 3 }));
  expect(pending).toHaveLength(1);
  act(() => {
    visibility = 'hidden';
    document.dispatchEvent(new Event('visibilitychange'));
    vi.advanceTimersByTime(6000);
  });
  expect(pending).toHaveLength(1);
  expect(pending[0].signal.aborted).toBe(true);
  act(() => {
    visibility = 'visible';
    document.dispatchEvent(new Event('visibilitychange'));
  });
  expect(pending).toHaveLength(2);
  hook.unmount();
  expect(pending[1].signal.aborted).toBe(true);
  vi.useRealTimers();
});

test('刷新失败保留上次连接，标记过期，重试成功后恢复', async () => {
  vi.spyOn(API, 'get').mockResolvedValue(response());
  const hook = renderHook(() => useActiveRequests());
  await waitFor(() =>
    expect(hook.result.current.snapshot?.summary.total).toBe(24),
  );
  API.get.mockRejectedValue(new Error('unavailable'));
  await act(async () => hook.result.current.refresh());
  await waitFor(() => expect(hook.result.current.error).toBe(true));
  expect(hook.result.current.snapshot.items[0].id).toBe('a0');
  API.get.mockResolvedValue(response());
  await act(async () => hook.result.current.refresh());
  await waitFor(() => expect(hook.result.current.error).toBe(false));
});

test('下一轮查询变慢时隐藏实时标记，快照过期后提示并在刷新成功后恢复', async () => {
  let resolveRefresh;
  vi.spyOn(API, 'get')
    .mockResolvedValueOnce(response())
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveRefresh = resolve;
        }),
    )
    .mockResolvedValue(response());
  render(<ActiveRequestsPanel subject={{ userId: '5215' }} />);
  await screen.findByText('Live updates');
  await waitFor(() => expect(API.get).toHaveBeenCalledTimes(2), {
    timeout: 4000,
  });
  expect(screen.queryByText('Live updates')).toBeNull();

  const now = Date.now();
  vi.spyOn(Date, 'now').mockReturnValue(now + 6000);
  await screen.findByText(
    'Connection data is delayed. Showing the last known status.',
  );
  expect(screen.getByRole('status').textContent).toContain(
    'Connection data is delayed. Showing the last known status.',
  );

  await act(async () => {
    resolveRefresh(response());
  });
  expect(screen.queryByRole('alert')).toBeNull();
  await waitFor(() => expect(screen.getByText('Live updates')).toBeTruthy());
});

test('实例范围明确显示提示且超长模型名保留完整可访问名称', async () => {
  const item = {
    ...entries[0],
    model_name: 'provider/model-with-a-very-long-deployment-name-2026-10-21',
  };
  const result = response([item], 1);
  result.data.data.scope = 'instance';
  vi.spyOn(API, 'get').mockResolvedValue(result);
  render(<ActiveRequestsPanel />);
  await screen.findByRole('article', { name: item.model_name });
  expect(
    screen.getByText('Only connections on the current instance are shown.'),
  ).toBeTruthy();
});

test.each([
  ['openai', 'OpenAI Chat Completions'],
  ['claude', 'Claude Messages'],
  ['gemini', 'Gemini'],
  ['openai_responses', 'OpenAI Responses'],
  ['openai_responses_compaction', 'OpenAI Responses Compaction'],
  ['openai_alpha_search', 'OpenAI Alpha Search'],
  ['openai_audio', 'OpenAI Audio'],
  ['openai_image', 'OpenAI Images'],
  ['openai_realtime', 'OpenAI Realtime'],
  ['rerank', 'Rerank'],
  ['embedding', 'Embeddings'],
  ['task', 'Task request'],
  ['mj_proxy', 'Midjourney'],
])('后端协议枚举 %s 显示为 %s', (requestType, label) => {
  render(
    <ActiveRequestCard
      request={{ ...entries[0], request_type: requestType }}
      serverNow={10000}
    />,
  );
  expect(screen.getByText(label)).toBeTruthy();
  expect(screen.getByText('Streaming request')).toBeTruthy();
});
