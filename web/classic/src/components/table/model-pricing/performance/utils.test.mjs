import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildLatencyBarHeights,
  buildPerformanceView,
  buildStatusSegments,
  getSuccessRateHex,
  getSuccessRateLevel,
  getSuccessRateTextClass,
  getSuccessRateTextColor,
  getStatusRateTextClass,
  getStatusSegmentHex,
  getUptimeAxisMin,
  normalizePerformanceSeries,
} from './utils.js';

test('最近 24 小时状态压缩为四段并保留空段', () => {
  const endTs = 24 * 60 * 60;
  const segments = buildStatusSegments(
    [
      { ts: 60 * 60, success_rate: 100, avg_latency_ms: 100 },
      { ts: 7 * 60 * 60, success_rate: 100, avg_latency_ms: 200 },
      { ts: 8 * 60 * 60, success_rate: 98, avg_latency_ms: 400 },
      { ts: 19 * 60 * 60, success_rate: 90, avg_latency_ms: 800 },
      { ts: 20 * 60 * 60, success_rate: 101, avg_latency_ms: 100 },
    ],
    endTs,
  );

  assert.equal(segments.length, 4);
  assert.deepEqual(
    segments.map((segment) => segment.success_rate),
    [100, 99, null, 90],
  );
  assert.equal(segments[1].avg_latency_ms, 300);
  assert.equal(segments[2].sample_count, 0);
});

test('状态柱高度随延迟升高而降低，并忽略无效延迟的缩放影响', () => {
  const heights = buildLatencyBarHeights([
    { avg_latency_ms: 100 },
    { avg_latency_ms: 1000 },
    { avg_latency_ms: 10000 },
    { avg_latency_ms: 0 },
  ]);

  assert.equal(heights[0], 100);
  assert.ok(heights[0] > heights[1]);
  assert.ok(heights[1] > heights[2]);
  assert.equal(heights[2], 50);
  assert.equal(heights[3], 50);
  assert.deepEqual(
    buildLatencyBarHeights([{ avg_latency_ms: 500 }, { avg_latency_ms: 500 }]),
    [100, 100],
  );
});

test('性能序列按时间排序并过滤无效时间桶', () => {
  const series = normalizePerformanceSeries([
    { ts: 200, success_rate: 98 },
    { ts: 0, success_rate: 100 },
    { ts: 100, success_rate: 101 },
  ]);

  assert.deepEqual(
    series.map((point) => [point.ts, point.success_rate]),
    [
      [100, 100],
      [200, 98],
    ],
  );
});

test('详情指标按分组等权聚合并生成趋势', () => {
  const view = buildPerformanceView([
    {
      group: 'group-a',
      avg_ttft_ms: 100,
      avg_latency_ms: 1000,
      success_rate: 100,
      avg_tps: 20,
      series: [
        { ts: 100, avg_ttft_ms: 100, success_rate: 100 },
        { ts: 200, avg_ttft_ms: 200, success_rate: 98 },
      ],
    },
    {
      group: 'group-b',
      avg_ttft_ms: 300,
      avg_latency_ms: 3000,
      success_rate: 98,
      avg_tps: 40,
      series: [
        { ts: 100, avg_ttft_ms: 300, success_rate: 98 },
        { ts: 200, avg_ttft_ms: 0, success_rate: 100 },
      ],
    },
  ]);

  assert.equal(view.avgTps, 30);
  assert.equal(view.avgLatency, 2000);
  assert.equal(view.successRate, 99);
  assert.equal(view.incidentCount, 2);
  assert.deepEqual(view.latencySeries, [
    { ts: 100, avg_ttft_ms: 200 },
    { ts: 200, avg_ttft_ms: 200 },
  ]);
  assert.deepEqual(view.uptimeSeries, [
    { ts: 100, success_rate: 99, incidents: 1 },
    { ts: 200, success_rate: 99, incidents: 1 },
  ]);
});

test('详情成功率按官方阈值统一数字、柱与趋势点的颜色', () => {
  const cases = [
    [100, 'healthy', '#10b981', 'success'],
    [99.99, 'healthy', '#34d399', 'success'],
    [98.6, 'healthy', '#34d399', 'success'],
    [90, 'healthy', '#34d399', 'success'],
    [89.99, 'warning', '#f59e0b', 'warning'],
    [70, 'warning', '#f59e0b', 'warning'],
    [69.99, 'critical', '#ef4444', 'danger'],
    [0, 'critical', '#ef4444', 'danger'],
    [Number.NaN, 'unknown', '#9ca3af', 'text-2'],
    [Number.POSITIVE_INFINITY, 'unknown', '#9ca3af', 'text-2'],
  ];

  for (const [rate, level, hex, semanticColor] of cases) {
    assert.equal(getSuccessRateLevel(rate), level, String(rate));
    assert.equal(getSuccessRateHex(rate), hex, String(rate));
    assert.equal(
      getSuccessRateTextClass(rate),
      `text-semi-color-${semanticColor}`,
      String(rate),
    );
    assert.equal(
      getSuccessRateTextColor(rate),
      `var(--semi-color-${semanticColor})`,
      String(rate),
    );
  }
});

test('可用率趋势轴下限保持稳定', () => {
  assert.equal(getUptimeAxisMin([99.9, 98]), 95);
  assert.equal(getUptimeAxisMin([94.5]), 90);
  assert.equal(getUptimeAxisMin([83]), 70);
});

test('四段式状态条使用统一的成功、提醒和异常阈值', () => {
  assert.equal(getStatusSegmentHex(99.9), '#10b981');
  assert.equal(getStatusSegmentHex(99), '#f59e0b');
  assert.equal(getStatusSegmentHex(98.99), '#f43f5e');
  assert.equal(getStatusRateTextClass(99.9), 'text-semi-color-success');
  assert.equal(getStatusRateTextClass(99), 'text-semi-color-warning');
  assert.equal(getStatusRateTextClass(98.99), 'text-semi-color-danger');
});
