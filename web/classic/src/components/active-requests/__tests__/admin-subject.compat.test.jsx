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
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';
import { API } from '../../../helpers/api';
import ActiveRequestsPanel from '../ActiveRequestsPanel';
import { useActiveRequests } from '../useActiveRequests';

function response(id, username, model = null) {
  return {
    data: {
      success: true,
      data: {
        subject: { id, username },
        items: model
          ? [
              {
                id: `request-${id}`,
                model_name: model,
                status: 'processing',
                token_name: 'Test token',
                started_at: 1000,
                is_stream: false,
                first_response_ms: null,
                request_type: 'openai_responses',
              },
            ]
          : [],
        total: model ? 1 : 0,
        page: 1,
        page_size: 3,
        scope: 'cluster',
        observed_at: 2000,
        summary: {
          total: model ? 1 : 0,
          streaming: 0,
          non_streaming: model ? 1 : 0,
          waiting: 0,
          active_tokens: 0,
          by_token: {},
        },
      },
    },
  };
}

test('指定用户零连接仍展示查询对象及空状态，抽屉沿用管理员查询', async () => {
  const get = vi.spyOn(API, 'get').mockResolvedValue(response(5215, 'alice'));
  render(<ActiveRequestsPanel subject={{ userId: '5215' }} />);
  await screen.findByText('Connections for alice (ID: 5215)');
  expect(
    screen.getByText('This user has no current connections.'),
  ).toBeTruthy();
  expect(get.mock.calls[0][0]).toBe('/api/log/active');
  expect(get.mock.calls[0][1].params.user_id).toBe('5215');
  await userEvent.click(
    screen.getByRole('button', { name: 'View all connections' }),
  );
  const drawer = await screen.findByRole('dialog');
  await within(drawer).findByText('Connections for alice (ID: 5215)');
  expect(get.mock.calls.at(-1)[1].params.user_id).toBe('5215');
  expect(get.mock.calls.at(-1)[1].params.page_size).toBe(10);
});

test('切换用户立即移除旧连接并取消旧请求，迟到响应不覆盖新用户', async () => {
  const pending = [];
  vi.spyOn(API, 'get').mockImplementation(
    (url, config) =>
      new Promise((resolve) => pending.push({ url, config, resolve })),
  );
  const hook = renderHook(({ subject }) => useActiveRequests({ subject }), {
    initialProps: { subject: { userId: '5215' } },
  });
  await act(async () =>
    pending[0].resolve(response(5215, 'alice', 'alice-model')),
  );
  expect(hook.result.current.snapshot.items[0].model_name).toBe('alice-model');
  act(() => hook.result.current.refresh());
  hook.rerender({ subject: { username: 'bob&user_id=5215' } });
  expect(pending[1].config.signal.aborted).toBe(true);
  expect(hook.result.current.snapshot).toBeNull();
  expect(pending[2].url).toBe('/api/log/active');
  expect(pending[2].config.params.username).toBe('bob&user_id=5215');
  expect(pending[2].config.params.user_id).toBeUndefined();
  await act(async () =>
    pending[2].resolve(response(5216, 'bob&user_id=5215', 'bob-model')),
  );
  await act(async () =>
    pending[1].resolve(response(5215, 'alice', 'late-alice-model')),
  );
  expect(hook.result.current.snapshot.items[0].model_name).toBe('bob-model');
  hook.rerender({ subject: null });
  expect(hook.result.current.snapshot).toBeNull();
  expect(pending[3].url).toBe('/api/log/self/active');
  expect(pending[3].config.params.username).toBeUndefined();
});

test('切换用户关闭旧抽屉，查询失败显示错误而非空连接', async () => {
  vi.spyOn(API, 'get').mockResolvedValue(
    response(5215, 'alice', 'alice-model'),
  );
  const view = render(<ActiveRequestsPanel subject={{ userId: '5215' }} />);
  await screen.findByRole('article', { name: 'alice-model' });
  await userEvent.click(
    screen.getByRole('button', { name: 'View all connections' }),
  );
  await screen.findByRole('dialog');
  API.get.mockRejectedValue(new Error('unavailable'));
  view.rerender(<ActiveRequestsPanel subject={{ userId: '5216' }} />);
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  expect(await screen.findByRole('alert')).toBeTruthy();
  expect(screen.queryByRole('article', { name: 'alice-model' })).toBeNull();
  expect(
    screen.queryByText('This user has no current connections.'),
  ).toBeNull();
  expect(screen.getByText('User ID: 5216')).toBeTruthy();
});

test('连接观察限定等待时间，超时显示错误且后续轮询可以恢复', async () => {
  vi.useFakeTimers();
  try {
    const get = vi
      .spyOn(API, 'get')
      .mockRejectedValueOnce(new Error('timeout'))
      .mockResolvedValue(response(5215, 'alice', 'recovered-model'));
    const hook = renderHook(() =>
      useActiveRequests({ subject: { userId: '5215' } }),
    );
    await act(async () => {});
    expect(get.mock.calls[0][1].timeout).toBe(10000);
    expect(hook.result.current.error).toBe(true);
    expect(hook.result.current.loading).toBe(false);
    await act(async () => vi.advanceTimersByTimeAsync(2000));
    expect(hook.result.current.error).toBe(false);
    expect(hook.result.current.snapshot.items[0].model_name).toBe(
      'recovered-model',
    );
    hook.unmount();
  } finally {
    vi.useRealTimers();
  }
});
