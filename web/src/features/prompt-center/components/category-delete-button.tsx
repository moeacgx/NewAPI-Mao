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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'

import { deletePromptCategory } from '../api'
import type { PromptCategory } from '../types'

export function CategoryDeleteButton(props: {
  category: PromptCategory
  disabled?: boolean
  onDeleted: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirming, setConfirming] = useState(false)
  const mutation = useMutation({
    mutationFn: () =>
      deletePromptCategory(props.category.id, props.category.version),
    onSuccess: async (result) => {
      setConfirming(false)
      queryClient.setQueryData<PromptCategory[]>(
        ['prompt-categories'],
        (current) =>
          current?.filter((category) => category.id !== props.category.id)
      )
      props.onDeleted()
      toast.success(
        t('Category deleted. {{count}} skill(s) moved to Uncategorized.', {
          count: result.moved_templates,
        })
      )
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['prompt-categories'] }),
        queryClient.invalidateQueries({ queryKey: ['prompt-center'] }),
        queryClient.invalidateQueries({ queryKey: ['token-prompts'] }),
      ])
    },
  })

  return (
    <>
      <Button
        size='sm'
        variant='destructive'
        aria-label={t('Delete category {{name}}', {
          name: props.category.name,
        })}
        disabled={props.disabled || mutation.isPending}
        onClick={() => {
          mutation.reset()
          setConfirming(true)
        }}
      >
        <Trash2 aria-hidden='true' />
        {t('Delete')}
      </Button>
      <ConfirmDialog
        open={confirming}
        onOpenChange={(open) => {
          if (!mutation.isPending) setConfirming(open)
        }}
        title={t('Delete skill category?')}
        desc={t(
          'Deleting "{{name}}" moves its skills to Uncategorized. This action cannot be undone.',
          { name: props.category.name }
        )}
        confirmText={t('Delete')}
        destructive
        isLoading={mutation.isPending}
        handleConfirm={() => {
          if (!mutation.isPending) mutation.mutate()
        }}
      >
        {mutation.isError && (
          <p className='text-destructive text-sm' role='alert'>
            {t(mutation.error.message || 'Failed to delete skill category')}
          </p>
        )}
      </ConfirmDialog>
    </>
  )
}
