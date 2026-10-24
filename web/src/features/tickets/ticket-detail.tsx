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
import { zodResolver } from '@hookform/resolvers/zod'
import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query'
import type { RefObject } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { hasPermission } from '@/lib/admin-permissions'
import { formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { useAuthStore } from '@/stores/auth-store'

import { getTicket, replyToTicket, updateTicketStatus } from './api'
import { getTicketErrorKey } from './errors'
import { getTicketLabels } from './labels'
import { ticketReplySchema, type TicketReplyInput } from './schema'
import type { TicketStatus } from './types'

interface TicketDetailProps {
  id: number
  admin: boolean
  onClose: () => void
  returnFocus?: RefObject<HTMLButtonElement | null>
}

export function TicketDetailDialog(props: TicketDetailProps) {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const queryClient = useQueryClient()
  const labels = getTicketLabels(t)
  const canWrite = !props.admin || hasPermission(user, 'tickets', 'write')
  const form = useForm<TicketReplyInput>({
    resolver: zodResolver(ticketReplySchema),
    defaultValues: { content: '' },
  })
  const query = useInfiniteQuery({
    queryKey: ['tickets', user?.id, props.admin, 'detail', props.id],
    queryFn: ({ pageParam, signal }) =>
      getTicket(props.admin, props.id, pageParam, signal),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (page) =>
      page.has_more ? page.messages[0]?.id : undefined,
  })
  const ticket = query.data?.pages[0]?.ticket
  const isInactive =
    ticket?.status === 'closed' || ticket?.status === 'archived'
  const messages = [...(query.data?.pages ?? [])]
    .reverse()
    .flatMap((page) => page.messages)
  const seenIds = new Set<number>()
  const uniqueMessages = messages.filter((message) => {
    if (seenIds.has(message.id)) return false
    seenIds.add(message.id)
    return true
  })
  const replyMutation = useMutation({
    mutationFn: (input: TicketReplyInput) =>
      replyToTicket(props.admin, props.id, input),
    onMutate: () => useAuthStore.getState().auth.user?.id,
    onSuccess: (_message, _input, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      form.reset()
      void queryClient.invalidateQueries({
        queryKey: ['tickets', submittedUserId],
      })
      toast.success(t('Reply sent.'))
    },
    onError: (error, _input, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      handleServerError(error)
    },
  })
  const statusMutation = useMutation({
    mutationFn: (status: TicketStatus) =>
      updateTicketStatus(props.admin, props.id, status),
    onMutate: () => useAuthStore.getState().auth.user?.id,
    onSuccess: (_ticket, _status, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      void queryClient.invalidateQueries({
        queryKey: ['tickets', submittedUserId],
      })
      toast.success(t('Ticket status updated.'))
    },
    onError: (error, _status, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      handleServerError(error)
    },
  })
  const busy = replyMutation.isPending || statusMutation.isPending
  const disabled = busy || !canWrite || query.isError || query.isFetching

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) props.onClose()
      }}
    >
      <DialogContent
        className='max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] grid-cols-1 overflow-y-auto sm:max-w-3xl'
        showCloseButton={!busy}
        finalFocus={props.returnFocus}
      >
        <DialogHeader>
          <DialogTitle className='pr-6 [overflow-wrap:anywhere] break-words'>
            {ticket ? `#${ticket.id} ${ticket.title}` : t('Ticket details')}
          </DialogTitle>
          <DialogDescription>
            {t('View the conversation and track ticket progress.')}
          </DialogDescription>
        </DialogHeader>
        {query.isPending && (
          <p role='status' className='bg-muted/30 rounded-lg border p-4'>
            {t('Loading tickets...')}
          </p>
        )}
        {query.isError && (
          <Alert variant='destructive'>
            <AlertDescription className='flex flex-wrap items-center gap-2'>
              {t(getTicketErrorKey(query.error, 'Could not load the ticket.'))}
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
        {ticket && (
          <>
            <section
              aria-label={t('Ticket information')}
              className='bg-muted/20 space-y-3 rounded-lg border p-4'
            >
              <div className='flex flex-wrap gap-2'>
                <Badge variant='outline'>{labels.status[ticket.status]}</Badge>
                <Badge variant='outline'>
                  {labels.category[ticket.category]}
                </Badge>
                <Badge variant='outline'>
                  {labels.priority[ticket.priority]}
                </Badge>
                {props.admin && (
                  <span className='text-sm'>
                    {t('User ID')}: {ticket.user_id}
                  </span>
                )}
              </div>
              {props.admin ? (
                <div className='flex flex-wrap items-center gap-2'>
                  <Label htmlFor='ticket-status'>{t('Ticket status')}</Label>
                  <NativeSelect
                    id='ticket-status'
                    value={ticket.status}
                    disabled={disabled}
                    onChange={(event) =>
                      statusMutation.mutate(event.target.value as TicketStatus)
                    }
                  >
                    {Object.entries(labels.status).map(([value, label]) => (
                      <NativeSelectOption key={value} value={value}>
                        {label}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </div>
              ) : (
                <Button
                  variant='outline'
                  disabled={disabled}
                  onClick={() =>
                    statusMutation.mutate(isInactive ? 'open' : 'closed')
                  }
                >
                  {isInactive ? t('Reopen ticket') : t('Close ticket')}
                </Button>
              )}
              {!canWrite && (
                <p className='text-muted-foreground text-sm'>
                  {t('You have read-only access to tickets.')}
                </p>
              )}
              {statusMutation.isError && (
                <p role='alert' className='text-destructive text-sm'>
                  {t(
                    getTicketErrorKey(
                      statusMutation.error,
                      'Could not update ticket status. Try again.'
                    )
                  )}
                </p>
              )}
            </section>
            <section
              aria-label={t('Ticket conversation')}
              className='bg-card space-y-3 rounded-lg border p-4'
            >
              <Button
                variant='outline'
                disabled={busy || query.isFetching}
                onClick={() => void query.refetch()}
              >
                {t('Refresh')}
              </Button>
              {query.hasNextPage && (
                <Button
                  variant='outline'
                  disabled={query.isFetching}
                  onClick={() => void query.fetchNextPage()}
                >
                  {query.isFetchingNextPage
                    ? t('Loading...')
                    : t('Load earlier messages')}
                </Button>
              )}
              <ol className='space-y-3' aria-live='polite'>
                {uniqueMessages.map((message) => (
                  <li
                    key={message.id}
                    className='bg-muted/20 min-w-0 rounded-lg border p-3'
                  >
                    <div className='text-muted-foreground mb-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
                      <span className='font-medium'>
                        {message.author_role === 'admin'
                          ? t('Support team')
                          : t('Requester')}{' '}
                        #{message.author_id}
                      </span>
                      <time
                        dateTime={new Date(
                          message.created_at * 1000
                        ).toISOString()}
                      >
                        {formatTimestampToDate(message.created_at)}
                      </time>
                    </div>
                    <p className='[overflow-wrap:anywhere] break-words whitespace-pre-wrap'>
                      {message.content}
                    </p>
                  </li>
                ))}
              </ol>
            </section>
            {isInactive ? (
              <p
                role='status'
                className='bg-muted/30 rounded-lg border p-4 text-sm'
              >
                {ticket.status === 'archived'
                  ? t('This ticket is archived. Reopen it before replying.')
                  : t('This ticket is closed. Reopen it before replying.')}
              </p>
            ) : (
              <Form {...form}>
                <form
                  className='bg-card space-y-3 rounded-lg border p-4'
                  onSubmit={form.handleSubmit((values) =>
                    replyMutation.mutate(values)
                  )}
                >
                  <FormField
                    control={form.control}
                    name='content'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Reply')}</FormLabel>
                        <FormControl>
                          <Textarea
                            {...field}
                            className='min-h-28 resize-y'
                            disabled={disabled}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  {replyMutation.isError && (
                    <p role='alert' className='text-destructive text-sm'>
                      {t(
                        getTicketErrorKey(
                          replyMutation.error,
                          'Could not send the reply. Your draft is preserved. Try again.'
                        )
                      )}
                    </p>
                  )}
                  <Button type='submit' disabled={disabled}>
                    {replyMutation.isPending
                      ? t('Sending...')
                      : t('Send reply')}
                  </Button>
                </form>
              </Form>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
