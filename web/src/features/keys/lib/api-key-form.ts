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
import { z } from 'zod'

import { getCurrencyDisplay } from '@/lib/currency'
import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'

import { DEFAULT_GROUP } from '../constants'
import {
  API_KEY_QUOTA_PERIODS,
  type ApiKey,
  type ApiKeyFormData,
} from '../types'
import {
  buildGroupSelectionPayload,
  resolveApiKeyGroups,
  type GroupSelectionOption,
} from './group-selection'
import {
  loadRatioProtection,
  isValidRatioLimit,
  serializeRatioProtection,
} from './ratio-protection'

// ============================================================================
// Form Schema
// ============================================================================

export function getApiKeyFormSchema(t: TFunction, maxAutoGroups = 0) {
  const autoGroupLimit =
    Number.isInteger(maxAutoGroups) && maxAutoGroups > 0 ? maxAutoGroups : 0

  return z
    .object({
      name: z.string().min(1, t('Please enter a name')),
      remain_quota_dollars: z.number().optional(),
      expired_time: z.date().optional(),
      unlimited_quota: z.boolean(),
      quota_period: z.enum(API_KEY_QUOTA_PERIODS),
      quota_period_limit_dollars: z
        .number()
        .min(0, t('Quota must be zero or greater')),
      model_limits: z.array(z.string()),
      allow_ips: z.string().optional(),
      groups: z.array(z.string()),
      ratioProtection: z
        .object({
          values: z.record(z.string(), z.string()),
          invalid: z.boolean(),
        })
        .optional(),
      group_channel_ids: z
        .record(
          z.string(),
          z
            .number()
            .int(t('Channel ID must be a positive integer.'))
            .min(1, t('Channel ID must be a positive integer.'))
            .max(2147483647, t('Channel ID must be a positive integer.'))
        )
        .optional(),
      auto_groups_mode: z.enum(['inherit', 'custom']),
      auto_groups: z.array(z.string()),
      cross_group_retry: z.boolean().optional(),
      tokenCount: z.number().min(1).optional(),
    })
    .superRefine((data, ctx) => {
      if (
        data.ratioProtection?.invalid ||
        Object.values(data.ratioProtection?.values || {}).some(
          (value) => !isValidRatioLimit(value)
        )
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['ratioProtection'],
          message: t('Check ratio protection before saving.'),
        })
      }
      if (data.groups.length === 1 && data.groups[0] === 'auto') {
        if (
          data.auto_groups_mode === 'custom' &&
          data.auto_groups.length === 0
        ) {
          ctx.addIssue({
            code: 'custom',
            path: ['auto_groups'],
            message: t(
              'Select at least one Auto group or restore global Auto.'
            ),
          })
        }

        if (autoGroupLimit > 0 && data.auto_groups.length > autoGroupLimit) {
          ctx.addIssue({
            code: 'custom',
            path: ['auto_groups'],
            message: t('Select at most {{max}} Auto groups', {
              max: autoGroupLimit,
            }),
          })
        }

        if (new Set(data.auto_groups).size !== data.auto_groups.length) {
          ctx.addIssue({
            code: 'custom',
            path: ['auto_groups'],
            message: t('Auto groups must not contain duplicates'),
          })
        }
      }

      const isAutoOnly = data.groups.length === 1 && data.groups[0] === 'auto'
      if (
        !isAutoOnly &&
        autoGroupLimit > 0 &&
        data.groups.length > autoGroupLimit
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['groups'],
          message: t('A token can select at most {{max}} groups', {
            max: autoGroupLimit,
          }),
        })
      }

      // The backend validates the same bound (1e9 currency units at the configured
      // quota per unit); the form shows it in the displayed currency unit.
      const maxQuotaUnits = 1e9 * getCurrencyDisplay().config.quotaPerUnit
      const maxQuotaMessage = t('Quota cannot exceed {{max}}', {
        max: quotaUnitsToDollars(maxQuotaUnits).toLocaleString('en-US', {
          maximumFractionDigits: 2,
        }),
      })

      if (data.quota_period !== 'none') {
        if (!(data.quota_period_limit_dollars > 0)) {
          ctx.addIssue({
            code: 'custom',
            path: ['quota_period_limit_dollars'],
            message: t('Period quota must be greater than 0'),
          })
        } else if (
          parseQuotaFromDollars(data.quota_period_limit_dollars) > maxQuotaUnits
        ) {
          ctx.addIssue({
            code: 'custom',
            path: ['quota_period_limit_dollars'],
            message: maxQuotaMessage,
          })
        }
      }

      if (data.unlimited_quota) {
        return
      }

      if (
        data.remain_quota_dollars === undefined ||
        data.remain_quota_dollars < 0
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['remain_quota_dollars'],
          message: t('Quota must be zero or greater'),
        })
      } else if (
        parseQuotaFromDollars(data.remain_quota_dollars) > maxQuotaUnits
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['remain_quota_dollars'],
          message: maxQuotaMessage,
        })
      }
    })
}

