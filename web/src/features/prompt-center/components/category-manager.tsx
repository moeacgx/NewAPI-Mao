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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { getPromptCategories, savePromptCategory } from '../api'
import type { PromptCategory } from '../types'

export function CategoryManager(props: { onClose: () => void }) {
  const { t } = useTranslation()
  const nameId = useId()
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState<PromptCategory>()
  const [name, setName] = useState('')
  const query = useQuery({
    queryKey: ['prompt-categories'],
    queryFn: getPromptCategories,
  })
  const mutation = useMutation({
    mutationFn: () =>
      savePromptCategory(
        { name: name.trim(), version: editing?.version ?? 0 },
        editing?.id
      ),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['prompt-categories'] }),
        queryClient.invalidateQueries({ queryKey: ['prompt-center'] }),
        queryClient.invalidateQueries({ queryKey: ['token-prompts'] }),
      ])
      setEditing(undefined)
      setName('')
    },
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) props.onClose()
      }}
    >
      <DialogContent className='max-h-[85dvh] overflow-y-auto rounded-2xl sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Manage skill categories')}</DialogTitle>
          <DialogDescription>
            {t(
              'Categories organize skills and do not change group permissions.'
            )}
          </DialogDescription>
        </DialogHeader>
        <div className='bg-muted/20 space-y-2 rounded-xl border p-3'>
          {query.isPending && <p role='status'>{t('Loading...')}</p>}
          {query.isError && (
            <div role='alert'>
              <p>{t('Failed to load skill categories')}</p>
              <Button variant='outline' onClick={() => query.refetch()}>
                {t('Retry')}
              </Button>
            </div>
          )}
          {query.isSuccess && query.data.length === 0 && (
            <p className='text-muted-foreground text-sm'>
              {t('No skill categories yet')}
            </p>
          )}
          {query.data?.map((category) => (
            <div
              key={category.id}
              className='bg-background flex items-center justify-between gap-3 rounded-lg border p-3'
            >
              <span className='min-w-0 [overflow-wrap:anywhere]'>
                {category.name}
              </span>
              <Button
                variant='outline'
                size='sm'
                disabled={mutation.isPending}
                onClick={() => {
                  setEditing(category)
                  setName(category.name)
                  mutation.reset()
                }}
              >
                {t('Rename')}
              </Button>
            </div>
          ))}
        </div>
        <form
          className='space-y-3 rounded-xl border p-4'
          onSubmit={(event) => {
            event.preventDefault()
            mutation.mutate()
          }}
        >
          <Label htmlFor={nameId}>
            {editing ? t('Rename skill category') : t('New skill category')}
          </Label>
          <Input
            id={nameId}
            value={name}
            maxLength={40}
            disabled={mutation.isPending}
            onChange={(event) => setName(event.target.value)}
          />
          {mutation.isError && (
            <p role='alert' className='text-destructive text-sm'>
              {t(mutation.error.message)}
            </p>
          )}
          <div className='flex justify-end gap-2'>
            {editing && (
              <Button
                type='button'
                variant='outline'
                disabled={mutation.isPending}
                onClick={() => {
                  setEditing(undefined)
                  setName('')
                  mutation.reset()
                }}
              >
                {t('Cancel')}
              </Button>
            )}
            <Button
              type='submit'
              disabled={
                !name.trim() ||
                [...name.trim()].length > 40 ||
                mutation.isPending ||
                !query.isSuccess
              }
            >
              {editing ? t('Save') : t('Add category')}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
