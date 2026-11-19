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
import type { QueryClient } from '@tanstack/react-query'
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { KeysPanel } from '../components/keys-panel'
import { McpManagement } from '../index'
import { key, renderMcp, response, tools } from './fixtures'

const originalAdapter = api.defaults.adapter
let client: QueryClient | undefined
beforeEach(() => {
  // 隔离浏览器存储边界，避免 Node 原生 Web Storage 覆盖 jsdom 的实现。
  for (const name of ['localStorage', 'sessionStorage']) {
    vi.stubGlobal(name, {
      getItem: vi.fn().mockReturnValue(null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    })
  }
})
afterEach(() => {
  cleanup()
  client?.clear()
  api.defaults.adapter = originalAdapter
  vi.unstubAllGlobals()
})

test('空名称、过去的有效期和未选工具阻止创建，字段错误可访问', async () => {
  const user = userEvent.setup()
  const writes: unknown[] = []
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.method === 'post') writes.push(config.data)
    return response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [], total: 0 }
    )
  }))
  await screen.findByRole('checkbox', { name: /newapi_context/ })
  await user.type(screen.getByLabelText('Key name'), '   ')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2020-01-01T12:00' },
  })
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  expect(await screen.findByText('Enter a key name')).toBeVisible()
  expect(screen.getByText('Choose a future expiration time')).toBeVisible()
  expect(screen.getByText('Select at least one tool')).toBeVisible()
  expect(screen.getByLabelText('Key name')).toHaveAttribute(
    'aria-invalid',
    'true'
  )
  expect(writes).toEqual([])
})

test('分类批选与只读批选提交显式工具名称，筛选不丢失隐藏授权', async () => {
  const user = userEvent.setup()
  const writes: unknown[] = []
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.method === 'post') {
      writes.push(JSON.parse(config.data))
      return response(config, { key, secret: 'mcp_bulk_selection_receipt' })
    }
    return response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [], total: 0 }
    )
  }))
  await screen.findByRole('checkbox', { name: /newapi_context/ })
  await user.type(screen.getByLabelText('Key name'), 'Scoped client')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2099-01-01T12:00' },
  })
  await user.click(
    screen.getByRole('button', { name: 'Select visible read-only tools' })
  )
  await user.selectOptions(screen.getByLabelText('Category'), 'channels')
  await user.click(
    screen.getByRole('button', {
      name: 'Select all visible tools in this category (including write)',
    })
  )
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  await screen.findByText('mcp_bulk_selection_receipt')
  expect(writes).toEqual([
    {
      name: 'Scoped client',
      tools: ['newapi_context', 'update_channel'],
      expires_at: new Date('2099-01-01T12:00').getTime() / 1000,
    },
  ])
})

test('逐项选择工具后按秒提交，完整密钥关闭即消失且不进入缓存或持久化存储', async () => {
  const user = userEvent.setup()
  const writes: unknown[] = []
  let created = false
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.method === 'post') {
      writes.push(JSON.parse(config.data))
      created = true
      return response(config, { key, secret: 'mcp_one_time_secret' })
    }
    return response(
      config,
      config.url?.endsWith('/tools')
        ? tools
        : { items: created ? [key] : [], total: created ? 1 : 0 }
    )
  }))
  const read = await screen.findByRole('checkbox', { name: /newapi_context/ })
  const write = screen.getByRole('checkbox', { name: /update_channel/ })
  expect(read).not.toBeChecked()
  expect(write).not.toBeChecked()
  await user.type(screen.getByLabelText('Key name'), '  Development client  ')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2099-01-01T12:00' },
  })
  await user.click(read)
  expect(write).not.toBeChecked()
  await user.click(write)
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  expect(await screen.findByText('mcp_one_time_secret')).toBeVisible()
  expect(writes).toEqual([
    {
      name: 'Development client',
      tools: ['newapi_context', 'update_channel'],
      expires_at: new Date('2099-01-01T12:00').getTime() / 1000,
    },
  ])
  await screen.findByText(key.name)
  expect(
    JSON.stringify(
      client
        ?.getQueryCache()
        .getAll()
        .map((query) => query.state.data)
    )
  ).not.toContain('mcp_one_time_secret')
  expect(
    JSON.stringify(
      client
        ?.getMutationCache()
        .getAll()
        .map((mutation) => mutation.state)
    )
  ).not.toContain('mcp_one_time_secret')
  expect(window.localStorage.setItem).not.toHaveBeenCalled()
  expect(window.sessionStorage.setItem).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button', { name: 'Close key receipt' }))
  expect(screen.queryByText('mcp_one_time_secret')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Create MCP key' })).toBeVisible()
})

