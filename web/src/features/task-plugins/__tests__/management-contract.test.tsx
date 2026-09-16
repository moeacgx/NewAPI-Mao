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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, cleanup, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { afterEach, expect, test, vi } from 'vitest'

import { Form } from '@/components/ui/form'
import type { ChannelFormValues } from '@/features/channels/lib/channel-form'
import { api } from '@/lib/api'

import {
  getTaskPluginEnabledOption,
  setTaskPluginEnabledOption,
  listTaskPlugins,
} from '../api'
import { PluginDetailSheet } from '../components/plugin-detail-sheet'
import { PluginsTable } from '../components/plugins-table'
import { TaskPluginBindingField } from '../components/task-plugin-binding-field'
import { TaskPlugins } from '../index'
import type { TaskPluginListItem } from '../types'

vi.mock('@/lib/lobe-icon', () => ({ getLobeIcon: () => null }))
vi.mock('@/components/layout', () => {
  const Box = (props: { children?: ReactNode }) => <div>{props.children}</div>
  return {
    SectionPageLayout: Object.assign(Box, {
      Title: Box,
      Actions: Box,
      Content: Box,
    }),
  }
})

const clients: QueryClient[] = []
afterEach(() => {
  cleanup()
  clients.forEach((client) => client.clear())
  clients.length = 0
})
const item: TaskPluginListItem = {
  meta: {
    apiVersion: 1,
    key: 'sora',
    name: 'Sora',
    version: '1.0.0',
    author: { name: 'QuantumNous' },
    models: ['sora-2'],
    fetchMode: 'per_task',
  },
  source: 'factory',
  enabled: false,
  active: false,
  source_hash: 'hash',
  remark: '',
  channel_count: 2,
  in_flight_count: 3,
}
function client() {
  const value = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  clients.push(value)
  return value
}

function BindingForm() {
  const form = useForm<ChannelFormValues>({
    defaultValues: { task_plugin_key: 'sora' },
  })
  return (
    <Form {...form}>
      <TaskPluginBindingField form={form} disabled={false} />
    </Form>
  )
}

test.each(['status', 'activate'])(
  '%s 成功后已打开的渠道绑定显示最新选项',
  async (action) => {
    const query = client()
    query.setQueryData(
      ['task-plugin-options'],
      [
        {
          key: 'sora',
          name: 'Sora',
          version: '1.0.0',
          models: [],
          channel_type: 62,
        },
      ]
    )
    query.setQueryData(['task-plugins'], [{ ...item, active: true }])
    query.setQueryData(['task-plugin', 'sora'], { meta: item.meta })
    query.setQueryData(
      ['task-plugin-versions', 'sora'],
      [{ id: 1, version: '1.0.0', active: false }]
    )
    vi.spyOn(api, 'get').mockImplementation(async (url) => {
      let data: unknown = [{ ...item, active: true, enabled: true }]
      if (url === '/api/task_plugin_options') {
        data = [
          {
            key: 'sora',
            name: 'Sora',
            version: '2.0.0',
            models: [],
            channel_type: 62,
          },
        ]
      } else if (String(url).endsWith('/versions')) {
        data = [{ id: 1, version: '1.0.0', active: true }]
      } else if (url === '/api/plugin/task/sora') {
        data = { meta: item.meta }
      }
      return { data: { success: true, data } }
    })
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true, data: null } })
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={query}>
        <BindingForm />
        {action === 'status' ? (
          <PluginsTable canManage onDetails={() => {}} />
        ) : (
          <PluginDetailSheet canManage plugin={item} onOpenChange={() => {}} />
        )}
      </QueryClientProvider>
    )
    expect(
      screen.getByRole('option', { name: 'Sora (1.0.0)', hidden: true })
    ).toBeInTheDocument()
    if (action === 'activate') {
      await user.click(screen.getByRole('tab', { name: 'Version history' }))
      await user.click(
        screen.getByRole('button', { name: 'Activate / Roll back' })
      )
    } else {
      await user.click(
        screen.getByRole('switch', { name: 'Enable plugin sora' })
      )
    }
    await waitFor(() =>
      expect(
        screen.getByRole('option', { name: 'Sora (2.0.0)', hidden: true })
      ).toBeInTheDocument()
    )
    expect(post).toHaveBeenCalledWith(
      `/api/plugin/task/sora/${action}`,
      action === 'activate' ? { version: '1.0.0' } : { enabled: true },
      expect.any(Object)
    )
  }
)

test('runtime reads and writes the supported endpoint with an explicit false', async () => {
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { enabled: false, channel_type: 62 } },
  })
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, data: { enabled: false, channel_type: 62 } },
  })
  expect(await getTaskPluginEnabledOption()).toBe(false)
  await setTaskPluginEnabledOption(false)
  expect(get).toHaveBeenCalledWith('/api/plugin/task/runtime/status')
  expect(put).toHaveBeenCalledWith(
    '/api/plugin/task/runtime/status',
    { enabled: false },
    expect.any(Object)
  )
})

