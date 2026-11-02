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

import React, { useState } from 'react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Toast } from '@douyinfe/semi-ui';
import TelegramTemplateEditor from '../TelegramTemplateEditor';
import TelegramPreview, { TelegramMessage } from '../TelegramPreview';
import NotificationCenter from '../index';
import { API } from '../../../helpers/api';

const originalAdapter = API.defaults.adapter;
const ticketEvent = {
  value: 'ticket_user_message',
  variables: ['content', 'title'],
};
const locales = import.meta.glob('../../../i18n/locales/*.json', {
  eager: true,
  import: 'default',
});

function EditorFixture(props) {
  const [value, onChange] = useState(props.value || 'before selected after');
  return (
    <TelegramTemplateEditor
      value={value}
      onChange={onChange}
      event={ticketEvent}
      eventType={ticketEvent.value}
    />
  );
}

beforeEach(() => {
  API.defaults.adapter = async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: {
      success: true,
      data: { html: '<b>Sample message</b>', text_length: 14 },
    },
  });
});
afterEach(() => {
  API.defaults.adapter = originalAdapter;
  vi.useRealTimers();
});

test('选中文本后点击加粗会包裹选区并恢复编辑焦点及选区', async () => {
  render(<EditorFixture />);
  const input = screen.getByRole('textbox', { name: '消息模板' });
  input.focus();
  input.setSelectionRange(7, 15);
  fireEvent.select(input);
  await userEvent.click(screen.getByRole('button', { name: 'Bold' }));
  expect(input.value).toBe('before <b>selected</b> after');
  expect(document.activeElement).toBe(input);
  expect(input.value.slice(input.selectionStart, input.selectionEnd)).toBe(
    'selected',
  );
});

test.each([
  ['Italic', '<i>selected</i>'],
  ['Underline', '<u>selected</u>'],
  ['Strikethrough', '<s>selected</s>'],
  ['Quote', '<blockquote>selected</blockquote>'],
  ['Expandable quote', '<blockquote expandable>selected</blockquote>'],
  ['Inline code', '<code>selected</code>'],
  ['Code block', '<pre>selected</pre>'],
  ['Spoiler', '<tg-spoiler>selected</tg-spoiler>'],
])('%s 工具在选区生成对应的 Telegram HTML 格式', async (label, expected) => {
  render(<EditorFixture value='selected' />);
  const input = screen.getByRole('textbox', { name: '消息模板' });
  input.focus();
  input.setSelectionRange(0, 8);
  fireEvent.select(input);
  await userEvent.click(screen.getByRole('button', { name: label }));
  expect(input.value).toBe(expected);
  expect(document.activeElement).toBe(input);
});

test('键盘触发变量和表情按钮在保存的光标位置插入且保留焦点', async () => {
  render(<EditorFixture value='start end' />);
  const input = screen.getByRole('textbox', { name: '消息模板' });
  input.focus();
  input.setSelectionRange(6, 6);
  fireEvent.select(input);
  const variable = screen.getByRole('button', { name: '{{content}}' });
  variable.focus();
  await userEvent.keyboard('{Enter}');
  expect(input.value).toBe('start {{content}}end');
  expect(document.activeElement).toBe(input);
  await userEvent.click(
    screen.getByRole('button', { name: 'Insert emoji ✅' }),
  );
  expect(input.value).toBe('start {{content}}✅end');
  expect(input.selectionStart).toBe('start {{content}}✅'.length);
});

test('显式点击工单预设前保留自定义模板，点击后生成带变量的表情与引用', async () => {
  render(<EditorFixture value='My custom prefix' />);
  const input = screen.getByRole('textbox', { name: '消息模板' });
  expect(input.value).toBe('My custom prefix');
  await userEvent.click(
    screen.getByRole('button', { name: 'Use ticket emoji and quote preset' }),
  );
  expect(input.value).toContain('🎫 <b>{{action_label}}');
  expect(input.value).toContain('<blockquote>{{content}}</blockquote>');
  expect(document.activeElement).toBe(input);
});

