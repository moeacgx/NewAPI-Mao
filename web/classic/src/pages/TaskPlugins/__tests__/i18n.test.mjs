import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

test('任务插件页面和渠道选择的文案在八个 Classic translation 命名空间内完整', async () => {
  const sources = [
    '../index.jsx',
    '../PluginDetails.jsx',
    '../api.js',
    '../Marketplace.jsx',
    '../MarketplaceSources.jsx',
    '../marketplace-utils.js',
    '../../../components/table/channels/modals/TaskPluginSelect.jsx',
  ];
  const keys = new Set();
  for (const source of sources) {
    const text = await readFile(new URL(source, import.meta.url), 'utf8');
    for (const match of text.matchAll(/\bt\(\s*'([^']+)'/g)) keys.add(match[1]);
  }
  assert.ok(keys.size > 30);
  for (const locale of ['en', 'zh', 'zh-CN', 'zh-TW', 'fr', 'ja', 'ru', 'vi']) {
    const data = JSON.parse(
      await readFile(
        new URL('../../../i18n/locales/' + locale + '.json', import.meta.url),
        'utf8',
      ),
    );
    assert.deepEqual(Object.keys(data), ['translation']);
    for (const key of keys) {
      assert.ok(data.translation[key]?.trim(), locale + ': ' + key);
      assert.deepEqual(
        [...data.translation[key].matchAll(/{{([^}]+)}}/g)]
          .map((m) => m[1])
          .sort(),
        [...key.matchAll(/{{([^}]+)}}/g)].map((m) => m[1]).sort(),
      );
    }
    if (locale.startsWith('zh'))
      assert.notEqual(data.translation['Enabled channels'], 'Enabled channels');
    if (locale !== 'en') {
      for (const key of [
        'Upload custom task plugin',
        'Manage plugin sources',
        'Plugin source',
        'Review and install',
        'Save plugin sources',
      ]) {
        assert.notEqual(data.translation[key], key, locale + ': ' + key);
        assert.doesNotMatch(
          data.translation[key],
          /\?{2,}|\ufffd/,
          locale + ': ' + key,
        );
      }
    }
  }
});
