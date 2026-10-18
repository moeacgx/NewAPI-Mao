import test from 'node:test';
import assert from 'node:assert/strict';
import { buildRedemptionExport } from '../redemptionExport.js';
test('默认TXT仅导出代码，保留多行顺序', () => {
  assert.deepEqual(
    buildRedemptionExport({ codes: ['code-a', 'code-b'], name: '活动' }),
    { filename: '活动.txt', text: 'code-a\ncode-b\n' },
  );
});
test('Markdown显式附加名称额度并转义单元格，文件名不含路径', () => {
  const result = buildRedemptionExport({
    codes: ['a|b'],
    name: 'a/b\nx',
    quota: '$0.01',
    format: 'md',
    includeName: true,
    includeQuota: true,
  });
  assert.equal(result.filename, 'a_b x.md');
  assert.equal(
    result.text,
    '| Code | Name | Quota |\n| --- | --- | --- |\n| a\\|b | a/b x | $0.01 |\n',
  );
});

test('Markdown名称中的链接语法和实体保持原文', () => {
  const result = buildRedemptionExport({
    codes: ['code-a'],
    name: '[claim](https://example.test) &copy;',
    format: 'md',
    includeName: true,
  });
  assert.ok(
    result.text.includes('\\[claim\\](https://example.test) &amp;copy;'),
  );
});
