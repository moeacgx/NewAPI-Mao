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
import NotificationCenter from '../index';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const events = [
  {
    value: 'channel_disabled',
    label: 'Channel disabled',
    default_template: '{{mention}} Channel {{channel_name}} disabled',
    variables: ['mention', 'channel_name'],
  },
  {
    value: 'ticket_user_message',
    label: 'Ticket created or user replied',
    description:
      'Triggered when a user creates a support ticket or adds a reply.',
    default_template: '{{mention}} {{action_label}} #{{ticket_id}}: {{title}}',
    variables: [
      'mention',
      'ticket_id',
      'title',
      'user_id',
      'category',
      'priority',
      'status',
      'actor_id',
      'actor_role',
      'create_time',
      'action',
      'action_label',
    ],
  },
];
let saved;
beforeEach(() => {
  saved = null;
  localStorage.setItem('user', JSON.stringify({ id: 1, role: 100 }));
  API.defaults.adapter = async (config) => {
    let data = [];
    if (config.url.endsWith('/event-types')) data = events;
    else if (config.url.endsWith('/bots'))
      data = [{ id: 1, name: 'Support bot', enabled: true }];
    else if (config.url.endsWith('/tasks'))
      data = [
        {
          id: 2,
          name: 'Support notifications',
          event_type: events[0].value,
          event_name: events[0].label,
          bot_id: 1,
          template: events[0].default_template,
          enabled: true,
          targets: [{ id: 3, chat_id: '-1001', enabled: true }],
        },
      ];
    else if (config.method === 'put') saved = JSON.parse(config.data);
    return {
      data: { success: true, data },
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

test('通知中心只提供合并后的工单事件，切换使用其默认模板且保存原有目标', async () => {
  render(<NotificationCenter />);
  await screen.findByText('Support notifications');
  await userEvent.click(screen.getByRole('button', { name: '编辑' }));
  await screen.findByText('{{channel_name}}');
  await userEvent.click(screen.getByRole('combobox', { name: '事件类型' }));
  expect(screen.queryByText('Ticket administrator replied')).toBeNull();
  expect(screen.queryByText('Ticket status changed')).toBeNull();
  await userEvent.click(
    screen.getByRole('option', { name: /Ticket created or user replied$/ }),
  );
  // jsdom 不执行 CSS 动画，显式完成下拉框关闭动画。
  fireEvent.animationEnd(screen.getByRole('listbox'));
  await screen.findByDisplayValue(events[1].default_template);
  expect(await screen.findByText('{{action_label}}')).toBeTruthy();
  expect(
    screen.getByText('Ticket actor role (user, admin or system)'),
  ).toBeTruthy();
  expect(
    screen.getByText(
      'Ticket notifications include the title, category, priority, status, requester and actor IDs. Message content is excluded to protect sensitive information.',
    ),
  ).toBeTruthy();
  expect(
    screen.getByText(
      'Triggered when a user creates a support ticket or adds a reply.',
    ),
  ).toBeTruthy();
  await userEvent.click(screen.getByRole('button', { name: 'confirm' }));
  await waitFor(() =>
    expect(saved).toMatchObject({
      event_type: 'ticket_user_message',
      bot_id: 1,
      template: events[1].default_template,
      targets: [{ id: 3, chat_id: '-1001', enabled: true }],
    }),
  );
  expect(saved).not.toHaveProperty('filter_config');
});
