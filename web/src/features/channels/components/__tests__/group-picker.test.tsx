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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, test } from 'vitest'

import {
  ChannelGroupPicker,
  type ChannelGroupOption,
} from '../channel-group-picker'

// Display names deliberately differ from codes; `legacy_code` has no ID.
const options: ChannelGroupOption[] = [
  { id: 11, value: 'code_std', label: '当前标准组' },
  { id: 12, value: 'code_pro', label: '当前专业组' },
  { id: 13, value: 'code_cn', label: '国产直连组' },
  { value: 'legacy_code', label: 'legacy_code' },
]
const tags = [
  { id: 1, name: '海外平台', icons: [], group_ids: [11, 12] },
  { id: 2, name: '国产模型', icons: [], group_ids: [13, 999] },
  { id: 3, name: '空标签', icons: [], group_ids: [999] },
]

function Harness(props: {
  initial?: string[]
  tags?: typeof tags
  opts?: ChannelGroupOption[]
}) {
  const [selected, setSelected] = useState(props.initial ?? [])
  return (
    <>
      <ChannelGroupPicker
        options={props.opts ?? options}
        tags={props.tags ?? tags}
        selected={selected}
        onChange={setSelected}
        placeholder='Select groups'
      />
      <output aria-label='Selected codes'>{selected.join(',')}</output>
    </>
  )
}

const listedNames = () =>
  within(screen.getByRole('listbox'))
    .getAllByRole('option')
    .map((option) => option.textContent)

describe('channel group tag picker', () => {
  test('tag card opens a multi-select list filtered by group ID and picks accumulate across tags', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    expect(
      screen.queryByRole('button', { name: /空标签/ })
    ).not.toBeInTheDocument()
    const overseas = screen.getByRole('button', { name: /海外平台/ })
    await user.click(overseas)
    expect(overseas).toHaveAttribute('aria-expanded', 'true')
    expect(listedNames()).toEqual(['当前标准组', '当前专业组'])

    await user.click(screen.getByRole('option', { name: '当前专业组' }))
    await user.click(screen.getByRole('option', { name: '当前标准组' }))
    // The list stays open for multiple picks.
    expect(screen.getByRole('option', { name: '当前标准组' })).toHaveAttribute(
      'aria-checked',
      'true'
    )
    await user.keyboard('{Escape}')
    await waitFor(() =>
      expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    )
    await waitFor(() => expect(overseas).toHaveFocus())

    await user.click(screen.getByRole('button', { name: /国产模型/ }))
    expect(listedNames()).toEqual(['国产直连组'])
    await user.click(screen.getByRole('option', { name: '国产直连组' }))

    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      'code_pro,code_std,code_cn'
    )
  })

  test('search narrows the active tag and unchecking removes only that group', async () => {
    const user = userEvent.setup()
    render(<Harness initial={['code_std', 'code_cn']} />)

    await user.click(screen.getByRole('button', { name: /海外平台/ }))
    await user.type(screen.getByPlaceholderText('Search...'), '专业')
    expect(listedNames()).toEqual(['当前专业组'])
    await user.clear(screen.getByPlaceholderText('Search...'))
    await user.type(screen.getByPlaceholderText('Search...'), '国产')
    expect(screen.getByText('No group found.')).toBeInTheDocument()
    await user.clear(screen.getByPlaceholderText('Search...'))
    await user.click(screen.getByRole('option', { name: '当前标准组' }))

    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      /^code_cn$/
    )
  })

  test('existing selection shows current names, keeps ID-less groups under All groups and can be cleared', async () => {
    const user = userEvent.setup()
    render(<Harness initial={['legacy_code', 'code_pro']} />)

    const chips = screen.getByRole('group', { name: 'Selected groups' })
    expect(within(chips).getByText('当前专业组')).toBeInTheDocument()
    expect(within(chips).queryByText('code_pro')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /All groups/ }))
    expect(listedNames()).toContain('legacy_code')
    expect(screen.getByRole('option', { name: 'legacy_code' })).toHaveAttribute(
      'aria-checked',
      'true'
    )
    await user.keyboard('{Escape}')

    await user.click(screen.getByRole('button', { name: 'Remove 当前专业组' }))
    expect(screen.getByLabelText('Selected codes')).toHaveTextContent(
      /^legacy_code$/
    )
    await user.click(screen.getByRole('button', { name: 'Clear all' }))
    expect(screen.getByLabelText('Selected codes')).toBeEmptyDOMElement()
    expect(screen.getByRole('button', { name: 'Clear all' })).toBeDisabled()
  })

  test('without usable tags the original chip multi-select is rendered', () => {
    render(<Harness initial={['code_std']} tags={[tags[2]]} />)

    expect(
      screen.queryByRole('group', { name: 'Group tags' })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('combobox')).toBeInTheDocument()
    expect(screen.getByText('当前标准组')).toBeInTheDocument()
  })
})
