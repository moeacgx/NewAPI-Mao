import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { BillingHistoryDialog } from '../billing-history-dialog'

const mocks = vi.hoisted(() => ({
  isAdmin: true,
  getAllBillingHistory: vi.fn(),
  getUserBillingHistory: vi.fn(),
}))

vi.mock('@/hooks/use-admin', () => ({ useIsAdmin: () => mocks.isAdmin }))
vi.mock('@/features/wallet/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/features/wallet/api')>()),
  getAllBillingHistory: mocks.getAllBillingHistory,
  getUserBillingHistory: mocks.getUserBillingHistory,
}))

const LONG_TRADE_NO = 'OKPAY_SUBUSR123456789NOabcdef1700000000000000000'
const LONG_USERNAME = 'very.long.user.name.that.would.clip.on.a.320px.screen'

function mockRecords(paymentMethod: string) {
  const response = {
    success: true,
    data: {
      total: 1,
      items: [
        {
          id: 1,
          user_id: 987654321,
          username: LONG_USERNAME,
          amount: 10,
          money: 72,
          trade_no: LONG_TRADE_NO,
          payment_method: paymentMethod,
          create_time: 1700000000,
          status: 'pending',
        },
      ],
    },
  }
  mocks.getAllBillingHistory.mockResolvedValue(response)
  mocks.getUserBillingHistory.mockResolvedValue(response)
}

function renderDialog() {
  render(<BillingHistoryDialog open onOpenChange={() => undefined} />)
}

describe('billing history dialog on narrow screens', () => {
  beforeEach(() => {
    mocks.isAdmin = true
  })

  test('shows the full order number and keeps the identity in its own row', async () => {
    mockRecords('okpay')
    renderDialog()

    const orderNo = await screen.findByText(LONG_TRADE_NO)
    const identity = screen.getByText(`${LONG_USERNAME} (User ID: 987654321)`, {
      selector: 'span span',
    })

    expect(orderNo.parentElement).not.toContainElement(identity)
    expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument()
  })

  test('shows the provider logo next to the label for a known method', async () => {
    mockRecords('bepusdt')
    renderDialog()

    const label = await screen.findByText('USDT', {
      selector: 'span.break-words',
    })
    const cell = label.parentElement as HTMLElement

    expect(cell.querySelector('img[alt="USDT"]')).not.toBeNull()
  })

  test('keeps an unknown method readable with the fallback icon', async () => {
    mockRecords('constructor')
    renderDialog()

    const label = await screen.findByText('constructor', {
      selector: 'span.break-words',
    })
    const cell = label.parentElement as HTMLElement

    expect(cell.querySelector('svg')).not.toBeNull()
  })

  test('leaves vertical scrolling to the dialog body on mobile so the pager stays reachable', async () => {
    mockRecords('okpay')
    renderDialog()

    const orderNo = await screen.findByText(LONG_TRADE_NO)
    const list = orderNo.closest('[class*="sm:max-h-"]') as HTMLElement

    expect(list).not.toBeNull()
    expect(list.classList.contains('overflow-y-auto')).toBe(false)
    expect(list.classList.contains('sm:overflow-y-auto')).toBe(true)
  })

  test('hides the identity row for regular users', async () => {
    mocks.isAdmin = false
    mockRecords('alipay')
    renderDialog()

    await screen.findByText(LONG_TRADE_NO)

    expect(
      screen.queryByText(new RegExp(LONG_USERNAME))
    ).not.toBeInTheDocument()
    expect(mocks.getUserBillingHistory).toHaveBeenCalled()
    expect(mocks.getAllBillingHistory).not.toHaveBeenCalled()
  })
})
