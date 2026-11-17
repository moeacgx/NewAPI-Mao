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
import { act, render, screen } from '@testing-library/react'
import { afterEach, expect, test } from 'vitest'

import { api } from '@/lib/api'

import { UserInfoDialog } from '../user-info-dialog'

const adapter = api.defaults.adapter
afterEach(() => {
  api.defaults.adapter = adapter
})

function respond(group_name?: string) {
  api.defaults.adapter = async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: {
      success: true,
      data: {
        id: Number(config.url?.split('/').pop()),
        username: 'fixture-user',
        group: 'legacy-code',
        group_name,
        quota: 0,
        used_quota: 0,
        request_count: 0,
      },
    },
  })
}

test('展示当前名称并在刷新后更新', async () => {
  respond('当前分组')
  const view = render(
    <UserInfoDialog userId={7} open onOpenChange={() => {}} />
  )
  expect(await screen.findByText('当前分组')).toBeInTheDocument()
  expect(screen.queryByText('legacy-code')).not.toBeInTheDocument()
  respond('另一用户分组')
  view.rerender(<UserInfoDialog userId={8} open onOpenChange={() => {}} />)
  expect(await screen.findByText('另一用户分组')).toBeInTheDocument()
  expect(screen.queryByText('当前分组')).not.toBeInTheDocument()
})

test.each([undefined, '', '   '])(
  '旧响应或空名称%j回退原标识',
  async (group_name) => {
    respond(group_name)
    render(<UserInfoDialog userId={7} open onOpenChange={() => {}} />)
    expect(await screen.findByText('legacy-code')).toBeInTheDocument()
  }
)

test('切换用户后旧请求晚到不能覆盖当前用户名称', async () => {
  let finishFirst: () => void = () => {}
  const first = new Promise<void>((resolve) => {
    finishFirst = resolve
  })
  api.defaults.adapter = async (config) => {
    if (config.url === '/api/user/7') await first
    return {
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: {
        success: true,
        data: {
          id: config.url === '/api/user/7' ? 7 : 8,
          username: 'fixture-user',
          group: 'legacy-code',
          group_name: config.url === '/api/user/7' ? '旧用户组' : '当前用户组',
          quota: 0,
          used_quota: 0,
          request_count: 0,
        },
      },
    }
  }
  const view = render(
    <UserInfoDialog userId={7} open onOpenChange={() => {}} />
  )
  view.rerender(<UserInfoDialog userId={8} open onOpenChange={() => {}} />)
  expect(await screen.findByText('当前用户组')).toBeInTheDocument()
  await act(async () => {
    finishFirst()
    await first
  })
  expect(screen.getByText('当前用户组')).toBeInTheDocument()
  expect(screen.queryByText('旧用户组')).not.toBeInTheDocument()
})
