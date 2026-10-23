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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PromptTemplateEditor from '../PromptTemplateEditor';
import PromptCatalog from '../PromptCatalog';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const resource = {
  path: 'references/examples.md',
  content: 'Private examples',
};
const template = {
  id: 31,
  title: 'Research assistant',
  description: 'Choose this skill to research a topic.',
  kind: 'prompt',
  load_mode: 'progressive',
  content: '中'.repeat(12000),
  skills: [{ id: 'research', path: 'SKILL.md', resources: [resource] }],
  enabled: true,
  version: 4,
  group_ids: [9],
};
let calls;
let denied;

beforeEach(() => {
  calls = [];
  denied = false;
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ url: config.url, method: config.method, body });
    let data = body;
    if (config.url === '/api/group/details')
      data = [{ id: 9, name: 'Visible research group', code: 'private-code' }];
    if (config.url === '/api/prompt-center/categories/') data = [];
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

test('按需单技能允许超过 32 KiB，编辑资源后保存保留技能身份和路径', async () => {
  render(<PromptTemplateEditor template={template} onSaved={() => {}} />);
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  fireEvent.change(screen.getByLabelText('Resource 1 text'), {
    target: { value: 'Revised private examples' },
  });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls.find((call) => call.method === 'put')?.body).toEqual(
      expect.objectContaining({
        content: template.content,
        load_mode: 'progressive',
        version: 4,
        group_ids: [9],
        skills: [
          expect.objectContaining({
            id: 'research',
            path: 'SKILL.md',
            resources: [{ ...resource, content: 'Revised private examples' }],
          }),
        ],
      }),
    ),
  );
});

test('按需合集分别保存大文件，排序和失败重试不丢失技能用途或引用资源', async () => {
  const skills = [
    {
      id: 'outline',
      title: 'Outline',
      description: 'Plan a report.',
      path: 'skills/outline/SKILL.md',
      content: 'A'.repeat(40000),
      resources: [resource],
    },
    {
      id: 'review',
      title: 'Review',
      description: 'Review the report.',
      path: 'skills/review/SKILL.md',
      content: 'B'.repeat(40000),
      resources: [],
    },
  ];
  render(
    <PromptTemplateEditor
      template={{ ...template, kind: 'bundle', content: '', skills }}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  await userEvent.click(
    screen.getByRole('button', { name: 'Move skill 2 up' }),
  );
  denied = true;
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByText('Save rejected');
  expect(document.activeElement).toBe(screen.getByRole('alert'));
  expect(screen.getByLabelText('Skill 2 description').value).toBe(
    'Plan a report.',
  );
  expect(screen.getByLabelText('Skill 2 resource 1 text').value).toBe(
    resource.content,
  );
  expect(calls.find((call) => call.method === 'put')?.body.skills).toEqual([
    skills[1],
    skills[0],
  ]);
});

test('旧模板默认始终装载并继续拒绝超过 32 KiB 的正文', async () => {
  render(
    <PromptTemplateEditor
      template={{ ...template, load_mode: undefined }}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  expect(screen.getByLabelText('Loading mode').value).toBe('eager');
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('新模板默认按需加载，缺少用途禁止发布，补全后可保存', async () => {
  render(
    <PromptTemplateEditor
      template={{ title: 'New skill', content: 'Read on demand' }}
      onSaved={() => {}}
    />,
  );
  await userEvent.click(
    await screen.findByRole('checkbox', { name: 'Visible research group' }),
  );
  expect(screen.getByLabelText('Loading mode').value).toBe('progressive');
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(calls.some((call) => call.method === 'post')).toBe(false);
  fireEvent.change(screen.getByLabelText('Prompt description'), {
    target: { value: 'Use this skill when answering research questions.' },
  });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls.find((call) => call.method === 'post')?.body.load_mode).toBe(
      'progressive',
    ),
  );
});

test('按需资源可新增和删除，越界路径会阻止保存', async () => {
  render(<PromptTemplateEditor template={template} onSaved={() => {}} />);
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  await userEvent.click(
    screen.getByRole('button', { name: 'Add text resource' }),
  );
  fireEvent.change(screen.getByLabelText('Resource 2 path'), {
    target: { value: '../outside.md' },
  });
  fireEvent.change(screen.getByLabelText('Resource 2 text'), {
    target: { value: 'Text' },
  });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(calls.some((call) => call.method === 'put')).toBe(false);
  await userEvent.click(
    screen.getByRole('button', { name: 'Remove resource 2' }),
  );
  expect(screen.queryByLabelText('Resource 2 path')).toBeNull();
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(
      calls.find((call) => call.method === 'put')?.body.skills[0].resources,
    ).toEqual([resource]),
  );
});

test('普通用户看到按需模式摘要，但看不到正文或资源路径和内容', () => {
  render(
    <PromptCatalog
      templates={[template]}
      bindings={[]}
      canInstall
      onInstall={() => {}}
    />,
  );
  expect(screen.getByText('On-demand skill')).toBeTruthy();
  expect(screen.queryByText(template.content)).toBeNull();
  expect(screen.queryByText(resource.path)).toBeNull();
  expect(screen.queryByText(resource.content)).toBeNull();
});

test('按需合集超过十六项仍能保存，每个技能独立保留描述', async () => {
  const skills = Array.from({ length: 17 }, (_, index) => ({
    id: `skill-${index}`,
    title: `Skill ${index + 1}`,
    description: `Use for topic ${index + 1}.`,
    content: 'Instructions',
    path: `skills/skill-${index}/SKILL.md`,
    resources: [],
  }));
  render(
    <PromptTemplateEditor
      template={{ ...template, kind: 'bundle', content: '', skills }}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  expect(screen.getByRole('button', { name: 'Add skill' }).disabled).toBe(
    false,
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls.find((call) => call.method === 'put')?.body.skills).toEqual(
      skills,
    ),
  );
});

test('未填子技能用途时不能发布，但可以下架保存草稿', async () => {
  const skills = [
    { title: 'Outline', description: '', content: 'Outline instructions' },
    {
      title: 'Review',
      description: 'Review a draft',
      content: 'Review instructions',
    },
  ];
  render(
    <PromptTemplateEditor
      template={{ ...template, kind: 'bundle', content: '', skills }}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(calls.some((call) => call.method === 'put')).toBe(false);
  await userEvent.click(screen.getByRole('checkbox', { name: 'Published' }));
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls.find((call) => call.method === 'put')?.body.enabled).toBe(
      false,
    ),
  );
});

