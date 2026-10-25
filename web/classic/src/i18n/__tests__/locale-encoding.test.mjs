import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';

const root = new URL('../locales/', import.meta.url);

// 经非 UTF-8 代码页写入的译文会把每个非 ASCII 字符替换成 "?"，界面显示成一串问号。
// 特征：整条译文没有任何非 ASCII 字符，却比原文多出紧贴文字的 "?"；
// 法语问句在 "?" 前有空格，不属于此类。
const isEncodingDamaged = (key, value) => {
  const hasNonAscii = [...value].some((char) => char.codePointAt(0) > 0x7f);
  if (!value.includes('?') || hasNonAscii) return false;
  const keyMarks = (key.match(/[?？]/g) || []).length;
  const valueMarks = (value.match(/\?/g) || []).length;
  return valueMarks > keyMarks && /(^|\S)\?/.test(value);
};

test('Classic 译文没有被非 UTF-8 写入替换成问号', () => {
  const damaged = [];
  for (const file of readdirSync(root).filter(
    (name) => name.endsWith('.json') && name !== 'en.json',
  )) {
    const data = JSON.parse(readFileSync(new URL(file, root), 'utf8'));
    for (const [key, value] of Object.entries(data.translation)) {
      if (typeof value === 'string' && isEncodingDamaged(key, value)) {
        damaged.push(`${file}: ${key} => ${value}`);
      }
    }
  }
  assert.deepEqual(damaged, []);
});
