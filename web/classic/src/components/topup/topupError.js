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

export const TOPUP_QUOTA_LIMIT_ERROR = 'top-up quota limit exceeded';
export const TOPUP_QUOTA_LIMIT_MESSAGE =
  'Top-up would exceed the wallet quota limit. Please reduce the amount or contact an administrator.';

export function getPaymentRequestErrorMessage(error, translate) {
  if (error?.response?.status === 429) {
    const rawRetryAfter =
      error.response.headers?.['retry-after'] ??
      error.response.headers?.get?.('Retry-After');
    const retryAfterSeconds = Number(rawRetryAfter);
    if (Number.isFinite(retryAfterSeconds) && retryAfterSeconds > 0) {
      return translate('请求过于频繁，请在 {{seconds}} 秒后重试', {
        seconds: Math.ceil(retryAfterSeconds),
      });
    }
    return translate('请求过于频繁，请稍后再试');
  }

  return translate('支付请求失败');
}

export function getTopupErrorMessage(
  message,
  data,
  translate,
  fallback = '支付请求失败',
) {
  const dataMessage = typeof data === 'string' ? data.trim() : '';
  const responseMessage = typeof message === 'string' ? message.trim() : '';
  const rawMessage = dataMessage || responseMessage;

  if (rawMessage === TOPUP_QUOTA_LIMIT_ERROR) {
    return translate(TOPUP_QUOTA_LIMIT_MESSAGE);
  }

  if (!rawMessage || rawMessage === 'error') {
    return translate(fallback);
  }

  return rawMessage;
}
