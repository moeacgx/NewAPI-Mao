import assert from 'node:assert/strict';
import { createHash, webcrypto } from 'node:crypto';
import test from 'node:test';
import fs from 'node:fs/promises';
import {
  EXTENSION_CATALOG_URL,
  EXTENSION_REPOSITORY_URL,
  MAX_EXTENSION_ARCHIVE_BYTES,
  parseExtensionMarketplaceMetadata,
  parseExtensionCatalog,
  resolveExtensionArchiveUrl,
  fetchExtensionBytes,
  downloadExtensionArchive,
  loadExtensionCatalog,
} from '../marketplace-utils.js';

const zip = new Uint8Array([80, 75, 3, 4, 0, 255]);
const entry = {
  id: 'demo',
  name: '演示模块',
  version: '0.2.0',
  path: 'published/demo/0.2.0/demo-0.2.0.zip',
  sha256: createHash('sha256').update(zip).digest('hex'),
  size: zip.length,
  host: { min: 'v1.0.0-rc.10.1.10.326' },
  status: 'requires-host-upgrade',
  compatibility: '需要宿主能力',
  source: {
    repository: 'https://github.com/example/source',
    path: 'src',
    working_tree: true,
  },
};
const catalog = {
  catalogVersion: 1,
  purpose: 'extension-catalog',
  name: '官方模块',
  modules: [entry],
};

test('固定来源必须完整有效，拒绝其他来源和无效下载上限', () => {
  const metadata = {
    catalog_url: EXTENSION_CATALOG_URL,
    repository_url: EXTENSION_REPOSITORY_URL,
    host_version: 'v1.0.0',
    max_archive_bytes: MAX_EXTENSION_ARCHIVE_BYTES,
  };
  assert.deepEqual(parseExtensionMarketplaceMetadata(metadata), metadata);
  for (const patch of [
    { catalog_url: 'https://evil.example/catalog.json' },
    { repository_url: 'javascript:alert(1)' },
    { max_archive_bytes: 0 },
  ]) {
    assert.throws(
      () => parseExtensionMarketplaceMetadata({ ...metadata, ...patch }),
      /configuration/,
    );
  }
});

test('发布路径严格绑定模块身份版本，不允许任何 URL 或编码逃逸', () => {
  assert.equal(
    resolveExtensionArchiveUrl(EXTENSION_CATALOG_URL, entry),
    new URL(entry.path, EXTENSION_CATALOG_URL).href,
  );
  for (const path of [
    '../demo.zip',
    '/published/demo/0.2.0/demo-0.2.0.zip',
    '//evil.example/demo.zip',
    'https://evil.example/demo.zip',
    'published/demo/0.2.0/other.zip',
    entry.path + '?token=x',
    entry.path + '#x',
    entry.path.replace('demo/', '%64emo/'),
    entry.path.replace('demo/', '%2564emo/'),
    entry.path.replace('/', '\\'),
    'published/demo/../demo.zip',
  ]) {
    assert.throws(
      () =>
        resolveExtensionArchiveUrl(EXTENSION_CATALOG_URL, { ...entry, path }),
      /path/,
    );
  }
  assert.throws(
    () =>
      resolveExtensionArchiveUrl(EXTENSION_CATALOG_URL, {
        ...entry,
        id: '../demo',
      }),
    /path/,
  );
});

test('清单保留明确版本与结构来源，空清单可用，重复或畸形条目整体拒绝', () => {
  assert.equal(
    parseExtensionCatalog(catalog, EXTENSION_CATALOG_URL)[0].status,
    'requires-host-upgrade',
  );
  assert.deepEqual(
    parseExtensionCatalog({ ...catalog, modules: [] }, EXTENSION_CATALOG_URL),
    [],
  );
  for (const invalid of [
    { ...catalog, purpose: 'task-plugin-index' },
    { ...catalog, catalogVersion: 2 },
    { ...catalog, modules: [entry, entry] },
    ...[
      { sha256: '' },
      { size: 0 },
      { size: MAX_EXTENSION_ARCHIVE_BYTES + 1 },
      { name: '' },
      { host: null },
    ].map((patch) => ({ ...catalog, modules: [{ ...entry, ...patch }] })),
  ])
    assert.throws(
      () => parseExtensionCatalog(invalid, EXTENSION_CATALOG_URL),
      /catalog/,
    );
});

