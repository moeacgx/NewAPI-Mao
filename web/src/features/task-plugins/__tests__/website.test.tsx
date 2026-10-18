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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, test, vi } from 'vitest'

import { PluginWebsiteLink } from '../components/plugin-website-link'

const website = 'https://example.com/plugin'

test('website link is reachable and activated using the keyboard', async () => {
  const user = userEvent.setup()
  render(<PluginWebsiteLink website={website} />)
  const link = screen.getByRole('link', { name: 'Plugin website' })
  const click = vi.fn((event: Event) => event.preventDefault())
  link.addEventListener('click', click)
  await user.tab()
  expect(link).toHaveFocus()
  await user.keyboard('{Enter}')
  expect(click).toHaveBeenCalledOnce()
  link.removeEventListener('click', click)
})
