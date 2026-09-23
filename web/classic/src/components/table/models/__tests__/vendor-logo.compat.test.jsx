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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import EditVendorModal from '../modals/EditVendorModal';
import { API } from '../../../../helpers/api';

describe('供应商图片配置', () => {
  it('新增供应商时将图片 URL 保存到原 icon 字段', async () => {
    const post = vi
      .spyOn(API, 'post')
      .mockResolvedValue({ data: { success: true } });
    render(
      <EditVendorModal visible handleClose={() => {}} refresh={() => {}} />,
    );
    fireEvent.change(screen.getByLabelText(/供应商名称/), {
      target: { value: 'typesafe' },
    });
    const icon = screen.getByLabelText(/供应商图标/);
    expect(icon.maxLength).toBe(128);
    fireEvent.change(icon, {
      target: { value: 'https://cdn.example.com/typesafe.svg' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'confirm' }));
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/api/vendors/',
        expect.objectContaining({
          name: 'typesafe',
          icon: 'https://cdn.example.com/typesafe.svg',
          status: 1,
        }),
      ),
    );
  });

  it('编辑时加载的超长地址不能绕过表单校验提交', async () => {
    const url = `https://cdn.example.com/${'a'.repeat(128)}.svg`;
    vi.spyOn(API, 'get').mockResolvedValue({
      data: {
        success: true,
        data: { id: 7, name: 'typesafe', icon: url, status: 1 },
      },
    });
    const put = vi
      .spyOn(API, 'put')
      .mockResolvedValue({ data: { success: true } });
    render(
      <EditVendorModal
        visible
        editingVendor={{ id: 7 }}
        handleClose={() => {}}
        refresh={() => {}}
      />,
    );
    await waitFor(() =>
      expect(screen.getByLabelText(/供应商图标/).value).toBe(url),
    );
    fireEvent.click(screen.getByRole('button', { name: 'confirm' }));
    await screen.findByText('Use at most 128 characters for the icon.');
    expect(put).not.toHaveBeenCalled();
  });
});
