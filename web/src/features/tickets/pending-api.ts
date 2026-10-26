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
import { isAxiosError } from 'axios'

import { hasPermission } from '@/lib/admin-permissions'
import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { requireServerSuccess } from '@/lib/server-error-message'
import type { AuthUser } from '@/stores/auth-store'

type PendingTicketSummary =
  | { state: 'ready'; total: number }
  | { state: 'denied' }
  | { state: 'error' }

export async function getPendingTicketSummary(
  userId: number,
  signal: AbortSignal
): Promise<PendingTicketSummary> {
  const config = {
    signal,
    timeout: 10000,
    skipErrorHandler: true,
    skipBusinessError: true,
  }
  try {
    const self = await api.get<{
      success: boolean
      message: string
      data: AuthUser
    }>('/api/user/self', config)
    const user = requireServerSuccess(self.data).data
    if (
      signal.aborted ||
      user.id !== userId ||
      user.role < ROLE.ADMIN ||
      !hasPermission(user, 'tickets', 'read')
    ) {
      return { state: 'denied' }
    }
    const response = await api.get<{
      success: boolean
      message: string
      data: { total: number }
    }>('/api/ticket-admin', {
      ...config,
      params: { p: 1, page_size: 1, status: 'open' },
    })
    const total = requireServerSuccess(response.data).data.total
    if (!Number.isSafeInteger(total) || total < 0) return { state: 'error' }
    return { state: 'ready', total }
  } catch (error) {
    if (
      isAxiosError(error) &&
      [401, 403].includes(error.response?.status ?? 0)
    ) {
      return { state: 'denied' }
    }
    // 可选提醒失败只在卡片内展示，不触发全局查询错误页或阻断后台。
    return { state: 'error' }
  }
}
