/*
Copyright (C) 2023-2026 QuantumNous

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
import { afterEach, expect, test } from 'vitest'

import { SIDEBAR_MODULES_DEFAULT } from '@/features/system-settings/maintenance/config'
import { isModuleEnabled, parseSidebarConfig } from '@/hooks/use-sidebar-config'
import { ROLE } from '@/lib/roles'
import { Route } from '@/routes/_authenticated/mcp-management'
import { useAuthStore } from '@/stores/auth-store'

const initialState = useAuthStore.getState()
afterEach(() => useAuthStore.setState(initialState, true))

test('MCP 导航默认启用并遵循管理员、用户及权限的独立开关', () => {
  const defaults = parseSidebarConfig(SIDEBAR_MODULES_DEFAULT)
  expect(isModuleEnabled('/mcp-management', defaults, null)).toBe(true)
  expect(
    isModuleEnabled(
      '/mcp-management/',
      parseSidebarConfig({ admin: { mcp: false } }),
      null
    )
  ).toBe(false)
  expect(
    isModuleEnabled('/mcp-management', defaults, {
      admin: { enabled: true, mcp: false },
    })
  ).toBe(false)
  expect(
    isModuleEnabled('/mcp-management', defaults, null, {
      admin: { mcp: false },
    })
  ).toBe(false)
  expect(
    isModuleEnabled(
      '/mcp-management',
      parseSidebarConfig({ admin: { enabled: false } }),
      null
    )
  ).toBe(false)
})

type BeforeLoadContext = NonNullable<
  Parameters<NonNullable<typeof Route.options.beforeLoad>>[0]
>

test.each([null, ROLE.USER])(
  '角色为 %s 时直接访问 MCP 管理路由重定向到无权限页',
  (role) => {
    useAuthStore
      .getState()
      .auth.setUser(role === null ? null : { id: 5, username: 'member', role })
    expect(() => Route.options.beforeLoad?.({} as BeforeLoadContext)).toThrow(
      expect.objectContaining({
        options: expect.objectContaining({ to: '/403' }),
      })
    )
  }
)

test.each([ROLE.ADMIN, ROLE.SUPER_ADMIN])(
  '角色为 %s 时允许访问 MCP 管理路由',
  (role) => {
    useAuthStore.getState().auth.setUser({ id: 5, username: 'admin', role })
    expect(Route.options.beforeLoad?.({} as BeforeLoadContext)).toBeUndefined()
  }
)
