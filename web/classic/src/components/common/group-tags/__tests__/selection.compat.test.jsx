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
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
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

function renderToken({
  editing = false,
  initial = [],
  defaultAuto = false,
} = {}) {
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (url === '/api/user/self/groups')
      return {
        data: {
          success: true,
          data: Object.fromEntries(
            groups.map((group) => [
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
            ...buildGroupSelectionPayload(initial, groups),
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
    <StatusContext.Provider
      value={[{ status: { default_use_auto_group: defaultAuto } }, () => {}]}
    >
      <EditTokenModal
        visiable
        editingToken={editing ? { id: 9 } : {}}
        refresh={() => {}}
        handleClose={() => {}}
      />
    </StatusContext.Provider>,
  );
  return save;
}

async function findTagGroupOption(tagName, groupName) {
  let option;
  await waitFor(() => {
    const popup = screen
      .getAllByRole('dialog', { name: '选择分组' })
      .find((dialog) => dialog.parentElement.style.display !== 'none');
    expect(popup).toBeTruthy();
    option = within(popup).getByRole('button', { name: groupName });
  });
  return option;
}

describe('分组标签筛选', () => {
  it('有已选分组时显示带图标的已选分组标题，空选择不显示空标题', () => {
    const view = render(<Picker initial={['internal-a']} />);

    const heading = screen
      .getByText('已选分组')
      .closest('.token-group-selected-heading');
    expect(heading).not.toBeNull();
    expect(heading?.querySelector('svg')).not.toBeNull();

    view.unmount();
    render(<Picker initial={[]} />);
    expect(screen.queryByText('已选分组')).toBeNull();
  });

  it('令牌隐藏未分类与无可用分组的标签，失效筛选回到全部', async () => {
    const emptyTags = [
      ...tags,
      { id: 3, name: '空平台', icons: [], group_ids: [] },
      { id: 4, name: '不可用平台', icons: [], group_ids: [999] },
    ];
    const { rerender } = render(<Picker tags={emptyTags} />);
    expect(screen.queryByRole('radio', { name: /Untagged/ })).toBeNull();
    expect(
      screen.queryByRole('radio', { name: /空平台|不可用平台/ }),
    ).toBeNull();
    fireEvent.click(screen.getByRole('radio', { name: /OpenAI/ }));
    rerender(
      <Picker
        tags={emptyTags.map((tag) =>
          tag.id === 1 ? { ...tag, group_ids: [] } : tag,
        )}
      />,
    );
    await waitFor(() =>
      expect(screen.getByRole('radio', { name: /All groups/ }).checked).toBe(
        true,
      ),
    );
    rerender(<Picker tags={[]} />);
    fireEvent.click(screen.getByRole('button', { name: /选择分组/ }));
    expect(
      await screen.findByRole('button', { name: /开放平台旗舰/ }),
    ).toBeTruthy();
    expect(screen.getByRole('button', { name: /自动选择/ })).toBeTruthy();
  });

  it.each([false, true])(
    '完整令牌表单通过用户标签选组后提交稳定标识，编辑模式=%s',
    async (editing) => {
      const user = userEvent.setup();
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
        await user.click(screen.getByRole('radio', { name: /OpenAI/ }));
        await user.click(await findTagGroupOption(/OpenAI/, /开放平台旗舰/));
      } else {
        await screen.findByText('开放平台旗舰');
      }
      if (!editing) {
        await user.click(screen.getByRole('button', { name: /提交/ }));
        await waitFor(() =>
          expect(save).toHaveBeenCalledWith(
            '/api/token/',
            expect.objectContaining({
              group: 'internal-a',
              group_ids: [11],
              group_mode: 'explicit',
            }),
          ),
        );
        return;
      }
      fireEvent.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() =>
        expect(save).toHaveBeenCalledWith(
          '/api/token/',
          expect.objectContaining({
            group: 'internal-a',
            group_ids: [11],
            group_mode: 'explicit',
          }),
        ),
      );
    },
  );
  it('跨标签选择保留已选分组顺序并提交原 code 和 ID', async () => {
    render(<Picker initial={['internal-a']} />);
    fireEvent.click(screen.getByRole('radio', { name: /国产平台/ }));
    expect(screen.getByText('开放平台旗舰')).toBeTruthy();
    fireEvent.click(await findTagGroupOption(/国产平台/, /国产模型优选/));
    expect(JSON.parse(screen.getByLabelText('submitted').textContent)).toEqual({
      group: 'internal-a,internal-b',
      group_ids: [11, 22],
      group_mode: 'explicit',
    });
  });

  it.each(['auto', 'exclusive'])(
    '草稿允许在 %s 后追加分组并保留原顺序',
    async (initial) => {
      render(<Picker initial={[initial]} />);
      fireEvent.click(screen.getByRole('radio', { name: /OpenAI/ }));
      fireEvent.click(await findTagGroupOption(/OpenAI/, /开放平台旗舰/));
      expect(
        JSON.parse(screen.getByLabelText('submitted').textContent).group,
      ).toBe(`${initial},internal-a`);
      expect(
        screen.queryByRole('button', { name: /选择分组|添加分组/ }),
      ).toBeNull();
    },
  );

  it.each(['auto', 'exclusive'])(
    '普通分组之后添加 %s 不会替换已有分组',
    async (added) => {
      render(<Picker initial={['internal-a']} />);
      fireEvent.click(
        screen.getByRole('radio', {
          name: added === 'auto' ? /All groups/ : /国产平台/,
        }),
      );
      fireEvent.click(
        await screen.findByRole('button', {
          name: added === 'auto' ? /自动选择/ : /独立服务/,
        }),
      );
      expect(
        JSON.parse(screen.getByLabelText('submitted').textContent).group,
      ).toBe(`internal-a,${added}`);
    },
  );

  it.each([false, true])(
    '重复激活已选标签可重新打开浮窗，键盘激活=%s',
    async (keyboard) => {
      const user = userEvent.setup();
      render(<Picker />);
      const tag = screen.getByRole('radio', { name: /国产平台/ });
      await user.click(tag);
      await user.click(await findTagGroupOption(/国产平台/, /国产模型优选/));
      await waitFor(() =>
        expect(tag.getAttribute('aria-expanded')).toBe('false'),
      );
      if (keyboard) {
        tag.focus();
        await user.keyboard(' ');
      } else {
        await user.click(tag);
      }
      expect(
        await screen.findByRole('button', { name: /独立服务/ }),
      ).toBeTruthy();
      expect(screen.queryByRole('button', { name: /国产模型优选/ })).toBeNull();
      await user.keyboard('{Escape}');
      await waitFor(() =>
        expect(tag.getAttribute('aria-expanded')).toBe('false'),
      );
      expect(document.activeElement).toBe(tag);
      await user.keyboard('{Enter}');
      await waitFor(() =>
        expect(tag.getAttribute('aria-expanded')).toBe('true'),
      );
      expect(await findTagGroupOption(/国产平台/, /独立服务/)).toBeTruthy();
    },
  );

  it('浮窗可按显示名称搜索，切换标签清空上次搜索词', async () => {
    const user = userEvent.setup();
    render(<Picker />);
    await user.click(screen.getByRole('radio', { name: /国产平台/ }));
    await user.type(
      await screen.findByPlaceholderText('搜索分组...'),
      '独立服',
    );
    expect(screen.getByRole('button', { name: /独立服务/ })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /国产模型优选/ })).toBeNull();
    await user.keyboard('{Escape}');
    const tag = screen.getByRole('radio', { name: /OpenAI/ });
    await user.click(tag);
    await waitFor(() => {
      const popupId = tag
        .closest('[aria-controls]')
        .getAttribute('aria-controls');
      const popup = within(document.getElementById(popupId));
      expect(popup.getByPlaceholderText('搜索分组...')).toHaveProperty(
        'value',
        '',
      );
      expect(popup.getByRole('button', { name: /开放平台旗舰/ })).toBeTruthy();
    });
  });

  it.each([
    { editing: false, initial: 'auto', defaultAuto: true, label: '自动选择' },
    { editing: true, initial: 'auto', defaultAuto: false, label: '自动选择' },
    {
      editing: false,
      initial: 'exclusive',
      defaultAuto: false,
      label: '独立服务',
    },
    {
      editing: true,
      initial: 'exclusive',
      defaultAuto: false,
      label: '独立服务',
    },
  ])(
    '保存时拦截 $initial 冲突并保留草稿，编辑=$editing',
    async ({ editing, initial, defaultAuto, label }) => {
      const user = userEvent.setup();
      const save = renderToken({
        editing,
        initial: editing ? [initial] : [],
        defaultAuto,
      });
      await screen.findByRole('radio', { name: /OpenAI/ });
      if (editing) await screen.findByText(label);
      fireEvent.change(screen.getByLabelText('名称'), {
        target: { value: '冲突后保留的草稿' },
      });
      if (!editing && initial === 'exclusive') {
        await user.click(screen.getByRole('radio', { name: /国产平台/ }));
        await user.click(
          await screen.findByRole('button', { name: /独立服务/ }),
        );
      }
      await user.click(screen.getByRole('radio', { name: /OpenAI/ }));
      await waitFor(() =>
        expect(
          screen
            .getByRole('radio', { name: /OpenAI/ })
            .getAttribute('aria-expanded'),
        ).toBe('true'),
      );
      await user.click(await findTagGroupOption(/OpenAI/, /开放平台旗舰/));
      fireEvent.click(screen.getByRole('button', { name: /提交/ }));
      const alert = await screen.findByRole('alert', { name: '' });
      expect(alert.textContent).toContain(label);
      expect(alert.textContent).toMatch(/删除其他分组|Remove the other groups/);
      expect(alert.textContent).toMatch(/独立|单独|separate token/);
      expect(alert.textContent).not.toContain('internal-a');
      expect(alert.textContent).not.toContain('exclusive');
      expect(save).not.toHaveBeenCalled();
      expect(screen.getByLabelText('名称').value).toBe('冲突后保留的草稿');
      expect(screen.getByText(label)).toBeTruthy();
      expect(screen.getByText('开放平台旗舰')).toBeTruthy();
      fireEvent.click(
        screen.getByRole('button', { name: `Remove group ${label}` }),
      );
      fireEvent.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() =>
        expect(save).toHaveBeenCalledWith(
          '/api/token/',
          expect.objectContaining({
            name: '冲突后保留的草稿',
            group: 'internal-a',
            group_ids: [11],
            group_mode: 'explicit',
          }),
        ),
      );
    },
  );

  it.each([
    { initial: 'auto', label: '自动选择', ids: [], mode: 'auto' },
    { initial: 'exclusive', label: '独立服务', ids: [33], mode: 'explicit' },
  ])(
    '单独保存 $initial 保持原有效契约',
    async ({ initial, label, ids, mode }) => {
      const save = renderToken({ defaultAuto: initial === 'auto' });
      await screen.findByRole('radio', { name: /OpenAI/ });
      fireEvent.change(screen.getByLabelText('名称'), {
        target: { value: '独立令牌' },
      });
      if (initial === 'exclusive') {
        fireEvent.click(screen.getByRole('radio', { name: /国产平台/ }));
        fireEvent.click(
          await screen.findByRole('button', { name: new RegExp(label) }),
        );
      }
      fireEvent.click(screen.getByRole('button', { name: /提交/ }));
      await waitFor(() =>
        expect(save).toHaveBeenCalledWith(
          '/api/token/',
          expect.objectContaining({
            group: initial,
            group_ids: ids,
            group_mode: mode,
          }),
        ),
      );
      expect(screen.queryByRole('alert', { name: '' })).toBeNull();
    },
  );

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
