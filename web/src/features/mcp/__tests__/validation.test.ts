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

import { mcpKeySchema } from '../schema'

test('名称按 Unicode 字符限制为 100 字，保留合法中文及表情并去除首尾空格', () => {
  const input = {
    name: `  ${'猫🔑'.repeat(50)}  `,
    tools: ['newapi_context'],
    expires: '2099-01-01T12:00',
  }
  expect(mcpKeySchema.parse(input).name).toBe('猫🔑'.repeat(50))
  const tooLong = mcpKeySchema.safeParse({
    ...input,
    name: `${'猫🔑'.repeat(50)}a`,
  })
  expect(tooLong.success).toBe(false)
  if (!tooLong.success) {
    expect(tooLong.error.issues).toEqual([
      expect.objectContaining({
        path: ['name'],
        message: 'Use no more than 100 characters',
      }),
    ])
  }
})

test.each(['', 'not-a-date', '2020-01-01T00:00'])(
  '有效期为 %s 时阻止无效或已到期的密钥创建',
  (expires) => {
    expect(
      mcpKeySchema.safeParse({
        name: 'Client',
        tools: ['newapi_context'],
        expires,
      }).success
    ).toBe(false)
  }
)

test('有效期等于后端允许的最后一秒时可创建', () => {
  expect(
    mcpKeySchema.safeParse({
      name: 'Client',
      tools: ['newapi_context'],
      expires: '9999-12-31T23:59:59Z',
    }).success
  ).toBe(true)
})

test.each(['+010000-01-01T00:00:00Z', '+010001-01-01T00:00:00Z'])(
  '有效期为 %s 时拒绝超过后端上限的密钥创建',
  (expires) => {
    const result = mcpKeySchema.safeParse({
      name: 'Client',
      tools: ['newapi_context'],
      expires,
    })
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues).toEqual([
        expect.objectContaining({
          path: ['expires'],
          message: 'Choose a future expiration time',
        }),
      ])
    }
  }
)
