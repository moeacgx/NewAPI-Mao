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
type MultiKeyConversionEligibility = {
  isEditing: boolean
  isMultiKeyChannel: boolean
  originalType?: number
  type: number
  isIonetLocked: boolean
}

export function canConvertChannelToMultiKey({
  isEditing,
  isMultiKeyChannel,
  originalType,
  type,
  isIonetLocked,
}: MultiKeyConversionEligibility): boolean {
  return Boolean(
    isEditing &&
    !isMultiKeyChannel &&
    !isIonetLocked &&
    originalType !== undefined &&
    Number.isInteger(originalType) &&
    originalType > 0 &&
    Number.isInteger(type) &&
    type > 0 &&
    ![41, 57].includes(originalType) &&
    ![41, 57].includes(type)
  )
}

export function isIonetManagedChannel(otherInfo?: string | null): boolean {
  if (!otherInfo) return false
  try {
    const metadata: unknown = JSON.parse(otherInfo)
    return (
      typeof metadata === 'object' &&
      metadata !== null &&
      'source' in metadata &&
      metadata.source === 'ionet'
    )
  } catch {
    return false
  }
}

type MultiKeyConversionFieldsOptions = {
  isMultiKeyChannel?: boolean
  convertToMultiKey?: boolean
  canWriteSensitive: boolean
  hasNewKey: boolean
  keyMode?: 'append' | 'replace'
  multiKeyMode?: 'random' | 'polling'
}

export function buildMultiKeyConversionFields({
  isMultiKeyChannel = false,
  convertToMultiKey = false,
  canWriteSensitive,
  hasNewKey,
  keyMode = 'append',
  multiKeyMode = 'random',
}: MultiKeyConversionFieldsOptions): {
  convert_to_multi_key?: true
  key_mode?: 'append' | 'replace'
  multi_key_mode?: 'random' | 'polling'
} {
  if (!canWriteSensitive) return {}

  const enabled = isMultiKeyChannel || convertToMultiKey
  if (!enabled) return {}

  return {
    ...(convertToMultiKey ? { convert_to_multi_key: true as const } : {}),
    ...(hasNewKey ? { key_mode: keyMode } : {}),
    multi_key_mode: multiKeyMode,
  }
}
