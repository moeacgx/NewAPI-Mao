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
import { expect, test } from 'vitest'

const sources = import.meta.glob<string>(
  ['../**/*.{ts,tsx}', '!../__tests__/**'],
  {
    eager: true,
    query: '?raw',
    import: 'default',
  }
)
const locales = import.meta.glob<{ translation: Record<string, string> }>(
  '../../../i18n/locales/*.json',
  { eager: true, import: 'default' }
)

test('MCP 全部界面文案在七种语言中存在，插值变量保持一致', () => {
  const keys = new Set(['MCP Management'])
  for (const source of Object.values(sources)) {
    for (const match of source.matchAll(/\bt\(\s*'([^']+)'/g)) {
      keys.add(match[1])
    }
  }
  expect(Object.keys(locales)).toHaveLength(7)
  expect(keys.has('Close key receipt')).toBe(true)
  for (const [locale, resource] of Object.entries(locales)) {
    for (const key of keys) {
      const value = resource.translation[key]
      expect(value, `${locale}: ${key}`).toBeTruthy()
      expect(value, `${locale}: ${key}`).not.toMatch(/\?\?|\uFFFD/)
      expect(
        (value.match(/\{\{[^}]+\}\}/g) ?? []).sort(),
        `${locale}: ${key}`
      ).toEqual((key.match(/\{\{[^}]+\}\}/g) ?? []).sort())
    }
  }
})
