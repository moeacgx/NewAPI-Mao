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
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { expect, test, vi, beforeEach, afterEach } from 'vitest';
import { API } from '../../../../helpers';
import AddEditSubscriptionModal from '../modals/AddEditSubscriptionModal';
import { getSubscriptionsColumns } from '../SubscriptionsColumnDefs';
import SubscriptionPlansCard from '../../../topup/SubscriptionPlansCard';
import SubscriptionPurchaseModal from '../../../topup/modals/SubscriptionPurchaseModal';

const t = (key) => key;
const record = {
  plan: {
    id: 7,
    title: '测试套餐',
    enabled: true,
    price_amount: 10,
    currency: 'USD',
    duration_unit: 'month',
    duration_value: 1,
    total_amount: 500000,
    upgrade_group: 'legacy-code',
    upgrade_group_name: '当前分组名称',
  },
};

const originalRangeRect = Object.getOwnPropertyDescriptor(
  Range.prototype,
  'getBoundingClientRect',
);
beforeEach(() => {
  Object.defineProperty(Range.prototype, 'getBoundingClientRect', {
    configurable: true,
    value: () => new DOMRect(),
  });
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  if (originalRangeRect)
    Object.defineProperty(
      Range.prototype,
      'getBoundingClientRect',
      originalRangeRect,
    );
  else delete Range.prototype.getBoundingClientRect;
});

test('真实管理列展示当前名称，名称缺失回退code', () => {
  const column = getSubscriptionsColumns({ t }).find(
    (item) => item.title === '升级分组',
  );
  const { rerender } = render(column.render(null, record));
  expect(screen.getByText('当前分组名称')).toBeTruthy();
  rerender(
    column.render(null, {
      plan: { ...record.plan, upgrade_group_name: undefined },
    }),
  );
  expect(screen.getByText('legacy-code')).toBeTruthy();
});

test('真实套餐详情弹层展示当前名称', async () => {
  const column = getSubscriptionsColumns({ t }).find(
    (item) => item.title === '套餐',
  );
  render(column.render('测试套餐', record));
  fireEvent.mouseEnter(screen.getByText('测试套餐'));
  expect(await screen.findByText('当前分组名称')).toBeTruthy();
});

test('真实购买列表展示当前名称，不追加code', () => {
  render(
    <SubscriptionPlansCard
      t={t}
      plans={[record]}
      onChangeBillingPreference={() => {}}
    />,
  );
  expect(screen.getByText('升级分组: 当前分组名称')).toBeTruthy();
  expect(screen.queryByText(/legacy-code/)).toBeNull();
});

test('真实购买确认跟随改名，删除名称后回退code', () => {
  const props = { t, visible: true, selectedPlan: record, onCancel: () => {} };
  const { rerender } = render(<SubscriptionPurchaseModal {...props} />);
  expect(screen.getByText('当前分组名称')).toBeTruthy();
  rerender(
    <SubscriptionPurchaseModal
      {...props}
      selectedPlan={{
        plan: { ...record.plan, upgrade_group_name: '改名后的名称' },
      }}
    />,
  );
  expect(screen.getByText('改名后的名称')).toBeTruthy();
  rerender(
    <SubscriptionPurchaseModal
      {...props}
      selectedPlan={{ plan: { ...record.plan, upgrade_group_name: undefined } }}
    />,
  );
  expect(screen.getByText('legacy-code')).toBeTruthy();
});

test('真实编辑表单保留历史alias名称并原样提交code', async () => {
  vi.spyOn(API, 'get').mockResolvedValue({
    data: {
      success: true,
      data: [{ id: 2, code: 'stable-code', name: '当前分组名称' }],
      group_names: { 'legacy-code': '当前分组名称' },
    },
  });
  vi.spyOn(API, 'put').mockResolvedValue({ data: { success: true } });
  render(
    <AddEditSubscriptionModal
      t={t}
      visible
      editingPlan={record}
      handleClose={() => {}}
    />,
  );
  expect(await screen.findByText('当前分组名称')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() =>
    expect(API.put).toHaveBeenCalledWith(
      '/api/subscription/admin/plans/7',
      expect.objectContaining({
        plan: expect.objectContaining({ upgrade_group: 'legacy-code' }),
      }),
    ),
  );
});

test('真实创建表单选项展示名称并提交code', async () => {
  vi.spyOn(API, 'get').mockResolvedValue({
    data: {
      success: true,
      data: [{ id: 2, code: 'stable-code', name: '当前分组名称' }],
      group_names: { 'stable-code': '当前分组名称' },
    },
  });
  vi.spyOn(API, 'post').mockResolvedValue({ data: { success: true } });
  render(
    <AddEditSubscriptionModal
      t={t}
      visible
      editingPlan={null}
      handleClose={() => {}}
    />,
  );
  fireEvent.change(await screen.findByPlaceholderText('例如：基础套餐'), {
    target: { value: '新套餐' },
  });
  const trigger = screen
    .getAllByRole('combobox')
    .find((el) => el.textContent.includes('不升级'));
  fireEvent.click(trigger);
  fireEvent.click(await screen.findByRole('option', { name: /当前分组名称/ }));
  // jsdom 不运行动画；模拟退场动画结束，使 Semi 提交受控选择值。
  document
    .querySelectorAll('[class*="animation-hide"]')
    .forEach((node) => fireEvent.animationEnd(node));
  await waitFor(() =>
    expect(
      screen
        .getAllByRole('combobox')
        .some((el) => el.textContent.includes('当前分组名称')),
    ).toBe(true),
  );
  fireEvent.click(screen.getByRole('button', { name: /提交/ }));
  await waitFor(() =>
    expect(API.post).toHaveBeenCalledWith(
      '/api/subscription/admin/plans',
      expect.objectContaining({
        plan: expect.objectContaining({
          title: '新套餐',
          upgrade_group: 'stable-code',
        }),
      }),
    ),
  );
});
