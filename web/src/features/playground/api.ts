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

import { API_ENDPOINTS } from './constants'
import {
  createPlaygroundGroupOptions,
  type PlaygroundUserGroupMap,
  type UserGroupsResponse,
} from './lib/options/group-options'
import { preparePlaygroundRequest } from './lib/streaming/token-request'
import { parseGatewaySkillsHeader } from './lib/streaming/gateway-skills'
import type {
  ChatCompletionRequest,
  ChatCompletionResponse,
  GatewaySkillsMetadata,
  ModelOption,
  GroupOption,
} from './types'

export interface ChatCompletionResult {
  response: ChatCompletionResponse
  gatewaySkills?: GatewaySkillsMetadata
}

/**
 * Send chat completion request (non-streaming)
 */
export async function sendChatCompletion(
  payload: ChatCompletionRequest,
  signal?: AbortSignal,
  tokenId?: number
): Promise<ChatCompletionResult> {
  const request = preparePlaygroundRequest(payload, tokenId)
  const res = await api.post(API_ENDPOINTS.CHAT_COMPLETIONS, request.payload, {
    headers: request.headers,
    signal,
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return {
    response: res.data as ChatCompletionResponse,
    gatewaySkills: parseGatewaySkillsHeader(res.headers),
  }
}

export async function getTokenModels(tokenId: number): Promise<ModelOption[]> {
  const res = await api.get('/pg/token/models', {
    headers: { 'X-Playground-Token-Id': String(tokenId) },
  })
  if (res.data.success === false || !Array.isArray(res.data.data)) {
    throw new Error(
      res.data.error?.message ||
        res.data.message ||
        'Failed to load playground models'
    )
  }
  return res.data.data.map((model: { id: string }) => ({
    label: model.id,
    value: model.id,
  }))
}

/**
 * Get user available models
 */
export async function getUserModels(group: string): Promise<ModelOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_MODELS, {
    params: { group },
  })
  const { data } = res

  if (!data.success || !Array.isArray(data.data)) {
    return []
  }

  return data.data.map((model: string) => ({
    label: model,
    value: model,
  }))
}

/**
 * Get user groups
 */
export async function getUserGroups(): Promise<GroupOption[]> {
  const res = await api.get<UserGroupsResponse>(API_ENDPOINTS.USER_GROUPS)
  const { data } = res

  if (!data.success || !data.data) {
    return []
  }

  return createPlaygroundGroupOptions(data.data as PlaygroundUserGroupMap)
}

export async function getUserGroupsWithDefault(): Promise<{
  groups: GroupOption[]
  defaultGroup: string
}> {
  const res = await api.get<UserGroupsResponse>(API_ENDPOINTS.USER_GROUPS)
  const { data } = res

  if (!data.success || !data.data) {
    return { groups: [], defaultGroup: '' }
  }

  return {
    groups: createPlaygroundGroupOptions(data.data),
    defaultGroup: data.canvas_default_group?.trim() ?? '',
  }
}
