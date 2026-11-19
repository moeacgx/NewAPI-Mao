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
import { useRef, useState } from 'react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test } from 'vitest'

import { ApiKeyGroupCombobox } from '../api-key-group-combobox'
import { GroupTagCards } from '../group-tag-cards'

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
    expect(within(list).queryByText('Auto')).not.toBeInTheDocument()
    fireEvent.click(within(list).getByText('当前标准组'))
    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      'stable_b,stable_a'
    )
    expect(screen.getByRole('combobox')).not.toHaveTextContent('当前专业组')
    expect(screen.getByRole('combobox')).not.toHaveTextContent('当前标准组')
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

  test('tag card opens the picker with its filter and returns focus on dismiss', async () => {
    const user = userEvent.setup()
    const longName = '超长标签名称'.repeat(12)
    const longTags = [
      { id: 1, name: longName, group_ids: [11], icons: [] },
      { id: 3, name: '专业平台', group_ids: [12], icons: [] },
    ]
    function CardsHarness() {
      const [tag, setTag] = useState('all')
      const [open, setOpen] = useState(false)
      const returnFocus = useRef<HTMLElement | null>(null)
      const cardsRef = useRef<HTMLDivElement | null>(null)
      const [selected, setSelected] = useState(['stable_b'])
      return (
        <>
          <GroupTagCards
            tags={longTags}
            options={options}
            value={tag}
            onChange={setTag}
            expanded={open}
            containerRef={cardsRef}
            onActivate={(element) => {
              returnFocus.current = element
              setOpen(true)
            }}
          />
          <ApiKeyGroupCombobox
            options={options}
            tags={longTags}
            activeTag={tag}
            onActiveTagChange={setTag}
            hideTagFilter
            open={open}
            onOpenChange={setOpen}
            returnFocusRef={returnFocus}
            anchorRef={cardsRef}
            selectedValues={selected}
            onValueChange={(value) =>
              setSelected((current) =>
                current.includes(value)
                  ? current.filter((item) => item !== value)
                  : [...current, value]
              )
            }
          />
          <output aria-label='Selected codes'>{selected.join(',')}</output>
        </>
      )
    }
    render(<CardsHarness />)
    const group = screen.getByRole('group', { name: 'Group tags' })
    expect(group.parentElement).toHaveClass('@container')
    expect(group).toHaveClass('@md:grid-cols-3')

    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
    const longCard = screen.getByRole('button', { name: new RegExp(longName) })
    longCard.focus()
    await user.keyboard('{Enter}')
    expect(longCard).toHaveAttribute('aria-pressed', 'true')
    expect(longCard).toHaveAttribute('aria-expanded', 'true')
    const list = await screen.findByRole('listbox')
    expect(within(list).getByText('当前标准组')).toBeVisible()
    expect(within(list).queryByText('当前专业组')).not.toBeInTheDocument()
    await user.click(within(list).getByText('当前标准组'))
    await waitFor(() =>
      expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    )
    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      'stable_b,stable_a'
    )
    await waitFor(() => expect(longCard).toHaveFocus())

    const proCard = screen.getByRole('button', { name: /专业平台/ })
    await user.click(proCard)
    const proList = await screen.findByRole('listbox')
    expect(within(proList).getByText('当前专业组')).toBeVisible()
    expect(within(proList).queryByText('当前标准组')).not.toBeInTheDocument()
    await user.keyboard('{Escape}')
    await waitFor(() =>
      expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    )
    await waitFor(() => expect(proCard).toHaveFocus())
    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      'stable_b,stable_a'
    )
  })

  test('Auto is the first option only under All groups, including while the popup stays open', async () => {
    render(<Harness />)
    fireEvent.click(screen.getByRole('combobox'))
    const names = () =>
      within(screen.getByRole('listbox'))
        .getAllByRole('option')
        .map((option) => option.textContent)
    expect(names()[0]).toContain('Auto')
    fireEvent.click(screen.getByRole('button', { name: '模型平台' }))
    expect(names().join('|')).not.toContain('Auto')
    fireEvent.click(screen.getByRole('button', { name: 'All groups' }))
    expect(names()[0]).toContain('Auto')
    fireEvent.change(screen.getByPlaceholderText('Search...'), {
      target: { value: '专业' },
    })
    expect(names().join('|')).not.toContain('Auto')
    expect(names()).toHaveLength(1)
  })
})
