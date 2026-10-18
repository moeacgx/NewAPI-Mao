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

import { useCallback, useEffect, useRef, useState } from 'react';
import { API } from '../../helpers/api';
import { GROUP_TAGS_UPDATED_EVENT } from '../../helpers/groupTags';
import { GROUP_DETAILS_UPDATED_EVENT } from '../../helpers/groupDetails';

// 管理页共享目录；标签请求失败时仍保留已有分组编辑能力。
export default function useGroupTags() {
  const [tags, setTags] = useState([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const requestVersion = useRef(0);
  const reload = useCallback(async () => {
    const version = ++requestVersion.current;
    setLoading(true);
    try {
      const response = await API.get('/api/group/tags', {
        disableDuplicate: true,
      });
      if (version !== requestVersion.current) return;
      if (!response.data.success) throw new Error(response.data.message);
      setTags(response.data.data || []);
      setError('');
    } catch (error) {
      if (version === requestVersion.current)
        setError(error.message || 'Failed to load group tags');
    } finally {
      if (version === requestVersion.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    reload();
    window.addEventListener(GROUP_TAGS_UPDATED_EVENT, reload);
    window.addEventListener(GROUP_DETAILS_UPDATED_EVENT, reload);
    return () => {
      requestVersion.current += 1;
      window.removeEventListener(GROUP_TAGS_UPDATED_EVENT, reload);
      window.removeEventListener(GROUP_DETAILS_UPDATED_EVENT, reload);
    };
  }, [reload]);
  return { tags, error, loading, reload };
}
