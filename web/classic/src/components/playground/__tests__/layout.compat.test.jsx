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
import { render, renderHook, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';
import ChatArea from '../ChatArea';
import { usePlaygroundState } from '../../../hooks/playground/usePlaygroundState';

function chatProps(overrides = {}) {
  return {
    message: [],
    inputs: { model: 'model-a', group: 'group-code' },
    models: [{ label: 'model-a', value: 'model-a' }],
    groups: [{ label: 'Premium display name', value: 'group-code' }],
    modelsReady: true,
    roleInfo: {},
    onMessageSend: vi.fn(),
    onClearMessages: vi.fn(),
    onOpenSettings: vi.fn(),
    onToggleDebugPanel: vi.fn(),
    onInputChange: vi.fn(),
    onTokenChange: vi.fn(),
    onStopGenerator: vi.fn(),
    ...overrides,
  };
}

test('empty chat matches the centered layout and starter prompts fill the real composer without sending', async () => {
  const props = chatProps();
  render(<ChatArea {...props} />);
  expect(
    screen.getByRole('heading', { name: 'Start a playground chat' }),
  ).toBeTruthy();
  for (const name of ['Analyze data', 'Summarize text', 'Code', 'Get advice']) {
    expect(screen.getByRole('button', { name })).toBeTruthy();
  }
  await userEvent.click(screen.getByRole('button', { name: 'Code' }));
  await waitFor(() =>
    expect(screen.getByPlaceholderText('Ask anything').value).toBe('Code'),
  );
  expect(props.onMessageSend).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole('button', { name: 'Parameters' }));
  expect(props.onOpenSettings).toHaveBeenCalledOnce();
  await userEvent.click(screen.getByRole('button', { name: 'Debug' }));
  expect(props.onToggleDebugPanel).toHaveBeenCalledOnce();
  expect(screen.getByLabelText('分组')).toBeTruthy();
  expect(screen.getByText('Premium display name')).toBeTruthy();
});

test('token mode omits account group, loading failures disable send, generation retains stop', async () => {
  const props = chatProps({
    selectedToken: { id: 7, name: 'Review token', prompt_count: 2 },
    modelsReady: false,
    modelError: 'Token disabled',
    onRetryModels: vi.fn(),
  });
  const view = render(<ChatArea {...props} />);
  expect(screen.queryByLabelText('分组')).toBeNull();
  expect(screen.getByText('Skills · 2')).toBeTruthy();
  await userEvent.click(
    screen.getByRole('button', { name: 'Use account mode' }),
  );
  expect(props.onTokenChange).toHaveBeenCalledWith(null);
  expect(screen.getByRole('alert').textContent).toContain('Token disabled');
  expect(screen.getByRole('button', { name: 'Send' }).disabled).toBe(true);
  expect(screen.getByPlaceholderText('Ask anything').disabled).toBe(true);
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
  expect(props.onRetryModels).toHaveBeenCalledOnce();
  view.rerender(<ChatArea {...props} modelsReady isRequesting modelError='' />);
  expect(
    screen.getByRole('button', { name: 'Switch API token' }).disabled,
  ).toBe(true);
  expect(
    screen.getByRole('button', { name: 'Use account mode' }).disabled,
  ).toBe(true);
  expect(screen.getByRole('button', { name: 'Parameters' }).disabled).toBe(
    true,
  );
  await userEvent.click(
    screen.getByRole('button', { name: 'Stop generating' }),
  );
  expect(props.onStopGenerator).toHaveBeenCalledOnce();
});

test('未选模型禁止普通发送，自定义请求正文的模型不受下拉框限制', async () => {
  const props = chatProps({ inputs: { model: '', group: 'group-code' } });
  const view = render(<ChatArea {...props} />);
  await userEvent.click(screen.getByRole('button', { name: 'Code' }));
  expect(screen.getByRole('button', { name: 'Send' }).disabled).toBe(true);
  view.rerender(<ChatArea {...props} customRequestMode />);
  expect(screen.getByRole('button', { name: 'Send' }).disabled).toBe(false);
});

test('saved conversations survive the layout migration while new sessions start empty', () => {
  const first = renderHook(() => usePlaygroundState());
  expect(first.result.current.message).toEqual([]);
  first.unmount();
  const saved = [
    {
      id: 'real-user-message',
      role: 'user',
      content: 'Keep this conversation',
      status: 'complete',
    },
  ];
  localStorage.setItem(
    'playground_messages',
    JSON.stringify({ messages: saved }),
  );
  const restored = renderHook(() => usePlaygroundState());
  expect(restored.result.current.message).toEqual(saved);
});
