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
import './active-requests.css';

export default function ActiveRequestsPanel() {
  const { t } = useTranslation();
  const mobile = useIsMobile();
  const activity = useActiveRequests({ pageSize: 3 });
  const serverNow = useRequestClock(activity.snapshot, activity.error);
  const [expanded, setExpanded] = useState(true);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const snapshot = activity.snapshot;
  const summary = snapshot?.summary;
  if (!activity.error && (!summary || summary.total === 0)) return null;
  const items = snapshot?.items || [];
  const hiddenCount = Math.max(0, (summary?.total || 0) - items.length);
  return (
    <section
      className='active-requests-panel'
      aria-label={t('Current connections')}
    >
      <div className='active-requests-header'>
        <div className='active-requests-title'>
          <Activity size={18} aria-hidden='true' />
          <h2>{t('Current connections')}</h2>
          {summary && (
            <span className='active-requests-count'>{summary.total}</span>
          )}
        </div>
        <div className='active-requests-controls'>
          {!activity.error && (
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
          <div className='active-requests-cards' data-count={items.length}>
            {items.map((request) => (
              <ActiveRequestCard
                key={request.id}
                request={request}
                serverNow={serverNow}
                compact={mobile}
                wide={!mobile && items.length === 1}
              />
            ))}
          </div>
          <div className='active-requests-more'>
            <span>
              {hiddenCount > 0
                ? t('{{count}} more connections', { count: hiddenCount })
                : t('Showing your in-progress requests')}
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
        </>
      )}
      {drawerOpen && (
        <ActiveRequestsDrawer visible onClose={() => setDrawerOpen(false)} />
      )}
    </section>
  );
}
