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
import { afterEach, beforeEach, expect, test } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PromptTemplateEditor from '../PromptTemplateEditor';
import PromptCatalog from '../PromptCatalog';
import TokenPromptManager from '../TokenPromptManager';
import PromptCenter from '../index';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const ALL_GROUPS = 'All groups (no restriction)';
const restricted = {
  id: 7,
  title: 'Restricted skill',
  description: 'Only for one group',
  content: 'Follow the group rules.',
  enabled: true,
  version: 3,
  all_groups: false,
  group_ids: [9],
  group_details: [{ id: 9, name: 'Visible group', code: 'hidden-code' }],
};
const unrestricted = {
  id: 8,
  title: 'Open skill',
  description: 'For every group',
  content: 'Help everyone.',
  enabled: true,
  version: 4,
  all_groups: true,
  group_ids: [],
  group_details: [],
};
let calls, bindings, catalogTemplates;

beforeEach(() => {
  calls = [];
  bindings = [];
  catalogTemplates = [unrestricted, restricted];
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 10 }));
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ url: config.url, method: config.method, body });
    let data;
    if (config.url === '/api/group/details')
      data = [{ id: 9, name: 'Visible group', code: 'hidden-code' }];
    else if (config.url === '/api/prompt-center/categories/') data = [];
    else if (config.url === '/api/prompt-center/') data = catalogTemplates;
    else if (config.url === '/api/token/4/prompts') data = bindings;
    else if (config.method === 'post' || config.method === 'put')
      data = { ...body, id: 12 };
    else throw new Error(`Unexpected ${config.method} ${config.url}`);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

const writes = () =>
  calls.filter((call) => call.method === 'post' || call.method === 'put');

async function fillNewSkill() {
  await userEvent.type(screen.getByLabelText('Prompt title'), 'Shared skill');
  await userEvent.type(
    screen.getByLabelText('Prompt description'),
    'Use for shared tasks.',
  );
  await userEvent.type(screen.getByLabelText('Prompt text'), 'Be helpful.');
}

test('新建技能勾选不限分组后隐藏分组列表，发布提交 all_groups=true 与空白名单', async () => {
  render(
    <PromptTemplateEditor
      template={{}}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  await fillNewSkill();
  const allGroups = screen.getByRole('checkbox', { name: ALL_GROUPS });
  expect(allGroups.checked).toBe(false);
  await userEvent.click(allGroups);
  expect(allGroups.checked).toBe(true);
  expect(screen.queryByRole('checkbox', { name: 'Visible group' })).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() => expect(writes()).toHaveLength(1));
  expect(writes()[0]).toMatchObject({
    url: '/api/prompt-center/admin/',
    method: 'post',
    body: { enabled: true, all_groups: true, group_ids: [] },
  });
});

