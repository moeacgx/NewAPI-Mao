/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import type { NotificationTaskFilterConfig } from './types'

export const CHANNEL_DISABLED_EVENT = 'channel_disabled'

export const INSUFFICIENT_BALANCE_KEYWORDS: string[] = [
  '预扣费额度失败',
  '余额不足',
]

export const INSUFFICIENT_BALANCE_PREFIX_DEDUP_SECONDS = 300

export function parsePrefixDedupSeconds(value: unknown): number {
  const seconds = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(seconds) || seconds <= 0) return 0
  return Math.trunc(seconds)
}

export function normalizeTaskFilterConfig(
  config: NotificationTaskFilterConfig | undefined
): NotificationTaskFilterConfig | undefined {
  if (!config) return undefined
  const statusCodes = config.status_codes?.trim() || ''
  const errorKeywords = (config.error_keywords ?? [])
    .map((keyword) => keyword.trim())
    .filter(Boolean)
  const prefixDedupSeconds = parsePrefixDedupSeconds(
    config.prefix_dedup_seconds
  )
  if (!statusCodes && errorKeywords.length === 0 && prefixDedupSeconds <= 0) {
    return undefined
  }
  return {
    ...(statusCodes ? { status_codes: statusCodes } : {}),
    ...(errorKeywords.length > 0 ? { error_keywords: errorKeywords } : {}),
    ...(prefixDedupSeconds > 0
      ? { prefix_dedup_seconds: prefixDedupSeconds }
      : {}),
  }
}

export function applyInsufficientBalanceDedupPreset(
  config: NotificationTaskFilterConfig | undefined
): NotificationTaskFilterConfig {
  const keywords: string[] = [...INSUFFICIENT_BALANCE_KEYWORDS]
  const existing = (config?.error_keywords ?? [])
    .map((keyword) => keyword.trim())
    .filter(Boolean)
  const seen = new Set(keywords.map((keyword) => keyword.toLowerCase()))
  for (const keyword of existing) {
    const identity = keyword.toLowerCase()
    if (seen.has(identity)) continue
    seen.add(identity)
    keywords.push(keyword)
  }
  return {
    ...(config?.status_codes?.trim()
      ? { status_codes: config.status_codes.trim() }
      : {}),
    error_keywords: keywords,
    prefix_dedup_seconds: INSUFFICIENT_BALANCE_PREFIX_DEDUP_SECONDS,
  }
}
