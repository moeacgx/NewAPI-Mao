import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { PricingModel } from '../../types'
import { ModelDetailsContent } from '../model-details'

vi.mock('@lobehub/icons', () => ({}))
vi.mock('../model-details-charts', () => ({
  LatencyTrendChart: () => null,
  UptimeTrendChart: () => null,
}))

const groups = ['vip_code', 'free_code', 'std_code', 'unset_code']

const groupNames = {
  vip_code: 'VIP 显示名',
  free_code: '免费显示名',
  std_code: '标准显示名',
  unset_code: '未配置显示名',
}

// unset_code 没有配置倍率，应按实际定价回退为 1。
const groupRatio = { vip_code: 2.4, free_code: 0, std_code: 1 }

const usableGroup = Object.fromEntries(
  groups.map((group) => [group, { desc: group, ratio: 1 }])
)

const tokenModel: PricingModel = {
  id: 1,
  model_name: 'ratio-token-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: groups,
}

const fixedModel: PricingModel = {
  id: 2,
  model_name: 'ratio-fixed-model',
  quota_type: 1,
  model_ratio: 0,
  completion_ratio: 0,
  model_price: 0.1,
  enable_groups: groups,
}

const dynamicModel: PricingModel = {
  id: 3,
  model_name: 'ratio-dynamic-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 1,
  billing_mode: 'tiered_expr',
  billing_expr:
    'len <= 272000 ? tier("standard", p * 0.5 + c * 3) : tier("long_context", p * 1 + c * 6)',
  enable_groups: groups,
}

const invalidDynamicModel: PricingModel = {
  ...dynamicModel,
  id: 4,
  model_name: 'ratio-invalid-dynamic-model',
  billing_expr: 'p * max(1, 2)',
}

const expectedRatios: Array<[string, string]> = [
  ['VIP 显示名', '2.4x'],
  ['免费显示名', '0x'],
  ['标准显示名', '1x'],
  ['未配置显示名', '1x'],
]

function renderDetails(model: PricingModel, priceRate = 1) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { groups: [] } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ModelDetailsContent
        model={model}
        groupRatio={groupRatio}
        usableGroup={usableGroup}
        endpointMap={{}}
        groupNames={groupNames}
        autoGroups={[]}
        priceRate={priceRate}
        usdExchangeRate={1}
        tokenUnit='M'
      />
    </QueryClientProvider>
  )
}

function getPricingTableRow(name: string) {
  const rows = screen
    .getAllByText(name)
    .map((node) => node.closest('tr'))
    .filter((row): row is HTMLTableRowElement => row !== null)
  expect(rows).toHaveLength(1)
  return rows[0]
}

function getCellTexts(row: Element | null | undefined, selector: string) {
  return [...(row?.querySelectorAll(selector) ?? [])].map(
    (cell) => cell.textContent
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('模型详情按分组定价倍率', () => {
  it.each([
    ['按 Token', tokenModel],
    ['按次', fixedModel],
  ])('%s 定价在分组后单列显示配置倍率，0 和 1 也显示', (_, model) => {
    renderDetails(model)

    const headerRow = screen.getByText('Group').closest('tr')
    expect(getCellTexts(headerRow, 'th')[1]).toBe('Multiplier')

    for (const [name, ratio] of expectedRatios) {
      const cells = getCellTexts(getPricingTableRow(name), 'td')
      expect(cells[1]).toBe(ratio)
    }
    expect(screen.queryByText('vip_code')).not.toBeInTheDocument()
  })

  it('充值折扣独立显示，不混入倍率列，价格保持不变', () => {
    renderDetails(tokenModel, 0.5)

    const standardCells = getCellTexts(getPricingTableRow('标准显示名'), 'td')
    expect(standardCells[0]).toContain('5 fold')
    expect(standardCells[1]).toBe('1x')
    expect(standardCells[2]).toBe('$2')

    const vipCells = getCellTexts(getPricingTableRow('VIP 显示名'), 'td')
    expect(vipCells[0]).not.toContain('fold')
    expect(vipCells[1]).toBe('2.4x')
    expect(vipCells[2]).toBe('$4.8')
  })

  it('可解析表达式的分组卡片标题右侧显示倍率', () => {
    renderDetails(dynamicModel)

    for (const [name, ratio] of expectedRatios) {
      const ratioNode = screen
        .getAllByText(ratio)
        .find((node) => node.parentElement?.textContent?.includes(name))
      expect(ratioNode, `${name} 卡片缺少 ${ratio}`).toBeDefined()
      expect(ratioNode?.closest('table')).toBeNull()
      expect(ratioNode).toHaveClass('whitespace-nowrap')
    }
    expect(screen.queryByText('vip_code')).not.toBeInTheDocument()
  })

  it('不可解析表达式保留提示，不伪造分组倍率卡片', () => {
    renderDetails(invalidDynamicModel)

    expect(
      screen.getByText(
        'Group prices cannot be expanded because this expression is not a standard tiered pricing expression.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('2.4x')).not.toBeInTheDocument()
    expect(screen.queryByText('0x')).not.toBeInTheDocument()
  })
})
