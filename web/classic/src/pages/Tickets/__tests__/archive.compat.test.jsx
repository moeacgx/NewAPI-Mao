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
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TicketArchiveSettings from '../TicketArchiveSettings';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
let calls, failSave;
beforeEach(() => {
  calls = [];
  failSave = false;
  API.defaults.adapter = async (config) => {
    const body = config.data ? JSON.parse(config.data) : undefined;
    calls.push({ url: config.url, method: config.method, body });
    if (config.method === 'put' && failSave) throw new Error('Save failed');
    return {
      data: {
        success: true,
        data: body || {
          enabled: false,
          inactive_days: 30,
          statuses: ['resolved', 'closed'],
        },
      },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
});
afterEach(() => {
  API.defaults.adapter = originalAdapter;
});

test('超时归档默认关闭且保存天数和管理员显式选择的状态', async () => {
  render(<TicketArchiveSettings canWrite />);
  const days = await screen.findByLabelText('Inactive days');
  expect(days.value).toBe('30');
  expect(screen.getByLabelText('Enable automatic archive').checked).toBe(false);
  await userEvent.click(screen.getByLabelText('Enable automatic archive'));
  await userEvent.clear(days);
  await userEvent.type(days, '45');
  await userEvent.click(screen.getByLabelText('Open'));
  await userEvent.click(
    screen.getByRole('button', { name: 'Save archive settings' }),
  );
  await waitFor(() =>
    expect(calls.find((call) => call.method === 'put')).toMatchObject({
      url: '/api/ticket-admin/automation/auto-archive',
      body: {
        enabled: true,
        inactive_days: 45,
        statuses: ['resolved', 'closed', 'open'],
      },
    }),
  );
});

test('非法天数与空状态拒绝提交并保留配置表单', async () => {
  render(<TicketArchiveSettings canWrite />);
  const days = await screen.findByLabelText('Inactive days');
  await userEvent.clear(days);
  await userEvent.type(days, '1.5');
  await userEvent.click(
    screen.getByRole('button', { name: 'Save archive settings' }),
  );
  await screen.findByText('Enter a whole number of days from 1 to 3650.');
  expect(screen.getByLabelText('Inactive days').value).toBe('1.5');
  await userEvent.clear(days);
  await userEvent.type(days, '1');
  await userEvent.click(screen.getByLabelText('Resolved'));
  await userEvent.click(screen.getByLabelText('Closed'));
  await userEvent.click(
    screen.getByRole('button', { name: 'Save archive settings' }),
  );
  await screen.findByText('Select at least one status to archive.');
  expect(calls.some((call) => call.method === 'put')).toBe(false);
});

test('归档配置保存失败保留可编辑草稿并能重试保存', async () => {
  failSave = true;
  render(<TicketArchiveSettings canWrite />);
  const days = await screen.findByLabelText('Inactive days');
  await userEvent.clear(days);
  await userEvent.type(days, '90');
  await userEvent.click(
    screen.getByRole('button', { name: 'Save archive settings' }),
  );
  await screen.findByText('Save failed');
  expect(screen.getByLabelText('Inactive days').value).toBe('90');
  failSave = false;
  await userEvent.click(
    screen.getByRole('button', { name: 'Save archive settings' }),
  );
  await waitFor(() =>
    expect(calls.filter((call) => call.method === 'put')).toHaveLength(2),
  );
});

test('只读管理员不能修改归档配置', async () => {
  render(<TicketArchiveSettings canWrite={false} />);
  expect(await screen.findByLabelText('Inactive days')).toHaveProperty(
    'disabled',
    true,
  );
  expect(screen.getByLabelText('Enable automatic archive')).toHaveProperty(
    'disabled',
    true,
  );
  expect(screen.getByLabelText('Resolved').matches(':disabled')).toBe(true);
  await userEvent.click(screen.getByLabelText('Resolved'));
  expect(screen.getByLabelText('Resolved').checked).toBe(true);
  expect(
    screen.getByRole('button', { name: 'Save archive settings' }),
  ).toHaveProperty('disabled', true);
});

test('归档配置加载失败显示重试且不会展示可保存的伪默认配置', async () => {
  let fail = true;
  API.defaults.adapter = async (config) => {
    if (fail) throw new Error('Load failed');
    return {
      data: {
        success: true,
        data: {
          enabled: false,
          inactive_days: 30,
          statuses: ['resolved', 'closed'],
        },
      },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
  render(<TicketArchiveSettings canWrite />);
  await screen.findByText('Load failed');
  expect(
    screen.queryByRole('button', { name: 'Save archive settings' }),
  ).toBeNull();
  fail = false;
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
  expect(await screen.findByLabelText('Inactive days')).toHaveProperty(
    'value',
    '30',
  );
});

test('新账号归档配置不接受旧账号延迟读取或保存结果', async () => {
  let finishSave;
  let days = 30;
  API.defaults.adapter = async (config) => {
    if (config.method === 'put')
      return new Promise((resolve) => {
        finishSave = () =>
          resolve({
            data: {
              success: true,
              data: { enabled: true, inactive_days: 90, statuses: ['open'] },
            },
            status: 200,
            statusText: 'OK',
            headers: {},
            config,
          });
      });
    return {
      data: {
        success: true,
        data: {
          enabled: false,
          inactive_days: days,
          statuses: ['resolved', 'closed'],
        },
      },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
  const view = render(<TicketArchiveSettings key='first' canWrite />);
  await screen.findByLabelText('Inactive days');
  await userEvent.click(
    screen.getByRole('button', { name: 'Save archive settings' }),
  );
  await waitFor(() => expect(finishSave).toBeTypeOf('function'));
  days = 60;
  view.rerender(<TicketArchiveSettings key='second' canWrite />);
  await screen.findByLabelText('Inactive days');
  await act(() => finishSave());
  expect(screen.getByLabelText('Inactive days')).toHaveProperty('value', '60');
  expect(screen.getByLabelText('Enable automatic archive').checked).toBe(false);
  expect(screen.queryByText('Archive settings saved')).toBeNull();
});
