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

import type { GroupTag, GroupTagInput } from './types'

type TagResponse<T> = { success: boolean; message?: string; data: T }

export async function getGroupTags(): Promise<GroupTag[]> {
  const { data } = await api.get<TagResponse<GroupTag[]>>('/api/group/tags')
  if (!data.success) throw new Error(data.message)
  return data.data ?? []
}

export async function saveGroupTag(
  input: GroupTagInput,
  id?: number
): Promise<GroupTag> {
  const { data } = id
    ? await api.put<TagResponse<GroupTag>>(`/api/group/tags/${id}`, input)
    : await api.post<TagResponse<GroupTag>>('/api/group/tags', input)
  if (!data.success) throw new Error(data.message)
  return data.data
}

export async function deleteGroupTag(id: number): Promise<void> {
  const { data } = await api.delete<TagResponse<null>>(`/api/group/tags/${id}`)
  if (!data.success) throw new Error(data.message)
}
