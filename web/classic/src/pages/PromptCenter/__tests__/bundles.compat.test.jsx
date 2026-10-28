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
import PromptCenter from '../index';
import PromptTemplateEditor from '../PromptTemplateEditor';
import PromptCatalog from '../PromptCatalog';
import TokenPromptManager from '../TokenPromptManager';
import SkillStore from '../SkillStore';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const bundle = {
  id: 7,
  title: 'Writing collection',
  kind: 'bundle',
  category_id: 4,
  category_name: 'Writing',
  enabled: true,
  version: 2,
  group_ids: [9],
  skill_count: 2,
  skills: [
    { title: 'Outline', content: 'Secret outline' },
    { title: 'Polish', content: 'Secret polish' },
  ],
};
let calls, denied, categories;
beforeEach(() => {
  calls = [];
  denied = false;
  categories = [{ id: 4, name: 'Writing', version: 3 }];
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 10 }));
  API.defaults.adapter = async (config) => {
    const body =
      config.data instanceof FormData
        ? config.data
        : config.data
          ? JSON.parse(config.data)
          : undefined;
    calls.push({ url: config.url, method: config.method, body });
    let data;
    if (config.url === '/api/prompt-center/admin/store/')
      data = [
        {
          id: 'bundled:writing-kit',
          title: 'Writing kit',
          source: 'bundled',
          kind: 'bundle',
          skill_count: 2,
        },
      ];
    else if (config.url === '/api/prompt-center/admin/store/?source=online')
      throw new Error('Store offline');
    else if (
      config.url ===
        '/api/prompt-center/admin/store/bundled%3Awriting-kit/import' ||
      config.url === '/api/prompt-center/admin/import/archive' ||
      config.url === '/api/prompt-center/admin/import/repository'
    )
      data = {
        ...bundle,
        id: undefined,
        enabled: false,
        version: 0,
        group_ids: [],
        category_id: 0,
      };
    else if (config.url === '/api/group/tags') data = [];
    else if (config.url === '/api/prompt-center/categories/') data = categories;
    else if (config.url === '/api/prompt-center/admin/categories/') {
      data = { id: 5, name: body.name, version: 1 };
      categories = [...categories, data];
    } else if (config.url === '/api/prompt-center/admin/categories/4') {
      data = { ...categories[0], name: body.name, version: 4 };
      categories = [data];
    } else if (
      config.url === '/api/prompt-center/' ||
      (config.method === 'get' && config.url === '/api/prompt-center/admin/')
    )
      data = [bundle, { id: 8, title: 'Legacy prompt', enabled: true }];
    else if (config.url === '/api/group/details')
      data = [{ id: 9, name: 'Visible group', code: 'hidden-code' }];
    else if (config.url === '/api/token/4/prompts')
      data = [{ template: bundle, enabled: true }];
    else if (config.method === 'put' || config.method === 'post') data = body;
    else throw new Error('Unexpected URL ' + config.url);
    return {
      data:
        denied && config.method !== 'get'
          ? { success: false, message: 'Save rejected' }
          : { success: true, data },
      status: 200,
      headers: {},
      config,
    };
  };
});
afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

test('合集卡片可展开技能标题但不展示正文，旧模板仍为单技能', async () => {
  render(
    <PromptCatalog
      templates={[bundle, { id: 8, title: 'Legacy prompt', enabled: true }]}
      bindings={[]}
      canInstall
      onInstall={() => {}}
    />,
  );
  await userEvent.click(screen.getByText('2 skills'));
  expect(screen.getByText('Outline')).toBeTruthy();
  expect(screen.getByText('Polish')).toBeTruthy();
  expect(screen.queryByText('Secret outline')).toBeNull();
  expect(screen.getByText('Single skill')).toBeTruthy();
});

test('分类与类型筛选联合工作，未分类保留旧模板', async () => {
  render(<PromptCenter />);
  await screen.findByText('Writing collection');
  await userEvent.selectOptions(screen.getByLabelText('Filter category'), '4');
  expect(screen.queryByText('Legacy prompt')).toBeNull();
  await userEvent.click(
    within(
      screen.getByRole('radiogroup', { name: 'Filter skill type' }),
    ).getByRole('radio', { name: /^Single skill/ }),
  );
  expect(screen.queryByText('Writing collection')).toBeNull();
  await userEvent.selectOptions(screen.getByLabelText('Filter category'), '0');
  expect(screen.getByText('Legacy prompt')).toBeTruthy();
});

