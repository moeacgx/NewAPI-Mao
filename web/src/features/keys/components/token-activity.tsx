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
import { Activity } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'

import type { TokenActivity } from '../hooks/use-token-activity'

export function TokenActivityBar(props: {
  activity: TokenActivity
  activeOnly: boolean
  onActiveOnlyChange: (value: boolean) => void
}) {
  const { t } = useTranslation()
  const summary = props.activity.snapshot?.summary
  return (
    <section
      aria-label={t('Token activity')}
      className='bg-card flex shrink-0 flex-col gap-2 rounded-lg border p-3 text-sm'
    >
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex min-h-6 flex-wrap items-center gap-x-3 gap-y-1 tabular-nums'>
          <Activity className='size-4 shrink-0' aria-hidden='true' />
          {summary ? (
            <>
              <strong>
                {t('{{count}} tokens in use', { count: summary.active_tokens })}
              </strong>
              <span className='text-muted-foreground'>
                {t('{{count}} active requests', { count: summary.total })}
              </span>
            </>
          ) : (
            <span>
              {props.activity.error
                ? t('Connection status is unavailable.')
                : t('Loading connection status…')}
            </span>
          )}
          {props.activity.stale && (
            <span className='text-warning'>{t('Stale')}</span>
          )}
        </div>
        <label className='flex cursor-pointer items-center gap-2 whitespace-nowrap'>
          <Checkbox
            checked={props.activeOnly}
            onCheckedChange={(value) =>
              props.onActiveOnlyChange(Boolean(value))
            }
          />
          {t('Only active tokens')}
        </label>
      </div>
      {props.activity.snapshot?.scope === 'instance' && (
        <p className='text-muted-foreground text-xs'>
          {t('Current instance only')}
        </p>
      )}
      {props.activity.error && (
        <div
          role='alert'
          className='text-destructive flex flex-wrap items-center justify-between gap-2'
        >
          <span>
            {t('Connection status is unavailable.')}
            {summary && ` ${t('Showing the last known state.')}`}
          </span>
          <Button
            variant='outline'
            size='sm'
            disabled={props.activity.refreshing}
            onClick={() => void props.activity.refresh()}
          >
            {t('Retry')}
          </Button>
        </div>
      )}
    </section>
  )
}

export function TokenActivityBadge(props: {
  tokenId: number
  count: number
  stale: boolean
}) {
  const { t } = useTranslation()
  if (!(props.count > 0)) return null
  return (
    <span
      data-token-id={props.tokenId}
      className='text-success bg-success/10 inline-flex max-w-full flex-wrap items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs tabular-nums'
    >
      {t('{{count}} active requests', { count: props.count })}
      {props.stale && <span>· {t('Stale')}</span>}
    </span>
  )
}
