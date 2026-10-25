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

import { StatusBadge } from '@/components/status-badge'
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
        <TooltipTrigger render={<span className='inline-flex' />}>
          <StatusBadge
            label={text.label}
            icon={Sparkles}
            variant={isUsed ? 'success' : 'neutral'}
            size='sm'
            showDot={false}
            copyable={false}
            title={undefined}
            aria-label={[text.label, ...text.lines].join('. ')}
            className={cn(
              'border',
              isUsed
                ? 'border-success/30 bg-success/10 text-success'
                : 'border-border bg-muted/50 text-muted-foreground'
            )}
          />
        </TooltipTrigger>
        <TooltipContent className='flex-col items-start gap-0.5'>
          {text.lines.map((line) => (
            <p key={line}>{line}</p>
          ))}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
