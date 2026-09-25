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

import { render, screen } from '@testing-library/react';
import i18next from 'i18next';
import { afterEach, beforeAll, expect, test } from 'vitest';

import zhCNResource from '../../../i18n/locales/zh-CN.json';
import zhTWResource from '../../../i18n/locales/zh-TW.json';
import ClaimableActivityCard from '../ClaimableActivityCard';

// The sibling `user-benefits-contract.test.mjs` suite only greps the source
// text (e.g. `assert.match(source, /formatDisplayAmount\(/)`), so it cannot
// tell a correct call from a buggy one — the stale-localStorage bug this
// fixes would have passed those assertions either way. These tests actually
// render through real zh-CN/zh-TW resource bundles, so a wrong translation
// or a currency symbol read from a separately-cached client setting instead
// of the `amount_display_type` the activity payload itself carries fails an
// assertion here.
beforeAll(() => {
  i18next.addResourceBundle(
    'zh-CN',
    'translation',
    zhCNResource.translation,
    true,
    true,
  );
  i18next.addResourceBundle(
    'zh-TW',
    'translation',
    zhTWResource.translation,
    true,
    true,
  );
});

function activity(overrides = {}) {
  return {
    id: 1,
    name: 'Weekend Boost',
    group_name_snapshot: 'Default',
    amount_mode: 'fixed',
    amount_display_type: 'USD',
    fixed_quota: 500000,
    min_quota: 0,
    max_quota: 0,
    total_count: 10,
    remaining_count: 5,
    claim_paid_threshold: 0,
    personal_valid_hours: 24,
    eligible: true,
    has_claimed: false,
    eligibility_reason: undefined,
    ...overrides,
  };
}

afterEach(async () => {
  await i18next.changeLanguage('zh');
});

test('zh-CN renders the domain-specific claim-eligibility text, not a duplicated generic key', async () => {
  await i18next.changeLanguage('zh-CN');
  render(
    <ClaimableActivityCard
      activity={activity({ eligible: false, eligibility_reason: 'sold_out' })}
      onClaim={() => {}}
      claiming={false}
    />,
  );

  expect(screen.getByText('活动券已领完')).toBeTruthy();
});

test('zh-TW renders every claim-ineligibility reason with its own real translation', async () => {
  await i18next.changeLanguage('zh-TW');
  const cases = [
    ['ineligible', '不符合領取條件'],
    ['inactive', '活動目前不可領取'],
    ['not_started', '活動尚未開始'],
    ['ended', '活動已結束'],
  ];

  for (const [reason, text] of cases) {
    const { unmount } = render(
      <ClaimableActivityCard
        activity={activity({ eligible: false, eligibility_reason: reason })}
        onClaim={() => {}}
        claiming={false}
      />,
    );
    expect(screen.getByText(text)).toBeTruthy();
    unmount();
  }
});

test('claim threshold amount follows the type in the activity payload, ignoring a stale localStorage cache', async () => {
  // The site-wide client cache says USD; the activity payload itself
  // carries CNY (the type the backend actually computed this threshold
  // with for this response). The card must render CNY, not USD.
  localStorage.setItem('quota_display_type', 'USD');
  await i18next.changeLanguage('zh-TW');
  render(
    <ClaimableActivityCard
      activity={activity({
        claim_paid_threshold: 100,
        amount_display_type: 'CNY',
        eligible: false,
        eligibility_reason: 'ineligible',
      })}
      onClaim={() => {}}
      claiming={false}
    />,
  );

  expect(screen.getByText(/¥100\.00/)).toBeTruthy();
  expect(screen.queryByText(/\$100/)).toBeNull();
  localStorage.removeItem('quota_display_type');
});

test('zh-CN renders the extra registration-age claim condition alongside the top-up requirement', async () => {
  await i18next.changeLanguage('zh-CN');
  render(
    <ClaimableActivityCard
      activity={activity()}
      onClaim={() => {}}
      claiming={false}
    />,
  );

  expect(screen.getByText('新注册账号需满一定时长后才能领取')).toBeTruthy();
});
