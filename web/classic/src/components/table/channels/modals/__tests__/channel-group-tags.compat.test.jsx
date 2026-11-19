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
import { afterEach, expect, test } from 'vitest';
import userEvent from '@testing-library/user-event';
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import EditChannelModal from '../EditChannelModal';
import { API } from '../../../../../helpers/api';

// 显示名称刻意与 code 不同，验证展示用名称、提交用 code/ID。
const GROUPS = [
  { id: 11, code: 'g_alpha', name: '阿尔法组', ratio: 1, status: 1 },
  { id: 12, code: 'g_beta', name: '贝塔组', ratio: 1, status: 1 },
  { id: 13, code: 'g_gamma', name: '伽马组', ratio: 1, status: 1 },
];
const TAGS = [
  { id: 1, name: '国产标签', icons: [], group_ids: [12] },
  { id: 2, name: '海外标签', icons: [], group_ids: [13] },
  { id: 3, name: '空标签', icons: [], group_ids: [] },
];

const adapter = API.defaults.adapter;
afterEach(() => {
  API.defaults.adapter = adapter;
  cleanup();
});

function mockApi(tags, captured) {
  API.defaults.adapter = async (config) => {
    let data = [];
    if (config.method === 'put') captured.payload = JSON.parse(config.data);
    if (config.url === '/api/group/details') data = GROUPS;
    if (config.url === '/api/group/tags') data = tags;
    if (config.url === '/api/channel/42')
      data = {
        id: 42,
        name: 'group channel',
        type: 1,
        models: 'gpt-4o',
        group: 'g_alpha',
        group_ids: [11],
        group_mode: 'explicit',
        key: '',
        model_mapping: '',
        status_code_mapping: '',
        setting: '{}',
        other: '{}',
        channel_info: { is_multi_key: false },
      };
    return {
      data: { success: true, data },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
}

function renderModal() {
  render(
    <EditChannelModal
      visible
      editingChannel={{ id: 42 }}
      handleClose={() => {}}
      refresh={() => {}}
    />,
  );
}

test('标签卡片跨标签累加多选，保留预填并按 code/ID 提交', async () => {
  const captured = {};
  mockApi(TAGS, captured);
  renderModal();
  const user = userEvent.setup();
  const selected = () => document.querySelector('.channel-group-selected-list');
  await waitFor(() => expect(selected()?.textContent).toContain('阿尔法组'));
  expect(screen.queryByRole('radio', { name: /空标签/ })).toBeNull();

  await user.click(screen.getByRole('radio', { name: /国产标签/ }));
  let dialog = await screen.findByRole('dialog', { name: '选择分组' });
  expect(within(dialog).queryByText('伽马组')).toBeNull();
  await user.click(within(dialog).getByRole('checkbox', { name: '贝塔组' }));

  await user.click(screen.getByRole('radio', { name: /海外标签/ }));
  dialog = await screen.findByRole('dialog', { name: '选择分组' });
  await user.type(
    within(dialog).getByRole('textbox', { name: '搜索分组...' }),
    '伽马',
  );
  await user.click(within(dialog).getByRole('checkbox', { name: '伽马组' }));
  // Esc 关闭后立刻点击其他标签必须能重新打开浮窗。
  await user.keyboard('{Escape}');
  expect(screen.queryByRole('dialog', { name: '选择分组' })).toBeNull();
  await user.click(screen.getByRole('radio', { name: /国产标签/ }));
  expect(await screen.findByRole('dialog', { name: '选择分组' })).toBeTruthy();
  expect(selected().textContent).toContain('阿尔法组');
  expect(selected().textContent).toContain('贝塔组');
  expect(selected().textContent).toContain('伽马组');

  await user.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() => expect(captured.payload).toBeTruthy());
  expect(captured.payload.group).toBe('g_alpha,g_beta,g_gamma');
  expect(captured.payload.group_ids).toEqual([11, 12, 13]);
});

test('清空后提交为空分组继承语义', async () => {
  const captured = {};
  mockApi(TAGS, captured);
  renderModal();
  const user = userEvent.setup();
  await waitFor(() =>
    expect(
      document.querySelector('.channel-group-selected-list')?.textContent,
    ).toContain('阿尔法组'),
  );
  await user.click(screen.getByRole('button', { name: '清空' }));
  expect(document.querySelector('.channel-group-selected-list')).toBeNull();
  await user.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() => expect(captured.payload).toBeTruthy());
  expect(captured.payload.group).toBe('');
  expect(captured.payload.group_ids).toEqual([]);
});

test('没有标签时保留原多选下拉并原样保存历史分组', async () => {
  const captured = {};
  mockApi([], captured);
  renderModal();
  const user = userEvent.setup();
  await waitFor(() =>
    expect(
      document.querySelector(
        '.semi-form-field[x-field-id="groups"] .semi-select',
      )?.textContent,
    ).toContain('阿尔法组'),
  );
  expect(document.querySelector('.channel-group-select')).toBeNull();
  expect(screen.queryByRole('radio', { name: /标签/ })).toBeNull();
  await user.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() => expect(captured.payload).toBeTruthy());
  expect(captured.payload.group).toBe('g_alpha');
  expect(captured.payload.group_ids).toEqual([11]);
});
