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
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { API } from '../../../helpers/api';
import PromptGroupSelector from '../PromptGroupSelector';
import PromptCatalog from '../PromptCatalog';
import PromptTemplateEditor from '../PromptTemplateEditor';

const adapter = API.defaults.adapter;
const groups = [
  { id: 9, name: 'GPT models', code: 'hidden-gpt', status: 1 },
  { id: 12, name: 'Claude models', code: 'hidden-claude', status: 2 },
];
const tags = [
  { id: 1, name: 'GPT', group_ids: [9], icons: [] },
  { id: 2, name: 'Claude', group_ids: [12], icons: [] },
];

beforeEach(() => {
  API.defaults.adapter = async (config) => ({
    data: { success: true, data: tags },
    status: 200,
    headers: {},
    config,
  });
});
afterEach(() => {
  API.defaults.adapter = adapter;
});

function Selector(props) {
  const [selected, setSelected] = useState(props.selected || []);
  return (
    <PromptGroupSelector
      groups={groups}
      selected={selected}
      onChange={setSelected}
    />
  );
}

test('跨标签多选后保留稳定分组 ID，搜索和移除不会丢失其他标签的选择', async () => {
  render(<Selector />);
  await userEvent.click(await screen.findByRole('radio', { name: /GPT/ }));
  let picker = screen.getByRole('dialog', { name: 'Allowed groups' });
  await userEvent.click(
    within(picker).getByRole('checkbox', { name: 'GPT models' }),
  );
  await userEvent.click(within(picker).getByRole('button', { name: 'Close' }));
  await userEvent.click(screen.getByRole('radio', { name: /Claude/ }));
  picker = screen.getByRole('dialog', { name: 'Allowed groups' });
  await userEvent.type(within(picker).getByRole('searchbox'), 'Claude');
  expect(
    within(picker).queryByRole('checkbox', { name: 'GPT models' }),
  ).toBeNull();
  await userEvent.click(
    within(picker).getByRole('checkbox', { name: 'Claude models' }),
  );
  await userEvent.click(within(picker).getByRole('button', { name: 'Close' }));
  expect(
    screen.getByRole('button', { name: 'Remove group GPT models' }),
  ).toBeTruthy();
  await userEvent.click(
    screen.getByRole('button', { name: 'Remove group Claude models' }),
  );
  expect(
    screen.queryByRole('button', { name: 'Remove group Claude models' }),
  ).toBeNull();
  await userEvent.click(screen.getByRole('radio', { name: /GPT/ }));
  expect(screen.getByRole('checkbox', { name: 'GPT models' }).checked).toBe(
    true,
  );
  expect(screen.queryByText('hidden-gpt')).toBeNull();
});

test('标签请求失败时回退全部分组，仍可移除失效组并选择停用实体组', async () => {
  API.defaults.adapter = async () => {
    throw new Error('Tag request failed');
  };
  render(<Selector selected={[99]} />);
  await screen.findByText('Failed to load group tags');
  await userEvent.click(
    screen.getByRole('checkbox', { name: /Unavailable group.*99/ }),
  );
  expect(
    screen.queryByRole('checkbox', { name: /Unavailable group.*99/ }),
  ).toBeNull();
  await userEvent.click(
    screen.getByRole('checkbox', { name: 'Claude models' }),
  );
  expect(screen.getByRole('checkbox', { name: 'Claude models' }).checked).toBe(
    true,
  );
});

test('编辑器跨标签选择多个实际分组后提交全部稳定 ID', async () => {
  const saves = [];
  API.defaults.adapter = async (config) => {
    let data;
    if (config.url === '/api/group/details') data = groups;
    else if (config.url === '/api/group/tags') data = tags;
    else if (config.method === 'put') {
      saves.push(JSON.parse(config.data));
      data = { id: 7 };
    } else throw new Error(`Unexpected ${config.method} ${config.url}`);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
  render(
    <PromptTemplateEditor
      template={{
        id: 7,
        version: 3,
        title: 'Review',
        content: 'Review code.',
        group_ids: [9],
      }}
      categories={[]}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await userEvent.click(await screen.findByRole('radio', { name: /Claude/ }));
  const picker = screen.getByRole('dialog', { name: 'Allowed groups' });
  await userEvent.click(
    within(picker).getByRole('checkbox', { name: 'Claude models' }),
  );
  await userEvent.click(within(picker).getByRole('button', { name: 'Close' }));
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() => expect(saves).toHaveLength(1));
  expect(saves[0]).toMatchObject({
    all_groups: false,
    group_ids: [9, 12],
    version: 3,
  });
});

test('技能卡片默认隐藏分组，标题键盘操作和主体点击展开，安装编辑删除操作保持折叠', async () => {
  const onInstall = vi.fn();
  const onEdit = vi.fn();
  render(
    <PromptCatalog
      templates={[
        {
          id: 7,
          title: 'Review',
          enabled: true,
          description: 'Review code',
          group_details: groups,
        },
      ]}
      bindings={[]}
      canInstall
      onInstall={onInstall}
      onEdit={onEdit}
      onDeleted={() => {}}
    />,
  );
  expect(screen.queryByText('GPT models')).toBeNull();
  const title = screen.getByRole('button', { name: 'Review' });
  title.focus();
  await userEvent.keyboard('{Enter}');
  expect(title.getAttribute('aria-expanded')).toBe('true');
  expect(screen.getByText('GPT models')).toBeTruthy();
  await userEvent.click(screen.getByText('Review code'));
  expect(screen.queryByText('GPT models')).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Install Review' }));
  await userEvent.click(screen.getByRole('button', { name: 'Edit Review' }));
  expect(onInstall).toHaveBeenCalledOnce();
  expect(onEdit).toHaveBeenCalledOnce();
  await userEvent.click(screen.getByRole('button', { name: 'Delete Review' }));
  await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
  expect(title.getAttribute('aria-expanded')).toBe('false');
});
