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
import { afterEach, beforeEach, expect, onTestFinished, test } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PromptCenter from '../index';
import TokenPromptManager from '../TokenPromptManager';
import TokenPromptBadge from '../../../components/table/tokens/TokenPromptBadge';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const single = {
  id: 7,
  title: 'Code guide',
  kind: 'prompt',
  category_id: 4,
  category_name: 'Writing',
  enabled: true,
  group_details: [{ id: 9, name: '高级模型组' }],
};
const bundle = {
  id: 8,
  title: 'Writing collection',
  kind: 'bundle',
  category_id: 4,
  category_name: 'Writing',
  enabled: true,
  skill_count: 2,
  group_details: [{ id: 9, name: '高级模型组' }],
};
const legacy = { id: 10, title: 'Legacy prompt', enabled: true };
let calls, bindings;

beforeEach(() => {
  calls = [];
  bindings = [];
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 1 }));
  API.defaults.adapter = async (config) => {
    calls.push({ url: config.url, method: config.method });
    let data;
    if (config.url === '/api/prompt-center/categories/')
      data = [{ id: 4, name: 'Writing', version: 1 }];
    else if (config.url === '/api/prompt-center/')
      data = [single, bundle, legacy];
    else if (config.url.startsWith('/api/token/'))
      data = config.url.endsWith('/prompts')
        ? bindings
        : { items: [{ id: 4, name: 'codexQ' }], total: 1 };
    else throw new Error(`Unexpected ${config.method} ${config.url}`);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

const writes = () => calls.filter((call) => call.method === 'put');

test('令牌徽标打开的安装弹窗与中心入口使用同一主题外壳和内容包装', async () => {
  bindings = [{ template: single, enabled: true, sort_order: 0 }];
  render(
    <TokenPromptBadge
      token={{
        id: 4,
        name: 'codexQ',
        prompt_count: 1,
        enabled_prompt_count: 1,
      }}
    />,
  );
  await userEvent.click(screen.getByRole('button', { name: /^codexQ:/ }));
  const region = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  const modal = region.closest('.prompt-installation-modal');
  expect(modal).toBeTruthy();
  expect(
    region.closest('.prompt-center-page.prompt-installation-content'),
  ).toBe(region.closest('.prompt-installation-content'));
  expect(
    within(modal).getByRole('link', { name: 'Open prompt center' }),
  ).toBeTruthy();
});

test('已安装列表按行展示，下架项标注状态，容量以可访问文本呈现', async () => {
  bindings = [
    { template: single, enabled: true, sort_order: 0 },
    {
      template: { ...bundle, enabled: false },
      enabled: true,
      sort_order: 1,
    },
  ];
  render(<TokenPromptManager token={{ id: 4 }} />);
  const region = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  const rows = await within(region).findAllByRole('listitem');
  expect(rows).toHaveLength(2);
  expect(within(rows[0]).queryByText('Unpublished')).toBeNull();
  expect(within(rows[1]).getByText('Unpublished')).toBeTruthy();
  expect(within(rows[1]).getByText('高级模型组')).toBeTruthy();
  expect(
    within(region).getByLabelText('2 of 8 prompt slots used'),
  ).toBeTruthy();
});

test('卸载后显示未保存状态，放弃修改恢复列表且不发送保存请求', async () => {
  bindings = [{ template: single, enabled: true, sort_order: 0 }];
  render(<TokenPromptManager token={{ id: 4 }} />);
  const region = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  await within(region).findByText('Code guide');
  expect(within(region).getByRole('status').textContent).toBe(
    'No unsaved changes',
  );
  expect(
    within(region).queryByRole('button', { name: 'Discard changes' }),
  ).toBeNull();
  await userEvent.click(
    within(region).getByRole('button', { name: 'Uninstall Code guide' }),
  );
  expect(within(region).queryByText('Code guide')).toBeNull();
  expect(within(region).getByText('Unsaved changes')).toBeTruthy();
  await userEvent.click(
    within(region).getByRole('button', { name: 'Discard changes' }),
  );
  expect(within(region).getByText('Code guide')).toBeTruthy();
  expect(within(region).getByRole('button', { name: 'Save' }).disabled).toBe(
    true,
  );
  expect(writes()).toHaveLength(0);
});

test('从目录选择安装的模板标注为待保存，保存后标注消失', async () => {
  render(<TokenPromptManager token={{ id: 4 }} templateToInstall={single} />);
  const region = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  const row = (await within(region).findAllByRole('listitem'))[0];
  expect(within(row).getByText('Not saved yet')).toBeTruthy();
  await userEvent.click(within(region).getByRole('button', { name: 'Save' }));
  await waitFor(() => expect(writes()).toHaveLength(1));
  await waitFor(() =>
    expect(within(region).queryByText('Not saved yet')).toBeNull(),
  );
});

test('技能类型单选组可用方向键切换，筛选值语义与分类联合生效', async () => {
  // jsdom 未实现 CSS.escape，user-event 按 name 查找同组单选框时需要它。
  const originalCSS = globalThis.CSS;
  globalThis.CSS = {
    escape: (value) => String(value).replace(/[^\w-]/g, (char) => `\\${char}`),
  };
  onTestFinished(() => {
    globalThis.CSS = originalCSS;
  });
  render(<PromptCenter />);
  await screen.findByText('Writing collection');
  const group = screen.getByRole('radiogroup', { name: 'Filter skill type' });
  const all = within(group).getByRole('radio', { name: /^All skill types/ });
  expect(all.checked).toBe(true);
  expect(all.value).toBe('all');
  all.focus();
  await userEvent.keyboard('{ArrowRight}');
  const singleRadio = within(group).getByRole('radio', {
    name: /^Single skill/,
  });
  expect(singleRadio.checked).toBe(true);
  expect(singleRadio.value).toBe('prompt');
  expect(screen.queryByText('Writing collection')).toBeNull();
  expect(screen.getByText('Legacy prompt')).toBeTruthy();
  await userEvent.selectOptions(screen.getByLabelText('Filter category'), '4');
  expect(screen.queryByText('Legacy prompt')).toBeNull();
  expect(screen.getByText('Code guide')).toBeTruthy();
  expect(
    within(group).getByRole('radio', { name: /^Skill bundle/ }).value,
  ).toBe('bundle');
});

test('非默认筛选时出现重置按钮，重置后恢复全部分类与类型', async () => {
  render(<PromptCenter />);
  await screen.findByText('Writing collection');
  expect(screen.queryByRole('button', { name: 'Reset filters' })).toBeNull();
  await userEvent.selectOptions(screen.getByLabelText('Filter category'), '0');
  await userEvent.click(screen.getByRole('radio', { name: /^Skill bundle/ }));
  expect(screen.getByText('No prompts available')).toBeTruthy();
  await userEvent.click(screen.getByRole('button', { name: 'Reset filters' }));
  expect(screen.getByLabelText('Filter category').value).toBe('all');
  expect(screen.getByRole('radio', { name: /^All skill types/ }).checked).toBe(
    true,
  );
  expect(screen.getByText('Writing collection')).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Reset filters' })).toBeNull();
});
