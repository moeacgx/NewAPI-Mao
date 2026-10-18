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
import axios from 'axios';
import { Toast } from '@douyinfe/semi-ui';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import {
  onGitHubOAuthClicked,
  onDiscordOAuthClicked,
  onLinuxDOOAuthClicked,
  onOIDCClicked,
  onCustomOAuthClicked,
  updateAPI,
} from '../api';
import OAuth2Callback from '../../components/auth/OAuth2Callback';
import { UserContext } from '../../context/User';

const user = {
  id: 7,
  username: 'existing',
  token: 'access',
  session: { sid: 'sid' },
};
const provider = {
  slug: 'custom',
  client_id: 'client',
  authorization_endpoint: 'https://identity.example/authorize',
};
const flows = [
  ['github', (options) => onGitHubOAuthClicked('client', options)],
  ['discord', (options) => onDiscordOAuthClicked('client', options)],
  ['linuxdo', (options) => onLinuxDOOAuthClicked('client', options)],
  [
    'oidc',
    (options) =>
      onOIDCClicked(
        'https://identity.example/authorize',
        'client',
        false,
        options,
      ),
  ],
  ['custom', (options) => onCustomOAuthClicked(provider, options)],
];
const originalAdapter = axios.defaults.adapter;
let requests;
let callbackResult;
let location;
let errorToast;
let opener;
let close;

beforeEach(() => {
  localStorage.setItem('user', JSON.stringify(user));
  sessionStorage.clear();
  requests = [];
  callbackResult = { success: true, data: { action: 'bind' } };
  location = { origin: 'http://localhost:3000', assign: vi.fn() };
  opener = null;
  close = vi.fn();
  vi.stubGlobal(
    'window',
    new Proxy(window, {
      get(target, property) {
        if (property === 'location') return location;
        if (property === 'opener') return opener;
        if (property === 'close') return close;
        return Reflect.get(target, property, target);
      },
    }),
  );
  errorToast = vi.spyOn(Toast, 'error').mockImplementation(() => {});
  vi.spyOn(Toast, 'success').mockImplementation(() => {});
  axios.defaults.adapter = async (config) => {
    requests.push(config);
    return {
      status: 200,
      statusText: 'OK',
      config,
      headers: {},
      data:
        config.url === '/api/oauth/state'
          ? { success: true, data: { flow_token: 'bind-state' } }
          : callbackResult,
    };
  };
  updateAPI();
});

afterEach(() => {
  axios.defaults.adapter = originalAdapter;
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

test.each(flows)(
  '%s 绑定携带 bind 意图和原会话，不登出',
  async (name, start) => {
    await start({ intent: 'bind', shouldLogout: true });
    expect(requests.map((request) => request.url)).toEqual([
      '/api/oauth/state',
    ]);
    expect(JSON.parse(requests[0].data)).toEqual({
      provider: name,
      intent: 'bind',
    });
    expect(requests[0].headers.Authorization).toBe('Bearer access');
    expect(JSON.parse(localStorage.getItem('user'))).toEqual(user);
    expect(location.assign).toHaveBeenCalledOnce();
  },
);

test.each(flows)('%s 登录保留 login 意图及显式登出', async (name, start) => {
  await start({ shouldLogout: true });
  expect(requests.map((request) => request.url)).toEqual([
    '/api/user/auth/logout',
    '/api/oauth/state',
  ]);
  expect(JSON.parse(requests[1].data)).toMatchObject({
    provider: name,
    intent: 'login',
  });
  expect(localStorage.getItem('user')).toBeNull();
});

function renderCallback(
  dispatch = vi.fn(),
  type = 'github',
  query = 'code=code&state=bind-state',
) {
  render(
    <MemoryRouter initialEntries={[`/oauth/${type}?${query}`]}>
      <UserContext.Provider value={[{ user }, dispatch]}>
        <Routes>
          <Route
            path='/oauth/:provider'
            element={<OAuth2Callback type={type} />}
          />
          <Route path='/console/personal' element={<div>个人中心</div>} />
          <Route path='/console/token' element={<div>令牌列表</div>} />
        </Routes>
      </UserContext.Provider>
    </MemoryRouter>,
  );
  return dispatch;
}

test('绑定回调不接受登录包、不替换原账号', async () => {
  await onGitHubOAuthClicked('client', { intent: 'bind' });
  callbackResult = {
    success: true,
    data: { user: { id: 99 }, access_token: 'other' },
  };
  const dispatch = renderCallback();
  await waitFor(() => expect(errorToast).toHaveBeenCalled());
  expect(dispatch).not.toHaveBeenCalled();
  expect(JSON.parse(localStorage.getItem('user'))).toEqual(user);
  expect(screen.queryByText('令牌列表')).toBeNull();
});

test('绑定完成只返回个人中心，保留原身份凭证', async () => {
  await onGitHubOAuthClicked('client', { intent: 'bind' });
  const dispatch = renderCallback();
  await screen.findByText('个人中心');
  expect(dispatch).not.toHaveBeenCalled();
  expect(JSON.parse(localStorage.getItem('user'))).toEqual(user);
});

test('普通登录回调仍接受并保存登录认证包', async () => {
  callbackResult = {
    success: true,
    data: { user: { id: 99 }, access_token: 'login-access' },
  };
  const dispatch = renderCallback();
  await screen.findByText('令牌列表');
  expect(dispatch).toHaveBeenCalled();
  expect(JSON.parse(localStorage.getItem('user'))).toMatchObject({
    id: 99,
    token: 'login-access',
  });
});

test('会话存储不可用时绑定拒绝跳转，保留原账号', async () => {
  const stored = Storage.prototype.setItem;
  vi.spyOn(Storage.prototype, 'setItem').mockImplementation(
    function (key, value) {
      if (this === sessionStorage) throw new Error('storage blocked');
      return stored.call(this, key, value);
    },
  );
  await onGitHubOAuthClicked('client', { intent: 'bind' });
  expect(location.assign).not.toHaveBeenCalled();
  expect(JSON.parse(localStorage.getItem('user'))).toEqual(user);
});

test('会话存储不可用仍可完成普通登录', async () => {
  const stored = Storage.prototype.getItem;
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(function (key) {
    if (this === sessionStorage) throw new Error('storage blocked');
    return stored.call(this, key);
  });
  callbackResult = {
    success: true,
    data: { user: { id: 99 }, access_token: 'login-access' },
  };
  renderCallback();
  await screen.findByText('令牌列表');
  expect(JSON.parse(localStorage.getItem('user'))).toMatchObject({
    id: 99,
    token: 'login-access',
  });
});

test('Telegram 回调仅通知原窗口，不重新登录或再次请求上游', async () => {
  opener = { closed: false, postMessage: vi.fn() };
  const dispatch = renderCallback(
    vi.fn(),
    'telegram',
    'telegram_bind=success&flow_token=telegram-flow',
  );
  await waitFor(() => expect(close).toHaveBeenCalled());
  expect(opener.postMessage).toHaveBeenCalledWith(
    {
      type: 'telegram:binding:result',
      flow_token: 'telegram-flow',
      success: true,
      code: undefined,
    },
    location.origin,
  );
  expect(requests).toEqual([]);
  expect(dispatch).not.toHaveBeenCalled();
  expect(JSON.parse(localStorage.getItem('user'))).toEqual(user);
});
