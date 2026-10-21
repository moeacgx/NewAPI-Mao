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
import { beforeEach, afterEach, expect, test } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PromptCenter from '../index';
import TokenPromptBadge from '../../../components/table/tokens/TokenPromptBadge';
import TokensMobileList from '../../../components/table/tokens/TokensMobileList';
import { getTokensColumns } from '../../../components/table/tokens/TokensColumnDefs';
import TokenPromptManager from '../TokenPromptManager';
import PromptTemplateEditor from '../PromptTemplateEditor';
import { API } from '../../../helpers/api';

const adapter = API.defaults.adapter;
const template = {
  id: 7,
  title: 'Code guide',
  description: 'Readable code',
  content: 'Write clear code.',
  enabled: true,
  version: 3,
  group_ids: [9],
  group_details: [{ id: 9, name: '高级模型组' }],
};
let bindings, calls, denyWrite, templates;

test('删除白名单分组后允许移除失效项并下架模板', async () => {
  render(
    <PromptTemplateEditor
      template={{ ...template, group_ids: [99] }}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await userEvent.click(
    await screen.findByRole('checkbox', { name: /Unavailable group.*99/ }),
  );
  await userEvent.click(screen.getByRole('checkbox', { name: 'Published' }));
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(
      calls.some(
        (call) =>
          call.method === 'put' &&
          call.body.enabled === false &&
          call.body.group_ids.length === 0,
      ),
    ).toBe(true),
  );
});
beforeEach(() => {
  calls = [];
  bindings = [];
  denyWrite = false;
  templates = [template];
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 1 }));
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ url: config.url, method: config.method, body });
    let data;
    if (config.url === '/api/prompt-center/') data = templates;
    else if (config.url === '/api/prompt-center/admin/')
      data = config.method === 'post' ? { id: 12, ...body } : templates;
    else if (
      config.url.startsWith('/api/token/?') ||
      config.url.startsWith('/api/token/search?')
    )
      data = { items: [{ id: 4, name: 'codexQ' }], total: 1 };
    else if (config.url === '/api/group/details')
      data = [{ id: 9, name: '高级模型组', code: 'premium-code' }];
    else if (config.url === '/api/token/4/prompts') {
      if (config.method === 'put') {
        if (denyWrite)
          return {
            data: { success: false, message: 'Save rejected' },
            status: 200,
            headers: {},
            config,
          };
        bindings = body.bindings.map((binding) => ({
          ...binding,
          template:
            templates.find((item) => item.id === binding.template_id) ||
            template,
        }));
      }
      data = bindings;
    } else if (config.url === '/api/prompt-center/admin/7')
      data = { ...template, ...body };
    else throw new Error('Unexpected URL ' + config.url);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});
afterEach(() => {
  API.defaults.adapter = adapter;
});

test('选择自己的令牌后安装，停用和卸载均保存完整关联且不读取密钥', async () => {
  render(<PromptCenter />);
  await screen.findByText('Code guide');
  await userEvent.selectOptions(screen.getByLabelText('Choose a token'), '4');
  await userEvent.click(
    await screen.findByRole('button', { name: 'Install Code guide' }),
  );
  const installed = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  await waitFor(() =>
    expect(within(installed).getByText('Code guide')).toBeTruthy(),
  );
  await userEvent.click(
    within(installed).getByRole('checkbox', { name: 'Enable Code guide' }),
  );
  await waitFor(() => expect(bindings[0].enabled).toBe(false));
  await userEvent.click(
    within(installed).getByRole('button', { name: 'Uninstall Code guide' }),
  );
  await waitFor(() => expect(bindings).toEqual([]));
  expect(calls.some((call) => call.url.includes('/key'))).toBe(false);
  expect(calls.some((call) => call.url.includes('/admin'))).toBe(false);
  expect(screen.queryByRole('button', { name: 'New template' })).toBeNull();
});

test('多项排序保存顺序且不改变启用状态', async () => {
  const second = { ...template, id: 8, title: 'Review guide' };
  templates.push(second);
  bindings = [
    { template, enabled: false, sort_order: 0 },
    { template: second, enabled: true, sort_order: 1 },
  ];
  render(<TokenPromptManager token={{ id: 4 }} />);
  await userEvent.click(
    await screen.findByRole('button', { name: 'Move Review guide up' }),
  );
  await waitFor(() =>
    expect(calls).toContainEqual({
      url: '/api/token/4/prompts',
      method: 'put',
      body: {
        bindings: [
          { template_id: 8, enabled: true, sort_order: 0 },
          { template_id: 7, enabled: false, sort_order: 1 },
        ],
      },
    }),
  );
});

test('下架安装项不能重新启用但可卸载，达到八项时禁用新增安装', async () => {
  bindings = Array.from({ length: 8 }, (_, index) => ({
    template: {
      ...template,
      id: index + 20,
      title: `Installed ${index}`,
      enabled: false,
    },
    enabled: false,
    sort_order: index,
  }));
  render(<TokenPromptManager token={{ id: 4 }} templates={templates} />);
  const disabled = await screen.findByRole('checkbox', {
    name: 'Enable Installed 0',
  });
  expect(disabled.disabled).toBe(true);
  expect(
    screen.getByRole('button', { name: 'Install Code guide' }).disabled,
  ).toBe(true);
  await userEvent.click(
    screen.getByRole('button', { name: 'Uninstall Installed 0' }),
  );
  await waitFor(() => expect(bindings).toHaveLength(7));
});

