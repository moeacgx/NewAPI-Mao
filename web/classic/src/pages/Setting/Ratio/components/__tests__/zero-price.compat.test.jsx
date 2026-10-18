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

import React, { useState } from 'react';
import { expect, test, vi } from 'vitest';
import {
  act,
  render,
  renderHook,
  screen,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TieredPricingEditor from '../TieredPricingEditor';
import { useModelPricingEditorState } from '../../hooks/useModelPricingEditorState';
import { API } from '../../../../../helpers/api';

const paid = 'tier("base", p * 0.5 + c * 2)';
const freeOutput = 'tier("base", p * 0.5 + c * 0)';
const translate = (key) => key;

function Editor({ initial = paid, onChange = () => {} }) {
  const [expr, setExpr] = useState(initial);
  return (
    <>
      <output aria-label='当前计费公式'>{expr}</output>
      <TieredPricingEditor
        model={{ name: 'typesafe-jev', billingExpr: expr }}
        requestRuleExpr=''
        onExprChange={(value) => {
          setExpr(value);
          onChange(value);
        }}
        onRequestRuleExprChange={() => {}}
        t={translate}
      />
    </>
  );
}

function priceField(label) {
  return within(screen.getByText(label).parentElement).getByRole('textbox');
}

test('已保存输出零价回填为真实的0而非空白占位符', () => {
  render(<Editor initial={freeOutput} />);
  expect(priceField('输出价格').value).toBe('0');
  expect(priceField('输入价格').value).toBe('0.5');
});

test('输出从收费改为0后模式往返及重新打开保留输入价和零输出', async () => {
  const changed = vi.fn();
  const user = userEvent.setup();
  const first = render(<Editor onChange={changed} />);
  await user.clear(priceField('输出价格'));
  await user.type(priceField('输出价格'), '0');
  await user.tab();
  expect(screen.getByLabelText('当前计费公式').textContent).toBe(freeOutput);
  await user.click(screen.getByRole('radio', { name: '表达式编辑' }));
  expect(screen.getByPlaceholderText('输入计费表达式...').value).toBe(
    freeOutput,
  );
  await user.click(screen.getByRole('radio', { name: '可视化编辑' }));
  expect(priceField('输出价格').value).toBe('0');
  const saved = changed.mock.lastCall[0];
  first.unmount();
  render(<Editor initial={saved} />);
  expect(priceField('输出价格').value).toBe('0');
  expect(priceField('输入价格').value).toBe('0.5');
});

test('同名模型从保存结果回填零价时同步展示且不主动改写配置', () => {
  const changed = vi.fn();
  const props = {
    model: { name: 'typesafe-jev', billingExpr: paid },
    requestRuleExpr: '',
    onExprChange: changed,
    onRequestRuleExprChange: vi.fn(),
    t: translate,
  };
  const view = render(<TieredPricingEditor {...props} />);
  expect(priceField('输出价格').value).toBe('2');
  view.rerender(
    <TieredPricingEditor
      {...props}
      model={{ ...props.model, billingExpr: freeOutput }}
    />,
  );
  expect(priceField('输出价格').value).toBe('0');
  expect(priceField('输入价格').value).toBe('0.5');
  expect(changed).not.toHaveBeenCalled();
});

test('价格草稿末尾小数点和原始公式编辑模式不会被父组件回传重置', async () => {
  const user = userEvent.setup();
  render(<Editor />);
  await user.clear(priceField('输出价格'));
  await user.type(priceField('输出价格'), '7.');
  expect(priceField('输出价格').value).toBe('7.');
  await user.click(screen.getByRole('radio', { name: '表达式编辑' }));
  const raw = screen.getByPlaceholderText('输入计费表达式...');
  await user.clear(raw);
  await user.type(raw, freeOutput);
  expect(raw.value).toBe(freeOutput);
  expect(screen.getByRole('radio', { name: '表达式编辑' }).checked).toBe(true);
});

test.each(['per-token', 'tiered_expr'])(
  '%s输出零价保存并从接口重新加载后保留零和原输入价',
  async (mode) => {
    const options = {
      ModelRatio: '{"typesafe-jev":0.25}',
      CompletionRatio: '{"typesafe-jev":4}',
      'billing_setting.billing_mode': '{}',
      'billing_setting.billing_expr': '{}',
    };
    if (mode === 'tiered_expr') {
      options['billing_setting.billing_mode'] =
        '{"typesafe-jev":"tiered_expr"}';
      options['billing_setting.billing_expr'] = JSON.stringify({
        'typesafe-jev': paid,
      });
    }
    const saved = {};
    vi.spyOn(API, 'put').mockImplementation(async (url, body) => {
      expect(url).toBe('/api/option/');
      saved[body.key] = body.value;
      return { data: { success: true } };
    });
    const refresh = vi.fn();
    const view = renderHook(
      ({ value }) =>
        useModelPricingEditorState({ options: value, refresh, t: translate }),
      { initialProps: { value: options } },
    );
    act(() => {
      if (mode === 'tiered_expr')
        view.result.current.handleBillingExprChange(freeOutput);
      else view.result.current.handleNumericFieldChange('completionPrice', '0');
    });
    await act(async () => view.result.current.handleSubmit());
    expect(refresh).toHaveBeenCalledOnce();
    if (mode === 'tiered_expr') {
      expect(
        JSON.parse(saved['billing_setting.billing_expr'])['typesafe-jev'],
      ).toBe(freeOutput);
    } else {
      expect(JSON.parse(saved.CompletionRatio)['typesafe-jev']).toBe(0);
      expect(JSON.parse(saved.ModelRatio)['typesafe-jev']).toBe(0.25);
    }
    view.rerender({ value: { ...options, ...saved } });
    const loaded = view.result.current.selectedModel;
    if (mode === 'tiered_expr') expect(loaded.billingExpr).toBe(freeOutput);
    else {
      expect(loaded.completionPrice).toBe('0');
      expect(loaded.inputPrice).toBe('0.5');
    }
  },
);

test('原始模式同名模型回填零价公式及规则时同步且不丢条件', async () => {
  const changed = vi.fn();
  const props = {
    model: { name: 'typesafe-jev', billingExpr: 'max(p * 2, 100)' },
    requestRuleExpr: '',
    onExprChange: changed,
    onRequestRuleExprChange: vi.fn(),
    t: translate,
  };
  const view = render(<TieredPricingEditor {...props} />);
  const rule = '(header("x-plan") == "free" ? 0 : 1)';
  view.rerender(
    <TieredPricingEditor
      {...props}
      model={{ ...props.model, billingExpr: freeOutput }}
      requestRuleExpr={rule}
    />,
  );
  expect(priceField('输出价格').value).toBe('0');
  await userEvent
    .setup()
    .click(screen.getByRole('radio', { name: '表达式编辑' }));
  expect(screen.getByPlaceholderText('输入计费表达式...').value).toBe(
    '(' + freeOutput + ') * ' + rule,
  );
  expect(changed).not.toHaveBeenCalled();
});
