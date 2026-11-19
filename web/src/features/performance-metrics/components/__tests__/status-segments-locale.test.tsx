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
import { act, render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { afterEach, beforeAll, describe, expect, test } from 'vitest'

import { StatusSegments } from '../status-segments'

const END_TS = 24 * 60 * 60
const SERIES = [
  { ts: 1 * 60 * 60, success_rate: 100 },
  { ts: 13 * 60 * 60, success_rate: 90 },
]

describe('StatusSegments interface language locale', () => {
  beforeAll(() => {
    // Register the project's camelCase interface codes so resolvedLanguage
    // matches production (where zhCN / zhTW resources exist).
    i18next.addResourceBundle('zhCN', 'translation', {}, true, true)
    i18next.addResourceBundle('zhTW', 'translation', {}, true, true)
  })

  afterEach(async () => {
    await act(() => i18next.changeLanguage('en'))
  })

  test('zhCN interface language renders Simplified Chinese 24h time ranges instead of throwing', async () => {
    await act(() => i18next.changeLanguage('zhCN'))

    render(<StatusSegments series={SERIES} endTs={END_TS} />)

    const label = screen.getByRole('img').getAttribute('aria-label') ?? ''
    expect(label).toMatch(/\d{2}\/\d{2} \d{2}:\d{2}/)
    expect(label).not.toMatch(/AM|PM|午/)
  })

  test('zhTW interface language renders Traditional Chinese time ranges instead of throwing', async () => {
    await act(() => i18next.changeLanguage('zhTW'))

    render(<StatusSegments series={SERIES} endTs={END_TS} />)

    const label = screen.getByRole('img').getAttribute('aria-label') ?? ''
    expect(label).toMatch(/午/)
  })

  test('malformed stored language falls back to the runtime default locale', async () => {
    await act(() => i18next.changeLanguage('en_US!'))

    render(<StatusSegments series={SERIES} endTs={END_TS} />)

    expect(screen.getByRole('img')).toHaveAttribute(
      'aria-label',
      expect.stringMatching(/\d{2}/)
    )
  })
})
