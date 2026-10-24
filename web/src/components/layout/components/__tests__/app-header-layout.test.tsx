/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ComponentProps } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { SidebarProvider } from '@/components/ui/sidebar'

import { AppHeader } from '../app-header'

const { setOpen } = vi.hoisted(() => ({ setOpen: vi.fn() }))
// 隔离导航及搜索弹层的应用上下文，保留真实页头、按钮与可访问名称。
vi.mock('@tanstack/react-router', () => ({
  Link: ({ to, ...props }: ComponentProps<'a'> & { to: string }) => (
    <a href={to} {...props} />
  ),
}))
vi.mock('@/context/search-provider', () => ({ useSearch: () => ({ setOpen }) }))
vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), put: vi.fn() },
  getStatus: vi.fn().mockResolvedValue({ system_name: 'NewAPI-Mao' }),
  getNotice: vi.fn().mockResolvedValue({ success: true, data: '' }),
}))

describe('Default 页头窄屏布局', () => {
  it('搜索入口在手机压缩为图标宽度，保留可访问名称和打开搜索行为', async () => {
    const user = userEvent.setup()
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <SidebarProvider>
          <AppHeader
            showTopNav={false}
            showNotifications={false}
            showConfigDrawer={false}
            showProfileDropdown={false}
          />
        </SidebarProvider>
      </QueryClientProvider>
    )
    const search = screen.getByRole('button', { name: 'Search' })
    expect(search).toHaveClass(
      'max-sm:w-8',
      'max-sm:flex-none',
      'max-sm:[&>span]:hidden'
    )
    expect(search).not.toHaveAttribute('aria-hidden')
    await user.click(search)
    expect(setOpen).toHaveBeenCalledWith(true)
    expect(
      screen.getByRole('button', { name: 'Change language' })
    ).toBeVisible()
    expect(screen.getByRole('link', { name: 'Go to home' })).toBeVisible()
  })
})
