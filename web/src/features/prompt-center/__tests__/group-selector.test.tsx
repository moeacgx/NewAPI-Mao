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
import { fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import type { GroupDetail } from '@/features/system-settings/types'

import { PromptGroupSelector } from '../components/prompt-group-selector'
import type { PromptGroupTag } from '../types'

vi.mock('@lobehub/icons', () => ({}))

const groupDefaults = {
  description: '',
  ratio: 1,
  user_selectable: true,
  exclusive: false,
  status: 1,
  auto_enabled: true,
  auto_order: 0,
}
const groups: GroupDetail[] = [
  { ...groupDefaults, id: 12, name: 'Premium models', code: 'premium-code' },
  { ...groupDefaults, id: 21, name: 'Review models', code: 'review-code' },
  { ...groupDefaults, id: 33, name: 'Writing models', code: 'writing-code' },
  { ...groupDefaults, id: 45, name: 'Other models', code: 'other-code' },
]
const tags: PromptGroupTag[] = [
  { id: 1, name: 'GPT', icons: [], group_ids: [12, 21, 999] },
  { id: 2, name: 'Claude', icons: [], group_ids: [33] },
]

function Selection(props: {
  tags?: PromptGroupTag[]
  initial?: number[]
  onChange?: (ids: number[]) => void
}) {
  const [selected, setSelected] = useState(props.initial ?? [])
  return (
    <PromptGroupSelector
      groups={groups}
      tags={props.tags ?? tags}
      selectedIds={selected}
      onChange={(ids) => {
        setSelected(ids)
        props.onChange?.(ids)
      }}
    />
  )
}

describe('提示词装载白名单分组标签选择', () => {
  it('快速切换标签时只暴露当前弹层的选项', () => {
    render(<Selection />)
    fireEvent.click(screen.getByRole('button', { name: /GPT/ }))
    fireEvent.click(screen.getByRole('button', { name: /Claude/ }))
    expect(screen.getAllByRole('dialog')).toHaveLength(1)
    expect(screen.getAllByRole('checkbox')).toHaveLength(1)
    expect(
      screen.getByRole('checkbox', { name: 'Writing models' })
    ).toBeInTheDocument()
  })

  it('点击标签才显示该标签实际存在的分组，并可连续勾选多个稳定 ID', async () => {
    const change = vi.fn()
    const user = userEvent.setup()
    render(<Selection onChange={change} />)
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /GPT/ }))
    const dialog = await screen.findByRole('dialog')
    expect(
      within(dialog).queryByRole('checkbox', { name: 'Writing models' })
    ).not.toBeInTheDocument()
    expect(
      within(dialog).queryByRole('checkbox', { name: 'Other models' })
    ).not.toBeInTheDocument()
    await user.click(
      within(dialog).getByRole('checkbox', { name: 'Premium models' })
    )
    await user.click(
      within(dialog).getByRole('checkbox', { name: 'Review models' })
    )
    expect(change).toHaveBeenLastCalledWith([12, 21])
    expect(
      within(dialog).getByRole('checkbox', { name: 'Premium models' })
    ).toBeChecked()
    expect(
      within(dialog).getByRole('checkbox', { name: 'Review models' })
    ).toBeChecked()
    expect(screen.queryByText('premium-code')).not.toBeInTheDocument()
  })

  it('跨标签选择保留已有分组，移除单组不会清除其他标签的选择', async () => {
    const change = vi.fn()
    const user = userEvent.setup()
    render(<Selection initial={[12, 21]} onChange={change} />)
    await user.click(screen.getByRole('button', { name: /Claude/ }))
    await user.click(
      await screen.findByRole('checkbox', { name: 'Writing models' })
    )
    expect(change).toHaveBeenLastCalledWith([12, 21, 33])
    await user.keyboard('{Escape}')
    await user.click(
      screen.getByRole('button', { name: 'Remove group Premium models' })
    )
    expect(change).toHaveBeenLastCalledWith([21, 33])
    await user.click(screen.getByRole('button', { name: /GPT/ }))
    expect(
      await screen.findByRole('checkbox', { name: 'Premium models' })
    ).not.toBeChecked()
    expect(
      screen.getByRole('checkbox', { name: 'Review models' })
    ).toBeChecked()
  })

  it('支持按名称或 code 搜索，但选项只展示当前名称，空结果保留搜索入口', async () => {
    const user = userEvent.setup()
    render(<Selection />)
    await user.click(screen.getByRole('button', { name: /All groups/ }))
    const dialog = await screen.findByRole('dialog')
    const search = within(dialog).getByRole('textbox', {
      name: 'Search groups...',
    })
    await user.type(search, 'Premium')
    expect(
      within(dialog).getByRole('checkbox', { name: 'Premium models' })
    ).toBeInTheDocument()
    expect(
      within(dialog).queryByRole('checkbox', { name: 'Review models' })
    ).not.toBeInTheDocument()
    await user.clear(search)
    await user.type(search, 'review-code')
    expect(
      within(dialog).getByRole('checkbox', { name: 'Review models' })
    ).toBeInTheDocument()
    expect(within(dialog).queryByText('review-code')).not.toBeInTheDocument()
    await user.clear(search)
    await user.type(search, 'missing-name')
    expect(within(dialog).getByText('No available groups')).toBeInTheDocument()
    expect(search).toHaveValue('missing-name')
  })

  it('未分类标签仅显示没有标签的分组', async () => {
    const user = userEvent.setup()
    render(<Selection />)
    await user.click(screen.getByRole('button', { name: /Untagged/ }))
    const dialog = await screen.findByRole('dialog')
    expect(
      within(dialog).getByRole('checkbox', { name: 'Other models' })
    ).toBeInTheDocument()
    expect(within(dialog).getAllByRole('checkbox')).toHaveLength(1)
  })

  it('失效分组保留可移除身份，不加入可选分组', () => {
    const change = vi.fn()
    render(<Selection tags={[]} initial={[12, 999]} onChange={change} />)
    expect(
      screen.queryByRole('checkbox', { name: 'Deleted (999)' })
    ).not.toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', { name: 'Remove group Deleted (999)' })
    )
    expect(change).toHaveBeenLastCalledWith([12])
    expect(
      screen.getByRole('checkbox', { name: 'Premium models' })
    ).toBeChecked()
  })

  it('没有标签时回退全量搜索列表，仍支持多选', () => {
    const change = vi.fn()
    render(<Selection tags={[]} onChange={change} />)
    expect(screen.getAllByRole('checkbox')).toHaveLength(4)
    expect(
      screen.getByRole('textbox', { name: 'Search groups...' })
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('checkbox', { name: 'Premium models' }))
    fireEvent.click(screen.getByRole('checkbox', { name: 'Writing models' }))
    expect(change).toHaveBeenLastCalledWith([12, 33])
  })

  it('分组列表为空时显示明确的空状态', () => {
    render(
      <PromptGroupSelector
        groups={[]}
        tags={[]}
        selectedIds={[]}
        onChange={vi.fn()}
      />
    )
    expect(screen.getByText('No available groups')).toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  })
})
