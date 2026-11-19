import { describe, expect, test } from 'vitest'

import { getPaymentMethodName, normalizePaymentMethod } from '../billing'

describe('billing history payment method labels', () => {
  const translate = (key: string) => `t:${key}`

  test.each([
    ['alipay', 't:Alipay'],
    ['wxpay', 't:WeChat Pay'],
    ['bepusdt', 't:USDT'],
    ['okpay', 't:OKPay'],
    ['stripe', 't:Stripe'],
    ['creem', 't:Creem'],
    ['waffo', 't:Waffo'],
    ['waffo_pancake', 't:Waffo Pancake'],
    ['balance', 't:Balance'],
  ])('localizes stored method %s', (method, expected) => {
    expect(getPaymentMethodName(method, translate)).toBe(expected)
  })

  test('resolves stored values that differ only by case or whitespace', () => {
    expect(normalizePaymentMethod('  WxPay ')).toBe('wxpay')
    expect(getPaymentMethodName('  Alipay ', translate)).toBe('t:Alipay')
  })

  test('keeps unknown custom methods readable without translating them', () => {
    expect(getPaymentMethodName('custom1', translate)).toBe('custom1')
  })

  test.each(['constructor', 'toString', '__proto__', 'hasOwnProperty'])(
    'renders inherited object property name %s as plain text',
    (method) => {
      expect(getPaymentMethodName(method, translate)).toBe(method)
    }
  )

  test('renders a dash when the stored method is empty', () => {
    expect(getPaymentMethodName('', translate)).toBe('-')
    expect(getPaymentMethodName(undefined, translate)).toBe('-')
  })
})
