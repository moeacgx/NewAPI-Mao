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
import { ChevronRight, FileText, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useFieldArray, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

import type { PromptTemplateForm } from '../lib/validation'

export function SkillResourceEditor(props: {
  form: UseFormReturn<PromptTemplateForm>
  skillIndex: number
  fieldId: string
}) {
  const { t } = useTranslation()
  const array = useFieldArray({
    control: props.form.control,
    name: `skills.${props.skillIndex}.resources`,
  })
  // 已有引用正文默认折叠，避免大仓库导入后表单过长；新增的引用不在集合中，保持展开便于填写。
  const [collapsedIds, setCollapsedIds] = useState(
    () => new Set(array.fields.map((field) => field.id))
  )
  const errors = props.form.formState.errors.skills?.[props.skillIndex]
  return (
    <section
      className='bg-muted/20 space-y-3 rounded-xl border p-3'
      aria-label={t('Skill reference files')}
    >
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h4 className='flex items-center gap-2 text-sm font-medium'>
          <FileText className='text-primary size-4' aria-hidden='true' />
          {t('Skill reference files')} · {array.fields.length}
        </h4>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => array.append({ path: '', content: '' })}
        >
          <Plus aria-hidden='true' />
          {t('Add reference file')}
        </Button>
      </div>
      <p className='text-muted-foreground text-xs'>
        {t(
          'References are read only when requested. Paths are relative to the skill folder; scripts are not executed.'
        )}
      </p>
      <div className='space-y-1.5'>
        <Label htmlFor={`${props.fieldId}-entry-path`}>
          {t('Skill entry path')}
        </Label>
        <Input
          id={`${props.fieldId}-entry-path`}
          placeholder='SKILL.md'
          aria-invalid={Boolean(errors?.path)}
          {...props.form.register(`skills.${props.skillIndex}.path`)}
        />
        {errors?.path && (
          <p role='alert' className='text-destructive text-xs'>
            {t(errors.path.message || '')}
          </p>
        )}
      </div>
      {array.fields.map((field, index) => {
        const contentId = `${props.fieldId}-resource-${field.id}-content`
        const contentError = errors?.resources?.[index]?.content
        // 校验失败的正文强制展开，确保错误与对应内容同时可见。
        const expanded = !collapsedIds.has(field.id) || Boolean(contentError)
        return (
          <div
            key={field.id}
            className='bg-background space-y-2 rounded-lg border p-3'
          >
            <div className='flex items-end gap-2'>
              <div className='min-w-0 flex-1 space-y-1.5'>
                <Label htmlFor={`${props.fieldId}-resource-${field.id}-path`}>
                  {t('Reference path')}
                </Label>
                <Input
                  id={`${props.fieldId}-resource-${field.id}-path`}
                  placeholder='references/guide.md'
                  aria-invalid={Boolean(errors?.resources?.[index]?.path)}
                  {...props.form.register(
                    `skills.${props.skillIndex}.resources.${index}.path`
                  )}
                />
              </div>
              <Button
                type='button'
                variant='outline'
                size='icon'
                aria-label={t('Remove reference {{number}}', {
                  number: index + 1,
                })}
                onClick={() => array.remove(index)}
              >
                <Trash2 aria-hidden='true' />
              </Button>
            </div>
            {errors?.resources?.[index]?.path && (
              <p role='alert' className='text-destructive text-xs'>
                {t(errors.resources[index].path.message || '')}
              </p>
            )}
            <button
              type='button'
              id={`${contentId}-toggle`}
              aria-expanded={expanded}
              aria-controls={contentId}
              className='hover:text-primary focus-visible:ring-ring/50 flex items-center gap-1 rounded-md text-sm font-medium outline-none focus-visible:ring-3'
              onClick={() =>
                setCollapsedIds((current) => {
                  const next = new Set(current)
                  if (expanded) next.add(field.id)
                  else next.delete(field.id)
                  return next
                })
              }
            >
              <ChevronRight
                className={cn(
                  'size-4 transition-transform',
                  expanded && 'rotate-90'
                )}
                aria-hidden='true'
              />
              {t('Reference content')}
            </button>
            {expanded && (
              <Textarea
                id={contentId}
                aria-labelledby={`${contentId}-toggle`}
                rows={4}
                className='max-h-64 font-mono'
                aria-invalid={Boolean(contentError)}
                {...props.form.register(
                  `skills.${props.skillIndex}.resources.${index}.content`
                )}
              />
            )}
            {contentError && (
              <p role='alert' className='text-destructive text-xs'>
                {t(contentError.message || '')}
              </p>
            )}
          </div>
        )
      })}
    </section>
  )
}
