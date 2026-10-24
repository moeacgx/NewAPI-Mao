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
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth-store'

import {
  getTicketAutoReplyConfig,
  updateTicketAutoReplyConfig,
  type TicketAutoReplyConfig,
} from './api'

const DEFAULT_AUTO_REPLY_CONTENT = '您好，已收到您的工单，我们会尽快处理。'
const MAX_CONTENT_LENGTH = 10000

export function TicketAutoReplySettings(props: { canWrite: boolean }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const [draft, setDraft] = useState<{
    userId: number | undefined
    config: TicketAutoReplyConfig
  }>()
  const [validationError, setValidationError] = useState('')
  const query = useQuery({
    queryKey: ['ticket-auto-reply-config', userId],
    queryFn: ({ signal }) => getTicketAutoReplyConfig(signal),
  })

  const savedConfig = query.data ?? {
    enabled: false,
    content: DEFAULT_AUTO_REPLY_CONTENT,
  }
  const config = draft?.userId === userId && draft ? draft.config : savedConfig
  const { enabled, content } = config

  const saveMutation = useMutation({
    mutationFn: updateTicketAutoReplyConfig,
    onMutate: () => userId,
    onSuccess: (nextConfig, _input, submittedUserId) => {
      if (useAuthStore.getState().auth.user?.id !== submittedUserId) return
      queryClient.setQueryData(
        ['ticket-auto-reply-config', submittedUserId],
        nextConfig
      )
      setDraft(undefined)
      setValidationError('')
      toast.success(t('Automatic ticket reply settings saved.'))
    },
    onError: (_error, _input, submittedUserId) => {
      if (useAuthStore.getState().auth.user?.id !== submittedUserId) return
      toast.error(t('Could not save automatic ticket reply settings.'))
    },
  })

  const normalizedContent = content.trim()
  const isDirty =
    enabled !== savedConfig.enabled || normalizedContent !== savedConfig.content
  const onSave = () => {
    if (enabled && normalizedContent.length === 0) {
      setValidationError(t('Enter an automatic reply message when enabled.'))
      return
    }
    if ([...normalizedContent].length > MAX_CONTENT_LENGTH) {
      setValidationError(
        t('Automatic reply must be 10000 characters or fewer.')
      )
      return
    }
    setValidationError('')
    saveMutation.mutate({ enabled, content: normalizedContent })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Automatic ticket reply')}</CardTitle>
        <CardDescription>
          {t(
            'Send one automatic reply when a user submits a new ticket. It is sent only once per ticket.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        {query.isPending && (
          <p role='status' className='text-muted-foreground text-sm'>
            {t('Loading automatic reply settings...')}
          </p>
        )}
        {query.isError && (
          <Alert variant='destructive'>
            <AlertDescription>
              {t('Could not load automatic reply settings.')}
            </AlertDescription>
          </Alert>
        )}
        {query.isSuccess && (
          <>
            <div className='bg-muted/20 flex items-center justify-between gap-4 rounded-lg border p-3'>
              <div className='space-y-1'>
                <Label htmlFor='ticket-auto-reply-enabled'>
                  {t('Enable automatic ticket reply')}
                </Label>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'New tickets receive the message below after they are created.'
                  )}
                </p>
              </div>
              <Switch
                id='ticket-auto-reply-enabled'
                checked={enabled}
                disabled={!props.canWrite || saveMutation.isPending}
                onCheckedChange={(nextEnabled) =>
                  setDraft({
                    userId,
                    config: { ...config, enabled: nextEnabled },
                  })
                }
                aria-label={t('Enable automatic ticket reply')}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='ticket-auto-reply-content'>
                {t('Automatic reply message')}
              </Label>
              <Textarea
                id='ticket-auto-reply-content'
                rows={5}
                maxLength={MAX_CONTENT_LENGTH}
                value={content}
                disabled={!props.canWrite || saveMutation.isPending}
                onChange={(event) => {
                  setDraft({
                    userId,
                    config: { ...config, content: event.target.value },
                  })
                  setValidationError('')
                }}
                aria-invalid={validationError ? 'true' : undefined}
              />
              <div className='flex flex-wrap justify-between gap-2'>
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'Use plain text. The reply is added to the ticket conversation as a support message.'
                  )}
                </p>
                <span className='text-muted-foreground text-sm'>
                  {`${[...content].length}/${MAX_CONTENT_LENGTH}`}
                </span>
              </div>
              {validationError && (
                <p role='alert' className='text-destructive text-sm'>
                  {validationError}
                </p>
              )}
            </div>
            {!props.canWrite && (
              <p className='text-muted-foreground text-sm'>
                {t('You have read-only access to ticket automation settings.')}
              </p>
            )}
            <div className='flex justify-end border-t pt-4'>
              <Button
                disabled={!props.canWrite || !isDirty || saveMutation.isPending}
                onClick={onSave}
              >
                {saveMutation.isPending
                  ? t('Saving...')
                  : t('Save automatic reply')}
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
