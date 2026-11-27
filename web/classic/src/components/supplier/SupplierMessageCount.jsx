/*
Copyright (C) 2023-2026 QuantumNous

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

import React from 'react';
import { MessageSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';

// 只展示列表接口带回的对话数量。没有消息时不渲染，内部备注不计数。
const SupplierMessageCount = ({ count = 0 }) => {
  const { t } = useTranslation();
  const total = Number(count) || 0;
  if (total <= 0) return null;
  return (
    <span
      role='img'
      aria-label={t('{{count}} messages', { count: total })}
      className='inline-flex items-center gap-1'
      style={{ color: 'var(--semi-color-text-2)', fontSize: 12 }}
    >
      <MessageSquare aria-hidden='true' size={14} />
      <span>{total}</span>
    </span>
  );
};

export default SupplierMessageCount;
