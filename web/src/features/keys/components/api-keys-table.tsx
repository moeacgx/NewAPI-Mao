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
import { getRouteApi, useRouter } from '@tanstack/react-router'
import type { Table as TanstackTable } from '@tanstack/react-table'
import { Database } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DISABLED_ROW_DESKTOP,
  DISABLED_ROW_MOBILE,
  DataTablePage,
  useDataTable,
} from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { TokenPromptBadge } from '@/features/prompt-center/components/token-prompt-badge'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import {
  API_KEY_STATUS,
  API_KEY_STATUS_OPTIONS,
  API_KEY_STATUSES,
  ERROR_MESSAGES,
} from '../constants'
import { useApiKeysQuery } from '../hooks/use-api-keys-query'
import { useTokenActivity } from '../hooks/use-token-activity'
import type { ApiKey } from '../types'
import { ApiKeyActivityContext } from './api-key-activity-context'
import { ApiKeySearchInput } from './api-key-search-input'
import {
  ApiKeyCell,
  PeriodQuotaSummary,
  UnlimitedQuotaBadge,
} from './api-keys-cells'
import { useApiKeysColumns } from './api-keys-columns'
import { useApiKeys } from './api-keys-provider'
import { DataTableBulkActions } from './data-table-bulk-actions'
import { DataTableRowActions } from './data-table-row-actions'
import { TokenActivityBar, TokenActivityBadge } from './token-activity'

const route = getRouteApi('/_authenticated/keys/')
const API_KEYS_COLUMN_VISIBILITY_STORAGE_KEY = 'api-keys:column-visibility'
const API_KEYS_MOBILE_SKELETON_IDS = Array.from(
  { length: 5 },
  (_, index) => `api-key-mobile-skeleton-${index + 1}`
)

function isDisabledApiKeyRow(apiKey: ApiKey) {
  return apiKey.status !== API_KEY_STATUS.ENABLED
}

function ApiKeysMobileSkeleton() {
  return (
    <div className='divide-border overflow-hidden rounded-lg border'>
      {API_KEYS_MOBILE_SKELETON_IDS.map((id) => (
        <div
          key={id}
          className='space-y-2 border-b px-3 py-2.5 last:border-b-0'
        >
          <div className='flex items-center justify-between'>
            <Skeleton className='h-4 w-32' />
            <Skeleton className='h-5 w-16 rounded-md' />
          </div>
          <div className='flex items-center justify-between gap-3'>
            <Skeleton className='h-7 w-44' />
            <Skeleton className='h-8 w-16' />
          </div>
          <Skeleton className='h-3 w-28' />
        </div>
      ))}
    </div>
  )
}

