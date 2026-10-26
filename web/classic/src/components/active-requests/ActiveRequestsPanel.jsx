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

import React, { useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { Activity, ChevronDown, ChevronUp, ChevronRight } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useActiveRequests, useRequestClock } from './useActiveRequests';
import ActiveRequestCard from './ActiveRequestCard';
import ActiveRequestsDrawer from './ActiveRequestsDrawer';
import ActiveRequestsNotice from './ActiveRequestsNotice';
import { getConnectionSubjectTitle } from './subject';
import './active-requests.css';

export default function ActiveRequestsPanel(props) {
  // 查询对象变化时重建面板，关闭旧抽屉及其筛选条件。
  return (
    <ActiveRequestsPanelContent
      key={JSON.stringify(props.subject || null)}
      {...props}
    />
  );
}

function ActiveRequestsPanelContent({ subject }) {
  const { t } = useTranslation();
  const mobile = useIsMobile();
  const activity = useActiveRequests({ pageSize: 3, subject });
  const snapshotStale =
    !!activity.snapshot && Date.now() - activity.snapshot.receivedAt > 5000;
  const serverNow = useRequestClock(
    activity.snapshot,
    activity.error || snapshotStale,
  );
  const [expanded, setExpanded] = useState(true);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const snapshot = activity.snapshot;
  const summary = snapshot?.summary;
  const title = getConnectionSubjectTitle(t, subject, snapshot?.subject);
  if (!subject && !activity.error && (!snapshot || snapshot.total === 0))
    return null;
  const items = snapshot?.items || [];
  const hiddenCount = Math.max(0, (snapshot?.total || 0) - items.length);
  return (
    <section
      className='active-requests-panel'
      aria-label={t('Current connections')}
    >
      <div className='active-requests-header'>
        <div className='active-requests-title'>
          <Activity size={18} aria-hidden='true' />
          <h2>{title}</h2>
          {summary && (
            <span className='active-requests-count'>{summary.total}</span>
          )}
        </div>
        <div className='active-requests-controls'>
          {snapshot && !activity.error && !snapshotStale && (
            <span className='active-requests-live'>{t('Live updates')}</span>
          )}
          <Button
            theme='borderless'
            size='small'
            aria-expanded={expanded}
            aria-label={
              expanded ? t('Collapse connections') : t('Expand connections')
            }
            icon={
              expanded ? <ChevronUp size={15} /> : <ChevronDown size={15} />
            }
            onClick={() => setExpanded(!expanded)}
          />
        </div>
      </div>
      <ActiveRequestsNotice activity={activity} />
      {expanded && !snapshot && activity.loading && (
        <p className='active-requests-empty' role='status'>
          {t('Loading connection status…')}
        </p>
      )}
      {expanded && summary && (
        <>
          <div className='active-requests-summary'>
            <span>
              {t('Streaming request')} <b>{summary.streaming}</b>
            </span>
            <span>
              {t('Non-streaming request')} <b>{summary.non_streaming}</b>
            </span>
            <span>
              {t('Waiting for response')} <b>{summary.waiting}</b>
            </span>
          </div>
          <div className='active-requests-cards'>
            {subject && snapshot.total === 0 && (
              <p className='active-requests-empty'>
                {t('This user has no current connections.')}
              </p>
            )}
            {items.map((request) => (
              <ActiveRequestCard
                key={request.id}
                request={request}
                serverNow={serverNow}
                compact={mobile}
              />
            ))}
          </div>
          <div className='active-requests-more'>
            <span>
              {hiddenCount > 0
                ? t('{{count}} more connections', { count: hiddenCount })
                : snapshot.total > 0
                  ? t('Ended requests remain visible for 5 seconds.')
                  : ''}
            </span>
            <Button
              theme='borderless'
              size='small'
              onClick={() => setDrawerOpen(true)}
            >
              {t('View all connections')}
              <ChevronRight size={14} />
            </Button>
          </div>
          {hiddenCount > 0 && (
            <p className='active-requests-retention'>
              {t('Ended requests remain visible for 5 seconds.')}
            </p>
          )}
        </>
      )}
      {drawerOpen && (
        <ActiveRequestsDrawer
          visible
          subject={subject}
          onClose={() => setDrawerOpen(false)}
        />
      )}
    </section>
  );
}
