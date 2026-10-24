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
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react'
import { useFieldArray, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import type { PromptTemplateForm } from '../lib/validation'
import { SkillResourceEditor } from './skill-resource-editor'

export function BundleSkillEditor(props: {
  form: UseFormReturn<PromptTemplateForm>
  fieldId: string
}) {
  const { t } = useTranslation()
  const array = useFieldArray({ control: props.form.control, name: 'skills' })
  const progressive = props.form.watch('load_mode') === 'progressive'
  const limit = progressive ? 128 : 16
  const error = props.form.formState.errors.skills
  return (
    <section
      className='bg-muted/20 space-y-3 rounded-xl border p-3'
      aria-label={t('Bundle skills')}
    >
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>
          {t('Bundle skills')} · {array.fields.length}/{limit}
        </h3>
        <Button
          type='button'
          size='sm'
          variant='outline'
          disabled={array.fields.length >= limit}
          onClick={() => array.append({ title: '', content: '' })}
        >
          <Plus aria-hidden='true' />
          {t('Add skill')}
        </Button>
      </div>
      <p className='text-muted-foreground text-xs'>
        {progressive
          ? t(
              'Each skill is discovered and read independently. Install and uninstall the bundle as one package.'
            )
          : t(
              'Skills load in this order. Install and uninstall the bundle as one item.'
            )}
      </p>
      {array.fields.map((field, index) => (
        <div
          key={field.id}
          className='bg-background space-y-2 rounded-xl border p-3'
        >
          <div className='flex items-center justify-between gap-2'>
            <span className='text-muted-foreground text-xs'>
              {t('Skill {{number}}', { number: index + 1 })}
            </span>
            <div className='flex gap-1'>
              <Button
                type='button'
                size='icon-sm'
                variant='outline'
                aria-label={t('Move skill {{number}} up', {
                  number: index + 1,
                })}
                disabled={index === 0}
                onClick={() => array.move(index, index - 1)}
              >
                <ArrowUp aria-hidden='true' />
              </Button>
              <Button
                type='button'
                size='icon-sm'
                variant='outline'
                aria-label={t('Move skill {{number}} down', {
                  number: index + 1,
                })}
                disabled={index === array.fields.length - 1}
                onClick={() => array.move(index, index + 1)}
              >
                <ArrowDown aria-hidden='true' />
              </Button>
              <Button
                type='button'
                size='icon-sm'
                variant='outline'
                aria-label={t('Remove skill {{number}}', { number: index + 1 })}
                disabled={array.fields.length <= 2}
                onClick={() => array.remove(index)}
              >
                <Trash2 aria-hidden='true' />
              </Button>
            </div>
          </div>
          <Label htmlFor={`${props.fieldId}-skill-${field.id}-title`}>
            {t('Skill title')}
          </Label>
          <Input
            id={`${props.fieldId}-skill-${field.id}-title`}
            maxLength={80}
            aria-invalid={Boolean(error?.[index]?.title)}
            {...props.form.register(`skills.${index}.title`)}
          />
          {error?.[index]?.title && (
            <p role='alert' className='text-destructive text-xs'>
              {t(error[index].title.message || '')}
            </p>
          )}
          {progressive && (
            <>
              <Label htmlFor={`${props.fieldId}-skill-${field.id}-description`}>
                {t('Skill description')}
              </Label>
              <Textarea
                id={`${props.fieldId}-skill-${field.id}-description`}
                rows={2}
                maxLength={512}
                aria-invalid={Boolean(error?.[index]?.description)}
                {...props.form.register(`skills.${index}.description`)}
              />
              {error?.[index]?.description && (
                <p role='alert' className='text-destructive text-xs'>
                  {t(error[index].description.message || '')}
                </p>
              )}
            </>
          )}
          <Label htmlFor={`${props.fieldId}-skill-${field.id}-content`}>
            {t('Skill content')}
          </Label>
          <Textarea
            id={`${props.fieldId}-skill-${field.id}-content`}
            rows={5}
            className='max-h-72 font-mono'
            aria-invalid={Boolean(error?.[index]?.content)}
            {...props.form.register(`skills.${index}.content`)}
          />
          {error?.[index]?.content && (
            <p role='alert' className='text-destructive text-xs'>
              {t(error[index].content.message || '')}
            </p>
          )}
          {progressive && (
            <SkillResourceEditor
              form={props.form}
              skillIndex={index}
              fieldId={`${props.fieldId}-skill-${field.id}`}
            />
          )}
        </div>
      ))}
      {(error?.message || error?.root?.message) && (
        <p role='alert' className='text-destructive text-xs'>
          {t(error.message || error.root?.message || '')}
        </p>
      )}
    </section>
  )
}
