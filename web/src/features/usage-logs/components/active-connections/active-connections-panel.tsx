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
import {
  Activity,
  Box,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Radio,
  RefreshCw,
} from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { getLobeIcon } from '@/lib/lobe-icon'
import { resolveModelProvider } from '@/lib/model-provider'
import { cn } from '@/lib/utils'

import {
  type ActiveConnection,
  type ConnectionSubject,
  useActiveConnections,
} from './use-active-connections'

const statusKeys = {
  waiting_response: 'Waiting for response',
  receiving: 'Receiving response',
  processing: 'Processing request',
  retrying: 'Retrying request',
  completed: 'Request ended',
} as const

const protocolLabels: Record<string, string> = {
  openai: 'OpenAI Chat Completions',
  openai_responses: 'OpenAI Responses',
  claude: 'Claude Messages',
  openai_responses_compaction: 'OpenAI Responses Compaction',
  openai_alpha_search: 'OpenAI Alpha Search',
  openai_audio: 'OpenAI Audio',
  openai_image: 'OpenAI Images',
  openai_realtime: 'OpenAI Realtime',
  rerank: 'Rerank',
  embedding: 'Embeddings',
  mj_proxy: 'Midjourney',
  gemini: 'Gemini',
}

function ConnectionRow({
  request,
  now,
  showRequestId = false,
}: {
  request: ActiveConnection
  now: number
  showRequestId?: boolean
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const detailsId = useId()
  const elapsed = `${(Math.max(0, (request.completed_at || now) - request.started_at) / 1000).toFixed(1)}s`
  const ended = request.status === 'completed'
  const provider = resolveModelProvider(request.model_name || '')
  return (
    <div className='bg-muted/20 min-w-0 rounded-lg border'>
      <button
        type='button'
        className='flex w-full min-w-0 items-center gap-2 px-3 py-2 text-left focus-visible:ring-2 focus-visible:outline-none sm:py-3'
        aria-expanded={expanded}
        aria-controls={detailsId}
        aria-label={t('Connection details for {{model}}', {
          model: request.model_name || t('Unknown model'),
        })}
        onClick={() => setExpanded(!expanded)}
      >
        <span
          className='flex size-[18px] shrink-0 items-center justify-center'
          aria-hidden='true'
          title={provider?.label ?? provider?.name}
        >
          {provider ? (
            getLobeIcon(provider.icon, 18)
          ) : (
            <Box className='text-muted-foreground size-[18px]' />
          )}
        </span>
        <span
          className='min-w-0 flex-1 truncate text-sm font-medium'
          title={request.model_name}
        >
          {request.model_name || t('Unknown model')}
        </span>
        <span
          className={cn(
            'shrink-0 rounded px-1.5 py-0.5 text-[10px]',
            ended
              ? 'bg-muted text-muted-foreground'
              : 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
          )}
        >
          {t(statusKeys[request.status] || 'Processing request')}
        </span>
        <span
          className='shrink-0 text-xs tabular-nums'
          title={t('Elapsed time')}
        >
          {elapsed}
        </span>
        <ChevronDown
          className={cn(
            'text-muted-foreground size-3 shrink-0 transition-transform sm:hidden',
            expanded && 'rotate-180'
          )}
        />
      </button>
      <div
        id={detailsId}
        className={cn(
          'text-muted-foreground space-y-2 border-t px-3 py-2 text-xs',
          expanded ? 'block' : 'hidden sm:block'
        )}
      >
        <div className='flex min-w-0 items-center justify-between gap-3'>
          <span className='truncate' title={request.token_name}>
            {request.token_name || t('Unnamed token')}
          </span>
          <span className='shrink-0'>
            {request.first_response_ms == null
              ? t(ended ? 'No response received' : 'Waiting for response')
              : `${t('First response')}: ${(request.first_response_ms / 1000).toFixed(2)}s`}
          </span>
        </div>
        <div className='flex min-w-0 items-center justify-between gap-2'>
          <span className='truncate'>
            {request.request_type === 'task'
              ? t('Task request')
              : protocolLabels[request.request_type] || request.request_type}
          </span>
          <span className='flex shrink-0 items-center gap-1'>
            <Radio className='size-3' />
            {t(
              request.is_stream ? 'Streaming request' : 'Non-streaming request'
            )}
          </span>
        </div>
        {showRequestId && request.request_id && (
          <div className='break-all'>
            {t('Request ID')}: {request.request_id}
          </div>
        )}
        {request.retry_count > 0 && (
          <div>
            {t('Retry count')}: {request.retry_count}
          </div>
        )}
      </div>
    </div>
  )
}

function ConnectionNotice({
  error,
  stale,
  retry,
  scope,
}: {
  error: boolean
  stale: boolean
  retry: () => void
  scope?: string
}) {
  const { t } = useTranslation()
  if (error) {
    return (
      <div
        role='alert'
        className='bg-muted/40 flex items-center justify-between gap-2 rounded-lg border p-3 text-xs'
      >
        <span>
          {t(
            stale
              ? 'Connection status is stale. Updates are unavailable.'
              : 'Connection status is unavailable.'
          )}
        </span>
        <Button variant='outline' size='sm' onClick={retry}>
          <RefreshCw />
          {t('Retry')}
        </Button>
      </div>
    )
  }
  return scope === 'instance' ? (
    <p className='text-muted-foreground text-xs'>
      {t('Only connections on the current instance are shown.')}
    </p>
  ) : null
}

function ConnectionsDrawer({
  subject,
  open,
  onOpenChange,
  title,
}: {
  subject?: ConnectionSubject
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
}) {
  const { t } = useTranslation()
  const [filters, setFilters] = useState({ search: '', status: '', page: 1 })
  const [searchInput, setSearchInput] = useState('')
  useEffect(() => {
    const timer = window.setTimeout(
      () =>
        setFilters((current) => ({
          ...current,
          search: searchInput.trim(),
          page: 1,
        })),
      300
    )
    return () => window.clearTimeout(timer)
  }, [searchInput])
  const { snapshot, now, loading, error, refresh } = useActiveConnections({
    subject,
    ...filters,
    pageSize: 10,
    enabled: open,
  })
  const pages = Math.max(1, Math.ceil((snapshot?.total || 0) / 10))
  // 轮询期间连接可能结束；页码越界时回到最后一页，避免误显示为空。
  useEffect(() => {
    if (snapshot && filters.page > pages && !error) {
      setFilters((current) => ({ ...current, page: pages }))
    }
  }, [snapshot, filters.page, pages, error])
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='w-full sm:max-w-2xl'>
        <SheetHeader className='border-b pr-12'>
          <SheetTitle>{t('Current connections')}</SheetTitle>
          <SheetDescription>{title}</SheetDescription>
        </SheetHeader>
        <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-auto px-4 pb-4'>
          <div className='bg-muted/20 flex flex-wrap gap-2 rounded-lg border p-3'>
            <Input
              className='min-w-0 flex-1 basis-40'
              placeholder={t('Search model, token or request ID')}
              aria-label={t('Search connections')}
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
            />
            <Select
              value={filters.status || 'all'}
              onValueChange={(value) =>
                setFilters({
                  ...filters,
                  status: value === 'all' ? '' : value || '',
                  page: 1,
                })
              }
            >
              <SelectTrigger aria-label={t('Connection status')}>
                <SelectValue>
                  {filters.status
                    ? t(statusKeys[filters.status as keyof typeof statusKeys])
                    : t('All statuses')}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>{t('All statuses')}</SelectItem>
                {Object.entries(statusKeys).map(([value, key]) => (
                  <SelectItem key={value} value={value}>
                    {t(key)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <ConnectionNotice
            error={!!error}
            stale={!!snapshot}
            retry={() => void refresh()}
            scope={snapshot?.scope}
          />
          {loading && (
            <p role='status' className='text-muted-foreground text-sm'>
              {t('Loading connection status…')}
            </p>
          )}
          {snapshot && (
            <div className='grid gap-2'>
              {snapshot.items.map((request) => (
                <ConnectionRow
                  key={request.id}
                  request={request}
                  now={now}
                  showRequestId
                />
              ))}
            </div>
          )}
          {!loading && !error && snapshot?.total === 0 && (
            <p className='text-muted-foreground rounded-lg border p-4 text-sm'>
              {t('No matching connections')}
            </p>
          )}
          <div className='mt-auto flex items-center justify-end gap-2 border-t pt-3'>
            <Button
              variant='outline'
              size='icon-sm'
              aria-label={t('Previous connections page')}
              disabled={filters.page <= 1 || loading}
              onClick={() => setFilters({ ...filters, page: filters.page - 1 })}
            >
              <ChevronLeft />
            </Button>
            <span className='text-muted-foreground text-xs tabular-nums'>
              {filters.page} / {pages}
            </span>
            <Button
              variant='outline'
              size='icon-sm'
              aria-label={t('Next connections page')}
              disabled={filters.page >= pages || loading}
              onClick={() => setFilters({ ...filters, page: filters.page + 1 })}
            >
              <ChevronRight />
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}

function ConnectionsForSubject({ subject }: { subject?: ConnectionSubject }) {
  const { t } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)
  const [open, setOpen] = useState(false)
  const { snapshot, now, loading, error, refresh } = useActiveConnections({
    subject,
  })
  const targeted = !!(subject?.userId || subject?.username)
  let title = t('Current connections')
  if (snapshot?.subject) {
    title = t('Connections for {{username}} (ID: {{id}})', snapshot.subject)
  } else if (subject?.userId) {
    title = t('User ID: {{id}}', { id: subject.userId })
  } else if (subject?.username) {
    title = t('Username: {{username}}', { username: subject.username })
  }
  if (!targeted && !loading && !error && !snapshot?.total) return null
  return (
    <>
      <Card className='shrink-0 gap-0 overflow-hidden py-0'>
        <CardHeader className='flex flex-row items-center justify-between gap-2 border-b px-3 py-2'>
          <CardTitle className='flex min-w-0 items-center gap-2 text-sm'>
            <Activity className='size-4 shrink-0' />
            <span className='truncate' title={title}>
              {title}
            </span>
            <span className='bg-muted rounded px-1.5 py-0.5 text-xs tabular-nums'>
              {snapshot ? snapshot.summary.total : '—'}
            </span>
          </CardTitle>
          <Button
            variant='ghost'
            size='icon-sm'
            aria-label={t(
              collapsed ? 'Expand connections' : 'Collapse connections'
            )}
            aria-expanded={!collapsed}
            onClick={() => setCollapsed(!collapsed)}
          >
            <ChevronDown
              className={cn('transition-transform', !collapsed && 'rotate-180')}
            />
          </Button>
        </CardHeader>
        {!collapsed && (
          <CardContent className='space-y-3 p-3'>
            <ConnectionNotice
              error={!!error}
              stale={!!snapshot}
              retry={() => void refresh()}
              scope={snapshot?.scope}
            />
            {loading && (
              <p role='status' className='text-muted-foreground text-xs'>
                {t('Loading connection status…')}
              </p>
            )}
            {snapshot && (
              <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-3'>
                {snapshot.items.slice(0, 3).map((request) => (
                  <ConnectionRow key={request.id} request={request} now={now} />
                ))}
              </div>
            )}
            {!loading && !error && targeted && snapshot?.total === 0 && (
              <p className='text-muted-foreground text-xs'>
                {t('This user has no current connections.')}
              </p>
            )}
            <div className='text-muted-foreground flex flex-wrap items-center justify-between gap-2 text-xs'>
              <div>
                {(snapshot?.total || 0) > 3 && (
                  <span className='mr-3'>
                    {t('{{count}} more connections', {
                      count: (snapshot?.total || 0) - 3,
                    })}
                  </span>
                )}
                {t('Ended requests remain visible for 5 seconds.')}
              </div>
              <Button variant='ghost' size='sm' onClick={() => setOpen(true)}>
                {t('View all connections')}
                <ChevronRight />
              </Button>
            </div>
          </CardContent>
        )}
      </Card>
      {open && (
        <ConnectionsDrawer
          subject={subject}
          open={open}
          onOpenChange={setOpen}
          title={title}
        />
      )}
    </>
  )
}

export function ActiveConnectionsPanel({
  subject,
}: {
  subject?: ConnectionSubject
}) {
  // 切换已提交的查询用户时清空抽屉、分页和旧数据，草稿输入不会进入这里。
  return (
    <ConnectionsForSubject
      key={`${subject?.userId || ''}:${subject?.username || ''}`}
      subject={subject}
    />
  )
}
