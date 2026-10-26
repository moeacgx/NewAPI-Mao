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
import { useQuery } from '@tanstack/react-query'
import { Link, useLocation } from '@tanstack/react-router'
import { Ticket, X } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { hasPermission } from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { getPendingTicketSummary } from './pending-api'
import { PendingTicketContext } from './pending-ticket-context'

function PendingTicketSession(props: {
  userId: number
  identity: string
  children: ReactNode
}) {
  const { t } = useTranslation()
  const pathname = useLocation({ select: (location) => location.pathname })
  const [dismissed, setDismissed] = useState(false)
  const query = useQuery({
    queryKey: ['tickets', props.userId, 'pending', props.identity],
    queryFn: ({ signal }) => getPendingTicketSummary(props.userId, signal),
    gcTime: 0,
    staleTime: 0,
    retry: false,
    refetchOnMount: 'always',
    refetchOnWindowFocus: 'always',
    refetchInterval: 60000,
    refetchIntervalInBackground: false,
  })
  const refetch = query.refetch
  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState !== 'hidden') void refetch()
    }
    window.addEventListener('focus', refresh)
    return () => window.removeEventListener('focus', refresh)
  }, [refetch])
  const summary = query.data
  const counts = summary?.state === 'ready' ? summary.counts : null
  const total = counts?.open ?? null
  const context =
    summary?.state === 'denied'
      ? null
      : {
          total,
          counts,
          error: summary?.state === 'error',
          refresh: () => {
            void refetch()
          },
        }
  return (
    <PendingTicketContext.Provider value={context}>
      {props.children}
      {total !== null &&
        total > 0 &&
        !dismissed &&
        !pathname.startsWith('/ticket-management') && (
          <aside
            aria-label={t('Pending ticket reminder')}
            className='bg-card text-card-foreground fixed top-20 right-3 z-40 w-80 max-w-[calc(100vw-1.5rem)] rounded-xl border p-4 shadow-lg sm:right-5'
          >
            <div className='flex items-center gap-2'>
              <Ticket className='size-4' aria-hidden='true' />
              <strong className='min-w-0 flex-1 break-words'>
                {t('Pending tickets')}
              </strong>
              <Button
                variant='ghost'
                size='icon'
                aria-label={t('Dismiss ticket reminder')}
                onClick={() => setDismissed(true)}
              >
                <X aria-hidden='true' />
              </Button>
            </div>
            <p role='status' className='my-3 text-sm'>
              {t('Tickets awaiting your response: {{total}}', { total })}
            </p>
            <Link
              to='/ticket-management'
              search={{ status: 'open' }}
              className='text-primary text-sm font-medium underline underline-offset-4'
            >
              {t('View pending tickets')}
            </Link>
          </aside>
        )}
    </PendingTicketContext.Provider>
  )
}

export function PendingTicketProvider(props: { children: ReactNode }) {
  const user = useAuthStore((state) => state.auth.user)
  const sessionId = useAuthStore((state) => state.auth.session?.sid)
  if (
    !user ||
    user.role < ROLE.ADMIN ||
    !hasPermission(user, 'tickets', 'read')
  ) {
    return (
      <PendingTicketContext.Provider value={null}>
        {props.children}
      </PendingTicketContext.Provider>
    )
  }
  const identity = `${user.id}:${user.role}:${sessionId ?? ''}`
  return (
    <PendingTicketSession key={identity} userId={user.id} identity={identity}>
      {props.children}
    </PendingTicketSession>
  )
}
