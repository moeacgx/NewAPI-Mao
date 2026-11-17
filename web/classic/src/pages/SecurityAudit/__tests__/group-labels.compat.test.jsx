/*
Copyright (C) 2023-2026 QuantumNous

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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, test, vi } from 'vitest';
import { API } from '../../../helpers/api';
import BuiltinPolicyTab from '../BuiltinPolicyTab';
import PolicyTab from '../PolicyTab';
import RequestArchiveTab from '../RequestArchiveTab';
import EventsTab from '../EventsTab';

const groups = [
  { id: 7, code: 'stable-code', name: '当前名称' },
  { id: 8, code: 'unknown', name: '' },
];

const auditEvent = {
  id: 17,
  created_at: 1700000000,
  source: 'prompt_guard',
  stage: 'response',
  decision: 'flagged',
  channel_id: 3,
  token_group_mode: 'explicit',
  group_id: 7,
  group_code: 'legacy-alias',
  group_name: '历史名称',
  group_display_name: '当前路由名称',
  token_groups: [
    {
      id: 7,
      code: 'legacy-alias',
      name: '历史名称',
      display_name: '当前令牌名称',
    },
  ],
  channel_groups: [],
};

beforeEach(() => {
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    let data = {};
    if (url.endsWith('/groups')) data = groups;
    if (url.endsWith('/channels')) data = [];
    if (url.endsWith('/events')) data = { items: [auditEvent], total: 1 };
    if (url === '/api/group/details')
      return {
        data: {
          success: true,
          data: groups,
          group_names: { 'legacy-alias': '旧别名当前名称' },
        },
      };
    if (url.endsWith('/builtin-policy'))
      data = {
        config_version: 1,
        upstream_policy_enabled: true,
        upstream_policy_target_type: 'groups',
        upstream_policy_group_codes: ['deleted', 'legacy-alias'],
        cyber_policy_auto_ban_enabled: true,
        cyber_policy_auto_ban_exempt_group_codes: [],
      };
    if (url.endsWith('/request-archive/config'))
      data = {
        config_version: 1,
        enabled: false,
        archive_scope: 'audit_events',
        event_group_codes: ['deleted', 'legacy-alias'],
        retention_days: 7,
        worker_count: 1,
        queue_capacity: 100,
        targets: [],
      };
    return { data: { success: true, data } };
  });
  vi.spyOn(API, 'put').mockImplementation(async (_url, data) => ({
    data: { success: true, data },
  }));
});

test('事件列表显示当前路由和令牌名称而不拼标识，原事件不改写', async () => {
  localStorage.setItem(
    'classic-security-audit-event-columns',
    JSON.stringify({ token_groups: true, group_id: true }),
  );
  render(<EventsTab endpoints={[]} />);
  expect(await screen.findByText('当前路由名称')).toBeTruthy();
  expect(await screen.findByText('当前令牌名称')).toBeTruthy();
  expect(screen.queryByText(/当前路由名称.*#7/)).toBeNull();
  expect(auditEvent.group_name).toBe('历史名称');
  expect(auditEvent.token_groups[0].name).toBe('历史名称');
});

test('策略范围显示名称，选择和改名后继续使用稳定 ID', async () => {
  const onChange = vi.fn();
  const draft = {
    mode: 'off',
    all_groups: false,
    group_ids: [],
    endpoints: [],
  };
  const view = render(
    <PolicyTab draft={draft} groups={groups} onChange={onChange} />,
  );
  fireEvent.click(screen.getByText('选择需要审计的用户分组'));
  fireEvent.click(
    await screen.findByRole('option', { name: /^(?:tick )?当前名称$/ }),
  );
  expect(onChange).toHaveBeenLastCalledWith({ group_ids: [7] });
  view.rerender(
    <PolicyTab
      draft={{ ...draft, group_ids: [7] }}
      groups={[{ ...groups[0], name: '改名后的名称' }]}
      onChange={onChange}
    />,
  );
  expect(screen.getAllByText('改名后的名称').length).toBeGreaterThan(0);
  expect(screen.queryByText('当前名称 (stable-code)')).toBeNull();
});

test('内置策略范围和自动禁用白名单仅显示名称，失效项仍可辨认', async () => {
  render(<BuiltinPolicyTab />);
  fireEvent.click(await screen.findByText('失效分组: deleted'));
  await screen.findByRole('option', { name: /旧别名当前名称$/ });
  fireEvent.click(
    await screen.findByRole('option', { name: /^(?:tick )?当前名称$/ }),
  );
  fireEvent.click(screen.getByText('选择免于自动禁用的分组'));
  const exemptOption = await waitFor(() => {
    const options = screen.getAllByRole('option', {
      name: /^(?:tick )?当前名称$/,
    });
    expect(options.length).toBeGreaterThan(1);
    return options[options.length - 1];
  });
  fireEvent.click(exemptOption);
  expect(screen.queryByText('当前名称 (stable-code)')).toBeNull();
  expect(screen.getAllByText('当前名称').length).toBeGreaterThan(0);
  fireEvent.click(screen.getByRole('button', { name: '保存屏蔽词过滤设置' }));
  await waitFor(() =>
    expect(API.put).toHaveBeenCalledWith(
      '/api/security-audit/builtin-policy',
      expect.objectContaining({
        upstream_policy_group_codes: ['deleted', 'legacy-alias', 'stable-code'],
        cyber_policy_auto_ban_exempt_group_codes: ['stable-code'],
      }),
      expect.any(Object),
    ),
  );
});

test('归档范围按名称选择与筛选，保存保留 code 和未知项', async () => {
  render(<RequestArchiveTab />);
  fireEvent.click(await screen.findByText('Unavailable group: deleted'));
  await screen.findByRole('option', { name: /旧别名当前名称$/ });
  fireEvent.click(
    await screen.findByRole('option', { name: /^(?:tick )?当前名称$/ }),
  );
  fireEvent.keyDown(document.activeElement, { key: 'Escape' });
  fireEvent.click(screen.getByRole('button', { name: '保存更改' }));
  await waitFor(() =>
    expect(API.put).toHaveBeenCalledWith(
      '/api/security-audit/request-archive/config',
      expect.objectContaining({
        event_group_codes: ['deleted', 'legacy-alias', 'stable-code'],
      }),
      expect.any(Object),
    ),
  );
});

test('归档范围输入名称筛选选项，选中后仍保存稳定 code', async () => {
  render(<RequestArchiveTab />);
  fireEvent.click(await screen.findByText('Unavailable group: deleted'));
  await screen.findByRole('option', { name: /旧别名当前名称$/ });
  const input = await waitFor(() => {
    const element = screen
      .getByText('Event groups')
      .closest('label')
      ?.querySelector('.semi-select-filterable input');
    expect(element).toBeTruthy();
    return element;
  });
  fireEvent.change(input, { target: { value: '当前' } });
  await waitFor(() => {
    const names = screen
      .queryAllByRole('option')
      .map((option) => option.textContent);
    expect(names).toContain('当前名称');
    expect(names).not.toContain('unknown');
  });
  const filteredOption = screen
    .queryAllByRole('option')
    .find((option) => option.textContent === '当前名称');
  expect(filteredOption).toBeTruthy();
  fireEvent.click(filteredOption);
  fireEvent.click(screen.getByRole('button', { name: '保存更改' }));
  await waitFor(() =>
    expect(API.put).toHaveBeenCalledWith(
      '/api/security-audit/request-archive/config',
      expect.objectContaining({
        event_group_codes: ['deleted', 'legacy-alias', 'stable-code'],
      }),
      expect.any(Object),
    ),
  );
});

test('分组名称读取失败时归档范围降级为失效标识且保存不丢 code', async () => {
  const baseGet = API.get.getMockImplementation();
  API.get.mockImplementation(async (url, ...rest) => {
    if (url === '/api/group/details') throw new Error('names unavailable');
    return baseGet(url, ...rest);
  });
  render(<RequestArchiveTab />);
  fireEvent.click(await screen.findByText('Unavailable group: deleted'));
  await screen.findByRole('option', {
    name: /Unavailable group: legacy-alias$/,
  });
  expect(screen.queryByRole('option', { name: /旧别名当前名称$/ })).toBeNull();
  fireEvent.click(
    await screen.findByRole('option', { name: /^(?:tick )?当前名称$/ }),
  );
  fireEvent.click(screen.getByRole('button', { name: '保存更改' }));
  await waitFor(() =>
    expect(API.put).toHaveBeenCalledWith(
      '/api/security-audit/request-archive/config',
      expect.objectContaining({
        event_group_codes: ['deleted', 'legacy-alias', 'stable-code'],
      }),
      expect.any(Object),
    ),
  );
});
