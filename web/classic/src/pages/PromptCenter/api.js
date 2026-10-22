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

export async function promptRequest(method, url, data, signal) {
  const response = await API.request({
    method,
    url,
    data,
    signal,
    skipErrorHandler: true,
  });
  if (!response.data?.success)
    throw new Error(response.data?.message || 'Prompt request failed');
  return response.data.data;
}

export function promptError(error, t) {
  if (error.response?.status === 401 || error.response?.status === 403)
    return t('No permission to manage prompts');
  return (
    error.response?.data?.message ||
    (error.message && t(error.message)) ||
    t('Prompt request failed')
  );
}
