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
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { authHeader } from '../auth';
import { AdminRoute, AuthRedirect, PrivateRoute, RootRoute } from '../auth';
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
    ['空对象', '{}'],
    ['缺少 role', '{"id":7}'],
    ['id 非正整数', '{"id":0,"role":100}'],
    ['id 非整数', '{"id":1.5,"role":100}'],
    ['role 非数字', '{"id":7,"role":"100"}'],
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

  it.each([
    ['登录页', '/login', <AuthRedirect>login</AuthRedirect>],
    ['普通用户', '/console', <PrivateRoute>private</PrivateRoute>],
    ['管理员', '/admin', <AdminRoute>admin</AdminRoute>],
    ['Root', '/root', <RootRoute>root</RootRoute>],
  ])('storage getter 被拒绝时%s路由按未登录处理', (_name, path, element) => {
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
      render(
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route path={path} element={element} />
            {path !== '/login' && (
              <Route path='/login' element={<span>login</span>} />
            )}
          </Routes>
        </MemoryRouter>,
      );
      expect(screen.getByText('login')).toBeTruthy();
    } finally {
      Object.defineProperty(globalThis, 'localStorage', descriptor);
    }
  });
});
