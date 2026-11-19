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
import type { AutoGroupConfig, GroupDetail } from '../types'

export type CanvasDefaultGroupOption = {
  value: string
  label: string
}

export function buildCanvasDefaultGroupOptions(
  groups: GroupDetail[] = [],
  autoGroup?: AutoGroupConfig
): CanvasDefaultGroupOption[] {
  const activeGroups = groups
    .filter((group) => group.status === 1)
    .map((group) => ({
      value: group.code,
      label: group.name || group.code,
    }))

  if (!autoGroup?.user_selectable) {
    return activeGroups
  }

  return [{ value: 'auto', label: 'Automatic selection' }, ...activeGroups]
}

// 当前已保存值若是停用分组，不属于可选项；这里只为 Select 的 items 补充当前名称，避免显示原始 code。
export function buildCanvasDefaultGroupDisplayItems(
  options: CanvasDefaultGroupOption[],
  groups: GroupDetail[] = [],
  currentValue = ''
): CanvasDefaultGroupOption[] {
  if (
    !currentValue ||
    options.some((option) => option.value === currentValue)
  ) {
    return options
  }
  const current = groups.find((group) => group.code === currentValue)
  const name = typeof current?.name === 'string' ? current.name.trim() : ''
  if (!name) {
    return options
  }
  return [...options, { value: currentValue, label: name }]
}
