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

export function buildRedemptionExport({
  codes,
  name,
  quota,
  format = 'txt',
  includeName = false,
  includeQuota = false,
  labels = { code: 'Code', name: 'Name', quota: 'Quota' },
}) {
  const clean = (value) => String(value ?? '').replace(/[\r\n\t]/g, ' ');
  const escape = (value) =>
    clean(value)
      .replace(/&/g, '&amp;')
      .replace(/([\\|`*_<>\[\]])/g, '\\$1');
  const rows = codes.map((code) => [
    code,
    ...(includeName ? [name] : []),
    ...(includeQuota ? [quota] : []),
  ]);
  const filename = `${clean(name).replace(/[<>:"/\\|?*]/g, '_') || 'redemptions'}.${format === 'md' ? 'md' : 'txt'}`;
  if (format !== 'md')
    return {
      filename,
      text: rows.map((row) => row.map(clean).join('\t')).join('\n') + '\n',
    };
  const headers = [
    labels.code,
    ...(includeName ? [labels.name] : []),
    ...(includeQuota ? [labels.quota] : []),
  ];
  return {
    filename,
    text:
      [headers, headers.map(() => '---'), ...rows]
        .map((row) => `| ${row.map(escape).join(' | ')} |`)
        .join('\n') + '\n',
  };
}
