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
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TieredPricingEditor from '../TieredPricingEditor';
import { combineBillingExpr } from '../requestRuleExpr';
function Editor({ base, rule, onChange }) {
  const [expr, setExpr] = useState(base);
  const [rules, setRules] = useState(rule);
  return (
    <>
      <output data-testid='saved'>{combineBillingExpr(expr, rules)}</output>
      <TieredPricingEditor
        model={{ name: 'model-a', billingExpr: expr }}
        requestRuleExpr={rules}
        onExprChange={(value) => {
          setExpr(value);
          onChange(value);
        }}
        onRequestRuleExprChange={setRules}
        t={(key) => key}
      />
    </>
  );
}
test('打开已有价格不发出清空更新，未知规则模式往返保留一次', async () => {
  const onChange = vi.fn();
  const base = 'tier("base", p * 2 + c * 4)';
  const rule = '(hour("UTC") >= 9 || hour("UTC") < 18 ? 0.5 : 1)';
  render(<Editor base={base} rule={rule} onChange={onChange} />);
  const user = userEvent.setup();
  expect(onChange).not.toHaveBeenCalled();
  await user.click(screen.getByRole('radio', { name: '表达式编辑' }));
  await user.click(screen.getByRole('radio', { name: '可视化编辑' }));
  expect(screen.getByRole('radio', { name: '可视化编辑' }).checked).toBe(true);
  expect(screen.getByTestId('saved').textContent).toBe(
    combineBillingExpr(base, rule),
  );
  expect(onChange).not.toHaveBeenCalled();
});
test('未知基础表达式不能被模式切换替换为默认价格', async () => {
  const base = 'max(p * 2, 100)';
  render(<Editor base={base} rule='' onChange={() => {}} />);
  await userEvent
    .setup()
    .click(screen.getByRole('radio', { name: '可视化编辑' }));
  expect(screen.getByTestId('saved').textContent).toBe(base);
});