test('合集编辑器按新顺序提交并在保存失败后保留草稿', async () => {
  render(
    <PromptTemplateEditor
      template={bundle}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  await userEvent.click(
    screen.getByRole('button', { name: 'Move skill 2 up' }),
  );
  denied = true;
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByText('Save rejected');
  expect(screen.getByLabelText('Skill 1 title').value).toBe('Polish');
  expect(screen.getByLabelText('Skill 1 text').value).toBe('Secret polish');
  expect(calls.find((call) => call.method === 'put').body).toEqual(
    expect.objectContaining({
      kind: 'bundle',
      category_id: 4,
      skills: [bundle.skills[1], bundle.skills[0]],
      version: 2,
      group_ids: [9],
    }),
  );
});

test('合集内部技能不能少于两项，重复标题阻止保存', async () => {
  render(
    <PromptTemplateEditor
      template={bundle}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  expect(screen.getByRole('button', { name: 'Remove skill 1' }).disabled).toBe(
    true,
  );
  await userEvent.clear(screen.getByLabelText('Skill 2 title'));
  await userEvent.type(screen.getByLabelText('Skill 2 title'), 'Outline');
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('合集可新增和移除技能，达到十六项后禁用新增', async () => {
  const view = render(
    <PromptTemplateEditor
      template={bundle}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await userEvent.click(screen.getByRole('button', { name: 'Add skill' }));
  expect(screen.getByLabelText('Skill 3 title')).toBeTruthy();
  await userEvent.click(screen.getByRole('button', { name: 'Remove skill 3' }));
  expect(screen.queryByLabelText('Skill 3 title')).toBeNull();
  view.unmount();
  render(
    <PromptTemplateEditor
      template={{
        ...bundle,
        skills: Array.from({ length: 16 }, (_, index) => ({
          title: `Skill ${index + 1}`,
          content: 'Text',
        })),
      }}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  expect(screen.getByRole('button', { name: 'Add skill' }).disabled).toBe(true);
});

test('安装详情展示合集明细但不展示正文', async () => {
  render(<TokenPromptManager token={{ id: 4 }} />);
  const region = await screen.findByRole('region', {
    name: 'Installed prompts',
  });
  await userEvent.click(within(region).getByText('2 skills'));
  expect(within(region).getByText('Polish')).toBeTruthy();
  expect(screen.queryByText('Secret polish')).toBeNull();
});

test('管理员可新增和改名分类，改名请求保留版本', async () => {
  render(<PromptCenter />);
  await userEvent.click(
    screen.getByRole('button', { name: 'Manage categories' }),
  );
  await screen.findByLabelText('Rename Writing');
  await userEvent.type(screen.getByLabelText('New category name'), 'Research');
  await userEvent.click(screen.getByRole('button', { name: 'Add category' }));
  await screen.findByLabelText('Rename Research');
  await userEvent.clear(screen.getByLabelText('Rename Writing'));
  await userEvent.type(
    screen.getByLabelText('Rename Writing'),
    'Writing tools',
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save Writing' }));
  await waitFor(() =>
    expect(calls).toContainEqual({
      url: '/api/prompt-center/admin/categories/4',
      method: 'put',
      body: { name: 'Writing tools', version: 3 },
    }),
  );
});

test('商店在线刷新失败保留内置条目，导入只打开副本不写模板', async () => {
  const imported = [];
  render(
    <SkillStore
      onCancel={() => {}}
      onImport={(draft) => imported.push(draft)}
    />,
  );
  await screen.findByText('Writing kit');
  await userEvent.click(
    screen.getByRole('button', { name: 'Refresh online store' }),
  );
  await screen.findByText('Store offline');
  expect(screen.getByText('Writing kit')).toBeTruthy();
  await userEvent.click(
    screen.getByRole('button', { name: 'Import Writing kit and edit' }),
  );
  await waitFor(() => expect(imported).toHaveLength(1));
  expect(imported[0].enabled).toBe(false);
  expect(imported[0].category_id).toBe(0);
  expect(calls.some((call) => call.url === '/api/prompt-center/admin/')).toBe(
    false,
  );
});

test('压缩包导入失败保留现有编辑内容，成功后为未发布新草稿', async () => {
  render(
    <PromptTemplateEditor
      template={bundle}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  const input = screen.getByLabelText('Upload skill archive');
  denied = true;
  await userEvent.upload(
    input,
    new File(['zip'], 'skills.zip', { type: 'application/zip' }),
  );
  await screen.findByText('Save rejected');
  expect(screen.getByLabelText('Prompt title').value).toBe(
    'Writing collection',
  );
  expect(screen.getByLabelText('Skill 1 text').value).toBe('Secret outline');
  denied = false;
  await userEvent.upload(
    input,
    new File(['zip'], 'skills.zip', { type: 'application/zip' }),
  );
  await waitFor(() =>
    expect(screen.getByRole('checkbox', { name: 'Published' }).checked).toBe(
      false,
    ),
  );
  expect(screen.getByLabelText('Skill category').value).toBe('0');
  expect(
    calls.filter(
      (call) => call.url === '/api/prompt-center/admin/import/archive',
    ),
  ).toHaveLength(2);
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('压缩包解析期间禁止保存和修改，完成后恢复编辑', async () => {
  const adapter = API.defaults.adapter;
  let finish;
  const pending = new Promise((resolve) => {
    finish = resolve;
  });
  API.defaults.adapter = async (config) => {
    if (config.url === '/api/prompt-center/admin/import/archive') await pending;
    return adapter(config);
  };
  render(
    <PromptTemplateEditor
      template={bundle}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  await userEvent.upload(
    screen.getByLabelText('Upload skill archive'),
    new File(['zip'], 'skills.zip', { type: 'application/zip' }),
  );
  expect(screen.getByRole('button', { name: 'Save template' }).disabled).toBe(
    true,
  );
  expect(screen.getByLabelText('Skill 1 text').matches(':disabled')).toBe(true);
  finish();
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Save template' }).disabled).toBe(
      false,
    ),
  );
});

test('仓库导入失败保留网址与原草稿，重试成功转为未发布副本', async () => {
  render(
    <PromptTemplateEditor
      template={bundle}
      onSaved={() => {}}
      onCancel={() => {}}
    />,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Import GitHub repository' }),
  );
  const url = screen.getByLabelText('Public GitHub repository URL');
  await userEvent.type(
    url,
    'https://github.com/example/skills/tree/main/writing',
  );
  denied = true;
  await userEvent.click(
    screen.getByRole('button', { name: 'Import repository and edit' }),
  );
  await screen.findByText('Save rejected');
  expect(url.value).toBe('https://github.com/example/skills/tree/main/writing');
  expect(screen.getByLabelText('Skill 1 text').value).toBe('Secret outline');
  denied = false;
  await userEvent.click(
    screen.getByRole('button', { name: 'Import repository and edit' }),
  );
  await waitFor(() =>
    expect(screen.getByRole('checkbox', { name: 'Published' }).checked).toBe(
      false,
    ),
  );
  expect(
    calls.filter(
      (call) => call.url === '/api/prompt-center/admin/import/repository',
    ),
  ).toEqual([
    {
      url: '/api/prompt-center/admin/import/repository',
      method: 'post',
      body: { url: 'https://github.com/example/skills/tree/main/writing' },
    },
    {
      url: '/api/prompt-center/admin/import/repository',
      method: 'post',
      body: { url: 'https://github.com/example/skills/tree/main/writing' },
    },
  ]);
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('商店仓库导入期间锁定网址、导入和关闭，完成后仅交付草稿', async () => {
  const adapter = API.defaults.adapter;
  let finish;
  const pending = new Promise((resolve) => {
    finish = resolve;
  });
  API.defaults.adapter = async (config) => {
    if (config.url === '/api/prompt-center/admin/import/repository')
      await pending;
    return adapter(config);
  };
  const imported = [];
  const canceled = [];
  render(
    <SkillStore
      onCancel={() => canceled.push(true)}
      onImport={(draft) => imported.push(draft)}
    />,
  );
  await screen.findByText('Writing kit');
  await userEvent.click(
    screen.getByRole('button', { name: 'Import GitHub repository' }),
  );
  await userEvent.type(
    screen.getByLabelText('Public GitHub repository URL'),
    'https://github.com/example/skills',
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Import repository and edit' }),
  );
  expect(screen.getByLabelText('Public GitHub repository URL').disabled).toBe(
    true,
  );
  expect(
    screen.getByRole('button', { name: 'Import repository and edit' }).disabled,
  ).toBe(true);
  expect(
    screen.getByRole('button', { name: 'Import ZIP archive' }).disabled,
  ).toBe(true);
  await userEvent.keyboard('{Escape}');
  expect(canceled).toHaveLength(0);
  finish();
  await waitFor(() => expect(imported).toHaveLength(1));
  expect(imported[0].enabled).toBe(false);
});
