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
import { expect, test } from 'vitest';
import { act, render, screen } from '@testing-library/react';
import { createInstance } from 'i18next';
import { I18nextProvider } from 'react-i18next';
import CreateTicketModal from '../CreateTicketModal';
import en from '../../../i18n/locales/en.json';
import zhCN from '../../../i18n/locales/zh-CN.json';
import zhTW from '../../../i18n/locales/zh-TW.json';
import fr from '../../../i18n/locales/fr.json';
import ja from '../../../i18n/locales/ja.json';
import ru from '../../../i18n/locales/ru.json';
import vi from '../../../i18n/locales/vi.json';

const resources = { en, 'zh-CN': zhCN, 'zh-TW': zhTW, fr, ja, ru, vi };
test('创建弹窗切换语言后更新表单可访问名称和提交按钮', async () => {
  const i18n = createInstance();
  await i18n.init({
    resources,
    lng: 'en',
    fallbackLng: 'en',
    interpolation: { escapeValue: false },
  });
  render(
    <I18nextProvider i18n={i18n}>
      <CreateTicketModal visible onClose={() => {}} />
    </I18nextProvider>,
  );
  expect(screen.getByLabelText('Ticket title')).toBeTruthy();
  await act(() => i18n.changeLanguage('zh-CN'));
  expect(screen.getByLabelText('工单标题')).toBeTruthy();
  expect(screen.getByRole('button', { name: '提交工单' })).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Submit ticket' })).toBeNull();
});

test('七语均覆盖工单通知事件、权限和关键服务错误且保留计数变量', () => {
  const keys = [
    'Pending tickets',
    'Pending ticket reminder',
    'Dismiss ticket reminder',
    'Tickets awaiting your response: {{total}}',
    'View pending tickets',
    'Pending ticket count',
    'Failed to load pending tickets',
    'No tickets awaiting your response',
    'New tickets and customer follow-ups',
    'Ticket created or user replied',
    'Triggered when a user creates a support ticket or adds a reply.',
    'Ticket Management',
    'Ticket settings',
    'Read all tickets',
    'Manage tickets',
    'Ticket not found',
    'Ticket changed, please refresh',
    'Ticket request exceeds 64 KiB',
    'Ticket automation',
    'Automatically reply once after a user creates a ticket.',
    'Enable automatic first reply',
    'Automatic reply content',
    'Save automation settings',
    'Automation settings saved',
    'Failed to load automation settings',
    'Failed to save automation settings',
    'Enter an automatic reply before enabling this rule.',
    'You have read-only access to ticket automation.',
    'Archived',
    'Select all tickets on this page',
    'Select ticket #{{id}}',
    '{{count}} selected',
    'Batch status',
    'Change selected status',
    'Delete selected tickets',
    'Confirm status change',
    'Confirm deletion',
    'Automatic ticket archive',
    'Enable automatic archive',
    'Inactive days',
    'Statuses to archive',
    'Save archive settings',
    'Enter a whole number of days from 1 to 3650.',
    'Select at least one status to archive.',
    'Reply to support tickets, update their status, delete tickets, and manage automation.',
    'Ticket actor role (user, admin or system)',
  ];
  for (const [locale, resource] of Object.entries(resources)) {
    for (const key of keys) {
      expect(resource.translation[key], `${locale}: ${key}`).toBeTruthy();
      if (locale !== 'en') expect(resource.translation[key]).not.toBe(key);
    }
    expect(resource.translation['{{count}} tickets']).toContain('{{count}}');
    expect(
      resource.translation['Tickets awaiting your response: {{total}}'],
    ).toContain('{{total}}');
    expect(resource.translation['{{count}} selected']).toContain('{{count}}');
    expect(resource.translation['Select ticket #{{id}}']).toContain('{{id}}');
  }
});
