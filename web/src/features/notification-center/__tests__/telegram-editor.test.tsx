/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import {
  TelegramTemplateEditor,
  TICKET_EMOJI_TEMPLATE,
} from '../telegram-template-editor'

function Editor(props: {
  initial?: string
  eventType?: string
  mention?: string
}) {
  const [value, setValue] = useState(props.initial ?? 'Custom message')
  return (
    <TelegramTemplateEditor
      value={value}
      onChange={setValue}
      eventType={props.eventType ?? 'ticket_user_message'}
      target={{
        chat_id: '-1001',
        mention_user_id: '123',
        mention_name: props.mention ?? 'Support',
      }}
    />
  )
}

function setup(initial?: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <Editor initial={initial} />
    </QueryClientProvider>
  )
}

async function updatePreview() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(300)
  })
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0)
  })
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      success: true,
      data: { html: '<b>Preview text</b>', text_length: 12 },
    },
  })
})

afterEach(() => vi.useRealTimers())

test('选中文本后加粗保留正文与选区焦点，emoji 替换选区', async () => {
  setup('Hello world')
  const input = screen.getByRole<HTMLTextAreaElement>('textbox', {
    name: 'Message template',
  })
  input.focus()
  input.setSelectionRange(6, 11)
  fireEvent.select(input)
  fireEvent.click(screen.getByRole('button', { name: 'Bold' }))
  expect(input).toHaveValue('Hello <b>world</b>')
  expect(input).toHaveFocus()
  expect(input.value.slice(input.selectionStart, input.selectionEnd)).toBe(
    'world'
  )
  fireEvent.click(screen.getByRole('button', { name: 'Insert emoji ✅' }))
  expect(input).toHaveValue('Hello <b>✅</b>')
  expect(input).toHaveFocus()
  await updatePreview()
})

test('空选区插入引用后光标位于标签内，继续键入可保留引用格式', () => {
  setup('Hello')
  const input = screen.getByRole<HTMLTextAreaElement>('textbox', {
    name: 'Message template',
  })
  input.focus()
  input.setSelectionRange(5, 5)
  fireEvent.select(input)
  fireEvent.click(screen.getByRole('button', { name: /^Quote$/ }))
  expect(input).toHaveValue('Hello<blockquote></blockquote>')
  expect(input.selectionStart).toBe(17)
  expect(input.selectionEnd).toBe(17)
  expect(input).toHaveFocus()
})

test('键盘激活格式按钮后包裹选区并将焦点返回正文', async () => {
  vi.useRealTimers()
  setup('Support')
  const user = userEvent.setup()
  const input = screen.getByRole<HTMLTextAreaElement>('textbox', {
    name: 'Message template',
  })
  input.focus()
  input.setSelectionRange(0, 7)
  fireEvent.select(input)
  screen.getByRole('button', { name: 'Bold' }).focus()
  await user.keyboard('{Enter}')
  expect(input).toHaveValue('<b>Support</b>')
  expect(input).toHaveFocus()
})

test('原有自定义模板保持原样，显式点击才应用工单 emoji 预设', () => {
  setup('Original custom prefix {{content}}')
  const input = screen.getByRole('textbox', { name: 'Message template' })
  expect(input).toHaveValue('Original custom prefix {{content}}')
  fireEvent.click(
    screen.getByRole('button', { name: 'Use ticket emoji preset' })
  )
  expect(input).toHaveValue(TICKET_EMOJI_TEMPLATE)
})

test('链接面板拒绝脚本协议，合法链接包裹原选中文本并恢复编辑焦点', () => {
  setup('Support')
  const input = screen.getByRole<HTMLTextAreaElement>('textbox', {
    name: 'Message template',
  })
  input.focus()
  input.setSelectionRange(0, 7)
  fireEvent.select(input)
  fireEvent.click(screen.getByRole('button', { name: 'Insert link' }))
  const url = screen.getByRole('textbox', { name: 'Link URL' })
  fireEvent.change(url, { target: { value: 'javascript:alert(1)' } })
  expect(
    screen.getAllByRole('button', { name: 'Insert link' })[1]
  ).toBeDisabled()
  fireEvent.change(url, {
    target: { value: 'https://example.com/?q="test"&x=1' },
  })
  fireEvent.click(screen.getAllByRole('button', { name: 'Insert link' })[1])
  expect(input).toHaveValue(
    '<a href="https://example.com/?q=&quot;test&quot;&amp;x=1">Support</a>'
  )
  expect(input).toHaveFocus()
})

test('实时预览使用原始模板与首个目标提及，不发送 Telegram 消息', async () => {
  setup('{{mention}} Custom')
  expect(api.post).not.toHaveBeenCalled()
  await updatePreview()
  expect(api.post).toHaveBeenCalledWith(
    '/api/notification/preview',
    {
      event_type: 'ticket_user_message',
      template: '{{mention}} Custom',
      target: { mention_user_id: '123', mention_name: 'Support' },
    },
    expect.objectContaining({ signal: expect.any(AbortSignal) })
  )
  expect(screen.getByText('Preview text')).toBeVisible()
  expect(screen.getByText('12 characters')).toBeVisible()
})

test('编辑后立即隐藏成功预览，新预览校验失败时仅展示错误', async () => {
  setup()
  await updatePreview()
  expect(screen.getByText('Preview text')).toBeVisible()
  vi.mocked(api.post).mockResolvedValue({
    data: { success: false, message: 'Unknown template variable' },
  })
  fireEvent.change(screen.getByRole('textbox', { name: 'Message template' }), {
    target: { value: '{{unknown}}' },
  })
  expect(screen.queryByText('Preview text')).not.toBeInTheDocument()
  await updatePreview()
  expect(screen.getByRole('alert')).toHaveTextContent(
    'Unknown template variable'
  )
  expect(screen.queryByText('Preview text')).not.toBeInTheDocument()
})

test('旧请求晚于新请求返回时，页面仍显示新模板预览', async () => {
  let resolveOld: (value: unknown) => void = () => undefined
  vi.mocked(api.post).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        resolveOld = resolve
      })
  )
  setup('Old')
  await updatePreview()
  fireEvent.change(screen.getByRole('textbox', { name: 'Message template' }), {
    target: { value: 'New' },
  })
  await updatePreview()
  expect(screen.getByText('Preview text')).toBeVisible()
  await act(async () => {
    resolveOld({
      data: { success: true, data: { html: 'Old preview', text_length: 11 } },
    })
  })
  expect(screen.queryByText('Old preview')).not.toBeInTheDocument()
  expect(screen.getByText('Preview text')).toBeVisible()
})

test('切换事件和目标提及时立即清除旧预览并请求新样例', async () => {
  const client = new QueryClient()
  const view = render(
    <QueryClientProvider client={client}>
      <Editor eventType='ticket_user_message' />
    </QueryClientProvider>
  )
  await updatePreview()
  expect(screen.getByText('Preview text')).toBeVisible()
  view.rerender(
    <QueryClientProvider client={client}>
      <Editor eventType='channel_disabled' mention='Operations' />
    </QueryClientProvider>
  )
  expect(screen.queryByText('Preview text')).not.toBeInTheDocument()
  await updatePreview()
  expect(api.post).toHaveBeenLastCalledWith(
    '/api/notification/preview',
    expect.objectContaining({
      event_type: 'channel_disabled',
      target: { mention_user_id: '123', mention_name: 'Operations' },
    }),
    expect.anything()
  )
  expect(
    screen.queryByRole('button', { name: 'Use ticket emoji preset' })
  ).not.toBeInTheDocument()
})