test('支持以数字开头的模块及带构建标识版本，宿主范围可省略但目录点号拒绝', () => {
  const valid = {
    ...entry,
    id: '2demo',
    version: '1.0+build',
    host: {},
    path: 'published/2demo/1.0+build/2demo-1.0+build.zip',
  };
  assert.equal(
    parseExtensionCatalog(
      { ...catalog, modules: [valid] },
      EXTENSION_CATALOG_URL,
    )[0].version,
    '1.0+build',
  );
  assert.throws(
    () =>
      resolveExtensionArchiveUrl(EXTENSION_CATALOG_URL, {
        ...valid,
        version: '1..0',
        path: 'published/2demo/1..0/2demo-1..0.zip',
      }),
    /path/,
  );
});

test('在线安装文案在 Classic 七语与历史中文别名均有非空翻译', async () => {
  const sources = await Promise.all(
    ['ExtensionMarketplace.jsx', 'marketplace-utils.js'].map((file) =>
      fs.readFile(new URL('../' + file, import.meta.url), 'utf8'),
    ),
  );
  const keys = new Set(
    sources.flatMap((source) =>
      [...source.matchAll(/\bt\(\s*'([^']+)'/g)].map((match) => match[1]),
    ),
  );
  for (const locale of ['en', 'zh-CN', 'zh-TW', 'fr', 'ja', 'ru', 'vi', 'zh']) {
    const data = JSON.parse(
      await fs.readFile(
        new URL(`../../../i18n/locales/${locale}.json`, import.meta.url),
        'utf8',
      ),
    );
    for (const key of keys)
      assert.ok(
        typeof data.translation[key] === 'string' &&
          data.translation[key].trim(),
        `${locale}: ${key}`,
      );
  }
});

test('ZIP 下载不带凭据、不跟随跳转，真实字节校验通过才返回 Blob', async (t) => {
  let options;
  t.mock.method(globalThis, 'fetch', async (_url, settings) => {
    options = settings;
    return new Response(zip);
  });
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'crypto');
  Object.defineProperty(globalThis, 'crypto', {
    value: webcrypto,
    configurable: true,
  });
  t.after(() => Object.defineProperty(globalThis, 'crypto', descriptor));
  const blob = await downloadExtensionArchive(EXTENSION_CATALOG_URL, entry);
  assert.deepEqual(new Uint8Array(await blob.arrayBuffer()), zip);
  assert.equal(options.credentials, 'omit');
  assert.equal(options.redirect, 'error');
  assert.equal(options.referrerPolicy, 'no-referrer');
  assert.equal(options.headers, undefined);
  await assert.rejects(
    downloadExtensionArchive(EXTENSION_CATALOG_URL, {
      ...entry,
      sha256: '0'.repeat(64),
    }),
    /SHA-256/,
  );
  await assert.rejects(
    downloadExtensionArchive(EXTENSION_CATALOG_URL, {
      ...entry,
      size: zip.length + 1,
    }),
    /size/,
  );
});

test('实际流字节超过限额会取消读取，错误响应和隐式跳转均拒绝', async (t) => {
  let cancelled = false;
  const response = new Response(
    new ReadableStream({
      pull(controller) {
        controller.enqueue(new Uint8Array(4));
      },
      cancel() {
        cancelled = true;
      },
    }),
    { headers: { 'Content-Length': '1' } },
  );
  const replies = [
    response,
    new Response('', { status: 500 }),
    { ok: true, redirected: true },
    new Response('x', { headers: { 'Content-Length': '50' } }),
  ];
  t.mock.method(globalThis, 'fetch', async () => replies.shift());
  await assert.rejects(
    fetchExtensionBytes(EXTENSION_CATALOG_URL, 6),
    /size limit/,
  );
  assert.equal(cancelled, true);
  await assert.rejects(
    fetchExtensionBytes(EXTENSION_CATALOG_URL, 6),
    /download/,
  );
  await assert.rejects(
    fetchExtensionBytes(EXTENSION_CATALOG_URL, 6),
    /download/,
  );
  await assert.rejects(
    fetchExtensionBytes(EXTENSION_CATALOG_URL, 6),
    /size limit/,
  );
});

test('损坏 UTF-8 清单与实际超出 2 MiB 的清单均不能加载', async (t) => {
  const replies = [
    new Response(new Uint8Array([0xc3, 0x28])),
    new Response(new Uint8Array(2 * 1024 * 1024 + 1)),
  ];
  t.mock.method(globalThis, 'fetch', async () => replies.shift());
  await assert.rejects(loadExtensionCatalog(EXTENSION_CATALOG_URL), /catalog/);
  await assert.rejects(
    loadExtensionCatalog(EXTENSION_CATALOG_URL),
    /size limit/,
  );
});