test('切换页签清除创建回执，返回密钥页不再显示完整密钥', async () => {
  const user = userEvent.setup()
  ;({ client } = renderMcp(<McpManagement />, async (config) => {
    if (config.method === 'post') {
      return response(config, { key, secret: 'mcp_leave_tab_secret' })
    }
    return response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [], total: 0 }
    )
  }))
  await user.click(screen.getByRole('tab', { name: 'Call keys' }))
  await user.click(
    await screen.findByRole('checkbox', { name: /newapi_context/ })
  )
  await user.type(screen.getByLabelText('Key name'), 'Client')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2099-01-01T12:00' },
  })
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  await screen.findByText('mcp_leave_tab_secret')
  const refreshWithReceipt = new Event('beforeunload', { cancelable: true })
  window.dispatchEvent(refreshWithReceipt)
  expect(refreshWithReceipt.defaultPrevented).toBe(false)
  await user.click(screen.getByRole('tab', { name: 'MCP connection' }))
  expect(screen.queryByText('mcp_leave_tab_secret')).not.toBeInTheDocument()
  await user.click(screen.getByRole('tab', { name: 'Call keys' }))
  expect(screen.queryByText('mcp_leave_tab_secret')).not.toBeInTheDocument()
})

test('接入导航切换页签后重新初始化表单，不拦截离开页面', async () => {
  const user = userEvent.setup()
  ;({ client } = renderMcp(<McpManagement />, async (config) =>
    response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [], total: 0 }
    )
  ))
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  await user.click(
    await screen.findByRole('checkbox', { name: /update_channel/ })
  )
  await user.type(screen.getByLabelText('Key name'), 'Unsubmitted client')
  const refreshWithDraft = new Event('beforeunload', { cancelable: true })
  window.dispatchEvent(refreshWithDraft)
  expect(refreshWithDraft.defaultPrevented).toBe(false)
  await user.click(screen.getByRole('tab', { name: 'MCP connection' }))
  await user.click(screen.getByRole('button', { name: 'Tool reference' }))
  expect(screen.getByRole('tab', { name: 'Tool reference' })).toHaveAttribute(
    'aria-selected',
    'true'
  )
  await user.click(screen.getByRole('tab', { name: 'Call keys' }))
  expect(screen.getByLabelText('Key name')).toHaveValue('')
  expect(
    await screen.findByRole('checkbox', { name: /update_channel/ })
  ).not.toBeChecked()
})

test('密钥列表在创建表单之前，键盘导航先到常用管理操作', async () => {
  const user = userEvent.setup()
  ;({ client } = renderMcp(<KeysPanel />, async (config) =>
    response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [key], total: 1 }
    )
  ))
  const permissions = await screen.findByRole('button', {
    name: 'View permissions',
  })
  const keyName = screen.getByLabelText('Key name')
  expect(
    permissions.compareDocumentPosition(keyName) &
      Node.DOCUMENT_POSITION_FOLLOWING
  ).toBeTruthy()
  permissions.focus()
  await user.tab()
  expect(screen.getByRole('button', { name: 'Revoke' })).toHaveFocus()
  await user.tab()
  expect(keyName).toHaveFocus()
})

test('撤销须确认，取消不请求，失败保留确认框，成功更新状态', async () => {
  const user = userEvent.setup()
  const deletions: string[] = []
  let revoked = false
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.method === 'delete') {
      deletions.push(config.url ?? '')
      revoked = deletions.length > 1
      return response(config, null, revoked)
    }
    return response(
      config,
      config.url?.endsWith('/tools')
        ? tools
        : {
            items: [{ ...key, revoked_at: revoked ? 1800000000 : 0 }],
            total: 1,
          }
    )
  }))
  await user.click(await screen.findByRole('button', { name: 'Revoke' }))
  let dialog = screen.getByRole('alertdialog')
  expect(
    within(dialog).getByText(
      'Clients using Development client will immediately lose access.'
    )
  ).toBeVisible()
  expect(deletions).toEqual([])
  await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
  expect(deletions).toEqual([])
  await user.click(screen.getByRole('button', { name: 'Revoke' }))
  dialog = screen.getByRole('alertdialog')
  await user.click(
    within(dialog).getByRole('button', { name: 'Confirm revocation' })
  )
  expect(await within(dialog).findByText('Service unavailable')).toBeVisible()
  await user.click(
    within(dialog).getByRole('button', { name: 'Confirm revocation' })
  )
  expect(await screen.findByText('Revoked')).toBeVisible()
  await waitFor(() =>
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  )
  expect(screen.getByRole('button', { name: 'Revoke' })).toBeDisabled()
  expect(deletions).toEqual(['/api/mcp/keys/17', '/api/mcp/keys/17'])
})

