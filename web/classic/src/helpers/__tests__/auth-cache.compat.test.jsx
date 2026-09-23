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

import { afterEach, describe, expect, it, vi } from 'vitest';
import { authHeader } from '../auth';
import { getUserIdFromLocalStorage } from '../utils';
import { readStoredUser } from '../auth-data';

const validUser = {
  id: 7,
  role: 100,
  token: 'access-token',
  session: { sid: 'session-id' },
};

afterEach(() => {
  localStorage.clear();
  vi.resetModules();
});

describe('Classic 用户缓存读取', () => {
  it.each([
    ['undefined', 'undefined'],
    ['损坏 JSON', '{'],
    ['null', 'null'],
    ['数组', '[]'],
  ])('清理 %s 缓存并返回未登录', (_name, rawValue) => {
    localStorage.setItem('user', rawValue);

    expect(readStoredUser()).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
  });

  it('Storage 读取失败时返回未登录且不抛异常', () => {
    const storage = vi
      .spyOn(Storage.prototype, 'getItem')
      .mockImplementation(() => {
        throw new DOMException('storage denied', 'SecurityError');
      });

    expect(readStoredUser()).toBeNull();
    expect(storage).toHaveBeenCalledWith('user');
  });

  it('访问 localStorage 本身失败时返回未登录且不抛异常', () => {
    const descriptor = Object.getOwnPropertyDescriptor(
      globalThis,
      'localStorage',
    );
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      get() {
        throw new DOMException('storage denied', 'SecurityError');
      },
    });

    try {
      expect(readStoredUser()).toBeNull();
    } finally {
      Object.defineProperty(globalThis, 'localStorage', descriptor);
    }
  });

  it('保留合法用户对象并让现有启动调用点读取用户身份', () => {
    localStorage.setItem('user', JSON.stringify(validUser));

    expect(readStoredUser()).toEqual(validUser);
    expect(getUserIdFromLocalStorage()).toBe(validUser.id);
    expect(authHeader()).toEqual({ Authorization: 'Bearer access-token' });
  });

  it('损坏缓存不会阻止真实 API 模块初始化', async () => {
    localStorage.setItem('user', '{');

    await expect(import('../api')).resolves.toBeDefined();
    expect(getUserIdFromLocalStorage()).toBe(-1);
    expect(authHeader()).toEqual({});
    expect(localStorage.getItem('user')).toBeNull();
  });
});
