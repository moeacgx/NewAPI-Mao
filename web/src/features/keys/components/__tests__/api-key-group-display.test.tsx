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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test } from 'vitest'

import { TooltipProvider } from '@/components/ui/tooltip'

import { ApiKeyGroupCell } from '../api-key-group-cell'

describe('令牌分组展示名称', () => {
  test('名称与 code 不同，稳定 ID 关联旧 alias，改名后徽标与提示同步更新', async () => {
    const props = {
      group: 'old_alias',
      groupIds: [11],
      groupDetails: [{ id: 11, code: 'stable_a', name: 'Standard Display' }],
      crossGroupRetry: false,
      shouldReduceMotion: true,
      ratio: 2,
    }
    const view = render(
      <TooltipProvider delay={0}>
        <ApiKeyGroupCell {...props} />
      </TooltipProvider>
    )
    expect(screen.getByText('Standard Display')).toBeVisible()
    expect(screen.queryByText('old_alias')).toBeNull()
    expect(screen.queryByText('stable_a')).toBeNull()
    expect(screen.getByText('2x')).toBeVisible()
    await userEvent.hover(screen.getByText('Standard Display'))
    await waitFor(() =>
      expect(
        document.querySelector('[data-slot="tooltip-content"]')
      ).toHaveTextContent('Standard Display')
    )
    expect(
      document.querySelector('[data-slot="tooltip-content"]')
    ).not.toHaveTextContent('old_alias')
    const trigger = view.container.querySelector(
      '[data-slot="tooltip-trigger"]'
    )
    if (!trigger) throw new Error('Tooltip trigger missing')
    await userEvent.unhover(trigger)
    view.rerender(
      <TooltipProvider delay={0}>
        <ApiKeyGroupCell
          {...props}
          groupDetails={[{ id: 11, code: 'stable_a', name: 'Renamed Display' }]}
        />
      </TooltipProvider>
    )
    expect(screen.getByText('Renamed Display')).toBeVisible()
    expect(view.container).not.toHaveTextContent('Standard Display')
    await userEvent.hover(screen.getByText('Renamed Display'))
    await waitFor(() =>
      expect(
        document.querySelector('[data-slot="tooltip-content"]')
      ).toHaveTextContent('Renamed Display')
    )
  })

  test('多组折叠为第一个显示名与倍率加 +N，悬停/点击/键盘展开完整顺序并可关闭', async () => {
    const user = userEvent.setup()
    render(
      <TooltipProvider delay={0}>
        <ApiKeyGroupCell
          group='legacy_a,stable_b,deleted_group'
          groupIds={[11, 22, 33]}
          groupDetails={[
            { id: 22, code: 'stable_b', name: 'Premium Display' },
            { id: 11, code: 'stable_a', name: 'Standard Display' },
          ]}
          ratios={{ stable_a: 1.5, stable_b: 2 }}
          crossGroupRetry={false}
          shouldReduceMotion
        />
      </TooltipProvider>
    )
    const trigger = document.querySelector<HTMLElement>(
      '[data-api-key-group-cell="multiple"]'
    )
    if (!trigger) throw new Error('multi-group trigger missing')
    expect(trigger).toHaveTextContent('Standard Display')
    expect(trigger).toHaveTextContent('1.5x')
    expect(trigger).toHaveTextContent('+2')
    expect(trigger).not.toHaveTextContent('Premium Display')
    expect(trigger).not.toHaveTextContent('legacy_a')
    const popup = () => document.querySelector('[data-slot="popover-content"]')

    // 鼠标悬停预览
    await user.hover(trigger)
    await waitFor(() => expect(popup()).not.toBeNull())
    const items = [...(popup()?.querySelectorAll('li') ?? [])].map(
      (li) => li.textContent
    )
    expect(items).toEqual([
      '1. Standard Display1.5x',
      '2. Premium Display2x',
      '3. deleted_group',
    ])
    expect(popup()).not.toHaveTextContent('legacy_a')
    expect(popup()).not.toHaveTextContent('stable_b')
    await user.unhover(trigger)
    await waitFor(() => expect(popup()).toBeNull())

    // 点击（触屏同路径）固定展开，关闭按钮收起
    await user.click(trigger)
    await waitFor(() => expect(popup()).not.toBeNull())
    await user.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(popup()).toBeNull())

    // 键盘：Enter 打开，Escape 关闭
    trigger.focus()
    await user.keyboard('{Enter}')
    await waitFor(() => expect(popup()).not.toBeNull())
    await user.keyboard('{Escape}')
    await waitFor(() => expect(popup()).toBeNull())
  })

  test('缺少详情名称复用批量分组映射，无法解析及已删除分组回退标识', () => {
    const props = {
      group: 'legacy_a,deleted_group',
      groupIds: [11, 99],
      groupDetails: [],
      groups: { stable_a: { id: 11, code: 'stable_a', name: 'Current Name' } },
      crossGroupRetry: false,
      shouldReduceMotion: true,
    }
    const view = render(<ApiKeyGroupCell {...props} />)
    const trigger = () =>
      view.container.querySelector('[data-api-key-group-cell="multiple"]')
    expect(trigger()).toHaveAccessibleName(
      'Selected groups: Current Name, deleted_group'
    )
    view.rerender(<ApiKeyGroupCell {...props} groupIds={[]} groups={{}} />)
    expect(trigger()).toHaveAccessibleName(
      'Selected groups: legacy_a, deleted_group'
    )
  })

  test('旧接口无 ID 时按 code 解析，已知 ID 冲突不能显示其他分组的名称', () => {
    const props = {
      group: 'stable_a',
      groupDetails: [{ id: 11, code: 'stable_a', name: 'Current Name' }],
      crossGroupRetry: false,
      shouldReduceMotion: true,
    }
    const view = render(<ApiKeyGroupCell {...props} />)
    expect(screen.getByText('Current Name')).toBeVisible()
    view.rerender(<ApiKeyGroupCell {...props} groupIds={[99]} />)
    expect(screen.getByText('stable_a')).toBeVisible()
    expect(screen.queryByText('Current Name')).toBeNull()
  })

  test('继承分组与空旧数据保留 User Group，不因详情显示普通组', () => {
    const view = render(
      <ApiKeyGroupCell
        group={null}
        groupDetails={[{ id: 11, code: 'stable_a', name: 'Standard Display' }]}
        crossGroupRetry={false}
        shouldReduceMotion
      />
    )
    expect(view.container).toHaveTextContent('User Group')
    expect(view.container).not.toHaveTextContent('Standard Display')
  })
})
