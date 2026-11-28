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
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, test, vi } from 'vitest';
import { API } from '../../../helpers';
import MySupplierSubmissions from '../MySupplierSubmissions';
import SupplierApplicationManagementTable from '../SupplierApplicationManagementTable';

const quiet = {
  id: 9,
  site_name: 'Quiet Site',
  site_url: 'https://quiet.example.com',
  contact_type: 'wechat',
  contact: 'wx-quiet',
  remark: '',
  status: 'pending',
  created_at: 1792634400,
  message_count: 0,
  messages: [],
  groups: [],
  admin_note: 'Internal: price is high',
  user_id: 7,
  username: 'supplier-user',
};

const busy = {
  ...quiet,
  id: 11,
  site_name: 'Busy Site',
  contact: 'wx-busy',
  message_count: 2,
  admin_note: 'Internal: talked already',
  user_id: 8,
  username: 'busy-user',
};

function page(items) {
  return {
    data: {
      success: true,
      data: { page: 1, page_size: 20, total: items.length, items },
    },
  };
}

function expectComposerDock(root) {
  const scroll = root.querySelector('[data-supplier-message-scroll]');
  const composer = root.querySelector('[data-supplier-composer]');
  expect(scroll).toBeTruthy();
  expect(composer).toBeTruthy();
  expect(scroll.contains(composer)).toBe(false);
  expect(scroll.parentElement).toBe(composer.parentElement);
  expect(scroll.parentElement.style.display).toBe('flex');
  expect(scroll.parentElement.style.flexDirection).toBe('column');
  expect(scroll.style.overflowY).toBe('auto');
  expect(scroll.style.flex.startsWith('1 1')).toBe(true);
  expect(scroll.style.minHeight).toBe('8rem');
  expect(parseFloat(scroll.style.paddingBottom)).toBeGreaterThan(0);
  expect(composer.style.flexShrink).toBe('0');
  expect(
    scroll.compareDocumentPosition(composer) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  const facts = root.querySelector('[data-supplier-detail-facts]');
  expect(facts).toBeTruthy();
  expect(facts.contains(composer)).toBe(false);
  expect(facts.style.overflowY).toBe('auto');
  expect(facts.style.minHeight === '0' || facts.style.minHeight === '0px').toBe(
    true,
  );
  expect(
    facts.compareDocumentPosition(composer) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  const detail = root.querySelector('[data-supplier-detail]');
  expect(detail.style.display).toBe('flex');
  expect(detail.style.flexDirection).toBe('column');
  expect(detail.style.overflow).toBe('hidden');
}

beforeEach(() => {
  localStorage.setItem(
    'user',
    JSON.stringify({
      id: 1,
      username: 'root',
      role: 100,
    }),
  );
});

test('我的申请只在有对话时显示消息数量，详情没有内部备注', async () => {
  const user = userEvent.setup();
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (String(url).includes('/messages')) throw new Error(`unexpected ${url}`);
    return page([quiet, busy]);
  });
  render(<MySupplierSubmissions />);
  expect(await screen.findByRole('img', { name: '2 messages' })).toBeTruthy();
  expect(screen.getByText('Quiet Site')).toBeTruthy();
  expect(screen.queryByRole('img', { name: '0 messages' })).toBeNull();
  expect(screen.queryByText('Internal: price is high')).toBeNull();
  expect(API.get).toHaveBeenCalledTimes(1);
  const row = screen.getByText('Quiet Site').closest('tr');
  await user.click(within(row).getByRole('button', { name: /View details/ }));
  expect(screen.queryByText('Internal note')).toBeNull();
  expect(screen.queryByText('Processing')).toBeNull();
  const dialog = await screen.findByRole('dialog');
  expectComposerDock(dialog);
  expect(
    within(dialog).getByRole('button', { name: 'Send message' }),
  ).toBeTruthy();
  expect(API.get).toHaveBeenCalledTimes(1);
});

test('管理列表显示消息数量，处理块在对话之前', async () => {
  const user = userEvent.setup();
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    const path = String(url);
    if (path.includes('/messages')) throw new Error(`unexpected ${path}`);
    if (path.endsWith('/11')) return { data: { success: true, data: busy } };
    return page([quiet, busy]);
  });
  render(<SupplierApplicationManagementTable />);
  expect(await screen.findByRole('img', { name: '2 messages' })).toBeTruthy();
  expect(screen.getByText('Quiet Site')).toBeTruthy();
  expect(screen.queryByRole('img', { name: '0 messages' })).toBeNull();
  await waitFor(() => expect(API.get).toHaveBeenCalledTimes(1));
  const row = screen.getByText('Busy Site').closest('tr');
  await user.click(within(row).getByRole('button', { name: /View details/ }));
  const dialog = await screen.findByRole('dialog');
  const processing = within(dialog).getByText('Processing');
  const conversation = within(dialog).getByText('Conversation');
  expect(
    processing.compareDocumentPosition(conversation) &
      Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  expect(
    processing.compareDocumentPosition(
      within(dialog).getByText('Site information'),
    ) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  expect(
    processing.compareDocumentPosition(
      within(dialog).getByText('Group quotes'),
    ) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  expect(
    processing.compareDocumentPosition(within(dialog).getByText(/^User:/)) &
      Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  expect(within(dialog).queryByText('Internal note')).toBeNull();
  const processingBlock = processing.closest('[data-supplier-processing]');
  expect(processingBlock).toBeTruthy();
  expect(within(processingBlock).getByText('Pending')).toBeTruthy();
  expect(
    within(processingBlock).queryByRole('button', { name: 'Save' }),
  ).toBeNull();
  await user.click(
    within(processingBlock).getByRole('button', { name: 'Edit' }),
  );
  expect(within(processingBlock).getByText('Internal note')).toBeTruthy();
  expect(
    within(processingBlock).getByRole('button', { name: 'Save' }),
  ).toBeTruthy();
  expect(
    processingBlock.compareDocumentPosition(
      within(dialog).getByText('Conversation'),
    ) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  expectComposerDock(dialog);
  expect(
    API.get.mock.calls.some((call) => String(call[0]).includes('/messages')),
  ).toBe(false);
});
