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

import type {
  CreateMcpKeyInput,
  McpKey,
  McpKeyReceipt,
  McpLog,
  McpPage,
  McpTool,
} from './types'

interface McpResponse<T> {
  success: boolean
  message: string
  data: T
}

function unwrap<T>(response: McpResponse<T>): T {
  if (!response.success) throw new Error(response.message || 'Request failed')
  return response.data
}

export const MCP_PAGE_SIZE = 20

export async function listMcpTools(): Promise<McpTool[]> {
  const response = await api.get<McpResponse<McpTool[]>>('/api/mcp/tools')
  return unwrap(response.data)
}

export async function listMcpKeys(page: number): Promise<McpPage<McpKey>> {
  const response = await api.get<McpResponse<McpPage<McpKey>>>(
    '/api/mcp/keys',
    {
      params: { p: page, page_size: MCP_PAGE_SIZE },
    }
  )
  return unwrap(response.data)
}

export async function createMcpKey(
  input: CreateMcpKeyInput
): Promise<McpKeyReceipt> {
  const response = await api.post<McpResponse<McpKeyReceipt>>(
    '/api/mcp/keys',
    input
  )
  return unwrap(response.data)
}

export async function revokeMcpKey(id: number): Promise<void> {
  const response = await api.delete<McpResponse<unknown>>(`/api/mcp/keys/${id}`)
  unwrap(response.data)
}

export async function listMcpLogs(page: number): Promise<McpPage<McpLog>> {
  const response = await api.get<McpResponse<McpPage<McpLog>>>(
    '/api/mcp/logs',
    {
      params: { p: page, page_size: MCP_PAGE_SIZE },
    }
  )
  return unwrap(response.data)
}
