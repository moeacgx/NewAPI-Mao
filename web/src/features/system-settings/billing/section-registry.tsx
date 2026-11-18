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
import { parseCurrencyDisplayType } from '@/lib/currency'

import { CheckinSettingsSection } from '../general/checkin-settings-section'
import { PricingSection } from '../general/pricing-section'
import { QuotaSettingsSection } from '../general/quota-settings-section'
import { PaymentSettingsSection } from '../integrations/payment-settings-section'
import { RatioSettingsCard } from '../models/ratio-settings-card'
import type { BillingSettings } from '../types'
import { createSectionRegistry } from '../utils/section-registry'
import { buildAffiliateFormValues } from './affiliate-config'
import { AffiliateSettingsSection } from './affiliate-settings-section'

const getModelDefaults = (settings: BillingSettings) => ({
  ModelPrice: settings.ModelPrice,
  ModelPriceUnit: settings.ModelPriceUnit,
  ModelPriceVariants: settings.ModelPriceVariants,
  ModelRoutePriceVariants: settings.ModelRoutePriceVariants,
  ModelRatio: settings.ModelRatio,
  CacheRatio: settings.CacheRatio,
  CreateCacheRatio: settings.CreateCacheRatio,
  CompletionRatio: settings.CompletionRatio,
  ImageRatio: settings.ImageRatio,
  AudioRatio: settings.AudioRatio,
  AudioCompletionRatio: settings.AudioCompletionRatio,
  ExposeRatioEnabled: settings.ExposeRatioEnabled,
  BillingMode: settings['billing_setting.billing_mode'],
  BillingExpr: settings['billing_setting.billing_expr'],
})

const getGroupDefaults = (settings: BillingSettings) => ({
  TopupGroupRatio: settings.TopupGroupRatio,
  GroupRatio: settings.GroupRatio,
  UserUsableGroups: settings.UserUsableGroups,
  GroupGroupRatio: settings.GroupGroupRatio,
  AutoGroups: settings.AutoGroups,
  MaxTokenAutoGroups: settings.MaxTokenAutoGroups,
  DefaultUseAutoGroup: settings.DefaultUseAutoGroup,
  GroupSpecialUsableGroup:
    settings['group_ratio_setting.group_special_usable_group'],
})

