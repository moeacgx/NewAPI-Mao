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
import { test, expect } from 'vitest';
import ActiveRequestCard from '../ActiveRequestCard';

const request = {
  id: 'lifecycle-1',
  request_id: 'req-visible-123',
  token_id: 12,
  model_name: 'claude-opus-5',
  token_name: 'personal-development-key',
  started_at: 1000,
  first_response_ms: 2070,
  is_stream: true,
  request_type: 'openai',
  status: 'receiving',
  retry_count: 0,
};

test('手机默认只展示模型状态耗时，键盘展开后可查看并收起详情', async () => {
  render(
    <ActiveRequestCard
      request={request}
      serverNow={13000}
      compact
      showRequestId
    />,
  );
  const row = screen.getByRole('button', { name: /claude-opus-5/ });
  expect(row.getAttribute('aria-expanded')).toBe('false');
  expect(row.hasAttribute('aria-controls')).toBe(false);
  expect(within(row).getByText('Receiving response')).toBeTruthy();
  expect(within(row).getByText('12.0 s')).toBeTruthy();
  expect(screen.queryByText(request.token_name)).toBeNull();
  expect(screen.queryByText('First response')).toBeNull();
  row.focus();
  await userEvent.keyboard('{Enter}');
  expect(row.getAttribute('aria-expanded')).toBe('true');
  expect(
    document.getElementById(row.getAttribute('aria-controls')),
  ).toBeTruthy();
  expect(screen.getByText(request.token_name)).toBeTruthy();
  expect(screen.getByRole('definition', { name: '2.07 s' })).toBeTruthy();
  expect(screen.getByText(request.request_id)).toBeTruthy();
  expect(screen.getByText('OpenAI Chat Completions')).toBeTruthy();
  await userEvent.keyboard(' ');
  expect(row.getAttribute('aria-expanded')).toBe('false');
  expect(screen.queryByText(request.request_id)).toBeNull();
});

test('手机长模型名保留完整标题且实时数据刷新不折叠已展开详情', async () => {
  const model = 'custom/provider-deployment-with-a-very-long-model-name-2026';
  const view = render(
    <ActiveRequestCard
      compact
      request={{
        ...request,
        model_name: model,
        first_response_ms: null,
        status: 'waiting_response',
      }}
      serverNow={3000}
    />,
  );
  const row = screen.getByRole('button', { name: new RegExp(model) });
  expect(screen.getByText(model).getAttribute('title')).toBe(model);
  await userEvent.click(row);
  expect(screen.getAllByText('Waiting for response')).toHaveLength(2);
  view.rerender(
    <ActiveRequestCard
      compact
      request={{ ...request, model_name: model }}
      serverNow={13000}
    />,
  );
  expect(row.getAttribute('aria-expanded')).toBe('true');
  expect(screen.getByRole('definition', { name: '2.07 s' })).toBeTruthy();
  expect(screen.queryByText('Waiting for response')).toBeNull();
});

test('桌面耗时保留零首响应，模型图标遵循模型名称而不是入站协议', () => {
  render(
    <ActiveRequestCard
      request={{ ...request, first_response_ms: 0 }}
      serverNow={13000}
    />,
  );
  const article = screen.getByRole('article', { name: request.model_name });
  expect(
    within(article).getByRole('definition', { name: '0.00 s' }),
  ).toBeTruthy();
  expect(article.querySelector('[data-provider="Anthropic"]')).toBeTruthy();
  expect(within(article).getByText('OpenAI Chat Completions')).toBeTruthy();
});