test('链接输入拒绝脚本协议且合法链接包裹原选区并转义属性字符', async () => {
  render(<EditorFixture value='link text' />);
  const input = screen.getByRole('textbox', { name: '消息模板' });
  input.focus();
  input.setSelectionRange(0, 4);
  fireEvent.select(input);
  await userEvent.click(screen.getByRole('button', { name: 'Insert link' }));
  const url = screen.getByRole('textbox', { name: 'Link URL' });
  fireEvent.change(url, { target: { value: 'javascript:alert(1)' } });
  await userEvent.click(
    screen.getAllByRole('button', { name: 'Insert link' })[1],
  );
  expect(screen.getByRole('alert').textContent).toBe(
    'Use an http, https, tg or mailto link.',
  );
  expect(input.value).toBe('link text');
  fireEvent.change(url, { target: { value: 'https://example.com/?a=1&b=2' } });
  await userEvent.click(
    screen.getAllByRole('button', { name: 'Insert link' })[1],
  );
  expect(input.value).toBe(
    '<a href="https://example.com/?a=1&amp;b=2">link</a> text',
  );
  expect(document.activeElement).toBe(input);
});

test('预览以白名单重建格式，不保留任意标签、事件属性或危险链接', () => {
  const { container } = render(
    <TelegramMessage
      html={
        '<b onclick="alert(1)">bold</b><script>bad()</script><img src=x onerror=alert(1)><a href="javascript:alert(1)">unsafe</a><a href="https://example.com" onclick="alert(1)">safe</a><blockquote expandable>quote</blockquote><tg-spoiler>secret</tg-spoiler><code>code</code>'
      }
    />,
  );
  expect(screen.getByText('bold').tagName).toBe('B');
  expect(
    container.querySelectorAll('script,img,[onclick],[onerror]'),
  ).toHaveLength(0);
  expect(screen.getByText('unsafe').tagName).toBe('SPAN');
  expect(screen.getByRole('link', { name: 'safe' }).getAttribute('href')).toBe(
    'https://example.com',
  );
  expect(screen.getByText('quote').closest('details').open).toBe(false);
  expect(screen.getByText('secret').closest('details').open).toBe(false);
});

test('预览使用后端样例与首个目标提及，输入变更防抖并清除旧结果', async () => {
  vi.useFakeTimers();
  const payloads = [];
  API.defaults.adapter = async (config) => {
    payloads.push(JSON.parse(config.data));
    return {
      config,
      status: 200,
      headers: {},
      data: {
        success: true,
        data: { html: '<b>Server sample</b>', text_length: 13 },
      },
    };
  };
  const { rerender } = render(
    <TelegramPreview
      eventType='ticket_user_message'
      template='one'
      target={{ mention_user_id: '42', mention_name: 'Worker' }}
    />,
  );
  await act(async () => vi.advanceTimersByTimeAsync(300));
  expect(screen.getByText('Server sample')).toBeTruthy();
  expect(payloads).toEqual([
    {
      event_type: 'ticket_user_message',
      template: 'one',
      target: { mention_user_id: '42', mention_name: 'Worker' },
    },
  ]);
  rerender(
    <TelegramPreview
      eventType='ticket_user_message'
      template='two'
      target={{ mention_user_id: '42', mention_name: 'Worker' }}
    />,
  );
  expect(screen.queryByText('Server sample')).toBeNull();
  await act(async () => vi.advanceTimersByTimeAsync(150));
  rerender(<TelegramPreview eventType='channel_disabled' template='three' />);
  await act(async () => vi.advanceTimersByTimeAsync(300));
  expect(payloads.map((payload) => payload.template)).toEqual(['one', 'three']);
});

