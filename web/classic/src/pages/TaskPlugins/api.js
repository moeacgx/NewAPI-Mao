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

import { API } from '../../helpers/api';

const base = '/api/plugin/task';
const config = { skipErrorHandler: true };

export async function taskPluginRequest(method, path = '', body, signal) {
  const response = await API.request({
    ...config,
    method,
    url: base + path,
    data: body,
    signal,
  });
  if (response.data?.success !== true) {
    throw new Error(response.data?.message || 'Task plugin request failed');
  }
  return response.data.data;
}

export function pluginError(error, t) {
  if (error.response?.status === 403 || error.response?.status === 401) {
    return t('No permission to access task plugins');
  }
  if (error.response?.status === 404 || error.response?.status === 501) {
    return t('Task plugin management is unavailable on this backend');
  }
  return (
    error.response?.data?.message ||
    error.message ||
    t('Task plugin request failed')
  );
}

export const taskPluginPath = (key) => '/' + encodeURIComponent(key);
