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
import { zodResolver } from '@hookform/resolvers/zod'
import { type ChangeEvent, useMemo } from 'react'
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const affiliateSchema = z.object({
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

type AffiliateFormValues = z.infer<typeof affiliateSchema>

type Props = {
  defaultValues: AffiliateFormValues
}

const SETTLEMENT_DELAY_KEY = 'affiliate_setting.settlement_delay_seconds'
const PAYOUT_METHODS_KEY = 'affiliate_setting.payout_methods'
const PAYOUT_METHOD_OPTIONS = [
  ['usdt', 'USDT'],
  ['alipay', 'Alipay'],
  ['wechat', 'WeChat'],
] as const
function secondsToMinutes(value: number) {
  return Math.max(0, Math.round((Number(value) || 0) / 60))
}

function minutesToSeconds(value: number | string | boolean) {
  return Math.max(0, Math.round(Number(value) || 0)) * 60
}

export function AffiliateSettingsSection({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const displayDefaultValues = useMemo<AffiliateFormValues>(
    () => ({
      affiliate_setting: {
        ...defaultValues.affiliate_setting,
        settlement_delay_seconds: secondsToMinutes(
          defaultValues.affiliate_setting.settlement_delay_seconds
        ),
      },
    }),
    [defaultValues]
  )
  const handleNumberChange =
    (onChange: (value: number | string) => void) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      onChange(
        event.target.value === '' ? '' : event.currentTarget.valueAsNumber
      )
    }

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<AffiliateFormValues>({
      resolver: zodResolver(affiliateSchema) as Resolver<
        AffiliateFormValues,
        unknown,
        AffiliateFormValues
      >,
      defaultValues: displayDefaultValues,
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({
            key: key === SETTLEMENT_DELAY_KEY ? SETTLEMENT_DELAY_KEY : key,
            value:
              key === SETTLEMENT_DELAY_KEY
                ? minutesToSeconds(value as string | number | boolean)
                : (value as string | number | boolean),
          })
        }
      },
    })

  const selectedPayoutMethods = (
    form.watch('affiliate_setting.payout_methods') || ''
  )
    .split(',')
    .map((method) => method.trim())
    .filter(Boolean)

  const togglePayoutMethod = (method: string, checked: boolean) => {
    if (!checked && selectedPayoutMethods.length <= 1) {
      toast.error(t('At least one payout method must remain enabled'))
      return
    }
    const next = checked
      ? [...selectedPayoutMethods, method]
      : selectedPayoutMethods.filter((item) => item !== method)
    const unique = PAYOUT_METHOD_OPTIONS.map(([value]) => value).filter(
      (value) => next.includes(value)
    )
    form.setValue(PAYOUT_METHODS_KEY, unique.join(','), {
      shouldDirty: true,
      shouldValidate: true,
    })
  }

  return (
    <SettingsSection title={t('Affiliate Commission')}>
      <Tabs defaultValue='rules' className='space-y-6'>
        <TabsList>
          <TabsTrigger value='rules'>{t('Commission rules')}</TabsTrigger>
          <TabsTrigger value='anti-fraud'>{t('Anti-Fraud')}</TabsTrigger>
        </TabsList>
        <TabsContent value='rules'>
          <FormNavigationGuard when={isDirty} />
          <Form {...form}>
            <form onSubmit={handleSubmit} className='space-y-6'>
              <FormDirtyIndicator isDirty={isDirty} />

              <div className='grid gap-4 md:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='affiliate_setting.first_level_enabled'
                  render={({ field }) => (
                    <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                      <div className='space-y-0.5'>
                        <FormLabel>{t('Enable level 1 commission')}</FormLabel>
                        <FormDescription>
                          {t('Reward the direct inviter after a paid order.')}
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='affiliate_setting.second_level_enabled'
                  render={({ field }) => (
                    <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                      <div className='space-y-0.5'>
                        <FormLabel>{t('Enable level 2 commission')}</FormLabel>
                        <FormDescription>
                          {t('Reward the inviter above the direct inviter.')}
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>

              <div className='grid gap-4 md:grid-cols-2'>
                {[
                  ['affiliate_setting.first_level_ratio', 'Level 1 ratio (%)'],
                  ['affiliate_setting.second_level_ratio', 'Level 2 ratio (%)'],
                  [
                    'affiliate_setting.settlement_delay_seconds',
                    'Settlement delay minutes',
                  ],
                  [
                    'affiliate_setting.min_withdrawal_amount',
                    'Minimum withdrawal',
                  ],
                ].map(([name, label]) => (
                  <FormField
                    key={name}
                    control={form.control}
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    name={name as any}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t(label)}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            value={field.value ?? ''}
                            onChange={handleNumberChange(field.onChange)}
                            name={field.name}
                            onBlur={field.onBlur}
                            ref={field.ref}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ))}
              </div>

              <div className='grid gap-4 md:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='affiliate_setting.trigger_topup_enabled'
                  render={({ field }) => (
                    <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                      <div className='space-y-0.5'>
                        <FormLabel>
                          {t('Top-up orders trigger commission')}
                        </FormLabel>
                        <FormDescription>
                          {t('Generate commission after successful top-ups.')}
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='affiliate_setting.trigger_subscription_enabled'
                  render={({ field }) => (
                    <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                      <div className='space-y-0.5'>
                        <FormLabel>
                          {t('Subscription orders trigger commission')}
                        </FormLabel>
                        <FormDescription>
                          {t(
                            'Generate commission after subscription purchases.'
                          )}
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name='affiliate_setting.filter_redemption_topup_enabled'
                render={({ field }) => (
                  <FormItem className='flex items-center justify-between rounded-lg border p-4'>
                    <div className='space-y-0.5'>
                      <FormLabel>
                        {t('Filter redemption-code top-ups')}
                      </FormLabel>
                      <FormDescription>
                        {t(
                          'When enabled, quota added by redemption codes does not generate commission.'
                        )}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='affiliate_setting.payout_methods'
                render={() => (
                  <FormItem>
                    <FormLabel>{t('Supported payout methods')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Only enabled methods are shown on the affiliate page.'
                      )}
                    </FormDescription>
                    <div className='grid gap-3 md:grid-cols-3'>
                      {PAYOUT_METHOD_OPTIONS.map(([method, label]) => (
                        <label
                          key={method}
                          className='flex items-center justify-between rounded-lg border p-4 text-sm'
                        >
                          <span>{t(label)}</span>
                          <Switch
                            checked={selectedPayoutMethods.includes(method)}
                            onCheckedChange={(checked) =>
                              togglePayoutMethod(method, checked)
                            }
                          />
                        </label>
                      ))}
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='affiliate_setting.usdt_chain'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('USDT payout chain')}</FormLabel>
                    <FormControl>
                      <Input placeholder='TRC20' {...field} />
                    </FormControl>
                    <FormDescription>
                      {t('Users see this chain when saving a USDT address.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='affiliate_setting.promotion_template'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Promotion copy template')}</FormLabel>
                    <FormControl>
                      <Textarea className='min-h-28' {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Use {invite_link} where the referral link should appear.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Button
                type='submit'
                disabled={isSubmitting || updateOption.isPending}
              >
                {isSubmitting || updateOption.isPending
                  ? t('Saving...')
                  : t('Save affiliate settings')}
              </Button>
            </form>
          </Form>
        </TabsContent>

        <TabsContent value='anti-fraud' className='space-y-6'>
          <Form {...form}>
            <form onSubmit={handleSubmit} className='space-y-6'>
              <div className='space-y-1'>
                <h3 className='text-base font-semibold'>
                  {t('Inviter Review')}
                </h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Require inviters to apply and be approved before earning commissions'
                  )}
                </p>
              </div>
              <FormField
                control={form.control}
                name='affiliate_setting.review_enabled'
                render={({ field }) => (
                  <FormItem className='flex items-center gap-3'>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormLabel>{t('Enable affiliate review')}</FormLabel>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='affiliate_setting.auto_approve_after_days'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Auto-approve after days')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        {...field}
                        onChange={handleNumberChange(field.onChange)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Set to 0 for manual approval only')}
                    </FormDescription>
                  </FormItem>
                )}
              />

              <div className='space-y-1 border-t pt-4'>
                <h3 className='text-base font-semibold'>{t('Agreement')}</h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Require inviters to accept anti-fraud agreement when applying'
                  )}
                </p>
              </div>
              <FormField
                control={form.control}
                name='affiliate_setting.agreement_enabled'
                render={({ field }) => (
                  <FormItem className='flex items-center gap-3'>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormLabel>{t('Enable agreement')}</FormLabel>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='affiliate_setting.agreement_text'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Agreement text')}</FormLabel>
                    <FormControl>
                      <Textarea rows={4} {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Displayed to users when applying for affiliate program'
                      )}
                    </FormDescription>
                  </FormItem>
                )}
              />

              <div className='space-y-1 border-t pt-4'>
                <h3 className='text-base font-semibold'>
                  {t('Inviter eligibility')}
                </h3>
              </div>
              <div className='grid gap-4 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='affiliate_setting.inviter_min_account_age_days'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Min account age (days)')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          min={0}
                          {...field}
                          onChange={handleNumberChange(field.onChange)}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='affiliate_setting.inviter_min_recharge_amount'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Min recharge amount (quota)')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          min={0}
                          {...field}
                          onChange={handleNumberChange(field.onChange)}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>

              <div className='space-y-1 border-t pt-4'>
                <h3 className='text-base font-semibold'>
                  {t('Invitee eligibility')}
                </h3>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Invitee must meet these conditions for commissions to be generated'
                  )}
                </p>
              </div>
              <div className='grid gap-4 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='affiliate_setting.invitee_min_account_age_days'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Min account age (days)')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          min={0}
                          {...field}
                          onChange={handleNumberChange(field.onChange)}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='affiliate_setting.invitee_min_recharge_amount'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Min recharge amount (quota)')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          min={0}
                          {...field}
                          onChange={handleNumberChange(field.onChange)}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>

              <Button type='submit' disabled={isSubmitting || !isDirty}>
                <FormDirtyIndicator isDirty={isDirty} />
                {isSubmitting ? t('Saving...') : t('Save settings')}
              </Button>
            </form>
          </Form>
        </TabsContent>
      </Tabs>
    </SettingsSection>
  )
}
