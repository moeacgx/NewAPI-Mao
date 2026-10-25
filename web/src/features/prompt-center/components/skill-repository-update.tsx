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
import { RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'

import { RepositoryLocalChangesError, updateSkillRepository } from '../api'
import type { PromptTemplate } from '../types'
import { SkillRepositoryDetails } from './skill-repository-details'

export function SkillRepositoryUpdate(props: { template: PromptTemplate }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOverwrite, setConfirmOverwrite] = useState(false)
  const mutation = useMutation({
    mutationFn: (overwriteLocal: boolean) =>
      updateSkillRepository(
        props.template.id,
        props.template.version,
        overwriteLocal
      ),
    onError: (reason) => {
      if (reason instanceof RepositoryLocalChangesError) {
        setConfirmOverwrite(true)
      }
    },
    onSuccess: async (result) => {
      setConfirmOverwrite(false)
      queryClient.setQueryData<PromptTemplate[]>(
        ['prompt-center', true],
        (current) =>
          current?.map((template) =>
            template.id === result.template.id ? result.template : template
          )
      )
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['prompt-center'] }),
        queryClient.invalidateQueries({ queryKey: ['token-prompts'] }),
        queryClient.invalidateQueries({ queryKey: ['keys'] }),
        queryClient.invalidateQueries({ queryKey: ['prompt-center-tokens'] }),
      ])
    },
  })
  if (!props.template.repository_source) return null

  return (
    <div className='space-y-2'>
      <SkillRepositoryDetails source={props.template.repository_source} />
      <div className='flex flex-wrap items-center justify-between gap-2'>
        {props.template.repository_source.locally_modified && (
          <span className='text-muted-foreground text-xs'>
            {t('Local edits')}
          </span>
        )}
        <Button
          className='ml-auto'
          size='sm'
          variant='outline'
          disabled={mutation.isPending}
          onClick={() => {
            if (mutation.isPending) return
            if (props.template.repository_source?.locally_modified) {
              mutation.reset()
              setConfirmOverwrite(true)
              return
            }
            mutation.mutate(false)
          }}
        >
          <RefreshCw
            className={mutation.isPending ? 'animate-spin' : undefined}
            aria-hidden='true'
          />
          {t('Update from GitHub')}
        </Button>
      </div>
      {mutation.isSuccess && (
        <p className='text-muted-foreground text-xs' role='status'>
          {mutation.data.updated
            ? t('Skill updated from GitHub')
            : t('Skill is already up to date')}
        </p>
      )}
      {mutation.isError &&
        !(mutation.error instanceof RepositoryLocalChangesError) && (
          <p className='text-destructive text-xs' role='alert'>
            {t(mutation.error.message || 'Failed to update skill')}
          </p>
        )}
      <ConfirmDialog
        open={confirmOverwrite}
        onOpenChange={(open) => {
          if (!mutation.isPending) setConfirmOverwrite(open)
        }}
        title={t('Replace locally edited skill content?')}
        desc={t(
          'Updating replaces skill instructions and references. Your title, description, category, allowed groups, publication status and sort order are kept.'
        )}
        confirmText={t('Replace and update')}
        isLoading={mutation.isPending}
        handleConfirm={() => {
          if (!mutation.isPending) mutation.mutate(true)
        }}
      />
    </div>
  )
}
