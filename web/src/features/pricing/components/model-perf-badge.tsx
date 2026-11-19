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
import { memo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  formatLatency,
  formatThroughput,
} from '@/features/performance-metrics/lib/format'
import { buildStatusSegments } from '@/features/performance-metrics/lib/status-segments'
import type {
  PerfModelSummary,
  PerformanceSeriesPoint,
} from '@/features/performance-metrics/types'
import { cn } from '@/lib/utils'

import {
  UptimeSparkline,
  type UptimeSparklinePoint,
} from './model-details-uptime-sparkline'

export type ModelPerfBadgeData = Pick<
  PerfModelSummary,
  'avg_latency_ms' | 'success_rate' | 'status_rate' | 'avg_tps' | 'series'
>

export interface ModelPerfBadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  perf: ModelPerfBadgeData | undefined
}

// 卡片与详情页同为最近 24 小时：每小时一根长条，分钟级分桶会压缩到同样的 24 根。
const MODEL_PERF_STATUS_BAR_COUNT = 24
const AGGREGATE_STATUS_SEGMENT_COUNT = 3

function buildAggregateStatusSeries(
  successRate: number
): PerformanceSeriesPoint[] {
  if (!Number.isFinite(successRate)) return []
  const now = Math.trunc(Date.now() / 1000)
  const segmentSeconds = Math.trunc(
    (24 * 60 * 60) / AGGREGATE_STATUS_SEGMENT_COUNT
  )
  const segmentMidpointSeconds = Math.trunc(segmentSeconds / 2)
  return Array.from({ length: AGGREGATE_STATUS_SEGMENT_COUNT }, (_, index) => {
    const offset = AGGREGATE_STATUS_SEGMENT_COUNT - index - 1
    return {
      ts: now - offset * segmentSeconds - segmentMidpointSeconds,
      avg_ttft_ms: 0,
      avg_latency_ms: 0,
      success_rate: successRate,
      avg_tps: 0,
    }
  })
}

export const ModelPerfBadge = memo(function ModelPerfBadge(
  props: ModelPerfBadgeProps
) {
  const { t } = useTranslation()

  if (!props.perf) {
    return null
  }

  const { avg_latency_ms, avg_tps, success_rate } = props.perf
  const statusRate = Number.isFinite(props.perf.status_rate)
    ? Number(props.perf.status_rate)
    : success_rate
  const rawStatusSeries = props.perf.series ?? []
  const statusSeries =
    rawStatusSeries.length > 0
      ? rawStatusSeries.map((point) => ({
          ...point,
          success_rate: Number.isFinite(point.status_rate)
            ? Number(point.status_rate)
            : point.success_rate,
        }))
      : buildAggregateStatusSeries(statusRate)
  const uptimeSeries = buildStatusSegments(
    statusSeries,
    Math.trunc(Date.now() / 1000),
    MODEL_PERF_STATUS_BAR_COUNT
  ).map<UptimeSparklinePoint>((segment) => ({
    date: new Date(segment.startTs * 1000).toISOString(),
    uptime_pct: segment.successRate,
    incidents: segment.successRate != null && segment.successRate < 100 ? 1 : 0,
    outage_minutes: 0,
  }))

  return (
    <div
      className={cn(
        'grid w-full grid-cols-[auto_auto_minmax(0,6rem)] items-end justify-end gap-x-1.5 text-left tabular-nums min-[460px]:grid-cols-[auto_auto_minmax(0,7.5rem)] min-[460px]:gap-x-2',
        props.className
      )}
    >
      <div title={t('Average latency')} className='min-w-0'>
        <div className='text-muted-foreground/55 text-[10px] leading-4'>
          {t('Latency short')}
        </div>
        <div className='text-muted-foreground/80 font-mono text-xs leading-4 whitespace-nowrap'>
          {avg_latency_ms > 0 ? formatLatency(avg_latency_ms) : '—'}
        </div>
      </div>
      <div title={t('Throughput')} className='min-w-0'>
        <div className='text-muted-foreground/55 truncate text-[10px] leading-4'>
          {t('Throughput short')}
        </div>
        <div className='text-muted-foreground/80 font-mono text-xs leading-4 whitespace-nowrap'>
          {formatThroughput(avg_tps).replace(' t/s', 't')}
        </div>
      </div>
      <div title={t('Status')} className='min-w-0'>
        <div className='text-muted-foreground/55 truncate text-[10px] leading-4'>
          {t('Status short')}
        </div>
        <UptimeSparkline
          series={uptimeSeries}
          windowLabel={t('Last 24 hours status')}
          tone='availability'
          size='fill'
          showOverall={false}
          className='w-full'
        />
      </div>
    </div>
  )
})
