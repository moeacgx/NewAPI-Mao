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
import { Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import { describeLogSkillUsage, type LogSkillUsage } from '../lib/skill-usage'

/** 使用日志中的技能标记：已使用技能醒目显示，发现轮次与未用技能弱化显示。 */
export function SkillUsageBadge(props: { usage: LogSkillUsage }) {
  const { t } = useTranslation()
  const text = describeLogSkillUsage(props.usage, t)
  const isUsed = props.usage.kind === 'used'

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          render={<button type='button' />}
          aria-label={[text.label, ...text.lines].join('. ')}
          onClick={(event) => event.stopPropagation()}
          className={cn(
            'absolute -top-1 -right-1 inline-flex size-4 items-center justify-center rounded-full border border-background p-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring',
            isUsed
              ? 'bg-success/15 text-success'
              : 'bg-muted text-muted-foreground'
          )}
        >
          <Sparkles className='size-2.5' aria-hidden='true' />
        </TooltipTrigger>
        <TooltipContent className='max-w-[min(320px,calc(100vw-48px))] flex-col items-start gap-0.5 wrap-anywhere'>
          <strong>{text.label}</strong>
          {text.lines.map((line) => (
            <p key={line}>{line}</p>
          ))}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
