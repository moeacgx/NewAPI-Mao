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
import { useEffect, useMemo, useState } from 'react'

import { api } from '@/lib/api'

export interface TokenActivitySnapshot {
  summary: {
    total: number
    active_tokens: number
    by_token: Record<string, number>
  }
  scope: 'instance' | 'cluster'
  observed_at: number
  receivedAt: number
}

export function useTokenActivity(
  userId: number | undefined,
  sessionId: string
) {
  const client = useQueryClient()
  const [visible, setVisible] = useState(
    () => document.visibilityState !== 'hidden'
  )
  const [clock, setClock] = useState(Date.now)
  const queryKey = useMemo(
    () => ['token-activity', userId, sessionId],
    [userId, sessionId]
  )

  useEffect(() => {
    const onVisibility = () => {
      const next = document.visibilityState !== 'hidden'
      setClock(Date.now())
      setVisible(next)
      if (!next) void client.cancelQueries({ queryKey, exact: true })
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      document.removeEventListener('visibilitychange', onVisibility)
      void client.cancelQueries({ queryKey, exact: true })
      client.removeQueries({ queryKey, exact: true })
    }
  }, [client, queryKey])

  useEffect(() => {
    if (!userId || !visible) return
    const timer = window.setInterval(() => setClock(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [userId, visible])

  const query = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      // 令牌页始终看本人；管理员身份也不能切换到其他用户接口。
      const response = await api
        .get('/api/log/self/active', {
          params: { page: 1, page_size: 1 },
          signal,
          timeout: 10000,
          skipErrorHandler: true,
          skipBusinessError: true,
          disableDuplicate: true,
        })
        .catch(() => {
          // 与日志页保持一致：局部观察故障不能触发全局 Axios 500 跳转。
          throw new Error('Connection status is unavailable.')
        })
      if (!response.data.success || !response.data.data?.summary?.by_token) {
        throw new Error('Connection status is unavailable.')
      }
      const data = response.data.data as Omit<
        TokenActivitySnapshot,
        'receivedAt'
      >
      // 只保留服务端的全量摘要，不把分页预览或结束保留记录当作活动集合。
      return {
        summary: data.summary,
        scope: data.scope,
        observed_at: data.observed_at,
        receivedAt: Date.now(),
      }
    },
    enabled: Boolean(userId) && visible,
    retry: false,
    gcTime: 0,
    staleTime: 0,
    refetchInterval: userId && visible ? 2000 : false,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  })

  return {
    snapshot: query.data,
    loading: query.isPending,
    refreshing: query.isFetching,
    error: query.error,
    stale: Boolean(
      query.data &&
      (query.error || !visible || clock - query.data.receivedAt > 5000)
    ),
    refresh: query.refetch,
  }
}

export type TokenActivity = ReturnType<typeof useTokenActivity>
