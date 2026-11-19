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
export interface McpTool {
  name: string
  description: string
  category: string
  write: boolean
  min_role: number
  inputSchema: Record<string, unknown>
}

export interface McpKey {
  id: number
  name: string
  key_prefix: string
  tools: string[]
  created_at: number
  expires_at: number
  last_used_at: number
  revoked_at: number
}

export interface McpLog {
  id: number
  key_id: number
  tool_name: string
  request_id: string
  success: boolean
  duration_ms: number
  created_at: number
}

export interface McpPage<T> {
  items: T[]
  total: number
}

export interface CreateMcpKeyInput {
  name: string
  tools: string[]
  expires_at: number
}

export interface McpKeyReceipt {
  key: McpKey
  secret: string
}
