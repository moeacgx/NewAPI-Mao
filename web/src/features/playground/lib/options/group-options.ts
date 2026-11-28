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
import type { GroupOption } from '../../types'

export interface PlaygroundUserGroupInfo {
  name?: string
  desc?: string
  ratio: number
}

export type UserGroupsResponse = {
  success: boolean
  message?: string
  data?: PlaygroundUserGroupMap
  canvas_default_group?: string
  canvas_text_default_group?: string
  canvas_image_default_group?: string
  canvas_audio_default_group?: string
  canvas_video_default_group?: string
}

export type PlaygroundUserGroupMap = Record<string, PlaygroundUserGroupInfo>

export type CanvasUserGroups = {
  groups: GroupOption[]
  defaultGroup: string
  textGroup: string
  imageGroup: string
  audioGroup: string
  videoGroup: string
}

const emptyCanvasUserGroups = (): CanvasUserGroups => ({
  groups: [],
  defaultGroup: '',
  textGroup: '',
  imageGroup: '',
  audioGroup: '',
  videoGroup: '',
})

function readOptionalText(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function getGroupDescription(
  description: string | undefined,
  code: string,
  name: string
): string | undefined {
  const normalized = description?.trim()
  if (!normalized || normalized === code || normalized === name) {
    return undefined
  }
  return normalized
}

export function createPlaygroundGroupOptions(
  groups: PlaygroundUserGroupMap
): GroupOption[] {
  return Object.entries(groups).map(([code, info]) => {
    // 个别条目缺失时保留该分组 code，不能让一次空值把已返回的分组整表丢掉。
    const record =
      info && typeof info === 'object' ? info : ({} as PlaygroundUserGroupInfo)
    const name = readOptionalText(record.name) || code
    return {
      label: name,
      value: code,
      ratio: record.ratio,
      desc: getGroupDescription(readOptionalText(record.desc), code, name),
    }
  })
}

// 画布页只消费用户分组接口里已经返回、且当前用户可用的分组。
export function readCanvasUserGroups(payload: unknown): CanvasUserGroups {
  if (!payload || typeof payload !== 'object') return emptyCanvasUserGroups()
  const body = payload as UserGroupsResponse
  if (
    !body.success ||
    !body.data ||
    typeof body.data !== 'object' ||
    Array.isArray(body.data)
  ) {
    return emptyCanvasUserGroups()
  }

  return {
    groups: createPlaygroundGroupOptions(body.data),
    defaultGroup: readOptionalText(body.canvas_default_group),
    textGroup: readOptionalText(body.canvas_text_default_group),
    imageGroup: readOptionalText(body.canvas_image_default_group),
    audioGroup: readOptionalText(body.canvas_audio_default_group),
    videoGroup: readOptionalText(body.canvas_video_default_group),
  }
}
