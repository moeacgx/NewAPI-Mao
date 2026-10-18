/*
Copyright (C) 2025 QuantumNous

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

export const EXTENSION_CATALOG_URL =
  'https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json';
export const EXTENSION_REPOSITORY_URL =
  'https://github.com/moeacgx/maolaonewapi-extensions';
export const MAX_EXTENSION_ARCHIVE_BYTES = 100 * 1024 * 1024;
const MAX_CATALOG_BYTES = 2 * 1024 * 1024;

export function parseExtensionMarketplaceMetadata(raw) {
  if (
    raw?.catalog_url !== EXTENSION_CATALOG_URL ||
    raw.repository_url !== EXTENSION_REPOSITORY_URL ||
    typeof raw.host_version !== 'string' ||
    !raw.host_version.trim() ||
    !Number.isSafeInteger(raw.max_archive_bytes) ||
    raw.max_archive_bytes < 1 ||
    raw.max_archive_bytes > MAX_EXTENSION_ARCHIVE_BYTES
  )
    throw new Error('Invalid extension marketplace configuration.');
  return raw;
}

export function resolveExtensionArchiveUrl(catalogUrl, entry) {
  if (
    catalogUrl !== EXTENSION_CATALOG_URL ||
    !/^[a-z0-9][a-z0-9_-]{0,63}$/.test(entry?.id ?? '') ||
    !/^[0-9A-Za-z][0-9A-Za-z._+-]{0,127}$/.test(entry?.version ?? '') ||
    entry.version.includes('..') ||
    entry.path !==
      `published/${entry.id}/${entry.version}/${entry.id}-${entry.version}.zip`
  )
    throw new Error('Invalid extension archive path.');
  const base = new URL(catalogUrl);
  const url = new URL(entry.path, base);
  const directory = base.pathname.slice(0, base.pathname.lastIndexOf('/') + 1);
  if (
    url.origin !== base.origin ||
    !url.pathname.startsWith(directory) ||
    url.search ||
    url.hash
  ) {
    throw new Error('Invalid extension archive path.');
  }
  return url.href;
}

// 清单只作为数据读取，不执行远程源码，也不使用清单提供的任意下载 URL。
export function parseExtensionCatalog(raw, catalogUrl) {
  if (
    raw?.catalogVersion !== 1 ||
    raw.purpose !== 'extension-catalog' ||
    typeof raw.name !== 'string' ||
    !raw.name.trim() ||
    !Array.isArray(raw.modules)
  ) {
    throw new Error('Invalid extension catalog.');
  }
  const seen = new Set();
  return raw.modules.map((entry) => {
    if (
      !entry ||
      typeof entry.name !== 'string' ||
      !entry.name.trim() ||
      typeof entry.id !== 'string' ||
      typeof entry.version !== 'string' ||
      typeof entry.sha256 !== 'string' ||
      !/^[0-9a-f]{64}$/i.test(entry.sha256) ||
      !Number.isSafeInteger(entry.size) ||
      entry.size <= 0 ||
      entry.size > MAX_EXTENSION_ARCHIVE_BYTES ||
      !entry.host ||
      typeof entry.host !== 'object' ||
      Array.isArray(entry.host) ||
      (entry.host.min !== undefined &&
        (typeof entry.host.min !== 'string' || !entry.host.min.trim())) ||
      (entry.host.max !== undefined &&
        (typeof entry.host.max !== 'string' || !entry.host.max.trim())) ||
      ['description', 'compatibility', 'status', 'verification'].some(
        (key) => entry[key] !== undefined && typeof entry[key] !== 'string',
      ) ||
      (entry.capabilities !== undefined &&
        (!Array.isArray(entry.capabilities) ||
          !entry.capabilities.every((item) => typeof item === 'string')))
    )
      throw new Error('Invalid extension catalog.');
    try {
      resolveExtensionArchiveUrl(catalogUrl, entry);
    } catch {
      throw new Error('Invalid extension catalog.');
    }
    const identity = `${entry.id}:${entry.version}`;
    if (seen.has(identity)) throw new Error('Invalid extension catalog.');
    seen.add(identity);
    return { ...entry, sha256: entry.sha256.toLowerCase() };
  });
}

// Content-Length 仅用于提前拒绝，实际读取流仍逐块累计字节数。
export async function fetchExtensionBytes(url, maximumBytes, signal) {
  const controller = new AbortController();
  const abort = () => controller.abort();
  if (signal?.aborted) abort();
  signal?.addEventListener('abort', abort, { once: true });
  const timer = setTimeout(abort, 90000);
  try {
    const response = await fetch(url, {
      signal: controller.signal,
      credentials: 'omit',
      referrerPolicy: 'no-referrer',
      redirect: 'error',
      cache: 'no-store',
    });
    if (
      !response.ok ||
      response.redirected ||
      (response.url && response.url !== url) ||
      !response.body
    ) {
      throw new Error('Could not download the extension catalog or archive.');
    }
    if (Number(response.headers.get('content-length')) > maximumBytes) {
      await response.body.cancel();
      throw new Error('Extension download exceeds the size limit.');
    }
    const reader = response.body.getReader();
    const chunks = [];
    let length = 0;
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        length += value.byteLength;
        if (length > maximumBytes) {
          await reader.cancel();
          throw new Error('Extension download exceeds the size limit.');
        }
        chunks.push(value);
      }
    } finally {
      reader.releaseLock();
    }
    const bytes = new Uint8Array(length);
    let offset = 0;
    for (const chunk of chunks) {
      bytes.set(chunk, offset);
      offset += chunk.byteLength;
    }
    return bytes;
  } finally {
    clearTimeout(timer);
    signal?.removeEventListener('abort', abort);
  }
}

export async function loadExtensionCatalog(catalogUrl, signal) {
  if (catalogUrl !== EXTENSION_CATALOG_URL)
    throw new Error('Invalid extension marketplace configuration.');
  const bytes = await fetchExtensionBytes(
    catalogUrl,
    MAX_CATALOG_BYTES,
    signal,
  );
  let raw;
  try {
    raw = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes));
  } catch {
    throw new Error('Invalid extension catalog.');
  }
  return parseExtensionCatalog(raw, catalogUrl);
}

export async function downloadExtensionArchive(
  catalogUrl,
  entry,
  signal,
  maximumBytes = MAX_EXTENSION_ARCHIVE_BYTES,
) {
  const url = resolveExtensionArchiveUrl(catalogUrl, entry);
  if (!globalThis.crypto?.subtle)
    throw new Error('Use HTTPS to verify extension integrity.');
  const bytes = await fetchExtensionBytes(
    url,
    Math.min(maximumBytes, MAX_EXTENSION_ARCHIVE_BYTES),
    signal,
  );
  if (bytes.length !== entry.size)
    throw new Error('Extension archive size does not match the catalog.');
  const hash = Array.from(
    new Uint8Array(await globalThis.crypto.subtle.digest('SHA-256', bytes)),
    (value) => value.toString(16).padStart(2, '0'),
  ).join('');
  if (hash !== entry.sha256.toLowerCase())
    throw new Error('Extension SHA-256 does not match the catalog.');
  return new Blob([bytes], { type: 'application/zip' });
}

export function extensionMarketplaceError(error, t) {
  switch (error?.message) {
    case 'Invalid extension marketplace configuration.':
      return t('Invalid extension marketplace configuration.');
    case 'Invalid extension archive path.':
      return t('Invalid extension archive path.');
    case 'Invalid extension catalog.':
      return t('Invalid extension catalog.');
    case 'Extension download exceeds the size limit.':
      return t('Extension download exceeds the size limit.');
    case 'Extension archive size does not match the catalog.':
      return t('Extension archive size does not match the catalog.');
    case 'Extension SHA-256 does not match the catalog.':
      return t('Extension SHA-256 does not match the catalog.');
    case 'Use HTTPS to verify extension integrity.':
      return t('Use HTTPS to verify extension integrity.');
    default:
      return t('Could not download the extension catalog or archive.');
  }
}