test('本地导入超过 32 KiB 的 Markdown 切换为按需加载，读取失败保留正文', async () => {
  render(
    <PromptTemplateEditor
      template={{ ...template, load_mode: 'eager', content: 'Original draft' }}
      onSaved={() => {}}
    />,
  );
  const content = '# Large skill\n' + 'x'.repeat(40000);
  const file = new File([content], 'SKILL.md', { type: 'text/markdown' });
  Object.defineProperty(file, 'text', { value: async () => content });
  const input = screen.getByLabelText('Import Markdown or SKILL.md as text');
  await userEvent.upload(input, file);
  await waitFor(() =>
    expect(screen.getByLabelText('Prompt text').value).toBe(content),
  );
  expect(screen.getByLabelText('Loading mode').value).toBe('progressive');
  const unreadable = new File(['bad'], 'unreadable.md', {
    type: 'text/markdown',
  });
  Object.defineProperty(unreadable, 'text', {
    value: async () => {
      throw new Error('Read denied');
    },
  });
  await userEvent.upload(input, unreadable);
  await screen.findByRole('alert');
  expect(screen.getByLabelText('Prompt text').value).toBe(content);
});

test('单技能改为合集时保留已有正文、用途、身份和引用资源', async () => {
  render(<PromptTemplateEditor template={template} onSaved={() => {}} />);
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  fireEvent.change(screen.getByLabelText('Skill type'), {
    target: { value: 'bundle' },
  });
  expect(screen.getByLabelText('Skill 1 text').value).toBe(template.content);
  expect(screen.getByLabelText('Skill 1 description').value).toBe(
    template.description,
  );
  expect(screen.getByLabelText('Skill 1 resource 1 text').value).toBe(
    resource.content,
  );
  fireEvent.change(screen.getByLabelText('Skill 2 title'), {
    target: { value: 'Second skill' },
  });
  fireEvent.change(screen.getByLabelText('Skill 2 description'), {
    target: { value: 'Use this to review the first skill output.' },
  });
  fireEvent.change(screen.getByLabelText('Skill 2 text'), {
    target: { value: 'Review the answer' },
  });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls.find((call) => call.method === 'put')?.body.skills[0]).toEqual(
      expect.objectContaining({
        ...template.skills[0],
        content: template.content,
        description: template.description,
      }),
    ),
  );
});

test('合集拒绝重复主文件路径，清空路径后保留给服务端生成', async () => {
  const skills = ['Outline', 'Review'].map((title) => ({
    title,
    description: `Use this to ${title.toLowerCase()}.`,
    content: 'Instructions',
    path: 'skills/common/SKILL.md',
  }));
  render(
    <PromptTemplateEditor
      template={{ ...template, kind: 'bundle', content: '', skills }}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await screen.findByRole('alert');
  expect(calls.some((call) => call.method === 'put')).toBe(false);
  fireEvent.change(screen.getByLabelText('Skill 2 file path'), {
    target: { value: '' },
  });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(
      calls.find((call) => call.method === 'put')?.body.skills[1].path,
    ).toBe(''),
  );
});

test.each([
  { description: '', titles: ['Outline', 'Review'] },
  { description: 'Research collection', titles: ['Research', 'Research'] },
])(
  '按需合集允许可选总说明和不同路径同名技能：%j',
  async ({ description, titles }) => {
    const skills = titles.map((title, index) => ({
      id: `research-${index}`,
      title,
      description: 'Use this skill for research.',
      path: `skills/research-${index}/SKILL.md`,
      content: 'Research instructions',
      resources: [],
    }));
    render(
      <PromptTemplateEditor
        template={{
          ...template,
          kind: 'bundle',
          description,
          content: '',
          skills,
        }}
        onSaved={() => {}}
      />,
    );
    await screen.findByRole('checkbox', { name: 'Visible research group' });
    await userEvent.click(
      screen.getByRole('button', { name: 'Save template' }),
    );
    await waitFor(() =>
      expect(calls.find((call) => call.method === 'put')?.body.skills).toEqual(
        skills,
      ),
    );
  },
);

test('未导入的无效仓库地址不阻断已有模板保存', async () => {
  render(<PromptTemplateEditor template={template} onSaved={() => {}} />);
  await screen.findByRole('checkbox', { name: 'Visible research group' });
  await userEvent.click(
    screen.getByRole('button', { name: 'Import GitHub repository' }),
  );
  fireEvent.change(screen.getByLabelText('Public GitHub repository URL'), {
    target: { value: 'not-a-repository-url' },
  });
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(calls.find((call) => call.method === 'put')?.body.title).toBe(
      template.title,
    ),
  );
  expect(calls.some((call) => call.url.includes('/import/repository'))).toBe(
    false,
  );
});
