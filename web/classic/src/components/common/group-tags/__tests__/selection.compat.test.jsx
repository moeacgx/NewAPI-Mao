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
import { afterEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import GroupTagFilter from '../GroupTagFilter';
import GroupTagLogos from '../GroupTagLogos';
import EditTokenModal, {
  GroupMultiPicker,
} from '../../../table/tokens/modals/EditTokenModal';
import { buildGroupSelectionPayload } from '../../../../helpers/groupDetails';
import { API } from '../../../../helpers/api';
import { StatusContext } from '../../../../context/Status';

const groups = [
  {
    id: 11,
    value: 'internal-a',
    code: 'internal-a',
    label: '开放平台旗舰',
    ratio: 1,
  },
  {
    id: 22,
    value: 'internal-b',
    code: 'internal-b',
    label: '国产模型优选',
    ratio: 0.5,
  },
  {
    id: 33,
    value: 'exclusive',
    code: 'exclusive',
    label: '独立服务',
    exclusive: true,
  },
  { id: null, value: 'auto', code: 'auto', label: '自动选择' },
];
const tags = [
  { id: 1, name: 'OpenAI', icons: ['OpenAI'], group_ids: [11] },
  {
    id: 2,
    name: '国产平台',
    icons: ['DeepSeek.Color', 'Moonshot'],
    description: '跨平台组合',
    group_ids: [22, 33],
  },
];

afterEach(() => vi.unstubAllGlobals());

function Picker(props) {
  const [selected, setSelected] = useState(props.initial || []);
  return (
    <>
      <GroupMultiPicker
        groups={groups}
        groupTags={props.tags ?? tags}
        selectedGroups={selected}
        onChange={setSelected}
        t={(key) => key}
      />
      <output aria-label='submitted'>
        {JSON.stringify(buildGroupSelectionPayload(selected, groups))}
      </output>
    </>
  );
}

describe('分组标签筛选', () => {
  it.each([false, true])(
    '完整令牌表单通过用户标签选组后提交稳定标识，编辑模式=%s',
    async (editing) => {
      vi.spyOn(API, 'get').mockImplementation(async (url) => {
        if (url === '/api/user/self/groups')
          return {
            data: {
              success: true,
              data: Object.fromEntries(
                groups
                  .filter((group) => group.id)
                  .map((group) => [
                    group.code,
                    { ...group, name: group.label },
                  ]),
              ),
              group_tags: tags,
            },
          };
        if (url === '/api/token/9')
          return {
            data: {
              success: true,
              data: {
                id: 9,
                name: '现有令牌',
                expired_time: -1,
                unlimited_quota: true,
                remain_quota: 0,
                model_limits: '',
                group: 'internal-a',
                group_ids: [11],
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
      await screen.findByRole('radio', { name: /OpenAI/ });
      if (!editing) {
        fireEvent.change(screen.getByLabelText('名称'), {
          target: { value: '组合令牌' },
        });
        fireEvent.click(screen.getByRole('radio', { name: /OpenAI/ }));
        fireEvent.click(screen.getByRole('button', { name: /选择分组/ }));
        fireEvent.click(
          await screen.findByRole('button', { name: /开放平台旗舰/ }),
        );
      } else {
        await screen.findByText('开放平台旗舰');
      }
      fireEvent.click(screen.getByRole('radio', { name: /国产平台/ }));
      fireEvent.click(screen.getByRole('button', { name: /添加分组/ }));
      fireEvent.click(
        await screen.findByRole('button', { name: /国产模型优选/ }),
      );
      fireEvent.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() =>
        expect(save).toHaveBeenCalledWith(
          '/api/token/',
          expect.objectContaining({
            group: 'internal-a,internal-b',
            group_ids: [11, 22],
            group_mode: 'explicit',
            cross_group_retry: true,
          }),
        ),
      );
    },
  );
  it('跨标签选择保留已选分组顺序并提交原 code 和 ID', async () => {
    render(<Picker initial={['internal-a']} />);
    fireEvent.click(screen.getByRole('radio', { name: /国产平台/ }));
    expect(screen.getByText('开放平台旗舰')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: /添加分组/ }));
    fireEvent.click(
      await screen.findByRole('button', { name: /国产模型优选/ }),
    );
    expect(JSON.parse(screen.getByLabelText('submitted').textContent)).toEqual({
      group: 'internal-a,internal-b',
      group_ids: [11, 22],
      group_mode: 'explicit',
    });
  });

  it('自动选择与独立分组仍禁止追加其他分组', async () => {
    const { rerender } = render(<Picker initial={['auto']} />);
    expect(screen.getByRole('button', { name: /添加分组/ }).disabled).toBe(
      true,
    );
    rerender(<Picker key='exclusive' initial={['exclusive']} />);
    fireEvent.click(screen.getByRole('radio', { name: /OpenAI/ }));
    expect(screen.getByText('独立服务')).toBeTruthy();
    expect(screen.getByRole('button', { name: /添加分组/ }).disabled).toBe(
      true,
    );
  });

  it('无标签保持原选择入口，标签删除后筛选回到全部', async () => {
    const { rerender } = render(<Picker />);
    fireEvent.click(screen.getByRole('radio', { name: /OpenAI/ }));
    rerender(<Picker tags={[]} />);
    expect(screen.queryByRole('radio')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /选择分组/ }));
    expect(
      await screen.findByRole('button', { name: /国产模型优选/ }),
    ).toBeTruthy();
    expect(screen.getByRole('button', { name: /自动选择/ })).toBeTruthy();
  });

  it('键盘可以切换原生单选标签且空标签不可选择', async () => {
    // jsdom 未实现 CSS.escape，补齐 user-event 模拟浏览器方向键所需的边界 API。
    vi.stubGlobal('CSS', {
      escape: (value) => value.replace(/[^a-zA-Z0-9_-]/g, '\\$&'),
    });
    const user = userEvent.setup();
    function Filter() {
      const [value, onChange] = useState('all');
      return (
        <GroupTagFilter
          groups={groups}
          tags={[...tags, { id: 3, name: '空平台', icons: [], group_ids: [] }]}
          value={value}
          onChange={onChange}
        />
      );
    }
    render(<Filter />);
    const all = screen.getByRole('radio', { name: /All groups/ });
    all.focus();
    await user.keyboard('{ArrowRight}');
    await waitFor(() =>
      expect(screen.getByRole('radio', { name: /Untagged/ }).checked).toBe(
        true,
      ),
    );
    expect(screen.getByRole('radio', { name: /空平台/ }).disabled).toBe(true);
  });

  it('多 Logo 可组合图片，失败后保留其他图标并回退', () => {
    const { container, rerender } = render(
      <GroupTagLogos icons={['OpenAI', 'https://cdn.example.com/logo.svg']} />,
    );
    expect(container.querySelector('svg')).not.toBeNull();
    const img = container.querySelector('img');
    expect(img.getAttribute('referrerpolicy')).toBe('no-referrer');
    fireEvent.error(img);
    expect(container.querySelector('img')).toBeNull();
    expect(screen.getByText('?')).toBeTruthy();
    rerender(<GroupTagLogos icons={['https://cdn.example.com/new.svg']} />);
    expect(container.querySelector('img').getAttribute('src')).toContain(
      'new.svg',
    );
  });
});
