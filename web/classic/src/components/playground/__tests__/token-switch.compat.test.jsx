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
  renderHook,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, test, vi } from 'vitest';
import { API } from '../../../helpers/api';
import { useApiRequest } from '../../../hooks/playground/useApiRequest';
import { useDataLoader } from '../../../hooks/playground/useDataLoader';
import TokenSourceSelector from '../TokenSourceSelector';

const adapter = API.defaults.adapter;
afterEach(() => {
  API.defaults.adapter = adapter;
  vi.unstubAllGlobals();
});

test('选择令牌显示技能与分组名称，可搜索翻页和返回账号模式', async () => {
  const calls = [];
  const onSelect = vi.fn();
  API.defaults.adapter = async (config) => {
    calls.push(config.url);
    return {
      status: 200,
      config,
      headers: {},
      data: {
        success: true,
        data: {
          items: [
            {
              id: 4,
              name: 'Review token',
              group_details: [
                { id: 'stable-id', code: 'premium-code', name: '高级模型组' },
              ],
              prompt_count: 2,
              key: 'secret',
            },
          ],
          total: 21,
        },
      },
    };
  };
  const view = render(<TokenSourceSelector token={null} onSelect={onSelect} />);
  await userEvent.click(
    screen.getByRole('button', { name: 'Switch API token' }),
  );
  await screen.findByRole('button', { name: 'Review token' });
  expect(screen.getByText('高级模型组')).toBeTruthy();
  expect(screen.queryByText('premium-code')).toBeNull();
  expect(screen.getByText('2 installed prompts')).toBeTruthy();
  expect(screen.queryByText('secret')).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Next' }));
  await waitFor(() =>
    expect(calls.some((url) => url.includes('p=2'))).toBe(true),
  );
  await userEvent.type(
    screen.getByRole('textbox', { name: 'Search token names' }),
    'review',
  );
  await waitFor(() =>
    expect(
      calls.some(
        (url) => url.includes('keyword=review') && url.includes('p=1'),
      ),
    ).toBe(true),
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Review token' }),
  );
  expect(onSelect).toHaveBeenLastCalledWith({
    id: 4,
    name: 'Review token',
    group_name: '高级模型组',
    prompt_count: 2,
  });
  view.rerender(
    <TokenSourceSelector
      token={{ id: 4, name: 'Review token' }}
      onSelect={onSelect}
    />,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Switch API token' }),
  );
  await userEvent.click(screen.getByRole('button', { name: 'Account mode' }));
  expect(onSelect).toHaveBeenLastCalledWith(null);
});

test('流式请求使用相同令牌身份并在完成前保持切换锁', () => {
  const xhrs = [];
  class TestXHR extends EventTarget {
    constructor() {
      super();
      this.headers = {};
      this.status = 200;
      this.readyState = 1;
      this.responseText = '';
    }
    open() {}
    setRequestHeader(name, value) {
      this.headers[name] = value;
    }
    getAllResponseHeaders() {
      return '';
    }
    send(body) {
      this.body = body;
      xhrs.push(this);
    }
    abort() {
      this.dispatchEvent(new Event('abort'));
    }
  }
  vi.stubGlobal('XMLHttpRequest', TestXHR);
  const source = { current: null };
  const { result } = renderHook(() =>
    useApiRequest(vi.fn(), vi.fn(), vi.fn(), source, vi.fn(), 9, true),
  );
  act(() =>
    result.current.sendRequest(
      {
        model: 'chat-model',
        group: 'account-only',
        stream: true,
        messages: [],
      },
      true,
    ),
  );
  expect(xhrs).toHaveLength(1);
  expect(xhrs[0].headers['X-Playground-Token-Id']).toBe('9');
  expect(JSON.parse(xhrs[0].body).group).toBeUndefined();
  expect(result.current.isRequesting).toBe(true);
  act(() => {
    xhrs[0].responseText = 'data: [DONE]\n\n';
    xhrs[0].dispatchEvent(new Event('progress'));
  });
  expect(result.current.isRequesting).toBe(false);
  expect(source.current).toBeNull();
});

