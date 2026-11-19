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
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { PricingModel } from '../../types'
import { ModelCard } from '../model-card'

const copyToClipboard = vi.hoisted(() => vi.fn())
vi.mock('@/hooks/use-copy-to-clipboard', () => ({
  useCopyToClipboard: () => ({ copyToClipboard }),
}))

vi.mock('@/lib/lobe-icon', () => ({ getLobeIcon: () => null }))

const tokenModel: PricingModel = {
  id: 1,
  model_name: 'card-token-model',
  description: 'Card description',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 2,
  enable_groups: ['code-a', 'code-b', 'code-c'],
  tags: 'reasoning,vision,long-context',
  supported_endpoint_types: ['openai', 'anthropic', 'gemini'],
  group_ratio: { 'code-a': 1, 'code-b': 0.5 },
}

const perf = {
  avg_latency_ms: 1200,
  success_rate: 0.99,
  status_rate: 0.99,
  avg_tps: 42,
  series: [],
}

describe('模型广场 Default 卡片', () => {
  it('不显示分组、端点、标签、+N 与重复单位，保留核心信息与操作', async () => {
    const onClick = vi.fn()
    render(
      <ModelCard
        model={tokenModel}
        onClick={onClick}
        selectedGroup='code-b'
        perf={perf}
      />
    )

    expect(screen.getByText('card-token-model')).toBeInTheDocument()
    expect(screen.getByText('Card description')).toBeInTheDocument()
    expect(screen.getByText('Token-based')).toBeInTheDocument()
    expect(screen.getByTitle('Average latency')).toBeInTheDocument()

    for (const hidden of [
      /code-[abc]/,
      'openai',
      'anthropic',
      'gemini',
      'reasoning',
      'vision',
      'long-context',
      /^\+\d+$/,
      /^1[MK]$/,
    ]) {
      expect(screen.queryByText(hidden)).not.toBeInTheDocument()
    }

    await userEvent.click(screen.getByRole('button', { name: /Details/ }))
    expect(onClick).toHaveBeenCalledTimes(1)
    await userEvent.click(screen.getByTitle('Copy'))
    expect(copyToClipboard).toHaveBeenCalledWith('card-token-model')
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('selectedGroup 继续决定价格与折扣徽标', () => {
    const { rerender } = render(
      <ModelCard model={tokenModel} onClick={vi.fn()} selectedGroup='code-a' />
    )
    const fullPrice =
      screen.getAllByText(/^\$/, { selector: 'span.font-mono' })[0]
        .textContent ?? ''
    expect(screen.queryByText(/fold/)).not.toBeInTheDocument()

    rerender(
      <ModelCard model={tokenModel} onClick={vi.fn()} selectedGroup='code-b' />
    )
    expect(screen.getByText('5 fold')).toBeInTheDocument()
    expect(
      screen.getAllByText(/^\$/, { selector: 'span.font-mono' })[0]
    ).not.toHaveTextContent(fullPrice)
  })

  it('按次计费模型保留价格内单位并隐藏页脚单位', () => {
    render(
      <ModelCard
        model={{
          ...tokenModel,
          quota_type: 1,
          model_price: 0.2,
          model_price_unit: 'request',
        }}
        onClick={vi.fn()}
      />
    )
    expect(screen.getByText('Per Request')).toBeInTheDocument()
    expect(screen.getAllByText(/request/)).toHaveLength(1)
  })
})
