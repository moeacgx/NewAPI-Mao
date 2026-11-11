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
import { describe, expect, test } from 'vitest'

import {
  buildMultiKeyConversionFields,
  canConvertChannelToMultiKey,
  isIonetManagedChannel,
} from '../multi-key-conversion'

describe('multi-key conversion eligibility', () => {
  test('allows an editable single-key channel with an eligible original and target type', () => {
    expect(
      canConvertChannelToMultiKey({
        isEditing: true,
        isMultiKeyChannel: false,
        originalType: 1,
        type: 1,
        isIonetLocked: false,
      })
    ).toBe(true)
  })

  test.each([
    ['new channel', { isEditing: false }],
    ['existing multi-key channel', { isMultiKeyChannel: true }],
    ['missing original type', { originalType: undefined }],
    ['unknown original type', { originalType: 0 }],
    ['invalid original type', { originalType: Number.NaN }],
    ['unknown target type', { type: 0 }],
    ['Vertex AI original type', { originalType: 41 }],
    ['Codex original type', { originalType: 57 }],
    ['Vertex AI target type', { type: 41 }],
    ['Codex target type', { type: 57 }],
    ['io.net locked channel', { isIonetLocked: true }],
  ] as const)('rejects conversion for %s', (_case, overrides) => {
    expect(
      canConvertChannelToMultiKey({
        isEditing: true,
        isMultiKeyChannel: false,
        originalType: 1,
        type: 1,
        isIonetLocked: false,
        ...overrides,
      })
    ).toBe(false)
  })
})

describe('io.net channel lock detection', () => {
  test('locks channels tagged as io.net managed and ignores unrelated or malformed metadata', () => {
    expect(isIonetManagedChannel('{"source":"ionet"}')).toBe(true)
    expect(isIonetManagedChannel('{"source":"manual"}')).toBe(false)
    expect(isIonetManagedChannel('{invalid')).toBe(false)
  })
})

describe('multi-key conversion request fields', () => {
  test('marks an explicit conversion and includes the selected key and strategy modes', () => {
    expect(
      buildMultiKeyConversionFields({
        convertToMultiKey: true,
        canWriteSensitive: true,
        hasNewKey: true,
        keyMode: 'replace',
        multiKeyMode: 'polling',
      })
    ).toEqual({
      convert_to_multi_key: true,
      key_mode: 'replace',
      multi_key_mode: 'polling',
    })
  })

  test('keeps an existing multi-key strategy editable when no new key is entered', () => {
    expect(
      buildMultiKeyConversionFields({
        isMultiKeyChannel: true,
        canWriteSensitive: true,
        hasNewKey: false,
        multiKeyMode: 'polling',
      })
    ).toEqual({ multi_key_mode: 'polling' })
  })

  test('omits conversion and sensitive fields when conversion is cancelled or permission is absent', () => {
    expect(
      buildMultiKeyConversionFields({
        convertToMultiKey: false,
        canWriteSensitive: true,
        hasNewKey: true,
        keyMode: 'replace',
        multiKeyMode: 'polling',
      })
    ).toEqual({})
    expect(
      buildMultiKeyConversionFields({
        convertToMultiKey: true,
        canWriteSensitive: false,
        hasNewKey: true,
        keyMode: 'replace',
        multiKeyMode: 'polling',
      })
    ).toEqual({})
  })
})
