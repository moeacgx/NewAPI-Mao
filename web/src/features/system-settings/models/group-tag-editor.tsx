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
import { useId } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import type * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { groupTagFormSchema } from '@/features/group-tags/lib/group-tag-form'
import type { GroupTag, GroupTagInput } from '@/features/group-tags/types'

export type TagBindingGroup = { id?: number; code: string; name: string }

export function GroupTagEditor(props: {
  tag?: GroupTag
  groups: TagBindingGroup[]
  saving: boolean
  error: string | null
  onSave: (input: GroupTagInput) => void
  onClose: () => void
}) {
  const { t } = useTranslation()
  const prefix = useId()
  const form = useForm<
    z.input<typeof groupTagFormSchema>,
    unknown,
    GroupTagInput
  >({
    resolver: zodResolver(groupTagFormSchema),
    defaultValues: {
      name: props.tag?.name ?? '',
      description: props.tag?.description ?? '',
      sort_order: props.tag?.sort_order ?? 0,
      icons: (props.tag?.icons ?? []).join('\n'),
      match_keywords: (props.tag?.match_keywords ?? []).join('\n'),
      group_ids: (props.tag?.group_ids ?? []).filter((id) =>
        props.groups.some((group) => group.id === id)
      ),
    },
  })
  const selectedIds = form.watch('group_ids')
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !props.saving) props.onClose()
      }}
    >
      <DialogContent className='max-h-[90dvh] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>
            {props.tag ? t('Edit group tag') : t('New group tag')}
          </DialogTitle>
          <DialogDescription>
            {t('Tags filter groups without changing access permissions.')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={form.handleSubmit(props.onSave)} className='space-y-4'>
          <fieldset disabled={props.saving} className='space-y-4'>
            <div className='space-y-2'>
              <Label htmlFor={`${prefix}-name`}>{t('Tag name')}</Label>
              <Input
                id={`${prefix}-name`}
                maxLength={64}
                aria-invalid={!!form.formState.errors.name}
                {...form.register('name')}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor={`${prefix}-description`}>
                {t('Description')}
              </Label>
              <Textarea
                id={`${prefix}-description`}
                maxLength={512}
                {...form.register('description')}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor={`${prefix}-sort`}>{t('Sort order')}</Label>
              <Input
                id={`${prefix}-sort`}
                type='number'
                min={-1000000}
                max={1000000}
                step={1}
                {...form.register('sort_order', { valueAsNumber: true })}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor={`${prefix}-icons`}>
                {t('Logos (one per line, up to 6)')}
              </Label>
              <Textarea id={`${prefix}-icons`} {...form.register('icons')} />
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Use built-in icon names or HTTP(S) image URLs without credentials.'
                )}
              </p>
            </div>
            <div className='space-y-2'>
              <Label htmlFor={`${prefix}-keywords`}>
                {t('Match keywords (one per line, up to 32)')}
              </Label>
              <Textarea
                id={`${prefix}-keywords`}
                {...form.register('match_keywords')}
              />
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Matching group names are added to bindings when saved. Existing bindings are preserved.'
                )}
              </p>
            </div>
            <fieldset className='bg-muted/20 space-y-2 rounded-lg border p-3'>
              <legend className='px-1 text-sm font-medium'>
                {t('Groups')}
              </legend>
              <div className='max-h-48 space-y-2 overflow-y-auto'>
                {props.groups
                  .filter(
                    (group): group is TagBindingGroup & { id: number } =>
                      typeof group.id === 'number' && group.id > 0
                  )
                  .map((group) => (
                    <label
                      key={group.id}
                      className='flex items-center gap-2 text-sm'
                    >
                      <input
                        type='checkbox'
                        checked={selectedIds.includes(group.id)}
                        onChange={(event) =>
                          form.setValue(
                            'group_ids',
                            event.target.checked
                              ? [...selectedIds, group.id]
                              : selectedIds.filter((id) => id !== group.id),
                            { shouldDirty: true, shouldValidate: true }
                          )
                        }
                      />
                      <span
                        className='min-w-0 truncate'
                        title={group.name || group.code}
                      >
                        {group.name || group.code}
                      </span>
                    </label>
                  ))}
                {props.groups.length === 0 && (
                  <p className='text-muted-foreground text-sm'>
                    {t('No available groups')}
                  </p>
                )}
              </div>
            </fieldset>
          </fieldset>
          {Object.keys(form.formState.errors).length > 0 && (
            <p role='alert' className='text-destructive text-sm'>
              {t('Check the tag name, sort order, logos, and keywords.')}
            </p>
          )}
          {props.error && (
            <p role='alert' className='text-destructive text-sm'>
              {props.error}
            </p>
          )}
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              disabled={props.saving}
              onClick={props.onClose}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' disabled={props.saving}>
              {props.saving ? t('Saving...') : t('Save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
