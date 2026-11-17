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
import type { TFunction } from 'i18next'
import { expect, test } from 'vitest'

import { apiKeySchema } from '../../types'
import {
  getApiKeyFormDefaultValues,
  getApiKeyFormSchema,
  transformApiKeyToFormDefaults,
  transformFormDataToPayload,
} from '../api-key-form'
import {
  isValidRatioLimit,
  loadRatioProtection,
  retainRatioProtection,
  serializeRatioProtection,
} from '../ratio-protection'

const token = apiKeySchema.parse({
  id: 1,
  name: 'protected',
  key: 'masked',
  status: 1,
  remain_quota: 0,
  used_quota: 0,
  unlimited_quota: true,
  expired_time: -1,
  created_time: 1,
  accessed_time: 0,
  model_limits_enabled: false,
  group: 'legacy_alias',
  group_ids: [11],
  group_details: [{ id: 11, code: 'stable_a', name: 'Current Name' }],
})

test('旧 alias 按稳定 ID 归一为 code，编辑无关字段不丢保护', () => {
  const values = transformApiKeyToFormDefaults({
    ...token,
    group_ratio_limits: '{"legacy_alias":1.23456789}',
  })
  expect(values.ratioProtection).toEqual({
    values: { stable_a: '1.23456789' },
    invalid: false,
  })
  expect(
    transformFormDataToPayload({ ...values, name: 'renamed' })
      .group_ratio_limits
  ).toBe('{"stable_a":1.23456789}')
})

test.each(['', ' ', '0', '0.0', '1.25', '1e2', '1.7976931348623157e308'])(
  '有效值 %s',
  (value) => {
    expect(isValidRatioLimit(value)).toBe(true)
  }
)
test.each(['-1', 'NaN', 'Infinity', '1e309', 'abc', '0x10'])(
  '非法输入 %s 阻止保存',
  (value) => {
    expect(isValidRatioLimit(value)).toBe(false)
    const schema = getApiKeyFormSchema(((key: string) => key) as TFunction)
    expect(
      schema.safeParse({
        ...getApiKeyFormDefaultValues(false),
        name: 'test',
        ratioProtection: { values: { stable_a: value }, invalid: false },
      }).success
    ).toBe(false)
  }
)
test.each([
  '{broken',
  '[]',
  'null',
  '{"stable_a":"2"}',
  '{"stable_a":-1}',
  '{"stable_a":1e309}',
])('非法历史 JSON %s 必须显式处理', (raw) => {
  expect(
    loadRatioProtection({ ...token, group_ratio_limits: raw }).invalid
  ).toBe(true)
})

test('空和0清除，显式切组清除移除组且 Auto/继承没有逐组项', () => {
  const draft = { values: { stable_a: '1.5', stable_b: '2' }, invalid: false }
  expect(retainRatioProtection(draft, ['stable_b'])).toEqual({
    values: { stable_b: '2' },
    invalid: false,
  })
  expect(serializeRatioProtection(retainRatioProtection(draft, ['auto']))).toBe(
    ''
  )
  expect(serializeRatioProtection(retainRatioProtection(draft, []))).toBe('')
  expect(
    serializeRatioProtection({ values: { a: '0', b: '  ' }, invalid: false })
  ).toBe('')
})

test('未知历史保护保留，非法数据不因切组静默清空', () => {
  expect(
    loadRatioProtection({ ...token, group_ratio_limits: '{"deleted_group":3}' })
  ).toEqual({ values: { deleted_group: '3' }, invalid: false })
  expect(
    retainRatioProtection({ values: {}, invalid: true }, ['auto']).invalid
  ).toBe(true)
})

test('既有 Auto 与继承的历史保护在无关修改时保留，不新增计费解释', () => {
  for (const mode of ['auto', 'inherit'] as const) {
    const defaults = transformApiKeyToFormDefaults({
      ...token,
      group: mode === 'auto' ? 'auto' : '',
      group_mode: mode,
      group_ratio_limits: '{"stable_a":2}',
    })
    expect(transformFormDataToPayload(defaults).group_ratio_limits).toBe(
      '{"stable_a":2}'
    )
  }
})
