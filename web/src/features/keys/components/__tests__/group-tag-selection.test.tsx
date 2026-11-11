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
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, test } from 'vitest'

import { ApiKeyGroupCombobox } from '../api-key-group-combobox'

const options = [
  { id: 11, value: 'stable_a', label: '当前标准组', ratio: 1 },
  { id: 12, value: 'stable_b', label: '当前专业组', ratio: 2 },
  { value: 'auto', label: 'Auto' },
]
const tags = [
  { id: 1, name: '模型平台', group_ids: [11, 999], icons: [] },
  { id: 2, name: '空标签', group_ids: [999], icons: [] },
]

function Harness() {
  const [selected, setSelected] = useState(['stable_b'])
  return (
    <>
      <ApiKeyGroupCombobox
        options={options}
        tags={tags}
        selectedValues={selected}
        onValueChange={(value) => setSelected([...selected, value])}
      />
      <output aria-label='Selected codes'>{selected.join(',')}</output>
    </>
  )
}

describe('token group tag filtering', () => {
  test('filters only authorized options and preserves selections across tags', async () => {
    render(<Harness />)
    fireEvent.click(screen.getByRole('combobox'))
    fireEvent.click(screen.getByRole('button', { name: '模型平台' }))
    await waitFor(() => expect(screen.getByText('当前标准组')).toBeVisible())
    expect(
      screen.queryByRole('button', { name: '空标签' })
    ).not.toBeInTheDocument()
    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      'stable_b'
    )
    const list = screen.getByRole('listbox')
    expect(within(list).queryByText('当前专业组')).not.toBeInTheDocument()
    expect(within(list).getByText('Auto')).toBeVisible()
    fireEvent.click(within(list).getByText('当前标准组'))
    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      'stable_b,stable_a'
    )
    expect(screen.getByRole('combobox')).toHaveTextContent(
      '当前专业组, 当前标准组'
    )
  })

  test('clears tag and search filters while retaining the current group name after refresh', async () => {
    const onChange = () => undefined
    const view = render(
      <ApiKeyGroupCombobox
        options={options}
        tags={tags}
        value='stable_b'
        onValueChange={onChange}
      />
    )
    fireEvent.click(screen.getByRole('combobox'))
    fireEvent.click(screen.getByRole('button', { name: '模型平台' }))
    fireEvent.change(screen.getByPlaceholderText('Search...'), {
      target: { value: '不存在' },
    })
    expect(await screen.findByText('No group found.')).toBeInTheDocument()
    fireEvent.change(screen.getByPlaceholderText('Search...'), {
      target: { value: '' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'All groups' }))
    expect(
      within(screen.getByRole('listbox')).getByText('当前专业组')
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '模型平台' }))
    view.rerender(
      <ApiKeyGroupCombobox
        options={options.map((option) =>
          option.id === 12 ? { ...option, label: '改名后的专业组' } : option
        )}
        tags={[]}
        value='stable_b'
        onValueChange={onChange}
      />
    )
    expect(
      within(screen.getByRole('listbox')).getByText('改名后的专业组')
    ).toBeInTheDocument()
    view.rerender(
      <ApiKeyGroupCombobox
        options={options}
        tags={tags}
        value='stable_b'
        onValueChange={onChange}
      />
    )
    expect(screen.getByRole('button', { name: 'All groups' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
  })
})
