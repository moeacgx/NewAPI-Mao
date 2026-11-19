import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import { AxiosError } from 'axios'
import type { PropsWithChildren } from 'react'
import { toast } from 'sonner'
import { expect, test, vi } from 'vitest'

import { api } from '@/lib/http-client'
import { useAuthStore } from '@/stores/auth-store'

import { useLogGroupOptions } from '../use-log-group-options'

test.each(
  [false, true].flatMap((isAdmin) =>
    ['old-public', '__proto__', 'constructor', 'toString'].map((alias) => ({
      isAdmin,
      alias,
    }))
  )
)(
  '管理视图=$isAdmin时改名失效后保留合法alias=$alias的当前名称与稳定code',
  async ({ isAdmin, alias }) => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const adapter = api.defaults.adapter
    const requests: string[] = []
    let name = '改名前当前名称'
    useAuthStore.getState().auth.setUser({
      id: 73,
      username: 'rename-reader',
      role: isAdmin ? 10 : 1,
      group: 'public',
    })
    // 保留真实客户端与查询缓存，只替换网络返回，映射包含合法自有的原型同名键。
    api.defaults.adapter = async (config) => {
      requests.push(config.url ?? '')
      const group = { id: 42, code: 'public', name }
      return {
        data: {
          success: true,
          data: isAdmin ? [group] : { public: group },
          group_names: Object.fromEntries(
            ['public', alias].map((key) => [key, name])
          ),
        },
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      }
    }
    function Wrapper({ children }: PropsWithChildren) {
      return (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      )
    }
    const view = renderHook(() => useLogGroupOptions(isAdmin), {
      wrapper: Wrapper,
    })
    try {
      await waitFor(() => expect(view.result.current.isSuccess).toBe(true))
      expect(Object.hasOwn(view.result.current.data?.names ?? {}, alias)).toBe(
        true
      )
      expect(view.result.current.data?.names?.[alias]).toBe('改名前当前名称')
      name = '改名后最新名称'
      await act(async () => {
        await client.invalidateQueries({
          queryKey: isAdmin
            ? ['system-settings', 'group-details']
            : ['user-groups'],
        })
      })
      await waitFor(() =>
        expect(view.result.current.data?.options).toEqual([
          { value: 'public', label: '改名后最新名称' },
        ])
      )
      expect(Object.hasOwn(view.result.current.data?.names ?? {}, alias)).toBe(
        true
      )
      expect(view.result.current.data?.names?.[alias]).toBe('改名后最新名称')
      const endpoint = isAdmin ? '/api/group/details' : '/api/user/self/groups'
      expect(requests).toEqual([endpoint, endpoint])
    } finally {
      view.unmount()
      client.clear()
      api.defaults.adapter = adapter
      useAuthStore.getState().auth.reset()
    }
  }
)

test('严格模式重新挂载后真实客户端仍能加载分组，不复用已取消请求', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const adapter = api.defaults.adapter
  const notification = vi.spyOn(toast, 'error')
  useAuthStore.getState().auth.setUser({
    id: 71,
    username: 'strict-reader',
    role: 1,
    group: 'public',
  })
  // 只替换网络适配层，保留真实客户端拦截器、请求去重和取消行为。
  api.defaults.adapter = async (config) => ({
    data: {
      success: true,
      data: { public: { id: 42, code: 'public', name: '当前分组名称' } },
      group_names: { public: '当前分组名称', 'old-public': '当前分组名称' },
    },
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  })
  function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  const view = renderHook(() => useLogGroupOptions(false), {
    wrapper: Wrapper,
    reactStrictMode: true,
  })
  try {
    await waitFor(() => expect(view.result.current.isSuccess).toBe(true))
    expect(view.result.current.data).toEqual({
      options: [{ value: 'public', label: '当前分组名称' }],
      names: { public: '当前分组名称', 'old-public': '当前分组名称' },
    })
    expect(notification).not.toHaveBeenCalled()
  } finally {
    view.unmount()
    client.clear()
    api.defaults.adapter = adapter
    useAuthStore.getState().auth.reset()
  }
})

test('服务器读取失败保留控件内的可恢复错误，重试能够重新获取名称', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const adapter = api.defaults.adapter
  const notification = vi.spyOn(toast, 'error')
  useAuthStore.getState().auth.setUser({
    id: 72,
    username: 'retry-reader',
    role: 1,
    group: 'public',
  })
  let failed = true
  api.defaults.adapter = async (config) => {
    if (failed) {
      throw new AxiosError(
        '暂时无法读取分组',
        'ERR_BAD_RESPONSE',
        config,
        null,
        {
          status: 500,
          statusText: 'Internal Server Error',
          data: {},
          headers: {},
          config,
        }
      )
    }
    return {
      status: 200,
      statusText: 'OK',
      data: { success: true, data: {}, group_names: {} },
      headers: {},
      config,
    }
  }
  function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  const view = renderHook(() => useLogGroupOptions(false), { wrapper: Wrapper })
  try {
    await waitFor(() => expect(view.result.current.isError).toBe(true))
    // 保留底层原因用于诊断，但不触发全局 Axios 500 路由跳转或重复通知。
    expect(view.result.current.error).not.toBeInstanceOf(AxiosError)
    expect(view.result.current.error?.cause).toBeInstanceOf(AxiosError)
    expect(notification).not.toHaveBeenCalled()
    failed = false
    await act(async () => {
      await view.result.current.refetch()
    })
    await waitFor(() => expect(view.result.current.isSuccess).toBe(true))
    expect(view.result.current.data).toEqual({ options: [], names: {} })
  } finally {
    view.unmount()
    client.clear()
    api.defaults.adapter = adapter
    useAuthStore.getState().auth.reset()
  }
})
