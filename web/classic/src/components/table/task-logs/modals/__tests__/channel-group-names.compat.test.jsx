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
import { render, screen } from '@testing-library/react';
import { beforeEach, expect, test } from 'vitest';
import ChannelInfoModal from '../ChannelInfoModal';

beforeEach(() => {
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: window.localStorage,
  });
});

const base = {
  id: 42,
  name: '渠道',
  type: 1,
  group: 'legacy,deleted',
  group_ids: [7, 99],
  group_details: [{ id: 7, code: '7', name: '当前名称' }],
  models: [],
};
function modal(data) {
  return (
    <ChannelInfoModal
      showChannelInfo
      setShowChannelInfoModal={() => {}}
      channelInfoData={data}
      t={(key) => key}
    />
  );
}
test('稳定 ID 与旧 alias 展示当前名称并支持改名', () => {
  const view = render(modal(base));
  expect(screen.getByText('当前名称')).toBeTruthy();
  expect(screen.getByText('deleted')).toBeTruthy();
  expect(screen.queryByText('legacy')).toBeNull();
  view.rerender(
    modal({ ...base, group_details: [{ id: 7, code: '7', name: '改名后' }] }),
  );
  expect(screen.getByText('改名后')).toBeTruthy();
  expect(screen.queryByText('当前名称')).toBeNull();
});
test('无 ID 时按 code 展示且缺失名称回退', () => {
  render(
    modal({
      ...base,
      group_ids: [],
      groups: ['7', 'empty', 'deleted'],
      group_details: [
        { id: 7, code: '7', name: '当前名称' },
        { id: 8, code: 'empty', name: '  ' },
      ],
    }),
  );
  expect(screen.getByText('当前名称')).toBeTruthy();
  expect(screen.getByText('empty')).toBeTruthy();
  expect(screen.getByText('deleted')).toBeTruthy();
});
test('历史响应保留标识', () => {
  render(
    modal({
      ...base,
      group: 'legacy',
      group_ids: undefined,
      group_details: undefined,
    }),
  );
  expect(screen.getByText('legacy')).toBeTruthy();
});
test('引用乱序时按稳定 ID 展示且不泄露旧标识到标题', () => {
  render(
    modal({
      ...base,
      group: 'reused,old',
      group_ids: [8, 7],
      group_details: [
        { id: 7, code: 'reused', name: '第一组当前名称' },
        { id: 8, code: '8', name: '第二组当前名称' },
      ],
    }),
  );
  const tags = screen.getAllByText(/第[一二]组当前名称/);
  expect(tags.map((tag) => tag.textContent)).toEqual([
    '第二组当前名称',
    '第一组当前名称',
  ]);
  expect(
    document.querySelector(
      '[title="reused"], [aria-label="reused"], [title="old"], [aria-label="old"]',
    ),
  ).toBeNull();
});
