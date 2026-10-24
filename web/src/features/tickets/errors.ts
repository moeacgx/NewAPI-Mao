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

const ticketErrorKeys = new Set([
  'Authentication required',
  'Permission denied',
  'Ticket service is unavailable',
  'Invalid ticket input',
  'Ticket not found',
  'Reopen the ticket before replying',
  'Ticket changed, please refresh',
  'Ticket request exceeds 64 KiB',
])

// 仅展示服务端公开约定的错误，未知网络或内部错误统一使用界面提示。
export function getTicketErrorKey(error: unknown, fallback: string): string {
  let message: string | undefined
  if (isAxiosError<{ message?: string }>(error)) {
    message = error.response?.data?.message
  } else if (error instanceof Error) {
    message = error.message
  }
  return message && ticketErrorKeys.has(message) ? message : fallback
}
