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
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { filterSidebarNavGroups } from '@/hooks/use-sidebar-config'
import { useSidebarData } from '@/hooks/use-sidebar-data'

const session = vi.hoisted(() => ({ auth: { user: { role: 10 } } }))
vi.mock('@/stores/auth-store', () => ({
  useAuthStore: Object.assign(
    (selector: (state: typeof session) => unknown) => selector(session),
    { getState: () => session }
  ),
}))
vi.mock('@/hooks/use-status', () => ({ useStatus: () => ({ status: null }) }))
vi.mock('@tanstack/react-query', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-query')>()),
  useQuery: () => ({ data: undefined }),
}))
vi.mock('@/lib/api', () => ({ api: {} }))

describe('返佣管理员入口', () => {
  it.each([10, 100])('role %s 的业务入口独立于 Root 系统设置', (role) => {
    session.auth.user.role = role
    const { result } = renderHook(() => useSidebarData())
    const entry = result.current.navGroups
      .find((group) => group.id === 'admin')
      ?.items.find((item) => item.title === 'Affiliate Commission')
    expect(entry).toMatchObject({
      url: '/affiliate-management',
      requiredRole: 10,
    })
  })

  it('原 affiliate_admin 开关关闭时新业务入口继续隐藏', () => {
    session.auth.user.role = 10
    const { result } = renderHook(() => useSidebarData())
    const filtered = filterSidebarNavGroups(
      result.current.navGroups,
      {
        admin: { enabled: true, affiliate_admin: false },
      },
      null
    )
    expect(
      filtered
        .find((group) => group.id === 'admin')
        ?.items.some((item) => item.title === 'Affiliate Commission')
    ).toBe(false)
  })
})