test('令牌切换不接受上一令牌的迟到模型，返回账号后恢复账号分组', async () => {
  const pending = new Map();
  const calls = [];
  const setModels = vi.fn();
  const onInputChange = vi.fn();
  const userState = { user: { group: 'default' } };
  const inputs = { model: 'old-model', group: 'premium-code' };
  const setGroups = vi.fn();
  API.defaults.adapter = async (config) => {
    calls.push(config);
    if (config.url === '/pg/token/models') {
      return new Promise((resolve) =>
        pending.set(config.headers['X-Playground-Token-Id'], (ids) =>
          resolve({
            status: 200,
            config,
            headers: {},
            data: { data: ids.map((id) => ({ id })) },
          }),
        ),
      );
    }
    return {
      status: 200,
      config,
      headers: {},
      data: {
        success: true,
        data: config.url.includes('models') ? ['account-model'] : {},
      },
    };
  };
  const hook = renderHook(
    ({ token }) =>
      useDataLoader(
        userState,
        inputs,
        onInputChange,
        setModels,
        setGroups,
        token,
      ),
    { initialProps: { token: 4 } },
  );
  await waitFor(() => expect(pending.has('4')).toBe(true));
  hook.rerender({ token: 5 });
  await waitFor(() => expect(pending.has('5')).toBe(true));
  await act(async () => pending.get('5')(['new-model']));
  expect(hook.result.current.modelsReady).toBe(true);
  expect(onInputChange).toHaveBeenLastCalledWith('model', 'new-model');
  await act(async () => pending.get('4')(['stale-model']));
  expect(
    setModels.mock.calls.flat(2).some((item) => item?.value === 'stale-model'),
  ).toBe(false);
  hook.rerender({ token: null });
  await waitFor(() =>
    expect(
      calls.some(
        (call) =>
          call.url.includes('models') && call.url !== '/pg/token/models',
      ),
    ).toBe(true),
  );
  const accountCall = calls.find(
    (call) => call.url.includes('models') && call.url !== '/pg/token/models',
  );
  expect(accountCall.params.group).toBe('premium-code');
  expect(accountCall.headers['X-Playground-Token-Id']).toBeUndefined();
});

test('请求期间不能切换令牌', () => {
  render(<TokenSourceSelector token={null} disabled onSelect={vi.fn()} />);
  expect(
    screen.getByRole('button', { name: 'Switch API token' }).disabled,
  ).toBe(true);
});

test('非流式自定义请求剔除账号分组并携带令牌 ID，完成后解除请求锁', async () => {
  let resolve;
  const fetchMock = vi.fn(
    () =>
      new Promise((done) => {
        resolve = done;
      }),
  );
  vi.stubGlobal('fetch', fetchMock);
  const { result } = renderHook(() =>
    useApiRequest(
      vi.fn(),
      vi.fn(),
      vi.fn(),
      { current: null },
      vi.fn(),
      4,
      true,
    ),
  );
  act(() =>
    result.current.sendRequest(
      { model: 'test', group: 'account-only', messages: [] },
      false,
    ),
  );
  expect(result.current.isRequesting).toBe(true);
  expect(fetchMock.mock.calls[0][1].headers['X-Playground-Token-Id']).toBe('4');
  expect(JSON.parse(fetchMock.mock.calls[0][1].body).group).toBeUndefined();
  await act(async () =>
    resolve({ ok: true, json: async () => ({ choices: [] }) }),
  );
  expect(result.current.isRequesting).toBe(false);
  vi.unstubAllGlobals();
});

test.each([
  { error: { message: 'Token disabled' } },
  { success: false, message: 'Token disabled', data: [] },
])(
  '令牌模型加载失败清除模型并拒绝发送，不回退账号模型：%j',
  async (failure) => {
    const calls = [];
    const setModels = vi.fn();
    const onInputChange = vi.fn();
    const userState = { user: { group: 'default' } };
    const inputs = { model: 'old-model', group: 'default' };
    const setGroups = vi.fn();
    API.defaults.adapter = async (config) => {
      calls.push(config);
      if (config.url === '/pg/token/models')
        return {
          status: 200,
          config,
          headers: {},
          data: failure,
        };
      return {
        status: 200,
        config,
        headers: {},
        data: { success: true, data: {} },
      };
    };
    const { result } = renderHook(() =>
      useDataLoader(userState, inputs, onInputChange, setModels, setGroups, 4),
    );
    await waitFor(() =>
      expect(result.current.modelError).toBe('Token disabled'),
    );
    expect(result.current.modelsReady).toBe(false);
    expect(setModels).toHaveBeenCalledWith([]);
    expect(onInputChange).toHaveBeenCalledWith('model', '');
    expect(
      calls.some(
        (call) =>
          call.url.includes('models') && call.url !== '/pg/token/models',
      ),
    ).toBe(false);
    expect(
      calls.find((call) => call.url === '/pg/token/models').headers[
        'X-Playground-Token-Id'
      ],
    ).toBe('4');
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    const request = renderHook(() =>
      useApiRequest(
        vi.fn(),
        vi.fn(),
        vi.fn(),
        { current: null },
        vi.fn(),
        4,
        false,
      ),
    );
    act(() =>
      request.result.current.sendRequest({ model: 'old-model' }, false),
    );
    expect(fetchMock).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
  },
);
