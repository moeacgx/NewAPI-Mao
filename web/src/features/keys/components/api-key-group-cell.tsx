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
import { useTranslation } from 'react-i18next'

import { BadgeCell, TruncatedCell } from '@/components/data-table'
import { GroupBadge } from '@/components/group-badge'
import { StatusBadge } from '@/components/status-badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { GroupIdentity } from '../lib/group-selection'
import {
  // AutoGroupBadge,
  GroupRatioBadge,
  type GroupRatio,
} from './auto-group-visuals'

type ApiKeyGroupCellProps = {
  crossGroupRetry: boolean
  group: string | null | undefined
  groupIds?: readonly number[]
  groupDetails?: readonly GroupIdentity[]
  groups?: Record<string, GroupIdentity>
  ratio?: GroupRatio
  shouldReduceMotion: boolean
}

export function ApiKeyGroupCell(props: ApiKeyGroupCellProps) {
  const { t } = useTranslation()

  if (props.group !== 'auto') {
    const ratio = typeof props.ratio === 'number' ? props.ratio : undefined
    const codes = (props.group || '')
      .split(',')
      .map((code) => code.trim())
      .filter(Boolean)
    const details = props.groupDetails || []
    const groups = Object.values(props.groups || {})
    const label = codes
      .map((code, index) => {
        // ID 与原标识完整对应时优先按 ID 关联，兼容改名后的历史 alias。
        // 缺失绑定导致长度不同时不能按位置猜测，避免把别组名称贴给已删除组。
        const id =
          props.groupIds?.length === codes.length
            ? props.groupIds[index]
            : undefined
        const detail = details.find((group) =>
          id !== undefined ? group.id === id : group.code === code
        )
        const known = groups.find((group) =>
          (id ?? detail?.id) !== undefined
            ? group.id === (id ?? detail?.id)
            : group.code === code
        )
        return detail?.name?.trim() || known?.name?.trim() || code
      })
      .join(', ')
    return (
      <TruncatedCell
        className='-ml-1.5'
        tooltipContent={label || '-'}
        tooltipClassName='break-all'
      >
        <GroupBadge
          group={props.group}
          label={label || undefined}
          ratio={ratio}
        />
      </TruncatedCell>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <BadgeCell
            data-api-key-group-cell='auto'
            className='gap-1.5 overflow-visible text-xs'
          />
        }
      >
        <StatusBadge label={t('Cross-group')} variant='info' copyable={false} />
        {/*<AutoGroupBadge shouldReduceMotion={props.shouldReduceMotion} />*/}
        <GroupRatioBadge
          ratio={props.ratio}
          isAuto
          shouldReduceMotion={props.shouldReduceMotion}
        />
      </TooltipTrigger>
      <TooltipContent>
        <span className='text-xs'>
          {t(
            'Automatically selects the best available group with circuit breaker mechanism'
          )}
        </span>
      </TooltipContent>
    </Tooltip>
  )
}
