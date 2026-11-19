import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, test, vi } from 'vitest'

import type { InvoiceConfig, InvoiceRequest } from '@/features/invoices/types'

import type { PaymentMethod } from '../../../types'
import { PaymentConfirmDialog } from '../payment-confirm-dialog'

const INVOICE_CONFIG: InvoiceConfig = {
  enabled: true,
  discount_disabled: false,
  types: ['personal', 'company'],
  kinds: ['normal', 'special'],
  currency: 'CNY',
  pay_methods: [],
  bepusdt_chains: [],
}

const PAYMENT_METHOD: PaymentMethod = {
  name: 'Fake Pay',
  type: 'fakepay',
} as PaymentMethod

function emptyInvoice(required: boolean): InvoiceRequest {
  return {
    required,
    type: 'personal',
    kind: 'normal',
    title: '',
    tax_no: '',
    email: '',
    phone: '',
    remark: '',
  }
}

function Harness(props: {
  initialInvoice: InvoiceRequest
  onConfirm: () => void
  onOpenChange: (open: boolean) => void
}) {
  const [invoice, setInvoice] = useState(props.initialInvoice)
  return (
    <PaymentConfirmDialog
      open
      onOpenChange={props.onOpenChange}
      onConfirm={props.onConfirm}
      topupAmount={10}
      paymentAmount={72}
      paymentMethod={PAYMENT_METHOD}
      calculating={false}
      processing={false}
      invoiceConfig={INVOICE_CONFIG}
      invoiceRequest={invoice}
      onInvoiceRequestChange={setInvoice}
      invoiceFee={6}
    />
  )
}

function renderDialog(required: boolean) {
  const onConfirm = vi.fn()
  const onOpenChange = vi.fn()
  render(
    <Harness
      initialInvoice={emptyInvoice(required)}
      onConfirm={onConfirm}
      onOpenChange={onOpenChange}
    />
  )
  const dialog = screen.getByRole('alertdialog')
  const scrollRegion = dialog.querySelector(
    '[data-slot="payment-confirm-scroll"]'
  ) as HTMLElement
  return { dialog, scrollRegion, onConfirm, onOpenChange }
}

describe('payment confirm dialog on narrow screens', () => {
  test('expanded invoice form scrolls inside the body while actions stay outside it', () => {
    const { dialog, scrollRegion } = renderDialog(true)

    expect(scrollRegion).not.toBeNull()
    expect(scrollRegion).toContainElement(
      screen.getByPlaceholderText('Optional invoice remark')
    )
    expect(scrollRegion).not.toContainElement(
      screen.getByRole('button', { name: 'Cancel' })
    )
    expect(scrollRegion).not.toContainElement(
      screen.getByRole('button', { name: 'Confirm Payment' })
    )
    expect(scrollRegion).toHaveClass('overflow-y-auto', 'min-h-0')
    expect(dialog).toHaveClass('overflow-hidden', 'flex-col')
    expect(dialog.className).toContain('100dvh')
    expect(dialog.className).toContain('safe-area-inset')
  })

  test('required invoice without a title keeps confirm disabled until filled', () => {
    const { onConfirm } = renderDialog(true)
    const confirm = screen.getByRole('button', { name: 'Confirm Payment' })

    expect(confirm).toBeDisabled()

    fireEvent.change(screen.getByPlaceholderText('Enter invoice title'), {
      target: { value: 'Example Ltd' },
    })
    expect(confirm).toBeEnabled()

    fireEvent.click(confirm)
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  test('cancel closes the dialog without confirming payment', () => {
    const { onConfirm, onOpenChange } = renderDialog(false)

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(onOpenChange).toHaveBeenCalledWith(false, expect.anything())
    expect(onConfirm).not.toHaveBeenCalled()
  })
})
