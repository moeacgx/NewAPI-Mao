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

import React, { useReducer } from 'react';
import axios, { AxiosError } from 'axios';
import { Toast } from '@douyinfe/semi-ui';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import PersonalSetting from '../PersonalSetting';
import { UserContext } from '../../../context/User';
import { StatusContext } from '../../../context/Status';
import { reducer } from '../../../context/User/reducer';
import { updateAPI } from '../../../helpers/api';

const user = {
  id: 7,
  username: 'existing',
  role: 1,
  email: '',
  quota: 1000,
  token: 'access',
  access_token: 'access',
  session: { sid: 'sid' },
  setting: '{}',
};
const originalAdapter = axios.defaults.adapter;
let requests;
let profile;
let status;
let verificationFailure;
let bindFailure;
let location;
let profileResponse;

beforeEach(() => {
  localStorage.setItem('user', JSON.stringify(user));
  sessionStorage.clear();
  requests = [];
  profile = {
    id: 7,
    username: 'existing',
    role: 1,
    email: '',
    quota: 1000,
    setting: '{}',
  };
  status = {};
  verificationFailure = false;
  bindFailure = false;
  profileResponse = null;
  location = { origin: 'http://localhost:3000', assign: vi.fn() };
  vi.stubGlobal(
    'window',
    new Proxy(window, {
      get(target, property) {
        if (property === 'location') return location;
        return Reflect.get(target, property, target);
      },
    }),
  );
  vi.spyOn(Toast, 'error').mockImplementation(() => {});
  vi.spyOn(Toast, 'success').mockImplementation(() => {});
  axios.defaults.adapter = async (config) => {
    requests.push(config);
    if (
      (config.url.startsWith('/api/verification') && verificationFailure) ||
      (config.url === '/api/oauth/email/bind' && bindFailure)
    ) {
      throw new AxiosError('Network Error', 'ERR_NETWORK', config);
    }
    if (config.url === '/api/oauth/email/bind')
      profile.email = 'alias+tag@example.com';
    const response = {
      status: 200,
      statusText: 'OK',
      config,
      headers: {},
      data: { success: true, data: {} },
    };
    if (config.url === '/api/user/self') {
      if (profileResponse) return await profileResponse;
      response.data.data = { ...profile };
    }
    if (config.url === '/api/status') response.data.data = status;
    if (config.url === '/api/user/oauth/bindings') response.data.data = [];
    if (config.url === '/api/oauth/state')
      response.data.data = { flow_token: 'state' };
    if (config.url === '/api/oauth/telegram/bind/start') {
      response.data.data = {
        flow_token: 'telegram-flow',
        callback_url: '/api/oauth/telegram/bind/telegram-flow',
      };
    }
    return response;
  };
  updateAPI();
});

afterEach(() => {
  cleanup();
  axios.defaults.adapter = originalAdapter;
  vi.unstubAllGlobals();
});

function Page({ initialState = { user } }) {
  const state = useReducer(reducer, initialState);
  return (
    <MemoryRouter>
      <StatusContext.Provider value={[{ status }, vi.fn()]}>
        <UserContext.Provider value={state}>
          <PersonalSetting />
        </UserContext.Provider>
      </StatusContext.Provider>
    </MemoryRouter>
  );
}

async function openEmail() {
  render(<Page />);
  const emailCard = (await screen.findByText('邮箱')).closest('.semi-card');
  fireEvent.click(within(emailCard).getByRole('button', { name: '绑定' }));
  return await screen.findByRole('dialog');
}

test('个人中心回读资料保留原用户的会话和访问令牌', async () => {
  render(<Page />);
  await waitFor(() =>
    expect(
      requests.some((request) => request.url === '/api/user/passkey'),
    ).toBe(true),
  );
  expect(JSON.parse(localStorage.getItem('user'))).toMatchObject(user);
});

test('冷启动个人中心时使用已持久化会话读取资料并初始化上下文', async () => {
  render(<Page initialState={{}} />);
  await screen.findByText('existing');
  expect(requests.some((request) => request.url === '/api/user/self')).toBe(
    true,
  );
  expect(JSON.parse(localStorage.getItem('user'))).toMatchObject(user);
});

