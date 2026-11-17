import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { PropsWithChildren } from 'react'
import { expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { getGroupDisplayName } from '../../lib/group-names'
import { usePricingData } from '../use-pricing-data'

it('真实价格查询改名重取保留原型自有名称，删除后回退且业务编码保持', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  let names = '{"__proto__":"原名称","legacy":"别名名称"}'
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data:
      url === '/api/pricing'
        ? {
            success: true,
            data: [
              { model_name: 'fixture', enable_groups: ['__proto__', 'legacy'] },
            ],
            vendors: [],
            group_ratio: {},
            group_names: JSON.parse(names),
            usable_group: {},
            supported_endpoint: {},
            auto_groups: [],
          }
        : { success: true, data: { price: 1, usd_exchange_rate: 1 } },
  }))
  function Wrapper(props: PropsWithChildren) {
    return (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    )
  }
  const hook = renderHook(() => usePricingData(), { wrapper: Wrapper })
  try {
    await waitFor(() =>
      expect(hook.result.current.groupNames['__proto__']).toBe('原名称')
    )
    names = '{"__proto__":"改名后","legacy":"别名改名后"}'
    await act(async () => {
      await hook.result.current.refetch()
    })
    await waitFor(() =>
      expect(
        getGroupDisplayName('__proto__', hook.result.current.groupNames)
      ).toBe('改名后')
    )
    expect(getGroupDisplayName('legacy', hook.result.current.groupNames)).toBe(
      '别名改名后'
    )
    names = '{}'
    await act(async () => {
      await hook.result.current.refetch()
    })
    await waitFor(() =>
      expect(
        getGroupDisplayName('__proto__', hook.result.current.groupNames)
      ).toBe('__proto__')
    )
    expect(hook.result.current.models[0].enable_groups).toEqual([
      '__proto__',
      'legacy',
    ])
  } finally {
    hook.unmount()
    client.clear()
  }
})
