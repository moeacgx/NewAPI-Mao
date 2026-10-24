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
import React, { useEffect, useId, useRef, useState } from 'react';
import { Button, Card, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers/api';
import { ticketError, ticketResponse, ticketStatuses } from './ticket-utils';

const ENDPOINT = '/api/ticket-admin/automation/auto-archive';
const ARCHIVE_SOURCES = ['open', 'processing', 'resolved', 'closed'];

export default function TicketArchiveSettings(props) {
  const { t } = useTranslation();
  const id = useId();
  const active = useRef(true);
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  useEffect(() => {
    let current = true;
    setLoading(true);
    setError('');
    API.get(ENDPOINT, { skipErrorHandler: true, disableDuplicate: true })
      .then((response) => {
        const data = ticketResponse(
          response,
          t('Failed to load archive settings'),
        );
        if (current)
          setConfig({
            enabled: data.enabled === true,
            inactive_days: String(data.inactive_days),
            statuses: data.statuses,
          });
      })
      .catch((failure) => {
        if (current)
          setError(
            t(ticketError(failure, t('Failed to load archive settings'))),
          );
      })
      .finally(() => {
        if (current) setLoading(false);
      });
    return () => {
      current = false;
    };
  }, [revision, t]);

  const save = async () => {
    if (!props.canWrite || saving || !config) return;
    setSaved(false);
    const days = Number(config.inactive_days);
    if (!Number.isInteger(days) || days < 1 || days > 3650) {
      setError(t('Enter a whole number of days from 1 to 3650.'));
      return;
    }
    if (!config.statuses.length) {
      setError(t('Select at least one status to archive.'));
      return;
    }
    setSaving(true);
    setError('');
    try {
      const data = ticketResponse(
        await API.put(
          ENDPOINT,
          { ...config, inactive_days: days },
          { skipErrorHandler: true },
        ),
        t('Failed to save archive settings'),
      );
      if (!active.current) return;
      setConfig({ ...data, inactive_days: String(data.inactive_days) });
      setSaved(true);
    } catch (failure) {
      if (active.current)
        setError(t(ticketError(failure, t('Failed to save archive settings'))));
    } finally {
      if (active.current) setSaving(false);
    }
  };

  return (
    <Card className='ticket-panel ticket-automation-panel'>
      <div className='ticket-automation-header'>
        <div>
          <h2>{t('Automatic ticket archive')}</h2>
          <p className='ticket-muted'>
            {t(
              'Archive inactive tickets after the selected number of days. Replies and status changes restart the timer. Existing tickets are included.',
            )}
          </p>
        </div>
        {loading && <Spin size='small' />}
      </div>
      {error && (
        <div role='alert' className='ticket-error'>
          {error}
          {!config && (
            <Button
              onClick={() => setRevision((value) => value + 1)}
              disabled={loading}
            >
              {t('Retry')}
            </Button>
          )}
        </div>
      )}
      {saved && <p role='status'>{t('Archive settings saved')}</p>}
      {config && !loading && (
        <div className='ticket-automation-form'>
          <label className='ticket-automation-switch'>
            <input
              type='checkbox'
              checked={config.enabled}
              disabled={!props.canWrite || saving}
              onChange={(event) => {
                setConfig({ ...config, enabled: event.target.checked });
                setSaved(false);
              }}
            />
            {t('Enable automatic archive')}
          </label>
          <div>
            <label htmlFor={`${id}-days`}>{t('Inactive days')}</label>
            <input
              className='ticket-select'
              id={`${id}-days`}
              type='number'
              min='1'
              max='3650'
              step='1'
              value={config.inactive_days}
              disabled={!props.canWrite || saving}
              onChange={(event) => {
                setConfig({ ...config, inactive_days: event.target.value });
                setSaved(false);
              }}
            />
          </div>
          <fieldset
            className='ticket-archive-statuses'
            disabled={!props.canWrite || saving}
          >
            <legend>{t('Statuses to archive')}</legend>
            {ARCHIVE_SOURCES.map((status) => (
              <label key={status}>
                <input
                  type='checkbox'
                  checked={config.statuses.includes(status)}
                  onChange={(event) => {
                    setConfig({
                      ...config,
                      statuses: event.target.checked
                        ? [...config.statuses, status]
                        : config.statuses.filter((value) => value !== status),
                    });
                    setSaved(false);
                  }}
                />
                {t(ticketStatuses[status])}
              </label>
            ))}
          </fieldset>
          {!props.canWrite && (
            <p className='ticket-muted'>
              {t('You have read-only access to ticket automation.')}
            </p>
          )}
          <div className='ticket-actions'>
            <Button
              theme='solid'
              onClick={save}
              loading={saving}
              disabled={!props.canWrite || saving}
            >
              {t('Save archive settings')}
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}
