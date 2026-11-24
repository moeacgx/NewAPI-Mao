/*
Copyright (C) 2025 QuantumNous

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
import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { Card, Tabs, TabPane, Spin } from '@douyinfe/semi-ui';
import { TrendingUp, DollarSign, Ticket } from 'lucide-react';
import { VChart } from '@visactor/react-vchart';
import { useTranslation } from 'react-i18next';
import { API, isAdmin, showError } from '../../helpers';
import { renderQuota, renderQuotaWithAmount } from '../../helpers/render';

const TIME_RANGE_OPTIONS = [
  { label: '1天', days: 1, granularity: 'hour' },
  { label: '7天', days: 7, granularity: 'day' },
  { label: '14天', days: 14, granularity: 'day' },
  { label: '29天', days: 29, granularity: 'day' },
];

function formatTimeLabel(timestamp, granularity) {
  const date = new Date(timestamp * 1000);
  if (granularity === 'hour') {
    return `${String(date.getHours()).padStart(2, '0')}:00`;
  }
  return `${date.getMonth() + 1}/${date.getDate()}`;
}

function quotaToLocalCurrency(quota) {
  let quotaPerUnit = parseFloat(localStorage.getItem('quota_per_unit')) || 500000;
  const quotaDisplayType = localStorage.getItem('quota_display_type') || 'USD';
  const usdRate = parseFloat(localStorage.getItem('usd_exchange_rate')) || 1;
  const amountUSD = quota / quotaPerUnit;
  if (quotaDisplayType === 'CNY') {
    return amountUSD * usdRate;
  }
  return amountUSD;
}

// PLACEHOLDER_REST

const StatMiniCard = ({ icon: Icon, label, value, sub, loading }) => (
  <div className='flex items-center gap-3 px-4 py-3'>
    <span className='flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800'>
      <Icon size={14} />
    </span>
    <div className='min-w-0'>
      <div className='text-xs text-gray-500'>{label}</div>
      {loading ? (
        <div className='mt-1 h-5 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700' />
      ) : (
        <>
          <div className='truncate text-base font-semibold tabular-nums'>
            {value}
          </div>
          {sub && (
            <div className='truncate text-xs text-gray-500'>{sub}</div>
          )}
        </>
      )}
    </div>
  </div>
);

const RevenuePanel = ({ CARD_PROPS, CHART_CONFIG }) => {
  const { t } = useTranslation();
  const [selectedDays, setSelectedDays] = useState(7);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState(null);

  const granularity = selectedDays <= 1 ? 'hour' : 'day';

  const loadRevenueData = useCallback(async () => {
    setLoading(true);
    try {
      const now = new Date();
      const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      const end = Math.floor(todayStart.getTime() / 1000) + 86400 - 1;
      const start = Math.floor(todayStart.getTime() / 1000) - (selectedDays - 1) * 86400;
      const tzOffset = now.getTimezoneOffset() * -60;
      const res = await API.get(
        `/api/data/revenue?start_timestamp=${start}&end_timestamp=${end}&granularity=${granularity}&timezone_offset=${tzOffset}`
      );
      const { success, message, data } = res.data;
      if (success) {
        setStats(data);
      } else {
        showError(message);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [selectedDays, granularity]);

  useEffect(() => {
    if (isAdmin()) {
      loadRevenueData();
    }
  }, [loadRevenueData]);

  const summary = stats?.summary;
  const dataPoints = stats?.data_points || [];

  const cashTotals = summary?.cash || [];
  const totalRedemptionQuota = summary?.total_redemption_quota ?? 0;
  const unresolvedCashCount = summary?.unresolved_cash_count ?? 0;

  // Sum all cash amounts (do not convert, just display by currency)
  const totalCashByCurrency = cashTotals.reduce((acc, c) => {
    acc[c.currency] = (acc[c.currency] || 0) + c.amount;
    return acc;
  }, {});
  const totalCashCount = cashTotals.reduce((sum, c) => sum + c.count, 0);

  if (!isAdmin()) return null;

  return (
    <Card
      {...CARD_PROPS}
      className='!rounded-2xl'
      title={
        <div className='flex flex-col sm:flex-row sm:items-center sm:justify-between w-full gap-3'>
          <div className='flex items-center gap-2'>
            <TrendingUp size={16} />
            {t('收入统计')}
          </div>
          <div className='flex items-center gap-1'>
            {TIME_RANGE_OPTIONS.map((opt) => (
              <button
                key={opt.days}
                onClick={() => setSelectedDays(opt.days)}
                className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                  selectedDays === opt.days
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'
                }`}
              >
                {t(opt.label)}
              </button>
            ))}
          </div>
        </div>
      }
      bodyStyle={{ padding: 0 }}
    >
      {/* Stats row */}
      <div className='grid grid-cols-1 sm:grid-cols-2 border-b'>
        {cashTotals.map((cash) => {
          const symbol = cash.currency === 'CNY' ? '¥' : cash.currency === 'USD' ? '$' : '';
          const displayValue = `${symbol}${cash.amount.toFixed(2)}`;
          return (
            <StatMiniCard
              key={cash.currency}
              icon={DollarSign}
              label={`${t('现金')} (${cash.currency})`}
              value={displayValue}
              sub={`${cash.count} ${t('笔')}`}
              loading={loading}
            />
          );
        })}
        <StatMiniCard
          icon={Ticket}
          label={t('兑换码')}
          value={renderQuota(totalRedemptionQuota)}
          sub={`${summary?.total_redemption_count ?? 0} ${t('次')}`}
          loading={loading}
        />
        {unresolvedCashCount > 0 && (
          <div className='col-span-full px-4 py-2 text-xs text-yellow-600 dark:text-yellow-400'>
            {t('有 {{count}} 笔订单币种待核验', { count: unresolvedCashCount })}
          </div>
        )}
      </div>

      {/* Chart */}
      <div className='h-72 p-2'>
        {loading ? (
          <div className='flex h-full items-center justify-center'>
            <Spin />
          </div>
        ) : dataPoints.length === 0 ? (
          <div className='flex h-full items-center justify-center text-sm text-gray-400'>
            {t('该时间段内无收入数据')}
          </div>
        ) : (
          <RevenueChart
            dataPoints={dataPoints}
            granularity={granularity}
            CHART_CONFIG={CHART_CONFIG}
            t={t}
          />
        )}
      </div>
    </Card>
  );
};

