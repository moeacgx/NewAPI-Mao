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
import { readFileSync } from 'node:fs'

import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query'
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AxiosError, CanceledError } from 'axios'
import { toast } from 'sonner'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { ApiKeyApiAddresses } from '../api-key-api-addresses'
import { ApiKeysPrimaryButtons } from '../api-keys-primary-buttons'
import { ApiKeysProvider } from '../api-keys-provider'

const originalExecCommand = Object.getOwnPropertyDescriptor(
  document,
  'execCommand'
)

afterEach(() => {
  if (originalExecCommand) {
    Object.defineProperty(document, 'execCommand', originalExecCommand)
  } else {
    Reflect.deleteProperty(document, 'execCommand')
  }
})

function renderAddresses() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ApiKeyApiAddresses />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  vi.spyOn(toast, 'success').mockImplementation(() => 1)
  vi.spyOn(toast, 'error').mockImplementation(() => 1)
})

test('令牌页操作区同时提供地址和创建入口，初始不读取密钥', async () => {
  const get = vi.spyOn(api, 'get')
  const post = vi.spyOn(api, 'post')
  const client = new QueryClient()
  render(
    <QueryClientProvider client={client}>
      <ApiKeysProvider>
        <ApiKeysPrimaryButtons />
      </ApiKeysProvider>
    </QueryClientProvider>
  )
  expect(screen.getByRole('button', { name: 'API addresses' })).toBeEnabled()
  expect(screen.getByRole('button', { name: 'Create API Key' })).toBeEnabled()
  expect(get).not.toHaveBeenCalled()
  expect(post).not.toHaveBeenCalled()
})

test('打开入口才读取公开状态并展示多线路，复制保留原始路径及查询参数', async () => {
  const user = userEvent.setup()
  const write = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        api_info: [
          {
            route: 'Main route',
            url: 'https://api.example.test/custom/?a=1',
            description: 'Primary endpoint',
          },
          { route: 'Backup route', url: 'https://backup.example.test/' },
        ],
      },
    },
  })
  renderAddresses()
  expect(get).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button', { name: 'API addresses' }))
  const main = await screen.findByRole('listitem', { name: 'Main route' })
  expect(within(main).getByText('Primary endpoint')).toBeInTheDocument()
  expect(screen.getByText('Backup route')).toBeInTheDocument()
  await user.click(within(main).getByRole('button', { name: 'Copy address' }))
  expect(write).toHaveBeenCalledWith('https://api.example.test/custom/?a=1')
  expect(toast.success).toHaveBeenCalledWith('Copied successfully')
  expect(get.mock.calls.every(([url]) => url === '/api/status')).toBe(true)
})

test.each([
  [
    '列表关闭',
    {
      api_info_enabled: false,
      api_info: [{ url: 'https://hidden.example.test' }],
      server_address: ' https://site.example.test/path/ ',
    },
    'https://site.example.test/path/',
    'Default API address',
  ],
  [
    '列表无有效地址',
    {
      api_info: [null, { url: '' }, { url: 12 }],
      server_address: 'https://site.example.test',
    },
    'https://site.example.test',
    'Default API address',
  ],
  [
    '列表缺失及站点为空',
    { server_address: '  ' },
    window.location.origin,
    'Current domain',
  ],
  [
    '列表类型异常',
    { api_info: {}, server_address: null },
    window.location.origin,
    'Current domain',
  ],
])('%s时遵循两级回退', async (_name, status, address, label) => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: status },
  })
  const user = userEvent.setup()
  renderAddresses()
  await user.click(screen.getByRole('button', { name: 'API addresses' }))
  expect(await screen.findByText(address)).toBeInTheDocument()
  expect(screen.getByText(label)).toBeInTheDocument()
  expect(
    screen.queryByText('https://hidden.example.test')
  ).not.toBeInTheDocument()
})

test('加载中显示状态，请求失败可重试且不会伪装为当前域名', async () => {
  let reject!: (reason: Error) => void
  const get = vi.spyOn(api, 'get').mockImplementationOnce(
    () =>
      new Promise((_resolve, rejectPromise) => {
        reject = rejectPromise
      })
  )
  const user = userEvent.setup()
  renderAddresses()
  await user.click(screen.getByRole('button', { name: 'API addresses' }))
  expect(screen.getByRole('status')).toHaveTextContent('Loading...')
  await act(async () => reject(new Error('offline')))
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Failed to load API addresses'
  )
  expect(screen.queryByText(window.location.origin)).not.toBeInTheDocument()
  get.mockResolvedValueOnce({ data: { success: true, data: {} } })
  await user.click(screen.getByRole('button', { name: 'Retry' }))
  expect(await screen.findByText(window.location.origin)).toBeInTheDocument()
})

test('接口业务失败时提供重试而非误报默认地址', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: false, message: 'failed' },
  })
  const user = userEvent.setup()
  renderAddresses()
  await user.click(screen.getByRole('button', { name: 'API addresses' }))
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Failed to load API addresses'
  )
  expect(screen.getByRole('button', { name: 'Retry' })).toBeEnabled()
})

