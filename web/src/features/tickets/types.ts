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
export type TicketCategory = 'general' | 'billing' | 'technical' | 'account'
export type TicketPriority = 'low' | 'normal' | 'high'
export type TicketStatus =
  | 'open'
  | 'processing'
  | 'resolved'
  | 'closed'
  | 'archived'

export type TicketVersion = Pick<Ticket, 'id' | 'version'>

export interface TicketAutoArchiveConfig {
  enabled: boolean
  inactive_days: number
  statuses: Exclude<TicketStatus, 'archived'>[]
}

export interface Ticket {
  id: number
  user_id: number
  title: string
  category: TicketCategory
  priority: TicketPriority
  status: TicketStatus
  version: number
  created_at: number
  updated_at: number
}

export interface TicketMessage {
  id: number
  ticket_id: number
  author_id: number
  author_role: 'user' | 'admin'
  content: string
  created_at: number
}

export interface TicketDetail {
  ticket: Ticket
  messages: TicketMessage[]
  has_more: boolean
}

export interface TicketList {
  page: number
  page_size: number
  total: number
  items: Ticket[]
}

export interface TicketFilters {
  p: number
  page_size: number
  keyword?: string
  status?: TicketStatus | ''
  user_id?: number
}
