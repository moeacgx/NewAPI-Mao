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
import { describe, expect, it, vi } from 'vitest';
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from '@testing-library/react';
import GroupTagEditor from '../components/GroupTagEditor';
import GroupTable from '../components/GroupTable';
import { API } from '../../../../helpers/api';
import useGroupTags from '../../../../hooks/common/useGroupTags';
import { GROUP_TAGS_UPDATED_EVENT } from '../../../../helpers/groupTags';
import { GROUP_DETAILS_UPDATED_EVENT } from '../../../../helpers/groupDetails';
import GroupTagSettings from '../GroupTagSettings';

const groups = [
  { id: 11, code: 'internal-a', name: '旗舰分组', ratio: 1, status: 1 },
  { id: 22, code: 'internal-b', name: '优选分组', ratio: 0.5, status: 1 },
];

describe('分组标签管理', () => {
  it('关键词按当前显示名称匹配，批量加入去重并保留已有绑定顺序', async () => {
    const put = vi
      .spyOn(API, 'put')
      .mockResolvedValue({ data: { success: true } });
    const matchingGroups = [
      { id: 11, code: 'internal-a', name: 'Codex 官方' },
      { id: 22, code: 'internal-b', name: 'Codex 优选' },
      { id: 33, code: 'codex-hidden', name: '独立分组' },
      { id: 44, code: 'codex-fallback', name: '' },
    ];
    render(
      <GroupTagEditor
        tag={{
          id: 4,
          name: '平台',
          description: '',
          sort_order: 0,
          icons: [],
          group_ids: [33, 22],
          match_keywords: [],
        }}
        groups={matchingGroups}
        onSaved={() => {}}
        onCancel={() => {}}
      />,
    );
    const keyword = screen.getByLabelText('Match group display name');
    const keywordInput = keyword.querySelector('input') || keyword;
    const add = screen.getByRole('button', { name: /Add current matches/ });
    expect(add.disabled).toBe(true);
    fireEvent.change(keywordInput, { target: { value: '  CODEX  ' } });
    fireEvent.keyDown(keywordInput, {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
    });
    fireEvent.change(keywordInput, { target: { value: 'fallback' } });
    fireEvent.keyDown(keywordInput, {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
    });
    expect(screen.getByText('Matching groups: 3')).toBeTruthy();
    fireEvent.click(add);
    fireEvent.click(add);
    fireEvent.click(screen.getByRole('button', { name: 'confirm' }));
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/group/tags/4',
        expect.objectContaining({
          group_ids: [33, 22, 11, 44],
          match_keywords: ['CODEX', 'fallback'],
        }),
      ),
    );
  });

  it('保存等待期间 Escape 不关闭草稿，完成后才通知保存', async () => {
    let finish;
    vi.spyOn(API, 'post').mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const onCancel = vi.fn();
    const onSaved = vi.fn();
    render(
      <GroupTagEditor groups={groups} onSaved={onSaved} onCancel={onCancel} />,
    );
    fireEvent.change(screen.getByLabelText('Tag name'), {
      target: { value: '待保存' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'confirm' }));
    fireEvent.keyDown(document, { key: 'Escape', code: 'Escape', keyCode: 27 });
    expect(onCancel).not.toHaveBeenCalled();
    expect(onSaved).not.toHaveBeenCalled();
    await act(async () => finish({ data: { success: true } }));
    expect(onSaved).toHaveBeenCalledOnce();
  });

  it('新目录先返回时，旧请求不能覆盖保存后的标签', async () => {
    let resolveOld;
    vi.spyOn(API, 'get')
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveOld = resolve;
        }),
      )
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: [{ id: 9, name: '新标签', group_ids: [] }],
        },
      });
    const { result } = renderHook(() => useGroupTags());
    act(() => window.dispatchEvent(new Event(GROUP_TAGS_UPDATED_EVENT)));
    await waitFor(() => expect(result.current.tags[0]?.name).toBe('新标签'));
    await act(async () => resolveOld({ data: { success: true, data: [] } }));
    expect(result.current.tags[0]?.name).toBe('新标签');
  });

  it('分组改名通知后卡片刷新当前显示名称', async () => {
    let name = '旧分组名称';
    vi.spyOn(API, 'get').mockImplementation(async (url) => ({
      data: {
        success: true,
        data: url.endsWith('/tags')
          ? [{ id: 1, name: '平台', icons: [], group_ids: [11] }]
          : [{ id: 11, code: 'internal-a', name }],
      },
    }));
    render(<GroupTagSettings />);
    expect(await screen.findByText('旧分组名称')).toBeTruthy();
    name = '当前分组名称';
    act(() => window.dispatchEvent(new Event(GROUP_DETAILS_UPDATED_EVENT)));
    expect(await screen.findByText('当前分组名称')).toBeTruthy();
    expect(screen.queryByText('旧分组名称')).toBeNull();
  });
  it('保存组合 Logo 与 ID 绑定，展示当前名称而非 code', async () => {
    const put = vi
      .spyOn(API, 'put')
      .mockResolvedValue({ data: { success: true } });
    const onSaved = vi.fn();
    render(
      <GroupTagEditor
        tag={{
          id: 4,
          name: '混合平台',
          description: '',
          sort_order: 0,
          icons: ['OpenAI', 'https://cdn.example.com/logo.png'],
          group_ids: [11],
        }}
        groups={groups}
        onSaved={onSaved}
        onCancel={() => {}}
      />,
    );
    expect(screen.getByText('旗舰分组')).toBeTruthy();
    expect(screen.queryByText('internal-a')).toBeNull();
    fireEvent.change(screen.getByLabelText('Tag name'), {
      target: { value: '新名称' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'confirm' }));
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/group/tags/4',
        expect.objectContaining({
          name: '新名称',
          group_ids: [11],
          icons: ['OpenAI', 'https://cdn.example.com/logo.png'],
        }),
      ),
    );
    expect(onSaved).toHaveBeenCalledOnce();
  });

  it('保存失败保留编辑内容且允许重试', async () => {
    const post = vi.spyOn(API, 'post').mockRejectedValue(new Error('保存失败'));
    const onSaved = vi.fn();
    render(
      <GroupTagEditor groups={groups} onSaved={onSaved} onCancel={() => {}} />,
    );
    fireEvent.change(screen.getByLabelText('Tag name'), {
      target: { value: '待保存' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'confirm' }));
    await waitFor(() => expect(post).toHaveBeenCalledOnce());
    expect(onSaved).not.toHaveBeenCalled();
    expect(screen.getByLabelText('Tag name').value).toBe('待保存');
  });

  it('新增 Logo 按输入顺序组合且上限为六个', () => {
    render(
      <GroupTagEditor groups={groups} onSaved={() => {}} onCancel={() => {}} />,
    );
    for (let index = 0; index < 6; index++)
      fireEvent.click(screen.getByRole('button', { name: /Add logo/ }));
    expect(screen.getByRole('button', { name: /Add logo/ }).disabled).toBe(
      true,
    );
    expect(
      screen.getAllByPlaceholderText('Icon name or image URL'),
    ).toHaveLength(6);
    fireEvent.click(screen.getByRole('button', { name: 'Remove logo 2' }));
    expect(screen.getByRole('button', { name: /Add logo/ }).disabled).toBe(
      false,
    );
  });

  it('筛选后编辑分组保留隐藏分组的完整数据', async () => {
    vi.spyOn(API, 'get').mockResolvedValue({
      data: {
        success: true,
        data: [{ id: 1, name: '平台 A', icons: ['OpenAI'], group_ids: [11] }],
      },
    });
    const onChange = vi.fn();
    render(<GroupTable groups={groups} autoGroup={{}} onChange={onChange} />);
    fireEvent.click(await screen.findByRole('radio', { name: /Untagged/ }));
    expect(screen.getByDisplayValue('优选分组')).toBeTruthy();
    expect(screen.queryByDisplayValue('旗舰分组')).toBeNull();
    fireEvent.click(await screen.findByRole('radio', { name: /平台 A/ }));
    expect(screen.queryByDisplayValue('优选分组')).toBeNull();
    fireEvent.change(screen.getByDisplayValue('旗舰分组'), {
      target: { value: '旗舰新名称' },
    });
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith([
        expect.objectContaining({
          id: 11,
          name: '旗舰新名称',
          code: 'internal-a',
        }),
        expect.objectContaining({
          id: 22,
          name: '优选分组',
          code: 'internal-b',
        }),
      ]),
    );
  });
});
