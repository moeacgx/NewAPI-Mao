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
import { useTranslation } from 'react-i18next';
import { Tooltip } from '@douyinfe/semi-ui';
import { timestamp2string } from '../../../helpers';
function formatTimestampRelative(timestamp, unit = 'seconds', locales, now) {
  if (!timestamp || timestamp === -1 || timestamp === 0) {
    return '-';
  }

  const ms = unit === 'seconds' ? timestamp * 1000 : timestamp;
  const diffSeconds = Math.round((ms - now) / 1000);
  const absSeconds = Math.abs(diffSeconds);
  const formatter = new Intl.RelativeTimeFormat(locales, {
    numeric: 'always',
  });

  if (absSeconds < 60) {
    return formatter.format(diffSeconds, 'second');
  }
  if (absSeconds < 3600) {
    return formatter.format(Math.round(diffSeconds / 60), 'minute');
  }
  if (absSeconds < 86400) {
    return formatter.format(Math.round(diffSeconds / 3600), 'hour');
  }
  if (absSeconds < 2592000) {
    return formatter.format(Math.round(diffSeconds / 86400), 'day');
  }
  if (absSeconds < 31536000) {
    return formatter.format(Math.round(diffSeconds / 2592000), 'month');
  }
  return formatter.format(Math.round(diffSeconds / 31536000), 'year');
}
export default function TokenTimestamp({ timestamp, now = Date.now() }) {
  const { t, i18n } = useTranslation();
  if (!timestamp || timestamp === -1 || !Number.isFinite(timestamp)) {
    return <span>-</span>;
  }
  const date = new Date(timestamp * 1000);
  if (!Number.isFinite(date.getTime())) return <span>-</span>;
  const justNow = timestamp * 1000 <= now && now - timestamp * 1000 < 60000;
  return (
    <Tooltip content={timestamp2string(timestamp)}>
      <time dateTime={date.toISOString()} tabIndex={0}>
        {justNow
          ? t('Just now')
          : formatTimestampRelative(timestamp, 'seconds', i18n.language, now)}
      </time>
    </Tooltip>
  );
}
