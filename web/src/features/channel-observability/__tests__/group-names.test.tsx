/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { StabilityView } from '../stability-view'
import type { AnalyticsFilters } from '../types'

const filters: AnalyticsFilters = {
  range: '1h',
  customStart: 0,
  customEnd: 0,
  granularity: 'auto',
  channelId: '',
  channelType: '',
  group: '',
  requestedModel: '',
  requestedModelHash: '',
  upstreamModel: '',
  upstreamModelHash: '',
  outcome: '',
  statusCode: '',
  stream: '',
  trafficSource: 'relay',
  dataOrigin: 'live,legacy',
}

afterEach(() => vi.restoreAllMocks())

describe('稳定性矩阵分组名称', () => {
  it('历史标识下刷新当前长名称且下钻仍提交原标识', async () => {
    let name = '当前企业分组'
    const requests: URLSearchParams[] = []
    vi.spyOn(api, 'get').mockImplementation(async (_url, config) => {
      const params = new URLSearchParams(config?.params)
      requests.push(params)
      return {
        data: {
          success: true,
          data: {
            dimension: params.get('dimension'),
            total: 1,
            page: 1,
            page_size: 30,
            items:
              params.get('dimension') === 'group'
                ? [
                    {
                      key: 'legacy-enterprise',
                      group: 'legacy-enterprise',
                      group_name: name,
                      windows: [],
                    },
                  ]
                : [
                    {
                      key: 'channel-9',
                      group: 'legacy-enterprise',
                      group_name: name,
                      channel_id: 9,
                      channel_name: '验证渠道',
                      windows: [],
                    },
                  ],
          },
        },
      }
    })
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const view = (refreshKey: number) => (
      <QueryClientProvider client={client}>
        <StabilityView
          filters={filters}
          refreshKey={refreshKey}
          onReset={() => {}}
        />
      </QueryClientProvider>
    )
    const { rerender } = render(view(0))
    expect(await screen.findByText('当前企业分组')).toBeInTheDocument()
    expect(screen.queryByText('legacy-enterprise')).not.toBeInTheDocument()
    const row = screen.getByText('当前企业分组').closest('tr')
    if (!row) throw new Error('未找到分组行')
    await userEvent.click(within(row).getByRole('button', { name: 'Expand' }))
    expect(await screen.findByText('验证渠道')).toBeInTheDocument()
    expect(
      requests
        .find((params) => params.get('dimension') === 'group_channel')
        ?.get('groups')
    ).toBe('legacy-enterprise')
    name = '改名后的企业专属模型稳定性观测分组'.repeat(4)
    rerender(view(1))
    expect(await screen.findByTitle(name)).toHaveTextContent(name)
    expect(screen.queryByText('当前企业分组')).not.toBeInTheDocument()
    expect(screen.queryByText('legacy-enterprise')).not.toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByText('验证渠道')).not.toBeInTheDocument()
    )
  })

  it('名称缺失或未知时回退原标识，无标识时展示未标记状态', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          dimension: 'group',
          total: 3,
          page: 1,
          page_size: 30,
          items: [
            {
              key: 'deleted',
              group: 'deleted-code',
              group_name: '',
              windows: [],
            },
            { key: 'unknown', group: 'unknown-code', windows: [] },
            { key: 'none', windows: [] },
          ],
        },
      },
    })
    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <StabilityView filters={filters} refreshKey={0} onReset={() => {}} />
      </QueryClientProvider>
    )
    expect(await screen.findByText('deleted-code')).toBeInTheDocument()
    expect(screen.getByText('unknown-code')).toBeInTheDocument()
    expect(screen.getByText('Unlabelled group')).toBeInTheDocument()
  })
})