test.each(['切换账号', '轮换令牌'])(
  '资料请求等待期间%s不会被旧凭证覆盖',
  async (action) => {
    let finish;
    profileResponse = new Promise((resolve) => {
      finish = resolve;
    });
    render(<Page />);
    await waitFor(() =>
      expect(requests.some((request) => request.url === '/api/user/self')).toBe(
        true,
      ),
    );
    const current =
      action === '切换账号'
        ? {
            ...user,
            id: 99,
            token: 'other',
            access_token: 'other',
            session: { sid: 'other-session' },
          }
        : { ...user, token: 'renewed', access_token: 'renewed' };
    localStorage.setItem('user', JSON.stringify(current));
    await act(async () =>
      finish({
        status: 200,
        headers: {},
        data: { success: true, data: profile },
      }),
    );
    expect(JSON.parse(localStorage.getItem('user'))).toMatchObject(current);
    if (action === '切换账号') expect(Toast.error).toHaveBeenCalled();
  },
);

test('邮箱别名完整传参，发送失败后允许重试', async () => {
  verificationFailure = true;
  const dialog = await openEmail();
  fireEvent.change(within(dialog).getByPlaceholderText('输入邮箱地址'), {
    target: { value: 'alias+tag@example.com' },
  });
  fireEvent.click(within(dialog).getByRole('button', { name: '获取验证码' }));
  await waitFor(() => expect(Toast.error).toHaveBeenCalled());
  const request = requests.find((item) =>
    item.url.startsWith('/api/verification'),
  );
  expect(request.url).toBe('/api/verification');
  expect(request.params.email).toBe('alias+tag@example.com');
  const send = within(dialog).getByRole('button', { name: '获取验证码' });
  await waitFor(() => expect(send.disabled).toBe(false));
  verificationFailure = false;
  fireEvent.click(send);
  await within(dialog).findByRole('button', { name: /重新发送/ });
});

test('邮箱验证码发送尝试后重新验证 Turnstile，未验证不发送请求', async () => {
  status = { turnstile_check: true, turnstile_site_key: 'site-key' };
  verificationFailure = true;
  let challenge;
  const turnstile = {
    render: vi.fn((_element, options) => {
      challenge = options;
      return 'widget';
    }),
    remove: vi.fn(),
  };
  vi.stubGlobal('turnstile', turnstile);
  const dialog = await openEmail();
  await act(async () => {
    globalThis.cf__reactTurnstileOnLoad?.();
  });
  await waitFor(() => expect(challenge).toBeTruthy());
  fireEvent.change(within(dialog).getByPlaceholderText('输入邮箱地址'), {
    target: { value: 'alias@example.com' },
  });
  const send = within(dialog).getByRole('button', { name: '获取验证码' });
  fireEvent.click(send);
  expect(requests.some((item) => item.url === '/api/verification')).toBe(false);
  expect(send.disabled).toBe(false);
  act(() => challenge.callback('challenge-1'));
  fireEvent.click(send);
  await waitFor(() => expect(turnstile.render).toHaveBeenCalledTimes(2));
  expect(
    requests.find((item) => item.url === '/api/verification').params.turnstile,
  ).toBe('challenge-1');
  fireEvent.click(send);
  expect(
    requests.filter((item) => item.url === '/api/verification'),
  ).toHaveLength(1);
  verificationFailure = false;
  act(() => challenge.callback('challenge-2'));
  fireEvent.click(send);
  await within(dialog).findByRole('button', { name: /重新发送/ });
  expect(
    requests.filter((item) => item.url === '/api/verification')[1].params
      .turnstile,
  ).toBe('challenge-2');
});

