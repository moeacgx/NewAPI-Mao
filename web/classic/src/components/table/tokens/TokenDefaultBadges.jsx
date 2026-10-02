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

// Copyright (C) 2023-2026 QuantumNous；移植自 Default upstream/main@789c97019，保留上游归属。
import React from 'react';
// 与 Default StatusBadge / stringToColor 使用相同的语义色顺序与映射。
const palette = [
  'amber',
  'blue',
  'cyan',
  'green',
  'grey',
  'indigo',
  'light-blue',
  'lime',
  'orange',
  'pink',
  'purple',
  'red',
  'teal',
  'violet',
  'yellow',
];
const variants = {
  amber: 'warning',
  blue: 'chart-1',
  cyan: 'chart-2',
  green: 'success',
  grey: 'neutral',
  indigo: 'chart-1',
  'light-blue': 'info',
  lime: 'chart-3',
  orange: 'warning',
  pink: 'chart-5',
  purple: 'chart-4',
  red: 'danger',
  teal: 'chart-2',
  violet: 'chart-4',
  yellow: 'warning',
};
export function TokenStatusBadge({
  label,
  variant = 'neutral',
  autoColor,
  className = '',
}) {
  let resolved = variant;
  if (autoColor) {
    let sum = 0;
    for (let i = 0; i < autoColor.length; i++) sum += autoColor.charCodeAt(i);
    resolved = variants[palette[sum % palette.length]];
  }
  return (
    <span
      data-slot='status-badge'
      className={'tokens-status tokens-status--' + resolved + ' ' + className}
      title={label}
    >
      {label}
    </span>
  );
}
export function TokenGroupBadge({ code, name, ratio }) {
  let variant = 'neutral';
  if (ratio > 1) variant = 'warning';
  else if (ratio < 1) variant = 'info';
  return (
    <span className='tokens-group'>
      <TokenStatusBadge
        label={name}
        autoColor={code && code !== 'auto' ? code : undefined}
        className='tokens-group-name'
      />
      {typeof ratio === 'number' && (
        <span
          data-slot='badge'
          className={'tokens-ratio tokens-ratio--' + variant}
        >
          {ratio}x
        </span>
      )}
    </span>
  );
}
