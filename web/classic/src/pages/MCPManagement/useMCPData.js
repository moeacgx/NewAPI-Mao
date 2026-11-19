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

import { useCallback, useEffect, useState } from 'react';
import { mcpRequest } from './api';

export default function useMCPData(path, paged = false) {
  const [page, setPage] = useState(1);
  const [revision, setRevision] = useState(0);
  const [state, setState] = useState({
    items: [],
    total: 0,
    loading: true,
    error: false,
  });
  const refresh = useCallback(() => setRevision((value) => value + 1), []);
  useEffect(() => {
    const controller = new AbortController();
    setState({ items: [], total: 0, loading: true, error: false });
    mcpRequest(
      'get',
      paged ? `${path}?p=${page}&page_size=20` : path,
      undefined,
      controller.signal,
    )
      .then((data) => {
        if (controller.signal.aborted) return;
        const items = paged ? data?.items : data;
        if (!Array.isArray(items) || (paged && !Number.isFinite(data?.total))) {
          throw new Error('Invalid MCP response');
        }
        if (paged && page > 1 && items.length === 0) {
          setPage(Math.max(1, Math.ceil(data.total / 20)));
        }
        setState({
          items,
          total: paged ? data.total : items.length,
          loading: false,
          error: false,
        });
      })
      .catch(() => {
        if (!controller.signal.aborted)
          setState({ items: [], total: 0, loading: false, error: true });
      });
    return () => controller.abort();
  }, [path, paged, page, revision]);
  return { ...state, page, setPage, refresh };
}
