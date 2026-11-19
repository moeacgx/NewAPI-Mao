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
import { afterEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { searchApiKeys } from '../api'

afterEach(() => vi.restoreAllMocks())

describe('令牌服务端筛选契约', () => {
  test.each([1, 2, 3, 4])(
    '仅状态 %i 仍发送搜索条件和服务端分页',
    async (status) => {
      const get = vi
        .spyOn(api, 'get')
        .mockResolvedValue({ data: { success: true } })
      await searchApiKeys({ status, p: 2, size: 20 })
      const url = new URL(get.mock.calls[0][0], 'http://localhost')
      expect(url.pathname).toBe('/api/token/search')
      expect(url.searchParams.get('status')).toBe(String(status))
      expect(url.searchParams.get('p')).toBe('2')
      expect(url.searchParams.get('size')).toBe('20')
    }
  )

  test('活动、状态、名称和密钥条件同时保留且正确编码', async () => {
    const get = vi
      .spyOn(api, 'get')
      .mockResolvedValue({ data: { success: true } })
    await searchApiKeys({
      keyword: '同名 & Claude',
      token: 'sk-a+b',
      status: 2,
      active_only: true,
      p: 1,
      size: 10,
    })
    const url = new URL(get.mock.calls[0][0], 'http://localhost')
    expect(Object.fromEntries(url.searchParams)).toEqual({
      keyword: '同名 & Claude',
      token: 'sk-a+b',
      status: '2',
      active_only: 'true',
      p: '1',
      size: '10',
    })
  })

  test('关闭活动筛选与清除状态后不遗留服务端条件', async () => {
    const get = vi
      .spyOn(api, 'get')
      .mockResolvedValue({ data: { success: true } })
    await searchApiKeys({ keyword: 'Claude', active_only: false })
    const url = new URL(get.mock.calls[0][0], 'http://localhost')
    expect(url.searchParams.has('active_only')).toBe(false)
    expect(url.searchParams.has('status')).toBe(false)
  })
})
