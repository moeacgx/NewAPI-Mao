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
import { Descriptions } from '@douyinfe/semi-ui';
import { render, renderHook, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';

import { API } from '../../../../helpers';
import { useLogsData } from '../../../../hooks/usage-logs/useUsageLogsData';

vi.mock('../../../../helpers', async (importOriginal) => ({
  ...(await importOriginal()),
  API: { get: vi.fn() },
}));

async function renderExpandedDetails(other) {
  const log = {
    id: 1,
    created_at: 1761350400,
    type: 2,
    content: '',
    model_name: 'claude-sonnet',
    quota: 1,
    prompt_tokens: 1,
    completion_tokens: 1,
    other: JSON.stringify(other),
  };
  API.get.mockResolvedValue({
    data: {
      success: true,
      data: { items: [log], page: 1, page_size: 10, total: 1 },
    },
  });
  const { result } = renderHook(() => useLogsData());
  await waitFor(() => expect(result.current.logs).toHaveLength(1));
  render(<Descriptions data={result.current.expandData[log.id]} />);
}

test('展开详情按装载方式列出本次请求使用的技能', async () => {
  await renderExpandedDetails({
    prompt_templates: [{ id: 7, title: 'Code Review Rules', version: 3 }],
    gateway_skills: {
      stage: 'generation',
      round: 2,
      catalogue_count: 5,
      files: [
        { template_id: 12, skill_id: 'go-testing', title: 'Go Testing' },
        {
          template_id: 12,
          skill_id: 'go-testing',
          title: 'Go Testing',
          path: 'references/table-tests.md',
        },
        { template_id: 12, skill_id: 'sql-style', title: 'SQL Style' },
      ],
    },
  });

  expect(screen.getByText('技能')).toBeTruthy();
  expect(screen.getByText('技能 3')).toBeTruthy();
  expect(screen.getByText('始终装载：Code Review Rules')).toBeTruthy();
  expect(screen.getByText('按需读取：Go Testing, SQL Style')).toBeTruthy();
});

test('展开详情说明私有技能发现轮次单独计费', async () => {
  await renderExpandedDetails({
    gateway_skills: { stage: 'discovery', round: 1, catalogue_count: 5 },
  });

  expect(screen.getByText('技能发现 · 第 1 轮')).toBeTruthy();
  expect(screen.getByText('私有技能发现轮次，单独计费')).toBeTruthy();
});

test('展开详情对无技能数据的日志不增加技能行', async () => {
  await renderExpandedDetails({ request_path: '/v1/messages' });

  expect(screen.getByText('/v1/messages')).toBeTruthy();
  expect(screen.queryByText('技能')).toBeNull();
});
