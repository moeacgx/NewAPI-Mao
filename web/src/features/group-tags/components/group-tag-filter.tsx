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

import { Button } from '@/components/ui/button'

import type { GroupTagOption } from '../types'
import { GroupTagLogos } from './group-tag-logos'

export function GroupTagFilter(props: {
  tags: GroupTagOption[]
  value: string
  onChange: (value: string) => void
  showUntagged?: boolean
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const valid =
    props.value === 'all' ||
    (props.showUntagged && props.value === 'untagged') ||
    props.tags.some((tag) => String(tag.id) === props.value)
  const selected = valid ? props.value : 'all'
  const { onChange } = props
  useEffect(() => {
    if (!valid) onChange('all')
  }, [valid, onChange])
  const choices = [
    { id: 'all', name: t('All groups'), icons: [] },
    ...(props.showUntagged
      ? [{ id: 'untagged', name: t('Untagged'), icons: [] }]
      : []),
    ...props.tags.map((tag) => ({ ...tag, id: String(tag.id) })),
  ]
  return (
    <div
      role='group'
      aria-label={t('Group tags')}
      className='bg-muted/20 flex flex-wrap gap-2 rounded-lg border p-2'
    >
      {choices.map((choice) => (
        <Button
          key={choice.id}
          type='button'
          size='sm'
          variant={selected === choice.id ? 'default' : 'outline'}
          aria-pressed={selected === choice.id}
          disabled={props.disabled}
          title={choice.name}
          onClick={() => props.onChange(choice.id)}
          className='max-w-full gap-2'
        >
          <GroupTagLogos icons={choice.icons} />
          <span className='truncate'>{choice.name}</span>
        </Button>
      ))}
    </div>
  )
}
