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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import type { PromptToken } from '../types'
import { TokenPromptDialog } from './token-prompt-dialog'

export function TokenPromptBadge(props: { token: PromptToken }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const count = props.token.prompt_count ?? 0
  if (count === 0) return null
  const active = (props.token.enabled_prompt_count ?? 0) > 0

  return (
    <>
      <button
        type='button'
        data-active={active}
        aria-label={
          active
            ? t('Manage installed skills ({{count}})', { count })
            : t('Manage installed skills ({{count}}), all inactive', { count })
        }
        onClick={() => setOpen(true)}
        className={cn(
          'focus-visible:ring-ring/50 inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-semibold shadow-sm transition-all hover:-translate-y-px hover:shadow focus-visible:ring-2 focus-visible:outline-none',
          active
            ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-300'
            : 'bg-muted/70 text-muted-foreground'
        )}
      >
        <span className='bg-background/70 flex size-4 items-center justify-center rounded-full'>
          <Sparkles className='size-2.5' aria-hidden='true' />
        </span>
        {t('Skills · {{count}}', { count })}
      </button>
      {open && (
        <TokenPromptDialog
          open={open}
          onOpenChange={setOpen}
          token={props.token}
        />
      )}
    </>
  )
}
