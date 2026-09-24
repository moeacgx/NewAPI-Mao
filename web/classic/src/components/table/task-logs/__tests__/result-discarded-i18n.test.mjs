import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

test('同步结果未保留的标题与说明覆盖八个Classic语言', async () => {
  const keys = [
    'Result not retained',
    'The synchronous result was returned through the API and was not saved for later viewing',
  ];
  for (const locale of ['en', 'zh', 'zh-CN', 'zh-TW', 'fr', 'ja', 'ru', 'vi']) {
    const resource = JSON.parse(
      await readFile(
        new URL(
          '../../../../i18n/locales/' + locale + '.json',
          import.meta.url,
        ),
        'utf8',
      ),
    );
    for (const key of keys) {
      assert.ok(resource.translation[key]?.trim(), locale + ': ' + key);
      if (locale !== 'en') assert.notEqual(resource.translation[key], key);
      assert.doesNotMatch(resource.translation[key], /\?{2,}|\ufffd/);
    }
  }
});
