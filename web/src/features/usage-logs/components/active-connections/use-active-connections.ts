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

export interface ConnectionSubject {
  userId?: string
  username?: string
}

export interface ActiveConnection {
  id: string
  request_id?: string
  token_name: string
  model_name: string
  request_type: string
  is_stream: boolean
  started_at: number
  completed_at?: number
  first_response_ms: number | null
  status:
    | 'waiting_response'
    | 'receiving'
    | 'processing'
    | 'retrying'
    | 'completed'
  retry_count: number
}

export interface ConnectionSnapshot {
  items: ActiveConnection[]
  total: number
  page: number
  page_size: number
  observed_at: number
  scope: 'cluster' | 'instance'
  summary: {
    total: number
    streaming: number
    non_streaming: number
    waiting: number
    active_tokens: number
  }
  subject?: { id: number; username: string }
  receivedAt: number
}

export function useActiveConnections({
  subject,
  page = 1,
  pageSize = 3,
  search = '',
  status = '',
  enabled = true,
}: {
  subject?: ConnectionSubject
  page?: number
  pageSize?: number
  search?: string
  status?: string
  enabled?: boolean
}) {
  const queryClient = useQueryClient()
  const [visible, setVisible] = useState(
    () => document.visibilityState !== 'hidden'
  )
  const userId = subject?.userId?.trim() || ''
  const username = userId ? '' : subject?.username?.trim() || ''
  const queryKey = useMemo(
    () => [
      'active-connections',
      userId,
      username,
      page,
      pageSize,
      search,
      status,
    ],
    [userId, username, page, pageSize, search, status]
  )

  useEffect(() => {
    const onVisibility = () => {
      const nextVisible = document.visibilityState !== 'hidden'
      setVisible(nextVisible)
      if (!nextVisible) {
        void queryClient.cancelQueries({ queryKey, exact: true })
      }
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => document.removeEventListener('visibilitychange', onVisibility)
  }, [queryClient, queryKey])

  const query = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      try {
        const identity: Record<string, string> = {}
        if (userId) {
          identity.user_id = userId
        } else if (username) {
          identity.username = username
        }
        const response = await api.get(
          userId || username ? '/api/log/active' : '/api/log/self/active',
          {
            params: {
              page,
              page_size: pageSize,
              ...identity,
              ...(search ? { search } : {}),
              ...(status ? { status } : {}),
            },
            signal,
            timeout: 10000,
            skipErrorHandler: true,
            skipBusinessError: true,
            disableDuplicate: true,
          }
        )
        if (!response.data.success || !response.data.data) {
          throw new Error('Unavailable')
        }
        return {
          ...response.data.data,
          receivedAt: Date.now(),
        } as ConnectionSnapshot
      } catch {
        // 连接观察失败只在模块内显示，避免全局 HTTP 错误处理跳离日志页。
        throw new Error('Connection status is unavailable.')
      }
    },
    enabled: enabled && visible,
    retry: false,
    gcTime: 0,
    staleTime: 0,
    refetchInterval: enabled && visible ? 2000 : false,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  })
  const [clock, setClock] = useState(Date.now)
  useEffect(() => {
    if (!enabled || !visible || query.error) return
    const timer = window.setInterval(() => setClock(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [enabled, visible, query.error])
  const snapshot = query.data
  const stale = !!snapshot && clock - snapshot.receivedAt > 5000
  const now = snapshot
    ? snapshot.observed_at +
      Math.max(
        0,
        (query.error
          ? query.errorUpdatedAt
          : Math.max(clock, snapshot.receivedAt)) - snapshot.receivedAt
      )
    : clock
  return {
    snapshot,
    now,
    loading: query.isPending,
    refreshing: query.isFetching,
    stale,
    error: query.error,
    refresh: query.refetch,
  }
}
