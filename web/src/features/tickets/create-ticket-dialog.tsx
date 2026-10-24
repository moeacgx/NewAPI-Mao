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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { RefObject } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
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
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'
import { useAuthStore } from '@/stores/auth-store'

import { createTicket } from './api'
import { getTicketErrorKey } from './errors'
import { getTicketLabels } from './labels'
import { createTicketSchema, type CreateTicketInput } from './schema'

interface CreateTicketDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  returnFocus?: RefObject<HTMLButtonElement | null>
}

export function CreateTicketDialog(props: CreateTicketDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const labels = getTicketLabels(t)
  const form = useForm<CreateTicketInput>({
    resolver: zodResolver(createTicketSchema),
    defaultValues: {
      title: '',
      category: 'general',
      priority: 'normal',
      content: '',
    },
  })
  const mutation = useMutation({
    mutationFn: createTicket,
    // 卸载不会取消 mutation 回调；用提交身份隔离切换账号后的界面。
    onMutate: () => useAuthStore.getState().auth.user?.id,
    onSuccess: (_ticket, _input, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      void queryClient.invalidateQueries({
        queryKey: ['tickets', submittedUserId],
      })
      form.reset()
      toast.success(t('Ticket created.'))
      props.onOpenChange(false)
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

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (!mutation.isPending) props.onOpenChange(open)
      }}
    >
      <DialogContent
        className='max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-xl'
        showCloseButton={!mutation.isPending}
        finalFocus={props.returnFocus}
      >
        <DialogHeader>
          <DialogTitle>{t('Create ticket')}</DialogTitle>
          <DialogDescription>
            {t('Describe your issue. Do not include passwords or API keys.')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            className='space-y-4'
            onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
          >
            <FormField
              control={form.control}
              name='title'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Ticket title')}</FormLabel>
                  <FormControl>
                    <Input {...field} autoFocus disabled={mutation.isPending} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='category'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Category')}</FormLabel>
                    <FormControl>
                      <NativeSelect
                        {...field}
                        className='w-full'
                        disabled={mutation.isPending}
                      >
                        {Object.entries(labels.category).map(
                          ([value, label]) => (
                            <NativeSelectOption key={value} value={value}>
                              {label}
                            </NativeSelectOption>
                          )
                        )}
                      </NativeSelect>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='priority'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Priority')}</FormLabel>
                    <FormControl>
                      <NativeSelect
                        {...field}
                        className='w-full'
                        disabled={mutation.isPending}
                      >
                        {Object.entries(labels.priority).map(
                          ([value, label]) => (
                            <NativeSelectOption key={value} value={value}>
                              {label}
                            </NativeSelectOption>
                          )
                        )}
                      </NativeSelect>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='content'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Message')}</FormLabel>
                  <FormControl>
                    <Textarea
                      {...field}
                      className='min-h-36 resize-y'
                      disabled={mutation.isPending}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {mutation.isError && (
              <Alert variant='destructive'>
                <AlertDescription>
                  {t(
                    getTicketErrorKey(
                      mutation.error,
                      'Could not create the ticket. Your draft is preserved. Try again.'
                    )
                  )}
                </AlertDescription>
              </Alert>
            )}
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                disabled={mutation.isPending}
                onClick={() => props.onOpenChange(false)}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={mutation.isPending}>
                {mutation.isPending ? t('Submitting...') : t('Submit ticket')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
