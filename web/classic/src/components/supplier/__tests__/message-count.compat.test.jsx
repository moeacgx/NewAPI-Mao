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
  message_count: 4,
  unread_count: 0,
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
  unread_count: 2,
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
  const panel = root.querySelector('[data-supplier-conversation-dialog]');
  expect(panel).toBeTruthy();
  expect(panel.style.width === '56rem' || panel.style.width === '100%').toBe(
    true,
  );
  expect(panel.style.maxWidth).toBe('100vw');
  expect(panel.style.maxHeight).toBe('100dvh');
  expect(panel.style.overflow).toBe('hidden');
  expect(panel.style.minHeight === '0' || panel.style.minHeight === '0px').toBe(
    true,
  );
  const slot = root.querySelector('[data-supplier-conversation-slot]');
  expect(slot.style.flex.startsWith('1 1')).toBe(true);
  expect(slot.style.overflow).toBe('hidden');
  expect(slot.style.minHeight === '0' || slot.style.minHeight === '0px').toBe(
    true,
  );
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
  vi.spyOn(API, 'post').mockResolvedValue({
    data: { success: true, data: { unread_count: 0 } },
  });
});

test('我的申请只在有对话时显示消息数量，详情没有内部备注', async () => {
  const user = userEvent.setup();
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (String(url).includes('/messages')) throw new Error(`unexpected ${url}`);
    return page([quiet, busy]);
  });
  render(<MySupplierSubmissions />);
  const busyButton = await screen.findByRole('button', {
    name: 'Conversation, 2 unread',
  });
  expect(busyButton.querySelector('svg')).toBeTruthy();
  expect(busyButton.textContent).toContain('2');
  expect(busyButton.textContent).not.toContain('4');
  const quietRow = screen.getByText('Quiet Site').closest('tr');
  const quietButton = within(quietRow).getByRole('button', { name: 'Conversation' });
  expect(quietButton.querySelector('svg')).toBeTruthy();
  expect(quietButton.textContent ?? '').not.toMatch(/\d/);
  expect(screen.queryByRole('button', { name: /0 unread/ })).toBeNull();
  expect(screen.queryByRole('button', { name: /4 unread/ })).toBeNull();
  expect(screen.queryByText('Internal: price is high')).toBeNull();
  expect(API.get).toHaveBeenCalledTimes(1);
  const row = screen.getByText('Quiet Site').closest('tr');
  await user.click(within(row).getByRole('button', { name: /View details/ }));
  expect(screen.queryByText('Internal note')).toBeNull();
  expect(screen.queryByText('Processing')).toBeNull();
  const dialog = await screen.findByRole('dialog');
  expect(dialog.querySelector('[data-supplier-conversation]')).toBeNull();
  expect(dialog.querySelector('[data-supplier-composer]')).toBeNull();
  expect(API.post).not.toHaveBeenCalled();
  await user.click(within(dialog).getByRole('button', { name: 'Conversation' }));
  const conversation = screen.getByRole('dialog', {
    name: /Conversation with Quiet Site/,
  });
  expect(within(conversation).getByText('No messages yet')).toBeTruthy();
  expectComposerDock(conversation);
  expect(
    within(conversation).getByRole('button', { name: 'Send message' }),
  ).toBeTruthy();
  expect(API.get).toHaveBeenCalledTimes(1);
  expect(
    API.post.mock.calls.some((call) => String(call[0]).endsWith('/9/read')),
  ).toBe(true);
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
  const busyButton = await screen.findByRole('button', {
    name: 'Conversation, 2 unread',
  });
  expect(busyButton.querySelector('svg')).toBeTruthy();
  expect(busyButton.textContent).toContain('2');
  expect(screen.getByText('Quiet Site')).toBeTruthy();
  const quietButton = within(screen.getByText('Quiet Site').closest('tr')).getByRole(
    'button',
    { name: 'Conversation' },
  );
  expect(quietButton.textContent ?? '').not.toMatch(/\d/);
  expect(screen.queryByRole('button', { name: /0 unread/ })).toBeNull();
  expect(screen.queryByRole('button', { name: /4 unread/ })).toBeNull();
  await waitFor(() => expect(API.get).toHaveBeenCalledTimes(1));
  const row = screen.getByText('Busy Site').closest('tr');
  await user.click(within(row).getByRole('button', { name: /View details/ }));
  const dialog = await screen.findByRole('dialog');
  const processing = within(dialog).getByText('Processing');
  expect(dialog.querySelector('[data-supplier-conversation]')).toBeNull();
  expect(API.post).not.toHaveBeenCalled();
  const conversationButton = within(dialog).getByRole('button', {
    name: /Conversation/,
  });
  expect(conversationButton.querySelector('svg')).toBeTruthy();
  expect(dialog.querySelector('[data-supplier-detail-facts]').contains(conversationButton)).toBe(
    false,
  );
  expect(
    conversationButton.compareDocumentPosition(processing) &
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
    within(dialog)
      .getByRole('button', { name: /Conversation/ })
      .compareDocumentPosition(processingBlock) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  const facts = dialog.querySelector('[data-supplier-detail-facts]');
  expect(facts.style.overflowY).toBe('auto');
  expect(facts.style.minHeight === '0' || facts.style.minHeight === '0px').toBe(
    true,
  );
  expect(facts.contains(dialog.querySelector('[data-supplier-composer]'))).toBe(
    false,
  );
  await user.click(
    within(dialog).getByRole('button', { name: /Conversation/ }),
  );
  const conversation = screen.getByRole('dialog', {
    name: /Conversation with Busy Site/,
  });
  expect(within(conversation).getByText('No messages yet')).toBeTruthy();
  expectComposerDock(conversation);
  expect(dialog.contains(conversation)).toBe(false);
  await waitFor(() =>
    expect(screen.queryByRole('button', { name: /2 unread/ })).toBeNull(),
  );
  expect(
    API.post.mock.calls.some((call) => String(call[0]).endsWith('/11/read')),
  ).toBe(true);
  expect(
    API.get.mock.calls.some((call) => String(call[0]).includes('/messages')),
  ).toBe(false);
});

