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
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test } from 'vitest'

import type { TopNavLink } from '../../types'
import { TopNav } from '../top-nav'

afterEach(cleanup)

async function renderNavigation(links: TopNavLink[]) {
  const route = createRootRoute({
    component: () => <TopNav links={links} />,
  })
  const router = createRouter({
    routeTree: route,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(<RouterProvider router={router} />)
}

test('导航菜单有可访问名称，键盘展开并保留完整长标题，Escape 返回触发按钮', async () => {
  await renderNavigation([
    { title: '主页', href: '/' },
    {
      title: '图片异步生成文档与渠道可观测性使用说明',
      href: 'https://example.invalid/docs',
      external: true,
    },
  ])
  const user = userEvent.setup()
  const trigger = screen.getByRole('button', { name: 'Navigation menu' })
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
  trigger.focus()
  await user.keyboard('{Enter}')
  const item = await screen.findByRole('menuitem', {
    name: '图片异步生成文档与渠道可观测性使用说明',
  })
  expect(trigger).toHaveAttribute('aria-expanded', 'true')
  expect(item).toBeVisible()
  expect(item).toHaveAttribute('href', 'https://example.invalid/docs')
  expect(item).toHaveAttribute('target', '_blank')
  expect(item).toHaveAttribute('rel', 'noopener noreferrer')
  await user.keyboard('{End}')
  await waitFor(() => expect(item).toHaveFocus())
  await user.keyboard('{Escape}')
  await waitFor(() => expect(screen.queryByRole('menu')).toBeNull())
  expect(trigger).toHaveFocus()
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
})

test('禁用项不能通过键盘激活，仍可继续导航到可用链接', async () => {
  await renderNavigation([
    { title: '主页', href: '/', isActive: true },
    {
      title: '不可用文档',
      href: 'https://example.invalid/disabled',
      external: true,
      disabled: true,
    },
    { title: '可用文档', href: 'https://example.invalid/docs', external: true },
  ])
  const user = userEvent.setup()
  const trigger = screen.getByRole('button', { name: 'Navigation menu' })
  trigger.focus()
  await user.keyboard('{ArrowDown}')
  const unavailable = await screen.findByRole('menuitem', {
    name: '不可用文档',
  })
  expect(unavailable).toHaveAttribute('aria-disabled', 'true')
  await user.keyboard('{Home}{ArrowDown}')
  await waitFor(() => expect(unavailable).toHaveFocus())
  await user.keyboard('{Enter}')
  expect(trigger).toHaveAttribute('aria-expanded', 'true')
  await user.keyboard('{ArrowDown}')
  await waitFor(() =>
    expect(screen.getByRole('menuitem', { name: '可用文档' })).toHaveFocus()
  )
  expect(screen.getByRole('link', { name: '主页' })).toHaveAttribute(
    'aria-current',
    'page'
  )
})

test('没有导航链接时不显示无内容菜单入口', async () => {
  await renderNavigation([])
  expect(screen.queryByRole('button', { name: 'Navigation menu' })).toBeNull()
  expect(screen.queryByRole('navigation')).toBeNull()
})
