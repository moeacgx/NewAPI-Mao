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

import React from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@douyinfe/semi-ui';

export default function ActiveRequestsNotice(props) {
  const { t } = useTranslation();
  return (
    <>
      {props.activity.error && (
        <div className='active-requests-notice' role='alert'>
          <span>
            {props.activity.snapshot
              ? t('Connection status is stale. Updates are unavailable.')
              : t('Connection status is unavailable.')}
          </span>
          <Button
            size='small'
            theme='borderless'
            onClick={props.activity.refresh}
          >
            {t('Retry')}
          </Button>
        </div>
      )}
      {!props.activity.error &&
        props.activity.refreshing &&
        props.activity.snapshot &&
        Date.now() - props.activity.snapshot.receivedAt > 5000 && (
          <p className='active-requests-scope' role='status'>
            {t('Connection data is delayed. Showing the last known status.')}
          </p>
        )}
      {!props.activity.error &&
        !props.activity.refreshing &&
        props.activity.snapshot &&
        Date.now() - props.activity.snapshot.receivedAt > 5000 && (
          <div className='active-requests-notice' role='status'>
            {t('Connection status is stale. Updates are unavailable.')}
          </div>
        )}
      {props.activity.snapshot?.scope === 'instance' && (
        <p className='active-requests-scope'>
          {t('Only connections on the current instance are shown.')}
        </p>
      )}
    </>
  );
}
