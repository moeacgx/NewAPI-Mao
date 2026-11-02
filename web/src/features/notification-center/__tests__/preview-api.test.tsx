/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import {
  AxiosError,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import { useState } from 'react'
import { toast } from 'sonner'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { TelegramTemplateEditor } from '../telegram-template-editor'

const originalAdapter = api.defaults.adapter
let client: QueryClient

function Editor() {
  const [template, setTemplate] = useState('Original template')
  return (
    <TelegramTemplateEditor
      value={template}
      eventType='ticket_user_message'
      onChange={setTemplate}
    />
  )
}

function renderEditor() {
  return render(
    <QueryClientProvider client={client}>
      <Editor />
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
  vi.spyOn(toast, 'error').mockImplementation(() => '')
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  api.defaults.adapter = originalAdapter
  vi.useRealTimers()
})

test('预览业务校验失败经过默认 HTTP 拦截器后，仅在面板显示错误', async () => {
  api.defaults.adapter = async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: { success: false, message: 'Unknown template variable' },
  })
  renderEditor()
  await updatePreview()
  expect(screen.getByRole('alert')).toHaveTextContent(
    'Unknown template variable'
  )
  expect(toast.error).not.toHaveBeenCalled()
})

test('预览网络失败经过默认 HTTP 拦截器后，仅在面板显示错误', async () => {
  api.defaults.adapter = async (config) => {
    throw new AxiosError('Network unavailable', 'ERR_NETWORK', config)
  }
  renderEditor()
  await updatePreview()
  expect(screen.getByRole('alert')).toHaveTextContent('Network unavailable')
  expect(toast.error).not.toHaveBeenCalled()
})

test('编辑取消旧预览请求经过默认 HTTP 拦截器后，不弹全局提示且保留新结果', async () => {
  let oldRequest: InternalAxiosRequestConfig | undefined
  let finishOld: (response: AxiosResponse) => void = () => undefined
  api.defaults.adapter = (config) => {
    const request = JSON.parse(config.data)
    if (request.template === 'Original template') {
      oldRequest = config
      return new Promise((resolve) => {
        finishOld = resolve
      })
    }
    return Promise.resolve({
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: { success: true, data: { html: 'New preview', text_length: 11 } },
    })
  }
  renderEditor()
  await updatePreview()
  fireEvent.change(screen.getByRole('textbox', { name: 'Message template' }), {
    target: { value: 'New template' },
  })
  await updatePreview()
  const pendingRequest = oldRequest
  if (!pendingRequest) throw new Error('Expected an original preview request')
  expect(pendingRequest.signal?.aborted).toBe(true)
  expect(screen.getByText('New preview')).toBeVisible()
  await act(async () => {
    finishOld({
      config: pendingRequest,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: { success: true, data: { html: 'Stale preview', text_length: 13 } },
    })
  })
  expect(screen.queryByText('Stale preview')).not.toBeInTheDocument()
  expect(screen.getByText('New preview')).toBeVisible()
  expect(toast.error).not.toHaveBeenCalled()
})
