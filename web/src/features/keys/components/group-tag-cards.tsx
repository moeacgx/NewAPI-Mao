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
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { GroupTagLogos } from '@/features/group-tags/components/group-tag-logos'
import type { GroupTagOption } from '@/features/group-tags/types'
import { cn } from '@/lib/utils'

import type { ApiKeyGroupOption } from './api-key-group-combobox'

type GroupTagCardsProps = {
  tags: GroupTagOption[]
  options: ApiKeyGroupOption[]
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}

// Large tag cards shown above the token group picker. Tags only filter the
// picker candidates; they never change the selected groups or their order.
export function GroupTagCards(props: GroupTagCardsProps) {
  const { t } = useTranslation()
  const concreteOptions = props.options.filter(
    (option) => option.value !== 'auto'
  )
  const choices = [
    {
      id: 'all',
      name: t('All groups'),
      icons: [] as string[],
      count: concreteOptions.length,
    },
    ...props.tags.map((tag) => ({
      id: String(tag.id),
      name: tag.name,
      icons: tag.icons,
      count: concreteOptions.filter(
        (option) => option.id !== undefined && tag.group_ids.includes(option.id)
      ).length,
    })),
  ].filter((choice) => choice.id === 'all' || choice.count > 0)
  const valid = choices.some((choice) => choice.id === props.value)
  const { onChange } = props
  useEffect(() => {
    if (!valid) onChange('all')
  }, [valid, onChange])

  if (props.tags.length === 0) return null

  return (
    <div
      role='group'
      aria-label={t('Group tags')}
      className='bg-muted/20 grid grid-cols-2 gap-2 rounded-lg border p-2 sm:grid-cols-3'
    >
      {choices.map((choice) => {
        const active = valid ? props.value === choice.id : choice.id === 'all'
        return (
          <button
            key={choice.id}
            type='button'
            aria-pressed={active}
            disabled={props.disabled}
            title={choice.name}
            onClick={() => props.onChange(choice.id)}
            className={cn(
              'bg-background hover:bg-muted/50 focus-visible:ring-ring/50 flex min-h-24 min-w-0 flex-col items-center gap-1.5 rounded-xl border px-2 py-2.5 text-center transition-colors outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50',
              active && 'border-primary bg-primary/5 ring-primary ring-1'
            )}
          >
            <span className='flex h-7 items-center justify-center'>
              <GroupTagLogos icons={choice.icons} />
            </span>
            <span className='line-clamp-2 w-full text-sm leading-5 font-semibold break-words'>
              {choice.name}
            </span>
            <span className='text-muted-foreground mt-auto text-xs'>
              {t('Groups: {{count}}', { count: choice.count })}
            </span>
          </button>
        )
      })}
    </div>
  )
}
