/*
Copyright (C) 2025 QuantumNous

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
import { describe, expect, it, vi } from 'vitest';
import { getPaymentRequestErrorMessage } from '../topupError';

describe('getPaymentRequestErrorMessage', () => {
  it('shows the remaining retry delay for a rate-limited payment', () => {
    const translate = vi.fn((message, values) =>
      message.replace('{{seconds}}', values.seconds),
    );

    const message = getPaymentRequestErrorMessage(
      { response: { status: 429, headers: { 'retry-after': '12' } } },
      translate,
    );

    expect(message).toBe('请求过于频繁，请在 12 秒后重试');
  });

  it('uses a later retry message when Retry-After is unavailable', () => {
    const translate = vi.fn((message) => message);

    const message = getPaymentRequestErrorMessage(
      { response: { status: 429, headers: {} } },
      translate,
    );

    expect(message).toBe('请求过于频繁，请稍后再试');
  });

  it('keeps the generic payment error for non-rate-limit failures', () => {
    const translate = vi.fn((message) => message);

    const message = getPaymentRequestErrorMessage(
      { response: { status: 502, headers: {} } },
      translate,
    );

    expect(message).toBe('支付请求失败');
  });
});