// PLACEHOLDER_CHART_COMPONENT

const RevenueChart = ({ dataPoints, granularity, CHART_CONFIG, t }) => {
  const spec = useMemo(() => {
    const chartData = [];
    for (const point of dataPoints) {
      const timeLabel = formatTimeLabel(point.timestamp, granularity);

      // Add cash by currency (do not stack with redemption)
      for (const cashTotal of point.cash) {
        chartData.push({
          time: timeLabel,
          value: cashTotal.amount,
          type: `${t('现金')} (${cashTotal.currency})`,
        });
      }

      // Add redemption as quota display (not converted)
      chartData.push({
        time: timeLabel,
        value: quotaToLocalCurrency(point.redemption_quota),
        type: t('兑换码（额度）'),
      });
    }

    return {
      type: 'line',
      data: [{ id: 'revenue', values: chartData }],
      xField: 'time',
      yField: 'value',
      seriesField: 'type',
      stack: false,
      line: { style: { lineWidth: 2, curveType: 'monotone' } },
      point: { visible: true, style: { size: 3 } },
      legends: {
        visible: true,
        orient: 'bottom',
        position: 'middle',
      },
      tooltip: {
        mark: {
          content: [
            {
              key: (d) => d.type,
              value: (d) => {
                const isCash = d.type.startsWith(t('现金'));
                if (isCash) {
                  return d.value.toFixed(2);
                }
                return renderQuota(d.value / quotaToLocalCurrency(1));
              },
            },
          ],
        },
      },
      axes: [
        {
          orient: 'bottom',
          label: { style: { fontSize: 10 } },
          tick: { visible: false },
        },
        {
          orient: 'left',
          label: {
            formatMethod: (val) => renderQuotaWithAmount(Number(val)),
            style: { fontSize: 10 },
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3] },
          },
        },
      ],
      color: ['#3b82f6', '#f59e0b'],
    };
  }, [dataPoints, granularity, t]);

  return <VChart spec={spec} option={CHART_CONFIG} />;
};

export default RevenuePanel;
