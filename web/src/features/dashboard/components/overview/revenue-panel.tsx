/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { Banknote, Ticket, TrendingUp } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { formatQuotaWithCurrency, formatCurrencyFromUSD } from '@/lib/currency'
import { formatNumber } from '@/lib/format'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import { getRevenueStats, type RevenueDataPoint } from '../../api'
import {
  buildRevenueChartData,
  buildRevenueRequest,
  quotaToLocalCurrency,
  type RevenueGranularity,
} from '../../lib/revenue'

const REVENUE_TIME_RANGES = [
  { label: '1 Day', days: 1 },
  { label: '7 Days', days: 7 },
  { label: '30 Days', days: 30 },
] as const

function getChartThemeTokens(resolvedTheme: string) {
  return {
    textColor:
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.68)'
        : 'rgba(15, 23, 42, 0.58)',
    gridColor:
      resolvedTheme === 'dark'
        ? 'rgba(255, 255, 255, 0.12)'
        : 'rgba(15, 23, 42, 0.12)',
  }
}

export function RevenuePanel() {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const [selectedDays, setSelectedDays] = useState(7)
  const request = useMemo(
    () => buildRevenueRequest(selectedDays, new Date().getTimezoneOffset()),
    [selectedDays]
  )
  const query = useQuery({
    queryKey: ['revenue-stats', request],
    queryFn: () => getRevenueStats(request),
    staleTime: 60_000,
  })
  const stats = query.data?.data
  const summary = stats?.summary
  const cash = summary?.cash ?? []
  const redemptionQuota = summary?.total_redemption_quota ?? 0
  const unresolvedCashCount = summary?.unresolved_cash_count ?? 0

  return (
    <div className='bg-card overflow-hidden rounded-2xl border shadow-xs'>
      <div className='flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3 sm:px-5'>
        <div className='flex items-center gap-2'>
          <TrendingUp className='text-muted-foreground size-4' />
          <h3 className='text-sm font-semibold'>{t('Revenue Statistics')}</h3>
        </div>
        <div className='flex items-center gap-1'>
          {REVENUE_TIME_RANGES.map((range) => (
            <Button
              key={range.days}
              variant={selectedDays === range.days ? 'default' : 'ghost'}
              size='sm'
              className='h-7 px-2.5 text-xs'
              onClick={() => setSelectedDays(range.days)}
            >
              {t(range.label)}
            </Button>
          ))}
        </div>
      </div>

      <div className='grid auto-cols-fr grid-flow-col gap-px border-b'>
        {cash.map((c) => (
          <RevenueStat
            key={c.currency}
            icon={Banknote}
            label={`${t('Cash')} (${c.currency})`}
            value={`${c.currency === 'CNY' ? '¥' : c.currency === 'USD' ? '$' : ''}${formatNumber(c.amount, 2)}`}
            sub={t('{{count}} payments', {
              count: formatNumber(c.count, 0),
            })}
            loading={query.isLoading}
          />
        ))}
        <RevenueStat
          icon={Ticket}
          label={t('Redemption Code')}
          value={formatQuotaWithCurrency(redemptionQuota)}
          sub={t('{{count}} redemptions', {
            count: formatNumber(summary?.total_redemption_count ?? 0, 0),
          })}
          loading={query.isLoading}
        />
      </div>

      {unresolvedCashCount > 0 && (
        <div className='border-b px-4 py-2 text-xs text-yellow-600 dark:text-yellow-500'>
          {t('{{count}} orders with unverifiable currency', { count: unresolvedCashCount })}
        </div>
      )}

      <div className='h-56 px-3 py-2 sm:h-64 sm:px-4'>
        <RevenueChart
          points={stats?.data_points ?? []}
          granularity={request.granularity}
          loading={query.isLoading}
          themeReady={themeReady}
          resolvedTheme={resolvedTheme}
        />
      </div>
    </div>
  )
}

function RevenueStat(props: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  sub?: string
  loading: boolean
}) {
  const Icon = props.icon
  return (
    <div className='flex items-center gap-3 px-4 py-3 sm:px-5'>
      <span className='bg-muted flex size-8 shrink-0 items-center justify-center rounded-lg'>
        <Icon className='size-3.5' aria-hidden='true' />
      </span>
      <div className='min-w-0'>
        <div className='text-muted-foreground text-xs'>{props.label}</div>
        {props.loading ? (
          <Skeleton className='mt-1 h-5 w-20' />
        ) : (
          <>
            <div className='truncate text-base font-semibold tabular-nums'>
              {props.value}
            </div>
            {props.sub ? (
              <div className='text-muted-foreground truncate text-xs'>
                {props.sub}
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  )
}

function RevenueChart(props: {
  points: RevenueDataPoint[]
  granularity: RevenueGranularity
  loading: boolean
  themeReady: boolean
  resolvedTheme: string
}) {
  const { t } = useTranslation()
  const { textColor, gridColor } = getChartThemeTokens(props.resolvedTheme)
  const chartData = useMemo(
    () =>
      buildRevenueChartData(props.points, props.granularity, {
        cash: (currency: string) => `${t('Cash')} (${currency})`,
        redemption: t('Redemption Code'),
      }),
    [props.points, props.granularity, t]
  )

  const spec = useMemo(
    () => ({
      type: 'line' as const,
      data: [{ id: 'revenue', values: chartData }],
      xField: 'time',
      yField: 'value',
      seriesField: 'type',
      stack: false,
      line: { style: { lineWidth: 2, curveType: 'monotone' as const } },
      point: { visible: true, style: { size: 3 } },
      legends: {
        visible: true,
        orient: 'bottom' as const,
        position: 'middle' as const,
        item: { label: { style: { fill: textColor, fontSize: 11 } } },
      },
      tooltip: {
        mark: {
          content: [
            {
              key: (datum: { type: string }) => datum.type,
              value: (datum: { value: number; category: 'cash' | 'quota' }) => {
                if (datum.category === 'cash') {
                  return formatNumber(datum.value, 2)
                }
                // datum.value is USD amount from quotaToLocalCurrency
                return formatCurrencyFromUSD(datum.value)
              },
            },
          ],
        },
      },
      axes: [
        {
          orient: 'bottom' as const,
          label: { style: { fill: textColor, fontSize: 10 } },
          tick: { visible: false },
        },
        {
          orient: 'left' as const,
          label: {
            formatMethod: (value: number | string) => formatNumber(Number(value), 2),
            style: { fill: textColor, fontSize: 10 },
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: gridColor },
          },
        },
        {
          orient: 'right' as const,
          label: {
            formatMethod: (value: number | string) => {
              const usdAmount = Number(value)
              if (usdAmount === 0) return '0'
              return formatCurrencyFromUSD(usdAmount, { showSymbol: false })
            },
            style: { fill: textColor, fontSize: 10 },
          },
          grid: { visible: false },
        },
      ],
      color: ['hsl(var(--chart-1))', 'hsl(var(--chart-2))', 'hsl(var(--chart-3))'],
      theme: props.resolvedTheme,
      background: 'transparent',
    }),
    [chartData, gridColor, textColor, t, props.resolvedTheme]
  )

  if (props.loading || !props.themeReady) {
    return <Skeleton className='h-full w-full rounded-lg' />
  }
  if (chartData.length === 0) {
    return (
      <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
        {t('No revenue data in this period')}
      </div>
    )
  }
  return (
    <VChart
      key={`revenue-${props.resolvedTheme}`}
      spec={{
        ...spec,
        theme: props.resolvedTheme === 'dark' ? 'dark' : 'light',
        background: 'transparent',
      }}
      option={VCHART_OPTION}
    />
  )
}
