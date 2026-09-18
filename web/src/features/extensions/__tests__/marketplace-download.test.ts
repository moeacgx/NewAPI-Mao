import { webcrypto } from 'node:crypto'

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import {
  downloadExtensionArchive,
  fetchExtensionCatalog,
  parseExtensionCatalog,
  resolveExtensionArchiveUrl,
} from '../marketplace-download'

const catalogUrl =
  'https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json'
const bytes = new Uint8Array([80, 75, 3, 4])
let digest: string
const entry = () => ({
  id: 'demo',
  name: 'Demo module',
  version: '0.1.0',
  path: 'published/demo/0.1.0/demo-0.1.0.zip',
  sha256: digest,
  size: bytes.length,
  host: { min: 'v1.0.0' },
  compatibility: 'Requires native v1',
  status: 'development',
})
const catalog = () => ({
  catalogVersion: 1,
  purpose: 'extension-catalog',
  name: 'Modules',
  modules: [entry()],
})
beforeAll(async () => {
  digest = Buffer.from(
    await webcrypto.subtle.digest('SHA-256', bytes)
  ).toString('hex')
})
afterEach(() => vi.unstubAllGlobals())

describe('extension marketplace boundaries', () => {
  it.each([
    '../demo.zip',
    '/published/demo/0.1.0/demo-0.1.0.zip',
    'https://evil.example/a.zip',
    '//evil.example/a.zip',
    'published/demo/%2e%2e/a.zip',
    'published/demo/%252e%252e/a.zip',
    'published/demo/0.1.0/demo-0.1.0.zip?q=1',
    'published/demo/0.1.0/demo-0.1.0.zip#x',
    'published\\demo\\0.1.0\\demo-0.1.0.zip',
    'published/other/0.1.0/demo-0.1.0.zip',
  ])('rejects unsafe archive path %s', (path) => {
    expect(() =>
      resolveExtensionArchiveUrl(catalogUrl, { ...entry(), path })
    ).toThrow()
  })
  it.each([
    'http://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json',
    'https://user:pass@raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json',
    `${catalogUrl}?q=1`,
    `${catalogUrl}#x`,
  ])('rejects unsafe catalog URL %s', async (url) => {
    const fetcher = vi.fn()
    vi.stubGlobal('fetch', fetcher)
    await expect(fetchExtensionCatalog(url)).rejects.toThrow()
    expect(fetcher).not.toHaveBeenCalled()
  })
  it('accepts catalog versions and downloads exact ZIP bytes without credentials', async () => {
    vi.stubGlobal('crypto', webcrypto)
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(catalog())))
      .mockResolvedValueOnce(new Response(bytes))
    vi.stubGlobal('fetch', fetcher)
    const parsed = await fetchExtensionCatalog(catalogUrl)
    const archive = await downloadExtensionArchive(
      catalogUrl,
      parsed.modules[0],
      104857600
    )
    expect(archive.name).toBe('demo-0.1.0.zip')
    expect(archive.size).toBe(4)
    expect(fetcher).toHaveBeenCalledWith(
      catalogUrl,
      expect.objectContaining({
        credentials: 'omit',
        redirect: 'error',
        referrerPolicy: 'no-referrer',
      })
    )
    expect(fetcher).toHaveBeenLastCalledWith(
      new URL(entry().path, catalogUrl).href,
      expect.objectContaining({
        credentials: 'omit',
        redirect: 'error',
        referrerPolicy: 'no-referrer',
      })
    )
  })
  it.each([
    () => ({ ...catalog(), purpose: 'task-plugins' }),
    () => ({ ...catalog(), catalogVersion: 2 }),
    () => ({ ...catalog(), modules: [entry(), entry()] }),
    () => ({ ...catalog(), modules: [{ ...entry(), sha256: 'invalid' }] }),
    () => ({ ...catalog(), modules: [{ ...entry(), id: '../demo' }] }),
    () => ({ ...catalog(), modules: [{ ...entry(), id: 'a'.repeat(65) }] }),
    () => ({ ...catalog(), modules: [{ ...entry(), version: '..' }] }),
    () => ({ ...catalog(), modules: [{ ...entry(), host: null }] }),
    () => ({ ...catalog(), modules: [{ ...entry(), size: 104857601 }] }),
  ])(
    'rejects invalid catalog identity, metadata or duplicates',
    (candidate) => {
      expect(() => parseExtensionCatalog(candidate(), catalogUrl)).toThrow()
    }
  )
  it('rejects mismatched hashes and archive size', async () => {
    vi.stubGlobal('crypto', webcrypto)
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(async () => new Response(bytes))
    )
    await expect(
      downloadExtensionArchive(
        catalogUrl,
        { ...entry(), sha256: '0'.repeat(64) },
        104857600
      )
    ).rejects.toThrow('SHA-256')
    await expect(
      downloadExtensionArchive(catalogUrl, { ...entry(), size: 5 }, 104857600)
    ).rejects.toThrow('size')
  })
  it('limits streamed bytes even without Content-Length and cancels the stream', async () => {
    const cancel = vi.fn()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          new ReadableStream({
            start(controller) {
              controller.enqueue(new Uint8Array(8))
            },
            cancel,
          })
        )
      )
    )
    await expect(
      downloadExtensionArchive(catalogUrl, entry(), 4)
    ).rejects.toThrow('size')
    expect(cancel).toHaveBeenCalled()
  })
  it('rejects oversized catalogs, redirects and HTTP errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(new Uint8Array(2 * 1024 * 1024 + 1))
        )
        .mockResolvedValueOnce({ ok: true, redirected: true })
        .mockResolvedValueOnce(new Response('', { status: 404 }))
    )
    await expect(fetchExtensionCatalog(catalogUrl)).rejects.toThrow('size')
    await expect(fetchExtensionCatalog(catalogUrl)).rejects.toThrow('download')
    await expect(fetchExtensionCatalog(catalogUrl)).rejects.toThrow('download')
  })
  it('rejects missing hashes before downloading and refuses unavailable Web Crypto', async () => {
    const fetcher = vi.fn().mockImplementation(async () => new Response(bytes))
    vi.stubGlobal('fetch', fetcher)
    await expect(
      downloadExtensionArchive(
        catalogUrl,
        { ...entry(), sha256: '' },
        104857600
      )
    ).rejects.toThrow('SHA-256')
    expect(fetcher).not.toHaveBeenCalled()
    vi.stubGlobal('crypto', {})
    await expect(
      downloadExtensionArchive(catalogUrl, entry(), 104857600)
    ).rejects.toThrow('unavailable')
  })
  it('rejects changed response URLs and invalid UTF-8 catalogs', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce({
          ok: true,
          redirected: false,
          url: 'https://evil.example/catalog.json',
        })
        .mockResolvedValueOnce(new Response(new Uint8Array([255])))
    )
    await expect(fetchExtensionCatalog(catalogUrl)).rejects.toThrow('download')
    await expect(fetchExtensionCatalog(catalogUrl)).rejects.toThrow('catalog')
  })
  it('does not read a response that declares an oversized archive', async () => {
    const cancel = vi.fn()
    const response = {
      ok: true,
      redirected: false,
      url: '',
      headers: new Headers({ 'content-length': '104857601' }),
      body: { cancel },
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
    await expect(
      downloadExtensionArchive(catalogUrl, entry(), 104857600)
    ).rejects.toThrow('size')
    expect(cancel).toHaveBeenCalledTimes(1)
  })
})
