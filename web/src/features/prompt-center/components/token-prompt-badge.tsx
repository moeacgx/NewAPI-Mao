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
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import type { PromptToken } from '../types'
import { TokenPromptDialog } from './token-prompt-dialog'

export function TokenPromptBadge(props: { token: PromptToken }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const rowRef = useRef<Element | null>(null)
  const count = props.token.prompt_count ?? 0
  const active = (props.token.enabled_prompt_count ?? 0) > 0

  // 列表快照把数量刷成 0 时只隐藏徽标，已打开的弹窗必须保持挂载，由用户自行关闭。
  if (count === 0 && !open) return null

  return (
    <>
      {count > 0 && (
        <button
          type='button'
          data-active={active}
          aria-label={
            active
              ? t('Manage installed skills ({{count}})', { count })
              : t('Manage installed skills ({{count}}), all inactive', {
                  count,
                })
          }
          ref={triggerRef}
          onClick={(event) => {
            // 记住所在行：计数归零后触发按钮消失，关闭时需要回到同一令牌的稳定控件。
            rowRef.current = event.currentTarget.closest('tr, [data-token-row]')
            setOpen(true)
          }}
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
      )}
      {open && (
        <TokenPromptDialog
          open={open}
          onOpenChange={setOpen}
          token={props.token}
          finalFocus={() => {
            // 焦点已被其他对话框或控件占用时不抢焦点。
            const active = document.activeElement
            if (
              active &&
              active !== document.body &&
              !active.closest('[data-token-prompt-dialog]')
            ) {
              return false
            }
            // 优先当前连接的徽标；归零后回到同一令牌行的第一个可操作控件。
            if (triggerRef.current?.isConnected) return triggerRef.current
            const row = rowRef.current
            if (!row?.isConnected) return null
            const controls = row.querySelectorAll<HTMLElement>(
              'button, input, a[href], [role="checkbox"], [tabindex]'
            )
            return (
              [...controls].find(
                (control) =>
                  control.tabIndex >= 0 && !control.matches(':disabled')
              ) ?? null
            )
          }}
        />
      )}
    </>
  )
}
