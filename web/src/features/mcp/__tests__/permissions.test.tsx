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
import { cleanup, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test } from 'vitest'

import { api } from '@/lib/api'

import { KeysPanel } from '../components/keys-panel'
import { key, renderMcp, response, tools } from './fixtures'

const originalAdapter = api.defaults.adapter
let client: QueryClient | undefined
afterEach(() => {
  cleanup()
  client?.clear()
  api.defaults.adapter = originalAdapter
})

test('数百项授权仅显示数量，弹窗可搜索原始名称且不可编辑', async () => {
  const user = userEvent.setup()
  const names = Array.from(
    { length: 368 },
    (_, index) => `historical_tool_${index}`
  )
  ;({ client } = renderMcp(<KeysPanel />, async (config) =>
    response(
      config,
      config.url?.endsWith('/tools')
        ? tools
        : { items: [{ ...key, tools: names }], total: 1 }
    )
  ))
  const table = await screen.findByRole('table')
  expect(within(table).getByText('368 authorized tools')).toBeVisible()
  expect(within(table).queryByText(/historical_tool/)).not.toBeInTheDocument()
  await user.click(
    within(table).getByRole('button', { name: 'View permissions' })
  )
  const dialog = screen.getByRole('dialog')
  expect(within(dialog).queryAllByRole('checkbox')).toHaveLength(0)
  expect(
    within(dialog).queryByText('Read only', { selector: '[data-slot="badge"]' })
  ).not.toBeInTheDocument()
  await user.type(
    within(dialog).getByRole('textbox', { name: 'Search tools' }),
    'historical_tool_367'
  )
  expect(within(dialog).getByText('historical_tool_367')).toBeVisible()
  expect(
    within(dialog).queryByText('historical_tool_0')
  ).not.toBeInTheDocument()
  expect(
    within(dialog).getByText('Unknown access', {
      selector: '[data-slot="badge"]',
    })
  ).toBeVisible()
  await user.keyboard('{Escape}')
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  expect(
    within(table).getByRole('button', { name: 'View permissions' })
  ).toHaveFocus()
})

test('目录失败仍显示保存名称，未知项不会归入只读筛选', async () => {
  const user = userEvent.setup()
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.url?.endsWith('/tools')) return response(config, null, false)
    return response(config, {
      items: [{ ...key, tools: ['removed_write_tool'] }],
      total: 1,
    })
  }))
  await user.click(
    await screen.findByRole('button', { name: 'View permissions' })
  )
  const dialog = screen.getByRole('dialog')
  expect(
    await within(dialog).findByText(
      'Tool catalog unavailable. Saved tool names are still shown; access types cannot be verified.'
    )
  ).toBeVisible()
  expect(within(dialog).getByText('removed_write_tool')).toBeVisible()
  await user.selectOptions(within(dialog).getByLabelText('Access'), 'read')
  expect(
    within(dialog).queryByText('removed_write_tool')
  ).not.toBeInTheDocument()
  await user.selectOptions(within(dialog).getByLabelText('Access'), 'unknown')
  expect(within(dialog).getByText('removed_write_tool')).toBeVisible()
})

test('弹窗仅包含已授权项，分类筛选保留准确读写标记', async () => {
  const user = userEvent.setup()
  ;({ client } = renderMcp(<KeysPanel />, async (config) =>
    response(
      config,
      config.url?.endsWith('/tools')
        ? tools
        : {
            items: [{ ...key, tools: ['update_channel', 'removed_tool'] }],
            total: 1,
          }
    )
  ))
  await screen.findByRole('checkbox', { name: /update_channel/ })
  await user.click(screen.getByRole('button', { name: 'View permissions' }))
  const dialog = screen.getByRole('dialog')
  expect(within(dialog).queryByText('newapi_context')).not.toBeInTheDocument()
  expect(within(dialog).getByText('removed_tool')).toBeVisible()
  await user.selectOptions(
    within(dialog).getByLabelText('Category'),
    'channels'
  )
  expect(within(dialog).getByText('update_channel')).toBeVisible()
  expect(
    within(dialog).getByText('Write access', {
      selector: '[data-slot="badge"]',
    })
  ).toBeVisible()
  expect(within(dialog).queryByText('removed_tool')).not.toBeInTheDocument()
})

test('目录失败选择未知分类后重试成功，分类回退全部并显示已授权工具', async () => {
  const user = userEvent.setup()
  let available = false
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.url?.endsWith('/tools')) {
      return response(config, available ? tools : null, available)
    }
    return response(config, {
      items: [{ ...key, tools: ['newapi_context', 'update_channel'] }],
      total: 1,
    })
  }))
  await user.click(
    await screen.findByRole('button', { name: 'View permissions' })
  )
  const dialog = screen.getByRole('dialog')
  const retry = await within(dialog).findByRole('button', { name: 'Retry' })
  await user.selectOptions(
    within(dialog).getByLabelText('Category'),
    '__unknown'
  )
  expect(within(dialog).getByLabelText('Category')).toHaveValue('__unknown')
  available = true
  await user.click(retry)
  expect(
    await within(dialog).findByText('Write access', {
      selector: '[data-slot="badge"]',
    })
  ).toBeVisible()
  expect(within(dialog).getByLabelText('Category')).toHaveValue('')
  expect(within(dialog).getByText('newapi_context')).toBeVisible()
  expect(within(dialog).getByText('update_channel')).toBeVisible()
  expect(
    within(dialog).queryByRole('option', { name: 'Unavailable tools' })
  ).not.toBeInTheDocument()
  expect(
    within(dialog).queryByText('No matching tools')
  ).not.toBeInTheDocument()
})

test('重试成功后仍有历史工具时保留有效未知分类及保存名称', async () => {
  const user = userEvent.setup()
  let available = false
  ;({ client } = renderMcp(<KeysPanel />, async (config) => {
    if (config.url?.endsWith('/tools')) {
      return response(config, available ? tools : null, available)
    }
    return response(config, {
      items: [{ ...key, tools: ['update_channel', 'removed_tool'] }],
      total: 1,
    })
  }))
  await user.click(
    await screen.findByRole('button', { name: 'View permissions' })
  )
  const dialog = screen.getByRole('dialog')
  const retry = await within(dialog).findByRole('button', { name: 'Retry' })
  await user.selectOptions(
    within(dialog).getByLabelText('Category'),
    '__unknown'
  )
  available = true
  await user.click(retry)
  await waitFor(() =>
    expect(
      within(dialog).queryByRole('button', { name: 'Retry' })
    ).not.toBeInTheDocument()
  )
  expect(within(dialog).getByLabelText('Category')).toHaveValue('__unknown')
  expect(within(dialog).getByText('removed_tool')).toBeVisible()
  expect(within(dialog).queryByText('update_channel')).not.toBeInTheDocument()
  expect(
    within(dialog).getByText('Unknown access', {
      selector: '[data-slot="badge"]',
    })
  ).toBeVisible()
})
