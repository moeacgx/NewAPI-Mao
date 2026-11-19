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
import { normalizeLanguage, supportedLanguages } from '../../i18n/language';

export async function mcpRequest(method, path, data, signal) {
  const response = await API.request({
    method,
    url: `/api/mcp/${path}`,
    data,
    signal,
    skipErrorHandler: true,
  });
  if (response.data?.success !== true) throw new Error('MCP request failed');
  return response.data.data;
}

export function formatTime(seconds, language) {
  if (!Number.isFinite(seconds) || seconds <= 0) return '-';
  const date = new Date(seconds * 1000);
  if (!Number.isFinite(date.getTime())) return '-';
  const normalizedLocale = normalizeLanguage(language);
  const locale = supportedLanguages.includes(normalizedLocale)
    ? normalizedLocale
    : 'zh-CN';
  return date.toLocaleString(locale);
}
