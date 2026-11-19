/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useEffect } from 'react'
import { afterEach, beforeEach, expect, test } from 'vitest'

import { api } from '@/lib/api'

import { ChannelsProvider, useChannels } from '../../channels-provider'
import { TagBatchEditDialog } from '../tag-batch-edit-dialog'

const adapter = api.defaults.adapter
let client: QueryClient
let payload: Record<string, unknown> | undefined
let name: string
let code: string
function Editor() {
  const { setCurrentTag } = useChannels()
  useEffect(() => {
    setCurrentTag('tag-fixture')
  }, [setCurrentTag])
  return <TagBatchEditDialog open onOpenChange={() => {}} />
}
beforeEach(() => {
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: window.localStorage,
  })
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  name = '当前名称'
  code = '7'
  payload = undefined
  api.defaults.adapter = async (config) => {
    let data: unknown = []
    if (config.url === '/api/group/') data = [code, 'deleted']
    if (config.url === '/api/group/details') data = [{ id: 7, code: '7', name }]
    if (config.url === '/api/channel/tag/models') data = ''
    if (config.method === 'put') payload = JSON.parse(config.data as string)
    return {
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: { success: true, data, group_names: { legacy: name } },
    }
  }
})
afterEach(() => {
  api.defaults.adapter = adapter
  client.clear()
})
test.each(['7', 'legacy'])(
  '选项 %s 展示当前名称且改名后仍提交原 code',
  async (selectedCode) => {
    code = selectedCode
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={client}>
        <ChannelsProvider>
          <Editor />
        </ChannelsProvider>
      </QueryClientProvider>
    )
    const input = await screen.findByRole('combobox')
    await user.click(input)
    await user.click(await screen.findByRole('option', { name: '当前名称' }))
    await user.keyboard('{Escape}')
    name = '改名后分组'
    await act(async () => {
      await client.invalidateQueries({ queryKey: ['group-details'] })
    })
    expect(await screen.findByText('改名后分组')).toBeInTheDocument()
    expect(document.querySelector('[title="7"], [aria-label="7"]')).toBeNull()
    await user.click(input)
    await user.click(await screen.findByRole('option', { name: 'deleted' }))
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('button', { name: 'Save Changes' }))
    await waitFor(() =>
      expect(payload).toMatchObject({
        tag: 'tag-fixture',
        groups: `${code},deleted`,
      })
    )
  }
)
