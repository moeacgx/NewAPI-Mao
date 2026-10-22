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
import { KeyRound, UserRound } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { TokenPicker } from '@/features/prompt-center/components/token-picker'
import type { PromptToken } from '@/features/prompt-center/types'

export function PlaygroundTokenSwitch(props: {
  token?: PromptToken
  disabled: boolean
  onChange: (token?: PromptToken) => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <div className='bg-background/80 mb-3 flex flex-wrap items-center gap-2 rounded-xl border px-3 py-2'>
      <span className='text-muted-foreground mr-auto flex min-w-0 items-center gap-2 text-xs'>
        {props.token ? (
          <KeyRound aria-hidden className='size-4 shrink-0' />
        ) : (
          <UserRound aria-hidden className='size-4 shrink-0' />
        )}
        <span className='text-foreground max-w-52 truncate font-medium'>
          {props.token?.name || t('Account mode')}
        </span>
        {props.token && (
          <span className='bg-primary/10 text-primary shrink-0 rounded-full px-2 py-0.5'>
            {t('Skills · {{count}}', { count: props.token.prompt_count ?? 0 })}
          </span>
        )}
      </span>
      {props.token && (
        <Button
          size='sm'
          variant='ghost'
          disabled={props.disabled}
          onClick={() => props.onChange()}
        >
          {t('Use account mode')}
        </Button>
      )}
      <Button
        size='sm'
        variant='outline'
        disabled={props.disabled}
        onClick={() => setOpen(true)}
      >
        <KeyRound className='size-3.5' aria-hidden />
        {t('Switch API key')}
      </Button>
      {open && !props.disabled && (
        <TokenPicker
          description={t(
            'Requests use the selected API key’s permissions, group, quota and installed prompts.'
          )}
          showGroups
          onClose={() => setOpen(false)}
          onSelect={(token) => {
            props.onChange(token)
            setOpen(false)
          }}
        />
      )}
    </div>
  )
}
