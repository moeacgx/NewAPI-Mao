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
import { Plus, Trash2 } from 'lucide-react'
import { useRef, useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth-store'

import { createSupplierApplication } from './api'
import { getSupplierErrorKey, isSupplierDuplicateError } from './errors'
import { getSupplierLabels } from './labels'
import { ModelTagInput } from './model-tag-input'
import {
  SUPPLIER_MAX_GROUPS,
  supplierApplicationSchema,
  supplierContactTypes,
  supplierPlatforms,
  type SupplierApplicationFormValues,
} from './schema'

const emptyGroup = {
  group_name: '',
  platform: 'openai',
  ratio: '1',
  models: [],
  description: '',
} satisfies SupplierApplicationFormValues['groups'][number]

const defaultValues: SupplierApplicationFormValues = {
  site_name: '',
  site_url: '',
  contact_type: 'wechat',
  contact: '',
  remark: '',
  groups: [emptyGroup],
}

export function SupplierApplicationForm(props: { onSubmitted: () => void }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const labels = getSupplierLabels(t)
  const [serverError, setServerError] = useState('')
  // A ref closes the gap between a second click and the pending re-render.
  const submitting = useRef(false)
  const form = useForm<SupplierApplicationFormValues>({
    resolver: zodResolver(supplierApplicationSchema),
    defaultValues,
  })
  const groups = useFieldArray({ control: form.control, name: 'groups' })
  const mutation = useMutation({
    mutationFn: createSupplierApplication,
    // Mutation callbacks survive unmounting; bind them to the submitting user.
    onMutate: () => useAuthStore.getState().auth.user?.id,
    onSuccess: (_application, _values, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      void queryClient.invalidateQueries({
        queryKey: ['supplier-applications', submittedUserId],
      })
      form.reset(defaultValues)
      setServerError('')
      toast.success(t('Application submitted.'))
      props.onSubmitted()
    },
    onError: (error, _values, submittedUserId) => {
      if (
        submittedUserId === undefined ||
        useAuthStore.getState().auth.user?.id !== submittedUserId
      ) {
        return
      }
      // Entered values stay in the form so the applicant can correct and retry.
      setServerError(
        getSupplierErrorKey(error, 'Could not submit the application.')
      )
      if (isSupplierDuplicateError(error)) {
        void queryClient.invalidateQueries({
          queryKey: ['supplier-applications', submittedUserId],
        })
      }
    },
    onSettled: () => {
      submitting.current = false
    },
  })
  const pending = mutation.isPending

  return (
    <Form {...form}>
      <form
        className='space-y-6'
        noValidate
        onSubmit={form.handleSubmit((values) => {
          if (submitting.current) return
          submitting.current = true
          setServerError('')
          mutation.mutate(values)
        })}
      >
        <fieldset disabled={pending} className='min-w-0 space-y-4'>
          <legend className='text-base font-semibold'>
            {t('Site information')}
          </legend>
          <div className='grid gap-4 md:grid-cols-2'>
            <FormField
              control={form.control}
              name='site_name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Site name')}</FormLabel>
                  <FormControl>
                    <Input {...field} autoComplete='off' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='site_url'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Site URL')}{' '}
                    <span className='text-muted-foreground font-normal'>
                      ({t('optional')})
                    </span>
                  </FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type='url'
                      inputMode='url'
                      placeholder='https://'
                      autoComplete='off'
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Saved as reference only. We never visit this address.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='contact_type'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Contact type')}</FormLabel>
                  <FormControl>
                    <NativeSelect {...field} className='w-full'>
                      {supplierContactTypes.map((value) => (
                        <NativeSelectOption key={value} value={value}>
                          {labels.contactType[value]}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='contact'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Contact')}</FormLabel>
                  <FormControl>
                    <Input {...field} autoComplete='off' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name='remark'
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('Remarks')}{' '}
                  <span className='text-muted-foreground font-normal'>
                    ({t('optional')})
                  </span>
                </FormLabel>
                <FormControl>
                  <Textarea {...field} className='min-h-24 resize-y' />
                </FormControl>
                <FormDescription>
                  {t(
                    'Top-up ratio, settlement method, and any other cooperation terms.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </fieldset>

        <fieldset disabled={pending} className='min-w-0 space-y-4'>
          <div className='flex flex-wrap items-center justify-between gap-2'>
            <legend className='text-base font-semibold'>
              {t('Groups you can supply')}
            </legend>
            <span
              className='text-muted-foreground text-sm tabular-nums'
              aria-live='polite'
            >
              {t('{{count}} / {{max}} groups', {
                count: groups.fields.length,
                max: SUPPLIER_MAX_GROUPS,
              })}
            </span>
          </div>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Enter each ratio exactly as your own site defines it. It is saved as entered and is not converted to this platform billing ratios.'
            )}
          </p>
          {groups.fields.map((item, index) => (
            <section
              key={item.id}
              aria-label={t('Group {{index}}', { index: index + 1 })}
              className='bg-muted/20 min-w-0 space-y-4 rounded-lg border p-4'
            >
              <div className='flex items-center justify-between gap-2'>
                <h3 className='text-sm font-semibold'>
                  {t('Group {{index}}', { index: index + 1 })}
                </h3>
                <Button
                  type='button'
                  variant='ghost'
                  size='sm'
                  disabled={groups.fields.length <= 1}
                  aria-label={t('Remove group {{index}}', { index: index + 1 })}
                  onClick={() => groups.remove(index)}
                >
                  <Trash2 aria-hidden='true' />
                  {t('Remove')}
                </Button>
              </div>
              <div className='grid gap-4 md:grid-cols-3'>
                <FormField
                  control={form.control}
                  name={`groups.${index}.group_name`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Group name')}</FormLabel>
                      <FormControl>
                        <Input {...field} autoComplete='off' />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`groups.${index}.platform`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Platform')}</FormLabel>
                      <FormControl>
                        <NativeSelect {...field} className='w-full'>
                          {supplierPlatforms.map((value) => (
                            <NativeSelectOption key={value} value={value}>
                              {labels.platform[value]}
                            </NativeSelectOption>
                          ))}
                        </NativeSelect>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`groups.${index}.ratio`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Ratio')}</FormLabel>
                      <FormControl>
                        <Input {...field} inputMode='decimal' />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name={`groups.${index}.models`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Main models')}{' '}
                      <span className='text-muted-foreground font-normal'>
                        ({t('optional')})
                      </span>
                    </FormLabel>
                    <ModelTagInput
                      id={`supplier-group-${item.id}-models`}
                      value={field.value}
                      onChange={field.onChange}
                      disabled={pending}
                    />
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name={`groups.${index}.description`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Notes')}{' '}
                      <span className='text-muted-foreground font-normal'>
                        ({t('optional')})
                      </span>
                    </FormLabel>
                    <FormControl>
                      <Textarea {...field} className='min-h-20 resize-y' />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'For example: genuine upstream or not, can be re-exposed, injection, distillation, concurrency limit.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </section>
          ))}
          {form.formState.errors.groups?.root?.message && (
            <p role='alert' className='text-destructive text-sm'>
              {t(form.formState.errors.groups.root.message)}
            </p>
          )}
          {form.formState.errors.groups?.message && (
            <p role='alert' className='text-destructive text-sm'>
              {t(form.formState.errors.groups.message)}
            </p>
          )}
          <Button
            type='button'
            variant='outline'
            disabled={groups.fields.length >= SUPPLIER_MAX_GROUPS}
            onClick={() => groups.append({ ...emptyGroup, models: [] })}
          >
            <Plus aria-hidden='true' />
            {t('Add group')}
          </Button>
        </fieldset>

        {serverError && (
          <Alert variant='destructive'>
            <AlertDescription>{t(serverError)}</AlertDescription>
          </Alert>
        )}
        <div className='flex flex-wrap items-center gap-3 border-t pt-4'>
          <Button type='submit' disabled={pending}>
            {pending ? t('Submitting...') : t('Submit application')}
          </Button>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Submissions cannot be edited. Review everything before sending.'
            )}
          </p>
        </div>
      </form>
    </Form>
  )
}
