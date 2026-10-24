/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
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
import { Switch } from '@/components/ui/switch'
import { useAuthStore } from '@/stores/auth-store'

import {
  getTicketAutoArchiveConfig,
  updateTicketAutoArchiveConfig,
} from './api'
import { getTicketLabels } from './labels'
import type { TicketAutoArchiveConfig } from './types'

type ArchiveDraft = Omit<TicketAutoArchiveConfig, 'inactive_days'> & {
  inactive_days: string
}
const eligibleStatuses: TicketAutoArchiveConfig['statuses'] = [
  'open',
  'processing',
  'resolved',
  'closed',
]

export function TicketAutoArchiveSettings(props: { canWrite: boolean }) {
  const { t } = useTranslation()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<{
    userId: number | undefined
    config: ArchiveDraft
  }>()
  const [validationError, setValidationError] = useState('')
  const labels = getTicketLabels(t)
  const query = useQuery({
    queryKey: ['ticket-auto-archive-config', userId],
    queryFn: ({ signal }) => getTicketAutoArchiveConfig(signal),
  })
  const saved = query.data ?? {
    enabled: false,
    inactive_days: 30,
    statuses: ['resolved', 'closed'] as TicketAutoArchiveConfig['statuses'],
  }
  const config =
    draft?.userId === userId && draft
      ? draft.config
      : { ...saved, inactive_days: String(saved.inactive_days) }
  const dirty =
    config.enabled !== saved.enabled ||
    config.inactive_days !== String(saved.inactive_days) ||
    config.statuses.join(',') !== saved.statuses.join(',')
  const mutation = useMutation({
    mutationFn: updateTicketAutoArchiveConfig,
    onMutate: () => userId,
    onSuccess: (next, _input, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      queryClient.setQueryData(
        ['ticket-auto-archive-config', submittedUserId],
        next
      )
      setDraft(undefined)
      setValidationError('')
      toast.success(t('Automatic archive settings saved.'))
    },
    onError: (_error, _input, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      toast.error(t('Could not save automatic archive settings.'))
    },
  })
  const disabled = !props.canWrite || mutation.isPending
  const update = (changes: Partial<ArchiveDraft>) => {
    setDraft({ userId, config: { ...config, ...changes } })
    setValidationError('')
    mutation.reset()
  }
  const save = () => {
    const days = Number(config.inactive_days)
    if (!Number.isInteger(days) || days < 1 || days > 3650) {
      setValidationError(t('Enter a whole number of days from 1 to 3650.'))
      return
    }
    if (config.statuses.length === 0) {
      setValidationError(t('Select at least one ticket status.'))
      return
    }
    setValidationError('')
    mutation.mutate({ ...config, inactive_days: days })
  }

  return (
    <Card className='min-w-0'>
      <CardHeader>
        <CardTitle>{t('Automatic ticket archive')}</CardTitle>
        <CardDescription>
          {t(
            'Archive tickets after no replies or status changes for the configured number of days. Existing tickets are included when enabled.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        {query.isPending && (
          <p role='status'>{t('Loading automatic archive settings...')}</p>
        )}
        {query.isError && (
          <Alert variant='destructive'>
            <AlertDescription>
              {t('Could not load automatic archive settings.')}
              <Button variant='outline' onClick={() => void query.refetch()}>
                {t('Retry')}
              </Button>
            </AlertDescription>
          </Alert>
        )}
        {query.isSuccess && (
          <>
            <div className='bg-muted/20 flex items-center justify-between gap-4 rounded-lg border p-3'>
              <Label htmlFor='ticket-auto-archive-enabled'>
                {t('Enable automatic ticket archive')}
              </Label>
              <Switch
                id='ticket-auto-archive-enabled'
                checked={config.enabled}
                disabled={disabled}
                onCheckedChange={(enabled) => update({ enabled })}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='ticket-auto-archive-days'>
                {t('Inactive days')}
              </Label>
              <Input
                id='ticket-auto-archive-days'
                type='number'
                min={1}
                max={3650}
                step={1}
                className='max-w-48'
                value={config.inactive_days}
                disabled={disabled}
                onChange={(event) =>
                  update({ inactive_days: event.target.value })
                }
              />
            </div>
            <fieldset
              className='min-w-0 space-y-3 rounded-lg border p-3'
              disabled={disabled}
            >
              <legend className='px-1 text-sm font-medium'>
                {t('Statuses eligible for automatic archive')}
              </legend>
              <div className='flex flex-wrap gap-x-6 gap-y-3'>
                {eligibleStatuses.map((status) => (
                  <Label key={status} className='flex items-center gap-2'>
                    <Checkbox
                      checked={config.statuses.includes(status)}
                      disabled={disabled}
                      onCheckedChange={(checked) =>
                        update({
                          statuses: eligibleStatuses.filter((value) =>
                            value === status
                              ? checked
                              : config.statuses.includes(value)
                          ),
                        })
                      }
                    />
                    {labels.status[status]}
                  </Label>
                ))}
              </div>
            </fieldset>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Archived tickets remain visible and can be reopened. Select open or in-progress tickets only if unattended requests should also be archived.'
              )}
            </p>
            {validationError && (
              <p role='alert' className='text-destructive text-sm'>
                {validationError}
              </p>
            )}
            {mutation.isError && (
              <Alert variant='destructive'>
                <AlertDescription>
                  {t('Could not save automatic archive settings.')}
                </AlertDescription>
              </Alert>
            )}
            {!props.canWrite && (
              <p className='text-muted-foreground text-sm'>
                {t('You have read-only access to ticket automation settings.')}
              </p>
            )}
            <div className='flex justify-end border-t pt-4'>
              <Button
                className='h-auto min-h-8 max-w-full py-1.5 whitespace-normal'
                disabled={disabled || !dirty}
                onClick={save}
              >
                {mutation.isPending
                  ? t('Saving...')
                  : t('Save automatic archive')}
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
