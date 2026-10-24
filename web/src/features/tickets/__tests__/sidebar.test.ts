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
import { describe, expect, it } from 'vitest'

import { SIDEBAR_MODULES_DEFAULT } from '@/features/system-settings/maintenance/config'
import {
  filterSidebarNavGroups,
  parseSidebarConfig,
} from '@/hooks/use-sidebar-config'

describe('工单侧栏共享配置契约', () => {
  it('旧配置缺少工单项时补齐个人与管理默认入口', () => {
    const config = parseSidebarConfig({
      personal: { invoice: true },
      admin: { user: true },
    })
    expect(config.personal.tickets).toBe(true)
    expect(config.admin.ticket_management).toBe(true)
    expect(SIDEBAR_MODULES_DEFAULT.personal).toMatchObject({ tickets: true })
    expect(SIDEBAR_MODULES_DEFAULT.admin).toMatchObject({
      ticket_management: true,
    })
  })
  it('共享设置关闭管理入口时仍保留管理员个人工单入口', () => {
    const config = parseSidebarConfig({ admin: { ticket_management: false } })
    const groups = filterSidebarNavGroups(
      [
        {
          id: 'personal',
          title: 'Personal',
          items: [{ title: 'Ticket Center', url: '/tickets' }],
        },
        {
          id: 'admin',
          title: 'Admin',
          items: [{ title: 'Ticket Management', url: '/ticket-management' }],
        },
      ],
      config,
      null
    )
    expect(groups.map((group) => group.id)).toEqual(['personal'])
    expect(groups[0]?.items[0]).toMatchObject({ url: '/tickets' })
  })
})
