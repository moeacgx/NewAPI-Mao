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

import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const keys = [
  '技能',
  '技能 {{count}}',
  '始终装载',
  '按需读取',
  '未用技能',
  '可用 {{count}} 个技能，本次未读取',
  '技能发现 · 第 {{round}} 轮',
  '私有技能发现轮次，单独计费',
  '已读技能',
];

test('使用日志技能标签文案覆盖八个Classic语言且保留插值', async () => {
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
    const filterLabel = resource.translation['Used skills'];
    assert.ok(filterLabel?.trim(), locale + ': Used skills');
    assert.doesNotMatch(filterLabel, /\?{2,}|�/);
    if (locale !== 'en') assert.notEqual(filterLabel, 'Used skills');
    for (const key of keys) {
      const value = resource.translation[key];
      assert.ok(value?.trim(), locale + ': ' + key);
      assert.doesNotMatch(value, /\?{2,}|�/);
      for (const placeholder of key.match(/{{\w+}}/g) ?? []) {
        assert.ok(value.includes(placeholder), locale + ': ' + key);
      }
      if (['zh', 'zh-CN'].includes(locale)) assert.equal(value, key);
      if (['en', 'fr', 'ja', 'ru', 'vi'].includes(locale)) {
        assert.notEqual(value, key);
      }
    }
  }
});
