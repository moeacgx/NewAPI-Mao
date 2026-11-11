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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { deleteGroupTag, saveGroupTag } from '@/features/group-tags/api'
import { GroupTagLogos } from '@/features/group-tags/components/group-tag-logos'
import type { GroupTag, GroupTagInput } from '@/features/group-tags/types'
import { useGroupTags } from '@/features/group-tags/use-group-tags'
import { useIsAdmin } from '@/hooks/use-admin'

import { GroupTagEditor, type TagBindingGroup } from './group-tag-editor'

export function GroupTagSettings(props: {
  groups: TagBindingGroup[]
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const query = useGroupTags()
  const client = useQueryClient()
  const [editing, setEditing] = useState<GroupTag | 'new' | null>(null)
  const [deleting, setDeleting] = useState<GroupTag | null>(null)
  const [error, setError] = useState<string | null>(null)
  const mutation = useMutation({
    mutationFn: async (
      action: { input: GroupTagInput; id?: number } | { deleteId: number }
    ) => {
      if (!isAdmin) throw new Error(t('Permission denied'))
      if ('deleteId' in action) return deleteGroupTag(action.deleteId)
      return saveGroupTag(action.input, action.id)
    },
    onSuccess: async () => {
      setEditing(null)
      setDeleting(null)
      setError(null)
      toast.success(t('Saved successfully'))
      await Promise.all([
        client.invalidateQueries({ queryKey: ['group-tags'] }),
        client.invalidateQueries({ queryKey: ['user-groups'] }),
        client.invalidateQueries({ queryKey: ['user-self-groups'] }),
        client.invalidateQueries({
          queryKey: ['prompt-center-admin-group-tags'],
        }),
      ])
    },
    onError: (failure) => setError(failure.message || t('Failed to save')),
  })
  if (!isAdmin) return null
  return (
    <Card className='bg-card border'>
      <CardHeader className='bg-muted/20 border-b'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <CardTitle>{t('Group tags')}</CardTitle>
            <CardDescription>
              {t('Tags filter groups without changing access permissions.')}
            </CardDescription>
          </div>
          <div className='flex gap-2'>
            <Button
              type='button'
              variant='outline'
              disabled={query.isFetching || mutation.isPending}
              onClick={() => void query.refetch()}
            >
              {t('Refresh')}
            </Button>
            <Button
              type='button'
              disabled={
                props.disabled ||
                query.isPending ||
                query.isError ||
                mutation.isPending
              }
              onClick={() => {
                setError(null)
                setEditing('new')
              }}
            >
              {t('New group tag')}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-3'>
        {query.isPending && <p role='status'>{t('Loading...')}</p>}
        {query.isError && (
          <div role='alert' className='bg-muted/20 rounded-lg border p-3'>
            {query.error.message || t('Failed to load groups')}{' '}
            <Button
              type='button'
              variant='outline'
              onClick={() => void query.refetch()}
            >
              {t('Retry')}
            </Button>
          </div>
        )}
        {!query.isPending && !query.isError && query.data?.length === 0 && (
          <p className='bg-muted/20 rounded-lg border p-4 text-sm'>
            {t('No group tags yet.')}
          </p>
        )}
        {!query.isError &&
          query.data?.map((tag) => (
            <article
              key={tag.id}
              aria-label={tag.name}
              className='bg-background space-y-2 rounded-lg border p-3'
            >
              <div className='flex flex-wrap items-center justify-between gap-2'>
                <div className='flex min-w-0 items-center gap-2'>
                  <GroupTagLogos icons={tag.icons} />
                  <span className='font-medium break-all'>{tag.name}</span>
                </div>
                <div className='flex gap-2'>
                  <Button
                    type='button'
                    size='sm'
                    variant='outline'
                    disabled={props.disabled || mutation.isPending}
                    onClick={() => {
                      setError(null)
                      setEditing(tag)
                    }}
                  >
                    {t('Edit')}
                  </Button>
                  <Button
                    type='button'
                    size='sm'
                    variant='destructive'
                    disabled={mutation.isPending}
                    onClick={() => {
                      setError(null)
                      setDeleting(tag)
                    }}
                  >
                    {t('Delete')}
                  </Button>
                </div>
              </div>
              {tag.description && (
                <p className='text-muted-foreground text-sm break-all'>
                  {tag.description}
                </p>
              )}
              <p className='text-muted-foreground text-sm'>
                {tag.group_ids
                  .map((id) => {
                    const group = props.groups.find((item) => item.id === id)
                    return (
                      group?.name ||
                      group?.code ||
                      t('Deleted ({{id}})', { id })
                    )
                  })
                  .join(', ') || t('No available groups')}
              </p>
            </article>
          ))}
      </CardContent>
      {editing && (
        <GroupTagEditor
          tag={editing === 'new' ? undefined : editing}
          groups={props.groups}
          saving={mutation.isPending}
          error={error}
          onClose={() => {
            setEditing(null)
            setError(null)
          }}
          onSave={(input) =>
            mutation.mutate({
              input,
              id: editing === 'new' ? undefined : editing.id,
            })
          }
        />
      )}
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open && !mutation.isPending) {
            setDeleting(null)
            setError(null)
          }
        }}
        title={t('Delete group tag')}
        desc={t(
          'Delete tag {{name}}? Group and token assignments will be preserved.',
          { name: deleting?.name }
        )}
        confirmText={t('Delete')}
        destructive
        isLoading={mutation.isPending}
        handleConfirm={() => {
          if (deleting) mutation.mutate({ deleteId: deleting.id })
        }}
      >
        {error && (
          <p role='alert' className='text-destructive text-sm'>
            {error}
          </p>
        )}
      </ConfirmDialog>
    </Card>
  )
}