test('新请求失败后清除成功预览，晚到的旧响应不覆盖当前错误，业务失败与取消不弹全局错误', async () => {
  vi.useFakeTimers();
  const toast = vi.spyOn(Toast, 'error');
  let finishOld;
  API.defaults.adapter = (config) => {
    const payload = JSON.parse(config.data);
    if (payload.template === 'old')
      return new Promise((resolve) => {
        finishOld = () =>
          resolve({
            config,
            status: 200,
            headers: {},
            data: {
              success: true,
              data: { html: 'Stale sample', text_length: 12 },
            },
          });
      });
    return Promise.resolve({
      config,
      status: 200,
      headers: {},
      data: { success: false, message: 'Unknown variable' },
    });
  };
  const { rerender } = render(
    <TelegramPreview eventType='ticket_user_message' template='old' />,
  );
  await act(async () => vi.advanceTimersByTimeAsync(300));
  rerender(
    <TelegramPreview eventType='ticket_user_message' template='invalid' />,
  );
  await act(async () => vi.advanceTimersByTimeAsync(300));
  expect(screen.getByRole('alert').textContent).toBe('Unknown variable');
  await act(async () => finishOld());
  expect(screen.queryByText('Stale sample')).toBeNull();
  expect(screen.getByRole('alert').textContent).toBe('Unknown variable');
  expect(toast).not.toHaveBeenCalled();
});

test('编辑未知历史事件任务时保留原模板并按真实事件请求预览，不回退发票事件', async () => {
  const previews = [];
  const template = 'Custom historical prefix {{legacy_value}}';
  localStorage.setItem('user', JSON.stringify({ id: 1, role: 100 }));
  API.defaults.adapter = async (config) => {
    let data = [];
    if (config.url.endsWith('/preview')) {
      previews.push(JSON.parse(config.data));
      return {
        config,
        status: 200,
        headers: {},
        data: { success: false, message: 'Unsupported historical event' },
      };
    }
    if (config.url.endsWith('/event-types')) {
      data = [
        {
          value: 'invoice_pending',
          label: 'Invoice pending',
          variables: ['invoice_id'],
        },
      ];
    } else if (config.url.endsWith('/bots')) {
      data = [{ id: 1, name: 'Support bot', enabled: true }];
    } else if (config.url.endsWith('/tasks')) {
      data = [
        {
          id: 2,
          name: 'Historical notification task',
          event_type: 'extension.uninstalled.old_event',
          event_name: 'Historical event',
          bot_id: 1,
          template,
          enabled: true,
          targets: [
            {
              chat_id: '-1001',
              mention_user_id: '42',
              mention_name: 'Support',
              enabled: true,
            },
          ],
        },
      ];
    }
    return { config, status: 200, headers: {}, data: { success: true, data } };
  };
  const toast = vi.spyOn(Toast, 'error');
  render(<NotificationCenter />);
  await screen.findByText('Historical notification task');
  await userEvent.click(screen.getByRole('button', { name: '编辑' }));
  expect(screen.getByRole('textbox', { name: '消息模板' }).value).toBe(
    template,
  );
  expect(screen.getByRole('button', { name: '{{invoice_id}}' })).toBeTruthy();
  expect(
    (await screen.findByRole('alert', {}, { timeout: 3000 })).textContent,
  ).toBe('Unsupported historical event');
  expect(previews).toEqual([
    {
      event_type: 'extension.uninstalled.old_event',
      template,
      target: { mention_user_id: '42', mention_name: 'Support' },
    },
  ]);
  expect(screen.getByRole('textbox', { name: '消息模板' }).value).toBe(
    template,
  );
  expect(toast).not.toHaveBeenCalled();
});

test('全部 Classic 语言完整翻译编辑器主要操作和预览说明', () => {
  for (const [file, resource] of Object.entries(locales)) {
    for (const key of [
      'Telegram HTML source',
      'Live message preview',
      'Telegram formatting',
      'Use ticket emoji and quote preset',
      'Sample data only. No message is sent. Uses the first recipient mention.',
    ]) {
      expect(resource.translation[key], `${file}: ${key}`).toBeTruthy();
      if (!file.endsWith('/en.json'))
        expect(resource.translation[key]).not.toBe(key);
    }
  }
});
