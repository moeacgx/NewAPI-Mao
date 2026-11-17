/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, expect, test } from 'vitest'

import { api } from '@/lib/api'

import { ChannelInfoDialog } from '../channel-info-dialog'

const adapter = api.defaults.adapter
let client: QueryClient
let channel: Record<string, unknown>
beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  channel = {
    id: 42,
    name: '渠道',
    type: 1,
    status: 1,
    models: '',
    group: 'legacy,deleted',
    group_ids: [7, 99],
    group_details: [{ id: 7, code: '7', name: '当前名称' }],
  }
  api.defaults.adapter = async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: { success: true, data: channel },
  })
})
afterEach(() => {
  api.defaults.adapter = adapter
  client.clear()
})
function show() {
  return render(
    <QueryClientProvider client={client}>
      <ChannelInfoDialog channelId={42} open onOpenChange={() => {}} />
    </QueryClientProvider>
  )
}
test('稳定 ID 与旧 alias 展示当前名称并支持改名', async () => {
  show()
  expect(await screen.findByText('当前名称, deleted')).toBeInTheDocument()
  expect(screen.queryByText('legacy,deleted')).not.toBeInTheDocument()
  channel = {
    ...channel,
    group_details: [{ id: 7, code: '7', name: '改名后' }],
  }
  await act(async () => {
    await client.invalidateQueries()
  })
  expect(await screen.findByText('改名后, deleted')).toBeInTheDocument()
})
test('无 ID 时按 code 展示且缺失名称回退', async () => {
  channel = {
    ...channel,
    group: '7,empty,deleted',
    group_ids: [],
    group_details: [
      { id: 7, code: '7', name: '当前名称' },
      { id: 8, code: 'empty', name: '  ' },
    ],
  }
  show()
  expect(
    await screen.findByText('当前名称, empty, deleted')
  ).toBeInTheDocument()
})
test('历史响应保留标识', async () => {
  channel = {
    ...channel,
    group: 'legacy',
    group_ids: undefined,
    group_details: undefined,
  }
  show()
  expect(await screen.findByText('legacy')).toBeInTheDocument()
})
test('引用顺序与字符串不一致时仍按稳定 ID 展示', async () => {
  channel = {
    ...channel,
    group: 'reused,old',
    group_ids: [8, 7],
    group_details: [
      { id: 7, code: 'reused', name: '第一组当前名称' },
      { id: 8, code: '8', name: '第二组当前名称' },
    ],
  }
  show()
  expect(
    await screen.findByText('第二组当前名称, 第一组当前名称')
  ).toBeInTheDocument()
})
