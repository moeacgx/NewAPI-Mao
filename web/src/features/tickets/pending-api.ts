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

import type { TicketStatusCounts } from './pending-ticket-context'

type PendingTicketSummary =
  | { state: 'ready'; counts: TicketStatusCounts }
  | { state: 'denied' }
  | { state: 'error' }

const statusKeys = [
  'open',
  'processing',
  'resolved',
  'closed',
  'archived',
] as const

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
      data: Partial<TicketStatusCounts>
    }>('/api/ticket-admin/stats', config)
    const data = requireServerSuccess(response.data).data
    const counts = {} as TicketStatusCounts
    for (const key of statusKeys) {
      const value = data[key]
      if (
        typeof value !== 'number' ||
        !Number.isSafeInteger(value) ||
        value < 0
      ) {
        return { state: 'error' }
      }
      counts[key] = value
    }
    return { state: 'ready', counts }
  } catch (error) {
    if (
      isAxiosError(error) &&
      [401, 403].includes(error.response?.status ?? 0)
    ) {
      return { state: 'denied' }
    }
    // 可选概览失败只在卡片内展示，不触发全局查询错误页或阻断后台。
    return { state: 'error' }
  }
}
