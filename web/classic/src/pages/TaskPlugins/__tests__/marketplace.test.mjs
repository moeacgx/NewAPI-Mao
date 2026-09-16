import assert from 'node:assert/strict';
import { createHash, webcrypto } from 'node:crypto';
import test from 'node:test';
import {
  downloadVerifiedPlugin,
  fetchMarketplaceText,
  parseMarketplaceIndex,
  resolveMarketplaceSource,
} from '../marketplace-utils.js';

const indexUrl = 'https://plugins.example/repository/index.json';

test('源码路径仅允许索引目录下的相对路径，拒绝编码逃逸和 URL 参数', () => {
  assert.equal(
    resolveMarketplaceSource(indexUrl, 'tasks/demo/1.0/plugin.js'),
    'https://plugins.example/repository/tasks/demo/1.0/plugin.js',
  );
  for (const path of [
    '../plugin.js',
    '%2e%2e/plugin.js',
    '%252e%252e/plugin.js',
    '//elsewhere.example/plugin.js',
    'https://plugins.example/repository/plugin.js',
    '/repository/plugin.js',
    'a\\plugin.js',
    'a%2f..%2fplugin.js',
    'plugin.js?token=x',
    'plugin.js#fragment',
    'plugin.js%3ftoken=x',
  ])
    assert.throws(
      () => resolveMarketplaceSource(indexUrl, path),
      /index directory/,
    );
  for (const url of [
    'http://plugins.example/index.json',
    'https://user:password@plugins.example/index.json',
    indexUrl + '?token=x',
    indexUrl + '#fragment',
  ])
    assert.throws(() => resolveMarketplaceSource(url), /HTTPS/);
});

test('索引保留任务插件，过滤未来 API 与非任务类型，拒绝未知索引版本', () => {
  const raw = {
    indexVersion: 1,
    plugins: [
      {
        key: 'demo',
        name: 'Demo',
        latest: '2.0',
        versions: [
          { version: '1.0', path: 'one.js', sha256: 'a'.repeat(64) },
          { version: '2.0', path: 'two.js', kind: 'task', minApiVersion: 1 },
          { version: 'future', path: 'future.js', minApiVersion: 2 },
          { version: 'tool', path: 'tool.js', kind: 'extension' },
        ],
      },
    ],
  };
  const parsed = parseMarketplaceIndex(raw);
  assert.equal(parsed[0].latest, '2.0');
  assert.deepEqual(
    parsed[0].versions.map((version) => version.version),
    ['1.0', '2.0'],
  );
  assert.throws(
    () => parseMarketplaceIndex({ ...raw, indexVersion: 2 }),
    /format/,
  );
  assert.deepEqual(parseMarketplaceIndex({ indexVersion: 1, plugins: [] }), []);
});

test('下载校验真实 UTF8 字节哈希且请求不携带凭据、不跟随重定向', async (t) => {
  const source = '\ufeff// 中文源码\nexport default {};';
  let request;
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    request = { url, ...options };
    return new Response(source);
  });
  const originalCrypto = Object.getOwnPropertyDescriptor(globalThis, 'crypto');
  Object.defineProperty(globalThis, 'crypto', {
    value: webcrypto,
    configurable: true,
  });
  t.after(() => Object.defineProperty(globalThis, 'crypto', originalCrypto));
  const version = {
    version: '1',
    path: 'plugin.js',
    sha256: createHash('sha256').update(source).digest('hex'),
  };
  assert.equal(await downloadVerifiedPlugin(indexUrl, version), source);
  assert.equal(request.credentials, 'omit');
  assert.equal(request.redirect, 'error');
  assert.equal(request.referrerPolicy, 'no-referrer');
  assert.equal(request.headers, undefined);
  await assert.rejects(
    downloadVerifiedPlugin(indexUrl, { ...version, sha256: '0'.repeat(64) }),
    /does not match/,
  );
  await assert.rejects(
    downloadVerifiedPlugin(indexUrl, { ...version, sha256: '' }),
    /required/,
  );
});

test('响应超限、损坏 UTF8 和重定向均不作为可安装源码返回', async (t) => {
  const responses = [
    new Response('oversized'),
    new Response(new Uint8Array([0xc3, 0x28])),
    { ok: true, redirected: true },
  ];
  t.mock.method(globalThis, 'fetch', async () => responses.shift());
  await assert.rejects(fetchMarketplaceText(indexUrl, 3), /size limit/);
  await assert.rejects(fetchMarketplaceText(indexUrl, 32), /encoded data/);
  await assert.rejects(fetchMarketplaceText(indexUrl, 32), /download/);
});
