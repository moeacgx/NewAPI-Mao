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
import { getModelsColumns } from '../ModelsColumnDefs';
import EditModelModal from '../modals/EditModelModal';
import { API } from '../../../../helpers/api';

const record = {
  id: 7,
  model_name: 'fixture-model',
  status: 1,
  enable_groups: ['g_7', 'legacy', 'deleted'],
  enable_group_names: {
    g_7: '当前名称',
    legacy: '历史别名对应名称',
    deleted: 'deleted',
  },
};

function GroupCell({ model }) {
  const columns = getModelsColumns({ t: (key) => key, vendorMap: {} });
  return columns
    .find((column) => column.dataIndex === 'enable_groups')
    .render(model.enable_groups, model);
}

describe('Classic 模型可用分组名称', () => {
  it('显示当前名称和历史别名名称，删除分组回退原编码', () => {
    render(<GroupCell model={record} />);
    expect(screen.getByText('当前名称')).toBeTruthy();
    expect(screen.getByText('历史别名对应名称')).toBeTruthy();
    expect(screen.getByText('deleted')).toBeTruthy();
    expect(screen.queryByText('g_7')).toBeNull();
  });

  it('改名刷新与旧后端缺少映射时使用各自响应，不残留旧名称', () => {
    const view = render(<GroupCell model={record} />);
    view.rerender(
      <GroupCell
        model={{
          ...record,
          enable_group_names: { g_7: '改名后', legacy: '别名改名后' },
        }}
      />,
    );
    expect(screen.getByText('改名后')).toBeTruthy();
    expect(screen.getByText('别名改名后')).toBeTruthy();
    expect(screen.queryByText('当前名称')).toBeNull();
    view.rerender(
      <GroupCell model={{ ...record, enable_group_names: undefined }} />,
    );
    expect(screen.getByText('g_7')).toBeTruthy();
  });

  it('原型别名有自有名称时显示名称，未命中的原型属性回退编码', () => {
    render(
      <GroupCell
        model={{
          ...record,
          enable_groups: ['__proto__', 'constructor', 'unknown'],
          enable_group_names: JSON.parse('{"__proto__":"原型别名名称"}'),
        }}
      />,
    );
    expect(screen.getByText('原型别名名称')).toBeTruthy();
    expect(screen.getByText('constructor')).toBeTruthy();
    expect(screen.getByText('unknown')).toBeTruthy();
  });

  it('点击编辑传递原模型编码数组，不以名称替换业务值', () => {
    const edit = vi.fn();
    const columns = getModelsColumns({
      t: (key) => key,
      vendorMap: {},
      setEditingModel: edit,
      setShowEdit: vi.fn(),
    });
    render(
      columns
        .find((column) => column.dataIndex === 'operate')
        .render(null, record, 0),
    );
    fireEvent.click(screen.getByRole('button', { name: '编辑' }));
    expect(edit).toHaveBeenCalledWith(record);
    expect(record.enable_groups).toEqual(['g_7', 'legacy', 'deleted']);
  });

  it('编辑加载名称映射后实际提交沿用元数据字段，不带只读分组数据', async () => {
    vi.spyOn(API, 'get').mockImplementation(async (url) => ({
      data: {
        success: true,
        data:
          url === '/api/models/7'
            ? { ...record, name_rule: 0, sync_official: 1 }
            : [],
      },
    }));
    const put = vi
      .spyOn(API, 'put')
      .mockResolvedValue({ data: { success: true } });
    render(
      <EditModelModal
        visiable
        editingModel={{ id: 7 }}
        handleClose={vi.fn()}
        refresh={vi.fn()}
      />,
    );
    await waitFor(() =>
      expect(screen.getByLabelText('模型名称').value).toBe('fixture-model'),
    );
    fireEvent.click(screen.getByRole('button', { name: '提交' }));
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/models/',
        expect.objectContaining({
          id: 7,
          model_name: 'fixture-model',
        }),
      ),
    );
    expect(put.mock.calls[0][1]).not.toHaveProperty('enable_group_names');
    expect(put.mock.calls[0][1]).not.toHaveProperty('enable_groups');
    expect(record.enable_groups).toEqual(['g_7', 'legacy', 'deleted']);
  });
});
