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
import { useTranslation } from 'react-i18next';
import { Button } from '@douyinfe/semi-ui';
import { Activity, ChevronRight } from 'lucide-react';
import ActiveRequestsNotice from './ActiveRequestsNotice';
import ActiveRequestsDrawer from './ActiveRequestsDrawer';
import './active-requests.css';

export default function TokenActivityBar(props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const summary = props.activity.snapshot?.summary;
  return (
    <section className='active-token-bar' aria-label={t('Token activity')}>
      <div className='active-token-bar-main'>
        <div className='active-token-bar-count'>
          <Activity size={15} aria-hidden='true' />
          {summary ? (
            <>
              <strong>
                {t('{{count}} tokens in use', { count: summary.active_tokens })}
              </strong>
              <span>
                {t('{{count}} active requests', { count: summary.total })}
              </span>
            </>
          ) : (
            <span>
              {props.activity.error
                ? t('Connection status is unavailable.')
                : t('Loading connection status…')}
            </span>
          )}
        </div>
        <Button theme='borderless' size='small' onClick={() => setOpen(true)}>
          {t('View connection details')}
          <ChevronRight size={14} />
        </Button>
      </div>
      <ActiveRequestsNotice activity={props.activity} />
      {open && <ActiveRequestsDrawer visible onClose={() => setOpen(false)} />}
    </section>
  );
}

export function TokenActivityBadge(props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  if (!(props.count > 0) && !open) return null;
  const label = (
    <>
      {t('{{count}} active requests', { count: props.count })}
      {props.stale && <span> · {t('Stale')}</span>}
    </>
  );
  if (!props.tokenId)
    return <span className='active-token-badge'>{label}</span>;
  return (
    <>
      {props.count > 0 && (
        <button
          type='button'
          className='active-token-badge'
          aria-label={`${t('View connection details')}: ${props.tokenName || props.tokenId}`}
          onClick={() => setOpen(true)}
        >
          {label}
        </button>
      )}
      {open && (
        <ActiveRequestsDrawer
          visible
          tokenId={props.tokenId}
          tokenName={props.tokenName}
          onClose={() => setOpen(false)}
        />
      )}
    </>
  );
}
