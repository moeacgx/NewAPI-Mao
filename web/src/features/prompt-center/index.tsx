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
import { useQuery } from '@tanstack/react-query'
import { Search, Sparkles } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { getPromptTemplates } from './api'
import { PromptTemplateEditor } from './components/prompt-template-editor'
import { TokenPicker } from './components/token-picker'
import { TokenPromptDialog } from './components/token-prompt-dialog'
import type { PromptTemplate, PromptToken } from './types'

export function PromptCenter() {
  const { t } = useTranslation()
  const isAdmin = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.ADMIN
  )
  const [manage, setManage] = useState(false)
  const [search, setSearch] = useState('')
  const [detail, setDetail] = useState<PromptTemplate>()
  const [editing, setEditing] = useState<PromptTemplate | 'new'>()
  const [picker, setPicker] = useState<{ template?: PromptTemplate }>()
  const [installation, setInstallation] = useState<{
    token: PromptToken
    template?: PromptTemplate
  }>()
  const admin = isAdmin && manage
  const query = useQuery({
    queryKey: ['prompt-center', admin],
    queryFn: () => getPromptTemplates(admin),
  })
  const templates =
    query.data?.filter((template) =>
      `${template.title} ${template.description}`
        .toLocaleLowerCase()
        .includes(search.toLocaleLowerCase())
    ) ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Prompt Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex flex-wrap items-center gap-2'>
          <Button variant='outline' onClick={() => setPicker({})}>
            {t('Manage token prompts')}
          </Button>
          {isAdmin && (
            <>
              <Button
                variant={admin ? 'secondary' : 'outline'}
                onClick={() => setManage((value) => !value)}
              >
                {admin ? t('Browse prompts') : t('Manage templates')}
              </Button>
              <Button onClick={() => setEditing('new')}>
                {t('Create prompt')}
              </Button>
            </>
          )}
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <Card>
            <CardHeader className='border-b'>
              <CardTitle className='flex items-center gap-2'>
                <Sparkles className='size-4' aria-hidden='true' />
                {t('Reusable prompts for your API keys')}
              </CardTitle>
              <CardDescription>
                {t(
                  'Install text prompts to a token. The gateway loads them only when the actual request group is allowed.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Supports Chat Completions, Responses, Claude Messages and Gemini conversations. Markdown and SKILL.md are text only; scripts, attachments and tools do not run.'
                )}
              </p>
              <div className='relative'>
                <Search
                  className='text-muted-foreground absolute top-2.5 left-3 size-4'
                  aria-hidden='true'
                />
                <Input
                  className='pl-9'
                  aria-label={t('Search prompts')}
                  placeholder={t('Search prompts')}
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                />
              </div>
            </CardContent>
          </Card>
          {query.isPending && (
            <Card>
              <CardContent role='status'>{t('Loading...')}</CardContent>
            </Card>
          )}
          {query.isError && (
            <Card>
              <CardContent role='alert' className='space-y-3'>
                <p>{t('Failed to load prompts')}</p>
                <Button variant='outline' onClick={() => query.refetch()}>
                  {t('Retry')}
                </Button>
              </CardContent>
            </Card>
          )}
          {query.isSuccess && templates.length === 0 && (
            <Card>
              <CardContent className='text-muted-foreground py-8 text-center'>
                {t('No prompts found')}
              </CardContent>
            </Card>
          )}
          <div className='grid min-w-0 gap-4 md:grid-cols-2 xl:grid-cols-3'>
            {templates.map((template) => (
              <Card key={template.id} className='min-w-0 border'>
                <CardHeader>
                  <div className='flex items-start justify-between gap-2'>
                    <CardTitle className='[overflow-wrap:anywhere]'>
                      {template.title}
                    </CardTitle>
                    {!template.enabled && (
                      <Badge variant='secondary'>{t('Unpublished')}</Badge>
                    )}
                  </div>
                  <CardDescription className='line-clamp-3 [overflow-wrap:anywhere]'>
                    {template.description}
                  </CardDescription>
                </CardHeader>
                <CardContent className='flex flex-1 flex-col gap-3'>
                  <div className='flex flex-wrap gap-1'>
                    {template.group_details?.map((group) => (
                      <Badge
                        key={group.id}
                        variant='outline'
                        className='max-w-full [overflow-wrap:anywhere] whitespace-normal'
                      >
                        {group.name}
                      </Badge>
                    ))}
                  </div>
                  <div className='mt-auto flex flex-wrap gap-2 border-t pt-3'>
                    <Button
                      size='sm'
                      variant='outline'
                      onClick={() => setDetail(template)}
                    >
                      {t('Details')}
                    </Button>
                    <Button
                      size='sm'
                      disabled={!template.enabled}
                      onClick={() => setPicker({ template })}
                    >
                      {t('Install to token')}
                    </Button>
                    {admin && (
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={() => setEditing(template)}
                      >
                        {t('Edit')}
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
        {detail && (
          <Dialog
            open
            onOpenChange={(open) => {
              if (!open) setDetail(undefined)
            }}
          >
            <DialogContent className='max-h-[85dvh] overflow-y-auto sm:max-w-2xl'>
              <DialogHeader>
                <DialogTitle className='[overflow-wrap:anywhere]'>
                  {detail.title}
                </DialogTitle>
                <DialogDescription className='[overflow-wrap:anywhere]'>
                  {detail.description}
                </DialogDescription>
              </DialogHeader>
              <p className='text-muted-foreground text-sm'>
                {t('Allowed groups')}:{' '}
                {detail.group_details?.map((group) => group.name).join(', ') ||
                  t('No available groups')}
              </p>
              <pre className='bg-muted/40 max-h-[50dvh] overflow-y-auto rounded-lg border p-4 font-sans text-sm [overflow-wrap:anywhere] whitespace-pre-wrap'>
                {detail.content}
              </pre>
              <Button
                disabled={!detail.enabled}
                onClick={() => {
                  setPicker({ template: detail })
                  setDetail(undefined)
                }}
              >
                {t('Install to token')}
              </Button>
            </DialogContent>
          </Dialog>
        )}
        {isAdmin && editing && (
          <PromptTemplateEditor
            template={editing === 'new' ? undefined : editing}
            onClose={() => setEditing(undefined)}
          />
        )}
        {picker && (
          <TokenPicker
            onClose={() => setPicker(undefined)}
            onSelect={(token) => {
              setInstallation({ token, template: picker.template })
              setPicker(undefined)
            }}
          />
        )}
        {installation && (
          <TokenPromptDialog
            open
            token={installation.token}
            templateToInstall={installation.template}
            onOpenChange={(open) => {
              if (!open) setInstallation(undefined)
            }}
          />
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
