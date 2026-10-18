import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { MarketplacePanel } from '../components/marketplace-panel'
import { MarketplaceSourcesDialog } from '../components/marketplace-sources-dialog'
import { UploadPluginDialog } from '../components/upload-plugin-dialog'

const clients: QueryClient[] = []
afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})
function client() {
  const value = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  clients.push(value)
  return value
}
const sources = [
  { name: 'Official', index_url: 'https://official.example/index.json' },
  { name: 'MaoLao', index_url: 'https://maolao.example/plugins/index.json' },
]
const code = 'const meta = {key: "demo", version: "1.0.0"}'
async function index() {
  const digest = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(code)
  )
  const sha256 = Array.from(new Uint8Array(digest), (value) =>
    value.toString(16).padStart(2, '0')
  ).join('')
  return {
    indexVersion: 1,
    name: 'Stable',
    plugins: [
      {
        key: 'demo',
        name: 'Demo',
        latest: '1.0.0',
        versions: [
          {
            version: '1.0.0',
            path: 'demo/plugin.js',
            sha256,
            kind: 'task',
            minApiVersion: 1,
          },
        ],
      },
    ],
  }
}
function setupApi() {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      message: '',
      data: String(url).endsWith('/sources') ? sources : [],
    },
  }))
}

test('Admin browses selected sources independently without installation or source writes', async () => {
  setupApi()
  const catalog = await index()
  const fetchMock = vi
    .fn()
    .mockRejectedValueOnce(new Error('offline'))
    .mockResolvedValueOnce(new Response(JSON.stringify(catalog)))
  vi.stubGlobal('fetch', fetchMock)
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={client()}>
      <MarketplacePanel canManage={false} />
    </QueryClientProvider>
  )
  expect(await screen.findByText('Could not load this source')).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'MaoLao' }))
  expect(await screen.findByText('Demo')).toBeVisible()
  expect(screen.queryByRole('button', { name: 'Manage sources' })).toBeNull()
  expect(
    screen.queryByRole('button', { name: 'Review and install' })
  ).toBeNull()
  expect(fetchMock.mock.calls.map((call) => call[0])).toEqual(
    sources.map((source) => source.index_url)
  )
})

test('Root reviews verified source and submits identity, origin and hash without enabling', async () => {
  setupApi()
  const catalog = await index()
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(catalog)))
      .mockResolvedValueOnce(new Response(code))
  )
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true, data: {} } })
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={client()}>
      <MarketplacePanel canManage />
    </QueryClientProvider>
  )
  await user.click(
    await screen.findByRole('button', { name: 'Review and install' })
  )
  expect(await screen.findByDisplayValue(code)).toBeVisible()
  expect(post).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button', { name: /^Install$/ }))
  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  expect(post.mock.calls[0][1]).toEqual({
    source: code,
    sourceSha256: catalog.plugins[0].versions[0].sha256,
    expectedKey: 'demo',
    expectedVersion: '1.0.0',
    remark: 'Official',
    marketplace: { ...sources[0], path: 'demo/plugin.js' },
  })
})

test('selected older version downloads and installs its matching source, path and hash', async () => {
  setupApi()
  const catalog = await index()
  const olderSource = 'const meta = {key: "demo", version: "0.9.0"}'
  const digest = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(olderSource)
  )
  const olderHash = Array.from(new Uint8Array(digest), (value) =>
    value.toString(16).padStart(2, '0')
  ).join('')
  catalog.plugins[0].versions.push({
    version: '0.9.0',
    path: 'demo/0.9.0/plugin.js',
    sha256: olderHash,
    kind: 'task',
    minApiVersion: 1,
  })
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(new Response(JSON.stringify(catalog)))
    .mockResolvedValueOnce(new Response(olderSource))
  vi.stubGlobal('fetch', fetchMock)
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true, data: {} } })
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={client()}>
      <MarketplacePanel canManage />
    </QueryClientProvider>
  )
  await user.selectOptions(
    await screen.findByRole('combobox', { name: 'Demo Version' }),
    '0.9.0'
  )
  await user.click(screen.getByRole('button', { name: 'Review and install' }))
  expect(await screen.findByDisplayValue(olderSource)).toBeVisible()
  expect(fetchMock.mock.calls[1][0]).toBe(
    'https://official.example/demo/0.9.0/plugin.js'
  )
  await user.click(screen.getByRole('button', { name: /^Install$/ }))
  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  expect(post.mock.calls[0][1]).toEqual({
    source: olderSource,
    sourceSha256: olderHash,
    expectedKey: 'demo',
    expectedVersion: '0.9.0',
    remark: 'Official',
    marketplace: { ...sources[0], path: 'demo/0.9.0/plugin.js' },
  })
})

test('Root can add and remove sources and save the bare array', async () => {
  setupApi()
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true, data: [] } })
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={client()}>
      <MarketplaceSourcesDialog open onOpenChange={() => {}} />
    </QueryClientProvider>
  )
  await user.click(
    await screen.findByRole('button', { name: 'Remove source Official' })
  )
  await user.click(screen.getByRole('button', { name: 'Remove source MaoLao' }))
  await user.click(screen.getByRole('button', { name: 'Save' }))
  expect(put).toHaveBeenCalledWith(
    '/api/plugin/task/marketplace/sources',
    [],
    expect.anything()
  )
})

test('Failed source settings load never enables empty destructive save', async () => {
  vi.spyOn(api, 'get').mockRejectedValue(new Error('offline'))
  render(
    <QueryClientProvider client={client()}>
      <MarketplaceSourcesDialog open onOpenChange={() => {}} />
    </QueryClientProvider>
  )
  expect(
    await screen.findByText('Could not load marketplace sources')
  ).toBeVisible()
  expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
})

test('Temporary upload enforces UTF-8 byte length instead of character count', async () => {
  const post = vi.spyOn(api, 'post')
  render(
    <QueryClientProvider client={client()}>
      <UploadPluginDialog open onOpenChange={() => {}} />
    </QueryClientProvider>
  )
  fireEvent.change(screen.getByLabelText('Plugin source'), {
    target: { value: '中'.repeat(349526) },
  })
  expect(screen.getByRole('button', { name: /^Upload$/ })).toBeDisabled()
  expect(screen.getByRole('alert')).toHaveTextContent(
    'Plugin source must be 1 MiB or smaller.'
  )
  expect(post).not.toHaveBeenCalled()
})
