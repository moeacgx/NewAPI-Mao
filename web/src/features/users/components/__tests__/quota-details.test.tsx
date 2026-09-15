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
import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test } from 'vitest'

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { UserQuotaCell } from '../user-quota-cell'

beforeEach(() => {
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
})
afterEach(() => {
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
  localStorage.clear()
})

test('零额度仍可通过键盘打开余额与已用详情', async () => {
  const user = userEvent.setup()
  render(<UserQuotaCell remaining={0} used={0} />)
  await user.tab()
  expect(screen.getByRole('button', { name: 'No Quota' })).toHaveFocus()
  await user.keyboard('{Enter}')
  const dialog = await screen.findByRole('dialog')
  expect(within(dialog).getByText('Available Balance')).toBeVisible()
  expect(within(dialog).getByText('Total Used')).toBeVisible()
  expect(screen.getByRole('button', { name: 'No Quota' })).toHaveAttribute(
    'aria-expanded',
    'true'
  )
})

test('负余额和已用之和为零时仍显示真实金额', async () => {
  render(<UserQuotaCell remaining={-500000} used={500000} />)
  expect(screen.queryByText('No Quota')).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button'))
  const dialog = await screen.findByRole('dialog')
  expect(within(dialog).getByText('-1')).toBeVisible()
  expect(within(dialog).getByText('1')).toBeVisible()
})

test('币种改变后已打开的额度详情同步更新', async () => {
  render(<UserQuotaCell remaining={500000} used={1000000} />)
  await userEvent.click(screen.getByRole('button'))
  expect(await screen.findByText('Quota ($)')).toBeVisible()
  act(() =>
    useSystemConfigStore.getState().setConfig({
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        quotaDisplayType: 'CNY',
        usdExchangeRate: 7,
      },
    })
  )
  expect(screen.getByText('Quota (¥)')).toBeVisible()
  expect(within(screen.getByRole('dialog')).getByText('7')).toBeVisible()
  expect(within(screen.getByRole('dialog')).getByText('14')).toBeVisible()
})

test('大额余额在详情中显示完整金额而不是缩写', async () => {
  render(<UserQuotaCell remaining={617280000} used={1000000} />)
  await userEvent.click(screen.getByRole('button'))
  const dialog = await screen.findByRole('dialog')
  expect(within(dialog).getByText('1,234.56')).toBeVisible()
})

test('Tokens 模式详情展示完整额度以便核对', async () => {
  useSystemConfigStore.getState().setConfig({
    currency: { ...DEFAULT_CURRENCY_CONFIG, quotaDisplayType: 'TOKENS' },
  })
  render(<UserQuotaCell remaining={500000} used={1000000} />)
  await userEvent.click(screen.getByRole('button'))
  const dialog = await screen.findByRole('dialog')
  expect(within(dialog).getByText('500000')).toBeVisible()
  expect(within(dialog).getByText('1000000')).toBeVisible()
})
