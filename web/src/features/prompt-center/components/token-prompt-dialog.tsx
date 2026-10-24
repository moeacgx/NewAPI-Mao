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
import { CheckCircle2, Layers3, Sparkles, Trash2 } from 'lucide-react'
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
import { SkillSummary } from './skill-summary'

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
      <DialogContent className='max-h-[85dvh] overflow-y-auto rounded-2xl sm:max-w-2xl'>
        <DialogHeader className='border-border/70 bg-muted/20 rounded-2xl border p-4'>
          <div className='flex items-start gap-3'>
            <div className='bg-primary/[0.1] text-primary flex size-10 shrink-0 items-center justify-center rounded-xl'>
              <Layers3 className='size-5' aria-hidden='true' />
            </div>
            <div className='min-w-0'>
              <DialogTitle className='truncate'>
                {t('Installed prompts')} · {props.token.name}
              </DialogTitle>
              <DialogDescription className='mt-1'>
                {t(
                  'Prompts load only for allowed request groups. Installed does not mean active for every group.'
                )}
              </DialogDescription>
            </div>
          </div>
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
  // 正文大小由后端按最终提交内容校验，用户端不读取正文。
  const invalid = bindings.length > 8

  return (
    <div className='space-y-3'>
      <p className='bg-muted/20 text-muted-foreground rounded-xl border p-3 text-xs leading-5'>
        {t(
          'Install up to 8 skill packages. Always-loaded prompts share a 65536-byte limit. Progressive skills add only the text read to model context; extra model rounds are billed.'
        )}
      </p>
      {invalid && (
        <p role='alert' className='text-destructive'>
          {t('Install at most 8 skill packages per token.')}
        </p>
      )}
      {bindings.length === 0 && (
        <div className='bg-muted/20 flex flex-col items-center gap-2 rounded-2xl border border-dashed px-6 py-10 text-center'>
          <div className='bg-background flex size-11 items-center justify-center rounded-xl border'>
            <Sparkles
              className='text-muted-foreground size-5'
              aria-hidden='true'
            />
          </div>
          <p className='font-medium'>{t('No installed prompts')}</p>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Install prompts without changing the key, quota or group permissions.'
            )}
          </p>
        </div>
      )}
      <div className='space-y-3'>
        {bindings.map((binding) => (
          <section
            key={binding.template.id}
            className='bg-background border-border/70 space-y-3 rounded-2xl border p-4 shadow-sm'
            aria-label={binding.template.title}
          >
            <div className='flex flex-wrap items-start justify-between gap-2'>
              <div className='min-w-0 flex-1'>
                <div className='flex items-center gap-2'>
                  <Sparkles
                    className='text-primary size-4 shrink-0'
                    aria-hidden='true'
                  />
                  <h3 className='font-medium [overflow-wrap:anywhere]'>
                    {binding.template.title}
                  </h3>
                </div>
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
            <SkillSummary template={binding.template} />
            <div className='border-border/60 flex flex-wrap items-center gap-3 border-t pt-3'>
              <div className='flex min-w-0 flex-1 flex-wrap gap-1.5'>
                {binding.template.group_details?.length ? (
                  binding.template.group_details.map((group) => (
                    <span
                      key={group.id}
                      className='bg-muted/40 text-muted-foreground inline-flex max-w-full items-center gap-1 rounded-md px-2 py-1 text-[11px]'
                    >
                      <CheckCircle2
                        className='text-primary size-3'
                        aria-hidden='true'
                      />
                      <span className='truncate'>{group.name}</span>
                    </span>
                  ))
                ) : (
                  <span className='text-muted-foreground text-xs'>
                    {t('No available groups')}
                  </span>
                )}
              </div>
            </div>
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
