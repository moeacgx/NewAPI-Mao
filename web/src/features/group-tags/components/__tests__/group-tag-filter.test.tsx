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
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, test } from 'vitest'

import { GroupTagFilter } from '../group-tag-filter'
import { GroupTagLogos } from '../group-tag-logos'

function Filter() {
  const [value, setValue] = useState('all')
  return (
    <GroupTagFilter
      tags={[{ id: 1, name: '平台标签', icons: [], group_ids: [11] }]}
      value={value}
      onChange={setValue}
    />
  )
}

describe('group tag controls', () => {
  test('supports keyboard selection and clearing with matching pressed state', async () => {
    render(<Filter />)
    const user = userEvent.setup()
    const tag = screen.getByRole('button', { name: '平台标签' })
    tag.focus()
    await user.keyboard('{Enter}')
    expect(tag).toHaveAttribute('aria-pressed', 'true')
    const all = screen.getByRole('button', { name: 'All groups' })
    all.focus()
    await user.keyboard(' ')
    expect(all).toHaveAttribute('aria-pressed', 'true')
    expect(tag).toHaveAttribute('aria-pressed', 'false')
  })

  test('loads image URLs without a referrer and replaces a failed image with the fallback', () => {
    const view = render(
      <GroupTagLogos icons={['https://cdn.example.com/logo.png']} />
    )
    const image = screen.getByAltText('')
    expect(image).toHaveAttribute('referrerpolicy', 'no-referrer')
    expect(image).toHaveAttribute('loading', 'lazy')
    fireEvent.error(image)
    expect(document.querySelector('img')).toBeNull()
    view.rerender(
      <GroupTagLogos icons={['https://cdn.example.com/replacement.png']} />
    )
    expect(document.querySelector('img')).toHaveAttribute(
      'src',
      'https://cdn.example.com/replacement.png'
    )
  })
})
