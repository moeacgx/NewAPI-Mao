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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import GroupRatioSettings from '../GroupRatioSettings';
import { API } from '../../../../helpers/api';
import { StatusProvider } from '../../../../context/Status';

const options = {
  GroupRatio: '{}',
  TopupGroupRatio: '{}',
  UserUsableGroups: '{}',
  GroupGroupRatio: '{}',
  'group_ratio_setting.group_special_usable_group': '{}',
  AutoGroups: '[]',
  DefaultUseAutoGroup: false,
  MaxTokenAutoGroups: '5',
};

for (const mode of ['可视化编辑', '手动编辑']) {
  test.each(['0', '8'])(`${mode}保存上限 %s 后重新读取原值`, async (limit) => {
    vi.spyOn(API, 'get').mockResolvedValue({
      data: { success: true, data: [] },
    });
    const put = vi.spyOn(API, 'put').mockResolvedValue({
      data: { success: true },
    });
    const refresh = vi.fn().mockResolvedValue(undefined);
    const view = render(
      <StatusProvider>
        <GroupRatioSettings options={options} refresh={refresh} />
      </StatusProvider>,
    );
    await waitFor(() =>
      expect(API.get).toHaveBeenCalledWith('/api/group/details'),
    );
    fireEvent.click(screen.getByRole('radio', { name: mode }));
    const input = screen.getByRole('spinbutton', {
      name: 'Maximum groups per token',
    });
    expect(input.value).toBe('5');
    fireEvent.change(input, { target: { value: limit } });
    fireEvent.blur(input);
    fireEvent.click(screen.getByRole('button', { name: '保存分组相关设置' }));
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/group/details',
        expect.objectContaining({
          option_updates: { MaxTokenAutoGroups: limit },
        }),
      ),
    );
    await waitFor(() => expect(refresh).toHaveBeenCalledOnce());
    view.rerender(
      <StatusProvider>
        <GroupRatioSettings
          options={{ ...options, MaxTokenAutoGroups: limit }}
          refresh={refresh}
        />
      </StatusProvider>,
    );
    await waitFor(() =>
      expect(
        screen.getByRole('spinbutton', { name: 'Maximum groups per token' })
          .value,
      ).toBe(limit),
    );
  });

  test.each(['', '-1', '1.5'])(
    `${mode}提交非法上限 %s 时阻止保存`,
    async (value) => {
      vi.spyOn(API, 'get').mockResolvedValue({
        data: { success: true, data: [] },
      });
      const put = vi.spyOn(API, 'put').mockResolvedValue({
        data: { success: true },
      });
      render(
        <StatusProvider>
          <GroupRatioSettings options={options} refresh={vi.fn()} />
        </StatusProvider>,
      );
      await waitFor(() =>
        expect(API.get).toHaveBeenCalledWith('/api/group/details'),
      );
      fireEvent.click(screen.getByRole('radio', { name: mode }));
      const input = screen.getByRole('spinbutton', {
        name: 'Maximum groups per token',
      });
      fireEvent.change(input, { target: { value } });
      fireEvent.blur(input);
      fireEvent.click(screen.getByRole('button', { name: '保存分组相关设置' }));
      expect(
        await screen.findByText(
          mode === '手动编辑'
            ? '错误：请检查输入'
            : '错误：Enter a non-negative integer (0 for unlimited)',
        ),
      ).toBeTruthy();
      expect(put).not.toHaveBeenCalled();
    },
  );

  test(`${mode}未配置时默认显示不限的 0`, async () => {
    vi.spyOn(API, 'get').mockResolvedValue({
      data: { success: true, data: [] },
    });
    const { MaxTokenAutoGroups: _limit, ...unsetOptions } = options;
    render(
      <StatusProvider>
        <GroupRatioSettings options={unsetOptions} refresh={vi.fn()} />
      </StatusProvider>,
    );
    await waitFor(() =>
      expect(API.get).toHaveBeenCalledWith('/api/group/details'),
    );
    fireEvent.click(screen.getByRole('radio', { name: mode }));
    expect(
      screen.getByRole('spinbutton', { name: 'Maximum groups per token' })
        .value,
    ).toBe('0');
  });
}
