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
import { Toast } from '@douyinfe/semi-ui';
import { AxiosError } from 'axios';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PromptCenter from '../index';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
let calls, result, categories, templates;

beforeEach(() => {
  calls = [];
  result = 'deleted';
  categories = [
    { id: 4, name: 'Writing', version: 3 },
    { id: 5, name: 'Research', version: 1 },
  ];
  templates = [
    {
      id: 7,
      title: 'Outline skill',
      category_id: 4,
      enabled: true,
      version: 2,
    },
    {
      id: 8,
      title: 'Survey skill',
      category_id: 5,
      enabled: true,
      version: 1,
    },
  ];
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 10 }));
  API.defaults.adapter = async (config) => {
    calls.push({ url: config.url, method: config.method });
    let data;
    if (config.url === '/api/prompt-center/categories/') data = categories;
    else if (
      config.method === 'get' &&
      (config.url === '/api/prompt-center/' ||
        config.url === '/api/prompt-center/admin/')
    )
      data = templates;
    else if (
      config.method === 'delete' &&
      config.url === '/api/prompt-center/admin/categories/4?version=3'
    ) {
      if (result === 'conflict') {
        const response = {
          status: 409,
          data: {
            success: false,
            message: '分类已被其他管理员更新，请刷新后重试',
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
      if (result === 'missing')
        return {
          data: { success: false, message: '分类不存在或已删除' },
          status: 200,
          headers: {},
          config,
        };
      categories = categories.filter((item) => item.id !== 4);
      templates = templates.map((item) =>
        item.category_id === 4 ? { ...item, category_id: 0 } : item,
      );
      data = { id: 4, moved_templates: 1 };
    } else throw new Error(`Unexpected ${config.method} ${config.url}`);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
  vi.restoreAllMocks();
});

const deleteCalls = () => calls.filter((call) => call.method === 'delete');

async function openCategoryDeleteConfirmation() {
  await userEvent.click(
    screen.getByRole('button', { name: 'Manage categories' }),
  );
  await userEvent.click(
    await screen.findByRole('button', { name: 'Delete category Writing' }),
  );
  return screen.findByRole('dialog', { name: 'Delete category?' });
}

test('管理员确认删除分类后按当前版本发送 DELETE，提示移动数量并刷新分类与技能', async () => {
  const successToast = vi.spyOn(Toast, 'success');
  render(<PromptCenter />);
  await screen.findByText('Outline skill');
  const dialog = await openCategoryDeleteConfirmation();
  expect(dialog.textContent).toContain('Writing');
  await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }));
  await waitFor(() => expect(successToast).toHaveBeenCalledTimes(1));
  const successMessage = successToast.mock.calls[0][0];
  expect(successMessage).toContain('Writing');
  expect(successMessage).toContain('1');
  expect(deleteCalls()).toEqual([
    {
      url: '/api/prompt-center/admin/categories/4?version=3',
      method: 'delete',
    },
  ]);
  await waitFor(() =>
    expect(screen.queryByLabelText('Rename Writing')).toBeNull(),
  );
  expect(screen.getByRole('textbox', { name: 'Rename Research' })).toBeTruthy();
  const filter = screen.getByLabelText('Filter category');
  await waitFor(() =>
    expect(
      within(filter).queryByRole('option', { name: 'Writing' }),
    ).toBeNull(),
  );
  await userEvent.selectOptions(filter, '0');
  expect(await screen.findByText('Outline skill')).toBeTruthy();
  expect(screen.queryByText('Survey skill')).toBeNull();
});

test('当前筛选的分类被删除后筛选回退到全部分类并显示所有技能', async () => {
  render(<PromptCenter />);
  await screen.findByText('Outline skill');
  const filter = screen.getByLabelText('Filter category');
  await userEvent.selectOptions(filter, '4');
  expect(screen.queryByText('Survey skill')).toBeNull();
  const dialog = await openCategoryDeleteConfirmation();
  await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }));
  await waitFor(() => expect(filter.value).toBe('all'));
  expect(await screen.findByText('Survey skill')).toBeTruthy();
  expect(screen.getByText('Outline skill')).toBeTruthy();
});

test('取消删除分类确认不发送请求，返回分类列表并保留分类', async () => {
  render(<PromptCenter />);
  await screen.findByText('Outline skill');
  const dialog = await openCategoryDeleteConfirmation();
  await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
  await screen.findByRole('dialog', { name: 'Manage categories' });
  expect(deleteCalls()).toHaveLength(0);
  expect(screen.getByRole('textbox', { name: 'Rename Writing' })).toBeTruthy();
});

test('打开删除确认时焦点进入确认内容，取消后焦点回到该分类的删除按钮', async () => {
  render(<PromptCenter />);
  await screen.findByText('Outline skill');
  const dialog = await openCategoryDeleteConfirmation();
  await waitFor(() =>
    expect(dialog.contains(document.activeElement)).toBe(true),
  );
  await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
  await waitFor(() =>
    expect(document.activeElement).toBe(
      screen.getByRole('button', { name: 'Delete category Writing' }),
    ),
  );
});

test.each([
  {
    reason: '版本冲突',
    mode: 'conflict',
    message: '分类已被其他管理员更新，请刷新后重试',
  },
  { reason: '分类已不存在', mode: 'missing', message: '分类不存在或已删除' },
])(
  '删除分类遇到$reason时在确认框显示服务端消息并保留分类',
  async ({ mode, message }) => {
    result = mode;
    render(<PromptCenter />);
    await screen.findByText('Outline skill');
    const dialog = await openCategoryDeleteConfirmation();
    const confirm = within(dialog).getByRole('button', { name: 'Delete' });
    await userEvent.click(confirm);
    expect((await within(dialog).findByRole('alert')).textContent).toBe(
      message,
    );
    expect(confirm.disabled).toBe(false);
    expect(deleteCalls()).toHaveLength(1);
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Cancel' }),
    );
    expect(
      await screen.findByRole('textbox', { name: 'Rename Writing' }),
    ).toBeTruthy();
    expect(
      within(screen.getByLabelText('Filter category')).getByRole('option', {
        name: 'Writing',
      }),
    ).toBeTruthy();
  },
);
