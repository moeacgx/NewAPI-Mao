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
import {
  AlertCircleIcon,
  ArrowLeft01Icon,
  ArrowRight01Icon,
  Key01Icon,
  PuzzleIcon,
  Search01Icon,
  Tick02Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { searchApiKeys } from '@/features/keys/api'
import { cn } from '@/lib/utils'

import type { PromptToken } from '../types'

export function TokenPicker(props: {
  onClose: () => void
  onSelect: (token: PromptToken) => void
  description?: string
  showGroups?: boolean
}) {
  const { t } = useTranslation()
  const fieldId = useId()
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<PromptToken>()
  const query = useQuery({
    queryKey: ['prompt-center-tokens', search, page],
    queryFn: async () => {
      const result = await searchApiKeys({ keyword: search, p: page, size: 20 })
      if (!result.success || !result.data) {
        throw new Error(result.message || 'Failed to load API keys')
      }
      return result.data
    },
  })

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
    >
      <DialogContent className='max-h-[calc(100dvh-2rem)] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden rounded-2xl p-0 font-sans sm:max-w-xl'>
        <DialogHeader className='border-b px-5 py-5 pr-10 sm:px-6'>
          <div className='mb-1 flex items-center gap-3'>
            <span className='border-primary/20 bg-primary/10 text-primary flex size-11 shrink-0 items-center justify-center rounded-xl border'>
              <HugeiconsIcon icon={Key01Icon} className='size-5' aria-hidden />
            </span>
            <DialogTitle className='font-sans text-lg font-semibold'>
              {t('Choose an API key')}
            </DialogTitle>
          </div>
          <DialogDescription className='text-xs leading-relaxed sm:text-sm'>
            {props.description ??
              t(
                'Install prompts without changing the key, quota or group permissions.'
              )}
          </DialogDescription>
        </DialogHeader>
        <div className='flex min-h-0 flex-col gap-4 px-5 py-4 sm:px-6'>
          <div className='relative'>
            <Label className='sr-only' htmlFor={`${fieldId}-search`}>
              {t('Search by name')}
            </Label>
            <HugeiconsIcon
              icon={Search01Icon}
              className='text-muted-foreground pointer-events-none absolute top-3 left-3 size-4'
              aria-hidden
            />
            <Input
              id={`${fieldId}-search`}
              type='search'
              className='bg-background h-10 pl-9'
              placeholder={t('Search by name')}
              value={search}
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(1)
              }}
            />
          </div>
          <div className='text-muted-foreground flex items-center justify-between text-xs'>
            <span className='font-medium'>{t('API Keys')}</span>
            {query.isSuccess && <span>{query.data.total}</span>}
          </div>
          <div
            className='max-h-[min(24rem,42dvh)] min-h-0 flex-1 overflow-y-auto overscroll-contain pr-1'
            aria-busy={query.isFetching}
          >
            {query.isPending && (
              <div role='status' className='space-y-2'>
                <span className='sr-only'>{t('Loading...')}</span>
                {[0, 1, 2].map((item) => (
                  <Skeleton key={item} className='h-20 w-full rounded-xl' />
                ))}
              </div>
            )}
            {query.isError && (
              <div
                role='alert'
                className='bg-destructive/5 border-destructive/20 flex min-h-40 flex-col items-center justify-center gap-3 rounded-xl border border-dashed p-5 text-center'
              >
                <HugeiconsIcon
                  icon={AlertCircleIcon}
                  className='text-destructive size-6'
                  aria-hidden
                />
                <p className='text-muted-foreground text-sm'>
                  {t('Failed to load API keys')}
                </p>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => query.refetch()}
                >
                  {t('Retry')}
                </Button>
              </div>
            )}
            {query.isSuccess && (
              <ul className='space-y-2' aria-label={t('API Keys')}>
                {query.data.items.map((token) => (
                  <li key={token.id}>
                    <Button
                      variant='outline'
                      aria-label={token.name}
                      aria-pressed={selected?.id === token.id}
                      className={cn(
                        'h-auto min-h-20 w-full justify-start gap-3 rounded-xl p-3 text-left whitespace-normal sm:p-4',
                        selected?.id === token.id &&
                          'border-primary bg-primary/5 hover:bg-primary/10 dark:border-primary dark:bg-primary/10 dark:hover:bg-primary/15'
                      )}
                      onClick={() =>
                        setSelected({
                          id: token.id,
                          name: token.name,
                          prompt_count: token.prompt_count,
                          enabled_prompt_count: token.enabled_prompt_count,
                        })
                      }
                    >
                      <span
                        className={cn(
                          'bg-muted text-muted-foreground flex size-10 shrink-0 items-center justify-center rounded-lg',
                          selected?.id === token.id &&
                            'bg-primary/10 text-primary'
                        )}
                      >
                        <HugeiconsIcon
                          icon={Key01Icon}
                          className='size-5'
                          aria-hidden
                        />
                      </span>
                      <span className='min-w-0 flex-1 space-y-1.5'>
                        <span className='block truncate font-medium'>
                          {token.name}
                        </span>
                        {props.showGroups && (
                          <span className='text-muted-foreground block truncate text-xs font-normal'>
                            {(token.group_mode !== 'inherit' &&
                              token.group_details
                                ?.map((group) => group.name || group.code)
                                .join(' · ')) ||
                              t(
                                token.group_mode === 'auto' ||
                                  token.group === 'auto'
                                  ? 'Automatic selection'
                                  : 'Account default group'
                              )}
                          </span>
                        )}
                        <span className='text-muted-foreground flex items-center gap-1 text-xs font-normal'>
                          <HugeiconsIcon
                            icon={PuzzleIcon}
                            className='size-3.5'
                            aria-hidden
                          />
                          {t('Skills · {{count}}', {
                            count: token.prompt_count ?? 0,
                          })}
                        </span>
                      </span>
                      <span
                        className={cn(
                          'border-border flex size-5 shrink-0 items-center justify-center rounded-full border',
                          selected?.id === token.id &&
                            'border-primary bg-primary text-primary-foreground'
                        )}
                      >
                        {selected?.id === token.id && (
                          <HugeiconsIcon
                            icon={Tick02Icon}
                            className='size-3.5'
                            aria-hidden
                          />
                        )}
                      </span>
                    </Button>
                  </li>
                ))}
                {query.data.items.length === 0 && (
                  <li className='bg-muted/20 text-muted-foreground flex min-h-40 flex-col items-center justify-center gap-3 rounded-xl border border-dashed p-5 text-center text-sm'>
                    <HugeiconsIcon
                      icon={Search01Icon}
                      className='size-7 opacity-50'
                      aria-hidden
                    />
                    {t('No API keys found')}
                  </li>
                )}
              </ul>
            )}
          </div>
          {query.isSuccess && (
            <div className='flex items-center justify-between gap-2 border-t pt-3'>
              <Button
                variant='outline'
                size='sm'
                disabled={page === 1 || query.isFetching}
                onClick={() => {
                  setPage((value) => value - 1)
                }}
              >
                <HugeiconsIcon
                  icon={ArrowLeft01Icon}
                  className='size-3.5'
                  aria-hidden
                />
                {t('Previous')}
              </Button>
              <span className='text-muted-foreground text-xs'>
                {t('Page {{page}}', { page })}
              </span>
              <Button
                variant='outline'
                size='sm'
                disabled={page * 20 >= query.data.total || query.isFetching}
                onClick={() => {
                  setPage((value) => value + 1)
                }}
              >
                {t('Next')}
                <HugeiconsIcon
                  icon={ArrowRight01Icon}
                  className='size-3.5'
                  aria-hidden
                />
              </Button>
            </div>
          )}
        </div>
        <div className='bg-muted/40 flex flex-col gap-3 border-t px-5 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6'>
          <div className='min-w-0' role='status' aria-live='polite'>
            <p className='text-muted-foreground text-xs'>
              {t(selected ? 'Selected' : 'Select API key')}
            </p>
            {selected && (
              <p className='mt-1 truncate text-sm font-medium'>
                {selected.name}
              </p>
            )}
          </div>
          <Button
            className='h-10 gap-2 px-5 sm:shrink-0'
            disabled={!selected || query.isFetching || !query.isSuccess}
            onClick={() => {
              if (selected) props.onSelect(selected)
            }}
          >
            {t('Continue')}
            <HugeiconsIcon
              icon={ArrowRight01Icon}
              className='size-4'
              aria-hidden
            />
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