test('我的申请从列表直接打开对话，不展开详情', async () => {
  const user = userEvent.setup();
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (String(url).includes('/messages')) throw new Error(`unexpected ${url}`);
    return page([
      {
        ...busy,
        messages: [
          {
            id: 4,
            author_id: 1,
            author_role: 'admin',
            author_name: 'reviewer',
            type: 'message',
            body: 'Please check the quote',
            created_at: 1792634500,
          },
        ],
      },
    ]);
  });
  render(<MySupplierSubmissions />);
  const row = (await screen.findByText('Busy Site')).closest('tr');
  await user.click(
    within(row).getByRole('button', { name: 'Conversation, 2 unread' }),
  );
  const conversation = screen.getByRole('dialog', {
    name: /Conversation with Busy Site/,
  });
  expect(within(conversation).getByText('Please check the quote')).toBeTruthy();
  expect(screen.queryByText('Site information')).toBeNull();
  expect(API.get).toHaveBeenCalledTimes(1);
  expect(
    API.post.mock.calls.some((call) => String(call[0]).endsWith('/11/read')),
  ).toBe(true);
});

test('管理列表从站点名称旁直接打开对话，不请求详情', async () => {
  const user = userEvent.setup();
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    const path = String(url);
    if (path.includes('/messages') || path.endsWith('/11')) {
      throw new Error(`unexpected ${path}`);
    }
    return page([
      {
        ...busy,
        message_count: 4,
        unread_count: 1,
        messages: [
          {
            id: 8,
            author_id: 8,
            author_role: 'applicant',
            author_name: 'busy-user',
            type: 'message',
            body: 'Can you review this quote?',
            created_at: 1792634500,
          },
        ],
      },
    ]);
  });
  render(<SupplierApplicationManagementTable />);
  const button = await screen.findByRole('button', {
    name: 'Conversation, 1 unread',
  });
  expect(button.textContent).toContain('1');
  expect(button.textContent).not.toContain('4');
  await user.click(button);
  const conversation = screen.getByRole('dialog', {
    name: /Conversation with Busy Site/,
  });
  expect(within(conversation).getByText('Can you review this quote?')).toBeTruthy();
  expect(screen.queryByText('Processing')).toBeNull();
  expect(API.get).toHaveBeenCalledTimes(1);
  expect(
    API.post.mock.calls.some((call) => String(call[0]).endsWith('/11/read')),
  ).toBe(true);
});
