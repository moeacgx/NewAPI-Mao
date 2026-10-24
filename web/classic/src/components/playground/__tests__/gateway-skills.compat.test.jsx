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
import { act, render, renderHook, screen } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import MessageContent from '../MessageContent';
import { parseGatewaySkillsHeader } from '../../../helpers/gateway-skills';
import { createPlaygroundRequestHeaders } from '../../../helpers/playground-auth';
import { formatMessageForAPI } from '../../../helpers/utils';
import { useApiRequest } from '../../../hooks/playground/useApiRequest';

afterEach(() => vi.unstubAllGlobals());

const encodeHeader = (value) => {
  const bytes = new TextEncoder().encode(JSON.stringify(value));
  let binary = '';
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
};

const skillHeader = encodeHeader({
  version: 1,
  total: 1,
  skills: [
    {
      template_id: 12,
      skill_id: 'review',
      version: 3,
      title: '代码审查助手',
    },
  ],
  truncated: false,
});

test('解析严格的网关技能响应头并拒绝损坏值', () => {
  expect(
    parseGatewaySkillsHeader({ 'x-new-api-gateway-skills': [skillHeader] }),
  ).toEqual({
    version: 1,
    total: 1,
    truncated: false,
    skills: [
      {
        template_id: 12,
        skill_id: 'review',
        version: 3,
        title: '代码审查助手',
      },
    ],
  });
  expect(
    parseGatewaySkillsHeader({ 'x-new-api-gateway-skills': ['not-base64'] }),
  ).toBeNull();
  expect(
    parseGatewaySkillsHeader({
      'x-new-api-gateway-skills': [skillHeader.slice(0, -1)],
    }),
  ).toBeNull();
});

test('操练场请求声明只接收网关技能元数据', () => {
  const headers = createPlaygroundRequestHeaders(
    7,
    { getItem: () => JSON.stringify({ token: 'access-token' }) },
    42,
  );
  expect(headers['X-New-API-Skills-Display']).toBe('metadata');
  expect(headers['X-Playground-Token-Id']).toBe('42');
});

test('SSE open 响应头绑定到本次成功完成的助手消息', () => {
  const xhrs = [];
  class TestXHR extends EventTarget {
    constructor() {
      super();
      this.headers = {};
      this.readyState = 1;
      this.status = 200;
      this.responseText = '';
    }

    open() {}

    setRequestHeader(name, value) {
      this.headers[name] = value;
    }

    getAllResponseHeaders() {
      return `x-new-api-gateway-skills: ${skillHeader}\r\n`;
    }

    send(body) {
      this.body = body;
      xhrs.push(this);
    }

    abort() {
      this.dispatchEvent(new Event('abort'));
    }
  }
  TestXHR.HEADERS_RECEIVED = 2;
  vi.stubGlobal('XMLHttpRequest', TestXHR);

  let messages = [
    { role: 'user', content: '检查代码' },
    { role: 'assistant', content: '', status: 'loading' },
  ];
  const setMessage = vi.fn((update) => {
    messages = typeof update === 'function' ? update(messages) : update;
  });
  const sseSourceRef = { current: null };
  const { result } = renderHook(() =>
    useApiRequest(setMessage, vi.fn(), vi.fn(), sseSourceRef, vi.fn()),
  );

  act(() =>
    result.current.sendRequest(
      { model: 'test-model', messages: [], stream: true },
      true,
    ),
  );

  expect(xhrs).toHaveLength(1);
  expect(xhrs[0].headers['X-New-API-Skills-Display']).toBe('metadata');
  expect(JSON.parse(xhrs[0].body).messages).toEqual([]);
  xhrs[0].readyState = TestXHR.HEADERS_RECEIVED;
  xhrs[0].dispatchEvent(new Event('readystatechange'));
  xhrs[0].responseText =
    'data: {"choices":[{"delta":{"content":"完成"}}]}\n\ndata: [DONE]\n\n';
  act(() => xhrs[0].dispatchEvent(new Event('progress')));

  expect(messages[1].status).toBe('complete');
  expect(messages[1].gatewaySkills).toEqual({
    version: 1,
    total: 1,
    truncated: false,
    skills: [
      {
        template_id: 12,
        skill_id: 'review',
        version: 3,
        title: '代码审查助手',
      },
    ],
  });
  expect(sseSourceRef.current).toBeNull();
});

test('助手成功消息显示技能徽标，但徽标不会进入后续 API 消息', () => {
  const message = {
    id: 'assistant-1',
    role: 'assistant',
    content: '已完成检查。',
    status: 'complete',
    gatewaySkills: {
      version: 1,
      total: 1,
      truncated: false,
      skills: [{ title: '代码审查助手' }],
    },
  };
  render(
    <MessageContent
      message={message}
      styleState={{ isMobile: false }}
      className='message'
    />,
  );
  expect(screen.getByText('Gateway cloud Skills')).toBeTruthy();
  expect(screen.getByText('代码审查助手')).toBeTruthy();
  expect(formatMessageForAPI(message)).toEqual({
    role: 'assistant',
    content: '已完成检查。',
  });
});
