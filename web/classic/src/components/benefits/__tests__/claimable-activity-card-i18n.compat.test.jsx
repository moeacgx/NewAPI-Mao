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

// 同目录下的 user-benefits-contract.test.mjs 只对源码文本做正则匹配（例如
// assert.match(source, /formatDisplayAmount\(/)），无法区分"调用对了"和"调用错了"——
// 本次要修的 bug 用那种断言方式一样能通过。这里改用真实 zh-CN/zh-TW 资源渲染，
// 译文错误或货币符号读到了另外缓存的客户端配置而非活动响应自带的 amount_display_type，
// 都会让断言失败。
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
  // 全站缓存是 USD，但活动响应自带的是 CNY（后端为这次响应实际换算用的类型）；
  // 卡片必须显示 CNY，不能显示 USD。
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

test('CUSTOM amount falls back to a neutral symbol when the cached unit is not CUSTOM', async () => {
  // 缓存的单位是 USD，活动本身是 CUSTOM：不能冒用 USD 的 "$" 标这个数值，
  // 应回退中性符号 ¤（getCurrencyConfig() 在 type=USD 时返回的 symbol 是 "$"，不是自定义符号）。
  localStorage.setItem('quota_display_type', 'USD');
  await i18next.changeLanguage('zh-TW');
  render(
    <ClaimableActivityCard
      activity={activity({
        claim_paid_threshold: 100,
        amount_display_type: 'CUSTOM',
        eligible: false,
        eligibility_reason: 'ineligible',
      })}
      onClaim={() => {}}
      claiming={false}
    />,
  );

  expect(screen.getByText(/¤100\.00/)).toBeTruthy();
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
