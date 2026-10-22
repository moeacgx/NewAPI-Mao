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
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { TokenPromptBadge } from '../components/token-prompt-badge'
import { TokenPromptDialog } from '../components/token-prompt-dialog'

vi.mock('@/lib/api', () => ({ api: { get: vi.fn(), put: vi.fn() } }))

const template = {
  id: 7,
  title: 'Review',
  description: 'Review code',
  content: 'Check correctness',
  enabled: true,
  sort_order: 0,
  version: 1,
  group_ids: [12],
  group_details: [{ id: 12, name: 'Premium models' }],
}
const token = {
  id: 3,
  name: 'Development',
  prompt_count: 1,
  enabled_prompt_count: 1,
}

function renderWithQuery(children: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
}

beforeEach(() => {
  vi.mocked(api.get).mockReset()
  vi.mocked(api.put).mockReset()
  vi.mocked(api.get).mockResolvedValue({
    data: { success: true, data: [{ template, enabled: true, sort_order: 0 }] },
  })
  vi.mocked(api.put).mockResolvedValue({ data: { success: true } })
})

describe('令牌技能徽标', () => {
  it('键盘聚焦后可以按回车打开安装详情', async () => {
    const user = userEvent.setup()
    renderWithQuery(<TokenPromptBadge token={token} />)
    await user.tab()
    expect(
      screen.getByRole('button', { name: 'Manage installed skills (1)' })
    ).toHaveFocus()
    await user.keyboard('{Enter}')
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })
  it('未安装时不显示徽标', () => {
    renderWithQuery(<TokenPromptBadge token={{ ...token, prompt_count: 0 }} />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('显示安装数量，点击后展示当前名称和白名单分组', async () => {
    renderWithQuery(<TokenPromptBadge token={token} />)
    fireEvent.click(
      screen.getByRole('button', { name: 'Manage installed skills (1)' })
    )
    const dialog = await screen.findByRole('dialog')
    expect(await within(dialog).findByText('Review')).toBeInTheDocument()
    expect(within(dialog).getByText(/Premium models/)).toBeInTheDocument()
    expect(api.get).toHaveBeenCalledWith('/api/token/3/prompts')
  })

  it('全部停用时保留数量并提供可访问的停用状态', () => {
    renderWithQuery(
      <TokenPromptBadge token={{ ...token, enabled_prompt_count: 0 }} />
    )
    const button = screen.getByRole('button', {
      name: 'Manage installed skills (1), all inactive',
    })
    expect(button).toHaveAttribute('data-active', 'false')
    expect(button).toHaveTextContent('Skills · 1')
  })
})

describe('令牌提示词管理', () => {
  it('用户管理已安装项时不显示提示词正文', async () => {
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    expect(await screen.findByText('Review')).toBeInTheDocument()
    expect(screen.queryByText('Check correctness')).not.toBeInTheDocument()
    expect(screen.queryByText('View prompt content')).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Uninstall Review' })
    ).toBeInTheDocument()
  })

  it('第九项安装禁用保存，移除后可以继续保存', async () => {
    const installed = Array.from({ length: 8 }, (_, index) => ({
      template: { ...template, id: index + 1, title: `Prompt ${index + 1}` },
      enabled: true,
      sort_order: index,
    }))
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: true, data: installed },
    })
    renderWithQuery(
      <TokenPromptDialog
        open
        onOpenChange={vi.fn()}
        token={token}
        templateToInstall={{ ...template, id: 9, title: 'New prompt' }}
      />
    )
    expect(
      await screen.findByRole('button', { name: 'Save installations' })
    ).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Uninstall Prompt 1' }))
    expect(
      screen.getByRole('button', { name: 'Save installations' })
    ).toBeEnabled()
  })
  it('多项安装保留原顺序，仅提供卸载操作', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: {
        success: true,
        data: [
          { template, enabled: true, sort_order: 0 },
          {
            template: { ...template, id: 9, title: 'Writing' },
            enabled: true,
            sort_order: 1,
          },
        ],
      },
    })
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    await screen.findByRole('button', { name: 'Uninstall Writing' })
    expect(
      screen
        .getAllByRole('region')
        .map((region) => region.getAttribute('aria-label'))
    ).toEqual(['Review', 'Writing'])
    expect(
      screen.queryByRole('button', { name: 'Move Writing up' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('checkbox', { name: 'Enable Writing' })
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Uninstall Writing' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save installations' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/token/3/prompts', {
        bindings: [{ template_id: 7, enabled: true, sort_order: 0 }],
      })
    )
  })

  it('下架且停用的安装项仍可卸载', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: {
        success: true,
        data: [
          {
            template: { ...template, enabled: false },
            enabled: false,
            sort_order: 0,
          },
        ],
      },
    })
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    expect(
      await screen.findByRole('button', { name: 'Uninstall Review' })
    ).toBeEnabled()
  })

  it('服务端返回正文时管理界面仍不渲染正文且保持安装信息', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      data: {
        success: true,
        data: [
          {
            template: { ...template, content: '中'.repeat(21846) },
            enabled: true,
            sort_order: 0,
          },
        ],
      },
    })
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    expect(
      await screen.findByRole('button', { name: 'Save installations' })
    ).toBeEnabled()
    expect(screen.queryByText('中'.repeat(20))).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Uninstall Review' })
    ).toBeEnabled()
  })

  it('新增安装保留既有安装，并通过独立接口保存稳定模板 ID', async () => {
    const candidate = { ...template, id: 9, title: 'Writing' }
    renderWithQuery(
      <TokenPromptDialog
        open
        onOpenChange={vi.fn()}
        token={token}
        templateToInstall={candidate}
      />
    )
    expect(await screen.findByText('Writing')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save installations' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/token/3/prompts', {
        bindings: [
          { template_id: 7, enabled: true, sort_order: 0 },
          { template_id: 9, enabled: true, sort_order: 1 },
        ],
      })
    )
  })

  it('卸载保持草稿，只有保存时提交', async () => {
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    await screen.findByRole('button', { name: 'Uninstall Review' })
    fireEvent.click(screen.getByRole('button', { name: 'Uninstall Review' }))
    expect(api.put).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Save installations' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/token/3/prompts', {
        bindings: [],
      })
    )
  })

  it('卸载最后一项提交空数组', async () => {
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    fireEvent.click(
      await screen.findByRole('button', { name: 'Uninstall Review' })
    )
    expect(screen.getByText('No installed prompts')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save installations' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/token/3/prompts', {
        bindings: [],
      })
    )
  })

  it('接口保存失败时保留安装草稿并允许再次提交', async () => {
    vi.mocked(api.put).mockResolvedValueOnce({
      data: { success: false, message: 'Conflict' },
    })
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    await screen.findByRole('button', { name: 'Uninstall Review' })
    fireEvent.click(screen.getByRole('button', { name: 'Uninstall Review' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save installations' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Conflict')
    expect(screen.getByText('No installed prompts')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save installations' }))
    await waitFor(() => expect(api.put).toHaveBeenCalledTimes(2))
  })

  it('加载失败时不提供空列表保存，避免意外卸载', async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new Error('Unavailable'))
    renderWithQuery(
      <TokenPromptDialog open onOpenChange={vi.fn()} token={token} />
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Failed to load installed prompts'
    )
    expect(
      screen.queryByRole('button', { name: 'Save installations' })
    ).not.toBeInTheDocument()
  })
})
