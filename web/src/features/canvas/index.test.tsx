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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test, vi } from 'vitest'

import { CanvasLauncher } from './index'

const { getUserGroupsWithDefault } = vi.hoisted(() => ({
  getUserGroupsWithDefault: vi.fn(),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({
    status: {
      SidebarModulesAdmin: JSON.stringify({
        chat: { canvasOrigin: 'https://canvas.example.com' },
      }),
    },
  }),
}))

vi.mock('@/features/playground/api', () => ({
  getUserGroupsWithDefault: (...args: unknown[]) =>
    getUserGroupsWithDefault(...args),
}))

beforeEach(() => {
  getUserGroupsWithDefault.mockReset()
  getUserGroupsWithDefault.mockResolvedValue({
    groups: [
      { label: '当前名称', value: 'stable-code', ratio: 1 },
      { label: '文本显示名', value: 'text-code', ratio: 1 },
      { label: '视频显示名', value: 'video-code', ratio: 1 },
      { label: '自动选择', value: 'auto', ratio: 1 },
    ],
    defaultGroup: 'stable-code',
    textGroup: 'text-code',
    imageGroup: 'missing-code',
    audioGroup: '',
    videoGroup: 'video-code',
  })
})

test('可用预设按名称选中，清空后不传对应参数', async () => {
  const user = userEvent.setup()
  const open = vi.spyOn(window, 'open').mockImplementation(() => null)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const view = render(
    <QueryClientProvider client={client}>
      <CanvasLauncher />
    </QueryClientProvider>
  )
  try {
    const modelGroup = screen.getByRole('combobox', { name: 'Model Group' })
    await waitFor(() => expect(modelGroup).toHaveTextContent('当前名称'))
    expect(modelGroup).not.toHaveTextContent('stable-code')
    expect(screen.queryByText('No group found.')).toBeNull()
    expect(
      screen.getByRole('combobox', { name: 'Text Group' })
    ).toHaveTextContent('文本显示名')
    expect(
      screen.getByRole('combobox', { name: 'Image Group' })
    ).toHaveTextContent('Follow default group')
    expect(
      screen.getByRole('combobox', { name: 'Audio Group' })
    ).toHaveTextContent('Follow default group')
    expect(
      screen.getByRole('combobox', { name: 'Video Group' })
    ).toHaveTextContent('视频显示名')

    await user.click(screen.getByRole('combobox', { name: 'Text Group' }))
    expect(
      await screen.findByRole('option', { name: '文本显示名' })
    ).toBeTruthy()
    expect(screen.queryByRole('option', { name: 'text-code' })).toBeNull()
    await user.click(
      await screen.findByRole('option', { name: 'Follow default group' })
    )

    await user.click(screen.getByRole('button', { name: 'Open in new tab' }))
    expect(open).toHaveBeenCalledTimes(1)
    const url = new URL(String(open.mock.calls[0]?.[0]))
    expect(url.origin).toBe('https://canvas.example.com')
    expect(url.searchParams.get('mode')).toBe('newapi')
    expect(url.searchParams.get('group')).toBe('stable-code')
    expect(url.searchParams.get('videoGroup')).toBe('video-code')
    expect(url.searchParams.has('textGroup')).toBe(false)
    expect(url.searchParams.has('imageGroup')).toBe(false)
    expect(url.searchParams.has('audioGroup')).toBe(false)
    expect(open.mock.calls[0]?.[1]).toBe('_blank')
    expect(open.mock.calls[0]?.[2]).toBe('noopener')
  } finally {
    view.unmount()
    open.mockRestore()
    client.clear()
  }
})
