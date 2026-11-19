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
import { Reorder } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { ApiKeyGroupOption } from './api-key-group-combobox'
import { AutoGroupOrderItem } from './auto-group-order-editor'

type SelectedGroupOrderListProps = {
  value: string[]
  options: ApiKeyGroupOption[]
  onChange: (groups: string[]) => void
}

// Ordered list for two or more concrete groups picked in the normal picker.
// The first entry is attempted first; the wire format stays explicit so
// group_ids, channel pins and ratio protection are preserved.
export function SelectedGroupOrderList(props: SelectedGroupOrderListProps) {
  const { t } = useTranslation()

  const handleMove = (index: number, direction: 'up' | 'down') => {
    const targetIndex = direction === 'up' ? index - 1 : index + 1
    if (targetIndex < 0 || targetIndex >= props.value.length) return
    const next = [...props.value]
    ;[next[index], next[targetIndex]] = [next[targetIndex], next[index]]
    props.onChange(next)
  }

  return (
    <div className='flex flex-col gap-2'>
      <span className='text-sm font-medium'>{t('Custom Auto')}</span>
      <p className='text-muted-foreground text-xs'>
        {t('Groups are tried from top to bottom. Drag to change the priority.')}
      </p>
      <Reorder.Group
        axis='y'
        values={props.value}
        onReorder={props.onChange}
        aria-label={t('Group priority order')}
        className='flex flex-col gap-2'
      >
        {props.value.map((group, index) => (
          <AutoGroupOrderItem
            key={group}
            group={group}
            label={
              props.options.find((option) => option.value === group)?.label ||
              group
            }
            index={index}
            count={props.value.length}
            showPriority
            onMove={handleMove}
            onRemove={(removed) =>
              props.onChange(props.value.filter((item) => item !== removed))
            }
          />
        ))}
      </Reorder.Group>
    </div>
  )
}
