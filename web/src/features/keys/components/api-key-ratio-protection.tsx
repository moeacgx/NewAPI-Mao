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
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import type { GroupSelectionOption } from '../lib/group-selection'
import {
  isValidRatioLimit,
  type RatioProtectionDraft,
} from '../lib/ratio-protection'

export function ApiKeyRatioProtection(props: {
  value?: RatioProtectionDraft
  onChange: (value: RatioProtectionDraft) => void
  selectedGroups: readonly string[]
  options: readonly GroupSelectionOption[]
}) {
  const { t } = useTranslation()
  const id = useId()
  const values = props.value?.values || {}
  // 保留可解释的历史保护，不因只编辑名称或 Auto 顺序而静默清空。
  const codes = [
    ...new Set([
      ...props.selectedGroups.filter((code) => code !== 'auto'),
      ...Object.keys(values),
    ]),
  ]
  return (
    <section
      aria-label={t('Ratio protection')}
      className='bg-muted/20 space-y-3 rounded-lg border p-3'
    >
      <h3 className='text-sm font-medium'>{t('Ratio protection')}</h3>
      <p className='text-muted-foreground text-xs'>
        {t('Set a maximum group ratio. Empty or zero removes protection.')}
      </p>
      {props.value?.invalid && (
        <p role='alert' className='text-destructive text-sm'>
          {t(
            'Stored ratio protection is invalid. Clear it explicitly before saving.'
          )}
        </p>
      )}
      {!props.value?.invalid &&
        codes.length > 0 &&
        codes.map((code, index) => {
          const name =
            props.options.find((option) => option.value === code)?.label || code
          const value = Object.hasOwn(values, code) ? values[code] : ''
          const invalid = !isValidRatioLimit(value)
          const inputId = `${id}-${index}`
          return (
            <div
              key={code}
              className='grid gap-2 sm:grid-cols-[1fr_9rem] sm:items-center'
            >
              <label htmlFor={inputId} className='min-w-0 text-sm break-words'>
                {name}
              </label>
              <Input
                id={inputId}
                aria-label={t('Ratio limit for {{name}}', { name })}
                aria-invalid={invalid}
                aria-describedby={invalid ? `${inputId}-error` : undefined}
                inputMode='decimal'
                placeholder={t('Unlimited')}
                value={value}
                onChange={(event) =>
                  props.onChange({
                    values: { ...values, [code]: event.target.value },
                    invalid: false,
                  })
                }
              />
              {invalid && (
                <p
                  id={`${inputId}-error`}
                  role='alert'
                  className='text-destructive text-xs sm:col-span-2'
                >
                  {t('Enter a finite ratio of zero or greater.')}
                </p>
              )}
            </div>
          )
        })}
      {!props.value?.invalid && codes.length === 0 && (
        <p className='text-muted-foreground text-xs'>
          {t('Select explicit groups to configure ratio protection.')}
        </p>
      )}
      {(props.value?.invalid || Object.keys(values).length > 0) && (
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => props.onChange({ values: {}, invalid: false })}
        >
          {t('Clear ratio protection')}
        </Button>
      )}
    </section>
  )
}
