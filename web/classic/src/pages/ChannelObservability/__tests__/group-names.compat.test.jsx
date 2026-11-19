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
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterAll, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../../helpers', () => ({ API: { get: vi.fn() } }));

import { API } from '../../../helpers';
import ChannelObservability from '../index';
import ChannelsView from '../ChannelsView';
import OperationsView from '../OperationsView';

const makeParams = (extra) => new URLSearchParams(extra);
const envelope = (data) => ({ data: { success: true, data } });

beforeEach(() => {
  API.get.mockReset();
  const storage = new Map();
  vi.stubGlobal('localStorage', {
    getItem: (key) => storage.get(key) ?? null,
    setItem: (key, value) => storage.set(key, String(value)),
    removeItem: (key) => storage.delete(key),
    clear: () => storage.clear(),
  });
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
});
afterAll(() => vi.unstubAllGlobals());

describe('原生可观测页面分组展示', () => {
  it('稳定性矩阵显示当前名称不附历史标识，展开请求保留原标识', async () => {
    let name = '当前企业分组';
    const requests = [];
    API.get.mockImplementation(async (_url, config) => {
      const params = new URLSearchParams(config.params);
      requests.push(params);
      return envelope({
        total: 1,
        page: 1,
        page_size: 30,
        items:
          params.get('dimension') === 'group'
            ? [{ group: 'legacy-enterprise', group_name: name, windows: [] }]
            : [
                {
                  group: 'legacy-enterprise',
                  group_name: name,
                  channel_id: 9,
                  channel_name: '验证渠道',
                  windows: [],
                },
              ],
      });
    });
    const { rerender } = render(
      <OperationsView
        makeParams={makeParams}
        queryKey='fixed'
        refreshKey={0}
      />,
    );
    const label = await screen.findByText('当前企业分组');
    expect(screen.queryByText(/legacy-enterprise/)).toBeNull();
    await userEvent.click(
      within(label.closest('tr')).getByRole('button', { name: '展开' }),
    );
    expect(await screen.findByText('验证渠道')).toBeTruthy();
    expect(
      requests
        .find((params) => params.get('dimension') === 'group_channel')
        ?.get('groups'),
    ).toBe('legacy-enterprise');
    name = '改名后的企业专属模型稳定性观测分组'.repeat(4);
    rerender(
      <OperationsView
        makeParams={makeParams}
        queryKey='fixed'
        refreshKey={1}
      />,
    );
    expect(await screen.findByText(name)).toBeTruthy();
    expect(screen.queryByText('当前企业分组')).toBeNull();
    expect(screen.queryByText(/legacy-enterprise/)).toBeNull();
  });

  it('稳定性矩阵删除或未知名称回退原标识，无分组保留未标记状态', async () => {
    API.get.mockResolvedValue(
      envelope({
        total: 3,
        items: [
          {
            key: 'deleted',
            group: 'deleted-code',
            group_name: '',
            windows: [],
          },
          { key: 'unknown', group: 'unknown-code', windows: [] },
          { key: 'none', windows: [] },
        ],
      }),
    );
    render(
      <OperationsView
        makeParams={makeParams}
        queryKey='fixed'
        refreshKey={0}
      />,
    );
    expect(await screen.findByText('deleted-code')).toBeTruthy();
    expect(screen.getByText('unknown-code')).toBeTruthy();
    expect(screen.getByText('未标记分组')).toBeTruthy();
  });

  it('渠道配置分组消费独立名称字段，缺名时才回退', async () => {
    API.get.mockResolvedValue(
      envelope({
        total: 2,
        items: [
          {
            channel_id: 9,
            channel_name: '验证渠道',
            channel_type: 1,
            group: 'legacy-enterprise,paid-code',
            group_name: '当前企业分组,付费分组',
          },
          {
            channel_id: 10,
            channel_name: '旧渠道',
            channel_type: 1,
            group: 'deleted-code',
          },
        ],
      }),
    );
    render(
      <ChannelsView makeParams={makeParams} queryKey='fixed' refreshKey={0} />,
    );
    expect(
      await screen.findByText(/配置分组.*当前企业分组,付费分组/),
    ).toBeTruthy();
    expect(screen.queryByText(/legacy-enterprise|paid-code/)).toBeNull();
    expect(screen.getByText(/配置分组.*deleted-code/)).toBeTruthy();
  });

  it('筛选仅显示当前名称或未知标识，选中后仍发送原code', async () => {
    const requests = [];
    API.get.mockImplementation(async (url, config) => {
      requests.push({ url, params: new URLSearchParams(config?.params) });
      if (url.endsWith('/filters'))
        return envelope({
          groups: [
            { code: 'legacy-enterprise', name: '当前企业分组' },
            { code: 'deleted-code', name: '' },
          ],
        });
      return envelope({ items: [], points: [], summary: {}, meta: {} });
    });
    render(<ChannelObservability />);
    await userEvent.click(screen.getByRole('button', { name: /高级筛选/ }));
    await userEvent.click(await screen.findByText('全部请求分组'));
    const option = await screen.findByText('当前企业分组', { exact: true });
    expect(option.closest('[role="option"]')).toBeTruthy();
    expect(screen.getByText('deleted-code', { exact: true })).toBeTruthy();
    expect(screen.queryByText(/legacy-enterprise/)).toBeNull();
    const optionNode = option.closest('[role="option"]');
    fireEvent.click(optionNode);
    await waitFor(() => {
      if (optionNode.isConnected) fireEvent.animationEnd(optionNode);
      expect(
        requests.map(({ url, params }) => ({ url, query: params.toString() })),
      ).toEqual(
        expect.arrayContaining([
          expect.objectContaining({
            query: expect.stringContaining('groups=legacy-enterprise'),
          }),
        ]),
      );
    });
    expect(
      requests.some(({ params }) => params.get('groups') === '当前企业分组'),
    ).toBe(false);
  });
});
