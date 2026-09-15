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
import { api, type ApiRequestConfig } from '@/lib/api'

import type {
  ApiResponse,
  TaskPluginDetail,
  TaskPluginListItem,
  TaskPluginRecord,
} from './types'

const mutationConfig: ApiRequestConfig = {
  skipBusinessError: true,
  skipErrorHandler: true,
}

function requireSuccess<T>(response: ApiResponse<T>): T {
  if (!response || response.success !== true) {
    throw new Error(response?.message || 'Unexpected server response')
  }
  return response.data
}

export async function listTaskPlugins() {
  const response =
    await api.get<ApiResponse<TaskPluginListItem[]>>('/api/plugin/task')
  return requireSuccess(response.data).map((plugin) => ({
    ...plugin,
    source: 'factory' as const,
    has_icon: false,
  }))
}

export async function getTaskPlugin(key: string, version?: string) {
  const response = await api.get<ApiResponse<TaskPluginDetail>>(
    `/api/plugin/task/${encodeURIComponent(key)}`,
    { params: version ? { version } : undefined }
  )
  return {
    ...requireSuccess(response.data),
    layer: 'factory' as const,
    has_icon: false,
  }
}

export async function getTaskPluginVersions(key: string) {
  const response = await api.get<ApiResponse<TaskPluginRecord[]>>(
    `/api/plugin/task/${encodeURIComponent(key)}/versions`
  )
  return requireSuccess(response.data)
}

export async function activateTaskPlugin(key: string, version: string) {
  const response = await api.post<ApiResponse<null>>(
    `/api/plugin/task/${encodeURIComponent(key)}/activate`,
    { version },
    mutationConfig
  )
  requireSuccess(response.data)
}

export async function setTaskPluginStatus(key: string, enabled: boolean) {
  const response = await api.post<ApiResponse<null>>(
    `/api/plugin/task/${encodeURIComponent(key)}/status`,
    { enabled },
    mutationConfig
  )
  requireSuccess(response.data)
}

export type TaskPluginRuntime = {
  enabled: boolean
  channel_type: 62
  builtin_only: boolean
  production_ready: boolean
}

export async function getTaskPluginEnabledOption() {
  const response = await api.get<ApiResponse<TaskPluginRuntime>>(
    '/api/plugin/task/runtime/status'
  )
  return requireSuccess(response.data).enabled
}

export async function setTaskPluginEnabledOption(enabled: boolean) {
  const response = await api.put<ApiResponse<TaskPluginRuntime>>(
    '/api/plugin/task/runtime/status',
    { enabled },
    mutationConfig
  )
  return requireSuccess(response.data)
}

export type TaskPluginOption = {
  key: string
  name: string
  version: string
  models: string[]
  channel_type: number
}
export async function getTaskPluginOptions() {
  const response = await api.get<ApiResponse<TaskPluginOption[]>>(
    '/api/task_plugin_options'
  )
  return requireSuccess(response.data)
}
