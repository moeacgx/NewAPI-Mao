import { parseMarketplaceIndex, resolvePluginSourceUrl } from './marketplace'

export const MAX_PLUGIN_SOURCE_BYTES = 1024 * 1024
const MAX_INDEX_BYTES = 2 * 1024 * 1024

export function validMarketplaceIndexUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return (
      value === value.trim() &&
      url.protocol === 'https:' &&
      !url.username &&
      !url.password &&
      !url.search &&
      !url.hash &&
      !/[\\\s]/.test(value)
    )
  } catch {
    return false
  }
}

/** 外部源请求不携带网关凭据，并在读取流时实施大小限制。 */
async function downloadMarketplaceBytes(
  url: string,
  limit: number
): Promise<Uint8Array<ArrayBuffer>> {
  if (!validMarketplaceIndexUrl(url)) throw new Error('Invalid marketplace URL')
  const response = await fetch(url, {
    credentials: 'omit',
    referrerPolicy: 'no-referrer',
    redirect: 'error',
    signal: AbortSignal.timeout(15000),
  })
  if (
    !response.ok ||
    response.redirected ||
    (response.url && response.url !== url)
  ) {
    throw new Error('Marketplace download failed')
  }
  if (Number(response.headers.get('content-length')) > limit) {
    await response.body?.cancel()
    throw new Error('Marketplace download is too large')
  }
  if (!response.body) throw new Error('Marketplace download failed')
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
        throw new Error('Marketplace download is too large')
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

export async function fetchMarketplaceIndex(indexUrl: string) {
  const bytes = await downloadMarketplaceBytes(indexUrl, MAX_INDEX_BYTES)
  return parseMarketplaceIndex(
    JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes))
  )
}

export async function downloadMarketplaceSource(
  indexUrl: string,
  path: string,
  sha256?: string
) {
  const url = resolvePluginSourceUrl(indexUrl, path)
  if (!url) throw new Error('Plugin path must stay inside the source directory')
  if (!sha256 || !/^[a-f0-9]{64}$/i.test(sha256)) {
    throw new Error('A valid SHA-256 is required to install')
  }
  const bytes = await downloadMarketplaceBytes(url, MAX_PLUGIN_SOURCE_BYTES)
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  const actual = Array.from(new Uint8Array(digest), (value) =>
    value.toString(16).padStart(2, '0')
  ).join('')
  if (actual !== sha256.toLowerCase()) {
    throw new Error('Plugin source SHA-256 does not match')
  }
  // 保留 BOM，确保提交文本的 UTF-8 字节与校验过的下载内容一致。
  const source = new TextDecoder('utf-8', {
    fatal: true,
    ignoreBOM: true,
  }).decode(bytes)
  return { source, sourceSha256: actual }
}
