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
import type { GroupOption } from '@/features/playground/types'
import { normalizeCanvasOrigin } from '@/lib/canvas-settings'

type CanvasLaunchUrlOptions = {
  canvasOrigin: string
  newApiOrigin: string
  group: string
  textGroup?: string
  imageGroup?: string
  audioGroup?: string
  videoGroup?: string
}

export function prioritizeCanvasAutoGroup(
  groups: GroupOption[]
): GroupOption[] {
  const autoGroup = groups.find((group) => group.value === 'auto')
  if (!autoGroup || groups[0] === autoGroup) {
    return groups
  }

  return [autoGroup, ...groups.filter((group) => group.value !== 'auto')]
}

export function buildCanvasLaunchUrl(options: CanvasLaunchUrlOptions): string {
  const canvasUrl = new URL('/', normalizeCanvasOrigin(options.canvasOrigin))
  const newApiOrigin = options.newApiOrigin.trim().replace(/\/+$/, '')

  canvasUrl.searchParams.set('mode', 'newapi')
  canvasUrl.searchParams.set('baseUrl', `${newApiOrigin}/canvas`)
  canvasUrl.searchParams.set('group', options.group)
  const modalityGroups = [
    ['textGroup', options.textGroup],
    ['imageGroup', options.imageGroup],
    ['audioGroup', options.audioGroup],
    ['videoGroup', options.videoGroup],
  ] as const
  for (const [key, value] of modalityGroups) {
    const trimmed = value?.trim()
    if (trimmed) canvasUrl.searchParams.set(key, trimmed)
  }

  return canvasUrl.toString()
}

export function resolveCanvasDefaultGroup(
  groups: GroupOption[],
  configuredGroup: string
): string {
  const configured = configuredGroup.trim()
  if (configured && groups.some((group) => group.value === configured)) {
    return configured
  }

  return (
    groups.find((group) => group.value === 'default')?.value ??
    groups[0]?.value ??
    ''
  )
}

// 四类分组留空表示跟随当前默认分组，不回退到列表第一项。
export function resolveCanvasCapabilityGroup(
  groups: GroupOption[],
  configuredGroup: string
): string {
  const configured = configuredGroup.trim()
  if (configured && groups.some((group) => group.value === configured)) {
    return configured
  }
  return ''
}
