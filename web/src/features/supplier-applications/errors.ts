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

const supplierErrorKeys = new Set([
  'Authentication required',
  'Permission denied',
  'Supplier application service is unavailable',
  'Supplier application not found',
  'The same supplier application was already submitted',
  'Too many supplier applications in 24 hours',
  'Supplier application changed, please refresh',
  'Supplier application request exceeds 64 KiB',
])

const INVALID_PREFIX = 'invalid supplier application input'

// Only the server's public error contract is shown; unknown errors fall back
// to the caller's generic message. Validation errors collapse to one key.
export function getSupplierErrorKey(error: unknown, fallback: string): string {
  let message: string | undefined
  if (isAxiosError<{ message?: string }>(error)) {
    message = error.response?.data?.message
  } else if (error instanceof Error) {
    message = error.message
  }
  if (!message) return fallback
  if (message.startsWith(INVALID_PREFIX)) {
    return 'Some fields are invalid. Check the form and try again.'
  }
  return supplierErrorKeys.has(message) ? message : fallback
}

export function isSupplierDuplicateError(error: unknown): boolean {
  return (
    isAxiosError(error) &&
    error.response?.status === 409 &&
    getSupplierErrorKey(error, '') ===
      'The same supplier application was already submitted'
  )
}
