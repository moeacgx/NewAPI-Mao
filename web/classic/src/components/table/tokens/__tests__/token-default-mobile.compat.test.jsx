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
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import i18n from 'i18next';
import { beforeEach, expect, test, vi } from 'vitest';
import TokensTable from '../TokensTable';

const record = {
  id: 83,
  name: 'Codex-Enterprise | 企业级 稳定不限客户端 | 超长令牌名称验收',
  status: 1,
  key: 'demo***123',
  group: 'internal-enterprise',
  group_details: [{ id: 9, code: 'internal-enterprise', name: '企业正式分组' }],
  unlimited_quota: false,
  remain_quota: 1500000,
  used_quota: 1000000,
  model_limits_enabled: true,
  model_limits: 'gpt-4.1,claude-sonnet-4',
  allow_ips: '192.0.2.10\n198.51.100.20',
  created_time: 1,
  accessed_time: 2,
  expired_time: -1,
};

beforeEach(() => {
  localStorage.setItem('quota_per_unit', '500000');
  localStorage.setItem('quota_display_type', 'USD');
  vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
    matches: query === '(max-width: 767px)',
    media: query,
    addEventListener() {},
    removeEventListener() {},
    addListener() {},
    removeListener() {},
  }));
});

function renderMobileTokens(tokens = [record]) {
  return render(
    <TokensTable
      tokens={tokens}
      showKeys={{}}
      resolvedTokenKeys={{}}
      loadingTokenKeys={{}}
      groupRatios={{ 'internal-enterprise': 0.9 }}
      t={i18n.t.bind(i18n)}
    />,
  );
}

test('移动端完整展示长名称、当前分组名称和剩余及已用额度', () => {
  renderMobileTokens();
  const card = within(screen.getByRole('article', { name: record.name }));

  expect(card.getByRole('heading', { name: record.name }).textContent).toBe(
    record.name,
  );
  expect(card.getByText('已启用')).toBeTruthy();
  expect(card.getByText('企业正式分组')).toBeTruthy();
  expect(screen.queryByText(record.group)).toBeNull();
  expect(card.getByText('0.9x')).toBeTruthy();
  const quota = within(card.getByRole('button', { name: '已用额度 2' }));
  expect(quota.getByText('剩余额度 ($)')).toBeTruthy();
  expect(quota.getByText('3', { exact: true })).toBeTruthy();
  expect(quota.getByText('已用额度', { exact: true })).toBeTruthy();
  expect(quota.getByText('2', { exact: true })).toBeTruthy();
  expect(card.getByText('永不过期')).toBeTruthy();
  expect(screen.queryByRole('table')).toBeNull();
});

test('移动端点击模型与 IP 数量可查看完整限制', async () => {
  const user = userEvent.setup();
  renderMobileTokens();
  const card = within(screen.getByRole('article', { name: record.name }));

  expect(screen.queryByText('gpt-4.1')).toBeNull();
  expect(screen.queryByText('192.0.2.10')).toBeNull();
  await user.click(card.getByText('2 models'));
  expect(await screen.findByText('gpt-4.1')).toBeTruthy();
  expect(screen.getByText('claude-sonnet-4')).toBeTruthy();
  await user.click(card.getByText('2 IP(s)'));
  expect(await screen.findByText('192.0.2.10')).toBeTruthy();
  expect(screen.getByText('198.51.100.20')).toBeTruthy();
});

test('移动端空列表显示搜索无结果且不残留令牌卡片', () => {
  renderMobileTokens([]);

  expect(screen.getByRole('status').textContent).toBe('搜索无结果');
  expect(screen.queryByRole('article')).toBeNull();
  expect(screen.queryByRole('table')).toBeNull();
});
