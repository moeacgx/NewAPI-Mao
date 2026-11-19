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
import { render, screen } from '@testing-library/react';
import { AxiosHeaders } from 'axios';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { API } from '../../../helpers/api';
import * as utils from '../../../helpers/utils';
import TopupHistoryModal from '../modals/TopupHistoryModal';

const originalAdapter = API.defaults.adapter;
const originalMatchMedia = window.matchMedia;
const LONG_TRADE_NO = 'OKPAY_SUBUSR123456789NOabcdef1700000000000000000';
const LONG_USERNAME = 'very.long.user.name.that.would.clip.on.a.320px.screen';

function mockHistory(paymentMethod) {
  API.defaults.adapter = async (config) => ({
    status: 200,
    statusText: 'OK',
    config,
    headers: new AxiosHeaders(),
    data: {
      success: true,
      data: {
        items: [
          {
            id: 1,
            user_id: 987654321,
            username: LONG_USERNAME,
            trade_no: LONG_TRADE_NO,
            status: 'pending',
            payment_method: paymentMethod,
            money: 10,
            amount: 10,
          },
        ],
        total: 1,
      },
    },
  });
}

function mockViewport(isMobile) {
  window.matchMedia = (query) => ({
    matches: isMobile,
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  });
}

function renderHistory() {
  render(<TopupHistoryModal visible onCancel={() => {}} t={(key) => key} />);
}

beforeEach(() => {
  vi.spyOn(utils, 'isAdmin').mockReturnValue(true);
  mockViewport(false);
});

afterEach(() => {
  API.defaults.adapter = originalAdapter;
  window.matchMedia = originalMatchMedia;
  vi.restoreAllMocks();
});

it('shows the OKPay logo with its label for a case-variant alias', async () => {
  mockHistory(' OKPay ');
  renderHistory();

  const label = await screen.findByText('OKPay');
  const cell = label.parentElement;

  expect(cell.querySelector('img[alt="OKPay"]')).not.toBeNull();
});

it('shows the USDT logo for bepusdt', async () => {
  mockHistory('bepusdt');
  renderHistory();

  const label = await screen.findByText('USDT');

  expect(label.parentElement.querySelector('img[alt="USDT"]')).not.toBeNull();
});

it('renders an unknown method as plain text with the fallback icon', async () => {
  mockHistory('constructor');
  renderHistory();

  const label = await screen.findByText('constructor');

  expect(label.parentElement.querySelector('img')).toBeNull();
  expect(label.parentElement.querySelector('svg')).not.toBeNull();
});

it('renders stacked cards with the full order number and identity on mobile', async () => {
  mockViewport(true);
  mockHistory('okpay');
  renderHistory();

  expect(await screen.findByText(LONG_TRADE_NO)).not.toBeNull();
  expect(screen.getByText(LONG_USERNAME)).not.toBeNull();
  expect(document.querySelector('table')).toBeNull();
  expect(screen.getByRole('button', { name: '重新支付' })).not.toBeNull();
  expect(screen.getByRole('button', { name: '补单' })).not.toBeNull();
});
