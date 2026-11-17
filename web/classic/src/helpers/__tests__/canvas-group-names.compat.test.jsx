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
import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { Select } from '@douyinfe/semi-ui';
import { expect, test, vi } from 'vitest';
import {
  buildCanvasDefaultGroupOptions,
  getGroupDisplayName,
} from '../groupDetails';

test('仅使用自有非空字符串名称，原型键与未知标识安全回退', () => {
  const names = JSON.parse(
    '{"__proto__":"原型名称","legacy-alias":"改名后的名称","empty":"","object":{}}',
  );
  expect(getGroupDisplayName('__proto__', names)).toBe('原型名称');
  expect(getGroupDisplayName('legacy-alias', names)).toBe('改名后的名称');
  for (const code of [
    'constructor',
    'toString',
    'empty',
    'object',
    'deleted',
  ]) {
    expect(getGroupDisplayName(code, names)).toBe(code);
  }
});

test('画布按当前名称选择并提交原 code，改名与缺名不改变值', async () => {
  const onChange = vi.fn();
  const groups = [
    { id: 7, code: 'stable-code', name: '当前名称', status: 1 },
    { id: 8, code: 'unknown', name: '', status: 1 },
    { id: 9, code: 'disabled', name: '已停用', status: 0 },
  ];
  const view = render(
    <Select
      optionList={buildCanvasDefaultGroupOptions(groups)}
      onChange={onChange}
    />,
  );
  fireEvent.click(screen.getByRole('combobox'));
  fireEvent.click(await screen.findByRole('option', { name: /当前名称$/ }));
  expect(onChange.mock.calls[0][0]).toBe('stable-code');
  expect(screen.queryByText('已停用')).toBeNull();
  view.rerender(
    <Select
      value='stable-code'
      optionList={buildCanvasDefaultGroupOptions([
        { ...groups[0], name: '改名后的名称' },
        groups[1],
      ])}
      onChange={onChange}
    />,
  );
  expect(screen.getAllByText('改名后的名称').length).toBeGreaterThan(0);
  expect(buildCanvasDefaultGroupOptions(groups)).toEqual([
    { value: 'stable-code', label: '当前名称' },
    { value: 'unknown', label: 'unknown' },
  ]);
});
