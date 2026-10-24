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
import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

import type { CreateTicketInput, TicketReplyInput } from './schema'
import type {
  Ticket,
  TicketAutoArchiveConfig,
  TicketDetail,
  TicketFilters,
  TicketList,
  TicketMessage,
  TicketStatus,
  TicketVersion,
} from './types'

interface TicketResponse<T> {
  success: boolean
  message: string
  data: T
}

export interface TicketAutoReplyConfig {
  enabled: boolean
  content: string
}

export async function getTickets(
  admin: boolean,
  filters: TicketFilters,
  signal?: AbortSignal
): Promise<TicketList> {
  const response = await api.get<TicketResponse<TicketList>>(
    admin ? '/api/ticket-admin' : '/api/tickets',
    { params: filters, signal }
  )
  return requireServerSuccess(response.data).data
}

export async function getTicket(
  admin: boolean,
  id: number,
  beforeMessageId?: number,
  signal?: AbortSignal
): Promise<TicketDetail> {
  const response = await api.get<TicketResponse<TicketDetail>>(
    `${admin ? '/api/ticket-admin' : '/api/tickets'}/${id}`,
    {
      params: beforeMessageId
        ? { before_message_id: beforeMessageId }
        : undefined,
      signal,
    }
  )
  return requireServerSuccess(response.data).data
}

export async function createTicket(input: CreateTicketInput): Promise<Ticket> {
  const response = await api.post<TicketResponse<Ticket>>('/api/tickets', input)
  return requireServerSuccess(response.data).data
}

export async function replyToTicket(
  admin: boolean,
  id: number,
  input: TicketReplyInput
): Promise<TicketMessage> {
  const response = await api.post<TicketResponse<TicketMessage>>(
    `${admin ? '/api/ticket-admin' : '/api/tickets'}/${id}/messages`,
    input
  )
  return requireServerSuccess(response.data).data
}

export async function updateTicketStatus(
  admin: boolean,
  id: number,
  status: TicketStatus
): Promise<Ticket> {
  const response = await api.put<TicketResponse<Ticket>>(
    `${admin ? '/api/ticket-admin' : '/api/tickets'}/${id}/status`,
    { status }
  )
  return requireServerSuccess(response.data).data
}

export async function getTicketAutoReplyConfig(
  signal?: AbortSignal
): Promise<TicketAutoReplyConfig> {
  const response = await api.get<TicketResponse<TicketAutoReplyConfig>>(
    '/api/ticket-admin/automation/auto-reply',
    { signal }
  )
  return requireServerSuccess(response.data).data
}

export async function updateTicketAutoReplyConfig(
  config: TicketAutoReplyConfig
): Promise<TicketAutoReplyConfig> {
  const response = await api.put<TicketResponse<TicketAutoReplyConfig>>(
    '/api/ticket-admin/automation/auto-reply',
    config
  )
  return requireServerSuccess(response.data).data
}

export async function batchUpdateTicketStatus(input: {
  tickets: TicketVersion[]
  status: TicketStatus
}): Promise<{ updated_count: number }> {
  const response = await api.put<TicketResponse<{ updated_count: number }>>(
    '/api/ticket-admin/batch/status',
    input
  )
  return requireServerSuccess(response.data).data
}

export async function batchDeleteTickets(
  tickets: TicketVersion[]
): Promise<{ deleted_count: number }> {
  const response = await api.post<TicketResponse<{ deleted_count: number }>>(
    '/api/ticket-admin/batch/delete',
    { tickets }
  )
  return requireServerSuccess(response.data).data
}

export async function getTicketAutoArchiveConfig(
  signal?: AbortSignal
): Promise<TicketAutoArchiveConfig> {
  const response = await api.get<TicketResponse<TicketAutoArchiveConfig>>(
    '/api/ticket-admin/automation/auto-archive',
    { signal }
  )
  return requireServerSuccess(response.data).data
}

export async function updateTicketAutoArchiveConfig(
  config: TicketAutoArchiveConfig
): Promise<TicketAutoArchiveConfig> {
  const response = await api.put<TicketResponse<TicketAutoArchiveConfig>>(
    '/api/ticket-admin/automation/auto-archive',
    config
  )
  return requireServerSuccess(response.data).data
}
