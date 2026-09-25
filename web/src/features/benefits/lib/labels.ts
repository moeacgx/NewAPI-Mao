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

import { getCurrencyDisplay } from '@/lib/currency'
import type { CurrencyDisplayType } from '@/stores/system-config-store'

import type { BenefitActivityStatus, BenefitVoucherStatus } from '../types'

/** Voucher status label. Explicit t() calls keep every value scannable for i18n sync. */
export function voucherStatusLabel(
  status: BenefitVoucherStatus,
  t: TFunction
): string {
  switch (status) {
    case 'active':
      return t('Active')
    case 'exhausted':
      return t('Exhausted')
    case 'expired':
      return t('Expired')
    case 'voided':
      return t('Voided')
    default:
      return status
  }
}

/** Activity status label. Explicit t() calls keep every value scannable for i18n sync. */
export function activityStatusLabel(
  status: BenefitActivityStatus,
  t: TFunction
): string {
  switch (status) {
    case 'draft':
      return t('Draft')
    case 'published':
      return t('Published')
    case 'paused':
      return t('Paused')
    case 'ended':
      return t('Ended')
    case 'terminated':
      return t('Terminated')
    default:
      return status
  }
}

/**
 * Claim eligibility reason label. Mirrors the backend's
 * `BenefitClaimReason*` constants (ineligible/claimed/sold_out/inactive/
 * not_started/ended); explicit t() calls keep every value scannable.
 *
 * Keys are namespaced ("... to claim" / "Benefit ...") rather than bare
 * words like "Eligible" or "Fully claimed": this flat i18n keys-are-strings
 * setup lets an unrelated feature reusing the same bare English phrase
 * silently win a duplicate JSON key and overwrite this translation (this
 * happened once already in zh-TW.json).
 */
export function claimEligibilityLabel(
  reason: string | undefined,
  t: TFunction
): string {
  switch (reason) {
    case 'ineligible':
      return t('Not eligible to claim')
    case 'claimed':
      return t('Already claimed')
    case 'sold_out':
      return t('Benefit fully claimed')
    case 'inactive':
      return t('Benefit activity is not active')
    case 'not_started':
      return t('Benefit activity has not started')
    case 'ended':
      return t('Benefit activity has ended')
    default:
      return t('Not eligible to claim')
  }
}

/** Bare currency symbol for the display type an activity payload arrived with. */
function benefitAmountSymbol(displayType: CurrencyDisplayType): string {
  switch (displayType) {
    case 'CNY':
      return '¥'
    case 'CUSTOM':
      // No backend response — activity or otherwise — ever returns a custom
      // symbol string, only the type name; fall back to the current global
      // custom symbol label (best effort, not a per-activity value).
      return getCurrencyDisplay().config.customCurrencySymbol
    case 'USD':
    default:
      return '$'
  }
}

/**
 * The backend converts this amount using its OWN current display setting at
 * response time (`controller.benefitCurrentDisplayValues` /
 * `model.CurrentBenefitAmountDisplayContext`, not a per-activity snapshot —
 * despite `amount_display_type_snapshot` existing as a column, the live read
 * path never consults it) and returns the resulting type alongside it as
 * `activity.amount_display_type`. Format using THAT type, not a value read
 * from this client's own (separately fetched, possibly stale-by-a-request)
 * global display config store: the two usually agree, but only the type
 * that travelled with this exact amount is guaranteed consistent with it.
 * Never re-convert through a quota/exchange-rate path either — the backend
 * has already produced the final display-unit number.
 */
export function formatBenefitDisplayAmount(
  amount: number,
  displayType: CurrencyDisplayType,
  t: TFunction
): string {
  const numericAmount = Number(amount)
  if (!Number.isFinite(numericAmount)) return '-'
  if (displayType === 'TOKENS') {
    return `${Math.round(numericAmount).toLocaleString()} ${t('Tokens')}`
  }
  return `${benefitAmountSymbol(displayType)}${numericAmount.toFixed(2)}`
}

export function ledgerEntryTypeLabel(type: string, t: TFunction): string {
  switch (type) {
    case 'pre_consume':
      return t('Pre-consume')
    case 'settle_delta':
      return t('Settlement adjustment')
    case 'settle_rollback':
      return t('Settlement rollback')
    case 'refund_additional':
      return t('Additional refund')
    case 'refund':
      return t('Refund')
    case 'void':
      return t('Voided')
    case 'expire':
      return t('Expired')
    default:
      return type
  }
}

/**
 * Activity batch-delete skip-reason label. Covers the backend's real
 * skip codes (not_found / has_claim_data / active_voucher / not_deletable);
 * an unrecognized code still renders a readable sentence instead of a raw
 * code, for any future code this list hasn't caught up with yet.
 */
export function activityDeleteSkipReasonLabel(
  reason: string,
  t: TFunction
): string {
  switch (reason) {
    case 'not_found':
      return t('Activity not found')
    case 'has_claim_data':
      return t('Draft activity already has claim data')
    case 'active_voucher':
      return t('Activity still has active vouchers')
    case 'not_deletable':
      return t('Activity is still active or not eligible for deletion')
    default:
      return t('Unknown reason')
  }
}
