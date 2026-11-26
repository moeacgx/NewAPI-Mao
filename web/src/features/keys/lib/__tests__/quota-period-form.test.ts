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
import { describe, expect, test } from 'vitest'

import type { ApiKey } from '../../types'
import {
  getApiKeyFormDefaultValues,
  getApiKeyFormSchema,
  transformApiKeyToFormDefaults,
  transformFormDataToPayload,
} from '../api-key-form'

const t = ((key: string, options?: Record<string, unknown>) =>
  options?.max !== undefined
    ? key.replace('{{max}}', String(options.max))
    : key) as TFunction

// Default display is USD with 500,000 quota units per USD.
const USD = 500_000

const baseApiKey: ApiKey = {
  id: 7,
  name: 'period-token',
  key: 'sk-test',
  status: 1,
  remain_quota: 0,
  used_quota: 0,
  unlimited_quota: true,
  quota_period: 'none',
  quota_period_limit: 0,
  quota_period_used: 0,
  quota_period_reset_at: 0,
  expired_time: -1,
  created_time: 1,
  accessed_time: 0,
  group: 'default',
  auto_groups: null,
  group_mode: 'explicit',
  group_ids: [],
  group_details: [],
  cross_group_retry: false,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
}

function formValues(overrides: Record<string, unknown>) {
  return {
    ...getApiKeyFormDefaultValues(false),
    name: 'period-token',
    ...overrides,
  } as ReturnType<typeof getApiKeyFormDefaultValues>
}

describe('token quota period form contract', () => {
  test('每周 500 USD 与总额度 10000 USD 提交内部额度单位', () => {
    const payload = transformFormDataToPayload(
      formValues({
        unlimited_quota: false,
        remain_quota_dollars: 10000,
        quota_period: 'weekly',
        quota_period_limit_dollars: 500,
      })
    )
    expect(payload.quota_period).toBe('weekly')
    expect(payload.quota_period_limit).toBe(500 * USD)
    expect(payload.remain_quota).toBe(10000 * USD)
    expect(payload.unlimited_quota).toBe(false)
  })

  test('总额度不限时每日与每周周期额度仍然提交', () => {
    for (const period of ['daily', 'weekly'] as const) {
      const payload = transformFormDataToPayload(
        formValues({
          unlimited_quota: true,
          quota_period: period,
          quota_period_limit_dollars: 500,
        })
      )
      expect(payload.unlimited_quota).toBe(true)
      expect(payload.remain_quota).toBe(0)
      expect(payload.quota_period).toBe(period)
      expect(payload.quota_period_limit).toBe(500 * USD)
    }
  })

  test('不设周期时不提交周期额度', () => {
    const payload = transformFormDataToPayload(
      formValues({ quota_period: 'none', quota_period_limit_dollars: 500 })
    )
    expect(payload.quota_period).toBe('none')
    expect(payload.quota_period_limit).toBe(0)
  })

  test('重新打开编辑回填周期、周期额度与总额度', () => {
    for (const period of ['daily', 'weekly'] as const) {
      const defaults = transformApiKeyToFormDefaults({
        ...baseApiKey,
        unlimited_quota: false,
        remain_quota: 10000 * USD,
        quota_period: period,
        quota_period_limit: 500 * USD,
      })
      expect(defaults.quota_period).toBe(period)
      expect(defaults.quota_period_limit_dollars).toBe(500)
      expect(defaults.remain_quota_dollars).toBe(10000)
      expect(defaults.unlimited_quota).toBe(false)
    }

    const unlimited = transformApiKeyToFormDefaults({
      ...baseApiKey,
      quota_period: 'daily',
      quota_period_limit: 500 * USD,
    })
    expect(unlimited.unlimited_quota).toBe(true)
    expect(unlimited.quota_period).toBe('daily')
    expect(unlimited.quota_period_limit_dollars).toBe(500)
  })

  test('旧令牌缺少周期字段时按不设周期回填', () => {
    const legacy = { ...baseApiKey } as Record<string, unknown>
    delete legacy.quota_period
    delete legacy.quota_period_limit
    const defaults = transformApiKeyToFormDefaults(legacy as unknown as ApiKey)
    expect(defaults.quota_period).toBe('none')
    expect(defaults.quota_period_limit_dollars).toBe(0)
  })
})

describe('token quota form validation', () => {
  const schema = getApiKeyFormSchema(t)
  const parse = (overrides: Record<string, unknown>) =>
    schema.safeParse(formValues(overrides))

  test('10000 USD 总额度与每周 500 USD 通过校验', () => {
    expect(
      parse({
        unlimited_quota: false,
        remain_quota_dollars: 10000,
        quota_period: 'weekly',
        quota_period_limit_dollars: 500,
      }).success
    ).toBe(true)
  })

  test('选择周期但周期额度为 0 时拒绝', () => {
    const result = parse({
      quota_period: 'daily',
      quota_period_limit_dollars: 0,
    })
    expect(result.success).toBe(false)
    expect(
      result.error?.issues.some(
        (issue) => issue.path[0] === 'quota_period_limit_dollars'
      )
    ).toBe(true)
  })

  test('总额度不限时仍校验周期额度', () => {
    expect(
      parse({
        unlimited_quota: true,
        quota_period: 'weekly',
        quota_period_limit_dollars: 0,
      }).success
    ).toBe(false)
  })

  test('总额度和周期额度超过显示单位上限时给出可理解提示', () => {
    const total = parse({
      unlimited_quota: false,
      remain_quota_dollars: 1_000_000_001,
    })
    expect(total.success).toBe(false)
    expect(total.error?.issues[0]?.message).toBe(
      'Quota cannot exceed 1,000,000,000'
    )
    const period = parse({
      quota_period: 'daily',
      quota_period_limit_dollars: 1_000_000_001,
    })
    expect(period.success).toBe(false)
    expect(
      parse({
        unlimited_quota: false,
        remain_quota_dollars: 1_000_000_000,
      }).success
    ).toBe(true)
  })

  test('负数总额度被拒绝', () => {
    expect(
      parse({ unlimited_quota: false, remain_quota_dollars: -1 }).success
    ).toBe(false)
  })
})