test('剪贴板及兼容复制均失败时反馈失败', async () => {
  const user = userEvent.setup()
  vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(
    new Error('denied')
  )
  Object.defineProperty(document, 'execCommand', {
    configurable: true,
    value: vi.fn(() => false),
  })
  vi.spyOn(console, 'warn').mockImplementation(() => undefined)
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: {} } })
  renderAddresses()
  await user.click(screen.getByRole('button', { name: 'API addresses' }))
  await user.click(await screen.findByRole('button', { name: 'Copy address' }))
  expect(toast.error).toHaveBeenCalledWith('Copy failed')
  expect(toast.success).not.toHaveBeenCalled()
})

test('键盘打开关闭恢复焦点，重新打开刷新配置', async () => {
  const get = vi
    .spyOn(api, 'get')
    .mockResolvedValue({ data: { success: true, data: {} } })
  const user = userEvent.setup()
  renderAddresses()
  const trigger = screen.getByRole('button', { name: 'API addresses' })
  trigger.focus()
  await user.keyboard('{Enter}')
  expect(await screen.findByText(window.location.origin)).toBeInTheDocument()
  expect(trigger).toHaveAttribute('aria-expanded', 'true')
  await user.keyboard('{Escape}')
  await waitFor(() => expect(trigger).toHaveFocus())
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
  get.mockResolvedValue({
    data: {
      success: true,
      data: { server_address: 'https://changed.example.test' },
    },
  })
  await user.keyboard('{Enter}')
  expect(
    await screen.findByText('https://changed.example.test')
  ).toBeInTheDocument()
})

// 提取当前入口的真实回调，避免测试复制旧规则而遗漏应用级导航。
function createApplicationClient() {
  const mainSource = readFileSync('src/main.tsx', 'utf8')
  const callback = mainSource.match(
    /queryCache: new QueryCache\(\{\s*onError: (\(error\) => \{[\s\S]*?\n {4}\}),/u
  )?.[1]
  if (!callback) throw new Error('Cannot locate application QueryCache.onError')
  const navigate = vi.fn()
  const onError = vi.fn(
    new Function(
      'AxiosError',
      'toast',
      'i18next',
      'router',
      `return (${callback})`
    )(AxiosError, toast, { t: (key: string) => key }, { navigate }) as (
      error: Error
    ) => void
  )
  const client = new QueryClient({
    queryCache: new QueryCache({ onError }),
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return { client, navigate, onError }
}

test.each([500, 503])(
  'HTTP %s 在真实全局错误处理下保留局部重试并恢复',
  async (status) => {
    const { client, navigate } = createApplicationClient()
    const failure = new AxiosError('server unavailable', 'ERR_BAD_RESPONSE')
    failure.response = { status } as NonNullable<AxiosError['response']>
    const get = vi
      .spyOn(api, 'get')
      .mockRejectedValueOnce(failure)
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: { server_address: 'https://recovered.example.test' },
        },
      })
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={client}>
        <ApiKeyApiAddresses />
      </QueryClientProvider>
    )
    await user.click(screen.getByRole('button', { name: 'API addresses' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Failed to load API addresses'
    )
    expect(navigate).not.toHaveBeenCalled()
    expect(toast.error).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(
      await screen.findByText('https://recovered.example.test')
    ).toBeInTheDocument()
    expect(get).toHaveBeenCalledTimes(2)
    expect(navigate).not.toHaveBeenCalled()
  }
)

test.each([401, 403])(
  'HTTP %s 保留认证错误供原有处理链使用',
  async (status) => {
    const { client, onError } = createApplicationClient()
    const failure = new AxiosError('authentication failure', 'ERR_BAD_REQUEST')
    failure.response = { status } as NonNullable<AxiosError['response']>
    vi.spyOn(api, 'get').mockRejectedValue(failure)
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={client}>
        <ApiKeyApiAddresses />
      </QueryClientProvider>
    )
    await user.click(screen.getByRole('button', { name: 'API addresses' }))
    await screen.findByRole('alert')
    expect(onError.mock.calls[0][0]).toBe(failure)
    expect(client.getQueryState(['keys', 'api-addresses'])?.error).toBe(failure)
  }
)

test('卸载时取消正在加载的请求，不触发全局错误处理', async () => {
  const { client, onError } = createApplicationClient()
  let signal: AbortSignal | undefined
  vi.spyOn(api, 'get').mockImplementation((_url, config) => {
    signal = config?.signal as AbortSignal | undefined
    return new Promise((_resolve, reject) => {
      signal?.addEventListener('abort', () => reject(new CanceledError()), {
        once: true,
      })
    })
  })
  const user = userEvent.setup()
  const view = render(
    <QueryClientProvider client={client}>
      <ApiKeyApiAddresses />
    </QueryClientProvider>
  )
  await user.click(screen.getByRole('button', { name: 'API addresses' }))
  expect(screen.getByRole('status')).toBeInTheDocument()
  expect(signal).toBeDefined()
  await act(async () => view.unmount())
  expect(signal?.aborted).toBe(true)
  expect(onError).not.toHaveBeenCalled()
})

test('Axios 取消错误保持原对象，不归一化为服务器故障', async () => {
  const { client, onError, navigate } = createApplicationClient()
  const canceled = new CanceledError('request canceled')
  vi.spyOn(api, 'get').mockRejectedValue(canceled)
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={client}>
      <ApiKeyApiAddresses />
    </QueryClientProvider>
  )
  await user.click(screen.getByRole('button', { name: 'API addresses' }))
  await waitFor(() => expect(onError).toHaveBeenCalled())
  expect(onError.mock.calls[0][0]).toBe(canceled)
  expect(navigate).not.toHaveBeenCalled()
})