export type ApiKeyFormValues = z.infer<ReturnType<typeof getApiKeyFormSchema>>

// ============================================================================
// Form Defaults
// ============================================================================

export const API_KEY_FORM_DEFAULT_VALUES: ApiKeyFormValues = {
  name: '',
  remain_quota_dollars: 10,
  expired_time: undefined,
  unlimited_quota: true,
  quota_period: 'none',
  quota_period_limit_dollars: 0,
  model_limits: [],
  allow_ips: '',
  groups: DEFAULT_GROUP ? [DEFAULT_GROUP] : [],
  ratioProtection: { values: {}, invalid: false },
  group_channel_ids: {},
  auto_groups_mode: 'inherit',
  auto_groups: [],
  cross_group_retry: true,
  tokenCount: 1,
}

export function getApiKeyFormDefaultValues(
  defaultUseAutoGroup: boolean
): ApiKeyFormValues {
  return {
    ...API_KEY_FORM_DEFAULT_VALUES,
    groups: defaultUseAutoGroup ? ['auto'] : API_KEY_FORM_DEFAULT_VALUES.groups,
    auto_groups_mode: 'inherit',
    auto_groups: [],
    cross_group_retry: defaultUseAutoGroup,
  }
}

// ============================================================================
// Form Data Transformation
// ============================================================================

/**
 * Transform form data to API payload
 */
export function transformFormDataToPayload(
  data: ApiKeyFormValues,
  groupOptions: readonly GroupSelectionOption[] = []
): ApiKeyFormData {
  const groupSelection = buildGroupSelectionPayload(data.groups, groupOptions)
  const isAuto = groupSelection.group_mode === 'auto'
  return {
    name: data.name,
    remain_quota: data.unlimited_quota
      ? 0
      : parseQuotaFromDollars(data.remain_quota_dollars || 0),
    expired_time: data.expired_time
      ? Math.floor(data.expired_time.getTime() / 1000)
      : -1,
    unlimited_quota: data.unlimited_quota,
    quota_period: data.quota_period,
    quota_period_limit:
      data.quota_period === 'none'
        ? 0
        : parseQuotaFromDollars(data.quota_period_limit_dollars || 0),
    model_limits_enabled: data.model_limits.length > 0,
    model_limits: data.model_limits.join(','),
    allow_ips: data.allow_ips || '',
    group: groupSelection.group,
    group_ids: groupSelection.group_ids,
    group_channel_ids: Object.fromEntries(
      Object.entries(data.group_channel_ids ?? {}).filter(([id]) =>
        groupSelection.group_ids.includes(Number(id))
      )
    ),
    group_mode: groupSelection.group_mode,
    group_ratio_limits: serializeRatioProtection(data.ratioProtection),
    auto_groups:
      isAuto && data.auto_groups_mode === 'custom' ? data.auto_groups : [],
    cross_group_retry:
      data.groups.length > 1 || (isAuto && !!data.cross_group_retry),
  }
}

/**
 * Transform API key data to form defaults
 */
export function transformApiKeyToFormDefaults(
  apiKey: ApiKey,
  availableAutoGroups: string[] = [],
  maxAutoGroups = 0
): ApiKeyFormValues {
  const availableSet = new Set(availableAutoGroups)
  const storedAutoGroups = apiKey.auto_groups ?? []
  const filteredAutoGroups = storedAutoGroups.filter((group) =>
    availableSet.has(group)
  )
  const autoGroups =
    maxAutoGroups > 0
      ? filteredAutoGroups.slice(0, maxAutoGroups)
      : filteredAutoGroups
  const autoGroupsMode = storedAutoGroups.length > 0 ? 'custom' : 'inherit'

  return {
    name: apiKey.name,
    remain_quota_dollars: apiKey.unlimited_quota
      ? 0
      : quotaUnitsToDollars(apiKey.remain_quota),
    expired_time:
      apiKey.expired_time > 0
        ? new Date(apiKey.expired_time * 1000)
        : undefined,
    unlimited_quota: apiKey.unlimited_quota,
    quota_period: apiKey.quota_period ?? 'none',
    quota_period_limit_dollars: quotaUnitsToDollars(
      apiKey.quota_period_limit ?? 0
    ),
    model_limits: apiKey.model_limits
      ? apiKey.model_limits.split(',').filter(Boolean)
      : [],
    allow_ips: apiKey.allow_ips || '',
    groups: resolveApiKeyGroups(apiKey),
    ratioProtection: loadRatioProtection(apiKey),
    group_channel_ids: apiKey.group_channel_ids ?? {},
    auto_groups_mode: autoGroupsMode,
    auto_groups: autoGroups,
    cross_group_retry: !!apiKey.cross_group_retry,
    tokenCount: 1,
  }
}
