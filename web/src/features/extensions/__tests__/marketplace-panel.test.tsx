import { webcrypto } from 'node:crypto'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import {
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { Extensions } from '../index'
import { EXTENSION_CATALOG_URL } from '../marketplace-download'

const bytes = new Uint8Array([80, 75, 3, 4])
let sha256: string
const entry = () => ({
  id: 'demo',
  name: 'Demo extension',
  version: '0.2.0',
  path: 'published/demo/0.2.0/demo-0.2.0.zip',
  sha256,
  size: bytes.length,
  host: { min: 'v1.0.0' },
  status: 'requires-host-upgrade',
  compatibility: 'Requires guard support',
})
const catalog = () => ({
  catalogVersion: 1,
  purpose: 'extension-catalog',
  name: 'Extensions',
  modules: [entry()],
})
const ok = (data: unknown) => ({ data: { success: true, data } })
let client: QueryClient
beforeAll(async () => {
  sha256 = Buffer.from(
    await webcrypto.subtle.digest('SHA-256', bytes)
  ).toString('hex')
})
beforeEach(() => {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'root', role: 100 })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/extension-admin/marketplace') {
      return ok({
        catalog_url: EXTENSION_CATALOG_URL,
        repository_url: 'https://github.com/moeacgx/maolaonewapi-extensions',
        host_version: 'v1.0.0',
        max_archive_bytes: 104857600,
      })
    }
    return ok({ root: 'data/modules', modules: [] })
  })
  vi.spyOn(api, 'post').mockResolvedValue(ok({}))
  vi.stubGlobal('crypto', webcrypto)
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(catalog())))
      .mockResolvedValueOnce(new Response(bytes))
  )
})
afterEach(() => {
  client.clear()
  vi.unstubAllGlobals()
  useAuthStore.getState().auth.reset()
})

describe('online extension installation', () => {
  it('confirms a specific version then uploads verified bytes and refreshes extension navigation', async () => {
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    render(
      <QueryClientProvider client={client}>
        <Extensions />
      </QueryClientProvider>
    )
    const install = await screen.findByRole('button', {
      name: 'Install Demo extension 0.2.0',
    })
    expect(screen.getByText('Host upgrade required')).toBeInTheDocument()
    expect(screen.getByText('Requires guard support')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Upload Module' })
    ).toBeInTheDocument()
    fireEvent.click(install)
    const dialog = screen.getByRole('alertdialog')
    expect(dialog).toHaveTextContent('Demo extension')
    expect(dialog).toHaveTextContent('0.2.0')
    expect(api.post).not.toHaveBeenCalled()
    expect(fetch).toHaveBeenCalledTimes(1)
    fireEvent.click(
      within(dialog).getByRole('button', { name: 'Confirm installation' })
    )
    await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1))
    const form = vi.mocked(api.post).mock.calls[0][1] as FormData
    expect(form.get('file')).toBeInstanceOf(File)
    expect((form.get('file') as File).name).toBe('demo-0.2.0.zip')
    expect(
      Object.fromEntries([...form.entries()].filter(([key]) => key !== 'file'))
    ).toEqual({
      archiveSha256: sha256,
      expectedId: 'demo',
      expectedVersion: '0.2.0',
      catalogUrl: EXTENSION_CATALOG_URL,
      archivePath: entry().path,
    })
    await waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ['extensions'] })
    )
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: ['extensions', 'admin'],
    })
  })
  it('does not download or install when the version confirmation is canceled', async () => {
    render(
      <QueryClientProvider client={client}>
        <Extensions />
      </QueryClientProvider>
    )
    fireEvent.click(
      await screen.findByRole('button', {
        name: 'Install Demo extension 0.2.0',
      })
    )
    fireEvent.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Cancel',
      })
    )
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(api.post).not.toHaveBeenCalled()
  })
  it('shows load failure with retry and retains the manual uploader', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new Error('Network unavailable'))
    )
    render(
      <QueryClientProvider client={client}>
        <Extensions />
      </QueryClientProvider>
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Failed to load online modules'
    )
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Upload Module' })
    ).toBeInTheDocument()
  })
  it('shows an empty catalog', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          new Response(JSON.stringify({ ...catalog(), modules: [] }))
        )
    )
    render(
      <QueryClientProvider client={client}>
        <Extensions />
      </QueryClientProvider>
    )
    expect(
      await screen.findByText('No online modules available')
    ).toBeInTheDocument()
  })
  it('does not expose management or fetch the catalog to an administrator', () => {
    useAuthStore.getState().auth.setUser({ id: 2, username: 'admin', role: 10 })
    render(
      <QueryClientProvider client={client}>
        <Extensions />
      </QueryClientProvider>
    )
    expect(
      screen.queryByRole('button', { name: 'Upload Module' })
    ).not.toBeInTheDocument()
    expect(screen.queryByText('Online modules')).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalled()
    expect(fetch).not.toHaveBeenCalled()
  })
  it('keeps installation errors visible and never uploads a hash mismatch', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(new Response(JSON.stringify(catalog())))
        .mockResolvedValueOnce(new Response(new Uint8Array([80, 75, 3, 5])))
    )
    render(
      <QueryClientProvider client={client}>
        <Extensions />
      </QueryClientProvider>
    )
    fireEvent.click(
      await screen.findByRole('button', {
        name: 'Install Demo extension 0.2.0',
      })
    )
    fireEvent.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Confirm installation',
      })
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Extension archive SHA-256 does not match'
    )
    expect(api.post).not.toHaveBeenCalled()
  })
  it('keeps the selected version visible when the host rejects the archive', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { success: false, message: 'Host capability is unavailable' },
    })
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    render(
      <QueryClientProvider client={client}>
        <Extensions />
      </QueryClientProvider>
    )
    fireEvent.click(
      await screen.findByRole('button', {
        name: 'Install Demo extension 0.2.0',
      })
    )
    fireEvent.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Confirm installation',
      })
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Host capability is unavailable'
    )
    expect(screen.getByRole('alertdialog')).toHaveTextContent('0.2.0')
    expect(invalidate).not.toHaveBeenCalled()
  })
})
