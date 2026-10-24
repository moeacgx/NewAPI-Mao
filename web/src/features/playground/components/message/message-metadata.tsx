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
import { Cloud } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

import { MESSAGE_ROLES, MESSAGE_STATUS } from '../../constants'
import type { MessageAlignment } from '../../lib'
import type { Message } from '../../types'

type MessageMetadataProps = {
  alignment: MessageAlignment
  message: Message
}

function formatMessageTime(timestamp?: number): string | undefined {
  if (typeof timestamp !== 'number' || !Number.isFinite(timestamp)) {
    return undefined
  }

  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(timestamp))
}

function formatDuration(
  durationMs: number | undefined,
  t: TFunction
): string | undefined {
  if (typeof durationMs !== 'number' || !Number.isFinite(durationMs)) {
    return undefined
  }

  if (durationMs < 1000) {
    return t('{{value}}ms', { value: Math.max(1, Math.round(durationMs)) })
  }

  return t('{{value}}s', { value: (durationMs / 1000).toFixed(2) })
}

export function MessageMetadata(props: MessageMetadataProps) {
  const { t } = useTranslation()
  const messageTime = formatMessageTime(props.message.createdAt)
  const duration = formatDuration(props.message.durationMs, t)
  const gatewaySkills = props.message.gatewaySkills
  const showGatewaySkills =
    props.message.from === MESSAGE_ROLES.ASSISTANT &&
    props.message.status !== MESSAGE_STATUS.LOADING &&
    props.message.status !== MESSAGE_STATUS.STREAMING &&
    props.message.status !== MESSAGE_STATUS.ERROR &&
    Boolean(gatewaySkills?.skills.length)

  if (!messageTime && !duration && !showGatewaySkills) {
    return null
  }

  const skillNames = gatewaySkills?.skills.map((skill) => skill.title) ?? []
  const skillsLabel = skillNames.join(', ')
  const gatewaySkillsLabel = t('Gateway cloud Skills: {{skills}}', {
    skills: skillsLabel,
  })

  return (
    <div
      className={cn(
        'text-muted-foreground mt-1 flex min-h-4 min-w-0 flex-wrap items-center gap-1.5 text-[11px] leading-none',
        props.alignment === 'right' && 'justify-end'
      )}
    >
      {showGatewaySkills && (
        <Badge
          aria-label={gatewaySkillsLabel}
          className='h-auto min-h-5 max-w-full min-w-0 whitespace-normal border-sky-500/25 bg-sky-500/8 px-1.5 py-0.5 text-[10px] leading-snug text-sky-700 dark:text-sky-300'
          title={gatewaySkillsLabel}
          variant='outline'
        >
          <Cloud aria-hidden='true' className='size-3 shrink-0' />
          <span className='shrink-0'>{t('Gateway cloud Skills')}</span>
          <span aria-hidden='true'>·</span>
          <span className='min-w-0 break-words'>{skillsLabel}</span>
        </Badge>
      )}
      {messageTime && <time>{messageTime}</time>}
      {duration && (
        <>
          {messageTime && <span aria-hidden='true'>·</span>}
          <span>{t('Response time: {{duration}}', { duration })}</span>
        </>
      )}
    </div>
  )
}
