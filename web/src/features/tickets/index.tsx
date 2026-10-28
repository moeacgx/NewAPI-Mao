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
import { Plus, RefreshCw } from 'lucide-react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout/components/section-page-layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { hasPermission } from '@/lib/admin-permissions'
import { formatTimestampToDate } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { getTickets } from './api'
import { CreateTicketDialog } from './create-ticket-dialog'
import { getTicketErrorKey } from './errors'
import { getTicketLabels } from './labels'
import { TicketAutoArchiveSettings } from './ticket-auto-archive-settings'
import { TicketAutoReplySettings } from './ticket-auto-reply-settings'
import { TicketBatchActions } from './ticket-batch-actions'
import { TicketDetailDialog } from './ticket-detail'
import type { TicketFilters, TicketStatus, TicketVersion } from './types'

export function Tickets(props: {
  admin?: boolean
  status?: TicketStatus | ''
  onStatusChange?: (status: TicketStatus | '') => void
}) {
  const { t } = useTranslation()
  const admin = props.admin ?? false
  const user = useAuthStore((state) => state.auth.user)
  const canRead = Boolean(
    user &&
    (!admin ||
      (user.role >= ROLE.ADMIN && hasPermission(user, 'tickets', 'read')))
  )
  const canWrite = Boolean(
    user &&
    (!admin ||
      (user.role >= ROLE.ADMIN && hasPermission(user, 'tickets', 'write')))
  )
  const [filters, setFilters] = useState<TicketFilters>({
    p: 1,
    page_size: 20,
    status: props.status,
  })
  const [routeStatus, setRouteStatus] = useState(props.status)
  if (routeStatus !== props.status) {
    setRouteStatus(props.status)
    setFilters((current) => ({ ...current, p: 1, status: props.status }))
  }
  const [keyword, setKeyword] = useState('')
  const [userId, setUserId] = useState('')
  const [creating, setCreating] = useState(false)
  const createTriggerRef = useRef<HTMLButtonElement>(null)
  const detailTriggerRef = useRef<HTMLButtonElement>(null)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const selectionScope = JSON.stringify([user?.id, admin, canWrite, filters])
  const [selection, setSelection] = useState<{
    scope: string
    tickets: TicketVersion[]
  }>({ scope: '', tickets: [] })
  const selectedTickets =
    selection.scope === selectionScope ? selection.tickets : []
  if (selection.scope !== selectionScope) {
    setSelection({ scope: selectionScope, tickets: [] })
  }
  const clearSelection = () =>
    setSelection((current) =>
      current.scope === selectionScope
        ? { scope: selectionScope, tickets: [] }
        : current
    )
  const labels = getTicketLabels(t)
  const query = useQuery({
    queryKey: ['tickets', user?.id, admin, 'list', filters],
    queryFn: ({ signal }) => getTickets(admin, filters, signal),
    enabled: canRead,
  })
  const totalPages = Math.max(
    1,
    Math.ceil((query.data?.total ?? 0) / filters.page_size)
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {admin ? t('Ticket Management') : t('Ticket Center')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        {canRead && (
          <Button
            variant='outline'
            disabled={query.isFetching}
            onClick={() => {
              clearSelection()
              void query.refetch()
            }}
          >
            <RefreshCw aria-hidden='true' />
            {t('Refresh')}
          </Button>
        )}
        {!admin && user && (
          <Button ref={createTriggerRef} onClick={() => setCreating(true)}>
            <Plus aria-hidden='true' />
            {t('Create ticket')}
          </Button>
        )}
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {!canRead ? (
          <Alert variant='destructive'>
            <AlertDescription>
              {t('You do not have permission to view tickets.')}
            </AlertDescription>
          </Alert>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>
                {admin ? t('All tickets') : t('My tickets')}
              </CardTitle>
              <CardDescription>
                {admin
                  ? t('Review requests and reply to users.')
                  : t(
                      'Submit requests and follow replies from the support team.'
                    )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <form
                className='bg-muted/20 flex flex-wrap items-end gap-3 rounded-lg border p-3'
                onSubmit={(event) => {
                  event.preventDefault()
                  setFilters((current) => ({
                    ...current,
                    p: 1,
                    keyword: keyword.trim() || undefined,
                    user_id: admin && userId ? Number(userId) : undefined,
                  }))
                }}
              >
                <div className='min-w-40 flex-1 space-y-2'>
                  <Label htmlFor='ticket-search'>{t('Search tickets')}</Label>
                  <Input
                    id='ticket-search'
                    value={keyword}
                    onChange={(event) => setKeyword(event.target.value)}
                  />
                </div>
                {admin && (
                  <div className='w-32 space-y-2'>
                    <Label htmlFor='ticket-user'>{t('User ID')}</Label>
                    <Input
                      id='ticket-user'
                      type='number'
                      min={1}
                      step={1}
                      value={userId}
                      onChange={(event) => setUserId(event.target.value)}
                    />
                  </div>
                )}
                <div className='space-y-2'>
                  <Label htmlFor='ticket-filter-status'>{t('Status')}</Label>
                  <NativeSelect
                    id='ticket-filter-status'
                    value={filters.status ?? ''}
                    onChange={(event) => {
                      const status = event.target.value as TicketStatus | ''
                      props.onStatusChange?.(status)
                      setFilters((current) => ({
                        ...current,
                        p: 1,
                        status,
                      }))
                    }}
                  >
                    <NativeSelectOption value=''>
                      {t('All statuses')}
                    </NativeSelectOption>
                    {Object.entries(labels.status).map(([value, label]) => (
                      <NativeSelectOption key={value} value={value}>
                        {label}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </div>
                <Button type='submit'>{t('Search')}</Button>
              </form>
              {query.isPending && (
                <p
                  role='status'
                  className='bg-muted/20 rounded-lg border p-6 text-center'
                >
                  {t('Loading tickets...')}
                </p>
              )}
              {query.isError && (
                <Alert variant='destructive'>
                  <AlertDescription className='flex flex-wrap items-center gap-2'>
                    {t(
                      getTicketErrorKey(query.error, 'Could not load tickets.')
                    )}
                    <Button
                      variant='outline'
                      disabled={query.isFetching}
                      onClick={() => void query.refetch()}
                    >
                      {t('Retry')}
                    </Button>
                  </AlertDescription>
                </Alert>
              )}
              {query.isSuccess && query.data.items.length === 0 && (
                <p
                  role='status'
                  className='bg-muted/20 rounded-lg border p-6 text-center'
                >
                  {t('No tickets found.')}
                </p>
              )}
              {query.isSuccess && query.data.items.length > 0 && (
                <>
                  {admin && canWrite && (
                    <TicketBatchActions
                      key={selectionScope}
                      selected={selectedTickets}
                      onComplete={clearSelection}
                      disabled={query.isFetching}
                    />
                  )}
                  <div className='overflow-hidden rounded-lg border'>
                    <Table
                      aria-label={admin ? t('All tickets') : t('My tickets')}
                    >
                      <TableHeader>
                        <TableRow>
                          {admin && canWrite && (
                            <TableHead className='w-10'>
                              <Checkbox
                                aria-label={t(
                                  'Select all tickets on this page'
                                )}
                                disabled={query.isFetching}
                                checked={
                                  selectedTickets.length ===
                                  query.data.items.length
                                }
                                indeterminate={
                                  selectedTickets.length > 0 &&
                                  selectedTickets.length <
                                    query.data.items.length
                                }
                                onCheckedChange={(checked) =>
                                  setSelection({
                                    scope: selectionScope,
                                    tickets: checked
                                      ? query.data.items.map(
                                          ({ id, version }) => ({ id, version })
                                        )
                                      : [],
                                  })
                                }
                              />
                            </TableHead>
                          )}
                          <TableHead>{t('Ticket title')}</TableHead>
                          {admin && <TableHead>{t('User ID')}</TableHead>}
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Category')}</TableHead>
                          <TableHead>{t('Priority')}</TableHead>
                          <TableHead>{t('Updated at')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {query.data.items.map((ticket) => (
                          <TableRow key={ticket.id}>
                            {admin && canWrite && (
                              <TableCell>
                                <Checkbox
                                  aria-label={t('Select ticket #{{id}}', {
                                    id: ticket.id,
                                  })}
                                  disabled={query.isFetching}
                                  checked={selectedTickets.some(
                                    (selected) => selected.id === ticket.id
                                  )}
                                  onCheckedChange={(checked) =>
                                    setSelection({
                                      scope: selectionScope,
                                      tickets: checked
                                        ? [
                                            ...selectedTickets,
                                            {
                                              id: ticket.id,
                                              version: ticket.version,
                                            },
                                          ]
                                        : selectedTickets.filter(
                                            (selected) =>
                                              selected.id !== ticket.id
                                          ),
                                    })
                                  }
                                />
                              </TableCell>
                            )}
                            <TableCell className='max-w-80 min-w-48 whitespace-normal'>
                              <button
                                type='button'
                                className='text-primary block max-w-72 text-left font-medium [overflow-wrap:anywhere] break-words whitespace-normal underline-offset-4 hover:underline focus-visible:underline'
                                onClick={(event) => {
                                  detailTriggerRef.current = event.currentTarget
                                  setSelectedId(ticket.id)
                                }}
                              >
                                #{ticket.id} {ticket.title}
                              </button>
                            </TableCell>
                            {admin && <TableCell>{ticket.user_id}</TableCell>}
                            <TableCell>
                              <Badge variant='outline'>
                                {labels.status[ticket.status]}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              {labels.category[ticket.category]}
                            </TableCell>
                            <TableCell>
                              {labels.priority[ticket.priority]}
                            </TableCell>
                            <TableCell>
                              {formatTimestampToDate(ticket.updated_at)}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </>
              )}
              <div className='flex flex-wrap items-center justify-between gap-3 border-t pt-4'>
                <p aria-live='polite' className='text-muted-foreground text-sm'>
                  {t('Page {{page}} of {{pages}} · {{count}} tickets', {
                    page: filters.p,
                    pages: totalPages,
                    count: query.data?.total ?? 0,
                  })}
                </p>
                <div className='flex gap-2'>
                  <Button
                    variant='outline'
                    disabled={filters.p <= 1 || query.isFetching}
                    onClick={() =>
                      setFilters((current) => ({
                        ...current,
                        p: current.p - 1,
                      }))
                    }
                  >
                    {t('Previous')}
                  </Button>
                  <Button
                    variant='outline'
                    disabled={
                      !query.isSuccess ||
                      filters.p >= totalPages ||
                      query.isFetching
                    }
                    onClick={() =>
                      setFilters((current) => ({
                        ...current,
                        p: current.p + 1,
                      }))
                    }
                  >
                    {t('Next')}
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
        {admin && canRead && (
          <details
            key={`settings-${user?.id}`}
            className='text-card-foreground mt-4 min-w-0'
          >
            <summary className='focus-visible:ring-ring cursor-pointer rounded-sm font-medium focus-visible:ring-2 focus-visible:outline-none'>
              {t('Ticket settings')}
            </summary>
            <div className='space-y-4 pt-2'>
              <TicketAutoReplySettings
                key={`reply-${user?.id}`}
                canWrite={canWrite}
              />
              <TicketAutoArchiveSettings
                key={`archive-${user?.id}`}
                canWrite={canWrite}
              />
            </div>
          </details>
        )}
        {!admin && user && (
          <CreateTicketDialog
            key={user.id}
            open={creating}
            onOpenChange={setCreating}
            returnFocus={createTriggerRef}
          />
        )}
        {canRead && selectedId !== null && (
          <TicketDetailDialog
            key={`${user?.id}-${admin}-${selectedId}`}
            id={selectedId}
            admin={admin}
            onClose={() => setSelectedId(null)}
            returnFocus={detailTriggerRef}
          />
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
