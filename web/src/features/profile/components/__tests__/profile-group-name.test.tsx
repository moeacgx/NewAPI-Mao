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
import { expect, test } from 'vitest'

import type { UserProfile } from '../../types'
import { ProfileHeader } from '../profile-header'

const profile: UserProfile = {
  id: 7,
  username: 'fixture-user',
  display_name: '测试用户',
  role: 1,
  status: 1,
  group: 'legacy-code',
  quota: 0,
  used_quota: 0,
  request_count: 0,
  aff_count: 0,
  aff_quota: 0,
  aff_history_quota: 0,
  created_time: 0,
}

test('展示当前名称并在刷新后更新', () => {
  const view = render(
    <ProfileHeader
      profile={{ ...profile, group_name: '当前分组' }}
      loading={false}
    />
  )
  expect(screen.getByText('当前分组')).toBeInTheDocument()
  expect(screen.queryByText('legacy-code')).not.toBeInTheDocument()
  view.rerender(
    <ProfileHeader
      profile={{ ...profile, group_name: '改名后分组' }}
      loading={false}
    />
  )
  expect(screen.getByText('改名后分组')).toBeInTheDocument()
  expect(screen.queryByText('当前分组')).not.toBeInTheDocument()
  expect(profile.group).toBe('legacy-code')
})

test.each([undefined, '', '   '])(
  '旧响应或空名称%j回退原标识',
  (group_name) => {
    render(
      <ProfileHeader profile={{ ...profile, group_name }} loading={false} />
    )
    expect(screen.getByText('legacy-code')).toBeInTheDocument()
  }
)
