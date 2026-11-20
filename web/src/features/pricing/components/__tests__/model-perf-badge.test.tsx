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
  const strip = screen.getByRole('img', { name: /^Status:/ })
  return strip.children
}

function renderBadge(series: ReturnType<typeof buildHourlySeries>, rate = 100) {
  return render(
    <ModelPerfBadge
      perf={{
        avg_latency_ms: 1200,
        success_rate: rate,
        status_rate: rate,
        avg_tps: 42,
        series,
      }}
    />
  )
}

describe('模型广场 Default 卡片状态信号', () => {
  it('24 小时逐桶数据压缩为三格信号，并保留延迟与吞吐量', () => {
    renderBadge(buildHourlySeries(20, 100))

    const bars = getStatusBars()
    expect(bars).toHaveLength(3)
    expect(bars[0]).toHaveClass('h-2', 'rounded-full')
    expect(bars[1]).toHaveClass('h-2.5')
    expect(bars[2]).toHaveClass('h-3')
    expect(screen.getByTitle('Average latency')).toHaveTextContent('1.20s')
    expect(screen.getByTitle('Throughput')).toHaveTextContent('42.0t')
  })

  it('某个 8 小时时段没有请求时该格保留为空位，其余格按样本着色', () => {
    const series = buildHourlySeries(24, 100).filter(
      (_, index) => index < 8 || index >= 16
    )
    renderBadge(series)

    const bars = getStatusBars()
    expect(bars).toHaveLength(3)
    expect(bars[0]).toHaveClass('bg-success')
    expect(bars[1]).toHaveClass('bg-muted-foreground/15')
    expect(bars[2]).toHaveClass('bg-success')
  })

  it('同一格内按样本平均可用率着色', () => {
    const series = buildHourlySeries(24, 100).map((point, index) =>
      index >= 16 && index % 2 === 0 ? { ...point, success_rate: 0 } : point
    )
    renderBadge(series)

    const bars = getStatusBars()
    expect(bars[0]).toHaveClass('bg-success')
    expect(bars[2]).toHaveClass('bg-warning')
  })

  it.each([
    [0, 'bg-destructive'],
    [50, 'bg-warning'],
    [95, 'bg-success'],
  ])(
    '时段可用率 %s%% 沿用原卡片分级，只有 0%% 显示为故障',
    (rate, expected) => {
      renderBadge(buildHourlySeries(24, rate), rate)

      for (const bar of Array.from(getStatusBars())) {
        expect(bar).toHaveClass(expected)
      }
    }
  )

  it('只有聚合状态率时仍显示三格信号而不是空占位', () => {
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

    const bars = getStatusBars()
    expect(bars).toHaveLength(3)
    for (const bar of Array.from(bars)) {
      expect(bar).toHaveClass('bg-success')
    }
    expect(screen.getByTitle('Average latency')).toHaveTextContent('—')
  })

  it('没有性能数据时不渲染任何内容', () => {
    const { container } = render(<ModelPerfBadge perf={undefined} />)
    expect(container).toBeEmptyDOMElement()
  })
})
