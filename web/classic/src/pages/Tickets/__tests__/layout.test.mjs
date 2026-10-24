import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import postcss from 'postcss';

test('工单页滚动时固定导航使用不透明主题底色且面板预留滚动定位空间', () => {
  const css = postcss.parse(
    readFileSync(new URL('../tickets.css', import.meta.url), 'utf8'),
  );
  const rules = new Map();
  css.walkRules((rule) => rules.set(rule.selector, rule));
  const header = rules.get(
    'body:has(.ticket-page) .classic-pricing-template-header',
  );
  assert.ok(header, '固定导航背景规则仅作用于工单页面');
  assert.ok(
    header.nodes.some(
      (node) =>
        node.prop === 'background' && node.value === 'var(--semi-color-bg-1)',
    ),
  );
  const panel = rules.get('.ticket-panel');
  assert.ok(
    panel.nodes.some(
      (node) => node.prop === 'scroll-margin-top' && node.value === '80px',
    ),
  );
});
