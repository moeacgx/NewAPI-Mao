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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, test } from 'vitest';
import SettingGlobalModel from '../../pages/Setting/Model/SettingGlobalModel';
import settingGlobalModelSource from '../../pages/Setting/Model/SettingGlobalModel.jsx?raw';

const formElement = (groups, saved) => (
  <SettingGlobalModel
    options={{ 'global.canvas_default_group': saved }}
    groups={groups}
    autoGroup={{ user_selectable: true }}
    refresh={() => {}}
  />
);

const defaultCombobox = () => screen.getAllByRole('combobox')[0];

const comboboxText = () => defaultCombobox().textContent;

test('真实画布表单：已存停用分组显示当前名称而不是 code', async () => {
  const view = render(
    formElement(
      [{ id: 8, code: 'disabled-code', name: '停用当前名', status: 0 }],
      'disabled-code',
    ),
  );
  try {
    await waitFor(() => expect(comboboxText()).toContain('停用当前名'));
    expect(comboboxText()).not.toContain('disabled-code');
  } finally {
    view.unmount();
  }
});

test('真实画布表单：停用分组改名后显示新名称', async () => {
  const groups = (name) => [{ id: 8, code: 'disabled-code', name, status: 0 }];
  const view = render(formElement(groups('旧名称'), 'disabled-code'));
  try {
    await waitFor(() => expect(comboboxText()).toContain('旧名称'));
    view.rerender(formElement(groups('新名称'), 'disabled-code'));
    await waitFor(() => expect(comboboxText()).toContain('新名称'));
    expect(comboboxText()).not.toContain('旧名称');
  } finally {
    view.unmount();
  }
});

test('真实画布表单：停用项不可新选，切换有效分组后不能再选回', async () => {
  const view = render(
    formElement(
      [
        { id: 8, code: 'disabled-code', name: '停用当前名', status: 0 },
        { id: 9, code: 'stable-code', name: '当前名称', status: 1 },
      ],
      'disabled-code',
    ),
  );
  try {
    await waitFor(() => expect(comboboxText()).toContain('停用当前名'));
    fireEvent.click(defaultCombobox());
    const disabledOption = await screen.findByRole('option', {
      name: /停用当前名/,
    });
    expect(disabledOption.getAttribute('aria-disabled')).toBe('true');
    // jsdom 中 Semi Form.Select 的点选不触发 onChange（最小用例已验证），
    // 因此以重渲染保存值为有效分组来验证切换后的结果。
    view.rerender(
      formElement(
        [
          { id: 8, code: 'disabled-code', name: '停用当前名', status: 0 },
          { id: 9, code: 'stable-code', name: '当前名称', status: 1 },
        ],
        'stable-code',
      ),
    );
    await waitFor(() => expect(comboboxText()).toContain('当前名称'));
    expect(comboboxText()).not.toContain('停用当前名');
    expect(screen.queryByRole('option', { name: /停用当前名/ })).toBeNull();
  } finally {
    view.unmount();
  }
});

test.each([
  [
    '未知值',
    [{ id: 9, code: 'stable-code', name: '当前名称', status: 1 }],
    'gone-code',
    'gone-code',
  ],
  ['已删除分组', [], 'deleted-code', 'deleted-code'],
  [
    '停用但无名称',
    [{ id: 8, code: 'no-name', name: '', status: 0 }],
    'no-name',
    'no-name',
  ],
  ['原型键 code', [], 'constructor', 'constructor'],
  [
    '继承而来的名称',
    [
      Object.assign(Object.create({ name: '继承名' }), {
        id: 8,
        code: 'proto-name',
        status: 0,
      }),
    ],
    'proto-name',
    'proto-name',
  ],
  ['auto', [], 'auto', '自动选择'],
  ['空值', [], '', '跟随系统默认'],
])('真实画布表单：%s回退展示', async (_label, groups, saved, expected) => {
  const view = render(formElement(groups, saved));
  try {
    await waitFor(() => expect(comboboxText()).toContain(expected));
    expect(comboboxText()).not.toContain('继承名');
  } finally {
    view.unmount();
  }
});

// Semi Form 的 withField 在未设置 allowEmptyString/allowEmpty 时会把 ''
// 转成 undefined，导致受控 Select 选回“跟随系统默认”后标签空白。
// jsdom 无法驱动 Semi 下拉产生 onChange，自然交互已在真实 Chrome 中验证；
// 这里锁定该字段必须保留空字符串的属性契约。
test('画布默认分组字段保留空字符串，选回跟随系统默认不会变空白', () => {
  const source = settingGlobalModelSource;
  const openingTag = source.match(
    /<Form\.Select\s[^>]*?label=\{t\('无限画布默认分组'\)\}[\s\S]*?optionList=/,
  );
  expect(openingTag).not.toBeNull();
  expect(openingTag[0]).toMatch(/\ballowEmptyString\b/);
  expect(source).toContain("{ value: '', label: t('跟随系统默认') }");
});
