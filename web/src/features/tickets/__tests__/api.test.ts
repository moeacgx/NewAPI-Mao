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
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { createTicket, getTickets } from '../api'
import { getTicketErrorKey } from '../errors'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn(), post: vi.fn() } }))
afterEach(() => vi.resetAllMocks())

describe('工单失败响应契约', () => {
  it('HTTP 成功但业务失败时列表查询拒绝而非显示空列表', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { success: false, message: 'Permission denied' },
    })
    await expect(getTickets(true, { p: 1, page_size: 20 })).rejects.toThrow(
      'Permission denied'
    )
  })
  it('创建业务失败时拒绝并保留可翻译的公开错误', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { success: false, message: 'Ticket request exceeds 64 KiB' },
    })
    await expect(
      createTicket({
        title: 'Title',
        category: 'general',
        priority: 'normal',
        content: 'Content',
      })
    ).rejects.toThrow('Ticket request exceeds 64 KiB')
    expect(
      getTicketErrorKey(
        new Error('Ticket request exceeds 64 KiB'),
        'Request failed'
      )
    ).toBe('Ticket request exceeds 64 KiB')
  })
  it('未知内部错误不直接泄露到工单界面', () => {
    expect(
      getTicketErrorKey(
        new Error('database password: private'),
        'Request failed'
      )
    ).toBe('Request failed')
  })
})
