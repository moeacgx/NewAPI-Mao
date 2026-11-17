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
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { AutoGroupOrderEditor } from '../auto-group-order-editor'

afterEach(cleanup)

test('Auto顺序显示当前名称和ARIA，重排仍提交稳定code，未知组保留回退', () => {
  const onChange = vi.fn()
  const { container } = render(
    <AutoGroupOrderEditor
      value={['stable_a', 'stable_b', 'deleted']}
      mode='custom'
      options={[
        { value: 'stable_a', label: 'Standard Display', id: 11 },
        { value: 'stable_b', label: 'Premium Display', id: 22 },
      ]}
      globalOptions={[]}
      maxCount={0}
      onChange={onChange}
    />
  )
  expect(screen.getByText('Standard Display')).toBeVisible()
  expect(screen.getByText('Premium Display')).toBeVisible()
  expect(screen.getByText('deleted')).toBeVisible()
  fireEvent.click(
    screen.getByRole('button', { name: 'Move Premium Display up' })
  )
  expect(onChange).toHaveBeenLastCalledWith({
    groups: ['stable_b', 'stable_a', 'deleted'],
    mode: 'custom',
  })
  fireEvent.keyDown(
    screen.getByRole('button', { name: 'Drag Standard Display to reorder' }),
    { key: 'ArrowDown' }
  )
  expect(onChange).toHaveBeenLastCalledWith({
    groups: ['stable_b', 'stable_a', 'deleted'],
    mode: 'custom',
  })
  expect(container.textContent).not.toContain('stable_a')
  for (const element of container.querySelectorAll('[aria-label],[title]')) {
    expect(
      `${element.getAttribute('aria-label')} ${element.getAttribute('title')}`
    ).not.toMatch(/stable_a|stable_b/)
  }
})

test('名称刷新立即更新全局顺序标题和自定义ARIA，倍率与业务标识不变', () => {
  const onChange = vi.fn()
  const options = [
    {
      value: 'stable_a',
      label: 'Current Name',
      desc: 'stable_a',
      ratio: 2,
      id: 11,
    },
  ]
  const props = {
    value: ['stable_a'],
    options,
    globalOptions: options,
    maxCount: 0,
    onChange,
  }
  const view = render(<AutoGroupOrderEditor {...props} mode='inherit' />)
  expect(screen.getByText('Current Name')).toBeVisible()
  expect(
    view.container.querySelector('[data-slot="global-auto-order-chip"]')
  ).toHaveAttribute('title', 'Current Name')
  expect(view.container.textContent).not.toContain('stable_a')
  const renamed = [{ ...options[0], label: 'Renamed Display' }]
  view.rerender(
    <AutoGroupOrderEditor
      {...props}
      options={renamed}
      globalOptions={renamed}
      mode='custom'
    />
  )
  expect(screen.getByText('Renamed Display')).toBeVisible()
  fireEvent.click(
    screen.getByRole('button', { name: 'Remove Renamed Display' })
  )
  expect(onChange).toHaveBeenLastCalledWith({ groups: [], mode: 'custom' })
  expect(
    screen.queryByRole('button', { name: /Current Name|stable_a/ })
  ).toBeNull()
})