function ApiKeysMobileList({
  table,
  isLoading,
  byToken,
  activityStale,
  failed,
}: {
  table: TanstackTable<ApiKey>
  isLoading: boolean
  byToken: Record<string, number>
  activityStale: boolean
  failed: boolean
}) {
  const { t } = useTranslation()
  const rows = table.getRowModel().rows

  if (isLoading) return <ApiKeysMobileSkeleton />

  if (!rows.length) {
    return (
      <div className='rounded-lg border p-8'>
        <Empty className='border-none p-0'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <Database className='size-6' />
            </EmptyMedia>
            <EmptyTitle>
              {t(failed ? ERROR_MESSAGES.LOAD_FAILED : 'No API Keys Found')}
            </EmptyTitle>
            <EmptyDescription>
              {t(
                failed
                  ? 'Retry'
                  : 'No API keys available. Create your first API key to get started.'
              )}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div className='divide-border overflow-hidden rounded-lg border'>
      {rows.map((row) => {
        const apiKey = row.original
        const statusConfig = API_KEY_STATUSES[apiKey.status]
        const total = apiKey.used_quota + apiKey.remain_quota

        return (
          <div
            key={row.id}
            data-token-row={apiKey.id}
            className={cn(
              'bg-card space-y-2.5 border-b px-3 py-2.5 last:border-b-0',
              isDisabledApiKeyRow(apiKey) && DISABLED_ROW_MOBILE
            )}
          >
            <div className='flex items-start justify-between gap-3'>
              <div className='min-w-0'>
                <div className='truncate text-sm font-semibold'>
                  {apiKey.name}
                </div>
                <div className='text-muted-foreground text-[11px]'>
                  {t('API Key')}
                </div>
                <TokenActivityBadge
                  tokenId={apiKey.id}
                  count={byToken[String(apiKey.id)] || 0}
                  stale={activityStale}
                />
                <div className='flex flex-wrap items-center gap-1.5 pt-1 empty:hidden'>
                  <TokenPromptBadge token={apiKey} />
                </div>
              </div>
              {statusConfig && (
                <StatusBadge
                  label={t(statusConfig.label)}
                  variant={statusConfig.variant}
                  copyable={false}
                />
              )}
            </div>

            <div className='flex min-w-0 items-center justify-between gap-2'>
              <div className='min-w-0 flex-1 [&_button:first-child]:max-w-full [&_button:first-child]:truncate [&_button:first-child]:px-0'>
                <ApiKeyCell apiKey={apiKey} />
              </div>
              <DataTableRowActions row={row} />
            </div>

            <div className='flex items-center justify-between gap-2 text-xs'>
              <span className='text-muted-foreground'>{t('Quota')}</span>
              {apiKey.unlimited_quota ? (
                <UnlimitedQuotaBadge used={apiKey.used_quota} />
              ) : (
                <span className='font-medium tabular-nums'>
                  {formatQuota(apiKey.remain_quota)}
                  <span className='text-muted-foreground font-normal'>
                    {' / '}
                    {formatQuota(total)}
                  </span>
                </span>
              )}
            </div>
            <PeriodQuotaSummary apiKey={apiKey} />
          </div>
        )
      })}
    </div>
  )
}

export function ApiKeysTable() {
  const userId = useAuthStore((state) => state.auth.user?.id)
  const sessionId = useAuthStore((state) => state.auth.session?.sid || '')
  // 身份切换直接卸载旧表格，清理选择、活动快照与未完成请求。
  return (
    <ApiKeysTableContent
      key={`${userId}:${sessionId}`}
      userId={userId}
      sessionId={sessionId}
    />
  )
}

function ApiKeysTableContent(props: {
  userId: number | undefined
  sessionId: string
}) {
  const { t } = useTranslation()
  const { refreshTrigger } = useApiKeys()
  const [now, setNow] = useState(() => Date.now())
  const activity = useTokenActivity(props.userId, props.sessionId)
  const byToken = activity.snapshot?.summary.by_token || {}
  const columns = useApiKeysColumns(now)
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const { history } = useRouter()
  const [searchResetVersion, setSearchResetVersion] = useState(0)
  const activeOnly = Boolean(search.active_only)
  const globalFilter = search.filter || ''

  useEffect(
    () =>
      history.subscribe(({ action }) => {
        if (
          action.type === 'BACK' ||
          action.type === 'FORWARD' ||
          action.type === 'GO'
        ) {
          // 历史恢复必须取消两个输入的计时器和输入法状态，即使字段值未变化。
          setSearchResetVersion((version) => version + 1)
        }
      }),
    [history]
  )

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNow(Date.now())
    }, 30_000)

    return () => window.clearInterval(intervalId)
  }, [])

  const {
    onGlobalFilterChange,
    columnFilters,
    onColumnFiltersChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search,
    navigate,
    pagination: { defaultPage: 1, defaultPageSize: 20 },
    globalFilter: { enabled: true, key: 'filter' },
    columnFilters: [
      { columnId: 'status', searchKey: 'status', type: 'array' },
      { columnId: '_tokenSearch', searchKey: 'token', type: 'string' },
    ],
  })

  const query = useApiKeysQuery({
    userId: props.userId,
    sessionId: props.sessionId,
    params: {
      keyword: globalFilter.trim(),
      token: search.token?.trim(),
      status: Number(search.status?.[0]) || undefined,
      active_only: activeOnly,
      p: pagination.pageIndex + 1,
      size: pagination.pageSize,
    },
    refreshTrigger,
    activity,
  })
  const { data, isLoading, isFetching } = query

  useEffect(() => {
    // 等到真实响应后再校正页码，结束活动导致 total=0 也必须回到第一页。
    if (data && !query.isPlaceholderData && !isFetching && !query.error) {
      ensurePageInRange(
        Math.max(1, Math.ceil(data.total / pagination.pageSize)),
        { resetTo: 'last' }
      )
    }
  }, [
    data,
    query.isPlaceholderData,
    isFetching,
    query.error,
    pagination.pageSize,
    ensurePageInRange,
  ])

  const apiKeys = data?.items || []

  const { table } = useDataTable({
    data: apiKeys,
    columns,
    enableRowSelection: true,
    columnFilters,
    columnVisibilityStorageKey: API_KEYS_COLUMN_VISIBILITY_STORAGE_KEY,
    globalFilter,
    pagination,
    globalFilterFn: () => true,
    onPaginationChange,
    onGlobalFilterChange,
    onColumnFiltersChange,
    manualPagination: true,
    manualFiltering: true,
    getRowId: (row) => String(row.id),
    totalCount: data?.total || 0,
  })

  return (
    <ApiKeyActivityContext.Provider value={{ byToken, stale: activity.stale }}>
      <div className='flex h-full min-h-0 flex-col gap-3'>
        <TokenActivityBar
          activity={activity}
          activeOnly={activeOnly}
          onActiveOnlyChange={(value) => {
            table.resetRowSelection()
            void navigate({
              search: (prev) => ({
                ...prev,
                active_only: value || undefined,
                page: undefined,
              }),
            })
          }}
        />
        {query.error && (
          <div
            role='alert'
            className='bg-card text-destructive flex shrink-0 flex-wrap items-center justify-between gap-2 rounded-lg border p-3 text-sm'
          >
            <span>
              {t(ERROR_MESSAGES.LOAD_FAILED)}
              {data && ` ${t('Showing the last known state.')}`}
            </span>
            <Button
              variant='outline'
              size='sm'
              disabled={isFetching}
              onClick={() => void query.refetch()}
            >
              {t('Retry')}
            </Button>
          </div>
        )}
        <DataTablePage
          table={table}
          columns={columns}
          isLoading={isLoading}
          isFetching={isFetching}
          emptyTitle={t(
            query.error ? ERROR_MESSAGES.LOAD_FAILED : 'No API Keys Found'
          )}
          emptyDescription={t(
            query.error
              ? 'Retry'
              : 'No API keys available. Create your first API key to get started.'
          )}
          skeletonKeyPrefix='api-keys-skeleton'
          applyHeaderSize
          toolbarProps={{
            hasAdditionalFilters: activeOnly,
            onReset: () => {
              setSearchResetVersion((version) => version + 1)
              table.resetRowSelection()
              void navigate({
                search: (prev) => ({
                  ...prev,
                  page: undefined,
                  filter: undefined,
                  token: undefined,
                  status: undefined,
                  active_only: undefined,
                }),
              })
            },
            customSearch: (
              <ApiKeySearchInput
                key={`name:${searchResetVersion}`}
                field='name'
                value={globalFilter}
                onCommit={(value) =>
                  void navigate({
                    search: (prev) => ({
                      ...prev,
                      filter: value || undefined,
                      page: undefined,
                    }),
                  })
                }
              />
            ),
            additionalSearch: (
              <ApiKeySearchInput
                key={`key:${searchResetVersion}`}
                field='key'
                value={search.token || ''}
                onCommit={(value) =>
                  void navigate({
                    search: (prev) => ({
                      ...prev,
                      token: value || undefined,
                      page: undefined,
                    }),
                  })
                }
              />
            ),
            filters: [
              {
                columnId: 'status',
                title: t('Status'),
                options: API_KEY_STATUS_OPTIONS,
                singleSelect: true,
              },
            ],
          }}
          mobile={
            <ApiKeysMobileList
              table={table}
              isLoading={isLoading}
              byToken={byToken}
              activityStale={activity.stale}
              failed={Boolean(query.error)}
            />
          }
          getRowClassName={(row) =>
            isDisabledApiKeyRow(row.original) ? DISABLED_ROW_DESKTOP : undefined
          }
          bulkActions={<DataTableBulkActions table={table} />}
        />
      </div>
    </ApiKeyActivityContext.Provider>
  )
}
