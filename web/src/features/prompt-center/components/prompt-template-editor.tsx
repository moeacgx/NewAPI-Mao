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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { getGroupDetails } from '@/features/system-settings/api'

import { getPromptCategories, savePromptTemplate } from '../api'
import {
  promptTemplateSchema,
  type PromptTemplateForm,
} from '../lib/validation'
import type { PromptTemplate } from '../types'
import { BundleSkillEditor } from './bundle-skill-editor'
import { SkillResourceEditor } from './skill-resource-editor'

export function PromptTemplateEditor(props: {
  template?: PromptTemplate
  onClose: () => void
}) {
  const { t } = useTranslation()
  const fieldId = useId()
  const [importError, setImportError] = useState('')
  const queryClient = useQueryClient()
  const groups = useQuery({
    queryKey: ['prompt-center-admin-groups'],
    queryFn: getGroupDetails,
  })
  const categories = useQuery({
    queryKey: ['prompt-categories'],
    queryFn: getPromptCategories,
  })
  const form = useForm<PromptTemplateForm>({
    resolver: zodResolver(promptTemplateSchema),
    defaultValues: {
      title: props.template?.title ?? '',
      description: props.template?.description ?? '',
      content: props.template?.content ?? '',
      enabled: props.template?.enabled ?? true,
      sort_order: props.template?.sort_order ?? 0,
      version: props.template?.version ?? 0,
      group_ids: props.template?.group_ids ?? [],
      kind: props.template?.kind ?? 'prompt',
      load_mode:
        props.template?.load_mode ||
        (props.template?.id ? 'eager' : 'progressive'),
      category_id: props.template?.category_id ?? 0,
      skills: props.template?.skills?.map((skill) => ({
        ...skill,
        content: skill.content ?? '',
      })) ?? [
        { title: '', content: '' },
        { title: '', content: '' },
      ],
    },
  })
  const selectedGroups = form.watch('group_ids')
  const kind = form.watch('kind')
  const progressive = form.watch('load_mode') === 'progressive'
  const categoryId = form.watch('category_id') ?? 0
  const mutation = useMutation({
    mutationFn: (values: PromptTemplateForm) => {
      let skills = values.skills
      if (values.kind !== 'bundle') {
        skills = []
        if (values.load_mode === 'progressive') {
          skills = [
            {
              ...values.skills?.[0],
              title: values.title,
              description: values.description,
              content: values.content,
            },
          ]
        }
      }
      return savePromptTemplate(
        {
          ...values,
          skills,
          content: values.kind === 'bundle' ? '' : values.content,
        },
        props.template?.id
      )
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['prompt-center'] })
      await queryClient.invalidateQueries({ queryKey: ['keys'] })
      props.onClose()
    },
  })

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) props.onClose()
      }}
    >
      <DialogContent className='max-h-[90dvh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {props.template?.id ? t('Edit prompt') : t('Create prompt')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Updates apply on subsequent requests. Skills are read by gateway tools; skill scripts are not executed.'
            )}
          </DialogDescription>
        </DialogHeader>
        <form
          className='space-y-4'
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <fieldset disabled={mutation.isPending} className='space-y-4'>
            <div className='bg-primary/[0.04] border-primary/15 space-y-2 rounded-xl border p-4'>
              <Label htmlFor={`${fieldId}-load-mode`}>
                {t('Loading mode')}
              </Label>
              <select
                id={`${fieldId}-load-mode`}
                className='bg-background h-9 w-full rounded-md border px-3 text-sm'
                {...form.register('load_mode')}
              >
                <option value='progressive'>
                  {t('Progressive skill disclosure')}
                </option>
                <option value='eager'>{t('Always load prompt')}</option>
              </select>
              <p className='text-muted-foreground text-xs leading-5'>
                {progressive
                  ? t(
                      'The model first sees the skill catalog. The gateway reads instructions and references only when needed. Extra model rounds are billed; scripts are not executed.'
                    )
                  : t(
                      'The full prompt is added to every matching request. Each prompt is limited to 32768 bytes, with 65536 enabled bytes per token.'
                    )}
              </p>
              {progressive && (
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Up to 8 MiB per file and 64 MiB per package. Files are read in pages; model context limits still apply.'
                  )}
                </p>
              )}
              {!progressive && (
                <p className='text-xs leading-5 text-amber-700 dark:text-amber-400'>
                  {t(
                    'Saving in always-loaded mode removes reference files. Choose progressive disclosure to keep them.'
                  )}
                </p>
              )}
            </div>
            <div className='grid gap-3 sm:grid-cols-2'>
              <div className='space-y-1.5'>
                <Label htmlFor={`${fieldId}-kind`}>{t('Skill type')}</Label>
                <select
                  id={`${fieldId}-kind`}
                  className='bg-background h-9 w-full rounded-md border px-3 text-sm'
                  {...form.register('kind')}
                >
                  <option value='prompt'>{t('Single skill')}</option>
                  <option value='bundle'>{t('Skill bundle')}</option>
                </select>
              </div>
              <div className='space-y-1.5'>
                <Label htmlFor={`${fieldId}-category`}>
                  {t('Skill category')}
                </Label>
                <select
                  id={`${fieldId}-category`}
                  className='bg-background h-9 w-full rounded-md border px-3 text-sm'
                  value={categoryId}
                  onChange={(event) =>
                    form.setValue('category_id', Number(event.target.value), {
                      shouldDirty: true,
                    })
                  }
                >
                  <option value={0}>{t('Uncategorized skills')}</option>
                  {categories.data?.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                  {categoryId > 0 &&
                    !categories.data?.some(
                      (category) => category.id === categoryId
                    ) && (
                      <option value={categoryId}>
                        {props.template?.category_name ||
                          t('Unavailable category')}
                      </option>
                    )}
                </select>
                {categories.isError && (
                  <div role='alert' className='text-destructive text-xs'>
                    <p>{t('Failed to load skill categories')}</p>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() => categories.refetch()}
                    >
                      {t('Retry')}
                    </Button>
                  </div>
                )}
              </div>
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor={`${fieldId}-title`}>{t('Title')}</Label>
              <Input
                id={`${fieldId}-title`}
                maxLength={80}
                aria-invalid={Boolean(form.formState.errors.title)}
                {...form.register('title')}
              />
              {form.formState.errors.title && (
                <p role='alert' className='text-destructive text-xs'>
                  {t('Enter a title of 1 to 80 characters')}
                </p>
              )}
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor={`${fieldId}-description`}>
                {t('Description')}
              </Label>
              <Textarea
                id={`${fieldId}-description`}
                maxLength={512}
                aria-invalid={Boolean(form.formState.errors.description)}
                {...form.register('description')}
              />
              {form.formState.errors.description && (
                <p role='alert' className='text-destructive text-xs'>
                  {t(form.formState.errors.description.message || '')}
                </p>
              )}
            </div>
            {kind === 'bundle' ? (
              <BundleSkillEditor form={form} fieldId={fieldId} />
            ) : (
              <div className='space-y-1.5'>
                <Label htmlFor={`${fieldId}-file`}>
                  {t('Import Markdown or SKILL.md as text')}
                </Label>
                <Input
                  id={`${fieldId}-file`}
                  type='file'
                  accept='.md,.txt,text/plain,text/markdown'
                  onChange={async (event) => {
                    const file = event.target.files?.[0]
                    if (!file) return
                    setImportError('')
                    try {
                      if (file.size > (progressive ? 8 * 1024 * 1024 : 32768)) {
                        throw new Error(
                          progressive
                            ? 'Enter skill text up to 8 MiB per file'
                            : 'Prompt content must not exceed 32768 bytes'
                        )
                      }
                      const text = await file.text()
                      if (text.includes('\0')) {
                        throw new Error('Only plain text files are supported')
                      }
                      form.setValue('content', text, {
                        shouldDirty: true,
                        shouldValidate: true,
                      })
                    } catch (error) {
                      setImportError(
                        error instanceof Error
                          ? error.message
                          : 'Failed to read file'
                      )
                    }
                  }}
                />
                {importError && (
                  <p role='alert' className='text-destructive text-xs'>
                    {t(importError)}
                  </p>
                )}
                <Label htmlFor={`${fieldId}-content`}>
                  {t('Prompt content')}
                </Label>
                <Textarea
                  id={`${fieldId}-content`}
                  rows={10}
                  className='max-h-96 font-mono'
                  aria-invalid={Boolean(form.formState.errors.content)}
                  {...form.register('content')}
                />
                {form.formState.errors.content && (
                  <p role='alert' className='text-destructive text-xs'>
                    {t(form.formState.errors.content.message || '')}
                  </p>
                )}
                {progressive && (
                  <SkillResourceEditor
                    form={form}
                    skillIndex={0}
                    fieldId={`${fieldId}-single`}
                  />
                )}
              </div>
            )}
            <fieldset className='bg-muted/20 space-y-2 rounded-lg border p-3'>
              <legend className='px-1 text-sm font-medium'>
                {t('Allowed groups')}
              </legend>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Whitelist applies to the actual request group, including automatic selection and retries.'
                )}
              </p>
              {groups.isPending && <p role='status'>{t('Loading...')}</p>}
              {groups.isError && (
                <div role='alert'>
                  <p>{t('Failed to load groups')}</p>
                  <Button
                    type='button'
                    variant='outline'
                    onClick={() => groups.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </div>
              )}
              {groups.isSuccess && (
                <div className='grid max-h-48 gap-2 overflow-y-auto sm:grid-cols-2'>
                  {selectedGroups
                    .filter(
                      (id) =>
                        !groups.data.groups.some((group) => group.id === id)
                    )
                    .map((id) => (
                      <label
                        key={`deleted-${id}`}
                        className='text-muted-foreground flex items-center gap-2'
                      >
                        <input
                          type='checkbox'
                          checked
                          onChange={() =>
                            form.setValue(
                              'group_ids',
                              selectedGroups.filter((value) => value !== id),
                              { shouldDirty: true, shouldValidate: true }
                            )
                          }
                        />
                        {t('Deleted ({{id}})', { id })}
                      </label>
                    ))}
                  {groups.data.groups
                    .filter((group) => group.id > 0)
                    .map((group) => (
                      <label
                        key={group.id}
                        className='flex min-w-0 items-center gap-2'
                      >
                        <input
                          type='checkbox'
                          checked={selectedGroups.includes(group.id)}
                          onChange={(event) =>
                            form.setValue(
                              'group_ids',
                              event.target.checked
                                ? [...selectedGroups, group.id]
                                : selectedGroups.filter(
                                    (id) => id !== group.id
                                  ),
                              { shouldDirty: true, shouldValidate: true }
                            )
                          }
                        />
                        <span className='[overflow-wrap:anywhere]'>
                          {group.name || group.code}
                        </span>
                      </label>
                    ))}
                  {groups.data.groups.length === 0 && (
                    <p>{t('No available groups')}</p>
                  )}
                </div>
              )}
              {form.formState.errors.group_ids && (
                <p role='alert' className='text-destructive text-xs'>
                  {t('Select at least one allowed group')}
                </p>
              )}
            </fieldset>
            <div className='flex flex-wrap items-center gap-4'>
              <label className='flex items-center gap-2 text-sm'>
                <input type='checkbox' {...form.register('enabled')} />
                {t('Published')}
              </label>
              <div className='flex items-center gap-2'>
                <Label htmlFor={`${fieldId}-order`}>{t('Sort order')}</Label>
                <Input
                  id={`${fieldId}-order`}
                  type='number'
                  min={-1000000}
                  max={1000000}
                  className='w-24'
                  {...form.register('sort_order', { valueAsNumber: true })}
                />
              </div>
            </div>
            {form.formState.errors.sort_order && (
              <p role='alert' className='text-destructive text-xs'>
                {t('Sort order')}: -1000000 – 1000000
              </p>
            )}
          </fieldset>
          {mutation.isError && (
            <p role='alert' className='text-destructive'>
              {t(mutation.error.message)}
            </p>
          )}
          <div className='flex justify-end gap-2 border-t pt-3'>
            <Button
              type='button'
              variant='outline'
              disabled={mutation.isPending}
              onClick={props.onClose}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='submit'
              disabled={
                mutation.isPending || !groups.isSuccess || !categories.isSuccess
              }
            >
              {t('Save')}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