test('目录为空展示空状态，未选令牌不发装载请求', async () => {
  templates = [];
  render(<PromptCenter />);
  await screen.findByText('No prompts available');
  expect(calls.some((call) => call.url.includes('/prompts'))).toBe(false);
});

test('桌面和手机令牌名称旁均呈现启用技能数量，长名称保留完整可访问名称', () => {
  const token = {
    id: 4,
    name: 'A'.repeat(100),
    prompt_count: 2,
    enabled_prompt_count: 1,
  };
  const name = `${token.name}: 2 installed prompts, 1 enabled`;
  const column = getTokensColumns({ t: (key) => key }).find(
    (item) => item.dataIndex === 'name',
  );
  const view = render(column.render(token.name, token));
  expect(screen.getByRole('button', { name }).getAttribute('data-active')).toBe(
    'true',
  );
  view.unmount();
  render(<TokensMobileList tokens={[token]} columns={[]} t={(key) => key} />);
  expect(screen.getByRole('button', { name })).toBeTruthy();
  expect(
    screen.getByRole('heading', { name: new RegExp(token.name) }),
  ).toBeTruthy();
});

test('管理员填写超长多字节正文被阻止且不发送写入', async () => {
  render(
    <PromptTemplateEditor
      template={{ ...template, content: '中'.repeat(11000) }}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Save template' }).disabled).toBe(
      false,
    ),
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(
    calls.some((call) => call.method === 'put' || call.method === 'post'),
  ).toBe(false);
});

test('管理员导入 SKILL.md 按原始文本创建模板，白名单只提交稳定 ID', async () => {
  render(
    <PromptTemplateEditor
      template={{}}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  const content = '# Review\n```sh\nrm -rf example\n```';
  const file = new File([content], 'SKILL.md', { type: 'text/markdown' });
  Object.defineProperty(file, 'text', { value: async () => content });
  await userEvent.upload(
    screen.getByLabelText('Import Markdown or SKILL.md as text'),
    file,
  );
  await waitFor(() =>
    expect(screen.getByLabelText('Prompt text').value).toBe(content),
  );
  await userEvent.type(screen.getByLabelText('Prompt title'), 'Review text');
  await userEvent.click(
    await screen.findByRole('checkbox', { name: '高级模型组' }),
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls).toContainEqual(
      expect.objectContaining({
        url: '/api/prompt-center/admin/',
        method: 'post',
        body: expect.objectContaining({
          title: 'Review text',
          content,
          group_ids: [9],
        }),
      }),
    ),
  );
});

test('启用正文总量超过上限时保留原安装状态并提示', async () => {
  const second = {
    ...template,
    id: 8,
    title: 'Review guide',
    content: 'b'.repeat(32768),
  };
  const third = { ...template, id: 9, title: 'Extra guide', content: 'c' };
  bindings = [
    {
      template: { ...template, content: 'a'.repeat(32768) },
      enabled: true,
      sort_order: 0,
    },
    { template: second, enabled: true, sort_order: 1 },
  ];
  render(<TokenPromptManager token={{ id: 4 }} templates={[third]} />);
  await screen.findByRole('checkbox', { name: 'Enable Review guide' });
  await userEvent.click(
    screen.getByRole('button', { name: 'Install Extra guide' }),
  );
  await screen.findByText(
    'Enabled prompts must total no more than 65536 bytes',
  );
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('保存失败保留安装列表并展示错误，用户可重试', async () => {
  bindings = [{ template, enabled: true, sort_order: 0 }];
  render(<PromptCenter />);
  await userEvent.selectOptions(
    await screen.findByLabelText('Choose a token'),
    '4',
  );
  const installed = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  await within(installed).findByText('Code guide');
  denyWrite = true;
  await userEvent.click(
    within(installed).getByRole('button', { name: 'Uninstall Code guide' }),
  );
  await screen.findByText('Save rejected');
  expect(within(installed).getByText('Code guide')).toBeTruthy();
  denyWrite = false;
  await userEvent.click(
    within(installed).getByRole('button', { name: 'Uninstall Code guide' }),
  );
  await within(installed).findByText('No prompts installed');
});

test('未安装隐藏徽标，全部停用灰显，键盘打开详情显示当前分组名称', async () => {
  const view = render(
    <TokenPromptBadge token={{ id: 4, name: 'codexQ', prompt_count: 0 }} />,
  );
  expect(screen.queryByRole('button')).toBeNull();
  bindings = [{ template, enabled: false, sort_order: 0 }];
  view.rerender(
    <TokenPromptBadge
      token={{
        id: 4,
        name: 'codexQ',
        prompt_count: 1,
        enabled_prompt_count: 0,
      }}
    />,
  );
  const badge = screen.getByRole('button', {
    name: 'codexQ: 1 installed prompts, 0 enabled',
  });
  expect(badge.getAttribute('data-active')).toBe('false');
  badge.focus();
  await userEvent.keyboard('{Enter}');
  await screen.findByText(/高级模型组/);
  expect(screen.queryByText('premium-code')).toBeNull();
});

test('管理员编辑提交版本及稳定分组 ID，普通用户没有编辑入口', async () => {
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 10 }));
  render(<PromptCenter />);
  await userEvent.click(
    await screen.findByRole('button', { name: 'Edit Code guide' }),
  );
  const title = screen.getByLabelText('Prompt title');
  await userEvent.clear(title);
  await userEvent.type(title, 'Updated guide');
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls).toContainEqual(
      expect.objectContaining({
        url: '/api/prompt-center/admin/7',
        method: 'put',
        body: expect.objectContaining({
          title: 'Updated guide',
          version: 3,
          group_ids: [9],
        }),
      }),
    ),
  );
});
