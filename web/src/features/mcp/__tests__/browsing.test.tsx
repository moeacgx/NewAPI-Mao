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
import { cleanup, fireEvent, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { ConnectionPanel } from '../components/connection-panel'
import { LogsPanel } from '../components/logs-panel'
import { ToolsPanel } from '../components/tools-panel'
import { renderMcp, response, tools } from './fixtures'

const originalAdapter = api.defaults.adapter
let client: QueryClient | undefined
afterEach(() => {
  cleanup()
  client?.clear()
  api.defaults.adapter = originalAdapter
})

test('连接配置复制当前站点地址并引用环境变量，说明可复制', async () => {
  const user = userEvent.setup()
  const copy = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
  ;({ client } = renderMcp(<ConnectionPanel />, async (config) =>
    response(config, [])
  ))
  await user.click(screen.getByRole('button', { name: 'Copy MCP URL' }))
  expect(copy).toHaveBeenLastCalledWith(`${window.location.origin}/mcp`)
  expect(screen.getByRole('tab', { name: 'Cursor' })).toHaveAttribute(
    'aria-selected',
    'true'
  )
  await user.click(screen.getByRole('tab', { name: 'Codex' }))
  expect(
    screen.queryByRole('button', { name: 'Copy Cursor configuration' })
  ).not.toBeInTheDocument()
  await user.click(
    screen.getByRole('button', { name: 'Copy Codex configuration' })
  )
  expect(copy).toHaveBeenLastCalledWith(
    `[mcp_servers.newapi_mao]\nurl = "${window.location.origin}/mcp"\nbearer_token_env_var = "NEWAPI_MCP_KEY"`
  )
  await user.click(screen.getByRole('tab', { name: 'Cursor' }))
  await user.click(
    screen.getByRole('button', { name: 'Copy Cursor configuration' })
  )
  const config = JSON.parse(copy.mock.calls.at(-1)?.[0] ?? '{}')
  expect(config).toEqual({
    mcpServers: {
      newapi_mao: {
        url: `${window.location.origin}/mcp`,
        headers: { Authorization: 'Bearer ${env:NEWAPI_MCP_KEY}' },
      },
    },
  })
  await user.click(
    screen.getByRole('button', { name: 'Copy connection instructions' })
  )
  expect(copy.mock.calls.at(-1)?.[0]).toContain('newapi_context')
})

test('工具数组可按搜索、分类及读写筛选，展开后展示服务端 schema', async () => {
  const user = userEvent.setup()
  ;({ client } = renderMcp(<ToolsPanel />, async (config) =>
    response(config, tools)
  ))
  await screen.findByText('newapi_context')
  await user.selectOptions(screen.getByLabelText('Category'), 'channels')
  expect(screen.queryByText('newapi_context')).not.toBeInTheDocument()
  await user.click(screen.getByText('update_channel'))
  expect(screen.getByText(/"required":/)).toHaveTextContent('request_id')
  await user.selectOptions(screen.getByLabelText('Access'), 'read')
  expect(screen.getByText('No matching tools')).toBeVisible()
  await user.selectOptions(screen.getByLabelText('Category'), '')
  await user.selectOptions(screen.getByLabelText('Access'), 'all')
  await user.type(screen.getByLabelText('Search tools'), 'CURRENT PERMISSIONS')
  expect(screen.getByText('newapi_context')).toBeVisible()
  expect(screen.queryByText('update_channel')).not.toBeInTheDocument()
})

test('工具加载失败显示重试，空列表不伪装成成功数据', async () => {
  const user = userEvent.setup()
  let succeeded = false
  ;({ client } = renderMcp(<ToolsPanel />, async (config) =>
    response(config, [], succeeded)
  ))
  expect(await screen.findByText('Service unavailable')).toBeVisible()
  expect(screen.queryByText('No matching tools')).not.toBeInTheDocument()
  succeeded = true
  await user.click(screen.getByRole('button', { name: 'Retry' }))
  expect(await screen.findByText('No matching tools')).toBeVisible()
})

test('工具参考完整展示动态 body 属性、嵌套约束和更新必填字段', async () => {
  const user = userEvent.setup()
  const inputSchema = {
    type: 'object',
    additionalProperties: false,
    required: ['path', 'body', 'request_id'],
    properties: {
      request_id: { type: 'string', minLength: 1 },
      path: {
        type: 'object',
        required: ['id'],
        properties: { id: { type: 'string', pattern: '^[1-9][0-9]*$' } },
        additionalProperties: false,
      },
      body: {
        type: 'object',
        required: [
          'name',
          'description',
          'icons',
          'sort_order',
          'group_ids',
          'match_keywords',
        ],
        properties: {
          name: { type: 'string', minLength: 1, maxLength: 64 },
          description: { type: 'string', maxLength: 512 },
          icons: {
            type: 'array',
            maxItems: 6,
            items: { type: 'string', minLength: 1, maxLength: 2048 },
          },
          sort_order: {
            type: 'integer',
            minimum: -1000000,
            maximum: 1000000,
          },
          group_ids: {
            type: 'array',
            maxItems: 10000,
            items: { type: 'integer', minimum: 1, maximum: 2147483647 },
          },
          match_keywords: {
            type: 'array',
            maxItems: 32,
            items: { type: 'string', maxLength: 128 },
          },
        },
        additionalProperties: false,
      },
    },
  }
  ;({ client } = renderMcp(<ToolsPanel />, async (config) =>
    response(config, [
      {
        name: 'newapi_group_tag_update',
        description: 'Save the complete tag configuration.',
        category: 'groups',
        write: true,
        min_role: 10,
        inputSchema,
      },
    ])
  ))
  await user.click(await screen.findByText('newapi_group_tag_update'))
  expect(screen.getByText('Save the complete tag configuration.')).toBeVisible()
  const displayedSchema = screen.getByText(/"match_keywords":/)
  expect(displayedSchema).toBeVisible()
  expect(JSON.parse(displayedSchema.textContent ?? '')).toEqual(inputSchema)
})

test('调用记录显示成功失败、request_id 和时长，翻页仅请求目标分页', async () => {
  const user = userEvent.setup()
  const pages: unknown[] = []
  ;({ client } = renderMcp(<LogsPanel />, async (config) => {
    pages.push(config.params)
    const first = config.params.p === 1
    return response(config, {
      items: [
        {
          id: first ? 1 : 2,
          key_id: 17,
          tool_name: 'newapi_context',
          request_id: first ? 'req-first' : 'req-second',
          success: first,
          duration_ms: first ? 35 : 12,
          created_at: 1700000000,
        },
      ],
      total: 21,
    })
  }))
  expect(await screen.findByText('req-first')).toBeVisible()
  expect(screen.getByText('Success')).toBeVisible()
  expect(screen.getByText('35')).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'Next' }))
  expect(await screen.findByText('req-second')).toBeVisible()
  expect(screen.getByText('Failed')).toBeVisible()
  expect(screen.getByText('12')).toBeVisible()
  expect(screen.queryByText('req-first')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  expect(pages).toEqual([
    { p: 1, page_size: 20 },
    { p: 2, page_size: 20 },
  ])
})

test('翻页加载期间禁用分页控件，响应到达后展示空状态', async () => {
  const user = userEvent.setup()
  let finish: (() => void) | undefined
  ;({ client } = renderMcp(<LogsPanel />, async (config) => {
    if (config.params.p === 2) {
      await new Promise<void>((resolve) => {
        finish = resolve
      })
    }
    return response(config, { items: [], total: 21 })
  }))
  await screen.findByText('No MCP calls yet')
  await user.click(screen.getByRole('button', { name: 'Next' }))
  expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  finish?.()
  expect(await screen.findByText('No MCP calls yet')).toBeVisible()
})

test('工具名称和描述中的 HTML 按文本渲染，不执行服务端标记', async () => {
  ;({ client } = renderMcp(<ToolsPanel />, async (config) =>
    response(config, [
      { ...tools[0], description: '<img src=x onerror=alert(1)>' },
    ])
  ))
  fireEvent.click(await screen.findByText('newapi_context'))
  expect(screen.getByText('<img src=x onerror=alert(1)>')).toBeVisible()
  expect(screen.queryByRole('img')).not.toBeInTheDocument()
})
