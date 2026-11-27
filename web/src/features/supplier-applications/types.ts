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
export type SupplierApplicationStatus =
  | 'pending'
  | 'contacted'
  | 'cooperating'
  | 'rejected'

export type SupplierContactType =
  | 'wechat'
  | 'qq'
  | 'telegram'
  | 'email'
  | 'phone'
  | 'other'

export type SupplierPlatform =
  | 'openai'
  | 'anthropic'
  | 'gemini'
  | 'grok'
  | 'deepseek'
  | 'other'

export interface SupplierQuote {
  id: number
  group_name: string
  platform: SupplierPlatform
  ratio: number
  models: string[]
  description: string
}

export type SupplierMessageType =
  | 'message'
  | 'negotiation'
  | 'offer'
  | 'test_quota'

export type SupplierQuotaStatus = 'pending' | 'issued'

export type SupplierMessageAuthorRole = 'applicant' | 'admin'

// Conversation records never include internal notes or account changes.
export interface SupplierMessage {
  id: number
  author_id: number
  author_role: SupplierMessageAuthorRole
  author_name: string
  type: SupplierMessageType
  body: string
  group_name?: string
  ratio?: number
  quantity?: number
  unit?: string
  quota_status?: SupplierQuotaStatus
  created_at: number
}

export interface SupplierMessageInput {
  type: SupplierMessageType
  body: string
  group_name?: string
  ratio?: number
  quantity?: number
  unit?: string
  quota_status?: SupplierQuotaStatus
}

// Applicant-visible shape: never carries internal notes or handler data.
export interface SupplierApplication {
  id: number
  site_name: string
  site_url: string
  contact_type: SupplierContactType
  contact: string
  remark: string
  status: SupplierApplicationStatus
  created_at: number
  groups: SupplierQuote[]
  messages?: SupplierMessage[]
}

export interface SupplierApplicationAdmin extends SupplierApplication {
  user_id: number
  username: string
  admin_note: string
  handled_by: number
  handled_by_name: string
  handled_at: number
  version: number
  updated_at: number
}

export interface SupplierApplicationFilters {
  p: number
  page_size: number
  status?: SupplierApplicationStatus | ''
  keyword?: string
  user_id?: number
}

export interface SupplierApplicationList<T> {
  page: number
  page_size: number
  total: number
  items: T[]
}

export interface SupplierApplicationReviewInput {
  status: SupplierApplicationStatus
  admin_note: string
  version: number
}
