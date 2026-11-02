import assert from 'node:assert/strict';
import fs from 'node:fs';
import test from 'node:test';
import { JSDOM } from 'jsdom';

const css = fs.readFileSync(
  new URL('../telegram-editor.css', import.meta.url),
  'utf8',
);
const dom = new JSDOM('<style></style>');
dom.window.document.querySelector('style').textContent = css;
const rules = Array.from(dom.window.document.styleSheets[0].cssRules);

test('编辑和预览宽屏并排，600px 以下上下排列，两个面板均可收缩', () => {
  const desktop = rules.find(
    (rule) => rule.selectorText === '.classic-telegram-editor-layout',
  );
  const mobile = rules.find(
    (rule) => rule.conditionText === '(max-width: 600px)',
  );
  const mobileLayout = Array.from(mobile.cssRules).find(
    (rule) => rule.selectorText === '.classic-telegram-editor-layout',
  );
  assert.equal(
    desktop.style.getPropertyValue('grid-template-columns'),
    'repeat(2, minmax(0, 1fr))',
  );
  assert.equal(
    mobileLayout.style.getPropertyValue('grid-template-columns'),
    'minmax(0, 1fr)',
  );
  const panels = rules.find((rule) =>
    rule.selectorText?.includes('.classic-telegram-source,'),
  );
  assert.equal(panels.style.getPropertyValue('min-width'), '0');
  assert.equal(
    panels.style.getPropertyValue('border'),
    '1px solid var(--semi-color-border)',
  );
  assert.equal(
    panels.style.getPropertyValue('background'),
    'var(--semi-color-bg-1)',
  );
});

test('超长模板和预览可换行，输入框仅允许纵向缩放，不影响其他页面组件', () => {
  const input = rules.find(
    (rule) => rule.selectorText === '.classic-telegram-source-input',
  );
  const message = rules.find(
    (rule) => rule.selectorText === '.classic-telegram-message',
  );
  assert.equal(input.style.getPropertyValue('resize'), 'vertical');
  assert.equal(input.style.getPropertyValue('box-sizing'), 'border-box');
  assert.equal(message.style.getPropertyValue('overflow-wrap'), 'anywhere');
  assert.equal(message.style.getPropertyValue('white-space'), 'pre-wrap');
  for (const rule of rules.filter((rule) => rule.selectorText)) {
    for (const selector of rule.selectorText.split(','))
      assert.ok(selector.trim().startsWith('.classic-telegram-'));
  }
});
