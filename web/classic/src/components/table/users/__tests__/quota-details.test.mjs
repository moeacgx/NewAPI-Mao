import test from 'node:test';
import assert from 'node:assert/strict';
import { formatQuotaDetail } from '../quotaDetails.js';
test('Tokens保留完整数字和负值，货币保留小额精度', () => {
  assert.equal(formatQuotaDetail(123456789, { type: 'TOKENS' }), '123456789');
  assert.equal(formatQuotaDetail(-123456, { type: 'TOKENS' }), '-123456');
  assert.equal(
    formatQuotaDetail(1, { type: 'USD', symbol: '$', rate: 1 }, 500000),
    '$0.000002',
  );
  assert.equal(
    formatQuotaDetail(500000, { type: 'CUSTOM', symbol: '¤', rate: 2 }, 500000),
    '¤2.000000',
  );
});
