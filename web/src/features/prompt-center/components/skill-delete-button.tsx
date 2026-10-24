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

import { deletePromptTemplate } from '../api'
import type { PromptTemplate } from '../types'

export function SkillDeleteButton(props: { template: PromptTemplate }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirming, setConfirming] = useState(false)
  const mutation = useMutation({
    mutationFn: () =>
      deletePromptTemplate(props.template.id, props.template.version),
    onSuccess: async (result) => {
      setConfirming(false)
      queryClient.setQueriesData<PromptTemplate[]>(
        { queryKey: ['prompt-center'] },
        (current) =>
          current?.filter((template) => template.id !== props.template.id)
      )
      toast.success(
        t('Skill deleted. Uninstalled from {{count}} token(s).', {
          count: result.uninstalled_tokens,
        })
      )
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['prompt-center'] }),
        queryClient.invalidateQueries({ queryKey: ['token-prompts'] }),
        queryClient.invalidateQueries({ queryKey: ['keys'] }),
        queryClient.invalidateQueries({ queryKey: ['prompt-center-tokens'] }),
      ])
    },
  })

  return (
    <>
      <Button
        size='sm'
        variant='destructive'
        aria-label={t('Delete skill {{title}}', {
          title: props.template.title,
        })}
        disabled={mutation.isPending}
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
        title={t('Delete skill?')}
        desc={t(
          'Deleting "{{title}}" automatically uninstalls it from every token that has it installed. This action cannot be undone.',
          { title: props.template.title }
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
            {t(mutation.error.message || 'Failed to delete skill')}
          </p>
        )}
      </ConfirmDialog>
    </>
  )
}
