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
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

// `JSON.parse` silently keeps only the LAST of two duplicate keys in an
// object literal — exactly how the zh-TW claim-eligibility strings once got
// silently overwritten by an unrelated pre-existing entry further down the
// file. A test built on JSON.parse can never see that class of bug, so this
// scans the raw file text instead.
//
// Deliberately scoped to the keys this benefit-claim locale fix touches
// (not a full-file audit): a repo-wide duplicate-key sweep would pull in
// unrelated pre-existing locale debt outside this change's scope.
const TOUCHED_KEYS = [
  'Not eligible to claim',
  'Eligible to claim',
  'Benefit activity is not active',
  'Benefit activity has not started',
  'Benefit activity has ended',
  'Benefit fully claimed',
  'Claim requirement',
  'Historical top-up of at least {{amount}}',
  'No historical top-up requirement',
  'New accounts must wait before they can claim',
]

function countRawKeyOccurrences(rawJsonText: string, key: string): number {
  const escaped = key.replaceAll(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const pattern = new RegExp(`"${escaped}"\\s*:`, 'g')
  return (rawJsonText.match(pattern) ?? []).length
}

const __dirname = path.dirname(fileURLToPath(import.meta.url))

const DEFAULT_LOCALES_DIR = path.resolve(__dirname, '../../../../i18n/locales')
const CLASSIC_LOCALES_DIR = path.resolve(
  __dirname,
  '../../../../../classic/src/i18n/locales'
)

// Classic ships a `zh.json`, but `web/classic/src/i18n/i18n.js` never
// imports it — only `zh-CN.json`/`zh-TW.json` are wired into the running
// app, so an unused legacy file is intentionally excluded here.
const CLASSIC_ACTIVE_LOCALE_FILES = [
  'en.json',
  'zh-CN.json',
  'zh-TW.json',
  'fr.json',
  'ru.json',
  'ja.json',
  'vi.json',
]

describe('benefit claim locale keys are not duplicated in raw JSON text', () => {
  const defaultLocaleFiles = fs
    .readdirSync(DEFAULT_LOCALES_DIR)
    .filter((f) => f.endsWith('.json'))

  it.each(defaultLocaleFiles)(
    'Default %s has each touched key at most once',
    (file) => {
      const raw = fs.readFileSync(path.join(DEFAULT_LOCALES_DIR, file), 'utf8')
      for (const key of TOUCHED_KEYS) {
        expect(
          countRawKeyOccurrences(raw, key),
          `"${key}" in web/src/i18n/locales/${file}`
        ).toBeLessThanOrEqual(1)
      }
    }
  )

  it.each(CLASSIC_ACTIVE_LOCALE_FILES)(
    'Classic %s has each touched key at most once',
    (file) => {
      const raw = fs.readFileSync(path.join(CLASSIC_LOCALES_DIR, file), 'utf8')
      for (const key of TOUCHED_KEYS) {
        expect(
          countRawKeyOccurrences(raw, key),
          `"${key}" in web/classic/src/i18n/locales/${file}`
        ).toBeLessThanOrEqual(1)
      }
    }
  )

  it('Default zh-TW actually carries the claim-specific translation (not an echoed English key)', () => {
    const raw = fs.readFileSync(
      path.join(DEFAULT_LOCALES_DIR, 'zh-TW.json'),
      'utf8'
    )
    const parsed = JSON.parse(raw) as { translation: Record<string, string> }
    expect(parsed.translation['Not eligible to claim']).toBe('不符合領取條件')
  })
})
