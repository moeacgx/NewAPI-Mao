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
import * as z from 'zod'

export const AFFILIATE_SETTING_PREFIX = 'affiliate_setting.'

export const PROMOTION_NOTICE_TITLE_MAX_LENGTH = 100
export const PROMOTION_NOTICE_TEXT_MAX_LENGTH = 2000
// Both limits count Unicode characters (code points), matching the backend
// setting.ValidateAffiliatePromotionNoticeOption, not UTF-16 code units.

// Single source of truth for affiliate option defaults. System Settings and the
// affiliate management Commission Rules tab both read these keys, so the form,
// validation and persisted option names never diverge between the two entries.
export const AFFILIATE_DEFAULT_SETTINGS = {
  'affiliate_setting.first_level_enabled': false,
  'affiliate_setting.first_level_ratio': 0,
  'affiliate_setting.second_level_enabled': false,
  'affiliate_setting.second_level_ratio': 0,
  'affiliate_setting.settlement_delay_seconds': 0,
  'affiliate_setting.min_withdrawal_amount': 10,
  'affiliate_setting.trigger_topup_enabled': true,
  'affiliate_setting.trigger_subscription_enabled': false,
  'affiliate_setting.filter_redemption_topup_enabled': false,
  'affiliate_setting.payout_methods': 'usdt,alipay,wechat',
  'affiliate_setting.usdt_chain': 'TRC20',
  'affiliate_setting.promotion_template': '邀请链接：{invite_link}',
  'affiliate_setting.promotion_notice_enabled': false,
  'affiliate_setting.promotion_notice_title': '',
  'affiliate_setting.promotion_notice_text': '',
  'affiliate_setting.review_enabled': false,
  'affiliate_setting.auto_approve_after_days': 0,
  'affiliate_setting.agreement_enabled': false,
  'affiliate_setting.agreement_text': '',
  'affiliate_setting.inviter_min_account_age_days': 0,
  'affiliate_setting.inviter_min_recharge_amount': 0,
  'affiliate_setting.invitee_min_account_age_days': 0,
  'affiliate_setting.invitee_min_recharge_amount': 0,
}

export type AffiliateFlatSettings = typeof AFFILIATE_DEFAULT_SETTINGS

export const affiliateSchema = z.object({
  affiliate_setting: z.object({
    first_level_enabled: z.boolean(),
    first_level_ratio: z.coerce.number().min(0).max(100),
    second_level_enabled: z.boolean(),
    second_level_ratio: z.coerce.number().min(0).max(100),
    settlement_delay_seconds: z.coerce.number().min(0),
    min_withdrawal_amount: z.coerce.number().min(0),
    trigger_topup_enabled: z.boolean(),
    trigger_subscription_enabled: z.boolean(),
    filter_redemption_topup_enabled: z.boolean(),
    payout_methods: z.string().min(1),
    usdt_chain: z.string().min(1),
    promotion_template: z.string().min(1),
    promotion_notice_enabled: z.boolean(),
    promotion_notice_title: z
      .string()
      .refine(
        (value) => [...value].length <= PROMOTION_NOTICE_TITLE_MAX_LENGTH
      ),
    promotion_notice_text: z
      .string()
      .refine((value) => [...value].length <= PROMOTION_NOTICE_TEXT_MAX_LENGTH),
    review_enabled: z.boolean(),
    auto_approve_after_days: z.coerce.number().min(0),
    agreement_enabled: z.boolean(),
    agreement_text: z.string(),
    inviter_min_account_age_days: z.coerce.number().min(0),
    inviter_min_recharge_amount: z.coerce.number().min(0),
    invitee_min_account_age_days: z.coerce.number().min(0),
    invitee_min_recharge_amount: z.coerce.number().min(0),
  }),
})

export type AffiliateFormValues = z.infer<typeof affiliateSchema>

export function buildAffiliateFormValues(
  settings: Partial<AffiliateFlatSettings>
): AffiliateFormValues {
  const affiliateSetting: Record<string, string | number | boolean> = {}
  for (const [key, defaultValue] of Object.entries(
    AFFILIATE_DEFAULT_SETTINGS
  )) {
    const value = settings[key as keyof AffiliateFlatSettings]
    affiliateSetting[key.slice(AFFILIATE_SETTING_PREFIX.length)] =
      value ?? defaultValue
  }
  return { affiliate_setting: affiliateSetting } as AffiliateFormValues
}
