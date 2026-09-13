/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { describe, expect, test } from 'vitest'

import {
  applyInsufficientBalanceDedupPreset,
  INSUFFICIENT_BALANCE_KEYWORDS,
  INSUFFICIENT_BALANCE_PREFIX_DEDUP_SECONDS,
  normalizeTaskFilterConfig,
} from '../filter-config'

describe('notification task filter config', () => {
  test('keeps prefix dedup seconds with keywords', () => {
    expect(
      normalizeTaskFilterConfig({
        status_codes: ' 403 ',
        error_keywords: ['预扣费额度失败', ''],
        prefix_dedup_seconds: 300,
      })
    ).toEqual({
      status_codes: '403',
      error_keywords: ['预扣费额度失败'],
      prefix_dedup_seconds: 300,
    })
  })

  test('allows prefix-only dedup config', () => {
    expect(
      normalizeTaskFilterConfig({
        prefix_dedup_seconds: 120,
      })
    ).toEqual({ prefix_dedup_seconds: 120 })
  })

  test('drops empty filter config', () => {
    expect(
      normalizeTaskFilterConfig({
        status_codes: ' ',
        error_keywords: [' '],
        prefix_dedup_seconds: 0,
      })
    ).toBeUndefined()
  })

  test('applies insufficient-balance preset without dropping extra keywords', () => {
    expect(
      applyInsufficientBalanceDedupPreset({
        status_codes: '403',
        error_keywords: ['quota', '余额不足'],
      })
    ).toEqual({
      status_codes: '403',
      error_keywords: [...INSUFFICIENT_BALANCE_KEYWORDS, 'quota'],
      prefix_dedup_seconds: INSUFFICIENT_BALANCE_PREFIX_DEDUP_SECONDS,
    })
  })
})
