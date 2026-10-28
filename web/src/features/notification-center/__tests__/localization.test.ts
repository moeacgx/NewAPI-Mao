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

const localeResources = import.meta.glob<{
  translation: Record<string, string>
}>('../../../i18n/locales/*.json', { eager: true, import: 'default' })

test('所有语言包含工单正文摘要说明、示例和投递边界提示', () => {
  const keys = [
    'Current user message excerpt',
    'I need help with this request.',
    'Ticket notifications send an excerpt of the current user message to the recipients configured by Root. The escaped excerpt is limited to 2000 characters; longer messages include a truncation notice. View the full message in ticket details.',
  ]
  for (const [locale, resource] of Object.entries(localeResources)) {
    for (const key of keys) {
      expect(resource.translation[key], `${locale}: ${key}`).toBeTruthy()
      if (!locale.endsWith('/en.json')) {
        expect(resource.translation[key], `${locale}: ${key}`).not.toBe(key)
      }
    }
  }
})
