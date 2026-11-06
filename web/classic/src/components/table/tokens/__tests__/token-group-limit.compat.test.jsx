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
import React, { useState } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createInstance } from 'i18next';
import { I18nextProvider } from 'react-i18next';
import { expect, test, vi } from 'vitest';
import EditTokenModal, { GroupMultiPicker } from '../modals/EditTokenModal';
import { API } from '../../../../helpers/api';
import { StatusContext } from '../../../../context/Status';

const groups = [
  { value: 'auto', label: 'Automatic routing', exclusive: false },
  { value: 'default', label: 'Standard models', exclusive: false },
  { value: 'vip', label: 'Priority models', exclusive: false },
];

const unlimitedGroups = [
  { id: 11, code: 'base', name: '标准显示名称' },
  { id: 12, code: 'plus', name: '高级显示名称' },
  { id: 13, code: 'fast', name: '快速显示名称' },
  { id: 14, code: 'smart', name: '智能显示名称' },
  { id: 15, code: 'media', name: '媒体显示名称' },
  { id: 16, code: 'backup', name: '备用显示名称' },
];

function translate(key, options = {}) {
  return String(key)
    .replace('{{count}}', options.count ?? '')
    .replace('{{max}}', options.max ?? '');
}

function LimitPicker({ maxCount, initial }) {
  const [selectedGroups, setSelectedGroups] = useState(initial);
  return (
    <GroupMultiPicker
      groups={groups}
      selectedGroups={selectedGroups}
      maxCount={maxCount}
      onChange={setSelectedGroups}
      t={translate}
    />
  );
}

test('disables another explicit group once the token reaches the configured limit', async () => {
  const i18n = createInstance();
  await i18n.init({
    lng: 'en',
    resources: { en: { translation: {} } },
    interpolation: { escapeValue: false },
  });
  const user = userEvent.setup();
  const view = render(
    <I18nextProvider i18n={i18n}>
      <LimitPicker maxCount={1} initial={['default']} />
    </I18nextProvider>,
  );

  expect(screen.getByText('1 / 1 groups selected')).toBeTruthy();
  await user.click(screen.getByRole('button', { name: /添加分组/ }));
  const blocked = screen.getByRole('button', { name: /Priority models/ });
  expect(blocked.disabled).toBe(true);
  expect(screen.getByText('Standard models')).toBeTruthy();

  view.rerender(
    <I18nextProvider i18n={i18n}>
      <LimitPicker key='over-limit' maxCount={1} initial={['default', 'vip']} />
    </I18nextProvider>,
  );
  expect(screen.getByText('A token can select at most 1 groups')).toBeTruthy();
});

test.each([false, true])(
  '不限分组的完整令牌表单保存六组稳定 ID，编辑模式=%s',
  async (editing) => {
    const user = userEvent.setup();
    vi.spyOn(API, 'get').mockImplementation(async (url) => {
      if (url === '/api/token/auto-groups')
        return { data: { success: true, data: { groups: [], max_count: 0 } } };
      if (url === '/api/user/self/groups')
        return {
          data: {
            success: true,
            data: Object.fromEntries(
              unlimitedGroups.map((group) => [
                group.code,
                { ...group, ratio: 1 },
              ]),
            ),
          },
        };
      if (url === '/api/token/9')
        return {
          data: {
            success: true,
            data: {
              id: 9,
              name: '六组令牌',
              expired_time: -1,
              unlimited_quota: true,
              remain_quota: 0,
              model_limits: '',
              group: unlimitedGroups.map((group) => group.code).join(','),
              group_ids: unlimitedGroups.map((group) => group.id),
              group_mode: 'explicit',
              group_ratio_limits: '{}',
            },
          },
        };
      return { data: { success: true, data: [] } };
    });
    const save = vi
      .spyOn(API, editing ? 'put' : 'post')
      .mockResolvedValue({ data: { success: true } });
    render(
      <StatusContext.Provider value={[{ status: {} }, () => {}]}>
        <EditTokenModal
          visiable
          editingToken={editing ? { id: 9 } : {}}
          refresh={() => {}}
          handleClose={() => {}}
        />
      </StatusContext.Provider>,
    );
    await waitFor(() =>
      expect(API.get).toHaveBeenCalledWith('/api/token/auto-groups'),
    );
    if (!editing) {
      fireEvent.change(screen.getByLabelText('名称'), {
        target: { value: '六组令牌' },
      });
      for (const group of unlimitedGroups) {
        await user.click(
          screen.getByRole('button', { name: /选择分组|添加分组/ }),
        );
        const option = await screen.findByRole('button', {
          name: new RegExp(group.name),
        });
        expect(option.disabled).toBe(false);
        await user.click(option);
      }
    }
    await screen.findByText('6 groups selected (unlimited)');
    expect(screen.queryByText(/6 \/ 0|at most 0 groups/)).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /提交/ }));
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        '/api/token/',
        expect.objectContaining({
          group: 'base,plus,fast,smart,media,backup',
          group_ids: [11, 12, 13, 14, 15, 16],
          group_mode: 'explicit',
        }),
      ),
    );
  },
);