test('邮箱绑定失败后可重试，成功回读邮箱同时保留原账号及会话', async () => {
  bindFailure = true;
  const dialog = await openEmail();
  fireEvent.change(within(dialog).getByPlaceholderText('输入邮箱地址'), {
    target: { value: 'ALIAS+tag@example.com' },
  });
  fireEvent.change(within(dialog).getByPlaceholderText('验证码'), {
    target: { value: '123456' },
  });
  fireEvent.click(within(dialog).getByRole('button', { name: 'confirm' }));
  await waitFor(() => expect(Toast.error).toHaveBeenCalled());
  expect(screen.getByRole('dialog')).toBeTruthy();
  bindFailure = false;
  fireEvent.click(within(dialog).getByRole('button', { name: 'confirm' }));
  await waitFor(() =>
    expect(JSON.parse(localStorage.getItem('user')).email).toBe(
      'alias+tag@example.com',
    ),
  );
  expect(JSON.parse(localStorage.getItem('user'))).toMatchObject({
    ...user,
    email: 'alias+tag@example.com',
  });
  expect(
    requests.findLastIndex((item) => item.url === '/api/user/self'),
  ).toBeGreaterThan(
    requests.findLastIndex((item) => item.url === '/api/oauth/email/bind'),
  );
});

test.each([
  ['GitHub', { github_oauth: true, github_client_id: 'client' }, 'github'],
  ['Discord', { discord_oauth: true, discord_client_id: 'client' }, 'discord'],
  [
    'OIDC',
    {
      oidc_enabled: true,
      oidc_client_id: 'client',
      oidc_authorization_endpoint: 'https://identity.example/authorize',
    },
    'oidc',
  ],
  ['LinuxDO', { linuxdo_oauth: true, linuxdo_client_id: 'client' }, 'linuxdo'],
  [
    '团队邮箱',
    {
      custom_oauth_providers: [
        {
          id: 2,
          name: '团队邮箱',
          slug: 'custom',
          client_id: 'client',
          authorization_endpoint: 'https://identity.example/authorize',
        },
      ],
    },
    'custom',
  ],
])('%s 个人中心绑定按钮使用 bind 流程', async (label, enabled, provider) => {
  status = enabled;
  render(<Page />);
  const card = (await screen.findByText(label)).closest('.semi-card');
  await waitFor(() =>
    expect(within(card).getByRole('button', { name: '绑定' }).disabled).toBe(
      false,
    ),
  );
  fireEvent.click(within(card).getByRole('button', { name: '绑定' }));
  await waitFor(() =>
    expect(requests.some((request) => request.url === '/api/oauth/state')).toBe(
      true,
    ),
  );
  expect(
    JSON.parse(
      requests.find((request) => request.url === '/api/oauth/state').data,
    ),
  ).toEqual({ provider, intent: 'bind' });
});

test('Telegram 获取一次性绑定地址，只采纳同源且 flow 匹配的结果', async () => {
  status = { telegram_oauth: true, telegram_bot_name: 'test_bot' };
  render(<Page />);
  const card = (await screen.findByText('Telegram')).closest('.semi-card');
  await waitFor(() =>
    expect(within(card).getByRole('button', { name: '绑定' }).disabled).toBe(
      false,
    ),
  );
  fireEvent.click(within(card).getByRole('button', { name: '绑定' }));
  await waitFor(() =>
    expect(
      requests.some(
        (request) => request.url === '/api/oauth/telegram/bind/start',
      ),
    ).toBe(true),
  );
  await waitFor(() =>
    expect(
      document
        .querySelector('script[data-auth-url]')
        ?.getAttribute('data-auth-url'),
    ).toBe('http://localhost:3000/api/oauth/telegram/bind/telegram-flow'),
  );
  const before = requests.filter(
    (request) => request.url === '/api/user/self',
  ).length;
  const message = {
    type: 'telegram:binding:result',
    flow_token: 'telegram-flow',
    success: true,
  };
  act(() =>
    window.dispatchEvent(
      new MessageEvent('message', {
        origin: 'https://untrusted.example',
        data: message,
      }),
    ),
  );
  act(() =>
    window.dispatchEvent(
      new MessageEvent('message', {
        origin: location.origin,
        data: { ...message, flow_token: 'other-flow' },
      }),
    ),
  );
  expect(
    requests.filter((request) => request.url === '/api/user/self'),
  ).toHaveLength(before);
  profile.telegram_id = '123';
  act(() =>
    window.dispatchEvent(
      new MessageEvent('message', { origin: location.origin, data: message }),
    ),
  );
  await waitFor(() =>
    expect(JSON.parse(localStorage.getItem('user')).telegram_id).toBe('123'),
  );
  expect(JSON.parse(localStorage.getItem('user')).id).toBe(7);
});
