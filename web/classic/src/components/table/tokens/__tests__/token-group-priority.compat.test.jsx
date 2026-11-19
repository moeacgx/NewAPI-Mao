/*
Copyright (C) 2023-2026 QuantumNous

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
import React, { useState } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import EditTokenModal, { GroupMultiPicker } from '../modals/EditTokenModal';
import { API } from '../../../../helpers/api';
import { StatusContext } from '../../../../context/Status';

const groups = [
  { value: 'default', label: 'Standard models', exclusive: false },
  { value: 'vip', label: 'Priority models', exclusive: false },
  { value: 'team', label: 'Team models', exclusive: false },
];

function translate(key, options = {}) {
  return Object.entries(options).reduce(
    (text, [name, value]) => text.replaceAll(`{{${name}}}`, String(value)),
    String(key),
  );
}

function OrderPicker({ initial, onOrder }) {
  const [selectedGroups, setSelectedGroups] = useState(initial);
  return (
    <GroupMultiPicker
      groups={groups}
      selectedGroups={selectedGroups}
      maxCount={0}
      onChange={(next) => {
        setSelectedGroups(next);
        onOrder(next);
      }}
      t={translate}
    />
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

test('shows priority numbers and reorders two or more groups with the move buttons', () => {
  const onOrder = vi.fn();
  const { container } = render(
    <OrderPicker initial={['default', 'vip', 'team']} onOrder={onOrder} />,
  );

  expect(
    [...container.querySelectorAll('.token-group-priority-index')].map(
      (item) => item.textContent,
    ),
  ).toEqual(['1', '2', '3']);
  expect(
    screen.getByRole('button', { name: 'Move Standard models up' }).disabled,
  ).toBe(true);
  expect(
    screen.getByRole('button', { name: 'Move Team models down' }).disabled,
  ).toBe(true);

  fireEvent.click(screen.getByRole('button', { name: 'Move Team models up' }));
  expect(onOrder).toHaveBeenLastCalledWith(['default', 'team', 'vip']);

  fireEvent.click(
    screen.getByRole('button', { name: 'Move Standard models down' }),
  );
  expect(onOrder).toHaveBeenLastCalledWith(['team', 'default', 'vip']);
});

test('a single selected group has no priority numbers or move buttons', () => {
  const { container } = render(
    <OrderPicker initial={['default']} onOrder={() => {}} />,
  );

  expect(container.querySelector('.token-group-priority-index')).toBeNull();
  expect(screen.queryByRole('button', { name: /^Move .+ (up|down)$/ })).toBeNull();
});

test('a legacy multi-group token keeps its stored order and saves the reordered explicit list', async () => {
  const catalog = [
    { id: 11, code: 'default', name: 'Standard models' },
    { id: 22, code: 'vip', name: 'Priority models' },
  ];
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (url === '/api/token/auto-groups')
      return { data: { success: true, data: { groups: [], max_count: 0 } } };
    if (url === '/api/user/self/groups')
      return {
        data: {
          success: true,
          data: Object.fromEntries(
            catalog.map((group) => [group.code, { ...group, ratio: 1 }]),
          ),
        },
      };
    if (url === '/api/token/9')
      return {
        data: {
          success: true,
          data: {
            id: 9,
            name: 'legacy-multi',
            expired_time: -1,
            unlimited_quota: true,
            remain_quota: 0,
            model_limits: '',
            group: 'vip,default',
            group_ids: [22, 11],
            group_mode: 'explicit',
            group_ratio_limits: '{}',
          },
        },
      };
    return { data: { success: true, data: [] } };
  });
  const save = vi
    .spyOn(API, 'put')
    .mockResolvedValue({ data: { success: true } });

  const { container } = render(
    <StatusContext.Provider value={[{ status: {} }, () => {}]}>
      <EditTokenModal
        visiable
        editingToken={{ id: 9 }}
        refresh={() => {}}
        handleClose={() => {}}
      />
    </StatusContext.Provider>,
  );

  await waitFor(() =>
    expect(
      container.ownerDocument.querySelectorAll('.token-group-priority-index')
        .length,
    ).toBe(2),
  );
  const moveDown = screen.getAllByRole('button', { name: /Move .+ down/ });
  fireEvent.click(moveDown[0]);
  fireEvent.click(screen.getByRole('button', { name: /提交/ }));

  await waitFor(() =>
    expect(save).toHaveBeenCalledWith(
      '/api/token/',
      expect.objectContaining({
        group: 'default,vip',
        group_ids: [11, 22],
        group_mode: 'explicit',
      }),
    ),
  );
});
