import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
const root = new URL('../../i18n/locales/', import.meta.url);
test('Classic本阶段新文案在所有实际语言文件中存在', () => {
  for (const file of readdirSync(root).filter((name) =>
    name.endsWith('.json'),
  )) {
    const data = JSON.parse(
      readFileSync(new URL(file, root), 'utf8'),
    ).translation;
    for (const key of [
      '时间范围',
      '额度详情',
      '导出兑换码',
      '文件格式',
      '包含名称',
      '包含额度',
    ])
      assert.ok(data[key], `${file}: ${key}`);
  }
});
