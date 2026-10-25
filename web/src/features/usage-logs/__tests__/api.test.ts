/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { afterEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { getAllTaskLogs, getUserTaskLogs } from '../api'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))

afterEach(() => {
  vi.clearAllMocks()
})

describe('任务日志 API 路径', () => {
  test('管理员列表使用后端注册的尾斜杠根路径', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { success: true } } as never)

    await getAllTaskLogs({ p: 1, page_size: 20 })

    expect(api.get).toHaveBeenCalledWith('/api/task/?p=1&page_size=20')
  })

  test('个人列表继续使用 self 路径', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { success: true } } as never)

    await getUserTaskLogs({ p: 1, page_size: 20 })

    expect(api.get).toHaveBeenCalledWith('/api/task/self?p=1&page_size=20')
  })
})
