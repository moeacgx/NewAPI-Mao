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

import axios, { AxiosError } from 'axios';
import { Toast } from '@douyinfe/semi-ui';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { API, updateAPI } from '../api';
import { showError } from '../utils';

const originalAdapter = axios.defaults.adapter;
const storedUser = {
  id: 7,
  role: 100,
  token: 'expired-access',
  session: { sid: 'browser-session' },
};
let location;
let requests;
let refreshStatus;
let refreshCode;
let refreshError;
let businessStatus;
let errorToast;

beforeEach(() => {
  location = { href: '/console/channel' };
  vi.stubGlobal(
    'window',
    new Proxy(window, {
      get(target, property) {
        if (property === 'location') return location;
        return Reflect.get(target, property, target);
      },
    }),
  );
  vi.spyOn(console, 'error').mockImplementation(() => {});
  errorToast = vi.spyOn(Toast, 'error').mockImplementation(() => {});
  localStorage.setItem('user', JSON.stringify(storedUser));
  requests = [];
  refreshStatus = 429;
  refreshCode = undefined;
  refreshError = undefined;
  businessStatus = 401;
  axios.defaults.adapter = async (config) => {
    requests.push(config);
    const refreshing = config.url === '/api/user/auth/refresh';
    const status = refreshing ? refreshStatus : businessStatus;
    const response = {
      status,
      statusText: String(status),
      config,
      headers: status === 429 ? { 'retry-after': '30' } : {},
      data: refreshing
        ? {
            success: true,
            data: {
              access_token: 'renewed-access',
              user: { id: 7, role: 100 },
              session: { sid: 'browser-session' },
            },
          }
        : { success: true },
    };
    if (status !== 200) {
      response.data = { success: false, code: refreshCode };
      const error = new AxiosError(
        status ? `Request failed with status code ${status}` : 'Network Error',
        status ? 'ERR_BAD_RESPONSE' : 'ERR_NETWORK',
        config,
        undefined,
        status ? response : undefined,
      );
      if (refreshing) refreshError = error;
      throw error;
    }
    if (refreshing) businessStatus = 200;
    return response;
  };
  updateAPI();
});

afterEach(() => {
  axios.defaults.adapter = originalAdapter;
  vi.unstubAllGlobals();
});

describe('Classic 会话刷新错误处理', () => {
  it.each([429, 503, 0])(
    '刷新返回 %s 时保留登录态并向调用方传递刷新错误',
    async (status) => {
      refreshStatus = status;
      const error = await API.get('/api/user/self').catch((error) => error);

      expect(error).toBe(refreshError);
      expect(JSON.parse(localStorage.getItem('user'))).toEqual(storedUser);
      expect(location.href).toBe('/console/channel');
      expect(() => showError(error)).not.toThrow();
      expect(JSON.parse(localStorage.getItem('user'))).toEqual(storedUser);
      expect(location.href).toBe('/console/channel');
      expect(requests.map((request) => request.url)).toEqual([
        '/api/user/self',
        '/api/user/auth/refresh',
      ]);
    },
  );

  it('业务请求直接返回 429 时不刷新也不退出', async () => {
    businessStatus = 429;
    await expect(API.get('/api/user/self')).rejects.toMatchObject({
      response: { status: 429 },
    });
    expect(requests).toHaveLength(1);
    expect(JSON.parse(localStorage.getItem('user'))).toEqual(storedUser);
    expect(location.href).toBe('/console/channel');
  });

  it('刷新明确返回 401 时清除登录态并跳转登录页', async () => {
    refreshStatus = 401;
    await expect(API.get('/api/user/self')).rejects.toMatchObject({
      response: { status: 401 },
    });
    expect(localStorage.getItem('user')).toBeNull();
    expect(location.href).toBe('/login?expired=true');
  });

  it('刷新明确报告会话身份不匹配时清除旧会话并保留真实 409 错误', async () => {
    refreshStatus = 409;
    refreshCode = 'AUTH_SESSION_MISMATCH';
    const error = await API.get('/api/user/self').catch((error) => error);
    expect(error).toBe(refreshError);
    expect(error.response.status).toBe(409);
    expect(localStorage.getItem('user')).toBeNull();
    expect(location.href).toBe('/login?expired=true');
  });

  it.each(['AUTH_REFRESH_RACE', 'OTHER_CONFLICT'])(
    '刷新返回 %s 冲突时不会误清登录态',
    async (code) => {
      refreshStatus = 409;
      refreshCode = code;
      const error = await API.get('/api/user/self').catch((error) => error);
      expect(error).toBe(refreshError);
      expect(JSON.parse(localStorage.getItem('user'))).toEqual(storedUser);
      expect(location.href).toBe('/console/channel');
    },
  );

  it('刷新成功后携带新令牌只重放一次原请求', async () => {
    refreshStatus = 200;
    await expect(API.get('/api/user/self')).resolves.toMatchObject({
      status: 200,
    });
    expect(requests.map((request) => request.url)).toEqual([
      '/api/user/self',
      '/api/user/auth/refresh',
      '/api/user/self',
    ]);
    expect(requests[1].headers.get('X-Auth-Session')).toBe('browser-session');
    expect(requests[2].headers.get('Authorization')).toBe(
      'Bearer renewed-access',
    );
    expect(JSON.parse(localStorage.getItem('user')).token).toBe(
      'renewed-access',
    );
    expect(location.href).toBe('/console/channel');
  });

  it('原业务请求跳过全局错误时刷新失败也不弹窗', async () => {
    const error = await API.get('/api/user/self', {
      skipErrorHandler: true,
    }).catch((error) => error);
    expect(error).toBe(refreshError);
    expect(errorToast).not.toHaveBeenCalled();
    expect(JSON.parse(localStorage.getItem('user'))).toEqual(storedUser);
    expect(location.href).toBe('/console/channel');
  });
});