test('密钥分页按契约请求，显示过期、撤销和最近使用信息', async () => {
  const user = userEvent.setup()
  const pages: unknown[] = []
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.url?.endsWith('/tools')) return response(config, tools)
    pages.push(config.params)
    const item =
      config.params.p === 1
        ? { ...key, expires_at: 1, last_used_at: 1700000010 }
        : { ...key, id: 18, name: 'Revoked client', revoked_at: 1700000020 }
    return response(config, { items: [item], total: 21 })
  }))
  expect(await screen.findByText('Expired')).toBeVisible()
  expect(screen.queryByText('Never used')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
  await user.click(screen.getByRole('button', { name: 'Next' }))
  expect(await screen.findByText('Revoked client')).toBeVisible()
  expect(screen.getByText('Revoked')).toBeVisible()
  expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  expect(pages).toEqual([
    { p: 1, page_size: 20 },
    { p: 2, page_size: 20 },
  ])
})

test('创建失败保留填写内容并允许重试，成功后可关闭回执再次创建', async () => {
  const user = userEvent.setup()
  let creations = 0
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.method === 'post') {
      creations += 1
      if (creations === 1) return response(config, null, false)
      return response(config, { key, secret: `mcp_retry_${creations}` })
    }
    return response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [], total: 0 }
    )
  }))
  await user.click(
    await screen.findByRole('checkbox', { name: /newapi_context/ })
  )
  await user.type(screen.getByLabelText('Key name'), 'Retry client')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2099-01-01T12:00' },
  })
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  expect(await screen.findByText('Service unavailable')).toBeVisible()
  expect(screen.getByLabelText('Key name')).toHaveValue('Retry client')
  expect(screen.getByRole('checkbox', { name: /newapi_context/ })).toBeChecked()
  expect(screen.queryByText('Save your MCP key')).not.toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  expect(await screen.findByText('mcp_retry_2')).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'Close key receipt' }))
  await user.click(screen.getByRole('checkbox', { name: /newapi_context/ }))
  await user.type(screen.getByLabelText('Key name'), 'Another client')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2099-01-01T12:00' },
  })
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  expect(await screen.findByText('mcp_retry_3')).toBeVisible()
  expect(creations).toBe(3)
})

test('同轮两次 submit 经异步表单校验后只创建一把密钥且不覆盖回执', async () => {
  const user = userEvent.setup()
  const completions: (() => void)[] = []
  let creations = 0
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.method === 'post') {
      const sequence = ++creations
      await new Promise<void>((resolve) => {
        completions.push(resolve)
      })
      return response(config, { key, secret: `mcp_same_turn_${sequence}` })
    }
    return response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [], total: 0 }
    )
  }))
  await user.click(
    await screen.findByRole('checkbox', { name: /newapi_context/ })
  )
  await user.type(screen.getByLabelText('Key name'), 'Client')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2099-01-01T12:00' },
  })
  const form = screen
    .getByRole('button', { name: 'Create MCP key' })
    .closest('form')
  if (!form) throw new Error('Expected key creation form')
  // 两次原生提交不让出执行权，保留异步 resolver 的真实竞争窗口。
  await act(async () => {
    fireEvent.submit(form)
    fireEvent.submit(form)
  })
  await screen.findByRole('button', { name: 'Creating...' })
  await act(async () => {
    completions.forEach((finish) => finish())
  })
  expect(creations).toBe(1)
  expect(await screen.findByText('mcp_same_turn_1')).toBeVisible()
  expect(screen.queryByText('mcp_same_turn_2')).not.toBeInTheDocument()
})

test('创建请求未完成时禁用提交和工具选择，避免重复创建', async () => {
  const user = userEvent.setup()
  let finish: (() => void) | undefined
  let creations = 0
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.method === 'post') {
      creations += 1
      await new Promise<void>((resolve) => {
        finish = resolve
      })
      return response(config, { key, secret: 'mcp_pending_secret' })
    }
    return response(
      config,
      config.url?.endsWith('/tools') ? tools : { items: [], total: 0 }
    )
  }))
  await user.click(
    await screen.findByRole('checkbox', { name: /newapi_context/ })
  )
  await user.type(screen.getByLabelText('Key name'), 'Client')
  fireEvent.change(screen.getByLabelText('Expiration time'), {
    target: { value: '2099-01-01T12:00' },
  })
  await user.click(screen.getByRole('button', { name: 'Create MCP key' }))
  const submit = await screen.findByRole('button', { name: 'Creating...' })
  expect(submit).toBeDisabled()
  expect(
    screen.getByRole('checkbox', { name: /update_channel/ })
  ).toHaveAttribute('aria-disabled', 'true')
  await user.click(submit)
  expect(creations).toBe(1)
  finish?.()
  expect(await screen.findByText('mcp_pending_secret')).toBeVisible()
})
