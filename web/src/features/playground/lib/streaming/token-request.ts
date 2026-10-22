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
import type { ChatCompletionRequest } from '../../types'

// 账号登录态仍由统一请求层维护，令牌模式只传递本人令牌 ID。
export function preparePlaygroundRequest(
  payload: ChatCompletionRequest,
  tokenId?: number
): { payload: ChatCompletionRequest; headers: Record<string, string> } {
  if (!tokenId) return { payload, headers: {} }
  const tokenPayload = { ...payload }
  delete tokenPayload.group
  return {
    payload: tokenPayload,
    headers: { 'X-Playground-Token-Id': String(tokenId) },
  }
}
