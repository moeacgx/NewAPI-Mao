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

export function resolveMarketplaceSource(indexUrl, path = '') {
  let base;
  try {
    base = new URL(indexUrl);
  } catch {
    throw new Error(
      'Plugin sources must use an HTTPS URL without credentials.',
    );
  }
  if (
    base.protocol !== 'https:' ||
    base.username ||
    base.password ||
    base.hash ||
    base.search
  )
    throw new Error(
      'Plugin sources must use an HTTPS URL without credentials.',
    );
  if (path) {
    let decoded;
    try {
      decoded = decodeURIComponent(path);
    } catch {
      throw new Error(
        'Plugin source paths must stay inside the index directory.',
      );
    }
    if (
      /^[a-z][a-z0-9+.-]*:/i.test(decoded) ||
      decoded.startsWith('/') ||
      /[\\?#%\s]/.test(decoded) ||
      decoded.split('/').some((segment) => segment === '.' || segment === '..')
    )
      throw new Error(
        'Plugin source paths must stay inside the index directory.',
      );
  }
  const target = new URL(path || indexUrl, base);
  if (
    target.origin !== base.origin ||
    target.username ||
    target.password ||
    target.hash ||
    target.search ||
    !target.pathname.startsWith(
      base.pathname.slice(0, base.pathname.lastIndexOf('/') + 1),
    )
  )
    throw new Error(
      'Plugin source paths must stay inside the index directory.',
    );
  return target.href;
}

// 索引不是可执行代码。仅提取页面展示和安装所需字段，忽略不支持的插件类型。
export function parseMarketplaceIndex(raw) {
  if (raw?.indexVersion !== 1 || !Array.isArray(raw.plugins))
    throw new Error('Unsupported plugin index format.');
  if (raw.plugins.length > 500)
    throw new Error('Plugin index contains too many entries.');
  const plugins = [];
  const seen = new Set();
  for (const plugin of raw.plugins) {
    if (
      !plugin ||
      typeof plugin.key !== 'string' ||
      !plugin.key.trim() ||
      seen.has(plugin.key) ||
      !Array.isArray(plugin.versions)
    )
      continue;
    const versions = plugin.versions
      .slice(0, 64)
      .filter((entry) => {
        return (
          typeof entry?.version === 'string' &&
          entry.version.trim() &&
          typeof entry.path === 'string' &&
          entry.path.trim() &&
          (!entry.kind || entry.kind === 'task') &&
          (entry.minApiVersion == null ||
            (Number.isInteger(entry.minApiVersion) &&
              entry.minApiVersion >= 0 &&
              entry.minApiVersion <= 1))
        );
      })
      .map((entry) => ({
        version: entry.version,
        path: entry.path,
        sha256: typeof entry.sha256 === 'string' ? entry.sha256 : '',
      }));
    if (!versions.length) continue;
    seen.add(plugin.key);
    plugins.push({
      key: plugin.key,
      name: typeof plugin.name === 'string' ? plugin.name : plugin.key,
      latest: versions.some((entry) => entry.version === plugin.latest)
        ? plugin.latest
        : versions[0].version,
      versions,
    });
  }
  return plugins;
}

// fetch 不复用带认证的 API 客户端，也不跟随源提供的重定向。
export async function fetchMarketplaceText(url, maximumBytes, signal) {
  const controller = new AbortController();
  const abort = () => controller.abort();
  if (signal?.aborted) abort();
  signal?.addEventListener('abort', abort, { once: true });
  const timer = setTimeout(abort, 15000);
  try {
    const response = await fetch(url, {
      signal: controller.signal,
      credentials: 'omit',
      referrerPolicy: 'no-referrer',
      redirect: 'error',
      cache: 'no-store',
    });
    if (!response.ok || response.redirected)
      throw new Error('Could not download the plugin source or index.');
    if (response.url && new URL(response.url).origin !== new URL(url).origin)
      throw new Error(
        'Plugin source paths must stay inside the index directory.',
      );
    if (Number(response.headers.get('content-length')) > maximumBytes)
      throw new Error('Plugin download exceeds the size limit.');
    if (!response.body)
      throw new Error('Could not download the plugin source or index.');
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
          throw new Error('Plugin download exceeds the size limit.');
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
    // 拒绝损坏的 UTF-8，避免解码替换字符导致安装字节与索引哈希不一致。
    return new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(
      bytes,
    );
  } finally {
    clearTimeout(timer);
    signal?.removeEventListener('abort', abort);
  }
}

export async function downloadVerifiedPlugin(indexUrl, version, signal) {
  if (!/^[a-f0-9]{64}$/i.test(version.sha256))
    throw new Error('A valid SHA-256 is required to install this version.');
  if (!globalThis.crypto?.subtle)
    throw new Error('Use HTTPS to verify plugin integrity.');
  const url = resolveMarketplaceSource(indexUrl, version.path);
  const source = await fetchMarketplaceText(url, 1024 * 1024, signal);
  const digest = await globalThis.crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(source),
  );
  const hash = Array.from(new Uint8Array(digest), (byte) =>
    byte.toString(16).padStart(2, '0'),
  ).join('');
  if (hash !== version.sha256.toLowerCase())
    throw new Error('Plugin SHA-256 does not match the index.');
  return source;
}

export function marketplaceError(error, t) {
  switch (error?.message) {
    case 'Plugin sources must use an HTTPS URL without credentials.':
      return t('Plugin sources must use an HTTPS URL without credentials.');
    case 'Plugin source paths must stay inside the index directory.':
      return t('Plugin source paths must stay inside the index directory.');
    case 'Unsupported plugin index format.':
      return t('Unsupported plugin index format.');
    case 'Plugin index contains too many entries.':
      return t('Plugin index contains too many entries.');
    case 'Plugin download exceeds the size limit.':
      return t('Plugin download exceeds the size limit.');
    case 'A valid SHA-256 is required to install this version.':
      return t('A valid SHA-256 is required to install this version.');
    case 'Use HTTPS to verify plugin integrity.':
      return t('Use HTTPS to verify plugin integrity.');
    case 'Plugin SHA-256 does not match the index.':
      return t('Plugin SHA-256 does not match the index.');
    case 'Provide up to 16 unique plugin sources with names and HTTPS index URLs.':
      return t(
        'Provide up to 16 unique plugin sources with names and HTTPS index URLs.',
      );
    default:
      return t(
        'Could not download the plugin source or index. Check the source URL and browser CORS access.',
      );
  }
}
