/*
Copyright (C) 2023-2026 QuantumNous

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
import { afterEach, beforeEach, describe, expect, test } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TokenPromptBadge from '../TokenPromptBadge';
import TokensMobileList from '../TokensMobileList';
import { API } from '../../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const template = {
  id: 7,
  title: 'Code guide',
  kind: 'prompt',
  enabled: true,
  group_details: [{ id: 9, name: 'Premium models' }],
};
const token = {
  id: 4,
  name: 'codexQ',
  status: 1,
  prompt_count: 1,
  enabled_prompt_count: 1,
};
const zeroed = { ...token, prompt_count: 0, enabled_prompt_count: 0 };
const t = (key) => key;
const badgeQuery = { name: /^codexQ: \d+ installed prompts/, hidden: true };

beforeEach(() => {
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 1 }));
  API.defaults.adapter = async (config) => {
    const data =
      config.url === '/api/token/4/prompts'
        ? [{ template, enabled: true, sort_order: 0 }]
        : [];
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

async function openDialog() {
  await userEvent.click(screen.getByRole('button', { name: badgeQuery.name }));
  await screen.findByRole('region', { name: 'Installed prompts' });
  return document.querySelector('.prompt-installation-modal');
}

test('首次加载且没有令牌时仍显示骨架，不渲染卡片', () => {
  render(<TokensMobileList tokens={[]} loading columns={[]} t={t} />);

  expect(screen.getByRole('status').textContent).toBe('加载中...');
  expect(screen.queryByRole('article')).toBeNull();
});

test('同一令牌刷新在途及计数 1→0 期间，已打开的技能弹窗保持同一节点', async () => {
  const view = render(<TokensMobileList tokens={[token]} columns={[]} t={t} />);
  const modal = await openDialog();
  expect(modal).toBeTruthy();

  view.rerender(
    <TokensMobileList tokens={[token]} loading columns={[]} t={t} />,
  );
  expect(screen.getByRole('article', { name: 'codexQ' })).toBeTruthy();
  expect(
    document.querySelector('.tokens-mobile-list').getAttribute('aria-busy'),
  ).toBe('true');
  expect(document.querySelector('.prompt-installation-modal')).toBe(modal);

  view.rerender(
    <TokensMobileList tokens={[zeroed]} searching columns={[]} t={t} />,
  );
  expect(document.querySelector('.prompt-installation-modal')).toBe(modal);
  expect(screen.queryByRole('button', badgeQuery)).toBeNull();

  view.rerender(<TokensMobileList tokens={[zeroed]} columns={[]} t={t} />);
  expect(document.querySelector('.prompt-installation-modal')).toBe(modal);
});

test('计数归零后按 Escape 关闭弹窗，0→1 后徽标恢复并可重新打开', async () => {
  const view = render(<TokensMobileList tokens={[token]} columns={[]} t={t} />);
  await openDialog();
  view.rerender(<TokensMobileList tokens={[zeroed]} columns={[]} t={t} />);

  // Semi 的 Esc 处理器挂在 document 上并检查 keyCode，这里按真实键盘事件的形态派发。
  fireEvent.keyDown(document, { key: 'Escape', keyCode: 27 });
  await waitFor(() =>
    expect(document.querySelector('.prompt-installation-modal')).toBeNull(),
  );
  expect(screen.queryByRole('button', badgeQuery)).toBeNull();

  view.rerender(<TokensMobileList tokens={[token]} columns={[]} t={t} />);
  expect(await openDialog()).toBeTruthy();
});

const operate = {
  key: 'operate',
  render: () => <button type='button'>行操作</button>,
};
const layouts = {
  手机卡片: (value) => (
    <TokensMobileList tokens={[value]} columns={[operate]} t={t} />
  ),
  桌面表格: (value) => (
    <table>
      <tbody>
        <tr>
          <td>
            <input type='checkbox' aria-label='选择行' />
          </td>
          <td>
            <TokenPromptBadge token={value} />
          </td>
        </tr>
      </tbody>
    </table>
  ),
};

async function pressEscape() {
  fireEvent.keyDown(document, { key: 'Escape', keyCode: 27 });
  await waitFor(() =>
    expect(document.querySelector('.prompt-installation-modal')).toBeNull(),
  );
}

describe.each(Object.entries(layouts))('弹窗关闭后的焦点：%s', (_, layout) => {
  test('无刷新直接打开后按 Escape，焦点回到同一个徽标', async () => {
    render(layout(token));
    const trigger = screen.getByRole('button', { name: badgeQuery.name });
    await openDialog();
    await pressEscape();
    await waitFor(() => expect(document.activeElement).toBe(trigger));
  });

  test('打开期间徽标经 1→0→1 重建，Escape 后焦点落到新连接的徽标', async () => {
    const view = render(layout(token));
    await openDialog();
    const original = screen.getByRole('button', badgeQuery);
    view.rerender(layout(zeroed));
    view.rerender(layout(token));
    const recreated = screen.getByRole('button', badgeQuery);
    expect(recreated).not.toBe(original);
    await pressEscape();
    await waitFor(() => expect(document.activeElement).toBe(recreated));
  });

  test('计数仍为 0 时 Escape 后焦点落到同一行的稳定控件，而不是页面根节点', async () => {
    const view = render(layout(token));
    await openDialog();
    view.rerender(layout(zeroed));
    await pressEscape();
    await waitFor(() => expect(document.activeElement).not.toBe(document.body));
    expect(document.activeElement.closest('tr, article')).not.toBeNull();
  });

  test('关闭时焦点已在其他控件上，不抢走焦点', async () => {
    render(layout(token));
    await openDialog();
    const other = document.createElement('button');
    document.body.append(other);
    other.focus();
    await pressEscape();
    expect(document.activeElement).toBe(other);
    other.remove();
  });
});
