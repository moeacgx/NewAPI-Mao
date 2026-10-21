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
import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { getTokenPrompts, saveTokenPrompts } from '../api'
import type { PromptTemplate, PromptToken, TokenPromptBinding } from '../types'

export interface TokenPromptDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  token: PromptToken
  templateToInstall?: PromptTemplate
}

export function TokenPromptDialog(props: TokenPromptDialogProps) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['token-prompts', props.token.id],
    queryFn: () => getTokenPrompts(props.token.id),
    enabled: props.open,
    staleTime: 0,
    refetchOnMount: 'always',
    refetchOnWindowFocus: false,
  })

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[85dvh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {t('Installed prompts')} · {props.token.name}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Prompts load only for allowed request groups. Installed does not mean active for every group.'
            )}
          </DialogDescription>
        </DialogHeader>
        {(query.isPending || !query.isFetchedAfterMount) && (
          <p role='status'>{t('Loading...')}</p>
        )}
        {query.isError && (
          <div
            className='bg-destructive/5 space-y-2 rounded-lg border p-3'
            role='alert'
          >
            <p>{t('Failed to load installed prompts')}</p>
            <Button variant='outline' onClick={() => query.refetch()}>
              {t('Retry')}
            </Button>
          </div>
        )}
        {query.isSuccess && query.isFetchedAfterMount && (
          <TokenPromptEditor
            key={props.token.id}
            bindings={query.data}
            token={props.token}
            templateToInstall={props.templateToInstall}
            onSaved={() => props.onOpenChange(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

function TokenPromptEditor(props: {
  bindings: TokenPromptBinding[]
  token: PromptToken
  templateToInstall?: PromptTemplate
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [bindings, setBindings] = useState<TokenPromptBinding[]>(() => {
    const sorted = [...props.bindings].sort(
      (a, b) => a.sort_order - b.sort_order || a.template.id - b.template.id
    )
    const candidate = props.templateToInstall
    if (
      !candidate ||
      sorted.some((item) => item.template.id === candidate.id)
    ) {
      return sorted
    }
    return [
      ...sorted,
      { template: candidate, enabled: true, sort_order: sorted.length },
    ]
  })
  const mutation = useMutation({
    mutationFn: () => saveTokenPrompts(props.token.id, bindings),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['token-prompts', props.token.id],
        }),
        queryClient.invalidateQueries({ queryKey: ['keys'] }),
        queryClient.invalidateQueries({ queryKey: ['prompt-center-tokens'] }),
      ])
      props.onSaved()
    },
  })
  const totalBytes = bindings
    .filter((binding) => binding.enabled && binding.template.enabled)
    .reduce(
      (sum, binding) =>
        sum + new TextEncoder().encode(binding.template.content).length,
      0
    )
  const invalid = bindings.length > 8 || totalBytes > 65536

  return (
    <div className='space-y-3'>
      {invalid && (
        <p role='alert' className='text-destructive'>
          {t(
            'Install at most 8 prompts with up to 65536 enabled bytes in total.'
          )}
        </p>
      )}
      {bindings.length === 0 && (
        <div className='bg-muted/30 rounded-lg border p-6 text-center'>
          {t('No installed prompts')}
        </div>
      )}
      <div className='space-y-3'>
        {bindings.map((binding, index) => (
          <section
            key={binding.template.id}
            className='bg-card space-y-2 rounded-lg border p-3'
            aria-label={binding.template.title}
          >
            <div className='flex flex-wrap items-start justify-between gap-2'>
              <div className='min-w-0 flex-1'>
                <h3 className='font-medium [overflow-wrap:anywhere]'>
                  {binding.template.title}
                </h3>
                {!binding.template.enabled && (
                  <span className='text-muted-foreground text-xs'>
                    {t('Unpublished')}
                  </span>
                )}
              </div>
              <div className='flex shrink-0 gap-1'>
                <Button
                  size='icon-sm'
                  variant='outline'
                  disabled={index === 0 || mutation.isPending}
                  aria-label={t('Move {{title}} up', {
                    title: binding.template.title,
                  })}
                  onClick={() =>
                    setBindings((current) => {
                      const next = [...current]
                      const previous = next[index - 1]
                      if (!previous) return current
                      next[index - 1] = binding
                      next[index] = previous
                      return next
                    })
                  }
                >
                  <ArrowUp className='size-3.5' aria-hidden='true' />
                </Button>
                <Button
                  size='icon-sm'
                  variant='outline'
                  disabled={index === bindings.length - 1 || mutation.isPending}
                  aria-label={t('Move {{title}} down', {
                    title: binding.template.title,
                  })}
                  onClick={() =>
                    setBindings((current) => {
                      const next = [...current]
                      const following = next[index + 1]
                      if (!following) return current
                      next[index] = following
                      next[index + 1] = binding
                      return next
                    })
                  }
                >
                  <ArrowDown className='size-3.5' aria-hidden='true' />
                </Button>
                <Button
                  size='icon-sm'
                  variant='outline'
                  disabled={mutation.isPending}
                  aria-label={t('Uninstall {{title}}', {
                    title: binding.template.title,
                  })}
                  onClick={() =>
                    setBindings((current) =>
                      current.filter(
                        (item) => item.template.id !== binding.template.id
                      )
                    )
                  }
                >
                  <Trash2 className='size-3.5' aria-hidden='true' />
                </Button>
              </div>
            </div>
            <p className='text-muted-foreground text-xs'>
              {t('Allowed groups')}:{' '}
              {binding.template.group_details
                ?.map((group) => group.name)
                .join(', ') || t('No available groups')}
            </p>
            <label className='flex items-center gap-2 text-sm'>
              <input
                type='checkbox'
                aria-label={t('Enable {{title}}', {
                  title: binding.template.title,
                })}
                checked={binding.enabled}
                disabled={
                  mutation.isPending ||
                  (!binding.template.enabled && !binding.enabled)
                }
                onChange={(event) => {
                  const enabled = event.target.checked
                  setBindings((current) =>
                    current.map((item) =>
                      item.template.id === binding.template.id
                        ? { ...item, enabled }
                        : item
                    )
                  )
                }}
              />
              {t('Enabled')}
            </label>
            <details>
              <summary className='cursor-pointer text-xs'>
                {t('View prompt content')}
              </summary>
              <pre className='bg-muted/40 mt-2 max-h-48 overflow-y-auto rounded-md p-3 font-sans text-xs [overflow-wrap:anywhere] whitespace-pre-wrap'>
                {binding.template.content}
              </pre>
            </details>
          </section>
        ))}
      </div>
      {mutation.isError && (
        <p role='alert' className='text-destructive'>
          {t(mutation.error.message)}
        </p>
      )}
      <div className='flex justify-end border-t pt-3'>
        <Button
          disabled={invalid || mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          {t('Save installations')}
        </Button>
      </div>
    </div>
  )
}
