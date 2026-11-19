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
import { cleanup, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, test } from 'vitest';
import { ToolCatalog } from '../ToolCatalog';

const tools = [
  {
    name: 'users_list',
    description: 'Inspect accounts',
    category: 'users',
    write: false,
    min_role: 10,
    inputSchema: {},
  },
  {
    name: 'channels_list',
    description: 'Inspect upstreams',
    category: 'channels',
    write: false,
    min_role: 10,
    inputSchema: {},
  },
  {
    name: 'channels_test',
    description: 'Probe upstreams',
    category: 'channels',
    write: true,
    min_role: 10,
    inputSchema: {},
  },
  {
    name: 'channels_delete',
    description: 'Delete upstreams',
    category: 'channels',
    write: true,
    min_role: 100,
    inputSchema: {},
  },
];

test('目录恢复后自动退出已消失的历史分类并显示授权工具', async () => {
  const user = userEvent.setup();
  const { rerender } = render(
    <ToolCatalog
      tools={[
        {
          name: 'users_list',
          category: 'unavailable',
          description: 'Unavailable',
        },
      ]}
    />,
  );
  await user.selectOptions(
    screen.getByLabelText('Tool category'),
    'unavailable',
  );
  rerender(<ToolCatalog tools={[tools[0]]} />);
  expect(screen.getByLabelText('Tool category').value).toBe('');
  expect(screen.getByText('users_list')).toBeTruthy();
  expect(screen.queryByText('No matching tools')).toBeNull();
});

function Selection(props) {
  const [selected, setSelected] = useState([]);
  return (
    <>
      <ToolCatalog
        tools={tools}
        selected={selected}
        onChange={setSelected}
        disabled={props.disabled}
      />
      <output aria-label='Submitted tools'>{JSON.stringify(selected)}</output>
    </>
  );
}
afterEach(cleanup);

test('键盘空格可自主勾选和取消单个写工具', async () => {
  const user = userEvent.setup();
  render(<Selection />);
  screen.getByRole('checkbox', { name: /channels_test/ }).focus();
  await user.keyboard('[Space]');
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual(['channels_test']);
  await user.keyboard('[Space]');
  expect(screen.getByLabelText('Submitted tools').textContent).toBe('[]');
});

test('初始空选且只读批选不授予写权限，重复批选保持唯一名称', async () => {
  const user = userEvent.setup();
  render(<Selection />);
  expect(screen.getByLabelText('Submitted tools').textContent).toBe('[]');
  expect(
    screen
      .getByRole('button', {
        name: 'Select all visible tools in this category (including write)',
      })
      .hasAttribute('disabled'),
  ).toBe(true);
  await user.click(
    screen.getByRole('button', { name: 'Select visible read-only tools' }),
  );
  await user.click(
    screen.getByRole('button', { name: 'Select visible read-only tools' }),
  );
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual(['users_list', 'channels_list']);
  expect(screen.getByRole('checkbox', { name: /channels_test/ }).checked).toBe(
    false,
  );
});

test('分类全选包含写入且搜索限制范围，隐藏已选项保留', async () => {
  const user = userEvent.setup();
  render(<Selection />);
  await user.click(screen.getByRole('checkbox', { name: /users_list/ }));
  await user.selectOptions(screen.getByLabelText('Tool category'), 'channels');
  await user.type(screen.getByLabelText('Search tools'), 'PROBE');
  await user.click(
    screen.getByRole('button', {
      name: 'Select all visible tools in this category (including write)',
    }),
  );
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual(['users_list', 'channels_test']);
  await user.click(
    screen.getByRole('button', { name: 'Clear visible selection' }),
  );
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual(['users_list']);
  await user.clear(screen.getByLabelText('Search tools'));
  await user.click(
    screen.getByRole('button', {
      name: 'Select all visible tools in this category (including write)',
    }),
  );
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual([
    'users_list',
    'channels_list',
    'channels_test',
    'channels_delete',
  ]);
});

test('明确批选写工具仅选可见写项，切换只读筛选不会清除隐藏写授权', async () => {
  const user = userEvent.setup();
  render(<Selection />);
  await user.type(screen.getByLabelText('Search tools'), 'probe');
  await user.click(
    screen.getByRole('button', {
      name: 'Explicitly select visible write tools',
    }),
  );
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual(['channels_test']);
  await user.clear(screen.getByLabelText('Search tools'));
  await user.selectOptions(screen.getByLabelText('Access'), 'read');
  await user.click(
    screen.getByRole('button', { name: 'Select visible read-only tools' }),
  );
  await user.click(
    screen.getByRole('button', { name: 'Clear visible selection' }),
  );
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual(['channels_test']);
  expect(
    screen
      .getAllByRole('status')
      .some((node) =>
        node.textContent?.includes('1 selected in all categories · 1 write'),
      ),
  ).toBe(true);
});

test('无匹配结果时所有批选禁用且保留选择，清空筛选恢复列表', async () => {
  const user = userEvent.setup();
  render(<Selection />);
  await user.click(screen.getByRole('checkbox', { name: /users_list/ }));
  await user.type(screen.getByLabelText('Search tools'), 'no-such-tool');
  expect(screen.getByText('No matching tools')).toBeTruthy();
  expect(screen.queryAllByRole('checkbox')).toHaveLength(0);
  for (const button of screen.getAllByRole('button'))
    expect(button.hasAttribute('disabled')).toBe(true);
  await user.clear(screen.getByLabelText('Search tools'));
  expect(
    JSON.parse(screen.getByLabelText('Submitted tools').textContent),
  ).toEqual(['users_list']);
});

test('提交期间禁用筛选勾选与批选', async () => {
  const user = userEvent.setup();
  render(<Selection disabled />);
  for (const control of [
    ...screen.getAllByRole('button'),
    ...screen.getAllByRole('combobox'),
    ...screen.getAllByRole('checkbox'),
    screen.getByRole('textbox'),
  ]) {
    expect(control.hasAttribute('disabled')).toBe(true);
  }
  await user.click(
    screen.getByRole('button', {
      name: 'Explicitly select visible write tools',
    }),
  );
  expect(screen.getByLabelText('Submitted tools').textContent).toBe('[]');
});

test('参考页按分类显示读写统计且筛选后统计更新', async () => {
  const user = userEvent.setup();
  render(<ToolCatalog tools={tools} />);
  const channels = screen.getByRole('region', { name: 'channels' });
  expect(
    within(channels).getByText('3 tools · 1 read-only · 2 write'),
  ).toBeTruthy();
  expect(screen.getByRole('status').textContent).toBe(
    '4 tools · 2 read-only · 2 write',
  );
  await user.selectOptions(screen.getByLabelText('Access'), 'write');
  expect(screen.queryByRole('region', { name: 'users' })).toBeNull();
  expect(screen.getByRole('status').textContent).toBe(
    '2 tools · 0 read-only · 2 write',
  );
});

test('空目录保留可访问筛选和明确空状态', () => {
  render(<ToolCatalog tools={[]} selected={[]} onChange={() => {}} />);
  expect(screen.getByText('No matching tools')).toBeTruthy();
  expect(screen.getByLabelText('Tool category')).toBeTruthy();
  for (const button of screen.getAllByRole('button'))
    expect(button.hasAttribute('disabled')).toBe(true);
});