const BILLING_SECTIONS = [
  {
    id: 'quota',
    titleKey: 'Quota Settings',
    build: (settings: BillingSettings) => (
      <QuotaSettingsSection
        defaultValues={{
          QuotaForNewUser: settings.QuotaForNewUser,
          PreConsumedQuota: settings.PreConsumedQuota,
          QuotaForInviter: settings.QuotaForInviter,
          QuotaForInvitee: settings.QuotaForInvitee,
          TopUpLink: settings.TopUpLink,
          general_setting: {
            docs_link: settings['general_setting.docs_link'],
          },
          quota_setting: {
            enable_free_model_pre_consume:
              settings['quota_setting.enable_free_model_pre_consume'],
          },
        }}
        complianceConfirmed={
          (settings['payment_setting.compliance_confirmed'] ?? false) &&
          settings['payment_setting.compliance_terms_version'] === 'v1'
        }
      />
    ),
  },
  {
    id: 'currency',
    titleKey: 'Currency & Display',
    build: (settings: BillingSettings) => (
      <PricingSection
        defaultValues={{
          QuotaPerUnit: settings.QuotaPerUnit,
          USDExchangeRate: settings.USDExchangeRate,
          DisplayInCurrencyEnabled: settings.DisplayInCurrencyEnabled,
          DisplayTokenStatEnabled: settings.DisplayTokenStatEnabled,
          general_setting: {
            quota_display_type: parseCurrencyDisplayType(
              settings['general_setting.quota_display_type']
            ),
            auto_usd_exchange_rate:
              settings['general_setting.auto_usd_exchange_rate'] ?? true,
            custom_currency_symbol:
              settings['general_setting.custom_currency_symbol'] ?? '¤',
            custom_currency_exchange_rate:
              settings['general_setting.custom_currency_exchange_rate'] ?? 1,
          },
        }}
      />
    ),
  },
  {
    id: 'model-pricing',
    titleKey: 'Model Pricing',
    build: (settings: BillingSettings) => (
      <RatioSettingsCard
        titleKey='Model Pricing'
        modelDefaults={getModelDefaults(settings)}
        groupDefaults={getGroupDefaults(settings)}
        toolPricesDefault={settings['tool_price_setting.prices']}
        visibleTabs={['models', 'unset-models', 'tool-prices', 'upstream-sync']}
      />
    ),
  },
  {
    id: 'group-pricing',
    titleKey: 'Group Pricing',
    build: (settings: BillingSettings) => (
      <RatioSettingsCard
        titleKey='Group Pricing'
        modelDefaults={getModelDefaults(settings)}
        groupDefaults={getGroupDefaults(settings)}
        toolPricesDefault={settings['tool_price_setting.prices']}
        visibleTabs={['groups']}
      />
    ),
  },
  {
    id: 'payment',
    titleKey: 'Payment Gateway',
    build: (settings: BillingSettings) => (
      <PaymentSettingsSection
        defaultValues={{
          PayAddress: settings.PayAddress,
          EpayId: settings.EpayId,
          EpayKey: settings.EpayKey,
          Price: settings.Price,
          MinTopUp: settings.MinTopUp,
          CustomCallbackAddress: settings.CustomCallbackAddress,
          PayMethods: settings.PayMethods,
          AmountOptions: settings['payment_setting.amount_options'],
          AmountDiscount: settings['payment_setting.amount_discount'],
          BalanceSubscriptionEnabled:
            settings['payment_setting.balance_subscription_enabled'] ?? true,
          BalanceSubscriptionPromoEnabled:
            settings['payment_setting.balance_subscription_promo_enabled'] ??
            true,
          InvoiceEnabled: settings.InvoiceEnabled ?? false,
          InvoiceDiscountDisabled: settings.InvoiceDiscountDisabled ?? false,
          InvoiceTypes: settings.InvoiceTypes ?? '["personal","company"]',
          InvoiceKinds: settings.InvoiceKinds ?? '["normal"]',
          InvoiceFeeRules:
            settings.InvoiceFeeRules ??
            '[{"min":0,"max":500,"type":"fixed","value":50},{"min":501,"max":2000,"type":"fixed","value":100},{"min":2001,"max":5000,"type":"fixed","value":175},{"min":5000,"type":"percent","value":5}]',
          StripeApiSecret: settings.StripeApiSecret,
          StripeWebhookSecret: settings.StripeWebhookSecret,
          StripePriceId: settings.StripePriceId,
          StripeUnitPrice: settings.StripeUnitPrice,
          StripeMinTopUp: settings.StripeMinTopUp,
          StripePromotionCodesEnabled: settings.StripePromotionCodesEnabled,
          CreemApiKey: settings.CreemApiKey,
          CreemWebhookSecret: settings.CreemWebhookSecret,
          CreemTestMode: settings.CreemTestMode,
          CreemProducts: settings.CreemProducts,
          BepusdtApiUrl: settings.BepusdtApiUrl ?? '',
          BepusdtAuthToken: settings.BepusdtAuthToken ?? '',
          BepusdtUnitPrice: settings.BepusdtUnitPrice ?? 7.2,
          BepusdtMinTopUp: settings.BepusdtMinTopUp ?? 1,
          BepusdtTimeout: settings.BepusdtTimeout ?? 1200,
          BepusdtChains: settings.BepusdtChains ?? '[]',
          OkpayGatewayUrl:
            settings.OkpayGatewayUrl ?? 'https://api.okaypay.me/shop',
          OkpayMerchantId: settings.OkpayMerchantId ?? '',
          OkpayMerchantToken: settings.OkpayMerchantToken ?? '',
          OkpayExchangeRate: settings.OkpayExchangeRate ?? 7.2,
          OkpayAutoExchangeEnabled: settings.OkpayAutoExchangeEnabled ?? true,
          OkpayUsdtCnyRate: settings.OkpayUsdtCnyRate ?? 7.2,
          OkpayRateApiUrl:
            settings.OkpayRateApiUrl ??
            'https://api.coingecko.com/api/v3/simple/price?ids=tether&vs_currencies=cny&include_last_updated_at=true',
          OkpayRateSource: settings.OkpayRateSource ?? 'coingecko',
          OkpayOkxSide: settings.OkpayOkxSide ?? 'buy',
          OkpayOkxTier: settings.OkpayOkxTier ?? 3,
          OkpayRateAdjustmentType:
            settings.OkpayRateAdjustmentType ?? 'absolute',
          OkpayRateAdjustmentValue: settings.OkpayRateAdjustmentValue ?? 0,
          OkpayMinTopUp: settings.OkpayMinTopUp ?? 1,
          OkpayCoin: settings.OkpayCoin ?? 'USDT',
        }}
        waffoDefaultValues={{
          WaffoEnabled: settings.WaffoEnabled ?? false,
          WaffoApiKey: settings.WaffoApiKey ?? '',
          WaffoPrivateKey: settings.WaffoPrivateKey ?? '',
          WaffoPublicCert: settings.WaffoPublicCert ?? '',
          WaffoSandboxPublicCert: settings.WaffoSandboxPublicCert ?? '',
          WaffoSandboxApiKey: settings.WaffoSandboxApiKey ?? '',
          WaffoSandboxPrivateKey: settings.WaffoSandboxPrivateKey ?? '',
          WaffoSandbox: settings.WaffoSandbox ?? false,
          WaffoMerchantId: settings.WaffoMerchantId ?? '',
          WaffoCurrency: settings.WaffoCurrency ?? 'USD',
          WaffoUnitPrice: settings.WaffoUnitPrice ?? 1,
          WaffoMinTopUp: settings.WaffoMinTopUp ?? 1,
          WaffoNotifyUrl: settings.WaffoNotifyUrl ?? '',
          WaffoReturnUrl: settings.WaffoReturnUrl ?? '',
          WaffoPayMethods: settings.WaffoPayMethods ?? '[]',
        }}
        waffoPancakeDefaultValues={{
          WaffoPancakeMerchantID: settings.WaffoPancakeMerchantID ?? '',
          WaffoPancakePrivateKey: settings.WaffoPancakePrivateKey ?? '',
          WaffoPancakeReturnURL: settings.WaffoPancakeReturnURL ?? '',
        }}
        waffoPancakeProvisionedStoreID={settings.WaffoPancakeStoreID ?? ''}
        waffoPancakeProvisionedProductID={settings.WaffoPancakeProductID ?? ''}
        complianceDefaults={{
          confirmed: settings['payment_setting.compliance_confirmed'] ?? false,
          termsVersion:
            settings['payment_setting.compliance_terms_version'] ?? '',
          confirmedAt: settings['payment_setting.compliance_confirmed_at'] ?? 0,
          confirmedBy: settings['payment_setting.compliance_confirmed_by'] ?? 0,
        }}
      />
    ),
  },
  {
    id: 'affiliate',
    titleKey: 'Affiliate Commission',
    descriptionKey: 'Configure paid-referral commission and payouts',
    build: (settings: BillingSettings) => (
      <AffiliateSettingsSection
        defaultValues={buildAffiliateFormValues(settings)}
      />
    ),
  },
  {
    id: 'checkin',
    titleKey: 'Check-in Rewards',
    build: (settings: BillingSettings) => (
      <CheckinSettingsSection
        defaultValues={{
          enabled: settings['checkin_setting.enabled'],
          minQuota: settings['checkin_setting.min_quota'],
          maxQuota: settings['checkin_setting.max_quota'],
        }}
      />
    ),
  },
] as const

export type BillingSectionId = (typeof BILLING_SECTIONS)[number]['id']

const billingRegistry = createSectionRegistry<
  BillingSectionId,
  BillingSettings
>({
  sections: BILLING_SECTIONS,
  defaultSection: 'quota',
  basePath: '/system-settings/billing',
  urlStyle: 'path',
})

export const BILLING_SECTION_IDS = billingRegistry.sectionIds
export const BILLING_DEFAULT_SECTION = billingRegistry.defaultSection
export const getBillingSectionNavItems = billingRegistry.getSectionNavItems
export const getBillingSectionContent = billingRegistry.getSectionContent
export const getBillingSectionMeta = billingRegistry.getSectionMeta
