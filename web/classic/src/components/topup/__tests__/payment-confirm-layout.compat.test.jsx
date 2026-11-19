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
import React from 'react';
import { readFileSync } from 'node:fs';
import { URL as NodeURL } from 'node:url';
import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import PaymentConfirmModal from '../modals/PaymentConfirmModal';

const styles = readFileSync(
  new NodeURL('../modals/payment-confirm-modal.css', import.meta.url),
  'utf8',
);

let stylesheet;
afterEach(() => stylesheet?.remove());

const t = (key) => key;

function renderModal({
  required,
  onlineTopUp = vi.fn(),
  handleCancel = vi.fn(),
}) {
  stylesheet = document.createElement('style');
  stylesheet.textContent = styles;
  document.head.appendChild(stylesheet);
  render(
    <PaymentConfirmModal
      t={t}
      open
      onlineTopUp={onlineTopUp}
      handleCancel={handleCancel}
      confirmLoading={false}
      topUpCount={10}
      renderQuotaWithAmount={(value) => String(value)}
      amountLoading={false}
      renderAmount={() => '72.00 元'}
      payWay='fakepay'
      payMethods={[{ type: 'fakepay', name: 'Fake Pay' }]}
      amountNumber={72}
      promoCode=''
      setPromoCode={() => {}}
      promoDiscount={null}
      amountText=''
      invoiceConfig={{
        enabled: true,
        types: ['personal', 'company'],
        kinds: ['normal', 'special'],
      }}
      invoiceRequest={{
        required,
        type: 'personal',
        kind: 'normal',
        title: '',
        tax_no: '',
        email: '',
        phone: '',
        remark: '',
      }}
      setInvoiceRequest={() => {}}
      invoiceFee={6}
    />,
  );
  const modal = document.querySelector('.classic-payment-confirm-modal');
  return {
    modal,
    body: modal?.querySelector('.semi-modal-body'),
    footer: modal?.querySelector('.semi-modal-footer'),
    onlineTopUp,
    handleCancel,
  };
}

describe('充值确认弹窗窄屏布局', () => {
  it('展开发票表单时只滚动内容区，操作按钮留在内容区之外', () => {
    const { modal, body, footer } = renderModal({ required: true });

    expect(modal).not.toBeNull();
    expect(body.contains(screen.getByPlaceholderText('发票备注'))).toBe(true);
    expect(body.contains(footer)).toBe(false);
    expect(footer.querySelectorAll('button')).toHaveLength(2);
    expect(getComputedStyle(body).overflowY).toBe('auto');
    expect(parseFloat(getComputedStyle(body).minHeight)).toBe(0);
    expect(getComputedStyle(footer).flexShrink).toBe('0');
    expect(
      getComputedStyle(modal.querySelector('.semi-modal-content')).overflow,
    ).toBe('hidden');
  });

  it('内容最大高度按动态视口扣除安全区计算', () => {
    expect(styles).toMatch(
      /max-height:\s*calc\(\s*100dvh[^;]*env\(safe-area-inset-top[^;]*env\(safe-area-inset-bottom/,
    );
  });

  it('底部按钮仍分别触发取消与确认支付', () => {
    const { footer, onlineTopUp, handleCancel } = renderModal({
      required: false,
    });
    const [cancel, confirm] = footer.querySelectorAll('button');

    fireEvent.click(cancel);
    expect(handleCancel).toHaveBeenCalledTimes(1);
    expect(onlineTopUp).not.toHaveBeenCalled();

    fireEvent.click(confirm);
    expect(onlineTopUp).toHaveBeenCalledTimes(1);
  });
});
