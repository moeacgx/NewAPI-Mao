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

import { convertDetectedLanguage, toIntlLocale } from '../languages'

describe('convertDetectedLanguage', () => {
  test.each([
    ['zhCN', 'zhCN'],
    ['zhTW', 'zhTW'],
  ])(
    'persisted interface code %s is kept as %s instead of being rewritten',
    (stored, expected) => {
      expect(convertDetectedLanguage(stored)).toBe(expected)
    }
  )

  test.each([
    ['zh-CN', 'zhCN'],
    ['zh', 'zhCN'],
    ['zh-Hans', 'zhCN'],
    ['zh-TW', 'zhTW'],
    ['zh_HK', 'zhTW'],
    ['zh-Hant-TW', 'zhTW'],
  ])('browser tag %s maps to interface code %s', (detected, expected) => {
    expect(convertDetectedLanguage(detected)).toBe(expected)
  })

  test('non-Chinese detected tags pass through unchanged', () => {
    expect(convertDetectedLanguage('fr-FR')).toBe('fr-FR')
  })
})

describe('toIntlLocale', () => {
  test.each([
    ['zhCN', 'zh-CN'],
    ['zhTW', 'zh-TW'],
    ['zh-tw', 'zh-TW'],
    ['en', 'en'],
  ])(
    'interface or BCP-47 code %s becomes Intl locale %s',
    (value, expected) => {
      expect(toIntlLocale(value)).toBe(expected)
    }
  )

  test.each(['en_US!', 'not a tag', '', null, undefined])(
    'malformed or empty value %j falls back to the runtime default (undefined)',
    (value) => {
      expect(toIntlLocale(value)).toBeUndefined()
    }
  )
})
