import test from 'node:test';
import assert from 'node:assert/strict';
import {
  buildRequestRuleExpr,
  tryParseRequestRuleExpr,
  splitBillingExprAndRequestRules,
  combineBillingExpr,
} from '../requestRuleExpr.js';
for (const [start, end, op] of [
  ['9', '18', '&&'],
  ['21', '6', '||'],
  ['0', '24', '&&'],
  ['9', '9', '&&'],
]) {
  test(`时间范围${start}-${end}使用${op}并可往返编辑`, () => {
    const groups = [
      {
        conditions: [
          {
            source: 'time',
            timeFunc: 'hour',
            timezone: 'UTC',
            mode: 'range',
            rangeStart: start,
            rangeEnd: end,
          },
        ],
        multiplier: '0.5',
      },
    ];
    const expr = `(hour("UTC") >= ${start} ${op} hour("UTC") < ${end} ? 0.5 : 1)`;
    assert.equal(buildRequestRuleExpr(groups), expr);
    assert.equal(buildRequestRuleExpr(tryParseRequestRuleExpr(expr)), expr);
  });
}
test('历史不一致运算符保持原意', () => {
  const rules = '(hour("UTC") >= 9 || hour("UTC") < 18 ? 0.5 : 1)';
  assert.equal(tryParseRequestRuleExpr(rules), null);
  const whole = combineBillingExpr('tier("base", p * 2)', rules);
  const split = splitBillingExprAndRequestRules(whole);
  assert.equal(
    combineBillingExpr(split.billingExpr, split.requestRuleExpr),
    whole,
  );
});
test('混合条件保留同日范围', () => {
  const expr =
    '(header("x") == "yes" && hour("UTC") >= 9 && hour("UTC") < 24 ? 2 : 1)';
  const groups = tryParseRequestRuleExpr(expr);
  assert.equal(groups[0].conditions.length, 2);
  assert.equal(buildRequestRuleExpr(groups), expr);
});
