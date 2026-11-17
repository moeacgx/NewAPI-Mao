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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'

import { getApiKeys, searchApiKeys } from '../api'
import type { SearchApiKeysParams } from '../types'
import type { TokenActivity } from './use-token-activity'

export function useApiKeysQuery(props: {
  userId: number | undefined
  sessionId: string
  params: SearchApiKeysParams
  refreshTrigger: number
  activity: TokenActivity
}) {
  const client = useQueryClient()
  const { userId, sessionId, params, refreshTrigger, activity } = props
  const filters = {
    keyword: params.keyword || '',
    token: params.token || '',
    status: params.status || 0,
    active_only: Boolean(params.active_only),
  }
  const filterKey = JSON.stringify(filters)
  const shouldSearch = Boolean(
    filters.keyword || filters.token || filters.status || filters.active_only
  )
  const query = useQuery({
    queryKey: [
      'keys',
      userId,
      sessionId,
      filterKey,
      params.p,
      params.size,
      refreshTrigger,
    ],
    queryFn: async ({ signal }) => {
      const result = shouldSearch
        ? await searchApiKeys(
            { ...filters, p: params.p, size: params.size },
            signal
          )
        : await getApiKeys({ p: params.p, size: params.size }, signal)
      if (!result.success || !result.data) {
        throw new Error(result.message || 'Failed to load API keys')
      }
      return result.data
    },
    enabled: Boolean(userId),
    retry: false,
    gcTime: 0,
    // 只有同一身份与相同筛选的翻页可保留旧行，不能跨用户或筛选显示过期结果。
    placeholderData: (previous, previousQuery) => {
      const key = previousQuery?.queryKey
      return key?.[1] === userId &&
        key?.[2] === sessionId &&
        key?.[3] === filterKey
        ? previous
        : undefined
    },
  })
  const activeIds = activity.snapshot
    ? Object.entries(activity.snapshot.summary.by_token)
        .filter(([, count]) => count > 0)
        .map(([id]) => id)
        .sort()
        .join(',')
    : undefined
  const previous = useRef<
    | { scope: string; ids?: string; failed: boolean; pending: boolean }
    | undefined
  >(undefined)
  const scope = JSON.stringify([
    userId,
    sessionId,
    filterKey,
    params.p,
    params.size,
    refreshTrigger,
  ])
  const failed = Boolean(activity.error)
  useEffect(() => {
    const old = previous.current
    const pending =
      old?.scope === scope &&
      (old.pending || old.ids !== activeIds || old.failed)
    previous.current = { scope, ids: activeIds, failed, pending }
    if (!filters.active_only || failed || activeIds === undefined) return
    // 首次无缓存请求在途时 invalidate 会复用旧 promise，完成后失效标记被清除。
    // 保留待刷新需求直到请求结束；期间多次活动变化合并为一次，切换查询则丢弃。
    if (pending && !query.isFetching) {
      previous.current.pending = false
      void client.invalidateQueries({
        queryKey: ['keys', userId, sessionId, filterKey],
      })
    }
  }, [
    activeIds,
    failed,
    scope,
    query.isFetching,
    filters.active_only,
    client,
    userId,
    sessionId,
    filterKey,
  ])

  return query
}