test('archived built-in version remains browsable and can be activated without an active version', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: [{ ...item, source_kind: 'builtin' }] },
  })
  expect((await listTaskPlugins())[0]).toMatchObject({
    enabled: false,
    active: false,
    meta: { key: 'sora' },
  })
  vi.mocked(api.get).mockImplementation(async (url) => ({
    data: {
      success: true,
      data: String(url).endsWith('/versions')
        ? [{ id: 1, version: '1.0.0', active: true }]
        : { meta: item.meta, source: 'const plugin = {}', layer: 'factory' },
    },
  }))
  const query = client()
  query.setQueryData(['task-plugin', 'sora'], {
    meta: item.meta,
    source: 'const plugin = {}',
    layer: 'factory',
  })
  query.setQueryData(
    ['task-plugin-versions', 'sora'],
    [{ id: 1, key: 'sora', version: '1.0.0', active: false, enabled: false }]
  )
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true, data: null } })
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={query}>
      <PluginDetailSheet canManage plugin={item} onOpenChange={() => {}} />
    </QueryClientProvider>
  )
  await user.click(screen.getByRole('tab', { name: 'Version history' }))
  const activate = screen.getByRole('button', { name: 'Activate / Roll back' })
  expect(activate).toBeEnabled()
  await user.click(activate)
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      '/api/plugin/task/sora/activate',
      { version: '1.0.0' },
      expect.any(Object)
    )
  )
})

test('disabled archived entry preserves counts and does not expose unsupported mutation controls', () => {
  const query = client()
  query.setQueryData(['task-plugins'], [item])
  const details = vi.fn()
  render(
    <QueryClientProvider client={query}>
      <PluginsTable onDetails={details} />
    </QueryClientProvider>
  )
  expect(screen.getByText('Not activated')).toBeVisible()
  expect(screen.getByText('2 channels, 3 in-flight tasks')).toBeVisible()
  expect(screen.getByRole('switch')).toHaveAttribute('aria-disabled', 'true')
  expect(screen.getByRole('button', { name: 'Delete' })).toBeDisabled()
})

test('list failure exits loading and retry can recover', async () => {
  const get = vi
    .spyOn(api, 'get')
    .mockRejectedValueOnce(new Error('Plugin service unavailable'))
    .mockResolvedValueOnce({ data: { success: true, data: [] } })
  const query = client()
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={query}>
      <PluginsTable onDetails={() => {}} />
    </QueryClientProvider>
  )
  expect(await screen.findByText('Plugin service unavailable')).toBeVisible()
  expect(screen.queryByText('Loading...')).toBeNull()
  await user.click(screen.getByRole('button', { name: 'Retry' }))
  expect(await screen.findByText('No task plugins found')).toBeVisible()
  expect(get).toHaveBeenCalledTimes(2)
})

test('administrator can browse marketplace but cannot upload or manage sources', async () => {
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: String(url).endsWith('/runtime/status')
        ? { enabled: false, channel_type: 62 }
        : [],
    },
  }))
  const query = client()
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={query}>
      <TaskPlugins />
    </QueryClientProvider>
  )
  expect(await screen.findByText('No task plugins found')).toBeVisible()
  const market = screen.getByRole('tab', { name: 'Marketplace' })
  const upload = screen.getByRole('button', { name: 'Upload plugin' })
  expect(market).toBeEnabled()
  expect(upload).toBeDisabled()
  await user.click(market)
  await user.click(upload)
  expect(get.mock.calls.map((call) => call[0]).sort()).toEqual(
    [
      '/api/plugin/task',
      '/api/plugin/task/runtime/status',
      '/api/plugin/task/marketplace/sources',
    ].sort()
  )
})

test('administrator can read versions but cannot activate or access source actions', async () => {
  const query = client()
  query.setQueryData(['task-plugin', 'sora'], {
    meta: item.meta,
    layer: 'factory',
  })
  query.setQueryData(
    ['task-plugin-versions', 'sora'],
    [{ id: 1, version: '1.0.0', active: false }]
  )
  const post = vi.spyOn(api, 'post')
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={query}>
      <PluginDetailSheet
        plugin={item}
        onOpenChange={() => {}}
        canManage={false}
      />
    </QueryClientProvider>
  )
  expect(screen.getByRole('tab', { name: 'Plugin source' })).toHaveAttribute(
    'aria-disabled',
    'true'
  )
  expect(screen.getByRole('tab', { name: 'Source diff' })).toHaveAttribute(
    'aria-disabled',
    'true'
  )
  await user.click(screen.getByRole('tab', { name: 'Version history' }))
  expect(
    screen.getByRole('button', { name: 'Activate / Roll back' })
  ).toBeDisabled()
  expect(post).not.toHaveBeenCalled()
})
