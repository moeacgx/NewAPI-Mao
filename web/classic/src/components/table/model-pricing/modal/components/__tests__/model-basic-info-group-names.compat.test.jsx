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
import { render } from '@testing-library/react';
import { expect, test } from 'vitest';
import ModelBasicInfo from '../ModelBasicInfo';

const modelData = {
  model_name: 'fixture-model',
  quota_type: 0,
  enable_groups: ['group_4', 'CC-KIRO', 'shared-a', 'shared-b', 'deleted'],
};

const groupNames = {
  group_4: '当前名称',
  'CC-KIRO': '历史别名对应名称',
  'shared-a': '同名展示',
  'shared-b': '同名展示',
};

const renderPills = (names) => {
  const { container } = render(
    <ModelBasicInfo modelData={modelData} groupNames={names} t={(key) => key} />,
  );
  return Array.from(
    container.querySelectorAll('.classic-pricing-detail-pill-list > span'),
  ).map((node) => node.textContent);
};

test('Classic 模型详情分组标签显示当前名称与历史别名名称，保持顺序并回退未解析标识', () => {
  const pills = renderPills(groupNames);
  expect(pills).toEqual([
    '当前名称',
    '历史别名对应名称',
    '同名展示',
    '同名展示',
    'deleted',
  ]);
  expect(pills).not.toContain('group_4');
  expect(pills).not.toContain('CC-KIRO');
});

test('Classic 模型详情分组标签随改名映射更新，移除映射后回退标识', () => {
  expect(renderPills({ ...groupNames, group_4: '改名后' })[0]).toBe('改名后');
  expect(renderPills({})[0]).toBe('group_4');
});
