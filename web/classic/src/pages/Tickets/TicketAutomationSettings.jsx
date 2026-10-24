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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Card, Spin, Switch, TextArea } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers/api';
import { showError, showSuccess } from '../../helpers/utils';
import { ticketError, ticketResponse } from './ticket-utils';

const AUTOMATION_ENDPOINT = '/api/ticket-admin/automation/auto-reply';
const DEFAULT_AUTO_REPLY_CONTENT = '您好，已收到您的工单，我们会尽快处理。';
const MAX_CONTENT_LENGTH = 10000;

export default function TicketAutomationSettings(props) {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(false);
  const [content, setContent] = useState(DEFAULT_AUTO_REPLY_CONTENT);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [retryKey, setRetryKey] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const active = useRef(true);

  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError('');
    API.get(AUTOMATION_ENDPOINT, {
      skipErrorHandler: true,
      disableDuplicate: true,
    })
      .then((response) => {
        const data =
          ticketResponse(response, t('Failed to load automation settings')) ||
          {};
        if (!active) return;
        setEnabled(data.enabled === true);
        setContent(data.content ?? DEFAULT_AUTO_REPLY_CONTENT);
        setLoaded(true);
      })
      .catch((failure) => {
        if (active) {
          setError(
            t(ticketError(failure, t('Failed to load automation settings'))),
          );
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [retryKey, t]);

  const save = async () => {
    if (!props.canWrite || saving || !loaded) return;
    const nextContent = content.trim();
    if (enabled && !nextContent) {
      setError(t('Enter an automatic reply before enabling this rule.'));
      return;
    }
    if (Array.from(content).length > MAX_CONTENT_LENGTH) {
      setError(
        t('Automatic reply must be {{count}} characters or fewer.', {
          count: MAX_CONTENT_LENGTH,
        }),
      );
      return;
    }
    setSaving(true);
    setError('');
    try {
      const response = await API.put(
        AUTOMATION_ENDPOINT,
        { enabled, content },
        { skipErrorHandler: true },
      );
      const data = ticketResponse(
        response,
        t('Failed to save automation settings'),
      );
      if (!active.current) return;
      setEnabled(data?.enabled === true);
      setContent(data?.content ?? content);
      showSuccess(t('Automation settings saved'));
    } catch (failure) {
      if (!active.current) return;
      setError(
        t(ticketError(failure, t('Failed to save automation settings'))),
      );
      showError(
        t(ticketError(failure, t('Failed to save automation settings'))),
      );
    } finally {
      if (active.current) setSaving(false);
    }
  };

  return (
    <Card className='ticket-panel ticket-automation-panel'>
      <div className='ticket-automation-header'>
        <div>
          <h2>{t('Ticket automation')}</h2>
          <p className='ticket-muted'>
            {t('Automatically reply once after a user creates a ticket.')}
          </p>
        </div>
        {loading && <Spin size='small' />}
      </div>
      {error && (
        <div role='alert' className='ticket-error'>
          {error}
          {!loaded && (
            <Button
              onClick={() => setRetryKey((value) => value + 1)}
              disabled={saving}
            >
              {t('Retry')}
            </Button>
          )}
        </div>
      )}
      {!loading && loaded && (
        <div className='ticket-automation-form'>
          <label className='ticket-automation-switch'>
            <Switch
              checked={enabled}
              onChange={setEnabled}
              disabled={!props.canWrite || saving}
              aria-label={t('Enable automatic first reply')}
            />
            <span>{t('Enable automatic first reply')}</span>
          </label>
          <div>
            <label htmlFor='ticket-automation-content'>
              {t('Automatic reply content')}
            </label>
            <TextArea
              id='ticket-automation-content'
              value={content}
              onChange={setContent}
              disabled={!props.canWrite || saving}
              maxLength={MAX_CONTENT_LENGTH}
              autosize={{ minRows: 4, maxRows: 10 }}
              placeholder={t(
                'For example: Hello, we received your ticket and will review it shortly.',
              )}
            />
            <small className='ticket-muted'>
              {t('{{count}} / {{max}} characters', {
                count: Array.from(content).length,
                max: MAX_CONTENT_LENGTH,
              })}
            </small>
          </div>
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
              disabled={!props.canWrite}
            >
              {t('Save automation settings')}
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}
