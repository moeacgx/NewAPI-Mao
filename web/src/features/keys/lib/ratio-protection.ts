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
import type { ApiKey } from '../types'

export type RatioProtectionDraft = {
  values: Record<string, string>
  invalid: boolean
}

export function loadRatioProtection(
  apiKey: Pick<
    ApiKey,
    'group' | 'group_ids' | 'group_details' | 'group_ratio_limits'
  >
): RatioProtectionDraft {
  const raw = apiKey.group_ratio_limits
  if (!raw?.trim()) return { values: {}, invalid: false }
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { values: {}, invalid: true }
    }
    const codes = (apiKey.group || '').split(',').map((code) => code.trim())
    const aliases = new Map<string, string>()
    if (codes.length === apiKey.group_ids?.length) {
      codes.forEach((code, index) => {
        const detail = apiKey.group_details?.find(
          (item) => item.id === apiKey.group_ids[index]
        )
        if (detail?.code) aliases.set(code, detail.code)
      })
    }
    const values = new Map<string, string>()
    for (const [key, value] of Object.entries(parsed)) {
      const code = aliases.get(key.trim()) || key.trim()
      if (
        !code ||
        typeof value !== 'number' ||
        !Number.isFinite(value) ||
        value < 0 ||
        values.has(code)
      ) {
        return { values: {}, invalid: true }
      }
      // 0 在现有协议中不启用保护，保留草稿可让用户看见原值。
      values.set(code, String(value))
    }
    return { values: Object.fromEntries(values), invalid: false }
  } catch {
    return { values: {}, invalid: true }
  }
}

export function isValidRatioLimit(value: string): boolean {
  if (!value.trim()) return true
  // 仅接受十进制数和科学计数法，不将十六进制或非有限值隐式转换。
  return (
    /^[+]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?$/i.test(value.trim()) &&
    Number.isFinite(Number(value)) &&
    Number(value) >= 0
  )
}

export function serializeRatioProtection(draft?: RatioProtectionDraft): string {
  if (!draft) return ''
  if (
    draft.invalid ||
    Object.values(draft.values).some((value) => !isValidRatioLimit(value))
  ) {
    throw new Error('Invalid ratio protection')
  }
  const entries = Object.entries(draft.values)
    .filter(([, value]) => value.trim() && Number(value) > 0)
    .map(([code, value]) => [code, Number(value)])
  return entries.length ? JSON.stringify(Object.fromEntries(entries)) : ''
}

export function retainRatioProtection(
  draft: RatioProtectionDraft | undefined,
  groups: readonly string[]
): RatioProtectionDraft {
  // 非法历史数据必须显式清除，切组不能无声丢失。
  if (draft?.invalid) return draft
  return {
    values: Object.fromEntries(
      Object.entries(draft?.values || {}).filter(
        ([code]) => code !== 'auto' && groups.includes(code)
      )
    ),
    invalid: false,
  }
}
