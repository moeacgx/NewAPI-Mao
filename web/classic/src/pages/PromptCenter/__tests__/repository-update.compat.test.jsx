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
import PromptTemplateEditor from '../PromptTemplateEditor';
import RepositorySource from '../RepositorySource';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const source = {
  url: 'https://github.com/acme/review/tree/main/skills/review',
  ref: 'main',
  path: 'skills/review',
  commit: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  imported_at: 1792886400,
  content_hash:
    'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
  locally_modified: false,
};
const template = {
  id: 7,
  title: 'Review guide',
  description: 'Review code changes',
  content: 'Original instructions',
  kind: 'prompt',
  load_mode: 'progressive',
  skills: [
    {
      id: 'stable-review-id',
      title: 'Review guide',
      description: 'Review code changes',
      path: 'skills/review/SKILL.md',
      content: 'Original instructions',
      resources: [],
    },
  ],
  group_ids: [9],
  group_details: [{ id: 9, name: 'Visible group' }],
  category_id: 4,
  sort_order: 8,
  version: 3,
  enabled: true,
  repository_source: source,
};
let calls, current, result, pendingUpdate;

beforeEach(() => {
  calls = [];
  current = structuredClone(template);
  result = 'updated';
  pendingUpdate = undefined;
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 10 }));
  API.defaults.adapter = async (config) => {
    let body;
    if (config.data instanceof FormData) body = config.data;
    else if (config.data) body = JSON.parse(config.data);
    calls.push({ url: config.url, method: config.method, body });
    let data;
    if (config.url === '/api/prompt-center/categories/')
      data = [{ id: 4, name: 'Development', version: 1 }];
    else if (config.url === '/api/group/details')
      data = [{ id: 9, name: 'Visible group', code: 'not-a-display-name' }];
    else if (config.url === '/api/prompt-center/')
      data = [{ ...current, repository_source: undefined, content: undefined }];
    else if (
      config.url === '/api/prompt-center/admin/' &&
      config.method === 'get'
    )
      data = [
        current,
        {
          ...current,
          id: 8,
          title: 'Legacy guide',
          repository_source: undefined,
        },
      ];
    else if (config.url === '/api/prompt-center/admin/import/repository')
      data = {
        ...current,
        id: 0,
        version: 0,
        group_ids: [],
        category_id: 0,
        enabled: false,
        sort_order: 0,
      };
    else if (config.url === '/api/prompt-center/admin/import/archive')
      data = {
        ...current,
        id: 0,
        version: 0,
        group_ids: [],
        category_id: 0,
        enabled: false,
        sort_order: 0,
        repository_source: undefined,
      };
    else if (config.url === '/api/prompt-center/admin/7/update-repository') {
      if (pendingUpdate) await pendingUpdate;
      if (result === 'local-changes' || result === 'version-conflict') {
        const response = {
          status: 409,
          data: {
            success: false,
            message:
              result === 'local-changes'
                ? 'Local changes need confirmation'
                : 'Version conflict',
            code:
              result === 'local-changes'
                ? 'repository_local_changes'
                : undefined,
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
          data: { success: false, message: 'Repository unavailable' },
          status: 200,
          headers: {},
          config,
        };
      if (result === 'updated') {
        current = {
          ...current,
          version: 4,
          repository_source: {
            ...source,
            commit: 'cccccccccccccccccccccccccccccccccccccccc',
            locally_modified: false,
          },
        };
      }
      data = { template: current, updated: result === 'updated' };
    } else if (
      config.url === '/api/prompt-center/admin/' ||
      config.url === '/api/prompt-center/admin/7'
    )
      data = { ...current, ...body };
    else throw new Error(`Unexpected URL ${config.url}`);
    return { data: { success: true, data }, status: 200, headers: {}, config };
  };
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

async function openManagement() {
  render(<PromptCenter />);
  await userEvent.click(
    await screen.findByRole('button', { name: 'Manage templates' }),
  );
  return screen.findByRole('button', {
    name: 'Update Review guide from GitHub',
  });
}

test('GitHub 导入后显示来源并在保存时提交完整来源', async () => {
  render(
    <PromptTemplateEditor
      template={{}}
      onCancel={() => {}}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  await userEvent.click(
    screen.getByRole('button', { name: 'Import GitHub repository' }),
  );
  await userEvent.type(
    screen.getByLabelText('Public GitHub repository URL'),
    source.url,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Import repository and edit' }),
  );
  const link = await screen.findByRole('link', { name: 'GitHub source' });
  expect(link.getAttribute('href')).toBe(source.url);
  expect(screen.getByText('skills/review')).toBeTruthy();
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(
      calls.find((call) => call.url === '/api/prompt-center/admin/')?.body
        .repository_source,
    ).toEqual(source),
  );
});

test('已有模板重新导入仓库后保存为携带来源的未发布副本，不覆盖原模板', async () => {
  render(
    <PromptTemplateEditor
      template={template}
      onCancel={() => {}}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  await userEvent.click(
    screen.getByRole('button', { name: 'Import GitHub repository' }),
  );
  await userEvent.type(
    screen.getByLabelText('Public GitHub repository URL'),
    source.url,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Import repository and edit' }),
  );
  await waitFor(() =>
    expect(screen.getByRole('checkbox', { name: 'Published' }).checked).toBe(
      false,
    ),
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(
      calls.find(
        (call) =>
          call.method === 'post' && call.url === '/api/prompt-center/admin/',
      )?.body,
    ).toMatchObject({ enabled: false, repository_source: source }),
  );
  expect(
    calls.find(
      (call) =>
        call.method === 'post' && call.url === '/api/prompt-center/admin/',
    ).body.version,
  ).toBeUndefined();
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('GitHub 模板中导入压缩包得到的副本不沿用原仓库来源', async () => {
  render(
    <PromptTemplateEditor
      template={template}
      onCancel={() => {}}
      onSaved={() => {}}
    />,
  );
  await screen.findByRole('checkbox', { name: 'Visible group' });
  expect(screen.getByRole('link', { name: 'GitHub source' })).toBeTruthy();
  await userEvent.upload(
    screen.getByLabelText('Upload skill archive'),
    new File(['zip'], 'skills.zip', { type: 'application/zip' }),
  );
  await waitFor(() =>
    expect(screen.queryByRole('link', { name: 'GitHub source' })).toBeNull(),
  );
  await userEvent.click(screen.getByRole('button', { name: 'Save template' }));
  await waitFor(() =>
    expect(
      calls.some(
        (call) =>
          call.method === 'post' && call.url === '/api/prompt-center/admin/',
      ),
    ).toBe(true),
  );
  expect(
    calls.find(
      (call) =>
        call.method === 'post' && call.url === '/api/prompt-center/admin/',
    ).body.repository_source,
  ).toBeUndefined();
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('管理员一键更新后显示新提交并保留卡片，历史模板无更新入口', async () => {
  const button = await openManagement();
  expect(
    screen.queryByRole('button', { name: 'Update Legacy guide from GitHub' }),
  ).toBeNull();
  expect(screen.getByText('aaaaaaaa')).toBeTruthy();
  await userEvent.click(button);
  await screen.findByText('Skills updated from GitHub');
  expect(screen.getByText('cccccccc')).toBeTruthy();
  expect(
    calls.find((call) => call.url.endsWith('/update-repository')).body,
  ).toEqual({ version: 3, overwrite_local: false });
  expect(screen.getByText('Review guide')).toBeTruthy();
});

test('仓库没有变更时明确提示已经最新', async () => {
  result = 'unchanged';
  await userEvent.click(await openManagement());
  await screen.findByText('Skills are already up to date');
  expect(screen.getByText('aaaaaaaa')).toBeTruthy();
});

test('本地修改先确认，取消不请求仓库，确认后携带覆盖标记', async () => {
  current.repository_source.locally_modified = true;
  await userEvent.click(await openManagement());
  let dialog = await screen.findByRole('dialog', {
    name: 'Overwrite local skill changes?',
  });
  expect(
    calls.filter((call) => call.url.endsWith('/update-repository')),
  ).toHaveLength(0);
  await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
  expect(
    calls.filter((call) => call.url.endsWith('/update-repository')),
  ).toHaveLength(0);
  await userEvent.click(
    screen.getByRole('button', { name: 'Update Review guide from GitHub' }),
  );
  dialog = await screen.findByRole('dialog', {
    name: 'Overwrite local skill changes?',
  });
  await userEvent.click(
    within(dialog).getByRole('button', { name: 'Update and overwrite' }),
  );
  await screen.findByText('Skills updated from GitHub');
  expect(
    calls.find((call) => call.url.endsWith('/update-repository')).body
      .overwrite_local,
  ).toBe(true);
});

test('更新中禁止重复提交，失败保留原来源且可以重试', async () => {
  let release;
  pendingUpdate = new Promise((resolve) => {
    release = resolve;
  });
  result = 'failure';
  const button = await openManagement();
  await userEvent.click(button);
  expect(button.disabled).toBe(true);
  await userEvent.click(button);
  expect(
    calls.filter((call) => call.url.endsWith('/update-repository')),
  ).toHaveLength(1);
  await act(async () => {
    release();
  });
  await screen.findByText('Repository unavailable');
  expect(screen.getByText('aaaaaaaa')).toBeTruthy();
  expect(button.disabled).toBe(false);
  result = 'updated';
  await userEvent.click(button);
  await screen.findByText('Skills updated from GitHub');
});

test('确认覆盖后锁定确认和取消按钮，不能重复发起覆盖更新', async () => {
  current.repository_source.locally_modified = true;
  let release;
  pendingUpdate = new Promise((resolve) => {
    release = resolve;
  });
  await userEvent.click(await openManagement());
  const dialog = await screen.findByRole('dialog', {
    name: 'Overwrite local skill changes?',
  });
  const confirm = within(dialog).getByRole('button', {
    name: 'Update and overwrite',
  });
  await userEvent.click(confirm);
  expect(confirm.disabled).toBe(true);
  expect(within(dialog).getByRole('button', { name: 'Cancel' }).disabled).toBe(
    true,
  );
  await userEvent.click(confirm);
  expect(
    calls.filter((call) => call.url.endsWith('/update-repository')),
  ).toHaveLength(1);
  await act(async () => {
    release();
  });
  await screen.findByText('Skills updated from GitHub');
});

test('服务端发现本地修改时允许确认重试，版本冲突不自动覆盖', async () => {
  result = 'local-changes';
  await userEvent.click(await openManagement());
  const dialog = await screen.findByRole('dialog', {
    name: 'Overwrite local skill changes?',
  });
  result = 'version-conflict';
  await userEvent.click(
    within(dialog).getByRole('button', { name: 'Update and overwrite' }),
  );
  await screen.findByText('Version conflict');
  expect(screen.getByText('aaaaaaaa')).toBeTruthy();
  expect(
    calls.filter((call) => call.url.endsWith('/update-repository')),
  ).toHaveLength(2);
});

test('普通用户看不到仓库来源或更新操作', async () => {
  localStorage.setItem('user', JSON.stringify({ id: 2, role: 1 }));
  render(<PromptCenter />);
  await screen.findByText('Review guide');
  expect(screen.queryByRole('link', { name: 'GitHub source' })).toBeNull();
  expect(
    screen.queryByRole('button', { name: 'Update Review guide from GitHub' }),
  ).toBeNull();
  expect(
    calls.some((call) => call.url.startsWith('/api/prompt-center/admin/')),
  ).toBe(false);
});

test('异常来源以纯文本显示，不能生成可执行链接', () => {
  render(
    <RepositorySource source={{ ...source, url: 'javascript:alert(1)' }} />,
  );
  expect(screen.queryByRole('link', { name: 'GitHub source' })).toBeNull();
  expect(screen.getByText('javascript:alert(1)')).toBeTruthy();
});
