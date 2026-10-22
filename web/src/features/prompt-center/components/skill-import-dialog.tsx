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
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  Download,
  Link2,
  Package,
  RefreshCw,
  Search,
  Upload,
} from 'lucide-react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import {
  getSkillStore,
  importSkillArchive,
  importSkillRepository,
  importSkillStoreEntry,
} from '../api'
import type { PromptTemplate, SkillStoreEntry } from '../types'

export function SkillImportDialog(props: {
  onClose: () => void
  onImported: (template: PromptTemplate) => void
}) {
  const { t } = useTranslation()
  const fileId = useId()
  const repositoryId = useId()
  const [repositoryUrl, setRepositoryUrl] = useState('')
  const [search, setSearch] = useState('')
  const [onlineEntries, setOnlineEntries] = useState<SkillStoreEntry[]>()
  const query = useQuery({
    queryKey: ['prompt-skill-store'],
    queryFn: () => getSkillStore(),
  })
  const online = useMutation({
    mutationFn: () => getSkillStore(true),
    onSuccess: setOnlineEntries,
  })
  const importing = useMutation({
    mutationFn: (input: { id: string } | { file: File } | { url: string }) => {
      if ('id' in input) return importSkillStoreEntry(input.id)
      if ('url' in input) return importSkillRepository(input.url)
      return importSkillArchive(input.file)
    },
    onSuccess: (template) => props.onImported(template),
  })
  const entries = (onlineEntries ?? query.data ?? []).filter((entry) =>
    `${entry.title} ${entry.description} ${entry.category}`
      .toLocaleLowerCase()
      .includes(search.toLocaleLowerCase())
  )
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !importing.isPending) props.onClose()
      }}
    >
      <DialogContent
        showCloseButton={!importing.isPending}
        className='max-h-[90dvh] overflow-y-auto rounded-2xl sm:max-w-3xl'
      >
        <DialogHeader>
          <DialogTitle>{t('Skill store')}</DialogTitle>
          <DialogDescription>
            {t(
              'Import a copy, review its text and choose permissions before publishing. Scripts and tools do not run.'
            )}
          </DialogDescription>
        </DialogHeader>
        <fieldset disabled={importing.isPending} className='space-y-4'>
          <form
            className='bg-muted/20 space-y-3 rounded-xl border p-4'
            onSubmit={(event) => {
              event.preventDefault()
              if (!importing.isPending && repositoryUrl.trim()) {
                importing.mutate({ url: repositoryUrl })
              }
            }}
          >
            <Label htmlFor={repositoryId} className='flex items-center gap-2'>
              <Link2 className='text-primary size-4' aria-hidden='true' />
              {t('Import from GitHub repository')}
            </Label>
            <div className='flex flex-col gap-2 sm:flex-row'>
              <Input
                id={repositoryId}
                type='url'
                required
                placeholder='https://github.com/owner/repository'
                value={repositoryUrl}
                onChange={(event) => setRepositoryUrl(event.target.value)}
              />
              <Button
                type='submit'
                variant='outline'
                disabled={!repositoryUrl.trim() || importing.isPending}
              >
                {t('Import repository')}
              </Button>
            </div>
            <p className='text-muted-foreground text-xs'>
              {t(
                'Use a public GitHub repository, directory or SKILL.md link. Only skill text is imported; scripts and tools do not run.'
              )}
            </p>
          </form>
          <div className='bg-muted/20 space-y-2 rounded-xl border p-4'>
            <Label htmlFor={fileId} className='flex items-center gap-2'>
              <Upload className='text-primary size-4' aria-hidden='true' />
              {t('Import skill ZIP archive')}
            </Label>
            <Input
              id={fileId}
              type='file'
              accept='.zip,application/zip'
              onChange={(event) => {
                const file = event.target.files?.[0]
                if (file) importing.mutate({ file })
                event.target.value = ''
              }}
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'ZIP files are checked on the server. Only skill text is imported.'
              )}
            </p>
          </div>
          <div className='flex flex-col gap-2 sm:flex-row'>
            <div className='relative flex-1'>
              <Search
                className='text-muted-foreground absolute top-2.5 left-3 size-4'
                aria-hidden='true'
              />
              <Input
                className='pl-9'
                aria-label={t('Search skill store')}
                placeholder={t('Search skill store')}
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
            </div>
            <Button
              variant='outline'
              disabled={online.isPending}
              onClick={() => online.mutate()}
            >
              <RefreshCw aria-hidden='true' />
              {t('Refresh online store')}
            </Button>
          </div>
          {(query.isPending || online.isPending || importing.isPending) && (
            <p
              role='status'
              className='bg-muted/20 rounded-lg border p-3 text-sm'
            >
              {t('Loading...')}
            </p>
          )}
          {query.isError && !onlineEntries && (
            <div role='alert' className='rounded-lg border p-3'>
              <p>{t('Failed to load skill store')}</p>
              <Button variant='outline' onClick={() => query.refetch()}>
                {t('Retry')}
              </Button>
            </div>
          )}
          {online.isError && (
            <p
              role='alert'
              className='bg-destructive/5 text-destructive rounded-lg border p-3 text-sm'
            >
              {t(
                'Online refresh failed. Your current store list is still available.'
              )}
            </p>
          )}
          {importing.isError && (
            <p
              role='alert'
              className='bg-destructive/5 text-destructive rounded-lg border p-3 text-sm'
            >
              {t(importing.error.message)}
            </p>
          )}
          {query.isSuccess && entries.length === 0 && (
            <p className='bg-muted/20 text-muted-foreground rounded-xl border border-dashed p-8 text-center'>
              {t('No skills found in store')}
            </p>
          )}
          <div className='grid gap-3 sm:grid-cols-2'>
            {entries.map((entry) => (
              <article
                key={entry.id}
                className='bg-background flex min-w-0 flex-col gap-3 rounded-2xl border p-4 shadow-sm'
              >
                <div className='flex items-start gap-2'>
                  <Package
                    className='text-primary mt-0.5 size-4 shrink-0'
                    aria-hidden='true'
                  />
                  <h3 className='font-medium [overflow-wrap:anywhere]'>
                    {entry.title}
                  </h3>
                </div>
                <p className='text-muted-foreground text-sm [overflow-wrap:anywhere]'>
                  {entry.description}
                </p>
                <div className='flex flex-wrap gap-1.5'>
                  <Badge variant='secondary'>
                    {entry.kind === 'bundle'
                      ? t('Skill bundle')
                      : t('Single skill')}
                  </Badge>
                  <Badge variant='outline'>{entry.category}</Badge>
                  <Badge variant='outline'>
                    {entry.source === 'online'
                      ? t('Online catalog')
                      : t('Bundled catalog')}
                  </Badge>
                </div>
                <p className='text-muted-foreground text-xs'>
                  {t('{{count}} skills in this bundle', {
                    count: entry.skill_count,
                  })}{' '}
                  · {entry.version}
                </p>
                <Button
                  className='mt-auto self-end'
                  size='sm'
                  onClick={() => importing.mutate({ id: entry.id })}
                >
                  <Download aria-hidden='true' />
                  {t('Import and edit')}
                </Button>
              </article>
            ))}
          </div>
        </fieldset>
      </DialogContent>
    </Dialog>
  )
}
