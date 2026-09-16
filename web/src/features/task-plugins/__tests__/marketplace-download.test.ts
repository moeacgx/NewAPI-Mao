import { afterEach, describe, expect, test, vi } from 'vitest'

import { resolvePluginSourceUrl } from '../lib/marketplace'
import {
  downloadMarketplaceSource,
  fetchMarketplaceIndex,
  validMarketplaceIndexUrl,
} from '../lib/marketplace-download'

const indexUrl = 'https://plugins.example/stable/index.json'
const source = '\uFEFFconst meta = {name:"中文"}'
const bytes = new TextEncoder().encode(source)
async function hash(value: Uint8Array<ArrayBuffer>) {
  return Array.from(
    new Uint8Array(await crypto.subtle.digest('SHA-256', value)),
    (byte) => byte.toString(16).padStart(2, '0')
  ).join('')
}
afterEach(() => vi.unstubAllGlobals())

describe('marketplace download boundaries', () => {
  test.each([
    '../plugin.js',
    '%2e%2e/plugin.js',
    '%252e%252e/plugin.js',
    '/plugin.js',
    '//evil.example/p.js',
    'https://plugins.example/stable/p.js',
    'x\\p.js',
    'p.js?token=foo',
    'p.js#fragment',
  ])('rejects unsafe path %s', (path) => {
    expect(resolvePluginSourceUrl(indexUrl, path)).toBeNull()
  })
  test.each([
    'http://plugins.example/index.json',
    'https://user:secret@plugins.example/index.json',
    'https://plugins.example/index.json?secret=x',
    'https://plugins.example/index.json#x',
  ])('rejects unsafe index %s', (url) => {
    expect(validMarketplaceIndexUrl(url)).toBe(false)
  })
  test('verifies original bytes, preserves BOM and omits credentials', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(bytes))
    vi.stubGlobal('fetch', fetchMock)
    const digest = await hash(bytes)
    expect(
      await downloadMarketplaceSource(indexUrl, 'demo/plugin.js', digest)
    ).toEqual({ source, sourceSha256: digest })
    expect(fetchMock).toHaveBeenCalledWith(
      'https://plugins.example/stable/demo/plugin.js',
      expect.objectContaining({
        credentials: 'omit',
        redirect: 'error',
        referrerPolicy: 'no-referrer',
      })
    )
  })
  test('rejects hash mismatch and missing hash without installing', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(bytes))
    vi.stubGlobal('fetch', fetchMock)
    await expect(
      downloadMarketplaceSource(indexUrl, 'plugin.js')
    ).rejects.toThrow('SHA-256')
    expect(fetchMock).not.toHaveBeenCalled()
    await expect(
      downloadMarketplaceSource(indexUrl, 'plugin.js', '0'.repeat(64))
    ).rejects.toThrow('does not match')
  })
  test('rejects invalid UTF-8 even when its bytes match the hash', async () => {
    const invalid = new Uint8Array([0xff])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(invalid)))
    await expect(
      downloadMarketplaceSource(indexUrl, 'plugin.js', await hash(invalid))
    ).rejects.toThrow()
  })
  test('rejects oversized streaming source without content-length', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(new Uint8Array(1024 * 1024 + 1)))
    )
    await expect(
      downloadMarketplaceSource(indexUrl, 'plugin.js', 'a'.repeat(64))
    ).rejects.toThrow('too large')
  })
  test('rejects oversized indexes and redirects', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response('{}', {
          headers: { 'content-length': String(2 * 1024 * 1024 + 1) },
        })
      )
      .mockResolvedValueOnce({ ok: true, redirected: true })
    vi.stubGlobal('fetch', fetchMock)
    await expect(fetchMarketplaceIndex(indexUrl)).rejects.toThrow('too large')
    await expect(fetchMarketplaceIndex(indexUrl)).rejects.toThrow('failed')
  })
})