test('新建技能既未勾选不限分组也未选分组时发布被阻止且不发送写入', async () => {
  render(
    <PromptTemplateEditor
      template={{}}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  await fillNewSkill();
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  expect((await screen.findByRole('alert')).textContent).toBe(
    'Enter a title and at least one allowed group before publishing',
  );
  expect(writes()).toHaveLength(0);
});

test('未发布草稿可不限分组也不选分组直接保存', async () => {
  render(
    <PromptTemplateEditor
      template={{}}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  await fillNewSkill();
  await userEvent.click(screen.getByRole('checkbox', { name: 'Published' }));
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() => expect(writes()).toHaveLength(1));
  expect(writes()[0].body).toMatchObject({
    enabled: false,
    all_groups: false,
    group_ids: [],
  });
});

test('编辑不限分组技能时复选框已勾选且不显示分组列表', async () => {
  render(
    <PromptTemplateEditor
      template={unrestricted}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  const allGroups = await screen.findByRole('checkbox', { name: ALL_GROUPS });
  await waitFor(() => expect(allGroups.disabled).toBe(false));
  expect(allGroups.checked).toBe(true);
  expect(screen.queryByRole('checkbox', { name: 'Visible group' })).toBeNull();
});

test('取消不限分组后必须重新选择分组才能发布，提交 all_groups=false 与所选分组 ID', async () => {
  render(
    <PromptTemplateEditor
      template={unrestricted}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  const allGroups = await screen.findByRole('checkbox', { name: ALL_GROUPS });
  await waitFor(() => expect(allGroups.disabled).toBe(false));
  await userEvent.click(allGroups);
  const group = screen.getByRole('checkbox', { name: 'Visible group' });
  expect(group.checked).toBe(false);
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(writes()).toHaveLength(0);
  await userEvent.click(group);
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() => expect(writes()).toHaveLength(1));
  expect(writes()[0]).toMatchObject({
    url: '/api/prompt-center/admin/8',
    method: 'put',
    body: { all_groups: false, group_ids: [9], version: 4 },
  });
});

test('缺少 all_groups 的旧技能按限定分组编辑并提交 all_groups=false', async () => {
  const legacy = { ...restricted };
  delete legacy.all_groups;
  render(
    <PromptTemplateEditor
      template={legacy}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  const group = await screen.findByRole('checkbox', { name: 'Visible group' });
  expect(group.checked).toBe(true);
  expect(screen.getByRole('checkbox', { name: ALL_GROUPS }).checked).toBe(
    false,
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() => expect(writes()).toHaveLength(1));
  expect(writes()[0].body).toMatchObject({
    all_groups: false,
    group_ids: [9],
  });
});

test('技能目录中不限分组卡片显示不限分组，限定分组卡片显示分组名称而非 code', () => {
  render(
    <PromptCatalog
      templates={[unrestricted, restricted]}
      bindings={[]}
      canInstall
      onInstall={() => {}}
    />,
  );
  const openCard = screen
    .getByRole('heading', { name: 'Open skill' })
    .closest('article');
  expect(within(openCard).getByText(ALL_GROUPS)).toBeTruthy();
  expect(within(openCard).queryByText('No eligible groups')).toBeNull();
  const restrictedCard = screen
    .getByRole('heading', { name: 'Restricted skill' })
    .closest('article');
  expect(within(restrictedCard).getByText('Visible group')).toBeTruthy();
  expect(within(restrictedCard).queryByText(ALL_GROUPS)).toBeNull();
  expect(screen.queryByText(/hidden-code/)).toBeNull();
});

test('令牌已安装列表中不限分组技能显示不限分组，限定分组技能显示分组名称而非 code', async () => {
  bindings = [
    { template: unrestricted, enabled: true, sort_order: 0 },
    { template: restricted, enabled: true, sort_order: 1 },
  ];
  render(<TokenPromptManager token={{ id: 4 }} />);
  const installed = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  const openCard = within(installed)
    .getByRole('heading', { name: 'Open skill' })
    .closest('li');
  expect(within(openCard).getByText(ALL_GROUPS)).toBeTruthy();
  expect(within(openCard).queryByText('No eligible groups')).toBeNull();
  const restrictedCard = within(installed)
    .getByRole('heading', { name: 'Restricted skill' })
    .closest('li');
  expect(within(restrictedCard).getByText('Visible group')).toBeTruthy();
  expect(within(restrictedCard).queryByText(ALL_GROUPS)).toBeNull();
  expect(screen.queryByText(/hidden-code/)).toBeNull();
});

test('顶部允许分组统计在存在不限分组技能时显示不限分组', async () => {
  render(<PromptCenter />);
  await screen.findByText('Open skill');
  const stat = screen.getAllByText('Allowed groups')[0].parentElement;
  expect(stat.textContent).toContain(ALL_GROUPS);
});

test('顶部允许分组统计在仅有白名单技能时显示去重后的分组数', async () => {
  catalogTemplates = [restricted, { ...restricted, id: 9 }];
  render(<PromptCenter />);
  await screen.findAllByText('Restricted skill');
  const stat = screen.getAllByText('Allowed groups')[0].parentElement;
  expect(stat.textContent).toContain('1');
  expect(stat.textContent).not.toContain(ALL_GROUPS);
});

test('顶部允许分组统计在存在不限分组技能时显示不限分组', async () => {
  render(<PromptCenter />);
  await screen.findByText('Open skill');
  const stat = screen.getAllByText('Allowed groups')[0].parentElement;
  expect(stat.textContent).toContain(ALL_GROUPS);
});

test('顶部允许分组统计在仅有白名单技能时显示去重后的分组数', async () => {
  catalogTemplates = [restricted, { ...restricted, id: 9 }];
  render(<PromptCenter />);
  await screen.findAllByText('Restricted skill');
  const stat = screen.getAllByText('Allowed groups')[0].parentElement;
  expect(stat.textContent).toContain('1');
  expect(stat.textContent).not.toContain(ALL_GROUPS);
});
