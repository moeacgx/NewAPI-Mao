/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import {
  ArrowUpRight,
  BookOpen,
  CheckCircle2,
  FolderTree,
  KeyRound,
  Layers3,
  Plus,
  Package,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { getPromptCategories, getPromptTemplates } from './api'
import { CategoryManager } from './components/category-manager'
import { PromptTemplateEditor } from './components/prompt-template-editor'
import { SkillDeleteButton } from './components/skill-delete-button'
import { SkillImportDialog } from './components/skill-import-dialog'
import { SkillRepositoryUpdate } from './components/skill-repository-update'
import { SkillSummary } from './components/skill-summary'
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
  const [category, setCategory] = useState('all')
  const [kind, setKind] = useState('all')
  const [categoryManager, setCategoryManager] = useState(false)
  const [importing, setImporting] = useState(false)
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
  const allTemplates = query.data ?? []
  const categories = useQuery({
    queryKey: ['prompt-categories'],
    queryFn: getPromptCategories,
  })
  const templates = useMemo(
    () =>
      query.data?.filter(
        (template) =>
          `${template.title} ${template.description}`
            .toLocaleLowerCase()
            .includes(search.toLocaleLowerCase()) &&
          (category === 'all' ||
            (template.category_id ?? 0) === Number(category)) &&
          (kind === 'all' || (template.kind ?? 'prompt') === kind)
      ) ?? [],
    [query.data, search, category, kind]
  )
  const publishedCount = allTemplates.filter(
    (template) => template.enabled
  ).length
  const groupCount = new Set(
    allTemplates.flatMap((template) =>
      (template.group_details ?? []).map((group) => group.id)
    )
  ).size

  return (
    <SectionPageLayout>
      <SectionPageLayout.Content>
        <Card className='border-border/70 bg-card mx-auto w-full overflow-hidden rounded-3xl shadow-sm'>
          <CardHeader className='border-border/70 bg-muted/20 relative overflow-hidden border-b px-5 py-6 sm:px-8 sm:py-8'>
            <div className='relative flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between'>
              <div className='flex min-w-0 items-start gap-4'>
                <div className='bg-primary text-primary-foreground shadow-primary/20 flex size-12 shrink-0 items-center justify-center rounded-2xl shadow-lg sm:size-14'>
                  <Sparkles className='size-6 sm:size-7' aria-hidden='true' />
                </div>
                <div className='min-w-0 space-y-2'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Badge variant='secondary' className='rounded-full px-2.5'>
                      <BookOpen aria-hidden='true' />
                      {t('Prompt Center')}
                    </Badge>
                    <Badge variant='outline' className='rounded-full'>
                      {t('Prompt')}
                    </Badge>
                  </div>
                  <h1 className='text-2xl font-semibold tracking-tight sm:text-3xl'>
                    {t('Reusable prompts for your API keys')}
                  </h1>
                  <CardDescription className='max-w-2xl text-sm leading-6 sm:text-base'>
                    {t(
                      'Install text prompts to a token. The gateway loads them only when the actual request group is allowed.'
                    )}
                  </CardDescription>
                </div>
              </div>
              <div className='flex flex-col gap-4 xl:items-end'>
                <div className='flex flex-wrap items-center gap-2'>
                  <Button variant='outline' onClick={() => setPicker({})}>
                    <KeyRound aria-hidden='true' />
                    {t('Manage token prompts')}
                  </Button>
                  {isAdmin && (
                    <>
                      <Button
                        variant='outline'
                        onClick={() => setImporting(true)}
                      >
                        <Package aria-hidden='true' />
                        {t('Skill store')}
                      </Button>
                      <Button
                        variant='outline'
                        onClick={() => setCategoryManager(true)}
                      >
                        <FolderTree aria-hidden='true' />
                        {t('Manage skill categories')}
                      </Button>
                      <Button
                        variant='outline'
                        onClick={() => setManage((value) => !value)}
                      >
                        <SlidersHorizontal aria-hidden='true' />
                        {admin ? t('Browse prompts') : t('Manage templates')}
                      </Button>
                      <Button onClick={() => setEditing('new')}>
                        <Plus aria-hidden='true' />
                        {t('Create prompt')}
                      </Button>
                    </>
                  )}
                </div>
                <div className='grid w-full grid-cols-3 gap-2 sm:gap-3 xl:min-w-[20rem]'>
                  <Stat label={t('Templates')} value={allTemplates.length} />
                  <Stat label={t('Published')} value={publishedCount} />
                  <Stat label={t('Allowed groups')} value={groupCount} />
                </div>
              </div>
            </div>
            <div className='border-border/60 relative mt-7 flex flex-col gap-3 border-t pt-5 sm:flex-row sm:items-center sm:justify-between'>
              <p className='text-muted-foreground flex items-start gap-2 text-xs leading-5 sm:max-w-xl'>
                <ShieldCheck
                  className='text-primary mt-0.5 size-4 shrink-0'
                  aria-hidden='true'
                />
                <span>
                  {t(
                    'Supports Chat Completions, Responses, Claude Messages and Gemini conversations. Skills and text references are read on demand by the gateway; skill scripts are not executed.'
                  )}
                </span>
              </p>
              <div className='relative w-full sm:max-w-xs'>
                <Search
                  className='text-muted-foreground absolute top-2.5 left-3.5 size-4'
                  aria-hidden='true'
                />
                <Input
                  className='border-border/70 bg-background/80 h-9 rounded-xl pl-10 shadow-none'
                  aria-label={t('Search prompts')}
                  placeholder={t('Search prompts')}
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                />
              </div>
            </div>
          </CardHeader>
          <CardContent className='space-y-5 px-5 py-6 sm:px-8 sm:py-8'>
            <div className='bg-muted/20 flex flex-wrap items-end gap-3 rounded-xl border p-3'>
              <label className='min-w-40 flex-1 space-y-1.5 text-xs font-medium'>
                <span>{t('Skill category')}</span>
                <select
                  aria-label={t('Skill category')}
                  className='bg-background h-9 w-full rounded-lg border px-3 text-sm'
                  value={category}
                  onChange={(event) => setCategory(event.target.value)}
                >
                  <option value='all'>{t('All skill categories')}</option>
                  <option value='0'>{t('Uncategorized skills')}</option>
                  {categories.data?.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className='min-w-40 flex-1 space-y-1.5 text-xs font-medium'>
                <span>{t('Skill type')}</span>
                <select
                  aria-label={t('Skill type')}
                  className='bg-background h-9 w-full rounded-lg border px-3 text-sm'
                  value={kind}
                  onChange={(event) => setKind(event.target.value)}
                >
                  <option value='all'>{t('All skill types')}</option>
                  <option value='prompt'>{t('Single skill')}</option>
                  <option value='bundle'>{t('Skill bundle')}</option>
                </select>
              </label>
              {categories.isError && (
                <div
                  role='alert'
                  className='text-destructive flex items-center gap-2 text-xs'
                >
                  <span>{t('Failed to load skill categories')}</span>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => categories.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </div>
              )}
            </div>
            <div className='flex flex-wrap items-center justify-between gap-3'>
              <div className='flex items-center gap-2'>
                <div className='bg-primary/[0.09] text-primary flex size-8 items-center justify-center rounded-lg'>
                  <Layers3 className='size-4' aria-hidden='true' />
                </div>
                <h2 className='text-base font-semibold tracking-tight'>
                  {t('Templates')}
                </h2>
              </div>
              {query.isSuccess && (
                <Badge variant='outline' className='rounded-full px-2.5'>
                  {templates.length}
                </Badge>
              )}
            </div>
            {query.isPending && (
              <div
                className='bg-muted/20 flex items-center gap-3 rounded-2xl border border-dashed p-8 text-sm'
                role='status'
              >
                <Sparkles
                  className='text-primary size-4 animate-pulse'
                  aria-hidden='true'
                />
                {t('Loading...')}
              </div>
            )}
            {query.isError && (
              <div
                className='bg-destructive/5 border-destructive/20 flex flex-col gap-3 rounded-2xl border p-5 sm:flex-row sm:items-center sm:justify-between'
                role='alert'
              >
                <p>{t('Failed to load prompts')}</p>
                <Button variant='outline' onClick={() => query.refetch()}>
                  {t('Retry')}
                </Button>
              </div>
            )}
            {query.isSuccess && templates.length === 0 && (
              <div className='bg-muted/20 flex flex-col items-center justify-center gap-2 rounded-2xl border border-dashed px-6 py-16 text-center'>
                <div className='bg-background flex size-12 items-center justify-center rounded-2xl border shadow-sm'>
                  <Layers3
                    className='text-muted-foreground size-5'
                    aria-hidden='true'
                  />
                </div>
                <p className='font-medium'>{t('No prompts found')}</p>
                <p className='text-muted-foreground text-sm'>
                  {t('Search prompts')}
                </p>
              </div>
            )}
            {query.isSuccess && templates.length > 0 && (
              <div className='grid min-w-0 gap-4 md:grid-cols-2 xl:grid-cols-3'>
                {templates.map((template) => (
                  <PromptTemplateCard
                    key={template.id}
                    template={template}
                    admin={admin}
                    onInstall={() => setPicker({ template })}
                    onEdit={() => setEditing(template)}
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {isAdmin && editing && (
          <PromptTemplateEditor
            template={editing === 'new' ? undefined : editing}
            onClose={() => setEditing(undefined)}
          />
        )}
        {isAdmin && categoryManager && (
          <CategoryManager onClose={() => setCategoryManager(false)} />
        )}
        {isAdmin && importing && (
          <SkillImportDialog
            onClose={() => setImporting(false)}
            onImported={(template) => {
              setImporting(false)
              setEditing(template)
            }}
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

function Stat(props: { label: string; value: number }) {
  return (
    <div className='border-border/60 border-l px-3 py-1 first:border-l-0 sm:px-4'>
      <p className='text-muted-foreground truncate text-[11px] font-medium tracking-wide uppercase'>
        {props.label}
      </p>
      <p className='mt-1 text-xl font-semibold tracking-tight sm:text-2xl'>
        {props.value}
      </p>
    </div>
  )
}

function PromptTemplateCard(props: {
  template: PromptTemplate
  admin: boolean
  onInstall: () => void
  onEdit: () => void
}) {
  const { t } = useTranslation()
  const groups = props.template.group_details ?? []
  return (
    <article className='group border-border/70 bg-background hover:border-primary/30 flex min-w-0 flex-col rounded-2xl border p-5 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md'>
      <div className='flex items-start justify-between gap-3'>
        <h3 className='min-w-0 truncate text-base font-semibold tracking-tight'>
          {props.template.title}
        </h3>
        <Badge
          variant={props.template.enabled ? 'secondary' : 'outline'}
          className='rounded-full'
        >
          {props.template.enabled ? <CheckCircle2 aria-hidden='true' /> : null}
          {props.template.enabled ? t('Published') : t('Unpublished')}
        </Badge>
      </div>
      <div className='min-w-0'>
        <p className='text-muted-foreground mt-2 line-clamp-3 min-h-[3.75rem] text-sm leading-5 [overflow-wrap:anywhere]'>
          {props.template.description || t('Prompt')}
        </p>
      </div>
      <div className='mb-4'>
        <SkillSummary template={props.template} />
      </div>
      {props.admin && props.template.repository_source && (
        <div className='mb-4'>
          <SkillRepositoryUpdate template={props.template} />
        </div>
      )}
      <div className='border-border/60 mt-auto flex flex-wrap items-end justify-between gap-4 border-t pt-4'>
        <div className='min-w-0 flex-1 basis-32'>
          <div className='text-muted-foreground mb-2 flex items-center gap-1.5 text-xs font-medium'>
            <ShieldCheck className='text-primary size-3.5' aria-hidden='true' />
            {t('Allowed groups')}
          </div>
          <div className='flex min-h-6 flex-wrap gap-1.5'>
            {groups.length > 0 ? (
              groups.map((group) => (
                <Badge
                  key={group.id}
                  variant='outline'
                  className='bg-muted/30 h-auto max-w-full rounded-md px-2 py-1 text-[11px] whitespace-normal'
                >
                  {group.name}
                </Badge>
              ))
            ) : (
              <span className='text-muted-foreground text-xs'>
                {t('No available groups')}
              </span>
            )}
          </div>
        </div>
        <div className='ml-auto flex flex-wrap justify-end gap-2'>
          <Button
            size='sm'
            disabled={!props.template.enabled}
            onClick={props.onInstall}
          >
            <ArrowUpRight aria-hidden='true' />
            {t('Install to token')}
          </Button>
          {props.admin && (
            <>
              <Button size='sm' variant='outline' onClick={props.onEdit}>
                <SlidersHorizontal aria-hidden='true' />
                {t('Edit')}
              </Button>
              <SkillDeleteButton template={props.template} />
            </>
          )}
        </div>
      </div>
    </article>
  )
}
