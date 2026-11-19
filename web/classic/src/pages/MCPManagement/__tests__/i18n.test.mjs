import assert from 'node:assert/strict';
import fs from 'node:fs';
import test from 'node:test';

const directory = new URL('../', import.meta.url);
const keys = new Set();
for (const file of fs.readdirSync(directory)) {
  if (!/\.(jsx|js)$/.test(file)) continue;
  const source = fs.readFileSync(new URL(file, directory), 'utf8');
  for (const match of source.matchAll(/\bt\(\s*'([^']+)'/g)) keys.add(match[1]);
}

for (const locale of ['en', 'zh-CN', 'zh-TW', 'fr', 'ru', 'ja', 'vi']) {
  test(`${locale} 提供全部 MCP 文案并保留插值参数`, () => {
    const { translation } = JSON.parse(
      fs.readFileSync(
        new URL(`../../../i18n/locales/${locale}.json`, import.meta.url),
        'utf8',
      ),
    );
    for (const key of keys) {
      const value = translation[key];
      assert.ok(typeof value === 'string' && value.trim(), `${locale}: ${key}`);
      assert.ok(!value.includes('\uFFFD'), `${locale}: ${key}`);
      assert.deepEqual(
        [...value.matchAll(/{{\w+}}/g)].map(([value]) => value).sort(),
        [...key.matchAll(/{{\w+}}/g)].map(([value]) => value).sort(),
        `${locale}: ${key}`,
      );
    }
    assert.match(translation['Enter a key name of 1-100 characters.'], /100/);
    assert.equal(
      translation['Enter a key name of 1-64 characters.'],
      undefined,
    );
  });
}
