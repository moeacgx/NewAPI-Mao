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

import { groupTagFormSchema } from '../group-tag-form'

const input = {
  name: '平台',
  description: '',
  sort_order: 0,
  icons: '',
  match_keywords: '',
  group_ids: [],
}
describe('group tag editor validation', () => {
  test('accepts empty bindings, preserves explicit zero and parses multiple logos and keywords', () => {
    expect(
      groupTagFormSchema.parse({
        ...input,
        icons: 'OpenAI\nhttps://cdn.example.com/logo.png',
        match_keywords: ' 标准 \n\n专业 ',
      })
    ).toEqual({
      ...input,
      icons: ['OpenAI', 'https://cdn.example.com/logo.png'],
      match_keywords: ['标准', '专业'],
    })
  })
  test.each([
    { name: '' },
    { sort_order: 0.5 },
    { sort_order: 1000001 },
    { icons: 'javascript:alert(1)' },
    { icons: 'https://name@example.com/logo.png' },
    { icons: 'OpenAI\nClaude\nGemini\nDeepSeek\nMoonshot\nQwen\nXAI' },
    { match_keywords: Array(33).fill('标准').join('\n') },
    { group_ids: [0] },
  ])('blocks values outside the existing tag contract: %j', (invalid) => {
    expect(groupTagFormSchema.safeParse({ ...input, ...invalid }).success).toBe(
      false
    )
  })
})
