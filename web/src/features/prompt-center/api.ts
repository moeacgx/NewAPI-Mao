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
  PromptCategory,
  PromptResponse,
  PromptTemplate,
  PromptTemplateInput,
  SkillStoreEntry,
  TokenPromptBinding,
} from './types'

export async function getSkillStore(
  online = false
): Promise<SkillStoreEntry[]> {
  const response = await api.get<PromptResponse<SkillStoreEntry[]>>(
    online
      ? '/api/prompt-center/admin/store/?source=online'
      : '/api/prompt-center/admin/store/'
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to load skill store')
  }
  return response.data.data ?? []
}

export async function importSkillStoreEntry(
  id: string
): Promise<PromptTemplate> {
  const response = await api.post<PromptResponse<PromptTemplate>>(
    `/api/prompt-center/admin/store/${encodeURIComponent(id)}/import`
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to import skills')
  }
  return { ...response.data.data, id: 0 }
}

export async function importSkillArchive(file: File): Promise<PromptTemplate> {
  const body = new FormData()
  body.append('file', file)
  const response = await api.post<PromptResponse<PromptTemplate>>(
    '/api/prompt-center/admin/import/archive',
    body
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to import skills')
  }
  return { ...response.data.data, id: 0 }
}

export async function importSkillRepository(
  url: string
): Promise<PromptTemplate> {
  const response = await api.post<PromptResponse<PromptTemplate>>(
    '/api/prompt-center/admin/import/repository',
    { url: url.trim() }
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || 'Failed to import skills')
  }
  return { ...response.data.data, id: 0 }
}

export async function getPromptCategories(): Promise<PromptCategory[]> {
  const response = await api.get<PromptResponse<PromptCategory[]>>(
    '/api/prompt-center/categories/'
  )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to load skill categories')
  }
  return response.data.data ?? []
}

export async function savePromptCategory(
  input: { name: string; version: number },
  id?: number
): Promise<void> {
  const response = id
    ? await api.put<PromptResponse<PromptCategory>>(
        `/api/prompt-center/admin/categories/${id}`,
        input
      )
    : await api.post<PromptResponse<PromptCategory>>(
        '/api/prompt-center/admin/categories/',
        input
      )
  if (!response.data.success) {
    throw new Error(response.data.message || 'Failed to save skill category')
  }
}

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
