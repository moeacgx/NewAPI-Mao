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
import { API } from '../../helpers/api';

export function useActiveRequests(options = {}) {
  const {
    enabled = true,
    page = 1,
    pageSize = 3,
    search = '',
    status = '',
    tokenId,
  } = options;
  const [snapshot, setSnapshot] = useState(null);
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshIndex, setRefreshIndex] = useState(0);
  const refresh = useCallback(() => setRefreshIndex((value) => value + 1), []);

  useEffect(() => {
    if (!enabled) return;
    let disposed = false;
    let timer;
    let controller;
    let sequence = 0;
    setLoading(true);
    // 每次轮询结束后再调度，避免慢请求与后续轮询重叠。
    const poll = async () => {
      if (disposed || document.visibilityState === 'hidden') return;
      const current = ++sequence;
      controller = new AbortController();
      try {
        const result = await API.get('/api/log/self/active', {
          params: {
            page,
            page_size: pageSize,
            search,
            status,
            token_id: tokenId,
          },
          signal: controller.signal,
          skipErrorHandler: true,
        });
        if (disposed || current !== sequence) return;
        if (!result.data?.success || !result.data?.data)
          throw new Error('active unavailable');
        const data = result.data.data;
        setSnapshot({ ...data, receivedAt: Date.now() });
        setError(false);
      } catch {
        if (disposed || current !== sequence) return;
        setError(true);
      } finally {
        if (!disposed && current === sequence) {
          setLoading(false);
          timer = window.setTimeout(poll, 2000);
        }
      }
    };
    const visibilityChanged = () => {
      window.clearTimeout(timer);
      ++sequence;
      controller?.abort();
      if (document.visibilityState !== 'hidden') poll();
    };
    document.addEventListener('visibilitychange', visibilityChanged);
    poll();
    return () => {
      disposed = true;
      ++sequence;
      controller?.abort();
      window.clearTimeout(timer);
      document.removeEventListener('visibilitychange', visibilityChanged);
    };
  }, [enabled, page, pageSize, search, status, tokenId, refreshIndex]);

  return { snapshot, error, loading, refresh };
}

export function useRequestClock(snapshot, stale) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!snapshot || stale) return;
    setNow(Date.now());
    const timer = window.setInterval(() => {
      if (document.visibilityState !== 'hidden') setNow(Date.now());
    }, 1000);
    return () => window.clearInterval(timer);
  }, [snapshot, stale]);
  if (!snapshot) return 0;
  return snapshot.observed_at + Math.max(0, now - snapshot.receivedAt);
}
