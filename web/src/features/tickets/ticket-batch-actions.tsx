/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { useAuthStore } from '@/stores/auth-store'

import { batchDeleteTickets, batchUpdateTicketStatus } from './api'
import { getTicketErrorKey } from './errors'
import { getTicketLabels } from './labels'
import type { TicketStatus, TicketVersion } from './types'

export function TicketBatchActions(props: {
  selected: TicketVersion[]
  onComplete: () => void
  disabled: boolean
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const labels = getTicketLabels(t)
  const [status, setStatus] = useState<TicketStatus>('processing')
  const [action, setAction] = useState<'status' | 'delete' | null>(null)
  const mutation = useMutation({
    mutationFn: async (input: {
      action: 'status' | 'delete'
      tickets: TicketVersion[]
      status: TicketStatus
    }) => {
      if (input.action === 'delete') return batchDeleteTickets(input.tickets)
      return batchUpdateTicketStatus({
        tickets: input.tickets,
        status: input.status,
      })
    },
    onMutate: () => useAuthStore.getState().auth.user?.id,
    onSuccess: (_result, input, userId) => {
      if (
        userId === undefined ||
        useAuthStore.getState().auth.user?.id !== userId
      ) {
        return
      }
      setAction(null)
      props.onComplete()
      void queryClient.invalidateQueries({ queryKey: ['tickets', userId] })
      toast.success(
        input.action === 'delete'
          ? t('Selected tickets deleted.')
          : t('Selected ticket statuses updated.')
      )
    },
    onError: (error, _input, userId) => {
      if (
        userId === undefined ||
        useAuthStore.getState().auth.user?.id !== userId
      ) {
        return
      }
      toast.error(
        t(getTicketErrorKey(error, 'Could not complete the batch operation.'))
      )
    },
  })
  const disabled =
    props.disabled || mutation.isPending || props.selected.length === 0
  const confirmLabel = action === 'delete' ? t('Confirm delete') : t('Confirm')

  return (
    <>
      <div className='bg-muted/20 flex min-w-0 flex-wrap items-end gap-3 rounded-lg border p-3'>
        <p role='status' className='w-full text-sm'>
          {t('Selected tickets: {{count}}', { count: props.selected.length })}
        </p>
        <div className='min-w-0 space-y-2'>
          <Label htmlFor='ticket-batch-status'>
            {t('Batch ticket status')}
          </Label>
          <NativeSelect
            id='ticket-batch-status'
            value={status}
            disabled={disabled}
            onChange={(event) => setStatus(event.target.value as TicketStatus)}
          >
            {Object.entries(labels.status).map(([value, label]) => (
              <NativeSelectOption key={value} value={value}>
                {label}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
        <Button
          variant='outline'
          className='h-auto min-h-8 max-w-full py-1.5 whitespace-normal'
          disabled={disabled}
          onClick={() => {
            mutation.reset()
            setAction('status')
          }}
        >
          {t('Change selected status')}
        </Button>
        <Button
          variant='destructive'
          className='h-auto min-h-8 max-w-full py-1.5 whitespace-normal'
          disabled={disabled}
          onClick={() => {
            mutation.reset()
            setAction('delete')
          }}
        >
          {t('Delete selected tickets')}
        </Button>
      </div>
      <AlertDialog
        open={action !== null}
        onOpenChange={(open) => {
          if (!open && !mutation.isPending) setAction(null)
        }}
      >
        <AlertDialogContent className='max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] grid-cols-1 overflow-y-auto'>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {action === 'delete'
                ? t('Delete selected tickets')
                : t('Change selected status')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {action === 'delete'
                ? t(
                    'Deleted tickets disappear from both users and administrators. Conversation and audit records are retained; there is no restore action.'
                  )
                : t('Update {{count}} selected tickets to {{status}}?', {
                    count: props.selected.length,
                    status: labels.status[status],
                  })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {action === 'delete' && (
            <p className='text-sm'>
              {t('Selected tickets: {{count}}', {
                count: props.selected.length,
              })}
            </p>
          )}
          {mutation.isError && (
            <Alert variant='destructive'>
              <AlertDescription>
                <p>
                  {t(
                    getTicketErrorKey(
                      mutation.error,
                      'Could not complete the batch operation.'
                    )
                  )}
                </p>
                <p>
                  {t(
                    'Refresh the list and select tickets again before retrying.'
                  )}
                </p>
              </AlertDescription>
            </Alert>
          )}
          <AlertDialogFooter>
            <Button
              variant='outline'
              disabled={mutation.isPending}
              onClick={() => setAction(null)}
            >
              {t('Cancel')}
            </Button>
            <Button
              variant={action === 'delete' ? 'destructive' : 'default'}
              disabled={disabled}
              onClick={() => {
                if (action) {
                  mutation.mutate({ action, tickets: props.selected, status })
                }
              }}
            >
              {mutation.isPending ? t('Saving...') : confirmLabel}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
