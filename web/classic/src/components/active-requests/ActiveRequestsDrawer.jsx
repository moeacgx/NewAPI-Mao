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

import React, { useEffect, useState } from 'react';
import { SideSheet, Input, Select, Button, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useActiveRequests, useRequestClock } from './useActiveRequests';
import ActiveRequestCard, { ACTIVE_STATUS_KEYS } from './ActiveRequestCard';
import ActiveRequestsNotice from './ActiveRequestsNotice';

export default function ActiveRequestsDrawer(props) {
  // 令牌身份改变时重建筛选状态，取消旧查询并从空条件第一页开始。
  return (
    <ActiveRequestsDrawerContent key={props.tokenId || 'all'} {...props} />
  );
}

function ActiveRequestsDrawerContent(props) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('');
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setPage(1);
      setQuery(search.trim());
    }, 300);
    return () => window.clearTimeout(timer);
  }, [search]);
  const activity = useActiveRequests({
    enabled: props.visible,
    page,
    pageSize: 10,
    search: query,
    status,
    tokenId: props.tokenId,
  });
  const serverNow = useRequestClock(activity.snapshot, activity.error);
  const snapshot = activity.snapshot;
  const pages = Math.max(1, Math.ceil((snapshot?.total || 0) / 10));
  useEffect(() => {
    if (snapshot && !activity.loading && page > pages) setPage(pages);
  }, [snapshot, activity.loading, page, pages]);
  return (
    <SideSheet
      visible={props.visible}
      onCancel={props.onClose}
      title={
        props.tokenName
          ? `${t('View connection details')}: ${props.tokenName}`
          : t('All current connections')
      }
      width={isMobile ? '100%' : 760}
      className='active-requests-drawer'
      closeOnEsc
    >
      <ActiveRequestsNotice activity={activity} />
      {snapshot && (
        <p className='active-requests-description'>
          {t('{{total}} active requests · {{tokens}} tokens', {
            total: snapshot.summary.total,
            tokens: snapshot.summary.active_tokens,
          })}
        </p>
      )}
      <div className='active-requests-filters'>
        <Input
          value={search}
          onChange={setSearch}
          aria-label={t('Search connections')}
          placeholder={t('Search model, token or request ID')}
          showClear
        />
        <Select
          value={status}
          aria-label={t('Connection status')}
          onChange={(value) => {
            setPage(1);
            setStatus(value);
          }}
          optionList={[
            { value: '', label: t('All statuses') },
            ...Object.entries(ACTIVE_STATUS_KEYS).map(([value, label]) => ({
              value,
              label: t(label),
            })),
          ]}
        />
      </div>
      <div className='active-requests-drawer-list' aria-busy={activity.loading}>
        {activity.loading ? (
          <Spin />
        ) : (
          snapshot?.items.map((request) => (
            <ActiveRequestCard
              key={request.id}
              request={request}
              serverNow={serverNow}
              compact={isMobile}
              showRequestId
            />
          ))
        )}
        {!activity.loading && snapshot?.total === 0 && (
          <p className='active-requests-empty'>
            {t('No matching connections')}
          </p>
        )}
      </div>
      <div className='active-requests-pagination'>
        <span>
          {snapshot
            ? t('{{count}} matching connections', { count: snapshot.total })
            : '—'}
        </span>
        <div>
          <Button
            icon={<ChevronLeft size={15} />}
            aria-label={t('Previous connections page')}
            disabled={page <= 1 || activity.loading}
            onClick={() => setPage(page - 1)}
          />
          <span>
            {page} / {pages}
          </span>
          <Button
            icon={<ChevronRight size={15} />}
            aria-label={t('Next connections page')}
            disabled={page >= pages || activity.loading}
            onClick={() => setPage(page + 1)}
          />
        </div>
      </div>
    </SideSheet>
  );
}
