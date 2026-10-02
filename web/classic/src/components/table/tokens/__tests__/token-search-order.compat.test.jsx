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

import { act, renderHook, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { API } from '../../../../helpers';
import { useTokensData } from '../../../../hooks/tokens/useTokensData';

test('实时搜索只接收最新响应，清空筛选不被旧结果覆盖', async () => {
  const requests = [];
  const response = (name) => ({
    data: {
      success: true,
      data: {
        items: name ? [{ id: 1, name }] : [],
        total: name ? 1 : 0,
        page: 1,
        page_size: 10,
      },
    },
  });
  vi.spyOn(API, 'get').mockImplementation((url) => {
    if (url.startsWith('/api/user/'))
      return Promise.resolve({ data: { success: true, data: {} } });
    if (!url.includes('search?')) return Promise.resolve(response('full list'));
    return new Promise((resolve) => requests.push({ url, resolve }));
  });
  const { result } = renderHook(() => useTokensData());
  await waitFor(() => expect(result.current.loading).toBe(false));
  let values = { searchKeyword: 'Gemini', searchToken: '', searchStatus: 1 };
  act(() => result.current.setFormApi({ getValues: () => values }));
  let first, second;
  act(() => {
    first = result.current.searchTokens();
  });
  values = { ...values, searchKeyword: 'Codex' };
  act(() => {
    second = result.current.searchTokens();
  });
  expect(requests[1].url).toContain('keyword=Codex');
  expect(requests[1].url).toContain('status=1');
  await act(async () => {
    requests[1].resolve(response('Codex'));
    await second;
  });
  await act(async () => {
    requests[0].resolve(response('Gemini'));
    await first;
  });
  expect(result.current.tokens[0].name).toBe('Codex');
  let pending;
  act(() => {
    pending = result.current.searchTokens();
  });
  values = { searchKeyword: '', searchToken: '', searchStatus: 0 };
  await act(async () => {
    await result.current.searchTokens();
  });
  await act(async () => {
    requests[2].resolve(response('stale'));
    await pending;
  });
  expect(result.current.tokens[0].name).toBe('full list');
  expect(result.current.searching).toBe(false);
  expect(result.current.loading).toBe(false);
});
