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

import {
  parseSupplierRatio,
  type SupplierApplicationFormValues,
} from './schema'
import type {
  SupplierApplication,
  SupplierApplicationAdmin,
  SupplierApplicationFilters,
  SupplierApplicationList,
  SupplierApplicationReviewInput,
  SupplierMessage,
  SupplierMessageInput,
} from './types'

interface SupplierResponse<T> {
  success: boolean
  message: string
  data: T
}

const PERSONAL_PATH = '/api/supplier-applications'
const ADMIN_PATH = '/api/supplier-application-admin'

export async function getSupplierApplications(
  filters: SupplierApplicationFilters,
  signal?: AbortSignal
): Promise<SupplierApplicationList<SupplierApplication>> {
  const response = await api.get<
    SupplierResponse<SupplierApplicationList<SupplierApplication>>
  >(PERSONAL_PATH, { params: filters, signal })
  return requireServerSuccess(response.data).data
}

export async function createSupplierApplicationMessage(
  id: number,
  input: SupplierMessageInput
): Promise<SupplierMessage> {
  const response = await api.post<SupplierResponse<SupplierMessage>>(
    `${PERSONAL_PATH}/${id}/messages`,
    input,
    { skipErrorHandler: true }
  )
  return requireServerSuccess(response.data).data
}

export async function markSupplierApplicationRead(
  id: number
): Promise<{ unread_count: number }> {
  const response = await api.post<SupplierResponse<{ unread_count: number }>>(
    `${PERSONAL_PATH}/${id}/read`,
    {},
    { skipErrorHandler: true }
  )
  return requireServerSuccess(response.data).data
}

export async function markAdminSupplierApplicationRead(
  id: number
): Promise<{ unread_count: number }> {
  const response = await api.post<SupplierResponse<{ unread_count: number }>>(
    `${ADMIN_PATH}/${id}/read`,
    {},
    { skipErrorHandler: true }
  )
  return requireServerSuccess(response.data).data
}

export async function createAdminSupplierApplicationMessage(
  id: number,
  input: SupplierMessageInput
): Promise<SupplierMessage> {
  const response = await api.post<SupplierResponse<SupplierMessage>>(
    `${ADMIN_PATH}/${id}/messages`,
    input,
    { skipErrorHandler: true }
  )
  return requireServerSuccess(response.data).data
}

export async function getAdminSupplierApplications(
  filters: SupplierApplicationFilters,
  signal?: AbortSignal
): Promise<SupplierApplicationList<SupplierApplicationAdmin>> {
  const response = await api.get<
    SupplierResponse<SupplierApplicationList<SupplierApplicationAdmin>>
  >(ADMIN_PATH, { params: filters, signal })
  return requireServerSuccess(response.data).data
}

export async function getAdminSupplierApplication(
  id: number,
  signal?: AbortSignal
): Promise<SupplierApplicationAdmin> {
  const response = await api.get<SupplierResponse<SupplierApplicationAdmin>>(
    `${ADMIN_PATH}/${id}`,
    { signal }
  )
  return requireServerSuccess(response.data).data
}

// The form keeps ratios as text; convert to numbers only at the wire boundary.
export async function createSupplierApplication(
  values: SupplierApplicationFormValues
): Promise<SupplierApplication> {
  const payload = {
    ...values,
    groups: values.groups.map((group) => ({
      ...group,
      ratio: parseSupplierRatio(group.ratio),
    })),
  }
  const response = await api.post<SupplierResponse<SupplierApplication>>(
    PERSONAL_PATH,
    payload,
    // Errors (including duplicate 409) are presented by the form itself.
    { skipErrorHandler: true }
  )
  return requireServerSuccess(response.data).data
}

export async function reviewSupplierApplication(
  id: number,
  input: SupplierApplicationReviewInput
): Promise<SupplierApplicationAdmin> {
  const response = await api.put<SupplierResponse<SupplierApplicationAdmin>>(
    `${ADMIN_PATH}/${id}`,
    input,
    { skipErrorHandler: true }
  )
  return requireServerSuccess(response.data).data
}
