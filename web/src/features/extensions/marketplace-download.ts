import type { ExtensionHostCompat } from './types'

export const EXTENSION_CATALOG_URL =
  'https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json'
export const MAX_EXTENSION_ARCHIVE_BYTES = 100 * 1024 * 1024
const MAX_CATALOG_BYTES = 2 * 1024 * 1024
const moduleIdPattern = /^[a-z0-9][a-z0-9_-]{0,63}$/
const versionPattern = /^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$/

export interface ExtensionCatalogModule {
  id: string
  name: string
  version: string
  path: string
  sha256: string
  size: number
  host: ExtensionHostCompat
  description?: string
  compatibility?: string
  status?: string
  verification?: string
  capabilities?: string[]
}

export interface ExtensionCatalog {
  catalogVersion: 1
  purpose: 'extension-catalog'
  name: string
  modules: ExtensionCatalogModule[]
}

export interface ExtensionMarketplaceConfig {
  catalog_url: string
  repository_url: string
  host_version: string
  max_archive_bytes: number
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export function resolveExtensionArchiveUrl(
  catalogUrl: string,
  entry: Pick<ExtensionCatalogModule, 'id' | 'version' | 'path'>
): string {
  if (
    catalogUrl !== EXTENSION_CATALOG_URL ||
    !moduleIdPattern.test(entry.id) ||
    !versionPattern.test(entry.version) ||
    entry.version.includes('..') ||
    entry.path !==
      `published/${entry.id}/${entry.version}/${entry.id}-${entry.version}.zip`
  ) {
    throw new Error('Invalid extension archive path')
  }
  const url = new URL(entry.path, catalogUrl)
  const directory = new URL('.', catalogUrl)
  if (
    url.origin !== directory.origin ||
    !url.pathname.startsWith(directory.pathname) ||
    url.search ||
    url.hash
  ) {
    throw new Error('Invalid extension archive path')
  }
  return url.href
}

export function parseExtensionCatalog(
  value: unknown,
  catalogUrl: string
): ExtensionCatalog {
  if (
    !isObject(value) ||
    value.catalogVersion !== 1 ||
    value.purpose !== 'extension-catalog' ||
    typeof value.name !== 'string' ||
    !value.name.trim() ||
    !Array.isArray(value.modules)
  ) {
    throw new Error('Invalid extension catalog')
  }
  const identities = new Set<string>()
  const modules = value.modules.map((item): ExtensionCatalogModule => {
    if (
      !isObject(item) ||
      typeof item.id !== 'string' ||
      typeof item.name !== 'string' ||
      !item.name.trim() ||
      typeof item.version !== 'string' ||
      typeof item.path !== 'string' ||
      typeof item.sha256 !== 'string' ||
      !/^[a-f0-9]{64}$/i.test(item.sha256) ||
      typeof item.size !== 'number' ||
      !Number.isSafeInteger(item.size) ||
      item.size <= 0 ||
      item.size > MAX_EXTENSION_ARCHIVE_BYTES ||
      !isObject(item.host)
    ) {
      throw new Error('Invalid extension catalog')
    }
    for (const key of [
      'description',
      'compatibility',
      'status',
      'verification',
    ]) {
      if (item[key] !== undefined && typeof item[key] !== 'string') {
        throw new Error('Invalid extension catalog')
      }
    }
    if (
      (item.host.min !== undefined && typeof item.host.min !== 'string') ||
      (item.host.max !== undefined && typeof item.host.max !== 'string') ||
      (item.capabilities !== undefined &&
        (!Array.isArray(item.capabilities) ||
          item.capabilities.some(
            (capability) => typeof capability !== 'string'
          )))
    ) {
      throw new Error('Invalid extension catalog')
    }
    const entry = item as unknown as ExtensionCatalogModule
    resolveExtensionArchiveUrl(catalogUrl, entry)
    const identity = `${entry.id}@${entry.version}`
    if (identities.has(identity)) {
      throw new Error('Duplicate extension version in catalog')
    }
    identities.add(identity)
    return entry
  })
  return {
    catalogVersion: 1,
    purpose: 'extension-catalog',
    name: value.name,
    modules,
  }
}

/** 外部下载独立于登录客户端，在消费响应流时限制真实字节。 */
async function downloadBytes(
  url: string,
  limit: number
): Promise<Uint8Array<ArrayBuffer>> {
  const response = await fetch(url, {
    credentials: 'omit',
    redirect: 'error',
    referrerPolicy: 'no-referrer',
    signal: AbortSignal.timeout(60000),
  })
  if (
    !response.ok ||
    response.redirected ||
    (response.url && response.url !== url)
  ) {
    throw new Error('Extension download failed')
  }
  if (Number(response.headers.get('content-length')) > limit) {
    await response.body?.cancel()
    throw new Error('Extension download exceeds the size limit')
  }
  if (!response.body) throw new Error('Extension download failed')
  const reader = response.body.getReader()
  const chunks: Uint8Array[] = []
  let size = 0
  try {
    while (true) {
      const result = await reader.read()
      if (result.done) break
      size += result.value.byteLength
      if (size > limit) {
        await reader.cancel()
        throw new Error('Extension download exceeds the size limit')
      }
      chunks.push(result.value)
    }
  } finally {
    reader.releaseLock()
  }
  const bytes = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) {
    bytes.set(chunk, offset)
    offset += chunk.byteLength
  }
  return bytes
}

export async function fetchExtensionCatalog(
  catalogUrl: string
): Promise<ExtensionCatalog> {
  if (catalogUrl !== EXTENSION_CATALOG_URL) {
    throw new Error('Invalid extension catalog URL')
  }
  const bytes = await downloadBytes(catalogUrl, MAX_CATALOG_BYTES)
  let value: unknown
  try {
    value = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes))
  } catch {
    throw new Error('Invalid extension catalog')
  }
  return parseExtensionCatalog(value, catalogUrl)
}

export async function downloadExtensionArchive(
  catalogUrl: string,
  entry: ExtensionCatalogModule,
  hostLimit: number
): Promise<File> {
  const url = resolveExtensionArchiveUrl(catalogUrl, entry)
  if (!/^[a-f0-9]{64}$/i.test(entry.sha256)) {
    throw new Error('Invalid extension SHA-256')
  }
  if (
    !Number.isSafeInteger(hostLimit) ||
    hostLimit <= 0 ||
    !Number.isSafeInteger(entry.size) ||
    entry.size <= 0 ||
    entry.size > Math.min(hostLimit, MAX_EXTENSION_ARCHIVE_BYTES)
  ) {
    throw new Error('Extension download exceeds the size limit')
  }
  const bytes = await downloadBytes(
    url,
    Math.min(hostLimit, MAX_EXTENSION_ARCHIVE_BYTES, entry.size)
  )
  if (bytes.length !== entry.size) {
    throw new Error('Extension archive size does not match')
  }
  if (!globalThis.crypto?.subtle) {
    throw new Error('SHA-256 verification is unavailable')
  }
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  const actual = Array.from(new Uint8Array(digest), (byte) =>
    byte.toString(16).padStart(2, '0')
  ).join('')
  if (actual !== entry.sha256.toLowerCase()) {
    throw new Error('Extension archive SHA-256 does not match')
  }
  return new File([bytes], `${entry.id}-${entry.version}.zip`, {
    type: 'application/zip',
  })
}
