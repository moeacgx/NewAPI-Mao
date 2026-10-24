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
import { AxiosError } from 'axios';
import { afterEach, beforeEach, expect, test } from 'vitest';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PromptCenter from '../index';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const templates = [
  {
    id: 12,
    title: 'Review guide',
    description: 'Review code changes',
    kind: 'prompt',
    load_mode: 'progressive',
    group_ids: [9],
    group_details: [{ id: 9, name: 'Visible group' }],
    version: 5,
    enabled: true,
  },
  {
    id: 13,
    title: 'Writing kit',
    description: 'Draft documents',
    kind: 'prompt',
    load_mode: 'progressive',
    group_ids: [9],
    group_details: [{ id: 9, name: 'Visible group' }],
    version: 2,
    enabled: true,
  },
];
let calls, result, pendingDelete;

beforeEach(() => {
  calls = [];
  result = 'deleted';
  pendingDelete = undefined;
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 10 }));
  API.defaults.adapter = async (config) => {
    calls.push({ url: config.url, method: config.method });
    let data;
    if (config.url === '/api/prompt-center/categories/') data = [];
    else if (
      config.method === 'get' &&
      (config.url === '/api/prompt-center/' ||
        config.url === '/api/prompt-center/admin/')
    )
      data = templates;
    else if (
      config.method === 'delete' &&
      config.url === '/api/prompt-center/admin/12?version=5'
    ) {
      if (pendingDelete) await pendingDelete;
      if (result === 'conflict') {
        const response = {
          status: 409,
          data: {
            success: false,
            message: '提示词已被其他管理员更新，请刷新后重试',
          },
          config,
          headers: {},
        };
        throw new AxiosError(
          'Conflict',
          'ERR_BAD_REQUEST',
          config,
          undefined,
          response,
        );
      }
      if (result === 'failure')
        return {
          data: { success: false, message: 'Template is being saved' },
          status: 200,
          headers: {},
          config,
        };
      data = { id: 12, uninstalled_tokens: 3 };
    } else throw new Error(`Unexpected ${config.method} ${config.url}`);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

async function openDeleteConfirmation() {
  render(<PromptCenter />);
  await userEvent.click(
    await screen.findByRole('button', { name: 'Manage templates' }),
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Delete Review guide' }),
  );
  return screen.findByRole('dialog', { name: 'Delete skill?' });
}

const deleteCalls = () => calls.filter((call) => call.method === 'delete');

test.each([
  { viewer: '普通用户', role: 1 },
  { viewer: '管理员浏览模式', role: 10 },
])('$viewer 查看技能目录时不显示删除按钮', async ({ role }) => {
  localStorage.setItem('user', JSON.stringify({ id: 2, role }));
  render(<PromptCenter />);
  await screen.findByText('Review guide');
  expect(
    screen.queryByRole('button', { name: 'Delete Review guide' }),
  ).toBeNull();
  expect(
    screen.queryByRole('button', { name: 'Edit Review guide' }),
  ).toBeNull();
});

test('管理员确认删除后按当前版本发送 DELETE，移除卡片并提示卸载令牌数', async () => {
  const dialog = await openDeleteConfirmation();
  expect(dialog.textContent).toContain('Review guide');
  expect(
    screen.getByRole('button', { name: 'Delete Writing kit' }),
  ).toBeTruthy();
  await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }));
  await waitFor(() => expect(screen.queryByText('Review guide')).toBeNull());
  expect(deleteCalls()).toEqual([
    { url: '/api/prompt-center/admin/12?version=5', method: 'delete' },
  ]);
  expect(
    await screen.findByText(
      'Deleted Review guide and uninstalled it from 3 tokens',
    ),
  ).toBeTruthy();
  expect(screen.getByText('Writing kit')).toBeTruthy();
});

test('取消删除确认不发送请求并保留卡片', async () => {
  const dialog = await openDeleteConfirmation();
  await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
  await waitFor(() =>
    expect(screen.queryByRole('dialog', { name: 'Delete skill?' })).toBeNull(),
  );
  expect(deleteCalls()).toHaveLength(0);
  expect(screen.getByText('Review guide')).toBeTruthy();
});

test.each([
  {
    reason: '版本冲突',
    mode: 'conflict',
    message: '提示词已被其他管理员更新，请刷新后重试',
  },
  { reason: '业务失败', mode: 'failure', message: 'Template is being saved' },
])(
  '删除遇到$reason时保留卡片并在确认框显示错误，可重新操作',
  async ({ mode, message }) => {
    result = mode;
    const dialog = await openDeleteConfirmation();
    const confirm = within(dialog).getByRole('button', { name: 'Delete' });
    await userEvent.click(confirm);
    expect((await within(dialog).findByRole('alert')).textContent).toBe(
      message,
    );
    expect(screen.getByText('Review guide')).toBeTruthy();
    expect(confirm.disabled).toBe(false);
    expect(
      within(dialog).getByRole('button', { name: 'Cancel' }).disabled,
    ).toBe(false);
    expect(deleteCalls()).toHaveLength(1);
  },
);

test('删除请求进行中锁定确认和取消，不能重复提交', async () => {
  let release;
  pendingDelete = new Promise((resolve) => {
    release = resolve;
  });
  const dialog = await openDeleteConfirmation();
  const confirm = within(dialog).getByRole('button', { name: 'Delete' });
  await userEvent.click(confirm);
  expect(confirm.disabled).toBe(true);
  expect(within(dialog).getByRole('button', { name: 'Cancel' }).disabled).toBe(
    true,
  );
  await userEvent.click(confirm);
  expect(deleteCalls()).toHaveLength(1);
  await act(async () => {
    release();
  });
  await waitFor(() => expect(screen.queryByText('Review guide')).toBeNull());
  expect(deleteCalls()).toHaveLength(1);
});
