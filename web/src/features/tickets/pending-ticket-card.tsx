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
import { ArrowRight, ListChecks, RefreshCw, Ticket } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { usePendingTickets } from './pending-ticket-context'

export function PendingTicketCard() {
  const { t } = useTranslation()
  const summary = usePendingTickets()
  if (!summary) return null
  return (
    <Card aria-label={t('Pending tickets')}>
      <CardHeader className='border-b'>
        <CardTitle className='flex items-center gap-2'>
          <span className='bg-primary/10 text-primary flex size-8 items-center justify-center rounded-lg'>
            <Ticket className='size-4' aria-hidden='true' />
          </span>
          {t('Pending tickets')}
        </CardTitle>
      </CardHeader>
      <CardContent className='flex flex-wrap items-center justify-between gap-4'>
        {summary.error ? (
          <>
            <p role='status'>{t('Failed to load pending tickets')}</p>
            <Button variant='outline' onClick={summary.refresh}>
              <RefreshCw aria-hidden='true' />
              {t('Retry')}
            </Button>
          </>
        ) : (
          <>
            <div>
              <strong
                className='text-primary text-3xl tabular-nums'
                aria-label={t('Pending ticket count')}
              >
                {summary.total === null ? '—' : summary.total.toLocaleString()}
              </strong>
              <p className='text-muted-foreground mt-2 text-sm'>
                {summary.total === 0
                  ? t('No tickets awaiting your response')
                  : t('New tickets and customer follow-ups')}
              </p>
            </div>
            <div className='flex flex-wrap items-center gap-2'>
              <Button variant='outline' size='sm' onClick={summary.refresh}>
                <RefreshCw aria-hidden='true' />
                {t('Refresh')}
              </Button>
              <Link
                to='/ticket-management'
                search={{ status: 'open' }}
                className='border-primary/20 bg-primary/8 text-primary hover:bg-primary/14 focus-visible:ring-ring inline-flex h-7 items-center gap-1.5 rounded-md border px-2.5 text-[0.8rem] font-medium whitespace-nowrap transition-colors outline-none focus-visible:ring-3'
              >
                <ListChecks className='size-3.5' aria-hidden='true' />
                {t('View pending tickets')}
                <ArrowRight className='size-3.5' aria-hidden='true' />
              </Link>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
