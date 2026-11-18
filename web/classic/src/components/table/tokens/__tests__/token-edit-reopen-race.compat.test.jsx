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
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import EditTokenModal from '../modals/EditTokenModal';
import TokensActions from '../TokensActions';
import { useTokensData } from '../../../../hooks/tokens/useTokensData';
import { API } from '../../../../helpers/api';
import { StatusContext } from '../../../../context/Status';

const tokenRecords = {
  1: { id: 1, name: '原令牌一' },
  2: { id: 2, name: '另一令牌二' },
};

beforeEach(() => {
  vi.useFakeTimers({
    toFake: ['setTimeout', 'clearTimeout'],
    shouldAdvanceTime: true,
  });
});

afterEach(() => {
  vi.useRealTimers();
});

function mockTokenApi() {
  vi.spyOn(API, 'get').mockImplementation(async (url) => {
    if (url === '/api/token/auto-groups')
      return { data: { success: true, data: { groups: [], max_count: 0 } } };
    if (url === '/api/user/self/groups')
      return {
        data: {
          success: true,
          data: {
            base: { id: 11, code: 'base', name: '标准显示名称', ratio: 1 },
          },
        },
      };
    const match = /^\/api\/token\/(\d+)$/.exec(url);
    if (match)
      return {
        data: {
          success: true,
          data: {
            ...tokenRecords[match[1]],
            expired_time: -1,
            unlimited_quota: true,
            remain_quota: 0,
            model_limits: '',
            group: 'base',
            group_ids: [11],
            group_mode: 'explicit',
            group_ratio_limits: '{}',
          },
        },
      };
    return { data: { success: true, data: [] } };
  });
  return {
    put: vi.spyOn(API, 'put').mockResolvedValue({ data: { success: true } }),
    post: vi.spyOn(API, 'post').mockResolvedValue({ data: { success: true } }),
  };
}

function renderTokensEditor(onClosed) {
  function Harness() {
    const data = useTokensData();
    return (
      <>
        <EditTokenModal
          refresh={data.refresh}
          editingToken={data.editingToken}
          visiable={data.showEdit}
          handleClose={() => {
            data.closeEdit();
            onClosed();
          }}
        />
        <TokensActions
          selectedKeys={[]}
          setEditingToken={data.setEditingToken}
          setShowEdit={data.setShowEdit}
          batchCopyTokens={data.batchCopyTokens}
          batchDeleteTokens={data.batchDeleteTokens}
          t={data.t}
        />
        {[1, 2].map((id) => (
          <button
            key={id}
            data-testid={`edit-${id}`}
            onClick={() => {
              data.setEditingToken({ id });
              data.setShowEdit(true);
            }}
          />
        ))}
      </>
    );
  }
  return render(
    <StatusContext.Provider value={[{ status: {} }, () => {}]}>
      <Harness />
    </StatusContext.Provider>,
  );
}

async function submitEdit(tokenId, closed, closedCount) {
  fireEvent.click(await screen.findByRole('button', { name: /提交/ }));
  await waitFor(() => expect(closed).toHaveBeenCalledTimes(closedCount));
  expect(API.put).toHaveBeenLastCalledWith(
    '/api/token/',
    expect.objectContaining({ id: tokenId }),
  );
}

test('提交关闭后窗口内重开同一令牌，再次保存仍以原 ID 更新且不新建', async () => {
  const api = mockTokenApi();
  const closed = vi.fn();
  renderTokensEditor(closed);

  fireEvent.click(screen.getByTestId('edit-1'));
  await screen.findByDisplayValue('原令牌一');
  await submitEdit(1, closed, 1);

  fireEvent.click(screen.getByTestId('edit-1'));
  await waitFor(() =>
    expect(
      API.get.mock.calls.filter(([url]) => url === '/api/token/1'),
    ).toHaveLength(2),
  );
  await vi.advanceTimersByTimeAsync(600);
  await screen.findByDisplayValue('原令牌一');
  await submitEdit(1, closed, 2);

  expect(api.put).toHaveBeenCalledTimes(2);
  expect(api.post).not.toHaveBeenCalled();
});

test('提交关闭后窗口内改开另一令牌，保存使用新令牌 ID 且不新建', async () => {
  const api = mockTokenApi();
  const closed = vi.fn();
  renderTokensEditor(closed);

  fireEvent.click(screen.getByTestId('edit-1'));
  await screen.findByDisplayValue('原令牌一');
  await submitEdit(1, closed, 1);

  fireEvent.click(screen.getByTestId('edit-2'));
  await vi.advanceTimersByTimeAsync(600);
  await screen.findByDisplayValue('另一令牌二');
  await submitEdit(2, closed, 2);

  expect(api.post).not.toHaveBeenCalled();
});

test('编辑关闭并过了动画窗口后点击新建，保存走 POST 而不是更新', async () => {
  const api = mockTokenApi();
  const closed = vi.fn();
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
  renderTokensEditor(closed);

  fireEvent.click(screen.getByTestId('edit-1'));
  await screen.findByDisplayValue('原令牌一');
  await submitEdit(1, closed, 1);
  await vi.advanceTimersByTimeAsync(600);

  fireEvent.click(screen.getByRole('button', { name: /Create API key/ }));
  fireEvent.change(await screen.findByLabelText('名称'), {
    target: { value: '全新令牌' },
  });
  await user.click(screen.getByRole('button', { name: /选择分组|添加分组/ }));
  await user.click(await screen.findByRole('button', { name: /标准显示名称/ }));
  fireEvent.click(screen.getByRole('button', { name: /提交/ }));

  await waitFor(() =>
    expect(api.post).toHaveBeenCalledWith(
      '/api/token/',
      expect.objectContaining({ name: '全新令牌' }),
    ),
  );
  expect(api.put).toHaveBeenCalledTimes(1);
});

test('关闭后窗口内重开，过了原定清空时间编辑身份仍保留；真正关闭后才清空', async () => {
  const { result } = renderHook(() => useTokensData());

  act(() => {
    result.current.setEditingToken({ id: 1 });
    result.current.setShowEdit(true);
  });
  act(() => result.current.closeEdit());
  act(() => {
    result.current.setEditingToken({ id: 1 });
    result.current.setShowEdit(true);
  });
  await act(() => vi.advanceTimersByTimeAsync(600));
  expect(result.current.showEdit).toBe(true);
  expect(result.current.editingToken.id).toBe(1);

  act(() => result.current.closeEdit());
  await act(() => vi.advanceTimersByTimeAsync(600));
  expect(result.current.showEdit).toBe(false);
  expect(result.current.editingToken.id).toBeUndefined();
});
