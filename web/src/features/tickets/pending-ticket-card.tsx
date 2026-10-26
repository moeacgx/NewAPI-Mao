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
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  CircleCheckBig,
  Inbox,
  Loader,
  Lock,
  RefreshCw,
  Ticket,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { usePendingTickets } from './pending-ticket-context'

const statusItems = [
  { status: 'open', icon: Inbox, tone: 'bg-warning/10 text-warning' },
  { status: 'processing', icon: Loader, tone: 'bg-info/10 text-info' },
  {
    status: 'resolved',
    icon: CircleCheckBig,
    tone: 'bg-success/10 text-success',
  },
  { status: 'closed', icon: Lock, tone: 'bg-muted text-muted-foreground' },
] as const

export function PendingTicketCard() {
  const { t } = useTranslation()
  const summary = usePendingTickets()
  if (!summary) return null
  const labels = {
    open: t('Pending tickets'),
    processing: t('In progress'),
    resolved: t('Resolved'),
    closed: t('Closed'),
  }
  return (
    <Card aria-label={t('Ticket overview')}>
      <CardHeader className='flex flex-wrap items-center justify-between gap-3 border-b'>
        <CardTitle className='flex items-center gap-2'>
          <span className='bg-primary/10 text-primary flex size-8 items-center justify-center rounded-lg'>
            <Ticket className='size-4' aria-hidden='true' />
          </span>
          {t('Ticket overview')}
        </CardTitle>
        <div className='flex items-center gap-2'>
          <Button
            variant='outline'
            size='icon'
            aria-label={t('Refresh')}
            title={t('Refresh')}
            onClick={summary.refresh}
          >
            <RefreshCw aria-hidden='true' />
          </Button>
          <Link
            to='/ticket-management'
            search={{ status: '' }}
            className='border-primary/20 bg-primary/8 text-primary hover:bg-primary/14 focus-visible:ring-ring inline-flex h-8 items-center gap-1.5 rounded-md border px-2.5 text-[0.8rem] font-medium whitespace-nowrap transition-colors outline-none focus-visible:ring-3'
          >
            {t('View tickets')}
            <ArrowRight className='size-3.5' aria-hidden='true' />
          </Link>
        </div>
      </CardHeader>
      <CardContent>
        {summary.error ? (
          <div className='flex flex-wrap items-center justify-between gap-4'>
            <p role='status'>{t('Failed to load ticket overview')}</p>
            <Button variant='outline' onClick={summary.refresh}>
              <RefreshCw aria-hidden='true' />
              {t('Retry')}
            </Button>
          </div>
        ) : (
          <ul className='grid grid-cols-2 gap-3 md:grid-cols-4'>
            {statusItems.map((item) => {
              const Icon = item.icon
              const count = summary.counts?.[item.status]
              return (
                <li key={item.status}>
                  <Link
                    to='/ticket-management'
                    search={{ status: item.status }}
                    className='hover:bg-muted/50 focus-visible:ring-ring flex items-center gap-3 rounded-lg border p-3 transition-colors outline-none focus-visible:ring-3'
                  >
                    <span
                      className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${item.tone}`}
                    >
                      <Icon className='size-4' aria-hidden='true' />
                    </span>
                    <span className='min-w-0'>
                      <span className='text-muted-foreground block truncate text-sm'>
                        {labels[item.status]}
                      </span>
                      <strong
                        className='block text-2xl tabular-nums'
                        aria-label={
                          item.status === 'open'
                            ? t('Pending ticket count')
                            : undefined
                        }
                      >
                        {count === undefined ? '—' : count.toLocaleString()}
                      </strong>
                    </span>
                  </Link>
                </li>
              )
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
