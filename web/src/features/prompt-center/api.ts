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
  PromptResponse,
  PromptTemplate,
  PromptTemplateInput,
  TokenPromptBinding,
} from './types'

export async function getPromptTemplates(
  admin: boolean
): Promise<PromptTemplate[]> {
  const response = await api.get<PromptResponse<PromptTemplate[]>>(
    admin ? '/api/prompt-center/admin/' : '/api/prompt-center/'
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to load prompts')
  }
  return response.data.data ?? []
}

export async function savePromptTemplate(
  input: PromptTemplateInput,
  id?: number
): Promise<void> {
  const response = id
    ? await api.put<PromptResponse<PromptTemplate>>(
        `/api/prompt-center/admin/${id}`,
        input
      )
    : await api.post<PromptResponse<PromptTemplate>>(
        '/api/prompt-center/admin/',
        input
      )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to save prompt')
  }
}

export async function getTokenPrompts(
  tokenId: number
): Promise<TokenPromptBinding[]> {
  const response = await api.get<PromptResponse<TokenPromptBinding[]>>(
    `/api/token/${tokenId}/prompts`
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to load installed prompts')
  }
  return response.data.data ?? []
}

export async function saveTokenPrompts(
  tokenId: number,
  bindings: TokenPromptBinding[]
): Promise<void> {
  const response = await api.put<PromptResponse<unknown>>(
    `/api/token/${tokenId}/prompts`,
    {
      bindings: bindings.map((binding, index) => ({
        template_id: binding.template.id,
        enabled: binding.enabled,
        sort_order: index,
      })),
    }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to save installations')
  }
}
