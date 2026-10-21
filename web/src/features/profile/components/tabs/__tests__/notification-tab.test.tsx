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
import { render, screen } from '@testing-library/react'
import { describe, expect, test } from 'vitest'

import { ROLE } from '@/lib/roles'

import type { UserProfile } from '../../../types'
import { NotificationTab } from '../notification-tab'

const profile: UserProfile = {
  id: 1,
  username: 'notification-user',
  display_name: '通知测试',
  role: ROLE.ADMIN,
  group: 'default',
  quota: 50,
  used_quota: 0,
  request_count: 0,
  status: 1,
  aff_count: 0,
  aff_quota: 0,
  aff_history_quota: 0,
  created_time: 0,
}

describe('个人通知范围', () => {
  test('管理员渠道开关只说明全渠道测试，并明确自动启停归通知中心', () => {
    render(
      <NotificationTab
        profile={profile}
        onUpdate={() => undefined}
        quotaRemindEnabled
        quotaRemindThreshold={100}
      />
    )

    expect(
      screen.getByRole('switch', { name: 'All-channel test notifications' })
    ).not.toBeChecked()
    expect(
      screen.getByText(
        'Notify me when a test of all channels finishes. Automatic channel status notifications are managed in the notification center.'
      )
    ).toBeVisible()
    expect(
      screen.getByText(
        'Reminders are sent below this value, subject to notification rate limits'
      )
    ).toBeVisible()
  })

  test('普通用户没有管理员通知开关，站点关闭余额提醒时不显示个人余额开关', () => {
    render(
      <NotificationTab
        profile={{ ...profile, role: ROLE.USER }}
        onUpdate={() => undefined}
        quotaRemindEnabled={false}
        quotaRemindThreshold={100}
      />
    )

    expect(
      screen.queryByRole('switch', { name: 'All-channel test notifications' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('switch', { name: 'Low balance reminder' })
    ).not.toBeInTheDocument()
  })
})
