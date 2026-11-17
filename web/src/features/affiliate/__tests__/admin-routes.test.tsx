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
import { describe, expect, it, vi } from 'vitest'

import { Route as ManagementRoute } from '@/routes/_authenticated/affiliate-management'
import { Route as SettingsRoute } from '@/routes/_authenticated/system-settings/route'

const session = vi.hoisted(() => ({
  auth: { user: { role: 1 } as { role: number } | null },
}))
vi.mock('@/stores/auth-store', () => ({
  useAuthStore: Object.assign(
    (selector: (state: typeof session) => unknown) => selector(session),
    { getState: () => session }
  ),
}))

describe('返佣路由权限边界', () => {
  it.each([
    { role: null, business: false, settings: false },
    { role: 1, business: false, settings: false },
    { role: 10, business: true, settings: false },
    { role: 100, business: true, settings: true },
  ])(
    'role $role 业务访问 $business、Root 设置访问 $settings',
    ({ role, business, settings }) => {
      session.auth.user = role === null ? null : { role }
      for (const [route, allowed] of [
        [ManagementRoute, business],
        [SettingsRoute, settings],
      ] as const) {
        const guard = route.options.beforeLoad as () => void
        if (allowed) {
          expect(guard).not.toThrow()
        } else {
          expect(guard).toThrow(
            expect.objectContaining({
              options: expect.objectContaining({ to: '/403' }),
            })
          )
        }
      }
    }
  )
})
