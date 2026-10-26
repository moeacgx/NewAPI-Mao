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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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

function renderCells(other, record = {}) {
  const columns = getLogsColumns({
    t: (key, options) => i18next.t(key, options),
    COLUMN_KEYS: { TYPE: 'type', GROUP: 'group' },
    copyText: () => Promise.resolve(),
    showUserInfoFunc: () => {},
    openChannelAffinityUsageCacheModal: () => {},
    isAdminUser: false,
  });
  const typeColumn = columns.find((column) => column.key === 'type');
  const groupColumn = columns.find((column) => column.key === 'group');
  const row = {
    id: 1,
    type: 2,
    group: 'default',
    other: JSON.stringify(other),
    ...record,
  };
  render(
    <>
      <div aria-label='日志类型'>{typeColumn.render(row.type, row)}</div>
      <div aria-label='日志分组'>{groupColumn.render(row.group, row)}</div>
    </>,
  );
}

test('分组右上角用图标显示技能，悬停列出去重数量与名称，类型列保持紧凑', async () => {
  renderCells(onDemandBundleOther);

  expect(screen.getByText('消费')).toBeTruthy();
  expect(screen.getByLabelText('日志类型').textContent).toBe('消费');
  expect(screen.getByLabelText('日志分组').textContent).not.toContain('技能');
  const tag = screen.getByRole('button', { name: /技能 2/ });
  await userEvent.setup().hover(tag);
  expect(await screen.findByText('技能 2')).toBeTruthy();
  expect(
    await screen.findByText('按需读取：Go Testing, SQL Style'),
  ).toBeTruthy();
});

test('分组角标在最终生成未读取技能时提示未用技能及可用数量', async () => {
  renderCells({
    gateway_skills: { stage: 'generation', round: 1, catalogue_count: 5 },
  });

  await userEvent
    .setup()
    .hover(screen.getByRole('button', { name: /未用技能/ }));
  expect(await screen.findByText('可用 5 个技能，本次未读取')).toBeTruthy();
});

test('分组角标的私有发现提示包含轮次、已读技能和单独计费说明', async () => {
  renderCells({
    gateway_skills: {
      stage: 'discovery',
      round: 2,
      catalogue_count: 5,
      files: [{ template_id: 12, skill_id: 'go-testing', title: 'Go Testing' }],
    },
  });

  await userEvent
    .setup()
    .hover(screen.getByRole('button', { name: /技能发现 · 第 2 轮/ }));
  expect(await screen.findByText('私有技能发现轮次，单独计费')).toBeTruthy();
  expect(screen.getByText('已读技能：Go Testing')).toBeTruthy();
});

test('无技能数据的日志保持原分组和类型，无技能角标', () => {
  renderCells({ ws: true });

  expect(screen.getByText('消费')).toBeTruthy();
  expect(screen.queryByText(/技能/)).toBeNull();
  expect(screen.queryByRole('button', { name: /技能/ })).toBeNull();
  expect(screen.getByText('default')).toBeTruthy();
});

test.each([
  [{ group_name: '显示名称', group: 'raw' }, {}, '显示名称'],
  [{ group: '' }, { group: 'legacy' }, 'legacy'],
  [{ group: '' }, { group: '' }, '用户分组'],
  [{ group: undefined }, {}, '—'],
])('保留分组来源优先级与历史回退 %#', (record, other, label) => {
  renderCells({ ...onDemandBundleOther, ...other }, record);
  expect(screen.getByText(label)).toBeTruthy();
  expect(screen.getAllByRole('button', { name: /技能 2/ })).toHaveLength(1);
});

test('多分组只显示一个技能角标', () => {
  renderCells(onDemandBundleOther, { group: 'pro,vip' });
  expect(screen.getByText('pro')).toBeTruthy();
  expect(screen.getByText('vip')).toBeTruthy();
  expect(screen.getAllByRole('button', { name: /技能/ })).toHaveLength(1);
});

test('键盘聚焦可以查看详情并用 Escape 关闭', async () => {
  renderCells(onDemandBundleOther);
  const user = userEvent.setup();
  await user.tab();
  await user.tab();
  expect(document.activeElement).toBe(
    screen.getByRole('button', { name: /技能 2/ }),
  );
  expect(
    await screen.findByText('按需读取：Go Testing, SQL Style'),
  ).toBeTruthy();
  await user.keyboard('{Escape}');
  // JSDOM 不播放 CSS 退场动画，补发浏览器动画结束事件。
  fireEvent.animationEnd(screen.getByRole('tooltip'));
  await waitFor(() =>
    expect(screen.queryByText('按需读取：Go Testing, SQL Style')).toBeNull(),
  );
});

test('点击技能角标不会触发行展开或复制分组', async () => {
  let clicks = 0;
  const columns = getLogsColumns({
    t: (key, options) => i18next.t(key, options),
    COLUMN_KEYS: { GROUP: 'group' },
    isAdminUser: false,
  });
  const group = columns.find((column) => column.key === 'group');
  render(
    <div onClick={() => clicks++}>
      {group.render('default', {
        type: 2,
        group: 'default',
        other: JSON.stringify(onDemandBundleOther),
      })}
    </div>,
  );
  await userEvent.click(screen.getByRole('button', { name: /技能 2/ }));
  expect(clicks).toBe(0);
  expect(screen.queryByText(/已复制/)).toBeNull();
});

test('历史日志 other 损坏时仍展示分组且不渲染技能徽标', () => {
  renderCells(null, { other: '{invalid', group: 'legacy' });
  expect(screen.getByText('legacy')).toBeTruthy();
  expect(screen.queryByRole('button', { name: /技能/ })).toBeNull();
});
