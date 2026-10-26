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

import { describe, expect, test } from 'vitest'

const PENDING_TICKET_KEYS = [
  'Pending tickets',
  'Pending ticket reminder',
  'Dismiss ticket reminder',
  'Tickets awaiting your response: {{total}}',
  'View pending tickets',
  'Pending ticket count',
  'Failed to load pending tickets',
  'No tickets awaiting your response',
  'New tickets and customer follow-ups',
  'Ticket overview',
  'View tickets',
  'Failed to load ticket overview',
]

const localesDir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../../i18n/locales'
)
const localeFiles = fs
  .readdirSync(localesDir)
  .filter((file) => file.endsWith('.json'))

describe('pending ticket locales', () => {
  test.each(localeFiles)(
    '%s translates every pending ticket text and keeps placeholders',
    (file) => {
      const translation = JSON.parse(
        fs.readFileSync(path.join(localesDir, file), 'utf8')
      ).translation as Record<string, string>

      for (const key of PENDING_TICKET_KEYS) {
        const value = translation[key]
        expect(value?.trim(), `${file}: ${key}`).toBeTruthy()
        expect(value).not.toMatch(/\?{2,}|�/)
        for (const placeholder of key.match(/{{\w+}}/g) ?? []) {
          expect(value).toContain(placeholder)
        }
        if (file !== 'en.json') expect(value).not.toBe(key)
      }
    }
  )
})
