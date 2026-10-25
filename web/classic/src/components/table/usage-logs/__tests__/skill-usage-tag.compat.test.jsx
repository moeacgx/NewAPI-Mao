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
import i18next from 'i18next';
import { expect, test } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { getLogsColumns } from '../UsageLogsColumnDefs';

const onDemandBundleOther = {
  gateway_skills: {
    stage: 'generation',
    round: 1,
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
};

function renderTypeCell(other) {
  const columns = getLogsColumns({
    t: (key, options) => i18next.t(key, options),
    COLUMN_KEYS: { TYPE: 'type' },
    copyText: () => Promise.resolve(),
    showUserInfoFunc: () => {},
    openChannelAffinityUsageCacheModal: () => {},
    isAdminUser: false,
  });
  const typeColumn = columns.find((column) => column.key === 'type');
  render(
    <>
      {typeColumn.render(2, { id: 1, type: 2, other: JSON.stringify(other) })}
    </>,
  );
}

test('类型列显示按需读取的去重技能数，悬停列出两个技能名称', async () => {
  renderTypeCell(onDemandBundleOther);

  expect(screen.getByText('消费')).toBeTruthy();
  const tag = screen.getByText('技能 2');
  await userEvent.setup().hover(tag);
  expect(
    await screen.findByText('按需读取：Go Testing, SQL Style'),
  ).toBeTruthy();
});

test('类型列在最终生成未读取技能时显示未用技能及可用数量', async () => {
  renderTypeCell({
    gateway_skills: { stage: 'generation', round: 1, catalogue_count: 5 },
  });

  await userEvent.setup().hover(screen.getByText('未用技能'));
  expect(await screen.findByText('可用 5 个技能，本次未读取')).toBeTruthy();
});

test('类型列为私有技能发现轮次显示轮次并说明单独计费', async () => {
  renderTypeCell({
    gateway_skills: {
      stage: 'discovery',
      round: 2,
      catalogue_count: 5,
      files: [{ template_id: 12, skill_id: 'go-testing', title: 'Go Testing' }],
    },
  });

  await userEvent.setup().hover(screen.getByText('技能发现 · 第 2 轮'));
  expect(await screen.findByText('私有技能发现轮次，单独计费')).toBeTruthy();
  expect(screen.getByText('已读技能：Go Testing')).toBeTruthy();
});

test('类型列对无技能数据的日志只显示类型标签', () => {
  renderTypeCell({ ws: true });

  expect(screen.getByText('消费')).toBeTruthy();
  expect(screen.queryByText(/技能/)).toBeNull();
});
