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
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ModelPerfBadge } from '../model-perf-badge'

const nowSeconds = Math.trunc(Date.now() / 1000)

function buildHourlySeries(count: number, successRate: number) {
  return Array.from({ length: count }, (_, index) => ({
    ts: nowSeconds - (count - index) * 3600 + 60,
    avg_ttft_ms: 300,
    avg_latency_ms: 1200,
    success_rate: successRate,
    avg_tps: 40,
  }))
}

function getStatusBars() {
  const strip = screen.getByRole('img', { name: /24 hours/i })
  return strip.children
}

describe('模型广场 Default 卡片状态条', () => {
  it('24 小时逐桶数据渲染为与详情页一致的多根长条，并保留延迟与吞吐量', () => {
    render(
      <ModelPerfBadge
        perf={{
          avg_latency_ms: 1200,
          success_rate: 100,
          status_rate: 100,
          avg_tps: 42,
          series: buildHourlySeries(20, 100),
        }}
      />
    )

    expect(getStatusBars().length).toBeGreaterThan(3)
    expect(screen.getByTitle('Average latency')).toHaveTextContent('1.20s')
    expect(screen.getByTitle('Throughput')).toHaveTextContent('42.0t')
  })

  it('没有请求的小时保留为空位，状态条固定为 24 根并播报 24 小时窗口', () => {
    const series = buildHourlySeries(24, 100).filter(
      (_, index) => index % 4 !== 1
    )
    render(
      <ModelPerfBadge
        perf={{
          avg_latency_ms: 1200,
          success_rate: 100,
          status_rate: 100,
          avg_tps: 42,
          series,
        }}
      />
    )

    const strip = screen.getByRole('img', { name: /24 hours/i })
    expect(strip.children).toHaveLength(24)
    expect(
      strip.querySelectorAll('.bg-muted-foreground\\/15').length
    ).toBeGreaterThan(0)
    expect(
      screen.queryByRole('img', { name: /30 day/i })
    ).not.toBeInTheDocument()
  })

  it.each([
    [0, 'bg-destructive'],
    [50, 'bg-warning'],
    [95, 'bg-success'],
  ])('桶可用率 %s%% 沿用原卡片分级，只有 0%% 显示为故障', (rate, expected) => {
    render(
      <ModelPerfBadge
        perf={{
          avg_latency_ms: 1200,
          success_rate: rate,
          status_rate: rate,
          avg_tps: 42,
          series: buildHourlySeries(24, rate),
        }}
      />
    )

    const strip = screen.getByRole('img', { name: /24 hours/i })
    const bar = strip.querySelector(`.${expected}`)
    expect(bar).not.toBeNull()
    for (const other of ['bg-destructive', 'bg-warning', 'bg-success']) {
      if (other === expected) continue
      expect(strip.querySelector(`.${other}`)).toBeNull()
    }
  })

  it('只有聚合状态率时仍显示状态条而不是空占位', () => {
    render(
      <ModelPerfBadge
        perf={{
          avg_latency_ms: 0,
          success_rate: 99,
          status_rate: 99,
          avg_tps: 0,
          series: [],
        }}
      />
    )

    expect(getStatusBars().length).toBeGreaterThan(0)
    expect(screen.getByTitle('Average latency')).toHaveTextContent('—')
  })

  it('没有性能数据时不渲染任何内容', () => {
    const { container } = render(<ModelPerfBadge perf={undefined} />)
    expect(container).toBeEmptyDOMElement()
  })
})
