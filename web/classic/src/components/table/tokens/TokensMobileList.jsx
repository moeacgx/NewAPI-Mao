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

// Copyright (C) 2023-2026 QuantumNous；移植自 Default upstream/main@789c97019 的 ApiKeysMobileList，保留上游归属。
import React, { useMemo } from 'react';
import { Database } from 'lucide-react';
import './tokens-mobile.css';

function MobileTokenCell({ column, record, index }) {
  if (!column) return null;
  const value = record[column.dataIndex];
  return column.render ? column.render(value, record, index) : value;
}

export default function TokensMobileList({
  tokens,
  columns,
  loading,
  searching,
  t,
}) {
  const cells = useMemo(
    () =>
      Object.fromEntries(
        columns.map((column) => [column.dataIndex || column.key, column]),
      ),
    [columns],
  );

  if (loading || searching) {
    return (
      <div className='tokens-mobile-list' aria-busy='true'>
        <span className='tokens-mobile-sr-only' role='status'>
          {t('加载中...')}
        </span>
        {[0, 1, 2, 3, 4].map((index) => (
          <div
            key={index}
            className='tokens-mobile-card tokens-mobile-skeleton'
            aria-hidden='true'
          >
            <div className='tokens-mobile-skeleton-heading'>
              <span />
              <span />
            </div>
            <span />
            <span />
            <span />
          </div>
        ))}
      </div>
    );
  }

  if (!tokens.length) {
    return (
      <div className='tokens-mobile-list'>
        <div className='tokens-mobile-empty' role='status'>
          <Database size={24} aria-hidden='true' />
          <p>{t('搜索无结果')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className='tokens-mobile-list'>
      {tokens.map((record, index) => (
        <article
          key={record.id}
          className={
            'tokens-mobile-card' +
            (record.status !== 1 ? ' tokens-mobile-card-disabled' : '')
          }
          aria-label={record.name}
        >
          <div className='tokens-mobile-heading'>
            <h2>{record.name}</h2>
            <div className='tokens-mobile-status'>
              <MobileTokenCell
                column={cells.status}
                record={record}
                index={index}
              />
            </div>
          </div>

          <div className='tokens-mobile-key-actions'>
            <div className='tokens-mobile-key'>
              <MobileTokenCell
                column={cells.token_key}
                record={record}
                index={index}
              />
            </div>
            <div className='tokens-mobile-actions'>
              <MobileTokenCell
                column={cells.operate}
                record={record}
                index={index}
              />
            </div>
          </div>

          <div className='tokens-mobile-group-quota'>
            <div className='tokens-mobile-group'>
              <MobileTokenCell
                column={cells.group}
                record={record}
                index={index}
              />
            </div>
            <div className='tokens-mobile-quota'>
              <MobileTokenCell
                column={cells.quota_usage}
                record={record}
                index={index}
              />
            </div>
          </div>

          <div className='tokens-mobile-restrictions'>
            <div>
              <span className='tokens-mobile-label'>{t('Models')}</span>
              <MobileTokenCell
                column={cells.model_limits}
                record={record}
                index={index}
              />
            </div>
            <div>
              <span className='tokens-mobile-label'>{t('IP限制')}</span>
              <MobileTokenCell
                column={cells.allow_ips}
                record={record}
                index={index}
              />
            </div>
          </div>

          <div className='tokens-mobile-footer'>
            <div className='tokens-mobile-time'>
              <MobileTokenCell
                column={cells.activity_time}
                record={record}
                index={index}
              />
            </div>
            <div className='tokens-mobile-expiry'>
              <span className='tokens-mobile-label'>{t('过期时间')}</span>
              <MobileTokenCell
                column={cells.expired_time}
                record={record}
                index={index}
              />
            </div>
          </div>
        </article>
      ))}
    </div>
  );
}
