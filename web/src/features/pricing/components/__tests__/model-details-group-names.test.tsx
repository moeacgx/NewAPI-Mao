import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { PricingModel } from '../../types'
import { ModelDetailsContent } from '../model-details'

vi.mock('@lobehub/icons', () => ({}))
vi.mock('../model-details-charts', () => ({
  LatencyTrendChart: () => null,
  UptimeTrendChart: () => null,
}))

const model: PricingModel = {
  id: 1,
  model_name: 'fixture-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['group_4', 'CC-KIRO', 'shared-a', 'shared-b', 'deleted'],
}

const groupNames = {
  group_4: '当前名称',
  'CC-KIRO': '历史别名对应名称',
  'shared-a': '同名展示',
  'shared-b': '同名展示',
}

function renderDetails(names: Record<string, string>) {
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
        groupRatio={{}}
        usableGroup={{}}
        endpointMap={{}}
        groupNames={names}
        autoGroups={[]}
        priceRate={1}
        usdExchangeRate={1}
        tokenUnit='M'
      />
    </QueryClientProvider>
  )
}

function getGroupsCell() {
  const label = screen.getByText('Groups')
  return label.parentElement as HTMLElement
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('模型详情 Groups 标签分组名称', () => {
  it('名称与编码不同时显示当前名称与历史别名名称，保持顺序，缺失映射回退编码', () => {
    renderDetails(groupNames)
    const cell = getGroupsCell()
    const pills = [...cell.querySelectorAll('span.rounded-md')].map(
      (node) => node.textContent
    )

    expect(pills).toEqual([
      '当前名称',
      '历史别名对应名称',
      '同名展示',
      '同名展示',
      'deleted',
    ])
    expect(within(cell).queryByText('group_4')).not.toBeInTheDocument()
    expect(within(cell).queryByText('CC-KIRO')).not.toBeInTheDocument()
  })

  it('改名后的映射立即生效，旧名称不残留', () => {
    const view = renderDetails(groupNames)
    expect(within(getGroupsCell()).getByText('当前名称')).toBeInTheDocument()
    view.unmount()

    renderDetails({ ...groupNames, group_4: '改名后' })
    expect(within(getGroupsCell()).getByText('改名后')).toBeInTheDocument()
    expect(
      within(getGroupsCell()).queryByText('当前名称')
    ).not.toBeInTheDocument()
  })
})
